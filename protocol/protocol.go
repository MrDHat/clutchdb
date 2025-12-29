package protocol

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/mrdhat/clutchdb/errors"
)

// Command constants
const (
	ACQUIRE = 1 // Acquire lock
	RENEW   = 2 // Renew lock
	RELEASE = 3 // Release lock
)

// Request represents the wire protocol request
type Request struct {
	Cmd       uint8    // Command type (ACQUIRE, RENEW, RELEASE)
	RequestID [16]byte // Unique request identifier
	LockID    [16]byte // Lock identifier
	OwnerID   [16]byte // Owner/client identifier
	TTLMS     uint64   // Time-to-live in milliseconds (used by ACQUIRE and RENEW)
}

// Response represents the wire protocol response
type Response struct {
	Status       errors.StatusCode // Response status code
	FencingToken uint64            // Fencing token (used by ACQUIRE and RENEW)
	ExpiresAt    uint64            // Expiration timestamp in milliseconds (used by ACQUIRE and RENEW)
}

// WriteRequest encodes a Request to the wire format and writes it to w
func WriteRequest(w io.Writer, req *Request) error {
	var buf [61]byte

	binary.BigEndian.PutUint32(buf[0:4], 57)
	buf[4] = req.Cmd
	copy(buf[5:21], req.RequestID[:])
	copy(buf[21:37], req.LockID[:])
	copy(buf[37:53], req.OwnerID[:])
	binary.BigEndian.PutUint64(buf[53:61], req.TTLMS)

	_, err := w.Write(buf[:])
	return err
}

// ReadRequest reads from r and decodes into a Request
func ReadRequest(r io.Reader) (*Request, error) {
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	if length != 57 {
		return nil, fmt.Errorf("invalid request length: expected 57, got %d", length)
	}

	var data [57]byte
	if _, err := io.ReadFull(r, data[:]); err != nil {
		return nil, err
	}

	cmd := data[0]
	var requestID [16]byte
	copy(requestID[:], data[1:17])
	var lockID [16]byte
	copy(lockID[:], data[17:33])
	var ownerID [16]byte
	copy(ownerID[:], data[33:49])
	ttlMS := binary.BigEndian.Uint64(data[49:57])

	return &Request{
		Cmd:       cmd,
		RequestID: requestID,
		LockID:    lockID,
		OwnerID:   ownerID,
		TTLMS:     ttlMS,
	}, nil
}

// WriteResponse encodes a Response to the wire format and writes it to w
func WriteResponse(w io.Writer, resp *Response) error {
	var buf [17]byte

	buf[0] = byte(resp.Status)
	binary.BigEndian.PutUint64(buf[1:9], resp.FencingToken)
	binary.BigEndian.PutUint64(buf[9:17], resp.ExpiresAt)

	_, err := w.Write(buf[:])
	return err
}

// ReadResponse reads from r and decodes into a Response
func ReadResponse(r io.Reader) (*Response, error) {
	var buf [17]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return nil, err
	}

	status := errors.StatusCode(buf[0])
	fencingToken := binary.BigEndian.Uint64(buf[1:9])
	expiresAt := binary.BigEndian.Uint64(buf[9:17])

	return &Response{
		Status:       status,
		FencingToken: fencingToken,
		ExpiresAt:    expiresAt,
	}, nil
}

// ReadRequestOrErrorResponse attempts to read a Request, returning an error Response if malformed
func ReadRequestOrErrorResponse(r io.Reader) (*Request, *Response) {
	req, err := ReadRequest(r)
	if err != nil {
		return nil, &Response{
			Status:       errors.STATUS_INVALID_REQUEST,
			FencingToken: 0,
			ExpiresAt:    0,
		}
	}
	return req, nil
}
