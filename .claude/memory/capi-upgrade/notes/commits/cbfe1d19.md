# cbfe1d19 — new crds
- **cluster:** webhook-runtime-infra · **plan decision (r2):** REGENERATE
- **tickets:** none
- **PR:** none — direct push (Deepak Sharma, Nov 2022, replayed)
- **behavior:** Hand-adds `ipAllocationMethod: {type: string}` to the generated CRD schemas for AzureCluster and AzureClusterTemplate (frontend-IP blocks). This is the generated SHADOW of the fork's private-cluster API field `IPAllocationMethod` on `LoadBalancerClassSpec` (`api/v1beta1/types_class.go:512` at S, defaulted in `azurecluster_default.go:451`, consumed in `azure/scope/cluster.go`) — the field that lets palette request a Static vs Dynamic front-end IP for private clusters. The Go field belongs to the private-cluster commit cluster (601023f0/3d6a0310/f51b1c5e etc.), not this one.
- **pr/ticket context:** none; behavior inferred from diff + S-tip code.
- **parity vs v1.26.0:** **NOT upstreamed** (`git grep -i ipallocationmethod v1.26.0 -- api/` is empty) — but this commit itself is pure generated artifact. Verdict: **REGEN** — correct as planned.
- **resolution guidance:** Do not cherry-pick. After the private-cluster CODE commits land on `spectro-v1.26.0-master`, `make generate` reproduces these schema lines automatically. If the code cluster's commits are skipped/deferred, this CRD delta must NOT appear either (schema without the field's controller logic is meaningless). T1 no-drift check covers it.
