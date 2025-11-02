package brightness

import (
	"testing"
	"time"
)

// Simple test to verify debounce timers are created and cleaned up
func TestManager_DebounceTimerCleanup(t *testing.T) {
	m := &Manager{
		subscribers:     make(map[string]chan State),
		debounceTimers:  make(map[string]*time.Timer),
		debouncePending: make(map[string]int),
		stopChan:        make(chan struct{}),
	}

	// Manually add some debounce state to test cleanup
	deviceID := "test:device"
	m.debouncePending[deviceID] = 50
	m.debounceTimers[deviceID] = time.NewTimer(100 * time.Millisecond)

	// Close the manager
	m.Close()

	// Verify timers and pending state are cleaned up
	if len(m.debounceTimers) != 0 {
		t.Errorf("Expected debounce timers to be cleared, got %d", len(m.debounceTimers))
	}

	if len(m.debouncePending) != 0 {
		t.Errorf("Expected pending debounce requests to be cleared, got %d", len(m.debouncePending))
	}
}
