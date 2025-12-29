package clutcherrors

// StatusCode represents response status codes
type StatusCode uint8

const (
	STATUS_SUCCESS         StatusCode = 0 // Success
	STATUS_LOCK_HELD       StatusCode = 1 // Lock already held (ACQUIRE failed)
	STATUS_LOCK_NOT_HELD   StatusCode = 2 // Lock not held (for RENEW)
	STATUS_INVALID_REQUEST StatusCode = 3 // Invalid request / malformed
	STATUS_NOT_LEADER      StatusCode = 4 // Not leader / redirect to leader
	STATUS_LOCK_EXPIRED    StatusCode = 5 // Lock expired (for RENEW/RELEASE)
	// 6+ reserved for future errors
)
