package plugin

import (
	"sync"
)

// StringInt64CMap is a concurrent map implementation of map[string]int64
type StringInt64CMap struct {
	mutex    sync.RWMutex
	internal map[string]int64
}

func newStringInt64CMap() *StringInt64CMap {
	return &StringInt64CMap{
		mutex:    sync.RWMutex{},
		internal: map[string]int64{},
	}
}

// Set map value
func (m *StringInt64CMap) Set(key string, value int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.internal[key] = value
}

// Get map value or default
func (m *StringInt64CMap) Get(key string) int64 {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.internal[key]
}

// Delete map value
func (m *StringInt64CMap) Delete(key string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	delete(m.internal, key)
}
