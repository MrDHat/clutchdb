# clutchdb

clutchdb is a small, opinionated database for distributed locking. It favors determinism and explicitness over convenience or hidden behavior.

## Why build this?

Because its fun! And I thought it'll be a good exercise to build a very small but _potentially_ useful database.

## Design Philosophy

#### Correctness is the Product

This database exists to answer one question:

> "Is it safe to enter the critical section?"

Performance, ergonomics, and convenience only matter after correctness. If the system is fast but allows two writers to proceed concurrently, it has failed completely.

#### The Server Owns Truth, the Client Owns Control

The server:

- Accepts commands
- Applies deterministic rules
- Returns facts

The client:

- Decides when to retry
- Decides how to react to failures

There are no hidden retries, no implicit sessions, and no magic recovery.

#### Explicit Is Better Than Helpful

- Nothing happens implicitly.
- Ownership is explicit
- TTL is explicit
- Idempotency is explicit
- Failures are explicit

If the system did something, it was because a command was sent and processed. This makes failures understandable and debuggable.

#### Commands, Not Queries

The protocol is command-based. Clients do not ask:

> "Who owns this lock?"

They say:

> "Acquire this lock under these conditions."

This avoids race conditions and keeps the state machine small and deterministic.

#### Time Is Treated as Data, Not a Side Effect

Time is:

- Measured explicitly
- Advanced by the leader
- Recorded in state

The database does not “wake up” to expire locks. Locks expire as a result of commands being processed. This keeps behavior predictable and replayable.

## Wire Protocol

The data is transmitted in binary format over TCP between clients and servers. Each message begins with a length field (u32) followed by the command and its associated fields. The protocol is deterministic and fixed-size per command type, ensuring consistent interpretation across all nodes.

### Request format

```
| u32 length | // total bytes after this field
| u8 cmd | // 1 = ACQUIRE, 2 = RENEW, 3 = RELEASE
| u128 request_id |
| u128 lock_id |
| u128 owner_id |
| u64 ttl_ms |
```

**ACQUIRE / RENEW Request (57 bytes total)**

| Field     | Size     | Description                       |
| --------- | -------- | --------------------------------- |
| Length    | 4 bytes  | u32: total bytes after this field |
| Cmd       | 1 byte   | ACQUIRE=1, RENEW=2                |
| RequestID | 16 bytes | Unique request identifier         |
| LockID    | 16 bytes | Lock identifier                   |
| OwnerID   | 16 bytes | Client/owner identifier           |
| TTLMS     | 8 bytes  | Time-to-live in milliseconds      |

---

**RELEASE Request (49 bytes total)**

| Field     | Size     | Description                       |
| --------- | -------- | --------------------------------- |
| Length    | 4 bytes  | u32: total bytes after this field |
| Cmd       | 1 byte   | RELEASE=3                         |
| RequestID | 16 bytes | Unique request identifier         |
| LockID    | 16 bytes | Lock identifier                   |
| OwnerID   | 16 bytes | Client/owner identifier           |

---

### Response format

```
| u8 status |
| u64 fencing_token |
| u64 expires_at |
```

**Response Status Codes**
| Status Code | Meaning |
| ----------- | ---------------------------------- |
| `0` | Success |
| `1` | Lock already held (ACQUIRE failed) |
| `2` | Lock not held (for RENEW) |
| `3` | Invalid request / malformed |
| `4` | Not leader / redirect to leader |
| `5` | Lock expired (for RENEW/RELEASE) |
| `6+` | Reserved for future errors |

## Development Setup

### Git Hooks

This project includes Git hooks to ensure code quality. To set them up:

```bash
./hooks/install-hooks.sh
```

This installs both pre-commit and pre-push hooks:

- **Pre-commit**: Runs linting checks (go vet + code formatting)
- **Pre-push**: Runs all tests before allowing pushes

If any checks fail, the commit/push will be aborted.