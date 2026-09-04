package bot

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// UserActionType represents the pending input expected from a user.
type UserActionType string

const (
	ActionWaitingDepositAmount UserActionType = "WAITING_DEPOSIT_AMOUNT"
	ActionWaitingQuantity      UserActionType = "WAITING_QUANTITY"
)

// UserState stores the pending interaction state for a Telegram user.
type UserState struct {
	Action    UserActionType
	ItemID    uuid.UUID
	ProductID uuid.UUID
	ItemName  string
	Stock     int
	CreatedAt time.Time
}

// StateManager is a thread-safe in-memory store for pending user actions.
type StateManager struct {
	mu     sync.RWMutex
	states map[int64]UserState
}

// NewStateManager creates a new StateManager.
func NewStateManager() *StateManager {
	return &StateManager{
		states: make(map[int64]UserState),
	}
}

// Set saves state for a user.
func (m *StateManager) Set(telegramID int64, state UserState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	state.CreatedAt = time.Now()
	m.states[telegramID] = state
}

// Get retrieves state for a user, returning false if expired (>15 mins) or not found.
func (m *StateManager) Get(telegramID int64) (UserState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	st, ok := m.states[telegramID]
	if !ok {
		return UserState{}, false
	}
	if time.Since(st.CreatedAt) > 15*time.Minute {
		return UserState{}, false
	}
	return st, true
}

// Clear removes state for a user.
func (m *StateManager) Clear(telegramID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.states, telegramID)
}

// ParseAmount parses flexible Vietnamese currency representations into int64.
// Supports: "50000", "50.000", "50,000", "50k", "50K", "150k", "1.5tr", "2tr".
func ParseAmount(input string) (int64, error) {
	s := strings.TrimSpace(strings.ToLower(input))
	s = strings.TrimSuffix(s, "đ")
	s = strings.TrimSuffix(s, "vnd")
	s = strings.TrimSuffix(s, "vnđ")
	s = strings.TrimSpace(s)

	isK := strings.HasSuffix(s, "k")
	isM := strings.HasSuffix(s, "m") || strings.HasSuffix(s, "tr")
	if isK {
		s = strings.TrimSuffix(s, "k")
	} else if isM {
		s = strings.TrimSuffix(s, "tr")
		s = strings.TrimSuffix(s, "m")
	}
	s = strings.TrimSpace(s)

	if isK || isM {
		s = strings.ReplaceAll(s, ",", ".")
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0, err
		}
		if isK {
			return int64(f * 1000), nil
		}
		return int64(f * 1000000), nil
	}

	// Plain number with possible thousand separators (dots or commas)
	s = strings.ReplaceAll(s, ".", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)

	return strconv.ParseInt(s, 10, 64)
}
