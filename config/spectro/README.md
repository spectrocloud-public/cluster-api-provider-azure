# config/spectro — Spectro fork manifest overlay

Fork-only manifest changes live HERE, as a kustomize overlay on top of the pristine
upstream `config/` tree — never as edits to upstream files. This mirrors the structure
palette uses for this provider (`palette:config/vendorcrd/azure/` — `crd-patch.yaml`,
`palette-extras.yaml`, per-bucket kustomization patches).

```
config/spectro/
├── kustomization.yaml    # overlay entrypoint: upstream ../default + fork-only patches
├── spectro-extras.yaml   # fork-only resources not present upstream (currently none)
├── crd-patch.yaml        # CANONICAL fork CRD schema delta (sync source for palette —
│                         #   NOT applied here; config/crd/bases already carry the fields)
└── README.md
```

`spectro/vendorcrd/` (repo root) is a historical artifact with the same crd-patch
content; it is not maintained going forward — this directory is the source of truth.

## Build

```bash
kustomize build config/spectro     # upstream v1.26.0 shape + fork-only changes
```

## Invariants

1. **Upstream `config/**` stays pristine.** The only permitted divergence is
   `config/crd/bases/` — those files are GENERATED (`make generate-manifests`) from the
   fork's Go types, so the fork's carried API fields land there mechanically.
2. **Future fork-only manifest changes are overlay edits** — add resources to
   `spectro-extras.yaml` (and uncomment it in `kustomization.yaml`) or add `patches:`
   entries to `kustomization.yaml`. Never edit files under `config/{default,manager,
   webhook,certmanager,aso,capz,crd,rbac}/`.
3. **`./crd-patch.yaml` is the canonical CRD delta** (NOT part of this overlay build —
   config/crd/bases already carry the fields). palette's
   `config/vendorcrd/azure/crd-patch.yaml` must stay identical to it (vendorcrd-sync
   layers it onto the UPSTREAM CRDs at sync time — the fork ships no release manifest).
   Regenerate/validate after each upstream bump (see Verification).

## Fork delta inventory (at upstream v1.26.0)

`kustomize build config/spectro` vs upstream `kustomize build config/default` differs by
exactly the fork CRD schema additions (4 CRDs, 19 ops — see `crd-patch.yaml`):
`ipAllocationMethod`/`privateIP` on the 3 load balancers, subnet/bastion `role: all`
enum, `enableAzureRBAC`, `fqdnSubdomain`.

### Dropped at the v1.26.0 upgrade (intentionally — do not restore)

The pre-upgrade fork (`spectro-master`, upstream base v1.18.0) carried four more
manifest edits, all superseded:

| Old fork edit | Disposition |
|---|---|
| manager arg `--metrics-bind-addr=localhost:8080` | Palette-owned: applied by `palette:config/vendorcrd/azure/global/kustomization.yaml` (`127.0.0.1:8080`). Fork code carries the flag support. |
| ASO arg pinned to `--crd-pattern=` (empty) | Superseded: palette's upstream-synced manifest pins the explicit v1.26 ASO CRD list. |
| `cert-manager.io/inject-ca-from` on 12 template CRDs | Obsoleted upstream: v1.26 restructured CRD cainjection; those CRDs no longer declare conversion webhooks. |
| webhook cert secretName `capz-webhook-service-webhook-service-cert` | Obsolete artifact of the old fork webhook wiring; nothing references it. |

The fork-authored files under `config/default/` (aad-pod-identity deployment, manager
patches, namespace) are ORPHANED — referenced by no kustomization on any branch, never
part of a build. Their deployable equivalents live in palette
(`palette-extras.yaml` / global kustomization patches). Each carries an ORPHANED header.

## Verification (run after every upstream bump)

```bash
# 1. Overlay build = upstream + fork delta only
kustomize build config/spectro > /tmp/fork.yaml
git -C . worktree add /tmp/wt-upstream <upstream-tag>
kustomize build /tmp/wt-upstream/config/default > /tmp/upstream.yaml
diff -u /tmp/upstream.yaml /tmp/fork.yaml     # additions must match crd-patch.yaml ops

# 2. crd-patch.yaml reproduces the fork CRDs from upstream ones
#    (kustomize build of upstream 4 CRDs + crd-patch == kustomize build of fork bases)
#    Mismatch means: regenerate crd-patch.yaml from `git diff <upstream-tag> -- config/crd/bases`
#    and sync palette's copy.
```
