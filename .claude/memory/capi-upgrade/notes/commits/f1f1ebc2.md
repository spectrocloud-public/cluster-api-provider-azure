# f1f1ebc2 — adding a lock when set up env for multiple controllers
- **cluster:** cloud-environments (#124) · **plan decision (r2):** NEEDS-DECISION (conflict: azure_secret_cloud.go)
- **tickets:** PEN-8516 (branch: PEN-8516-Fix-Azure-Secret-Map-Contention-error; not in JIRA batch context)
- **PR:** #124 "adding a lock when set up env for multiple controllers" — "fixed the thread contention when multiple controllers trying to call the setEnvironment function in Azure Secret Cloud"
- **behavior:** Adds `azureEnvironmentMutex sync.Mutex` and wraps `azure.SetEnvironment(env.Name, env)` in `processAzureEnvironmentJSON` with Lock/Unlock. Fixes a real crash/race: `autorest/azure.SetEnvironment` writes an unsynchronized package map; multiple reconcilers initializing scopes concurrently corrupted it (map contention error).
- **pr/ticket context:** PR body only; the failure was observed in production-like runs.
- **parity vs v1.26.0:** **FORK-ONLY must-carry.** Applies to fork-only code (the function it guards doesn't exist upstream).
- **resolution guidance:** Trivially preserved by taking S-tip `azure_secret_cloud.go` (the mutex + lock are lines ~44-45 and ~137-140 of the final file). Conflict is purely the replay-over-merge-gap artifact. Do NOT drop: without it the ConfigMap init path is racy under multi-controller startup.
