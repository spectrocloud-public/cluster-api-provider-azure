# 2c9ba2eb — Resolved NIC deletion issue
- **cluster:** networking · **plan decision (r2):** SKIP (empty on cumulative tree — duplicate)
- **tickets:** none (no ticket/PR context)
- **PR:** none — direct push
- **behavior:** EMPTY commit in the linearized fork history (0 files). Original content (twin 6a0f49ec3d, Dec 2022) patched old async.go `DeleteResource`: when a non-delete-type long-running-operation lookup errored, only bail out if the pending future was actually a DeleteFuture — so a stale non-delete future couldn't block NIC deletion.
- **pr/ticket context:** none.
- **parity vs v1.26.0:** SUPERSEDED architecturally. The go-autorest futures machinery this patched no longer exists: since the v1.18 base (and in v1.26.0), async.go uses track2 resume-token pollers and `GetLongRunningOperationState(name, service, futureType)` is KEYED BY FUTURE TYPE, so a pending PUT future can no longer shadow a DELETE — the failure mode is gone by design.
- **resolution guidance:** SKIP safe: empty diff, and the behavior (deletes not blocked by unrelated pending operations) is guaranteed by the target's per-future-type state model. No regression-watch entry needed beyond generic delete-path CBT coverage.
