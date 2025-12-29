package server

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mrdhat/clutchdb/clutcherrors"
)

var (
	FencingTokens sync.Map
	ActiveLocks   sync.Map
)

type Lock struct {
	ID           string
	OwnerID      string
	FencingToken uint64
	ExpiresAt    uint64
	mu           sync.Mutex
}

func Acquire(ctx context.Context, ownerID string, lockID string, ttl time.Duration) (clutcherrors.StatusCode, *Lock, error) {
	now := uint64(time.Now().UnixMilli())

	lockIface, loaded := ActiveLocks.LoadOrStore(lockID, &Lock{ID: lockID})
	lock := lockIface.(*Lock)

	lock.mu.Lock()
	defer lock.mu.Unlock()

	if loaded {
		if lock.ExpiresAt > now {
			// Lock is still valid, reject the acquire
			return clutcherrors.STATUS_LOCK_HELD, nil, errors.New("lock already held")
		}
		// Lock expired, allow re-acquire by reusing this lock object
	}

	// Increment fencing token atomically
	var zero uint64
	tokenPtrIface, _ := FencingTokens.LoadOrStore(lockID, &zero)
	tokenPtr := tokenPtrIface.(*uint64)
	fencingToken := atomic.AddUint64(tokenPtr, 1)

	lock.OwnerID = ownerID
	lock.FencingToken = fencingToken
	lock.ExpiresAt = now + uint64(ttl.Milliseconds())

	// TODO: persist lock & token
	return clutcherrors.STATUS_SUCCESS, lock, nil
}

func Renew(ctx context.Context, ownerID string, lockID string, fencingToken uint64, ttl time.Duration) (clutcherrors.StatusCode, *Lock, error) {
	now := uint64(time.Now().UnixMilli())

	lockIface, ok := ActiveLocks.Load(lockID)
	if !ok {
		return clutcherrors.STATUS_LOCK_NOT_HELD, nil, errors.New("lock not held")
	}
	lock := lockIface.(*Lock)

	lock.mu.Lock()
	defer lock.mu.Unlock()

	if lock.ExpiresAt < now {
		ActiveLocks.Delete(lockID)
		return clutcherrors.STATUS_LOCK_NOT_HELD, nil, errors.New("lock expired")
	}

	if lock.OwnerID != ownerID {
		return clutcherrors.STATUS_LOCK_NOT_HELD, nil, errors.New("owner mismatch")
	}

	if lock.FencingToken != fencingToken {
		return clutcherrors.STATUS_LOCK_NOT_HELD, nil, errors.New("fencing token mismatch")
	}

	lock.ExpiresAt = now + uint64(ttl.Milliseconds()) // TODO: in a distributed system, time can be a problem

	// TODO: persist lock

	return clutcherrors.STATUS_SUCCESS, lock, nil
}

func Release(ctx context.Context, lockID string, ownerID string, fencingToken uint64) (clutcherrors.StatusCode, error) {
	now := uint64(time.Now().UnixMilli())
	lockIface, ok := ActiveLocks.Load(lockID)
	if !ok {
		return clutcherrors.STATUS_LOCK_NOT_HELD, errors.New("lock not held")
	}
	lock := lockIface.(*Lock)

	lock.mu.Lock()
	defer lock.mu.Unlock()

	if lock.ExpiresAt < now {
		ActiveLocks.Delete(lockID)
		return clutcherrors.STATUS_LOCK_NOT_HELD, errors.New("lock expired")
	}

	if lock.OwnerID != ownerID {
		return clutcherrors.STATUS_LOCK_NOT_HELD, errors.New("owner mismatch")
	}

	if lock.FencingToken != fencingToken {
		return clutcherrors.STATUS_LOCK_NOT_HELD, errors.New("fencing token mismatch")
	}

	ActiveLocks.Delete(lockID)

	// TODO: persist lock

	return clutcherrors.STATUS_SUCCESS, nil
}
