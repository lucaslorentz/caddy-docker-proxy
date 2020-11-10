package plugin

import (
	"sync"
)

// StringBoolCMap is a concurrent map implementation of map[string]bool
type StringBoolCMap struct {
	mutex    sync.RWMutex
	internal map[string]bool
}

func newStringBoolCMap() *StringBoolCMap {
	return &StringBoolCMap{
		mutex:    sync.RWMutex{},
		internal: map[string]bool{},
	}
}

// Set map value
func (m *StringBoolCMap) Set(key string, value bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.internal[key] = value
}

// Get map value or default
func (m *StringBoolCMap) Get(key string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.internal[key]
}

// Delete map value
func (m *StringBoolCMap) Delete(key string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.internal, key)
}
