package service

import (
	"fmt"
	"sync"
	"time"
)

var (
	transactionLocks = make(map[int64]*sync.Mutex)
	locksMutex       sync.RWMutex
	userLastAction   = make(map[int64]time.Time)
	actionMutex      sync.RWMutex
)

// AcquireTransactionLock acquires a lock for user transaction to prevent race conditions
func AcquireTransactionLock(userID int64) *sync.Mutex {
	locksMutex.Lock()
	defer locksMutex.Unlock()

	if _, exists := transactionLocks[userID]; !exists {
		transactionLocks[userID] = &sync.Mutex{}
	}

	lock := transactionLocks[userID]
	lock.Lock()
	return lock
}

// ReleaseTransactionLock releases the transaction lock
func ReleaseTransactionLock(lock *sync.Mutex) {
	lock.Unlock()
}

// CheckUserActionCooldown checks if user is in cooldown period
func CheckUserActionCooldown(userID int64, cooldownSeconds int) error {
	actionMutex.RLock()
	lastAction, exists := userLastAction[userID]
	actionMutex.RUnlock()

	if exists {
		elapsed := time.Since(lastAction)
		if elapsed < time.Duration(cooldownSeconds)*time.Second {
			remaining := time.Duration(cooldownSeconds)*time.Second - elapsed
			return fmt.Errorf("mohon tunggu %d detik lagi sebelum melakukan transaksi", int(remaining.Seconds())+1)
		}
	}

	return nil
}

// SetUserActionTime sets the last action time for user
func SetUserActionTime(userID int64) {
	actionMutex.Lock()
	userLastAction[userID] = time.Now()
	actionMutex.Unlock()
}

// CleanupOldLocks removes old unused locks (call periodically)
func CleanupOldLocks() {
	locksMutex.Lock()
	defer locksMutex.Unlock()

	// Clean up locks that haven't been used for more than 1 hour
	cutoff := time.Now().Add(-1 * time.Hour)

	actionMutex.Lock()
	for userID, lastTime := range userLastAction {
		if lastTime.Before(cutoff) {
			delete(userLastAction, userID)
			delete(transactionLocks, userID)
		}
	}
	actionMutex.Unlock()
}

// StartCleanupRoutine starts a goroutine to periodically clean up old locks
func StartCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(30 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			CleanupOldLocks()
		}
	}()
}
