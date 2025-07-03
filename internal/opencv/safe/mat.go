package safe

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"gocv.io/x/gocv"
)

type MemoryTracker interface {
	TrackAllocation(ptr uintptr, size int64, tag string)
	TrackDeallocation(ptr uintptr, tag string)
}

type Mat struct {
	mat        gocv.Mat
	isValid    int32
	refCount   int32
	mu         sync.RWMutex
	id         uint64
	memTracker MemoryTracker
	tag        string
}

var (
	nextMatID uint64
	matPool   = sync.Pool{
		New: func() interface{} {
			return &Mat{}
		},
	}
)

func NewMat(rows, cols int, matType gocv.MatType) (*Mat, error) {
	return NewMatWithTracker(rows, cols, matType, nil, "")
}

func NewMatWithTracker(rows, cols int, matType gocv.MatType, memTracker MemoryTracker, tag string) (*Mat, error) {
	if err := validateDimensions(rows, cols); err != nil {
		return nil, err
	}

	mat := gocv.NewMatWithSize(rows, cols, matType)
	if mat.Empty() {
		mat.Close()
		return nil, fmt.Errorf("failed to create Mat with size %dx%d", cols, rows)
	}

	safeMat := matPool.Get().(*Mat)
	*safeMat = Mat{
		mat:        mat,
		isValid:    1,
		refCount:   1,
		id:         atomic.AddUint64(&nextMatID, 1),
		memTracker: memTracker,
		tag:        tag,
	}

	if memTracker != nil {
		size := int64(rows * cols * getMatTypeSize(matType))
		ptr := uintptr(unsafe.Pointer(&mat))
		memTracker.TrackAllocation(ptr, size, tag)
	}

	runtime.SetFinalizer(safeMat, (*Mat).finalize)
	return safeMat, nil
}

func NewMatFromMat(srcMat gocv.Mat) (*Mat, error) {
	return NewMatFromMatWithTracker(srcMat, nil, "")
}

func NewMatFromMatWithTracker(srcMat gocv.Mat, memTracker MemoryTracker, tag string) (*Mat, error) {
	if err := validateSourceMat(srcMat); err != nil {
		return nil, err
	}

	clonedMat := srcMat.Clone()
	if clonedMat.Empty() {
		clonedMat.Close()
		return nil, fmt.Errorf("failed to clone Mat")
	}

	safeMat := matPool.Get().(*Mat)
	*safeMat = Mat{
		mat:        clonedMat,
		isValid:    1,
		refCount:   1,
		id:         atomic.AddUint64(&nextMatID, 1),
		memTracker: memTracker,
		tag:        tag,
	}

	if memTracker != nil {
		size := int64(srcMat.Rows() * srcMat.Cols() * getMatTypeSize(srcMat.Type()))
		ptr := uintptr(unsafe.Pointer(&clonedMat))
		memTracker.TrackAllocation(ptr, size, tag)
	}

	runtime.SetFinalizer(safeMat, (*Mat).finalize)
	return safeMat, nil
}

func (sm *Mat) IsValid() bool {
	return atomic.LoadInt32(&sm.isValid) == 1
}

func (sm *Mat) Empty() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.IsValid() {
		return true
	}
	return sm.mat.Empty()
}

func (sm *Mat) Rows() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.IsValid() {
		return 0
	}
	return sm.mat.Rows()
}

func (sm *Mat) Cols() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.IsValid() {
		return 0
	}
	return sm.mat.Cols()
}

func (sm *Mat) Channels() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.IsValid() {
		return 0
	}
	return sm.mat.Channels()
}

func (sm *Mat) Type() gocv.MatType {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.IsValid() {
		return gocv.MatTypeCV8UC1
	}
	return sm.mat.Type()
}

func (sm *Mat) Clone() (*Mat, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.IsValid() {
		return nil, fmt.Errorf("cannot clone invalid Mat")
	}

	if sm.mat.Empty() {
		return nil, fmt.Errorf("cannot clone empty Mat")
	}

	return NewMatFromMatWithTracker(sm.mat, sm.memTracker, sm.tag+"_clone")
}

func (sm *Mat) CopyTo(dst *Mat) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.IsValid() {
		return fmt.Errorf("source Mat is invalid")
	}

	dst.mu.Lock()
	defer dst.mu.Unlock()

	if !dst.IsValid() {
		return fmt.Errorf("destination Mat is invalid")
	}

	if sm.mat.Empty() {
		return fmt.Errorf("source Mat is empty")
	}

	sm.mat.CopyTo(&dst.mat)
	return nil
}

func (sm *Mat) GetUCharAt(row, col int) (uint8, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if err := sm.validateCoordinates(row, col); err != nil {
		return 0, err
	}

	return sm.mat.GetUCharAt(row, col), nil
}

func (sm *Mat) SetUCharAt(row, col int, value uint8) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if err := sm.validateCoordinates(row, col); err != nil {
		return err
	}

	sm.mat.SetUCharAt(row, col, value)
	return nil
}

func (sm *Mat) GetUCharAt3(row, col, channel int) (uint8, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if err := sm.validateCoordinatesAndChannel(row, col, channel); err != nil {
		return 0, err
	}

	return sm.mat.GetUCharAt3(row, col, channel), nil
}

func (sm *Mat) SetUCharAt3(row, col, channel int, value uint8) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if err := sm.validateCoordinatesAndChannel(row, col, channel); err != nil {
		return err
	}

	sm.mat.SetUCharAt3(row, col, channel, value)
	return nil
}

func (sm *Mat) GetMat() gocv.Mat {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.mat
}

func (sm *Mat) ID() uint64 {
	return sm.id
}

func (sm *Mat) AddRef() {
	atomic.AddInt32(&sm.refCount, 1)
}

func (sm *Mat) Release() {
	if atomic.AddInt32(&sm.refCount, -1) == 0 {
		sm.Close()
	}
}

func (sm *Mat) Reset() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.mat.Empty() {
		sm.mat.Close()
	}
	sm.mat = gocv.Mat{}
	atomic.StoreInt32(&sm.isValid, 0)
	atomic.StoreInt32(&sm.refCount, 0)
	sm.memTracker = nil
	sm.tag = ""
	sm.id = 0
}

func (sm *Mat) Close() {
	if !atomic.CompareAndSwapInt32(&sm.isValid, 1, 0) {
		return
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sm.memTracker != nil {
		ptr := uintptr(unsafe.Pointer(&sm.mat))
		sm.memTracker.TrackDeallocation(ptr, sm.tag)
	}

	if !sm.mat.Empty() {
		sm.mat.Close()
	}

	runtime.SetFinalizer(sm, nil)
	sm.mat = gocv.Mat{}
	sm.memTracker = nil
	sm.tag = ""
	sm.refCount = 0
	sm.id = 0

	matPool.Put(sm)
}

func (sm *Mat) finalize() {
	if atomic.LoadInt32(&sm.isValid) == 1 {
		sm.Close()
	}
}

func (sm *Mat) validateCoordinates(row, col int) error {
	if !sm.IsValid() {
		return fmt.Errorf("Mat is invalid")
	}

	if row < 0 || row >= sm.mat.Rows() || col < 0 || col >= sm.mat.Cols() {
		return fmt.Errorf("coordinates out of bounds: (%d,%d) for size %dx%d",
			col, row, sm.mat.Cols(), sm.mat.Rows())
	}

	return nil
}

func (sm *Mat) validateCoordinatesAndChannel(row, col, channel int) error {
	if err := sm.validateCoordinates(row, col); err != nil {
		return err
	}

	if channel < 0 || channel >= sm.mat.Channels() {
		return fmt.Errorf("channel out of bounds: %d for %d channels", channel, sm.mat.Channels())
	}

	return nil
}

func validateDimensions(rows, cols int) error {
	if rows <= 0 || cols <= 0 {
		return fmt.Errorf("invalid dimensions: %dx%d", cols, rows)
	}

	if rows > 32768 || cols > 32768 {
		return fmt.Errorf("dimensions %dx%d exceed maximum size", cols, rows)
	}

	return nil
}

func validateSourceMat(srcMat gocv.Mat) error {
	if srcMat.Empty() {
		return fmt.Errorf("source Mat is empty")
	}

	if srcMat.Rows() <= 0 || srcMat.Cols() <= 0 {
		return fmt.Errorf("source Mat has invalid dimensions: %dx%d", srcMat.Cols(), srcMat.Rows())
	}

	return nil
}

func ValidateMatForOperation(mat *Mat, operation string) error {
	if mat == nil {
		return fmt.Errorf("Mat is nil for operation: %s", operation)
	}

	if !mat.IsValid() {
		return fmt.Errorf("Mat is invalid for operation: %s", operation)
	}

	if mat.Empty() {
		return fmt.Errorf("Mat is empty for operation: %s", operation)
	}

	if mat.Rows() <= 0 || mat.Cols() <= 0 {
		return fmt.Errorf("Mat has invalid dimensions %dx%d for operation: %s",
			mat.Cols(), mat.Rows(), operation)
	}

	return nil
}

func ValidateColorConversion(src *Mat, code gocv.ColorConversionCode) error {
	if err := ValidateMatForOperation(src, "CvtColor"); err != nil {
		return err
	}

	channels := src.Channels()

	switch code {
	case gocv.ColorBGRToGray, gocv.ColorRGBToGray:
		if channels != 3 {
			return fmt.Errorf("BGR/RGB to Gray conversion requires 3 channels, got %d", channels)
		}
	case gocv.ColorGrayToBGR:
		if channels != 1 {
			return fmt.Errorf("Gray to BGR conversion requires 1 channel, got %d", channels)
		}
	case gocv.ColorBGRToRGB:
		if channels != 3 {
			return fmt.Errorf("BGR/RGB conversion requires 3 channels, got %d", channels)
		}
	case gocv.ColorBGRToBGRA:
		if channels != 3 {
			return fmt.Errorf("BGR to BGRA conversion requires 3 channels, got %d", channels)
		}
	case gocv.ColorBGRAToBGR:
		if channels != 4 {
			return fmt.Errorf("BGRA to BGR conversion requires 4 channels, got %d", channels)
		}
	}

	return nil
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
