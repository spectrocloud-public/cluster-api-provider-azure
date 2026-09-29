# Cluster: networking — private DNS, subnets, private cluster, load balancer, VM/scaleset lifecycle

## Behavior contract (what Palette relies on)
1. **Private-DNS FQDN scheme (PEM-483/484):** with a Palette-supplied custom `NetworkSpec.PrivateDNSZoneName`, the API-server record/FQDN is `apiserver.<clusterName>.<zone>` (shared custom zones don't collide); default generated zone keeps upstream `apiserver.<zone>`.
2. **Private cluster API-server internal LB IP:** user-pinnable static private IP (`LoadBalancerClassSpec.PrivateIP` + `IPAllocationMethod: Static`) or Azure-assigned dynamic IP (default), with the assigned IP read back from the created LB and persisted into `APIServerLB().FrontendIPs[0].PrivateIPAddress`. Upstream instead hardcodes 10.0.0.100.
3. **Single-subnet topology:** subnet role `all` places control-plane AND workers into one subnet (Palette profiles emit `role: all` — the enum value must survive for existing clusters).
4. **Deletion resilience:** AzureMachine deletion tolerates spurious ARM 5xx on VM delete (US Secret Cloud, PCP-5171) by verifying actual VM absence before failing.
5. Legacy items (providerID lowercasing, isVnetManaged caching, autoscaler replica handling, NIC/future deletion fixes) — all absorbed upstream.

## Ticket set
PEM-483, PEM-484 (private DNS); PCP-5171 (VM delete 500, via PR #125 title); rest have no ticket/PR context.

## Net effect of the chains
- **Private-DNS revert chain** a8ff143f → adb3f75a (revert) → 72906f59 (revert-of-revert) → e176e7db (conditional fix). NET = 3-arg `GeneratePrivateFQDN(zone, cluster, isPrivateDNSZoneName)` + `getHostName()` in PrivateDNSSpec + conditional `APIServerHost()`. The revert existed because v1 applied the scheme unconditionally, breaking the default-zone path; e176e7db conditions it on a custom zone being set.
- **Private-cluster LB cascade** 601023f0 → c7b9d887 → b80d4dfe → 3d6a0310 (over-removal "cleanup") → 8c6bd5fa (re-add + fix). NET = contract #2 above, debug prints gone, `getFrontendIPConfigs` sends PrivateIPAddress only for Static.
- **PR #125 chain** c42079f9 → 4f838de6 → 9c0457e2 → 4693ac4e. NET diff (verified `git diff c42079f9^ 4693ac4e`) = ONLY +azure/errors.go `IsTransientServerError` and +controllers/azuremachine_reconciler.go bounded VM-deletion verification; async.go and azuremachine_controller.go are net-zero.

## Parity verdict (cluster as a whole)
Mixed: 3 fork-only must-carry units (private-DNS NET, private-cluster-LB NET, PR-125 NET, plus SubnetAll) + 7 commits fully UPSTREAMED/SUPERSEDED (fc8f2606, 1c224e7f, 8b5ab295, 16ca18d3, 42e6a132, aef95617, 2c9ba2eb/79165ac6). Per-commit: see notes/commits/.

Upstream-side movements that shape every resolution:
- Webhooks moved to `internal/webhooks/`, defaulting to `internal/api/v1beta1/` (fbb3b95524 et al.) — all api/v1beta1 validation/default edits must be re-targeted.
- `loadbalancers.Reconcile` collapsed to generic `azure.ReconcileAll` (azure/reconcile.go:34) which discards created resources — fork must re-inline the loop to keep the dynamic-IP read-back.
- New upstream features sharing the same lines: `PrivateDNSZoneMode None` guard (59c26c982f), `PrivateDNSZoneResourceGroup` (6e62f2cdd0 = fork fc8f2606), APIServerILB feature gate + FrontendIPConfigs selection in `LBSpecs`, zone-redundant LB `Zones` in `getFrontendIPConfigs` (f8729a7feb), clusterSubnetName fallback in `SetSubnetName`.
- go-autorest fully removed upstream; fork's spectro-master is already armnetwork-ported — resolve from the fork's CURRENT net state, never from the 2022 commit texts.

## ⚠ HIGH-risk regression finding
v1.26.0 `validateAPIServerLB` (internal/webhooks/azurecluster_validation.go) added an UNCONDITIONAL internal-LB private-IP immutability check (`old.FrontendIPs[0].PrivateIPAddress != lb...` → Forbidden) and `privateIPCount != 1` requirement. The fork's dynamic-IP flow updates the spec from `""` to the Azure-assigned IP after first reconcile — upstream's new check REJECTS that update and wedges every dynamic private cluster. The 3-way resolver MUST replace this with the fork's conditioned check (immutability only for `IPAllocationMethod == "Static"`), not merge both. This is the top Gate-1 verification item for this cluster.

## Recommended F4_HOTSPOTS paths
- azure/scope/cluster.go (PrivateDNSSpec, getHostName, APIServerHost, LBSpecs)
- azure/defaults.go (GeneratePrivateFQDN)
- azure/services/loadbalancers/loadbalancers.go (re-inlined Reconcile + read-back)
- azure/services/loadbalancers/spec.go (getFrontendIPConfigs Static gate × upstream Zones)
- internal/webhooks/azurecluster_validation.go (validateAPIServerLB immutability relaxation, validateSubnets role=all)
- internal/api/v1beta1/azurecluster_default.go (setDefaultAzureClusterAPIServerLB PrivateIP, subnet role=all defaults)
- api/v1beta1/types.go + types_class.go (SubnetAll, IPAllocationMethod/PrivateIP fields, enum `;all`)
- azure/scope/machine.go (SetSubnetName merge: subnetAll bypass × upstream cluster-subnet fallback)
- controllers/azuremachine_reconciler.go (PR-125 delete verification)

## Decision recommendation changes
- **8b5ab295 PICK-VERIFY → SKIP** (upstreamed; v1.26.0 lowercases providerIDs via azprovider in scalesets/VMs/converters/managedmachinepool).
- **16ca18d3 PICK-VERIFY → SKIP** (isVnetManaged caching in base+target; 1-line residual superseded by upstream FrontendIPConfigs selection).
- **aef95617 NEEDS-DECISION → SKIP** (dead code on fork itself — const/helpers have zero consumers; upstream uses standard `replicas-managed-by` annotation). Answers the #35/#42 asymmetry: 42e6a132 is an empty commit, aef95617 has a textual residual that conflicts, but both are behaviorally obsolete.
- **1c224e7f NEEDS-DECISION → SKIP/REGEN** (generated artifacts + whitespace only; `subnetName` on managed machine pools is upstream's own field in v1.26.0).
- **Chains:** force-PICK each chain root then re-apply the chain AS A UNIT (or squash to NET): {a8ff143f, adb3f75a, 72906f59, e176e7db}, {601023f0, c7b9d887, b80d4dfe, 3d6a0310, 8c6bd5fa}, {c42079f9, 4f838de6, 9c0457e2, 4693ac4e}. Middle links must never be picked in isolation (adb3f75a deletes the feature; 3d6a0310 deletes PrivateIP; c42079f9 puts 5xx-swallowing in generic async code).
