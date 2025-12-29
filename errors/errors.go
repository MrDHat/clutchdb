package errors

// StatusCode represents response status codes
type StatusCode uint8

const (
	STATUS_SUCCESS         StatusCode = 0 // Success
	STATUS_LOCK_HELD       StatusCode = 1 // Lock already held (ACQUIRE failed)
	STATUS_INVALID_REQUEST StatusCode = 2 // Invalid request / malformed
	STATUS_NOT_LEADER      StatusCode = 3 // Not leader / redirect to leader
	STATUS_LOCK_EXPIRED    StatusCode = 4 // Lock expired (for RENEW/RELEASE)
	// 5+ reserved for future errors
)
