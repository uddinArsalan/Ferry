# Ferry: repository overview, issue ledger, and completion roadmap

This guide is based on the code currently in this repository. It distinguishes
behaviour that exists today from intended behaviour, and separates confirmed
defects from product decisions that still need to be made. It is a learning and
implementation guide, not a claim that Ferry is ready for production.

## 1. What the repository currently contains

There are two independent Go modules:

| Module | Responsibility | Main entry point |
| --- | --- | --- |
| `ferry/` | Interactive client, local peer identity, group configuration, filesystem watching, and an early libp2p connection layer | [`ferry/cmd/main.go`](ferry/cmd/main.go) |
| `registry/` | gRPC authentication, peer/group/request records, PostgreSQL access, and SQL migrations | [`registry/cmd/main.go`](registry/cmd/main.go) |

The generated gRPC API is imported from the external
`github.com/uddinArsalan/ferry-proto` module; its `.proto` files are not in this
repository. There is no root README, root-level workspace/build/test command,
or CI workflow.

### Intended runtime flow

1. The registry starts, reads environment configuration, connects to PostgreSQL,
   loads TLS credentials, and registers auth, peer, and group gRPC services.
2. Ferry opens a TLS gRPC connection, retrieves or creates a libp2p private key
   in the OS keyring, starts a libp2p host, and starts a periodic group-snapshot
   worker.
3. A user logs in, initializes a peer, creates/joins a group, and chooses a
   local directory.
4. The client stores that directory in `~/.ferry/config/group/`, watches it with
   fsnotify, and periodically refreshes the group/peer snapshot.
5. The current network code tries to open streams to group peers and send a
   JSON event containing an operation and path.

### What works versus what is still scaffolding

- The registry and ferry modules compile; the registry test command reports
  success, but there are no registry tests.
- Registration/login/token code, local YAML configuration, directory watching,
  chunk metadata, and libp2p host creation are present.
- **This is not yet a file synchronization implementation.** A filesystem
  event is sent as a JSON message, but the receiving side only reads arbitrary
  stream bytes and logs them. There is no message framing, file transfer,
  receiver-side apply, startup reconciliation, conflict resolution, or
  end-to-end file-content verification.
- Peer discovery and connection establishment are not reliable in the current
  code. A particularly direct defect in `Node.ConnectToPeers` is that it
  continues when the connection lookup reports no connection, so it skips the
  first dial attempt.

## 2. Current verification results

Commands were run from each module directory:

| Command | Result |
| --- | --- |
| `cd registry && go test ./...` | Passes, but every package reports `[no test files]`. This verifies compilation only. |
| `cd ferry && go test ./tcp -run '^TestNode$' -count=1 -timeout 60s` | Passes. It is a host-initialization smoke test, not a sync test. |
| `cd ferry && go test ./tcp -run '^TestNodeInit$' -count=1 -timeout 8s` | Times out in `TestNodeInit` at `for peerID := range node.GetConnectedPeers()`. The returned channel is never closed, and no peer is required to connect for the test to finish. |
| `cd ferry && go test ./...` | Stalled for more than 180 seconds in the TCP tests; it was interrupted. The isolated test above confirms the hang. |
| `cd ferry && go vet ./...` and `cd registry && go vet ./...` | Pass. |
| `gofmt -l $(rg --files -g '*.go' ferry registry)` | Reports `ferry/watcher/watcher_test.go`, `registry/adapters/token/token.go`, `registry/internals/domain/user.go`, and `registry/internals/db/db.go`. |

Useful current commands:

```sh
cd ferry
go test ./... -run '^TestNode$' -count=1 -timeout 60s
go vet ./...

cd ../registry
go test ./...
go vet ./...
```

Do not treat the filtered ferry command as a full test run: it deliberately
avoids `TestNodeInit`. Fix that test and the underlying lifecycle issue before
relying on `go test ./...`.

## 3. Prioritized issue ledger

### P0 — Must fix before calling this a working sync or exposing the registry

1. **The P2P dialing path skips disconnected peers.** In
   [`ferry/tcp/libp2p.go`](ferry/tcp/libp2p.go), `ConnectToPeers` treats any
   error from `GetPeersConn` as “already exists” and continues. The
   `PeerManager` returns an error precisely when no connection exists. Correct
   the condition, add a focused test, and verify an actual two-host connection.
2. **The transfer path does not transfer or apply files.** `Start` converts
   watcher notifications to an operation/path string; `WriteData` marshals
   JSON; `readData` reads bytes and logs them. TCP stream reads can split or
   combine writes, and there is no receiver-side protocol. Do not describe the
   feature as working sync until a versioned, framed protocol and file apply
   path exist.
3. **Request and membership database operations contain blocking defects.**
   In [`registry/internals/repository/group_repository.go`](registry/internals/repository/group_repository.go):
   - `CreateJoinRequest` looks up the recipient using `recipientPeerID` (the
     local zero-valued database ID) instead of `localRecipientPeerID`.
   - `UpdateJoinRequestStatus` updates `request_status`, while the migration
     defines the column as `status`.
   - Acceptance only changes a request status. It does not create the
     `group_peer_membership` row needed for the accepted peer to appear in
     future snapshots.
   - Several database enum fields are modeled as Go integers and scanned
     directly from PostgreSQL enum text; verify/fix these conversions.
   - `GetPeerSyncSnapshot` scans a JSONB aggregate directly into a Go slice of
     peers; `database/sql` does not automatically unmarshal JSON into that
     struct slice. Decode explicitly or return normalized rows.
   - `PeerGroupMembership.JoinedAt` is a protobuf timestamp, not a
     `time.Time`/scanner type for a SQL timestamp. Use a storage-layer type and
     convert at the gRPC boundary.
4. **Authenticated callers are not consistently authorized.** The unary
   interceptor authenticates most calls, but group and peer handlers/repos
   often do not prove that the caller owns the group, owns the peer, is a
   member, or is allowed to review the request. An authenticated user can
   potentially read or modify records by guessing IDs. Enforce authorization
   in registry services/repositories, not only in the CLI. Review
   [`registry/interceptor/auth_interceptor.go`](registry/interceptor/auth_interceptor.go),
   [`registry/internals/server/group/group_service.go`](registry/internals/server/group/group_service.go),
   and [`registry/internals/server/peer/peer_service.go`](registry/internals/server/peer/peer_service.go).
5. **The request-review CLI omits auth call options.** Calls to
   `GetJoinRequest` in accept/reject flows in
   [`ferry/cli/main.go`](ferry/cli/main.go) do not pass `AuthContext`, unlike
   many neighboring calls. Once the interceptor requires authentication, these
   lookups will fail. Add auth options and test each command end to end.
6. **Accept/reject endpoints need a server-side state machine.** Verify request
   type, requester/recipient, current status, membership, and actor permissions
   in one transaction. Acceptance must atomically record membership and mark
   the request accepted; reject invalid/repeated transitions. Do not trust
   `PeerId`, `Role`, or request ownership supplied by a client.

### P1 — Required for dependable local use

1. **The CLI exit path does not stop the process.** `exit` makes
   `TakeUserParams` return, but `main` then waits unconditionally on
   `<-ctx.Done()`. EOF does the same. Add a cancellation/shutdown path and
   close the host, watcher, gRPC connection, and file logger in a controlled
   order.
2. **The TCP test hangs and leaks resources.** `GetConnectedPeers` returns an
   unbuffered channel that is never closed; its notification callback can also
   block sending to an unread channel. `TestNodeInit` ranges forever. The
   tests do not close their node/logger, and `TestNode` uses the real OS
   keyring. Replace this with bounded assertions, explicit cleanup, and
   generated test keys.
3. **Watcher ownership and event attribution are unsafe.** One fsnotify watcher
   is shared by multiple `WatchFile` calls, each of which starts another
   reader on the same event channel. Events can be consumed by the wrong
   group's goroutine and assigned the wrong group ID. There is no stop/close
   method, event sends can block, and repeated `start`/group acceptance can
   create duplicate watchers. Use one watcher loop with a directory-to-group
   mapping and a context-aware shutdown.
4. **Peer manager state is not consistently synchronized.** Some reads/writes
   (`currPeerId`, `GetPeerDetail`) omit locks, and getters return pointers/maps
   that callers access after releasing the lock. `GetPeersToConnect` holds an
   `RLock` and calls helpers that take another `RLock`; recursive read-locking
   can deadlock when a writer is waiting. Return safe snapshots and avoid
   nested locking; exercise it under `go test -race`.
5. **Peer address/identity publication is brittle.** The client binds libp2p
   to an ephemeral port on `0.0.0.0`, chooses an address by list index, splits
   a multiaddress by `/`, and stores a single address/port in the registry.
   Reachable addresses can change or be NAT/IPv6/relay addresses. Store and
   exchange a validated libp2p multiaddress (or a deliberate set of
   addresses), and decide how peers behind NAT connect. Avoid parsing
   multiaddresses by string indexes. Also handle the `NewConnManager` error
   before replacing `err` with the result of `libp2p.New`.
6. **The file chunker is untested and mishandles read edge cases.** In
   [`ferry/chunker/chunker.go`](ferry/chunker/chunker.go), `ChunkFile` returns
   an error for `io.EOF` rather than handling it as normal termination, and
   `ReadChunk` returns an error for a partial final chunk even when it has
   useful bytes. Validate positive chunk sizes and test empty files, exact
   boundaries, partial final chunks, large files, and hashes.
7. **Configuration errors are silently skipped.** `GetLocalDirs` ignores
   unreadable/invalid YAML files, which can make a group silently disappear.
   Return contextual errors (or report per-file errors), validate group IDs
   before using them in filenames, and use private directory permissions.
8. **Token expiry persistence is not connected to OAuth expiry.**
   [`ferry/keyring/keyring.go`](ferry/keyring/keyring.go) saves expiry values,
   but `GetTokens` does not read them or populate `oauth2.Token.Expiry`.
   `TokenSrc` therefore cannot refresh automatically. Implement expiry-aware
   token retrieval/refresh and test expired, missing, and rotated tokens.
9. **Database resources/startup are not production-safe.** `db.NewDB` opens
   without `PingContext` or explicit pool policy; registry does not close the
   pool or gracefully stop gRPC. The server requires `.env` even if variables
   are already exported, binds to loopback, and uses working-directory-relative
   certificate paths. Make environment loading optional, validate required
   settings explicitly, configure bind/TLS paths, ping on startup, and handle
   shutdown.
10. **Refresh rotation needs race/reuse handling.** The current flow reads a
    refresh token, then creates a replacement and revokes the old one in a
    later transaction. Concurrent refreshes can both pass the initial check.
    Make rotation conditional/atomic and define token-family revocation on
    reuse. Map unknown/expired/revoked tokens to unauthenticated responses,
    not internal errors.
11. **The logger path depends on the process working directory.**
    [`ferry/logger/file_logger/file_logger.go`](ferry/logger/file_logger/file_logger.go)
    uses `../../logs/app.log`; running the binary from the module root writes
    outside the repository. Use a platform-appropriate user data/log directory,
    make the path configurable, and decide rotation/retention. Avoid logging
    credentials, file contents, or sensitive paths.
12. **Local and remote operations can leave partial state.** `createGroup`
    creates the server group/membership before validating/saving the directory.
    Accept flows update request status before local persistence. Define
    transactional server behavior and retryable client behavior; do not report
    success when only part of an operation completed.

### P2 — Product quality, maintainability, and production hardening

- Define explicit policies for bidirectional vs one-way sync, deletion
  propagation, conflict handling, file replacement, symlinks, ignored files,
  permissions, and maximum file sizes. Current events include absolute local
  paths; a remote must receive a path relative to the configured group root.
- Add reconnection/backoff, stream cleanup, bounded retries, context deadlines,
  peer health/last-seen updates, and deduplication. `ConnectToPeers` currently
  has no clear lifecycle for incoming streams; `handleStream` does not register
  the remote stream in `PeerManager`.
- Add a real wire schema with protocol version, message type, group ID,
  relative path, operation ID, chunk/file metadata, and length-delimited
  messages. JSON writes alone are not message framing.
- Add registry request validation and stable gRPC status mapping. Several
  handlers pass raw database errors to gRPC; malformed input, missing records,
  conflicts, and internal errors should be distinct.
- Check `Rows.Err()` after row iteration in repositories. Some methods check it
  inside the loop or omit a final check, missing iteration errors.
- Validate names, email/password policy, address/port ranges, enum values, and
  empty JWT secrets. The password implementation uses Argon2id and constant
  time comparison, but registration currently accepts empty passwords and the
  token constructor does not reject an unset `JWT_SECRET`.
- `utils.FormatAuthResults` includes access and refresh tokens in its output.
  It is currently unused; remove it or make sure it can never be used in
  routine logs/terminal output.
- `GetPeer`/group lookup and sync snapshot queries need ownership scoping, and
  peer `last_seen` is not refreshed. Consider unique owner constraints,
  cascading/retention policy, and indexes after the domain rules are settled.
- The domain types in `registry/internals/domain/` mix persistence fields and
  protobuf types. Keep domain/storage types independent of generated API types;
  convert them in service code.
- The migration `down.sql` files are empty. This makes rollback impossible.
  Add safe reverse migrations, test both directions where feasible, and plan
  data backfills before adding `NOT NULL` columns to populated tables.
- `registry/Makefile` has a `protoc` target for a `proto/` directory that is
  absent here. Either include/version the protocol source or remove/document
  that target. The ferry `build` target writes to `/usr/local/bin` and may
  require elevated permissions; prefer a local output/install target.
- Tracked `.DS_Store` files add machine-specific noise. Remove them from
  version control and ignore them. The root `.gitignore` currently ignores
  certificate private keys; keep private keys and `.env` files out of version
  control. The tracked certificate is public material, but the certificate and
  matching private key must be provisioned reproducibly for local development
  without committing the private key.
- Test helpers currently read the real OS keyring; one logs the token object
  and another logs a refresh token. Never print real credentials in tests or
  CI. Inject a fake keyring/test key instead.
- Small cleanup: run `gofmt` on the four files listed in verification, remove
  or revive the commented-out watcher/TCP tests, and replace stale
  implementation-comment blocks with design documentation once behavior is
  settled. `ferry/logger/logger.go` is currently only a tiny abstraction and
  should either be used consistently or removed.

## 4. Peer IDs and how to test without tying tests to your machine

A libp2p peer ID is derived from its private key. This project stores that key
under one OS keyring service/account (`ferry` / `priv_key`), so one local OS
keyring normally represents one persistent peer identity. The ID is not a
hardware ID, but copying/reusing the same key makes two processes appear as the
same peer. A different machine/keyring produces a different identity. The
registry also stores a network address and port, which are separate from the
identity and can become stale or unreachable.

For repeatable tests, do **not** make unit tests read the developer's keyring:

1. Generate two independent libp2p keys in the test using
   `crypto.GenerateECDSAKeyPair(rand.Reader)`.
2. Start two hosts on loopback with ephemeral ports and explicit test
   addresses; seed isolated peer managers with each other's IDs/addresses.
3. Exercise dial, stream setup, framing, and transfer without the registry.
4. Use a temporary directory per peer and assert the resulting bytes and
   hashes, not just that a socket connected.
5. Add a separate registry integration suite using a disposable PostgreSQL
   database and migrations.
6. Add a CLI/system test only after injectable identity/configuration and a
   local registry fixture exist. For a manual two-process test before that,
   use separate OS user/keyring profiles or test keys; do not overwrite the
   normal keyring entry.

This removes the “peer ID belongs to this machine” obstacle from tests while
still testing the real identity model. For real cross-machine runs, test
reachable advertised addresses, firewall/NAT constraints, reconnect after a
peer's address changes, and use TLS/identity verification for the registry.

## 5. Recommended build order and learning roadmap

Work in small slices: read the involved code, write a failing test, implement
one behavior, run focused tests, then run the wider suite. Keep a short
decision record in pull requests (decision, alternatives, trade-off) so the
project remains yours to understand.

### Phase 0 — Define the product contract

Before designing conflict or deletion logic, decide:

- Is sync bidirectional or a one-way publish/subscribe?
- Are group directories equal replicas, or does one peer own canonical state?
- How are concurrent edits and deletions resolved?
- Which file types/metadata/symlinks/hidden files are in scope?
- Is the topology direct peer-to-peer only, or are relay servers required?
- Who may invite, accept, remove peers, and change group roles?

Write acceptance scenarios for those choices. Avoid implementing speculative
conflict behavior before the product semantics are clear.

### Phase 1 — Make the existing foundations deterministic

- Fix the hanging test, add cleanup, and replace keyring-dependent tests with
  injected test keys.
- Fix the direct connection-condition, SQL placeholder/column errors, SQL row
  scanning, and membership-on-accept flow.
- Add request validation, transaction boundaries, authentication/authorization
  checks, and stable gRPC errors.
- Make startup/configuration paths explicit; ensure `exit`, signal, and EOF all
  perform orderly shutdown.
- Add registry repository/service tests against PostgreSQL and make
  `go test ./...` complete reliably.

**Exit criteria:** repeatable tests, no database access bypass, create/invite/
join/accept flows produce durable correct memberships, and shutdown leaves no
blocked goroutines.

### Phase 2 — Stabilize identity, address discovery, and peer lifecycle

- Inject key storage and identity providers; keep production OS keyring as one
  implementation.
- Store complete, validated libp2p addresses and define advertised-address/NAT
  behavior.
- Establish and remove streams symmetrically; register incoming streams;
  deduplicate peers across groups; add retry/backoff and health state.
- Test two local hosts with separate identities, then two real devices.

**Exit criteria:** two independent peers reconnect after restart and communicate
using their own keys; a stale/unreachable peer does not block the rest.

### Phase 3 — Implement a minimal, correct sync protocol

- Start with a length-delimited, versioned message codec and round-trip tests.
- Send a group-relative path and typed create/update/delete metadata, never a
  sender's absolute path.
- Implement manifest/chunk transfer with hashes, bounded chunk sizes, safe
  temporary files, atomic rename, and receiver-side path containment checks.
- Add startup scans so missed events while offline are repaired.
- Test empty/small/large/binary files, final partial chunks, rename/delete,
  nested directories, disconnect mid-transfer, duplicate messages, and
  corruption.

**Exit criteria:** a peer can create/edit/delete a file and a second peer
reproduces the intended state with byte-for-byte verification and no writes
outside its configured root.

### Phase 4 — Consistency and recovery

- Implement the chosen conflict policy with operation IDs/version metadata and
  durable progress.
- Handle watcher event coalescing, atomic-save rename patterns, reconnect,
  partial transfer resume, disk full, permission errors, and process crash.
- Define ignore rules and symlink/permission behavior.

**Exit criteria:** documented behavior for simultaneous edits and deletes;
restarting either peer converges without data loss or duplicate loops.

### Phase 5 — Production operations and polish

- Add structured/redacted logs, metrics, bounded log retention, configurable
  endpoints, TLS/certificate lifecycle, and a deployment story.
- Add root documentation, a root `go.work` or explicit multi-module commands,
  CI (format, vet, unit tests, DB integration tests, race tests), and migration
  checks.
- Package binaries with unprivileged install paths; document backup, key
  recovery, peer removal, and secret rotation.
- Run a soak test with multiple peers and large/changing directories.

**Exit criteria:** a new developer can configure, run, test, and troubleshoot
the system from repository docs without relying on machine-specific files.

## 6. Suggested test matrix

| Layer | What to test | Current gap |
| --- | --- | --- |
| Unit | Chunk boundaries/hash, config validation, path safety, event-to-message conversion, auth token expiry, request state transitions | Chunker/config/service tests are absent |
| Concurrency | Peer manager map access, watcher shutdown/duplicate start, event backpressure | No race tests; unsafe shared watcher/state |
| Network | Two generated identities, dialing in both directions, framed multi-message streams, disconnect/reconnect | Current tests only initialize a host; the connection path is broken |
| Registry integration | Migrations, repository scans, enum/JSONB mapping, membership and request transactions, authorization | No registry tests; use disposable PostgreSQL |
| End-to-end | Login/init, create group, invite/join, accept, create/edit/delete files, restart and converge | Not currently possible: transfer/apply protocol is missing |
| Security/reliability | Unauthorized IDs, invalid/expired/reused tokens, malicious paths, corrupt/oversized frames, disk errors | No security or failure-injection suite |

Recommended commands once tests are deterministic:

```sh
cd ferry && go test ./... && go test -race ./...
cd ../registry && go test ./... && go test -race ./...
```

The race suite can be added after basic unit tests pass; it is especially
valuable for the peer manager and watcher. Run database integration tests with
a disposable database, never a personal or production database.

## 7. A practical first-week sequence

1. Read this guide alongside `ferry/tcp/libp2p.go`,
   `ferry/peers/peer.go`, and `registry/internals/repository/group_repository.go`.
   Draw the current peer/group/request data flow yourself.
2. Make `TestNodeInit` bounded and close resources; add a two-host test with
   independent generated keys.
3. Fix and test the connection lookup condition; prove one stream can be
   opened, then prove two framed messages arrive separately.
4. Add registry integration tests for join-request creation, status changes,
   and membership snapshots; fix the SQL/type issues those tests expose.
5. Add authorization tests before implementing file writes. A peer sync feature
   must not turn a guessed group/peer/request ID into access.
6. Only then choose and implement the first end-to-end file operation (a small
   create/update transfer with hash verification is a good vertical slice).

That sequence builds understanding in Go testing/concurrency, database
transactions, gRPC authorization, libp2p addressing, and filesystem semantics
while steadily producing working behavior.

## 8. File-by-file reading map

This map points out the main concern in each source area, including files
whose issue is primarily missing behavior rather than an incorrect line.

### Ferry client

| File(s) | Review note |
| --- | --- |
| [`ferry/cmd/main.go`](ferry/cmd/main.go) | Hardcodes `127.0.0.1:5051`; starts background services before the CLI; always waits for signal cancellation after the CLI returns; does not close the gRPC connection or watcher. |
| [`ferry/client/client.go`](ferry/client/client.go) | TLS certificate path is relative to the current directory and `GRPCClient` exposes no `Close` method. Keep the TLS server name/certificate setup configurable and test it from a different working directory. |
| [`ferry/cli/main.go`](ferry/cli/main.go) | Large CLI/service surface in one file; some command callbacks intentionally log and return nil, some `GetJoinRequest` calls omit auth, group/request workflows can partially commit, and `start` can add duplicate watchers. `strings.Fields` also means command arguments do not support shell-style quoting. |
| [`ferry/cli/auth_tokens.go`](ferry/cli/auth_tokens.go) | Token source only reads the keyring; it does not refresh on expiry. |
| [`ferry/keyring/keyring.go`](ferry/keyring/keyring.go) | Expiries are written but not loaded; multi-key writes can fail part-way and leave mixed token state. |
| [`ferry/utils/utils.go`](ferry/utils/utils.go) | `GetPrivKey` treats every keyring lookup error as “no key” before attempting generation; preserve/report backend errors rather than masking them. `FormatAuthResults` prints secrets. Peer ID generation itself is a good stable-key model. |
| [`ferry/worker/map_sync/main.go`](ferry/worker/map_sync/main.go) | Snapshot refresh replaces peer-group and group maps in separate steps; callers can observe a partially refreshed view. Error messages discard useful underlying causes, and `GetPeerId` is called as a side effect of refresh. |
| [`ferry/config/config.go`](ferry/config/config.go) | YAML read/parse errors are ignored; group ID is used as a filename component; config directory permissions are broader than needed. |
| [`ferry/watcher/watcher.go`](ferry/watcher/watcher.go) | Shared fsnotify event reader per `WatchFile`, no cancellation/close, blocking event send, no event coalescing or stable mapping of watched root to group. |
| [`ferry/chunker/chunker.go`](ferry/chunker/chunker.go) | EOF/partial-chunk handling and chunk-size validation need fixes; no tests and no current transfer caller. |
| [`ferry/types/event.go`](ferry/types/event.go), [`ferry/types/chunk.go`](ferry/types/chunk.go), [`ferry/types/peer.go`](ferry/types/peer.go) | These are currently simple in-memory structs. They do not define a durable/versioned wire protocol, operation IDs, relative paths, or conflict/version metadata. |
| [`ferry/tcp/libp2p.go`](ferry/tcp/libp2p.go) | Active but incomplete networking path: dial-condition bug, arbitrary address selection, no framing/apply path, unregistered incoming streams, background context for dialing, and lifecycle/close gaps. `GetAddr` also assumes at least one address. |
| [`ferry/tcp/tcp.go`](ferry/tcp/tcp.go) | Separate legacy TCP server is not wired into the main runtime. Its accept/read loops can spin forever on errors, its start block has no release path, and it does not establish the libp2p protocol. Remove it or clearly isolate/test its purpose. |
| [`ferry/logger/file_logger/file_logger.go`](ferry/logger/file_logger/file_logger.go), [`ferry/logger/ferry_logger/ferry_logger.go`](ferry/logger/ferry_logger/ferry_logger.go), [`ferry/logger/logger.go`](ferry/logger/logger.go) | File path is CWD-relative; there is no rotation/redaction policy; the `Logger` interface is not used consistently. |

### Ferry tests and build files

| File(s) | Review note |
| --- | --- |
| [`ferry/tcp/tcp_test.go`](ferry/tcp/tcp_test.go) | `TestNodeInit` ranges on a never-closed channel; tests use real keyring state or leave hosts/loggers open. Replace with deterministic generated identities, deadlines, and cleanup. |
| [`ferry/keyring/keyring_test.go`](ferry/keyring/keyring_test.go), [`ferry/tests/token_test.go`](ferry/tests/token_test.go) | Environment/keyring integration tests are not isolated; logs can contain credentials. Separate fake-backed unit tests from opt-in local integration tests and never log tokens. |
| [`ferry/watcher/watcher_test.go`](ferry/watcher/watcher_test.go) | Entire test is commented out, so watcher behavior is untested; it is also one of the files reported by `gofmt -l`. |
| [`ferry/logger/file_logger/file_logger_test.go`](ferry/logger/file_logger/file_logger_test.go) | Checks only that a log line can be written; it uses the real filesystem and does not verify rotation/path/close behavior. |
| [`ferry/Makefile`](ferry/Makefile) | `build` targets `/usr/local/bin`; local build is more useful, but should not be the only documented build path. |
| [`ferry/go.mod`](ferry/go.mod) | Module compiles with the current toolchain; keep dependency updates reproducible and add CI rather than relying on local Go installation state. |

### Registry

| File(s) | Review note |
| --- | --- |
| [`registry/cmd/main.go`](registry/cmd/main.go) | Requires `.env` even when env vars are exported; binds loopback; certificate paths depend on CWD; no graceful gRPC/DB shutdown. |
| [`registry/internals/db/db.go`](registry/internals/db/db.go) | `sql.Open` is lazy; no ping, pool policy, or close lifecycle. Also reported by `gofmt -l`. |
| [`registry/interceptor/auth_interceptor.go`](registry/interceptor/auth_interceptor.go) | Auth-service exemption is based on substring matching of method names; verify actual generated full-method paths and use an explicit allowlist. Authentication alone does not authorize access to a peer/group/request. |
| [`registry/adapters/password/password.go`](registry/adapters/password/password.go) | Argon2id + constant-time comparison are good foundations; validate empty password and bound parsed Argon parameters before expensive hashing. |
| [`registry/adapters/token/token.go`](registry/adapters/token/token.go) | Reject an empty signing secret at startup; maintain fixed algorithm validation and add tests for expiry, invalid signature, and refresh-token hashing. Also reported by `gofmt -l`. |
| [`registry/internals/domain/user.go`](registry/internals/domain/user.go) | Formatting only is reported by `gofmt`; broader domain/storage concerns are described above. |
| [`registry/internals/domain/peer.go`](registry/internals/domain/peer.go), [`registry/internals/domain/join.go`](registry/internals/domain/join.go) | Integer enums need explicit DB/protobuf conversions and unknown-value handling; do not rely on incidental numeric equality across API and domain enums. |
| [`registry/internals/domain/group.go`](registry/internals/domain/group.go), [`registry/internals/domain/refresh_token.go`](registry/internals/domain/refresh_token.go) | Keep these persistence/domain types independent of protobuf types and consistently named; the refresh-token lifecycle needs atomic rotation semantics. |
| [`registry/internals/repository/auth_repository.go`](registry/internals/repository/auth_repository.go) | Refresh-token transaction needs conditional old-token revocation and concurrency/reuse handling; add affected-row checks and repository tests. |
| [`registry/internals/repository/peer_repository.go`](registry/internals/repository/peer_repository.go) | Peer lookup is not scoped to the authenticated user; `Rows.Err` should be checked after iteration; last-seen is not maintained. |
| [`registry/internals/repository/group_repository.go`](registry/internals/repository/group_repository.go) | Contains the request lookup/status defects, enum/timestamp/JSONB scan mismatches, missing membership on acceptance, and incomplete row-iteration error checks described above. |
| [`registry/internals/server/auth/auth_service.go`](registry/internals/server/auth/auth_service.go) | Registration validates name/email but not password; login leaks account existence through different statuses; missing refresh rows become internal errors; add status, policy, and token-rotation tests. |
| [`registry/internals/server/group/group_service.go`](registry/internals/server/group/group_service.go) | Many methods do not check caller ownership/role and pass repository errors directly to gRPC; enforce authorization and map errors here. |
| [`registry/internals/server/peer/peer_service.go`](registry/internals/server/peer/peer_service.go) | Peer lookup is not owner-scoped; validate registration input and map uniqueness/validation errors to stable statuses. |
| [`registry/go.mod`](registry/go.mod) | Separate module; no repository-level workspace/CI currently ensures both modules are built and tested together. |
| [`registry/Makefile`](registry/Makefile) | Migration commands rely on `DB_URL`; the `protoc` target references a missing local `proto/` directory. Add safe environment checks and document the intended protocol-source location. |
| [`registry/migrations/`](registry/migrations/) | All current down migrations are empty. Review schema evolution/backfills, enum/index constraints, group-owner invariants, and reversibility before production data is introduced. |

At repository root, `.DS_Store` files are tracked, there is no user-facing
README, and there is no CI configuration. Keep `.env`, private keys, and real
keyring credentials untracked; document safe generation/configuration rather
than committing secrets.
