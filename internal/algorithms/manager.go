package algorithms

import (
	"context"
	"fmt"
	"sync"

	"otsu-obliterator/internal/algorithms/otsu2d"
	"otsu-obliterator/internal/algorithms/triclass"
	"otsu-obliterator/internal/opencv/safe"
)

// Algorithm defines the interface for image processing algorithms
type Algorithm interface {
	Process(input *safe.Mat, params map[string]interface{}) (*safe.Mat, error)
	ValidateParameters(params map[string]interface{}) error
	GetDefaultParameters() map[string]interface{}
	GetName() string
}

// ContextualAlgorithm extends Algorithm with context support for cancellation
type ContextualAlgorithm interface {
	Algorithm
	ProcessWithContext(ctx context.Context, input *safe.Mat, params map[string]interface{}) (*safe.Mat, error)
}

type Manager struct {
	algorithms       map[string]Algorithm
	currentAlgorithm string
	parameters       map[string]map[string]interface{}
	mu               sync.RWMutex
}

func NewManager() *Manager {
	manager := &Manager{
		algorithms:       make(map[string]Algorithm),
		currentAlgorithm: "2D Otsu",
		parameters:       make(map[string]map[string]interface{}),
	}

	manager.registerAlgorithms()
	manager.initializeDefaultParameters()

	return manager
}

func (m *Manager) registerAlgorithms() {
	otsu2dAlg := otsu2d.NewProcessor()
	triclassAlg := triclass.NewProcessor()

	m.algorithms[otsu2dAlg.GetName()] = otsu2dAlg
	m.algorithms[triclassAlg.GetName()] = triclassAlg
}

func (m *Manager) initializeDefaultParameters() {
	for name, algorithm := range m.algorithms {
		m.parameters[name] = algorithm.GetDefaultParameters()
	}
}

func (m *Manager) SetCurrentAlgorithm(algorithm string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.algorithms[algorithm]; !exists {
		return fmt.Errorf("unknown algorithm: %s", algorithm)
	}

	m.currentAlgorithm = algorithm
	return nil
}

func (m *Manager) GetCurrentAlgorithm() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentAlgorithm
}

func (m *Manager) GetParameters(algorithm string) map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if params, exists := m.parameters[algorithm]; exists {
		result := make(map[string]interface{})
		for k, v := range params {
			result[k] = v
		}
		return result
	}

	return make(map[string]interface{})
}

func (m *Manager) GetAllParameters(algorithm string) map[string]interface{} {
	return m.GetParameters(algorithm)
}

func (m *Manager) SetParameter(algorithm, name string, value interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if params, exists := m.parameters[algorithm]; exists {
		params[name] = value
		return nil
	}

	return fmt.Errorf("unknown algorithm: %s", algorithm)
}

func (m *Manager) GetAlgorithm(name string) (Algorithm, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if algorithm, exists := m.algorithms[name]; exists {
		return algorithm, nil
	}

	return nil, fmt.Errorf("unknown algorithm: %s", name)
}

func (m *Manager) GetAvailableAlgorithms() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	algorithms := make([]string, 0, len(m.algorithms))
	for name := range m.algorithms {
		algorithms = append(algorithms, name)
	}

	return algorithms
}
