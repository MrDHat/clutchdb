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
