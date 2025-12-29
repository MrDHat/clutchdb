package command

type CommandType uint8

const (
	CmdAcquire CommandType = 1
	CmdRenew   CommandType = 2
	CmdRelease CommandType = 3
)

type Command struct {
	Type             CommandType
	RequestID        [16]byte
	LockID           string
	OwnerID          string
	FencingToken     uint64
	CommitTimeMillis uint64
	TTLMillis        uint64
}
