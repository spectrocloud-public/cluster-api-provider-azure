# CAPI reconcile — cluster-api-provider-azure — 2026-07-10 — mode=plan

| field | value |
|--|--|
| source S | `origin/spectro-master` @ 0164d80e |
| target T | `v1.26.0` @ a9bd8578 |
| fork-point F | 1957fc65 |
| upstream U | upstream/main |
| CAPI | v1.10.4 -> v1.13.3 · go 1.25.0 · contract v1beta1 · axis 1(version-bump) · n-3 PASS(gap=3) |
| new branch | `spectro-v1.26.0-master` |
| engine hash | c28f3f6bd892406887669dfc65e3cfed |
| ⚠ merges in F..S (evil-merge check) | 0164d80e7fa224be62c5f67f55ea219c72c8ed5d da544497882aa717d5fec9f7f20c7ccd1534ab19 8a8689953746ea758897bc7c65d2e24b6106e2e4 0e33250f067c97adc0ed83a5e234e6c37243d724 31d965f74b5ca881e44bc3208ef4cbaeef262d72 e87a4c129c90a9913a5db1cfd9babc21ba3d5142 58fb03b8146df3d50b87a054ebaefbeae0a3724c acbf082d36ad893a9cd478852297f4690241df7c 849d5d6e979a83efba9496fe9028f9e07482661b 30277f42920cbb9a3acddccf74695a9636009b68 4aea7545c99774ff7e792504bd69ad873ffa6f8d 1a10eaf097cae5b984215521ed08837181447b5c 7af8521e577654294ba2b7aaa2971bf74810d31e 2071dd28c72867aeb575a03adac0d615c60f2f95 1f6189ffeccd2d65b25286af14c6bfb043b49e15 23953c3ddf4864789a01ee64fbb509cc0b49e7d7 31a7780f06a58cbc66b64e03926baf0b00281ddb  |

## ⚠️ Regression Watch
33 parity-review candidate(s) — reviewer/human must confirm the target covers the fork's behavior; set risk L/M/H. **High risk blocks auto-progression and requires explicit human sign-off.**

| sha | decision | files (parity-check) | risk | confirm |
|--|--|--|--|--|
| `248e1eb1` | PICK-VERIFY | main.go  | High | yes |
| `42eb65da` | MIXED | config/crd/bases/infrastructure.cluster.x-k8s.io_azurecluste | Med | yes |
| `0cb06ceb` | MIXED | exp/api/v1beta1/azuremanagedcontrolplane_webhook.go spectro/ | Med | yes |
| `15a9f3a5` | MIXED | Makefile azure/types.go exp/api/v1beta1/azuremanagedcontrolp | Med | yes |
| `7b180e1d` | MIXED | config/crd/bases/infrastructure.cluster.x-k8s.io_azuremanage | Med | yes |
| `debee817` | MIXED | config/crd/bases/infrastructure.cluster.x-k8s.io_azuremanage | Med | yes |
| `16ca18d3` | PICK-VERIFY | azure/scope/cluster.go  | Med | yes |
| `1c224e7f` | MIXED | azure/services/managedclusters/spec.go config/crd/bases/infr | Med | yes |
| `5fb2ad0a` | MIXED | api/v1beta1/azurecluster_default.go api/v1beta1/azurecluster | Med | yes |
| `a8ff143f` | PICK-VERIFY | azure/defaults.go azure/scope/cluster.go  | Med | yes |
| `adb3f75a` | PICK-VERIFY | azure/defaults.go azure/scope/cluster.go  | Low | yes |
| `72906f59` | PICK-VERIFY | azure/defaults.go azure/scope/cluster.go  | Low | yes |
| `e176e7db` | PICK-VERIFY | azure/defaults.go azure/scope/cluster.go  | Med | yes |
| `8b5ab295` | PICK-VERIFY | azure/services/scalesets/scalesets.go util/azure/azure.go ut | Med | yes |
| `aece2202` | MIXED | api/v1beta1/types_class.go azure/scope/managedcontrolplane.g | High | yes |
| `55d361d7` | MIXED | azure/services/managedclusters/spec.go config/crd/bases/infr | Med | yes |
| `e2975f92` | MIXED | azure/scope/managedcontrolplane.go azure/services/managedclu | High | yes |
| `15ec55de` | MIXED | api/v1beta1/azuremanagedcontrolplane_types.go azure/scope/ma | Med | yes |
| `a06f259d` | MIXED | azure/scope/managedcontrolplane.go azure/scope/managedcontro | Med | yes |
| `82068390` | MIXED | azure/scope/managedcontrolplane.go azure/scope/managedcontro | High | yes |
| `70464cd1` | MIXED | config/crd/bases/infrastructure.cluster.x-k8s.io_azuremanage | Med | yes |
| `f51b1c5e` | MIXED | api/v1beta1/azurecluster_default.go api/v1beta1/zz_generated | Med | yes |
| `79c0d269` | MIXED | azure/services/managedclusters/mock_managedclusters/managedc | Med | yes |
| `aaba2a2e` | MIXED | config/capz/manager_webhook_patch.yaml config/certmanager/ce | Med | yes |
| `aa405349` | MIXED | exp/api/v1beta1/azuremanagedcontrolplane_default.go exp/api/ | Med | yes |
| `07619357` | MIXED | config/aso/crds.yaml config/aso/kustomization.yaml spectro/g | Med | yes |
| `363e2980` | PICK-VERIFY | azure/scope/machinepool.go azure/scope/managedcontrolplane.g | Med | yes |
| `c42079f9` | PICK-VERIFY | azure/errors.go azure/services/async/async.go  | Med | yes |
| `4f838de6` | PICK-VERIFY | azure/services/async/async.go controllers/azuremachine_contr | Med | yes |
| `9c0457e2` | PICK-VERIFY | azure/services/async/async.go  | Low | yes |
| `4693ac4e` | PICK-VERIFY | controllers/azuremachine_controller.go controllers/azuremach | Med | yes |
| `604a2909` | MIXED | config/crd/bases/infrastructure.cluster.x-k8s.io_azureasoman | Med | yes |
| `24708cad` | MIXED | api/v1beta1/azuremanagedcontrolplane_types.go azure/scope/ma | High | yes |

## Decisions (baseline=T · cumulative rehearsal)
| # | sha | cat | subject | decision | reason |
|--|--|--|--|--|--|
| 1 | `248e1eb1` | F4? | Controller & Webhook Separation | PICK-VERIFY | clean apply but touches upstream-changed/F4-hotspot path — verify |
| 2 | `42eb65da` | MIXED | Added Spectro Manifests | NEEDS-DECISION | mixed generated+code — split: regen the generated part, pick the code part (human) |
| 3 | `faa067a7` | M | Spectro CICD integration | PICK | applies cleanly (fork-only) [confidence=heuristic] |
| 4 | `5e819cea` | F4? | logic to separate webhook and controller as separate pods | NEEDS-DECISION | conflict on: main.go  |
| 5 | `93449d5c` | F2? | fix for BET-3719 | PICK | applies cleanly (fork-only) [confidence=heuristic] |
| 6 | `1a8918cf` | M | Spectro CICD integration II | NEEDS-DECISION | conflict on: Dockerfile  |
| 7 | `a822bc84` | F2? | fix for BET-3963 | SKIP | empty on cumulative tree (duplicate — already applied by a prior pick) |
| 8 | `0cb06ceb` | MIXED | Validation disabled for AzureManagedControlPlane | NEEDS-DECISION | mixed generated+code — split: regen the generated part, pick the code part (human) |
| 9 | `15a9f3a5` | MIXED | windows support-added OSType to ammp and amcp | NEEDS-DECISION | mixed generated+code — split: regen the generated part, pick the code part (human) |
| 10 | `42e6a132` | F2? | Updated the check for updating machinepool when using autoscalar (#35) | SKIP | empty on cumulative tree (duplicate — already applied by a prior pick) |
| 11 | `9972a7a1` | F4? | Suppress warning messages for ServicePrincipal auth | NEEDS-DECISION | conflict on: controllers/azurejson_machine_controller.go controllers/azurejson_machinepool_controller.go controllers/azurejson_machinetemplate_controller.go  |
| 12 | `aef95617` | F4? | Updated the check for updating machinepool when using autoscalar (#42) | NEEDS-DECISION | conflict on: azure/const.go  |
| 13 | `7b180e1d` | MIXED | AKS static placement for cross RG | NEEDS-DECISION | mixed generated+code — split: regen the generated part, pick the code part (human) |
| 14 | `debee817` | MIXED | Fixed osType issue | NEEDS-DECISION | mixed generated+code — split: regen the generated part, pick the code part (human) |
| 15 | `35e14baa` | F2? | PCP-328 fix | SKIP | empty on cumulative tree (duplicate — already applied by a prior pick) |
| 16 | `858364c5` | F4? | Fixed upgrade issue from v1alpha4 to v1beta1 | NEEDS-DECISION | conflict on: spectro/generated/core-base.yaml spectro/generated/core-global.yaml  |
| 17 | `16ca18d3` | F4? | AKS: enable isVnetManaged, add caching | PICK-VERIFY | clean apply but touches upstream-changed/F4-hotspot path — verify |
| 18 | `0fc4e9d8` | F4? | updated computediff authorized ip range logic for nil check | NEEDS-DECISION | conflict on: azure/services/managedclusters/spec.go  |
| 19 | `1c224e7f` | MIXED | Added subnetName spec to azure machine pool. | NEEDS-DECISION | mixed generated+code — split: regen the generated part, pick the code part (human) |
| 20 | `5fb2ad0a` | MIXED | Use same subnet for cp and workers | NEEDS-DECISION | mixed generated+code — split: regen the generated part, pick the code part (human) |
| 21 | `601023f0` | F4? | private cluster change | NEEDS-DECISION | conflict on: azure/services/loadbalancers/loadbalancers.go internal/api/v1beta1/azurecluster_default.go  |
| 22 | `3d6a0310` | F4? | cleanup | NEEDS-DECISION | conflict on: api/v1beta1/types_class.go azure/scope/cluster.go azure/services/loadbalancers/loadbalancers.go azure/services/loadbalancers/spec.go internal/api/v1beta1/azurecluster_default.go  |
| 23 | `37d4d8ec` | F2? | make file | SKIP | empty on cumulative tree (duplicate — already applied by a prior pick) |
| 24 | `cbfe1d19` | REGEN | new crds | REGENERATE | generated artifact — regenerate via make (not cherry-picked) |
| 25 | `c7b9d887` | F4? |  add check for frontend ip | NEEDS-DECISION | conflict on: azure/services/loadbalancers/loadbalancers.go  |
| 26 | `a8ff143f` | F4? | PEM-484: palette changes to input privatednszone | PICK-VERIFY | clean apply but touches upstream-changed/F4-hotspot path — verify |
| 27 | `b80d4dfe` | F4? | more nil checks | NEEDS-DECISION | conflict on: azure/services/loadbalancers/loadbalancers.go  |
| 28 | `adb3f75a` | F4? | Revert "PEM-483: palette changes to input privatednszone" | PICK-VERIFY | clean apply but touches upstream-changed/F4-hotspot path — verify |
| 29 | `72906f59` | F4? | Revert "Revert "PEM-483: palette changes to input privatednszone"" | PICK-VERIFY | clean apply but touches upstream-changed/F4-hotspot path — verify |
| 30 | `e176e7db` | F4? | fix private dns issue | PICK-VERIFY | clean apply but touches upstream-changed/F4-hotspot path — verify |
| 31 | `2c9ba2eb` | F2? | Resolved NIC deletion issue | SKIP | empty on cumulative tree (duplicate — already applied by a prior pick) |
| 32 | `8b5ab295` | F4? | enforce lowercase providerID RG to match cloud-provider-azure | PICK-VERIFY | clean apply but touches upstream-changed/F4-hotspot path — verify |
| 33 | `79165ac6` | F2? | Ignore error for other future types in delete functionality | SKIP | empty on cumulative tree (duplicate — already applied by a prior pick) |
| 34 | `aece2202` | MIXED | adding UserAssignedIdentities to AzureManagedControlPlane | NEEDS-DECISION | mixed generated+code — split: regen the generated part, pick the code part (human) |
| 35 | `55d361d7` | MIXED | adding OutboundType to AzureManagedControlPlane | NEEDS-DECISION | mixed generated+code — split: regen the generated part, pick the code part (human) |
| 36 | `a2b1a593` | F4? | fix for add-on profile upgrade | NEEDS-DECISION | conflict on: azure/services/managedclusters/spec.go  |
| 37 | `1c814218` | F4? | update removal flow | NEEDS-DECISION | conflict on: azure/services/managedclusters/spec.go  |
| 38 | `cb8ef8b4` | F4? | Fix CodeSmell | NEEDS-DECISION | conflict on: azure/services/loadbalancers/loadbalancers.go  |
| 39 | `e2975f92` | MIXED |  PCP-1467: UserAssignedIdentities Fix | NEEDS-DECISION | mixed generated+code — split: regen the generated part, pick the code part (human) |
| 40 | `efd40404` | F4? | PCP-1467: PR Comments Resolution I | NEEDS-DECISION | conflict on: azure/services/managedclusters/spec.go  |
| 41 | `70b75f13` | F2? | PCP-1567: AKS ServiceCIDR Fix (#81) | SKIP | empty on cumulative tree (duplicate — already applied by a prior pick) |
| 42 | `15ec55de` | MIXED | PCP-1595 and PCP-1596 fix (#83) | NEEDS-DECISION | mixed generated+code — split: regen the generated part, pick the code part (human) |
| 43 | `b671d486` | F2? | fix webhook issue (#85) | SKIP | empty on cumulative tree (duplicate — already applied by a prior pick) |
| 44 | `ad0f208f` | F4? | PEM-2613: Fix Cipher Suit issue | NEEDS-DECISION | conflict on: go.mod go.sum  |
| 45 | `f9cfc76b` | F4? | Update | NEEDS-DECISION | conflict on: main.go  |
| 46 | `a06f259d` | MIXED | AKS UpgradeChannels (#92) | NEEDS-DECISION | mixed generated+code — split: regen the generated part, pick the code part (human) |
| 47 | `82068390` | MIXED | disableLocal accounts (#91) | NEEDS-DECISION | mixed generated+code — split: regen the generated part, pick the code part (human) |
| 48 | `d8e1ce40` | F4? | ensure to not downgrade an auto-upgraded cluster (#95) | NEEDS-DECISION | conflict on: azure/scope/managedcontrolplane.go azure/scope/managedcontrolplane_test.go azure/services/managedclusters/mock_managedclusters/managedclusters_mock.go azure/services/managedclusters/spec.go azure/services/managedclusters/spec_test.go go.mod go.sum spectro/generated/core-global.yaml  |
| 49 | `70464cd1` | MIXED | Cherrypicked upstream azure env changes for managed cluster aks on gov | NEEDS-DECISION | mixed generated+code — split: regen the generated part, pick the code part (human) |
| 50 | `f51b1c5e` | MIXED | Additional Changes & Fixes | NEEDS-DECISION | mixed generated+code — split: regen the generated part, pick the code part (human) |
| 51 | `79c0d269` | MIXED | Manifest Fixes | NEEDS-DECISION | mixed generated+code — split: regen the generated part, pick the code part (human) |
| 52 | `aaba2a2e` | MIXED | Manifest Fixes II | NEEDS-DECISION | mixed generated+code — split: regen the generated part, pick the code part (human) |
| 53 | `aa405349` | MIXED | Removing extra exp files and code | NEEDS-DECISION | mixed generated+code — split: regen the generated part, pick the code part (human) |
| 54 | `3f82514c` | F4? | Fix for Subnets & package changes | NEEDS-DECISION | conflict on: api/v1beta1/types.go azure/services/loadbalancers/loadbalancers.go internal/api/v1beta1/azurecluster_default.go internal/api/v1beta1/azuremachine_default.go  |
| 55 | `2c4e2633` | F4? | Fix for Webhook Validations & ManagedCluster Spec Updation | NEEDS-DECISION | conflict on: azure/services/managedclusters/spec.go internal/webhooks/azuremanagedcontrolplane_validation.go  |
| 56 | `07619357` | MIXED | Manifest Changes | NEEDS-DECISION | mixed generated+code — split: regen the generated part, pick the code part (human) |
| 57 | `9b1db285` | F2? | Removal of comments | NEEDS-DECISION | conflict on: internal/api/v1beta1/azurecluster_default.go  |
| 58 | `5f44feb4` | M* | Vulnerability fix | SKIP | skip-list (explicit human decision) |
| 59 | `034b0daf` | F2? | PEM-7501: Loaded custom azure environment from file | PICK | applies cleanly (fork-only) [confidence=heuristic] |
| 60 | `e3c6f868` | F2? | Fix for webhook validation of AzureManagedControlPlane VNET Resource G | PICK | applies cleanly (fork-only) [confidence=heuristic] |
| 61 | `b3215651` | F2? | PEM-7501: Changed to load all json files instead of a single file for  | PICK | applies cleanly (fork-only) [confidence=heuristic] |
| 62 | `6d749add` | F2? | Bypassing webhook validation for Azure SharedGallery | PICK | applies cleanly (fork-only) [confidence=heuristic] |
| 63 | `fc8f2606` | MIXED | Support for Azure Private DNS Zone to be present in any resource group | SKIP | already in T (patch-equivalent in target) |
| 64 | `8c6bd5fa` | F4? | Fix Azure Private Cluster | NEEDS-DECISION | conflict on: api/v1beta1/types_class.go azure/scope/cluster.go azure/services/loadbalancers/spec.go internal/api/v1beta1/azurecluster_default.go internal/webhooks/azurecluster_validation.go  |
| 65 | `fb81b479` | F2? | PEM-7501: Fixed log statements | PICK | applies cleanly (fork-only) [confidence=heuristic] |
| 66 | `45d9bb84` | F2? | Change AzureCluster Webhook Validation for Shared & Compute Gallery | PICK | applies cleanly (fork-only) [confidence=heuristic] |
| 67 | `c9809113` | F4? | fix: reuse global TracerProvider | SKIP | already in T (patch-equivalent in target) |
| 68 | `34c9590a` | F2? | Minor fix for the path reading | PICK | applies cleanly (fork-only) [confidence=heuristic] |
| 69 | `ee19ef20` | F4? | Removed Dockerfile and Makefile | NEEDS-DECISION | conflict on: azure/defaults.go azure/scope/cluster.go azure/scope/identity.go azure/scope/machinepoolmachine.go azure/scope/managedcontrolplane.go azure/services/loadbalancers/loadbalancers.go azure/services/managedclusters/spec_test.go controllers/azureasomanagedcontrolplane_controller.go  |
| 70 | `1b4fdec4` | F4? | Global vars for CAPZ | NEEDS-DECISION | conflict on: azure/defaults.go azure/scope/identity.go internal/api/v1beta1/azurecluster_default.go util/remote/client.go util/remote/client_test.go  |
| 71 | `fde3f374` | F4? | Changed init cert acquire method | NEEDS-DECISION | conflict on: azure/scope/azure_secret_cloud.go azure/scope/identity.go  |
| 72 | `a0e29487` | F4? | Removed unnecessary updates | NEEDS-DECISION | conflict on: azure/services/loadbalancers/loadbalancers.go azure/services/managedclusters/spec_test.go  |
| 73 | `7a89534e` | F4? | removed unused import | NEEDS-DECISION | conflict on: azure/scope/machinepoolmachine.go azure/scope/managedcontrolplane.go  |
| 74 | `7f0e76c4` | F4? | removed mngplane modifications | NEEDS-DECISION | conflict on: azure/scope/managedcontrolplane.go  |
| 75 | `b7f8d03b` | F4? | Reversal | NEEDS-DECISION | conflict on: azure/scope/machinepoolmachine.go  |
| 76 | `cfabdbf5` | F4? | Final Remove for unrelated files | NEEDS-DECISION | conflict on: azure/services/loadbalancers/loadbalancers.go azure/services/loadbalancers/spec_test.go  |
| 77 | `50c8656c` | F4? | Restore test files | NEEDS-DECISION | conflict on: azure/services/loadbalancers/spec_test.go azure/services/managedclusters/spec_test.go  |
| 78 | `c1a28ceb` | F2? | removed the real hard coded path for cert | NEEDS-DECISION | conflict on: azure/scope/azure_secret_cloud.go  |
| 79 | `b86eb725` | F4? | Further removed unnecessary stuff | NEEDS-DECISION | conflict on: azure/scope/azure_secret_cloud.go azure/services/managedclusters/managedclusters.go controllers/azureasomanagedcontrolplane_controller.go  |
| 80 | `eb14c310` | F4? | Applied Vishu's Comments | NEEDS-DECISION | conflict on: azure/const.go azure/scope/azure_secret_cloud.go azure/scope/cluster.go azure/services/managedclusters/managedclusters.go azure/services/privatedns/link_reconciler.go controllers/azureasomanagedcontrolplane_controller.go controllers/azuremanagedcontrolplane_reconciler.go  |
| 81 | `f918f3cf` | F4? | added new support for AzureSecret Region name and cfg map | NEEDS-DECISION | conflict on: azure/defaults.go azure/scope/azure_secret_cloud.go  |
| 82 | `d13d31d0` | F4? | Updated as comments, will do test later | NEEDS-DECISION | conflict on: azure/scope/azure_secret_cloud.go azure/scope/cluster.go azure/services/managedclusters/spec_test.go azure/services/privatedns/link_reconciler.go controllers/azureasomanagedcontrolplane_controller.go controllers/azuremanagedcontrolplane_reconciler.go main.go  |
| 83 | `c95a3413` | F4? | Fix Vishu's comment | NEEDS-DECISION | conflict on: main.go  |
| 84 | `e8b25112` | F4? | Adding support for ussec to eastus2 location mapping for azuresecret c | NEEDS-DECISION | conflict on: azure/scope/cluster.go  |
| 85 | `32e8a1f1` | F4? | Decouple the env checking and mapping | NEEDS-DECISION | conflict on: azure/scope/cluster.go azure/scope/machinepool.go azure/scope/managedcontrolplane.go util/azure/azure.go util/azure/azure_test.go  |
| 86 | `b3a9f3b5` | F4? | Recover cluster setting with ENV checking, and apply TLS version fix | NEEDS-DECISION | conflict on: azure/scope/azure_secret_cloud.go azure/scope/cluster.go  |
| 87 | `e5e74b7c` | F4? | Decouple location mapping patch from #119, Only fix ticket for this PR | SKIP | empty on cumulative tree (duplicate — already applied by a prior pick) |
| 88 | `2d63d1f9` | F4? | Using function wrapper for location mapping in cluster.go | NEEDS-DECISION | conflict on: azure/scope/cluster.go  |
| 89 | `363e2980` | F4? | Location mapping support for AzureSecret | PICK-VERIFY | clean apply but touches upstream-changed/F4-hotspot path — verify |
| 90 | `f1f1ebc2` | F2? | adding a lock when set up env for multiple controllers | NEEDS-DECISION | conflict on: azure/scope/azure_secret_cloud.go  |
| 91 | `c42079f9` | F4? | Adding enhanced logic for checking the vm deletions | PICK-VERIFY | clean apply but touches upstream-changed/F4-hotspot path — verify |
| 92 | `4f838de6` | F4? | Final fix for azuremachine_controller | PICK-VERIFY | clean apply but touches upstream-changed/F4-hotspot path — verify |
| 93 | `9c0457e2` | F4? | remove uncessary file changes | PICK-VERIFY | clean apply but touches upstream-changed/F4-hotspot path — verify |
| 94 | `4693ac4e` | F4? | Change the Fix to service level | PICK-VERIFY | clean apply but touches upstream-changed/F4-hotspot path — verify |
| 95 | `94233eeb` | F4? | Merge pull request #126 from spectrocloud/PCP-5293 | NEEDS-DECISION | conflict on: api/v1beta1/azuremanagedcluster_webhook.go api/v1beta1/azuremanagedclustertemplate_webhook.go go.mod go.sum internal/exp/webhooks/azuremachinepoolmachine_webhook.go internal/webhooks/azurecluster_webhook.go internal/webhooks/azureclusteridentity_webhook.go internal/webhooks/azureclustertemplate_webhook.go  |
| 96 | `97122011` | M* | Update Go version in spectro-release workflow to 1.24.12 | PICK | applies cleanly (fork-only) [confidence=heuristic] |
| 97 | `ae875a02` | M* | PCP-6027 Update go version in spectro release yaml (#132) | PICK | applies cleanly (fork-only) [confidence=heuristic] |
| 98 | `06a12b7f` | M* | PCP-6027 Update go version in spectro release yaml (#133) | SKIP | empty on cumulative tree (duplicate — already applied by a prior pick) |
| 99 | `79f5be32` | F4? | PCP-6137 Handle CAPZ vulenerability | SKIP | skip-list (explicit human decision) |
| 100 | `d3e72af4` | F4? | PCP-6137 Handle CAPZ vulenerability | SKIP | skip-list (explicit human decision) |
| 101 | `fc3e22b4` | F4? | add autobackport | NEEDS-DECISION | conflict on: .gitignore  |
| 102 | `604a2909` | MIXED | Fix Spectro Manifest Generation Script (#131) | NEEDS-DECISION | mixed generated+code — split: regen the generated part, pick the code part (human) |
| 103 | `24708cad` | MIXED | PCP-6412: added enableAzureRBAC flag for aadProfile (#142) | NEEDS-DECISION | mixed generated+code — split: regen the generated part, pick the code part (human) |
| 104 | `d0610cb2` | M* | PCP-6734 Update go version for build | NEEDS-DECISION | conflict on: Dockerfile go.mod go.sum  |
| 105 | `364fe27b` | F4? | dockerfile: fixed duplicate builds (#147) (#149) | NEEDS-DECISION | conflict on: Dockerfile  |
| 106 | `45c6fc69` | M* | PCP-6734 Fix go vulns | SKIP | skip-list (explicit human decision) |
| 107 | `cb5b4b16` | M* | PCP-6787 Fix CVEs | SKIP | skip-list (explicit human decision) |
| 108 | `f1abc73a` | F4? | fix(converters): forward OsSKU to embedded ManagedCluster agent pool p | NEEDS-DECISION | conflict on: azure/converters/managedagentpool.go  |
| 109 | `22c3de30` | M* | PCP-6959 : updated go version & packages to fix vulnerabilities | NEEDS-DECISION | conflict on: .github/workflows/spectro-release.yaml Makefile  |

## Summary
- PICK=23 (PICK-VERIFY=12 · PICK-RESOLVE=0) · SKIP=17 · REGENERATE=1 · NEEDS-DECISION=68
- INCOMPLETE: unresolved=68 · regen/verify-pending=1 · regression-watch=33 · commit-failures=0
- Branch NOT mergeable while INCOMPLETE>0 or any High regression unconfirmed.
- Legend: PICK auto · PICK-VERIFY auto+verify · PICK-RESOLVE force+resolver · SKIP · NEEDS-DECISION human · REGEN make · MIXED split(human)
