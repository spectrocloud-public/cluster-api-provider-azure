# Cluster: aks-managed-clusters — behavior contract & parity (v1.26.0)

22 commits: AzureManagedControlPlane / AzureManagedMachinePool / managedclusters-service behavior.
Tickets: PCP-328, PCP-1467, PCP-1567, PCP-1595, PCP-1596, PCP-1614 (PR #92), PCP-1615 (PR #91), PCP-6412, PCP-6938.

## Overall behavior contract (what Palette relies on)
1. AKS clusters configurable via AMCP/AMMP for: Windows/osType pools, osSKU (AzureLinux), outboundType, DNSPrefix, fqdnSubdomain (private cluster + custom private DNS zone), privateDNSZone, serviceCIDR+dnsServiceIP (both always sent), userAssignedIdentities, addonProfiles, upgradeChannel auto-upgrade, disableLocalAccounts (with token-based kubeconfig, no kubelogin), enableAzureRBAC opt-in (AAD authN with K8s RBAC authZ).
2. No downgrade of auto-upgraded clusters; no reconcile/PUT/event spam at steady state; no SP-auth warning-event spam.
3. Legacy (CAPZ v1.3.2-era) Palette clusters keep upgrading: empty `virtualNetwork.resourceGroup` tolerated on update; osType defaulted to Linux.

## Net effect of the chain (cascades resolved)
- **computeDiff cascade (0fc4e9d8 → a2b1a593 → 1c814218 → e2975f92 → efd40404 + PCP-328/35e14baa):** the SDK-era client-side diff/no-op-suppression/addon-removal/identity-forcing machinery is ALL dead at 0164d80e — commented out (spec.go:762-816) or stubbed (`computeDiffOfNormalizedClusters` spec.go:944, unused). NET behavior on today's fork: none. ASO supersedes it at v1.26.0. None of these should be carried.
- **PCP-1467 unit (aece2202 + e2975f92 + efd40404):** NET = a fork-only API field `spec.userAssignedIdentities` that is accepted but IGNORED at reconcile (scope populates `managedClusterSpec.UserAssignedIdentities`, spec.go never reads it; identity driven by upstream `spec.identity`/getIdentity). The ticket's contract is not actually implemented by fork code anymore.
- **Empty commits:** 35e14baa (PCP-328) and 70b75f13 (PCP-1567) have NO tree change in the rebased history — both SKIPs verified correct (see per-commit notes; ServiceCIDR contract exists verbatim upstream: spec.go:550-551 + calculateDNSServiceIP).
- **Webhook divergence (net of 0cb06ceb/7b180e1d/858364c5 era):** the fork's entire surviving webhook delta vs fork-point 1957fc65 is 2 hunks in `api/v1beta1/azuremanagedcontrolplane_webhook.go`: (a) `validateManagedClusterNetwork` tolerates a missing owner Cluster (skip instead of InternalError); (b) `validateVirtualNetworkUpdate` allows update when OLD `virtualNetwork.resourceGroup == ""` (legacy v1.3.2 Palette clusters). v1.26.0 moved webhooks to `internal/webhooks/` — these 2 hunks must be re-applied there.

## Parity verdict (cluster as a whole)
**Mostly UPSTREAMED — only 3 things genuinely carry.** Over the years LochanRn et al. upstreamed the fork's AKS features (auto-upgrade channel 631a4b27c5 is literally the same author; disableLocalAccounts/token-kubeconfig, downgrade-protection, outboundType, DNSPrefix, privateDNSZone, cross-RG VNet, osType/OsSKU all present in v1.26.0).

Per-commit: 0cb06ceb SUPERSEDED/REGEN · 15a9f3a5 UPSTREAMED(+dead code) · debee817 SUPERSEDED/REGEN · 858364c5 UPSTREAMED/REGEN · 7b180e1d UPSTREAMED/REGEN · 9972a7a1 **FORK-ONLY carry** · 0fc4e9d8 SUPERSEDED · aece2202 FORK-ONLY-DEAD (decision) · 55d361d7 UPSTREAMED · a2b1a593 SUPERSEDED-dead · 1c814218 SUPERSEDED-dead (latent product gap noted) · cb8ef8b4 SUPERSEDED · e2975f92 SUPERSEDED-dead (decision) · efd40404 SUPERSEDED-dead · 70b75f13 UPSTREAMED (SKIP ok) · 15ec55de **PARTIAL (FqdnSubdomain carries)** · a06f259d UPSTREAMED · 82068390 UPSTREAMED · d8e1ce40 UPSTREAMED · 35e14baa UPSTREAMED-by-architecture (SKIP ok) · 24708cad **FORK-ONLY carry, HIGH** · f1abc73a UPSTREAMED.

### Must-carry list (the real payload of this cluster)
1. **24708cad enableAzureRBAC (HIGH):** AADProfile field + scope wire at v1.26.0 `azure/scope/managedcontrolplane.go:595` (upstream hardcodes `EnableAzureRBAC = Managed`). Dropping silently flips AAD clusters to Azure RBAC.
2. **15ec55de FqdnSubdomain slice only:** AMCP field + scope pass + `managedCluster.Spec.FqdnSubdomain` in Parameters + CRD regen (upstream has no AMCP-level fqdnSubdomain). Drop DockerBridgeCidr (dead, AKS retired it).
3. **9972a7a1 SP-warning suppression:** re-comment log+event in the 3 azurejson controllers (upstream still warns for SP users = Palette).
4. **Webhook 2-hunk delta** (missing-Cluster tolerance + empty-VNet-RG upgrade path) re-applied in `internal/webhooks/azuremanagedcontrolplane_webhook.go` / `_validation.go`.

### Open decision (Gate 1)
- **PCP-1467 userAssignedIdentities:** field is dead on the fork today. Ask Palette: still emitted by the AKS pack? If yes → wire it to `Identity{UserAssigned, UserAssignedIdentityResourceID: uai[0].ProviderID}` at v1.26.0 (few lines in scope) or migrate the pack; if no → drop field + dead scope code. Don't blind-carry.
- **Addon removal (1c814218):** disable-on-removal is dead on the fork AND absent upstream — latent gap, not an upgrade regression. Flag to product; needs a new fix if CBT pack-ops expects it.

## Recommended F4_HOTSPOTS paths
- `azure/scope/managedcontrolplane.go` (enableAzureRBAC line, FqdnSubdomain pass, dead UAI code)
- `azure/services/managedclusters/spec.go` (FqdnSubdomain in Parameters; big upstream refactor into configure* helpers makes 3-way merges here treacherous)
- `internal/webhooks/azuremanagedcontrolplane_webhook.go` + `azuremanagedcontrolplane_validation.go` (new home for the fork's 2 webhook relaxations; DNSPrefix/outboundType immutability additions)
- `api/v1beta1/azuremanagedcontrolplane_types.go` + `api/v1beta1/types_class.go` (AADProfile.EnableAzureRBAC, FqdnSubdomain, userAssignedIdentities decision)
- `controllers/azurejson_{machine,machinepool,machinetemplate}_controller.go` (warning suppression)
- `azure/converters/managedagentpool.go` (take upstream full-field list; hub moved to v1api20250801/storage — fork test pinned to v1api20240901 will not compile)

## Decision recommendation changes vs r2 plan
- Confirm both SKIPs (35e14baa, 70b75f13) — verified safe with evidence.
- Downgrade to SKIP/take-upstream: 0fc4e9d8, a2b1a593, 1c814218, cb8ef8b4, e2975f92, efd40404, 55d361d7, a06f259d, 82068390, d8e1ce40, 15a9f3a5, debee817, 858364c5, 7b180e1d, 0cb06ceb (REGEN for their generated/spectro parts).
- Keep NEEDS-DECISION → force-PICK (reduced scope): 24708cad (carry as-is minus yaml), 9972a7a1, 15ec55de (FqdnSubdomain hunks only).
- aece2202 stays NEEDS-DECISION pending the Palette pack answer above.
- ASO hub-version note for the executor: fork code/tests referencing `v1api20240901` must move to `v1api20250801/storage` at v1.26.0 (converter, any carried tests).
