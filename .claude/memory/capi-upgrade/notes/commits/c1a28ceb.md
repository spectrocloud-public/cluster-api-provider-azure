# c1a28ceb — removed the real hard coded path for cert
- **cluster:** cloud-environments (#117 chain) · **plan decision (r2):** NEEDS-DECISION (conflict: azure_secret_cloud.go)
- **tickets:** PEM-7511
- **PR:** #117
- **behavior:** Removes the hardcoded fallback certificate path from `azure_secret_cloud.go` (-16/+4): if no environment folder/cert is configured, CAPZ now behaves exactly like public cloud (no CA cert loaded) instead of probing a developer's test path.
- **pr/ticket context:** implements guyni's review: "I don't think you need a fallback certPath… just like what happens on current public clouds."
- **parity vs v1.26.0:** **FORK-ONLY (behavior folded into S-tip file).** The final file has no path-based cert loading at all (ConfigMap-only), so this is a stepping stone.
- **resolution guidance:** Subsumed by taking S-tip `azure_secret_cloud.go` verbatim. Conflict exists only because the plan replays intermediates over the 2071dd28 merge gap — see cluster summary.
