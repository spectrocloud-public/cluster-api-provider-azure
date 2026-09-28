# adb3f75a — Revert "PEM-483: palette changes to input privatednszone"
- **cluster:** networking · **plan decision (r2):** PICK-VERIFY
- **tickets:** PEM-483 (private-DNS FQDN scheme apiserver.<clustername>.<customdns>)
- **PR:** none — direct push
- **behavior:** Pure revert of a8ff143f (exact inverse diff): restores 1-arg `GeneratePrivateFQDN` and hardcoded `apiserver` hostname. Reverted because the unconditional new FQDN scheme broke the DEFAULT (no custom zone) path — with the generated zone `<cluster>.capz.io`-style name the cluster name got duplicated in the FQDN. One day later it was re-applied (72906f59) and then properly conditioned (e176e7db).
- **pr/ticket context:** none beyond PEM-483; behavior inferred from diff + chain chronology (Nov 28 apply → Nov 29 revert → Nov 30 re-apply + fix).
- **parity vs v1.26.0:** N/A alone — middle link of a revert/revert-of-revert chain whose NET effect is captured in e176e7db. Reverting onto v1.26.0 in isolation is a no-op relative to target (target already matches the reverted state).
- **resolution guidance:** Never cherry-pick this commit in isolation. If the executor replays the chain in order, a8ff143f→adb3f75a→72906f59 cancel pairwise and only a8ff143f+e176e7db content survives; safest is to squash the 4-commit chain into one NET pick (see cluster summary). If any link conflicts, resolve by writing the NET state directly.
