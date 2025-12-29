package protocol

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mrdhat/clutchdb/errors"
)

func TestRequestRoundTrip(t *testing.T) {
	// Test round-trip encoding/decoding for Request
	lockID := [16]byte{}
	copy(lockID[:], "mylock") // Copy string bytes directly (padded with zeros)

	requestUUID := uuid.New()
	requestID := [16]byte{}
	copy(requestID[:], requestUUID[:])

	ownerUUID := uuid.New()
	ownerID := [16]byte{}
	copy(ownerID[:], ownerUUID[:])

	original := &Request{
		Cmd:       ACQUIRE,
		RequestID: requestID,
		LockID:    lockID,
		OwnerID:   ownerID,
		TTLMS:     1000,
	}

	var buf bytes.Buffer
	if err := WriteRequest(&buf, original); err != nil {
		t.Fatalf("WriteRequest failed: %v", err)
	}

	decoded, err := ReadRequest(&buf)
	if err != nil {
		t.Fatalf("ReadRequest failed: %v", err)
	}

	if decoded.Cmd != original.Cmd {
		t.Errorf("Cmd mismatch: got %d, want %d", decoded.Cmd, original.Cmd)
	}
	if decoded.RequestID != original.RequestID {
		t.Errorf("RequestID mismatch")
	}
	if decoded.LockID != original.LockID {
		t.Errorf("LockID mismatch")
	}
	if decoded.OwnerID != original.OwnerID {
		t.Errorf("OwnerID mismatch")
	}
	if decoded.TTLMS != original.TTLMS {
		t.Errorf("TTLMS mismatch: got %d, want %d", decoded.TTLMS, original.TTLMS)
	}
}

func TestResponseRoundTrip(t *testing.T) {
	// Test round-trip encoding/decoding for Response
	expiresAt := time.Now().UnixMilli() + 10000 // 10 seconds from now in milliseconds

	original := &Response{
		Status:       0,
		FencingToken: 12345,
		ExpiresAt:    uint64(expiresAt),
	}

	var buf bytes.Buffer
	if err := WriteResponse(&buf, original); err != nil {
		t.Fatalf("WriteResponse failed: %v", err)
	}

	decoded, err := ReadResponse(&buf)
	if err != nil {
		t.Fatalf("ReadResponse failed: %v", err)
	}

	if decoded.Status != original.Status {
		t.Errorf("Status mismatch: got %d, want %d", decoded.Status, original.Status)
	}
	if decoded.FencingToken != original.FencingToken {
		t.Errorf("FencingToken mismatch: got %d, want %d", decoded.FencingToken, original.FencingToken)
	}
	if decoded.ExpiresAt != original.ExpiresAt {
		t.Errorf("ExpiresAt mismatch: got %d, want %d", decoded.ExpiresAt, original.ExpiresAt)
	}
}

func TestAllCommands(t *testing.T) {
	// Test that different commands can be encoded/decoded with appropriate inputs
	testCases := []struct {
		name string
		cmd  uint8
		ttl  uint64
	}{
		{"ACQUIRE", ACQUIRE, 1000}, // ACQUIRE requires TTLMS
		{"RENEW", RENEW, 2000},     // RENEW requires TTLMS
		{"RELEASE", RELEASE, 0},    // RELEASE ignores TTLMS (set to 0)
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lockID := [16]byte{}
			copy(lockID[:], "testlock") // Copy string bytes directly

			requestUUID := uuid.New()
			requestID := [16]byte{}
			copy(requestID[:], requestUUID[:])

			ownerUUID := uuid.New()
			ownerID := [16]byte{}
			copy(ownerID[:], ownerUUID[:])

			req := &Request{
				Cmd:       tc.cmd,
				RequestID: requestID,
				LockID:    lockID,
				OwnerID:   ownerID,
				TTLMS:     tc.ttl,
			}

			var buf bytes.Buffer
			if err := WriteRequest(&buf, req); err != nil {
				t.Fatalf("WriteRequest failed: %v", err)
			}

			decoded, err := ReadRequest(&buf)
			if err != nil {
				t.Fatalf("ReadRequest failed: %v", err)
			}

			if decoded.Cmd != tc.cmd {
				t.Errorf("Cmd mismatch: got %d, want %d", decoded.Cmd, tc.cmd)
			}
			if decoded.TTLMS != tc.ttl {
				t.Errorf("TTLMS mismatch: got %d, want %d", decoded.TTLMS, tc.ttl)
			}
		})
	}
}

func TestReadRequestOrErrorResponse(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		lockID := [16]byte{}
		copy(lockID[:], "testlock")

		requestUUID := uuid.New()
		requestID := [16]byte{}
		copy(requestID[:], requestUUID[:])

		ownerUUID := uuid.New()
		ownerID := [16]byte{}
		copy(ownerID[:], ownerUUID[:])

		original := &Request{
			Cmd:       ACQUIRE,
			RequestID: requestID,
			LockID:    lockID,
			OwnerID:   ownerID,
			TTLMS:     1000,
		}

		var buf bytes.Buffer
		if err := WriteRequest(&buf, original); err != nil {
			t.Fatalf("WriteRequest failed: %v", err)
		}

		req, errResp := ReadRequestOrErrorResponse(&buf)
		if errResp != nil {
			t.Errorf("Expected no error response, got status %d", errResp.Status)
		}
		if req == nil {
			t.Fatal("Expected request, got nil")
		}
		if req.Cmd != ACQUIRE {
			t.Errorf("Cmd mismatch: got %d, want %d", req.Cmd, ACQUIRE)
		}
	})

	t.Run("invalid request - wrong length", func(t *testing.T) {
		// Write invalid data with wrong length
		var buf bytes.Buffer
		binary.Write(&buf, binary.BigEndian, uint32(999)) // Wrong length
		buf.Write([]byte{1})                              // cmd

		req, errResp := ReadRequestOrErrorResponse(&buf)
		if req != nil {
			t.Errorf("Expected nil request, got %v", req)
		}
		if errResp == nil {
			t.Fatal("Expected error response, got nil")
		}
		if errResp.Status != errors.STATUS_INVALID_REQUEST {
			t.Errorf("Expected status %d, got %d", errors.STATUS_INVALID_REQUEST, errResp.Status)
		}
	})
}
