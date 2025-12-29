package server

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/mrdhat/clutchdb/clutcherrors"
)

// resetState clears all locks and fencing tokens for test isolation
func resetState() {
	ActiveLocks.Range(func(key, value any) bool {
		ActiveLocks.Delete(key)
		return true
	})
	FencingTokens.Range(func(key, value any) bool {
		FencingTokens.Delete(key)
		return true
	})
}

func TestAcquire(t *testing.T) {
	resetState()
	ctx := context.Background()
	ownerID := "owner1"
	lockID := "lock1"
	ttl := 100 * time.Millisecond

	status, lock, err := Acquire(ctx, ownerID, lockID, ttl)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}
	if status != clutcherrors.STATUS_SUCCESS {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_SUCCESS, status)
	}
	if lock == nil {
		t.Fatal("Expected lock, got nil")
	}
	if lock.OwnerID != ownerID {
		t.Errorf("Expected owner %s, got %s", ownerID, lock.OwnerID)
	}
	if lock.ID != lockID {
		t.Errorf("Expected lock ID %s, got %s", lockID, lock.ID)
	}
	if lock.FencingToken == 0 {
		t.Error("Expected non-zero fencing token")
	}
}

func TestAcquireConflict(t *testing.T) {
	resetState()
	ctx := context.Background()
	ownerID := "owner1"
	ownerID2 := "owner2"
	lockID := "lock1"
	ttl := 100 * time.Millisecond

	// First acquire should succeed
	status1, lock1, err1 := Acquire(ctx, ownerID, lockID, ttl)
	if err1 != nil {
		t.Fatalf("First Acquire failed: %v", err1)
	}
	if status1 != clutcherrors.STATUS_SUCCESS {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_SUCCESS, status1)
	}

	// Second acquire should fail
	status2, lock2, err2 := Acquire(ctx, ownerID2, lockID, ttl)
	if err2 == nil {
		t.Fatal("Expected error for second Acquire, got nil")
	}
	if status2 != clutcherrors.STATUS_LOCK_HELD {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_LOCK_HELD, status2)
	}
	if lock2 != nil {
		t.Error("Expected nil lock for failed acquire")
	}

	// First lock should still be valid
	if lock1.OwnerID != ownerID {
		t.Errorf("Expected owner %s, got %s", ownerID, lock1.OwnerID)
	}
}

func TestAcquireExpired(t *testing.T) {
	resetState()
	ctx := context.Background()
	ownerID := "owner1"
	ownerID2 := "owner2"
	lockID := "lock1"
	ttl := 10 * time.Millisecond // Very short TTL

	// First acquire
	status1, lock1, err1 := Acquire(ctx, ownerID, lockID, ttl)
	if err1 != nil {
		t.Fatalf("First Acquire failed: %v", err1)
	}
	if status1 != clutcherrors.STATUS_SUCCESS {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_SUCCESS, status1)
	}

	firstFencingToken := lock1.FencingToken

	// Wait for lock to expire
	time.Sleep(ttl + 10*time.Millisecond)

	// Second acquire should succeed (lock expired)
	status2, lock2, err2 := Acquire(ctx, ownerID2, lockID, ttl)
	if err2 != nil {
		t.Fatalf("Second Acquire failed: %v", err2)
	}
	if status2 != clutcherrors.STATUS_SUCCESS {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_SUCCESS, status2)
	}
	if lock2.OwnerID != ownerID2 {
		t.Errorf("Expected owner %s, got %s", ownerID2, lock2.OwnerID)
	}
	// Fencing token should be higher than first lock
	if lock2.FencingToken <= firstFencingToken {
		t.Errorf("Expected fencing token > %d, got %d", firstFencingToken, lock2.FencingToken)
	}
}

func TestRenew(t *testing.T) {
	resetState()
	ctx := context.Background()
	ownerID := "owner1"
	lockID := "lock1"
	ttl := 100 * time.Millisecond

	// First acquire
	status1, lock1, err1 := Acquire(ctx, ownerID, lockID, ttl)
	if err1 != nil {
		t.Fatalf("Acquire failed: %v", err1)
	}
	if status1 != clutcherrors.STATUS_SUCCESS {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_SUCCESS, status1)
	}

	originalExpiresAt := lock1.ExpiresAt
	originalFencingToken := lock1.FencingToken

	// Wait a bit then renew
	time.Sleep(10 * time.Millisecond)
	newTTL := 200 * time.Millisecond
	status2, lock2, err2 := Renew(ctx, ownerID, lockID, lock1.FencingToken, newTTL)
	if err2 != nil {
		t.Fatalf("Renew failed: %v", err2)
	}
	if status2 != clutcherrors.STATUS_SUCCESS {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_SUCCESS, status2)
	}

	// Should be same lock instance
	if lock2 != lock1 {
		t.Error("Expected same lock instance")
	}
	// ExpiresAt should be extended
	if lock2.ExpiresAt <= originalExpiresAt {
		t.Errorf("Expected expiresAt > %d, got %d", originalExpiresAt, lock2.ExpiresAt)
	}
	// Fencing token should not change on renew
	if lock2.FencingToken != originalFencingToken {
		t.Errorf("Expected fencing token %d, got %d", originalFencingToken, lock2.FencingToken)
	}
}

func TestRenewNotHeld(t *testing.T) {
	resetState()
	ctx := context.Background()
	ownerID := "owner1"
	lockID := "lock1"
	ttl := 100 * time.Millisecond

	status, lock, err := Renew(ctx, ownerID, lockID, 1, ttl)
	if err == nil {
		t.Fatal("Expected error for renewing non-held lock, got nil")
	}
	if status != clutcherrors.STATUS_LOCK_NOT_HELD {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_LOCK_NOT_HELD, status)
	}
	if lock != nil {
		t.Error("Expected nil lock for failed renew")
	}
}

func TestRenewExpired(t *testing.T) {
	resetState()
	ctx := context.Background()
	ownerID := "owner1"
	lockID := "lock1"
	ttl := 10 * time.Millisecond

	// First acquire
	status1, lock1, err1 := Acquire(ctx, ownerID, lockID, ttl)
	if err1 != nil {
		t.Fatalf("Acquire failed: %v", err1)
	}
	if status1 != clutcherrors.STATUS_SUCCESS {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_SUCCESS, status1)
	}

	// Wait for lock to expire
	time.Sleep(ttl + 10*time.Millisecond)

	// Try to renew expired lock
	status2, lock2, err2 := Renew(ctx, ownerID, lockID, lock1.FencingToken, ttl)
	if err2 == nil {
		t.Fatal("Expected error for renewing expired lock, got nil")
	}
	if status2 != clutcherrors.STATUS_LOCK_NOT_HELD {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_LOCK_NOT_HELD, status2)
	}
	if lock2 != nil {
		t.Error("Expected nil lock for failed renew")
	}
}

func TestRenewOwnerMismatch(t *testing.T) {
	resetState()
	ctx := context.Background()
	ownerID := "owner1"
	wrongOwnerID := "owner2"
	lockID := "lock1"
	ttl := 100 * time.Millisecond

	// First acquire
	status1, lock1, err1 := Acquire(ctx, ownerID, lockID, ttl)
	if err1 != nil {
		t.Fatalf("Acquire failed: %v", err1)
	}
	if status1 != clutcherrors.STATUS_SUCCESS {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_SUCCESS, status1)
	}

	// Try to renew with wrong owner
	status2, lock2, err2 := Renew(ctx, wrongOwnerID, lockID, lock1.FencingToken, ttl)
	if err2 == nil {
		t.Fatal("Expected error for renewing with wrong owner, got nil")
	}
	if status2 != clutcherrors.STATUS_LOCK_NOT_HELD {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_LOCK_NOT_HELD, status2)
	}
	if lock2 != nil {
		t.Error("Expected nil lock for failed renew")
	}
}

func TestRenewFencingTokenMismatch(t *testing.T) {
	resetState()
	ctx := context.Background()
	ownerID := "owner1"
	lockID := "lock1"
	ttl := 100 * time.Millisecond

	// First acquire
	status1, lock1, err1 := Acquire(ctx, ownerID, lockID, ttl)
	if err1 != nil {
		t.Fatalf("Acquire failed: %v", err1)
	}
	if status1 != clutcherrors.STATUS_SUCCESS {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_SUCCESS, status1)
	}

	// Try to renew with wrong fencing token
	status2, lock2, err2 := Renew(ctx, ownerID, lockID, lock1.FencingToken+1, ttl)
	if err2 == nil {
		t.Fatal("Expected error for renewing with wrong fencing token, got nil")
	}
	if status2 != clutcherrors.STATUS_LOCK_NOT_HELD {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_LOCK_NOT_HELD, status2)
	}
	if lock2 != nil {
		t.Error("Expected nil lock for failed renew")
	}
}

func TestRelease(t *testing.T) {
	resetState()
	ctx := context.Background()
	ownerID := "owner1"
	lockID := "lock1"
	ttl := 100 * time.Millisecond

	// First acquire
	status1, lock1, err1 := Acquire(ctx, ownerID, lockID, ttl)
	if err1 != nil {
		t.Fatalf("Acquire failed: %v", err1)
	}
	if status1 != clutcherrors.STATUS_SUCCESS {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_SUCCESS, status1)
	}

	// Release the lock
	status2, err2 := Release(ctx, lockID, ownerID, lock1.FencingToken)
	if err2 != nil {
		t.Fatalf("Release failed: %v", err2)
	}
	if status2 != clutcherrors.STATUS_SUCCESS {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_SUCCESS, status2)
	}

	// Verify lock is gone by trying to renew
	status3, lock3, err3 := Renew(ctx, ownerID, lockID, lock1.FencingToken, ttl)
	if err3 == nil {
		t.Fatal("Expected error for renewing released lock, got nil")
	}
	if status3 != clutcherrors.STATUS_LOCK_NOT_HELD {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_LOCK_NOT_HELD, status3)
	}
	if lock3 != nil {
		t.Error("Expected nil lock for failed renew")
	}
}

func TestReleaseNotHeld(t *testing.T) {
	resetState()
	ctx := context.Background()
	ownerID := "owner1"
	lockID := "lock1"

	status, err := Release(ctx, lockID, ownerID, 1)
	if err == nil {
		t.Fatal("Expected error for releasing non-held lock, got nil")
	}
	if status != clutcherrors.STATUS_LOCK_NOT_HELD {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_LOCK_NOT_HELD, status)
	}
}

func TestReleaseExpired(t *testing.T) {
	resetState()
	ctx := context.Background()
	ownerID := "owner1"
	lockID := "lock1"
	ttl := 10 * time.Millisecond

	// First acquire
	status1, lock1, err1 := Acquire(ctx, ownerID, lockID, ttl)
	if err1 != nil {
		t.Fatalf("Acquire failed: %v", err1)
	}
	if status1 != clutcherrors.STATUS_SUCCESS {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_SUCCESS, status1)
	}

	// Wait for lock to expire
	time.Sleep(ttl + 10*time.Millisecond)

	// Try to release expired lock
	status2, err2 := Release(ctx, lockID, ownerID, lock1.FencingToken)
	if err2 == nil {
		t.Fatal("Expected error for releasing expired lock, got nil")
	}
	if status2 != clutcherrors.STATUS_LOCK_NOT_HELD {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_LOCK_NOT_HELD, status2)
	}
}

func TestReleaseOwnerMismatch(t *testing.T) {
	resetState()
	ctx := context.Background()
	ownerID := "owner1"
	wrongOwnerID := "owner2"
	lockID := "lock1"
	ttl := 100 * time.Millisecond

	// First acquire
	status1, lock1, err1 := Acquire(ctx, ownerID, lockID, ttl)
	if err1 != nil {
		t.Fatalf("Acquire failed: %v", err1)
	}
	if status1 != clutcherrors.STATUS_SUCCESS {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_SUCCESS, status1)
	}

	// Try to release with wrong owner
	status2, err2 := Release(ctx, lockID, wrongOwnerID, lock1.FencingToken)
	if err2 == nil {
		t.Fatal("Expected error for releasing with wrong owner, got nil")
	}
	if status2 != clutcherrors.STATUS_LOCK_NOT_HELD {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_LOCK_NOT_HELD, status2)
	}
}

func TestReleaseFencingTokenMismatch(t *testing.T) {
	resetState()
	ctx := context.Background()
	ownerID := "owner1"
	lockID := "lock1"
	ttl := 100 * time.Millisecond

	// First acquire
	status1, lock1, err1 := Acquire(ctx, ownerID, lockID, ttl)
	if err1 != nil {
		t.Fatalf("Acquire failed: %v", err1)
	}
	if status1 != clutcherrors.STATUS_SUCCESS {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_SUCCESS, status1)
	}

	// Try to release with wrong fencing token
	status2, err2 := Release(ctx, lockID, ownerID, lock1.FencingToken+1)
	if err2 == nil {
		t.Fatal("Expected error for releasing with wrong fencing token, got nil")
	}
	if status2 != clutcherrors.STATUS_LOCK_NOT_HELD {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_LOCK_NOT_HELD, status2)
	}
}

func TestConcurrentAcquire(t *testing.T) {
	resetState()
	ctx := context.Background()
	lockID := "lock1"
	ttl := 100 * time.Millisecond
	numGoroutines := 10
	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	conflictCount := 0

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ownerID := "owner" + string(rune('A'+id))

			status, _, err := Acquire(ctx, ownerID, lockID, ttl)
			if err != nil {
				if status != clutcherrors.STATUS_LOCK_HELD {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			mu.Lock()
			if status == clutcherrors.STATUS_SUCCESS {
				successCount++
			} else {
				conflictCount++
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Only one should succeed
	if successCount != 1 {
		t.Errorf("Expected 1 success, got %d", successCount)
	}
	if conflictCount != numGoroutines-1 {
		t.Errorf("Expected %d conflicts, got %d", numGoroutines-1, conflictCount)
	}
}

func TestFencingTokenIncrement(t *testing.T) {
	resetState()
	ctx := context.Background()
	ownerID := "owner1"
	lockID := "lock1"
	ttl := 100 * time.Millisecond

	// First acquire
	status1, lock1, err1 := Acquire(ctx, ownerID, lockID, ttl)
	if err1 != nil {
		t.Fatalf("First Acquire failed: %v", err1)
	}
	if status1 != clutcherrors.STATUS_SUCCESS {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_SUCCESS, status1)
	}

	firstToken := lock1.FencingToken

	// Release and acquire again
	status2, err2 := Release(ctx, lockID, ownerID, lock1.FencingToken)
	if err2 != nil {
		t.Fatalf("Release failed: %v", err2)
	}
	if status2 != clutcherrors.STATUS_SUCCESS {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_SUCCESS, status2)
	}

	// Second acquire
	status3, lock3, err3 := Acquire(ctx, ownerID, lockID, ttl)
	if err3 != nil {
		t.Fatalf("Second Acquire failed: %v", err3)
	}
	if status3 != clutcherrors.STATUS_SUCCESS {
		t.Errorf("Expected status %d, got %d", clutcherrors.STATUS_SUCCESS, status3)
	}

	// Fencing token should be incremented
	if lock3.FencingToken <= firstToken {
		t.Errorf("Expected fencing token > %d, got %d", firstToken, lock3.FencingToken)
	}
}
