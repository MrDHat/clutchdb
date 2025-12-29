package wal

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"os"

	"github.com/mrdhat/clutchdb/command"
)

/*
*

	WAL is a Write-Ahead Logging interface
	It is used to persist commands to disk before applying them to memory

	Record Format:
	┌───────────────────────────────────────┐
	│ uint32  record_length                 │  (bytes after this field)
	├───────────────────────────────────────┤
	│ uint32  crc32                         │  (of payload only)
	├───────────────────────────────────────┤
	│ uint8   command_type                  │
	├───────────────────────────────────────┤
	│ [16]byte request_id                   │
	├───────────────────────────────────────┤
	│ uint16  lock_id_length                │
	│ []byte  lock_id                       │
	├───────────────────────────────────────┤
	│ uint16  owner_id_length               │
	│ []byte  owner_id                      │
	├───────────────────────────────────────┤
	│ uint64  ttl_millis                    │
	├───────────────────────────────────────┤
	│ uint64  commit_unix_millis            │
	├───────────────────────────────────────┤
	│ uint64  fencing_token                 │
	├───────────────────────────────────────┤

*
*/
type WAL interface {
	Append(cmd command.Command) error
	Sync() error
	ReadAll() ([]command.Command, error)
}

type wal struct {
	file *os.File
}

func (w *wal) Append(cmd command.Command) error {

	// Serialize the payload (everything except record_length and crc32)
	payload := new(bytes.Buffer)

	// command_type (uint8)
	binary.Write(payload, binary.BigEndian, uint8(cmd.Type))

	// request_id ([16]byte)
	binary.Write(payload, binary.BigEndian, cmd.RequestID)

	// lock_id_length (uint16) + lock_id ([]byte)
	binary.Write(payload, binary.BigEndian, uint16(len(cmd.LockID)))
	payload.WriteString(cmd.LockID)

	// owner_id_length (uint16) + owner_id ([]byte)
	binary.Write(payload, binary.BigEndian, uint16(len(cmd.OwnerID)))
	payload.WriteString(cmd.OwnerID)

	// ttl_millis (uint64)
	binary.Write(payload, binary.BigEndian, cmd.TTLMillis)

	// commit_unix_millis (uint64)
	binary.Write(payload, binary.BigEndian, cmd.CommitTimeMillis)

	// fencing_token (uint64)
	binary.Write(payload, binary.BigEndian, cmd.FencingToken)

	payloadBytes := payload.Bytes()

	// Calculate CRC32 of the payload
	checksum := crc32.ChecksumIEEE(payloadBytes)

	// Build the final record: record_length + crc32 + payload
	finalRecord := new(bytes.Buffer)

	// record_length (uint32) - length of crc32 + payload
	recordLength := uint32(4 + len(payloadBytes)) // 4 bytes for crc32
	binary.Write(finalRecord, binary.BigEndian, recordLength)

	// crc32 (uint32)
	binary.Write(finalRecord, binary.BigEndian, checksum)

	// payload
	finalRecord.Write(payloadBytes)

	// Write to file
	_, err := w.file.Write(finalRecord.Bytes())
	return err
}

func (w *wal) Sync() error {
	return w.file.Sync()
}

func (w *wal) ReadAll() ([]command.Command, error) {
	// Seek to the beginning of the file
	if _, err := w.file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to seek to start: %w", err)
	}

	var commands []command.Command

	for {
		var recordLength uint32
		err := binary.Read(w.file, binary.BigEndian, &recordLength)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read record length: %w", err)
		}

		// Read the entire record (CRC32 + Payload)
		data := make([]byte, recordLength)
		if _, err := io.ReadFull(w.file, data); err != nil {
			return nil, fmt.Errorf("failed to read record data: %w", err)
		}

		// Extract CRC32
		expectedCRC := binary.BigEndian.Uint32(data[0:4])

		// Extract Payload
		payloadBytes := data[4:]

		// Verify CRC32
		actualCRC := crc32.ChecksumIEEE(payloadBytes)
		if actualCRC != expectedCRC {
			return nil, fmt.Errorf("checksum mismatch: expected %d, got %d", expectedCRC, actualCRC)
		}

		// Parse Payload
		payload := bytes.NewReader(payloadBytes)
		var cmd command.Command

		// command_type
		var cmdType uint8
		if err := binary.Read(payload, binary.BigEndian, &cmdType); err != nil {
			return nil, fmt.Errorf("failed to read command type: %w", err)
		}
		cmd.Type = command.CommandType(cmdType)

		// request_id
		if _, err := io.ReadFull(payload, cmd.RequestID[:]); err != nil {
			return nil, fmt.Errorf("failed to read request id: %w", err)
		}

		// lock_id
		var lockIDLen uint16
		if err := binary.Read(payload, binary.BigEndian, &lockIDLen); err != nil {
			return nil, fmt.Errorf("failed to read lock id length: %w", err)
		}
		lockID := make([]byte, lockIDLen)
		if _, err := io.ReadFull(payload, lockID); err != nil {
			return nil, fmt.Errorf("failed to read lock id: %w", err)
		}
		cmd.LockID = string(lockID)

		// owner_id
		var ownerIDLen uint16
		if err := binary.Read(payload, binary.BigEndian, &ownerIDLen); err != nil {
			return nil, fmt.Errorf("failed to read owner id length: %w", err)
		}
		ownerID := make([]byte, ownerIDLen)
		if _, err := io.ReadFull(payload, ownerID); err != nil {
			return nil, fmt.Errorf("failed to read owner id: %w", err)
		}
		cmd.OwnerID = string(ownerID)

		// ttl_millis
		if err := binary.Read(payload, binary.BigEndian, &cmd.TTLMillis); err != nil {
			return nil, fmt.Errorf("failed to read ttl millis: %w", err)
		}

		// commit_unix_millis
		if err := binary.Read(payload, binary.BigEndian, &cmd.CommitTimeMillis); err != nil {
			return nil, fmt.Errorf("failed to read commit millis: %w", err)
		}

		// fencing_token
		if err := binary.Read(payload, binary.BigEndian, &cmd.FencingToken); err != nil {
			return nil, fmt.Errorf("failed to read fencing token: %w", err)
		}

		commands = append(commands, cmd)
	}

	return commands, nil
}

func NewWAL(file *os.File) WAL {
	return &wal{file: file}
}
