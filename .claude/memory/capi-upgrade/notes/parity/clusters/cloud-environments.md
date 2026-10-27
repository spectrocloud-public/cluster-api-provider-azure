# Cluster: cloud-environments — gov cloud + AzureSecret (air-gapped) custom Azure environments

## Ticket set
- **PCP-1159** — gov-cloud (AzureUSGovernmentCloud) support for AKS via `AzureManagedControlPlaneSpec.AzureEnvironment` (upstream issue kubernetes-sigs#3489).
- **PEM-7501** — Azure Secret Cloud in CAPZ: JSON env model → ConfigMap → mounted → controller loads env.
- **PEM-7511** — AzureSecret cloud support in palette (PR #117 umbrella).
- **PEM-5484** — env checking + ussec→eastus2 location mapping (PRs #118/#119/#121/#122).
- **PEN-8516** — SetEnvironment map-contention fix (PR #124).

## Behavior contract (what must exist on spectro-v1.26.0-master)
The fork lets CAPZ operate against a **custom/air-gapped Azure cloud ("AzureUSSecretCloud")** whose endpoints and CA are not known to the Azure SDK, while leaving public/gov/china cloud behavior bit-identical to upstream:
1. **Env loading** (`azure/scope/azure_secret_cloud.go`, fork-only file): `InitializeAzureConfigForCluster` reads ConfigMaps `azure-capz-env-config` (`azure-capz-env.json`) and `azure-capz-cert-config` (`azure-ca.crt`) from the cluster namespace at `NewClusterScope` time (non-fatal if absent); JSON → `autorest.EnvironmentFromFile` → `azure.SetEnvironment` under `azureEnvironmentMutex` (PEN-8516); cert → global TLS12 transport/`http.Client` (+ system pool + custom CA), exposed via `Configure{AzureClient,ARMClient,AzIdentity,Rest}Options` and bridged to `azure` pkg globals (`AzSecretCertPool/AzSecretCertData/GlobalHTTPClient`).
2. **SDKv2 cloud config** (`azure/defaults.go`): `AzSecretCloudName="AzureUSSecretCloud"`; `ARMClientOptions` default branch resolves unknown cloud names via `getCloudConfigurationFromEnvironment` (dynamic env → `cloud.Configuration` incl. Graph/Gallery/Storage services) + injects GlobalHTTPClient transport.
3. **Credentials** (`azure/scope/identity.go`): per-credential transport injection (`ConfigureAzIdentityOptions`), `DisableInstanceDiscovery=true` (SP + cert), explicit `cloud.Configuration` (AAD authority + RM audience/endpoint) on cert & MSI paths. (v1.26.0 already has the client-secret Cloud config — PARTIAL overlap; keep upstream's, add fork extras.)
4. **Kubeconfig CA injection** (both `controllers/azuremanagedcontrolplane_reconciler.go` +102 and `controllers/azureasomanagedcontrolplane_controller.go` +81): append `AzSecretCertData` to `CertificateAuthorityData` of generated kubeconfigs.
5. **PrivateDNS dup-link prevention** (`azure/services/privatedns/`): `ListAllZonesByName` + `ListByZone`; skip vnet-link creation if the VNet is already linked to a same-named zone in ANY resource group.
6. **cluster.go env-split dispatch** (PEM-5484 #121): all LB/publicIP/APIServerHost/SetDNSName entry points dispatch `isAzureSecretCloudEnvironment()` → nil-safe `…ForAzureSecret` variants vs `…ForStandard` (upstream-copy) bodies; nil guards in `APIServerPublicIP()`/`APIServerPrivateIP()`.
7. **Location mapping** (#118/#122): `util/azure.NormalizeAzureRegion` (`ussec→eastus2`) applied in cluster/machinepool/managedcontrolplane `Location()` ONLY when cloud==AzureUSSecretCloud AND RM endpoint contains `.scombine.scloud` (Sequoia emulator detection — reviewer-mandated guard so real AzSecret is never remapped).
8. **main.go**: `AZURE_ENVIRONMENT==AzureUSSecretCloud` → restConfig Timeout=30s/QPS=20/Burst=30.
9. Misc: `azure/const.go` `DisablePrivateDNSAnnotation` (`capz.io/disable-private-dns`) honored in `PrivateDNSSpec()` (bypass); managedclusters.go leveled kubeconfig debug logging.

## Net effect of the cascades (revert pairs / evil merge)
- **#117 chain (15 commits)** contains: revert pair A 7a89534e↔7f0e76c4 (managedcontrolplane.go ±465, net zero), cleanup arc a0e29487+b7f8d03b (machinepoolmachine.go, util/remote, loadbalancers.go — ALL net zero vs base), revert pair B cfabdbf5↔50c8656c (LB spec_test net zero; managedclusters spec_test +29 residue), and the **conflict-resolution merge 2071dd28** ("Resolve merge conflicts in azure_secret_cloud.go", merges post-#113 master into the branch — NOT replayable as a non-merge cherry-pick; this is why every later azure_secret_cloud.go commit conflicts in rehearsal). NET #117 = `git diff 7af8521e57^1 7af8521e57` — 14 files, +796/-75, exactly items 1-5,8,9 above in pre-ConfigMap→ConfigMap final form.
- **Location-mapping cascade**: e8b25112 (unguarded) → 32e8a1f1 (guard) → e5e74b7c (removed from #121) → 363e2980 (re-added via #122). Net = guarded mapping only. e5e74b7c SKIP is only safe if the rest is applied as a unit/net (else double-apply risk).
- **Gov-cloud/previous-upgrade subgroup** (70464cd1 + f51b1c5e/79c0d269/aaba2a2e/aa405349/3f82514c/2c4e2633/07619357/9b1db285): NOT features — the 2025-02 replay-onto-v1.18 and its repair/regen commits (committer date 2025-02/04, author dates 2023-2024). Net across the whole subgroup ≈ zero-or-regen for this upgrade.

## Parity verdict (cluster)
**FORK-ONLY must-carry** for everything AzureSecret (items 1-9): nothing exists upstream in any form in v1.26.0. Exceptions:
- **70464cd1 = UPSTREAMED** (mirrors upstream PR #3509 / commit 571b77166e, in v1.10.0+ and therefore in both the v1.18 fork-point and v1.26.0) → recommend SKIP (flip from NEEDS-DECISION; downgrade its Regression-Watch HIGH).
- **f51b1c5e + the 7 repair/manifest commits = SUPERSEDED/REGEN** → SKIP; regenerate manifests (`make generate manifests` + palette vendorcrd-sync); their Regression-Watch entries are covered by Tier-1 build/no-drift gates, not by carrying them.
- identity.go client-secret Cloud config = PARTIAL (upstream has it; fork extras remain).

## Recommended F4_HOTSPOTS paths
- `azure/scope/cluster.go` (dispatch split + stale Standard copies — TOP risk)
- `azure/scope/azure_secret_cloud.go` (replay-over-merge-gap; take S-tip wholesale)
- `azure/scope/identity.go`, `azure/defaults.go`, `main.go`
- `azure/services/privatedns/{link_reconciler,link_client,zone_client,privatedns}.go`
- `controllers/azuremanagedcontrolplane_reconciler.go`, `controllers/azureasomanagedcontrolplane_controller.go`
- `azure/scope/machinepool.go` (upstream ±182 F..T; re-anchor Location())
- `util/azure/azure.go`

## Decision recommendations (changes to the r2 plan)
1. **70464cd1: NEEDS-DECISION → SKIP** (UPSTREAMED; its replayed content is stale exp/ resurrection + noise; skip aa405349 with it — cancelling pair).
2. **f51b1c5e, 79c0d269, aaba2a2e, aa405349, 3f82514c, 2c4e2633, 07619357, 9b1db285: → SKIP/REGEN** (previous-upgrade repair artifacts; any real behavior they touch is owned by other clusters' feature commits and must be resolved there to S-tip net form).
3. **#117 chain: resolve as ONE unit** — apply the merged net (`7af8521e57^1..7af8521e57`) or reconstruct per-file from S tip (`git show 0164d80e:<path>`), NOT commit-by-commit (reversal pairs + evil merge 2071dd28 make sequential replay conflict-maximizing). Then #113 group PICKs become optional stepping stones (fold in).
4. **b3a9f3b5 (and the cluster.go net): HIGH regression risk** — the `…ForStandard` bodies are frozen v1.18-era upstream copies; they MUST be regenerated from v1.26.0's implementations or public-cloud clusters silently lose 8 releases of upstream LB/publicIP fixes. Require explicit reviewer diff of each ForStandard body vs v1.26.0 original.
5. **Location-mapping unit (e8b25112, 32e8a1f1, e5e74b7c, 2d63d1f9, 363e2980): treat as unit** with net = guarded mapping; keep e5e74b7c SKIP only under unit application.
6. **f1f1ebc2 (mutex): must land** — trivial once azure_secret_cloud.go is taken from S tip.
7. **Do not carry**: fork's older ASO hub-storage import in managedclusters.go (v1.26.0 uses `v1api20250801/storage`), util/remote changes (net zero), `FilterByKeyPrefix` (dead code), exp/ file resurrections, `.DS_Store`.
