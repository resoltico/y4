package memory

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"otsu-obliterator/internal/logger"
	"otsu-obliterator/internal/opencv/safe"

	"gocv.io/x/gocv"
)

type Manager struct {
	mu           sync.RWMutex
	logger       logger.Logger
	maxMemory    int64
	usedMemory   int64
	allocCount   int64
	deallocCount int64
	activeMats   map[uint64]*MatInfo
	matPool      sync.Pool
	ctx          context.Context
	cancel       context.CancelFunc
}

type MatInfo struct {
	ID        uint64
	Tag       string
	Size      int64
	Timestamp time.Time
}

func NewManager(log logger.Logger) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &Manager{
		logger:     log,
		maxMemory:  2 * 1024 * 1024 * 1024,
		activeMats: make(map[uint64]*MatInfo),
		ctx:        ctx,
		cancel:     cancel,
		matPool: sync.Pool{
			New: func() interface{} {
				return &safe.Mat{}
			},
		},
	}

	go manager.monitorMemory()
	return manager
}

func (m *Manager) GetMat(rows, cols int, matType gocv.MatType, tag string) (*safe.Mat, error) {
	size := int64(rows * cols * getMatTypeSize(matType))

	m.mu.Lock()
	if m.usedMemory+size > m.maxMemory {
		m.mu.Unlock()
		runtime.GC()
		return nil, fmt.Errorf("memory limit exceeded: would use %d bytes, limit is %d",
			m.usedMemory+size, m.maxMemory)
	}
	m.mu.Unlock()

	mat, err := safe.NewMatWithTracker(rows, cols, matType, m, tag)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.usedMemory += size
	m.allocCount++
	m.activeMats[mat.ID()] = &MatInfo{
		ID:        mat.ID(),
		Tag:       tag,
		Size:      size,
		Timestamp: time.Now(),
	}
	m.mu.Unlock()

	return mat, nil
}

func (m *Manager) GetPooledMat() *safe.Mat {
	return m.matPool.Get().(*safe.Mat)
}

func (m *Manager) ReturnPooledMat(mat *safe.Mat) {
	if mat != nil {
		mat.Reset()
		m.matPool.Put(mat)
	}
}

func (m *Manager) TrackAllocation(ptr uintptr, size int64, tag string) {
}

func (m *Manager) TrackDeallocation(ptr uintptr, tag string) {
	m.mu.Lock()
	m.deallocCount++
	m.mu.Unlock()
}

func (m *Manager) ReleaseMat(mat *safe.Mat, tag string) {
	if mat == nil {
		return
	}

	m.mu.Lock()
	if info, exists := m.activeMats[mat.ID()]; exists {
		delete(m.activeMats, mat.ID())
		m.usedMemory -= info.Size
	}
	m.mu.Unlock()

	mat.Close()
}

func (m *Manager) GetUsedMemory() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.usedMemory
}

func (m *Manager) GetStats() (allocCount, deallocCount int64, usedMemory int64) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.allocCount, m.deallocCount, m.usedMemory
}

func (m *Manager) GetActiveMatCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.activeMats)
}

func (m *Manager) monitorMemory() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.performMonitoringCheck()
		case <-m.ctx.Done():
			return
		}
	}
}

func (m *Manager) performMonitoringCheck() {
	alloc, dealloc, used := m.GetStats()
	activeCount := m.GetActiveMatCount()

	m.logger.Debug("MemoryManager", "memory statistics", map[string]interface{}{
		"allocations":   alloc,
		"deallocations": dealloc,
		"used_bytes":    used,
		"active_mats":   activeCount,
	})

	if activeCount > 50 {
		m.logOldestMats(5)
	}

	if used > m.maxMemory*8/10 {
		runtime.GC()
	}
}

func (m *Manager) logOldestMats(count int) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.activeMats) == 0 {
		return
	}

	type matAge struct {
		info *MatInfo
		age  time.Duration
	}

	ages := make([]matAge, 0, len(m.activeMats))
	now := time.Now()

	for _, info := range m.activeMats {
		ages = append(ages, matAge{
			info: info,
			age:  now.Sub(info.Timestamp),
		})
	}

	for i := 0; i < len(ages)-1; i++ {
		for j := i + 1; j < len(ages); j++ {
			if ages[i].age < ages[j].age {
				ages[i], ages[j] = ages[j], ages[i]
			}
		}
	}

	limit := count
	if len(ages) < limit {
		limit = len(ages)
	}

	for i := 0; i < limit; i++ {
		mat := ages[i]
		m.logger.Warning("MemoryManager", "long-lived Mat detected", map[string]interface{}{
			"tag":  mat.info.Tag,
			"size": mat.info.Size,
			"age":  mat.age.String(),
		})
	}
}

func (m *Manager) Shutdown() {
	m.cancel()
	m.Cleanup()
}

func (m *Manager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	matCount := len(m.activeMats)
	for id, info := range m.activeMats {
		m.logger.Warning("MemoryManager", "cleaning up unreleased Mat", map[string]interface{}{
			"tag":  info.Tag,
			"size": info.Size,
		})
		delete(m.activeMats, id)
	}

	m.logger.Info("MemoryManager", "cleanup completed", map[string]interface{}{
		"mats_cleaned": matCount,
	})

	m.usedMemory = 0
	runtime.GC()
}

func getMatTypeSize(matType gocv.MatType) int {
	switch matType {
	case gocv.MatTypeCV8UC1:
		return 1
	case gocv.MatTypeCV8UC3:
		return 3
	case gocv.MatTypeCV8UC4:
		return 4
	case gocv.MatTypeCV16UC1:
		return 2
	case gocv.MatTypeCV16UC3:
		return 6
	case gocv.MatTypeCV16UC4:
		return 8
	case gocv.MatTypeCV32FC1:
		return 4
	case gocv.MatTypeCV32FC3:
		return 12
	case gocv.MatTypeCV32FC4:
		return 16
	default:
		return 1
	}
}
