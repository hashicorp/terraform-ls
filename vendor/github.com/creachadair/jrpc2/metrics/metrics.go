// Package metrics defines a concurrently-accessible metrics collector.
//
// A *metrics.M value exports methods to track integer counters and maximum
// values. A metric has a caller-assigned string name that is not interpreted
// by the collector except to locate its stored value.
package metrics

import "sync"

// An M collects counters and maximum value trackers.  A nil *M is valid, and
// discards all metrics. The methods of an *M are safe for concurrent use by
// multiple goroutines.
type M struct {
	mu      sync.Mutex
	counter map[string]int64
	maxVal  map[string]int64
	label   map[string]string
}

// New creates a new, empty metrics collector.
func New() *M {
	return &M{
		counter: make(map[string]int64),
		maxVal:  make(map[string]int64),
		label:   make(map[string]string),
	}
}

// Count adds n to the current value of the counter named, defining the counter
// if it does not already exist.
func (m *M) Count(name string, n int64) {
	if m != nil {
		m.mu.Lock()
		defer m.mu.Unlock()
		m.counter[name] += n
	}
}

// SetMaxValue sets the maximum value metric named to the greater of n and its
// current value, defining the value if it does not already exist.
func (m *M) SetMaxValue(name string, n int64) {
	if m != nil {
		m.mu.Lock()
		defer m.mu.Unlock()
		if old, ok := m.maxVal[name]; !ok || n > old {
			m.maxVal[name] = n
		}
	}
}

// CountAndSetMax adds n to the current value of the counter named, and also
// updates a max value tracker with the same name in a single step.
func (m *M) CountAndSetMax(name string, n int64) {
	if m != nil {
		m.mu.Lock()
		defer m.mu.Unlock()
		if old, ok := m.maxVal[name]; !ok || n > old {
			m.maxVal[name] = n
		}
		m.counter[name] += n
	}
}

// SetLabel sets the specified label to value. If value == "" the label is
// removed from the set.
func (m *M) SetLabel(name, value string) {
	if m != nil {
		m.mu.Lock()
		defer m.mu.Unlock()
		if value == "" {
			delete(m.label, name)
		} else {
			m.label[name] = value
		}
	}
}

// Snapshot copies an atomic snapshot of the collected metrics into the non-nil
// fields of the provided snapshot value. Only the fields of snap that are not
// nil are snapshotted.
func (m *M) Snapshot(snap Snapshot) {
	if m != nil {
		m.mu.Lock()
		defer m.mu.Unlock()
		if c := snap.Counter; c != nil {
			for name, val := range m.counter {
				c[name] = val
			}
		}
		if v := snap.MaxValue; v != nil {
			for name, val := range m.maxVal {
				v[name] = val
			}
		}
		if v := snap.Label; v != nil {
			for name, val := range m.label {
				v[name] = val
			}
		}
	}
}

// A Snapshot represents a point-in-time snapshot of a metrics collector.  The
// fields of this type are filled in by the Snapshot method of *M.
type Snapshot struct {
	Counter  map[string]int64
	MaxValue map[string]int64
	Label    map[string]string
}
