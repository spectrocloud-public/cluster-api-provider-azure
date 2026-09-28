# Cluster: webhook-runtime-infra

Webhook/controller pod separation, TLS hardening (PEM-2613), gallery & AMCP webhook validation relaxations, Spectro manifests/CICD/build plumbing, controller-runtime bump. 19 commits.

## Behavior contract (what must survive on spectro-v1.26.0-master)

1. **Split-pod runtime (main.go)** — one image, two modes: `--webhook-port` defaults to **0**; `0` → `registerControllers` only, `>0` → `registerWebhooks` only. Palette deploys a controller Deployment (no flag) and a webhook Deployment (`--webhook-port=9443`) behind `capz-webhook-service` (capi-webhook-system, targetPort `webhook-server`). If the default reverts to upstream's 9443, palette controller pods register NO reconcilers → total reconcile outage.
2. **TLS hardening (PEM-2613 / CoreSec)** — webhook server pins MinVersion=MaxVersion=TLS1.2 and exactly 4 ECDHE-GCM cipher suites via `GetTLSOptionOverrideFuncs`/`GetDefaultTLSCipherSuits` wired into `webhook.Options.TLSOpts`. Baked-in defaults, not flags.
3. **Flag compatibility** — `--metrics-bind-addr` must exist (fork maps it onto metricsOptions.BindAddress); palette deployment args pass it and would crash the manager on unknown flag. Do NOT register fork TLS flags (duplicate of CAPI `flags.AddManagerOptions` → pflag panic; f9cfc76b lesson).
4. **Admission relaxations for palette AKS lifecycles** — (a) dual-gallery images accepted: Image with BOTH `sharedGallery`+`computeGallery` set is valid, ComputeGallery preferred (no node repave during palette upgrade); (b) AMCP `virtualNetwork.resourceGroup` immutability tolerates empty→set (v1.3.2-era clusters must be upgradable).
5. **Spectro manifest generation** — `spectro/{base,global}` kustomizations + `config/default/*` patches + `spectro/run.sh` are generator INPUT (must-carry code); `spectro/generated/core-{base,global}.yaml` and CRD bases are OUTPUT (regen via make generate + vendorcrd-sync; verify with `vendorcrd-sync verify --provider azure`, per Spec Patch F).
6. **Build/release chain** — FIPS-capable palette Dockerfile (hardened builder, go-build-fips/static, assert+govulncheck), Makefile spectro vars, `spectro-release.yaml` (= Patch-H image stage), `backport.yaml` automation.

## Net effect of chains

- **main.go (248e1eb1 → 5e819cea → ad0f208f → f9cfc76b):** the four commits are ONE unit; individual replay is why 5e819cea/f9cfc76b conflict. NET delta vs v1.26.0's reworked main.go (verified via `git diff v1.26.0 0164d80e -- main.go`, fork-attributable parts only): webhook-port default 0 + mode-switch if/else; `TLSOptions` type + tlsOptions var; `GetTLSOptionOverrideFuncs` + `GetDefaultTLSCipherSuits` + `TLSOpts: tlsOptionOverrides` in webhook.NewServer; discard upstream's flag-derived tlsOptions (`_, metricsOptions, err := flags.GetManagerOptions(...)`); `--metrics-bind-addr` flag + `metricsOptions.BindAddress = metricsAddr`; two `setupLog.V(0).Info("register…")` markers; cliflag import. ad0f208f's go.mod/go.sum hunks are replay junk — drop. Recommended execute strategy: hand the resolver this net-delta spec against v1.26.0 rather than 4 sequential cherry-picks.
- **Gallery (6d749add → 45d9bb84):** net = 45d9bb84's else-if (ComputeGallery wins; both-set allowed). 6d749add is an intermediate comment-out.
- **BET remnants (93449d5c, a822bc84, b671d486, 37d4d8ec):** replay flattening left these empty or 0-byte; every original behavior is either upstreamed (b671d486 nil-safe DNSPrefix), abandoned (93449d5c last-system-node-pool removal — S enforces same check as upstream), or rewritten away (a822bc84 agentpools). Net contribution: zero.
- **Manifests (42eb65da → cbfe1d19 → 604a2909):** net = S-tip spectro/ inputs + patch-based CRD conversion injection; hand-edits to generated files (42eb65da conversion stanzas, cbfe1d19 ipAllocationMethod, 604a2909 config/webhook/manifests.yaml) are all superseded by regeneration — EXCEPT the AMCP/AMMP webhook removal intent, which currently exists ONLY as a generated-file edit (see risk below).
- **Build (1a8918cf → 364fe27b; faa067a7; fc3e22b4):** net = S-tip Dockerfile/Makefile/workflows. Take files at S-tip state, bump `TAG` v1.18.0→v1.26.0 and builder Go per Patch-G.

## Parity verdict (cluster)

**FORK-ONLY must-carry core** (split-pod wiring, TLS hardening, metrics-bind-addr, gallery + VNET-RG relaxations, spectro inputs, build chain) + **SUPERSEDED tail** (94233eeb controller-runtime bump — v1.26.0 has cr v0.23.3, go 1.25.0, internal/webhooks CustomValidator; all BET remnants) + **REGEN** (all generated artifacts). Nothing in this cluster is UPSTREAMED-as-is except b671d486's intent.

## ⚠ HIGH-risk findings / recommended plan changes

1. **93449d5c PICK is wrong → SKIP.** The commit is a 0-byte file creation (`exp/api/v1beta1/azuremanagedmachinepool_webhook.go`); picking it breaks the build (`expected 'package', found EOF`) unless aa405349 (NEEDS-DECISION) later deletes it. Zero behavioral content at S.
2. **e3c6f868 / 45d9bb84 (and 6d749add) "clean PICK" is a dead-path trap.** They edit `api/v1beta1/azuremanagedcontrolplane_webhook.go` / `api/v1beta1/azureimage_validation.go`, which don't exist at v1.26.0 — live logic moved to `internal/webhooks/azuremanagedcontrolplane_validation.go::validateAzureManagedControlPlaneVirtualNetworkUpdate` and `internal/webhooks/azureimage_validation.go::validateSingleDetailsOnly`, both still STRICT at v1.26.0. A clean apply = dead code + silent loss of both relaxations (repave-class + upgrade-blocker regressions). Flip to NEEDS-DECISION/PICK-RESOLVE with explicit re-target; **add all three to Regression Watch** (currently absent from it).
3. **94233eeb NEEDS-DECISION → SKIP** (shrinks human shortlist by 8 conflict files). Evidence: v1.26.0 go.mod `controller-runtime v0.23.3`/`go 1.25.0`; the 6 conflicted webhook files don't exist at v1.26.0; internal/webhooks already CustomValidator/CustomDefaulter.
4. **604a2909 webhook-disable drift needs a human call.** `config/webhook/manifests.yaml` removes AMCP/AMMP default+validation webhooks, but S-tip `spectro/generated/core-global.yaml` still SHIPS them — inputs and outputs disagree today. On v1.26.0, `make generate` will restore the manifests.yaml entries (markers live in internal/webhooks). If the disable is wanted, implement as a spectro/global kustomize delete-patch (generator input); if not, drop the hand-edit. Don't let the T1 no-drift gate be "fixed" by re-hand-editing generated files.
5. **Generic replay hazard:** several "F4 conflicts" in this cluster (ad0f208f go.mod/go.sum, 5e819cea/f9cfc76b main.go) are artifacts of replaying a flattened history, not real semantic conflicts — resolve at the unit level (net delta), not per-commit.

## Recommended F4_HOTSPOTS paths

- `main.go` (split-pod + TLS + flags — the one true F4 here)
- `internal/webhooks/azureimage_validation.go` (re-target of 45d9bb84)
- `internal/webhooks/azuremanagedcontrolplane_validation.go` (re-target of e3c6f868)
- `config/webhook/manifests.yaml` + `spectro/global/kustomization.yaml` (webhook-disable drift)
- `Dockerfile`, `Makefile` (fork build chain vs upstream churn)
- `spectro/**`, `config/default/**` (generator inputs — Patch-F boundary)

## Ticket set
PEM-2613 (TLS ciphers, CoreSec — carried), PCP-5293 (controller-runtime bump — superseded), PCP-6027/6734/6787/6959 (go-version/CVE bumps in adjacent M* commits — superseded by target+Patch-G), BET-3719/BET-3963 (historical, abandoned/rewritten), PCP-6412 & PCP-1467/1567/1595/1596 (AKS field work — other clusters, shares the AMCP webhook surface).
