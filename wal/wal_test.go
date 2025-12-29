package wal

import (
	"os"
	"testing"

	"github.com/mrdhat/clutchdb/command"
)

func TestWAL(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "wal_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	w := NewWAL(tmpFile)

	cmd1 := command.Command{
		Type:             command.CmdAcquire,
		RequestID:        [16]byte{1, 2, 3},
		LockID:           "lock1",
		OwnerID:          "owner1",
		TTLMillis:        1000,
		FencingToken:     123,
		CommitTimeMillis: 1678900000,
	}

	cmd2 := command.Command{
		Type:             command.CmdRelease,
		RequestID:        [16]byte{4, 5, 6},
		LockID:           "lock2",
		OwnerID:          "owner2",
		TTLMillis:        2000,
		FencingToken:     456,
		CommitTimeMillis: 1678900100,
	}

	if err := w.Append(cmd1); err != nil {
		t.Fatalf("failed to append cmd1: %v", err)
	}
	if err := w.Append(cmd2); err != nil {
		t.Fatalf("failed to append cmd2: %v", err)
	}

	if err := w.Sync(); err != nil {
		t.Fatalf("failed to sync: %v", err)
	}

	cmds, err := w.ReadAll()
	if err != nil {
		t.Fatalf("failed to read all: %v", err)
	}

	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(cmds))
	}

	// Verify cmd1
	if cmds[0].LockID != cmd1.LockID || cmds[0].Type != cmd1.Type || cmds[0].CommitTimeMillis != cmd1.CommitTimeMillis {
		t.Errorf("cmd1 mismatch: %+v", cmds[0])
	}

	// Verify cmd2
	if cmds[1].LockID != cmd2.LockID || cmds[1].Type != cmd2.Type || cmds[1].CommitTimeMillis != cmd2.CommitTimeMillis {
		t.Errorf("cmd2 mismatch: %+v", cmds[1])
	}
}
