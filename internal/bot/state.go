package bot

import (
	"log"
	"sync"
)

// UserState untuk tracking state user
type UserState struct {
	State       string // "waiting_phone", "waiting_otp", "verified", "waiting_admin_message", etc
	PhoneNumber string
	AuthID      string
	ProductCode string
	mu          sync.RWMutex
}

var userStates = make(map[int64]*UserState)
var statesMutex sync.RWMutex

func getUserState(chatID int64) *UserState {
	statesMutex.RLock()
	defer statesMutex.RUnlock()
	
	if state, exists := userStates[chatID]; exists {
		return state
	}
	
	// Create new state if doesn't exist
	state := &UserState{State: "start"}
	userStates[chatID] = state
	return state
}

func setUserState(chatID int64, state string) {
	statesMutex.Lock()
	defer statesMutex.Unlock()
	
	if userState, exists := userStates[chatID]; exists {
		userState.mu.Lock()
		userState.State = state
		userState.mu.Unlock()
	} else {
		userStates[chatID] = &UserState{State: state}
	}
}

func setUserData(chatID int64, phone, authID, productCode string) {
	statesMutex.Lock()
	defer statesMutex.Unlock()
	
	if userState, exists := userStates[chatID]; exists {
		userState.mu.Lock()
		if phone != "" {
			userState.PhoneNumber = phone
		}
		if authID != "" {
			userState.AuthID = authID
		}
		if productCode != "" {
			userState.ProductCode = productCode
		}
		userState.mu.Unlock()
	} else {
		// Create new state if doesn't exist
		userStates[chatID] = &UserState{
			State:       "start",
			PhoneNumber: phone,
			AuthID:      authID,
			ProductCode: productCode,
		}
	}
	
	// Debug log
	log.Printf("DEBUG setUserData - User %d: phone='%s', authID='%s', productCode='%s'", chatID, phone, authID, productCode)
}

func clearUserState(chatID int64) {
	statesMutex.Lock()
	defer statesMutex.Unlock()
	delete(userStates, chatID)
}