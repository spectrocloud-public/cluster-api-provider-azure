# 1b4fdec4 — Global vars for CAPZ
- **cluster:** cloud-environments (#117 chain) · **plan decision (r2):** NEEDS-DECISION (conflicts: defaults.go, identity.go, internal/api/v1beta1/azurecluster_default.go, util/remote/*)
- **tickets:** PEM-7511
- **PR:** #117
- **behavior:** Centralizes the custom-CA machinery into `azure_secret_cloud.go` package globals (+189: `globalCertPool/globalTransport/globalHTTPClient` + RWMutex + `InitializeGlobalTransport` + `Configure{AzureClient,ARMClient,AzIdentity,Rest}Options` helpers), and slims `defaults.go`/`identity.go`/`util/remote` to consume them instead of building per-call transports. main.go gains an `InitializeGlobalTransport()` startup call (later reshaped).
- **pr/ticket context:** responds to review that transport/cert handling was scattered; "global one" per author's main.go comment.
- **parity vs v1.26.0:** **FORK-ONLY must-carry** — the global-transport architecture is the surviving design at S (final `azure_secret_cloud.go` still built on these globals + `azurepkg.GlobalHTTPClient` bridge).
- **resolution guidance:** Land as part of the #117 net diff. The final shape matters, not this step: at S, initialization happens via `InitializeAzureConfigForCluster` (ConfigMap), and `azure/defaults.go` holds the `AzSecretCertPool/AzSecretCertData/GlobalHTTPClient` bridge vars. util/remote hunks are net-zero — drop.
