# Validator Jailing Corner Cases

This document is the formal review and verification record for the jailing/slashing corner-case fixes in dogfood/operator/oracle integration.

## Purpose

- Provide a stable reviewer guide for critical jailing/slashing behavior.
- Map checklist IDs (`A`–`K`) to code paths and tests.
- Record findings, fixes, and why the resulting behavior is safe.

## Scope

Primary commit scope reviewed:

- `d8dd3f60` fix: correctly jail and/or tombstone
- `af55e433` fix: call `AfterValidatorRemoved` hook
- `48ff8054` fix: copy missed block array
- `4f4a2fa6` fix: exclude frozen operators
- `bc84b46c` fix(operator): block frozen operators power
- `5edaed39` fix: copy signing info correctly
- `81185b27` fix: reset signing info on maturity

Major touched paths:

- `x/dogfood/keeper/impl_sdk.go`
- `x/dogfood/keeper/impl_operator_hooks.go`
- `x/dogfood/keeper/abci.go`
- `x/operator/keeper/consensus_keys.go`
- `x/operator/keeper/slash.go`
- `x/oracle/keeper/feedermanagement/feedermanager.go`
- tests in `x/dogfood/keeper/key_change_escape_test.go`, `x/dogfood/types/hooks_test.go`

## Critical Objects and Lifecycles

### Consensus key lifecycle

1. Operator current key: `(operator, chainID) -> current cons key`.
2. Reverse mapping retained during unbonding: `(chainID, consAddr) -> operator`.
3. Key replacement hook schedules old key prune at `GetUnbondingCompletionEpoch`.
4. Dogfood `EndBlock` prune path:
   - `AfterValidatorRemoved`
   - delete reverse mapping
   - clear missed-bit array
   - `ResetValidatorSigningInfo` (non-tombstoned only)

### Slashing/signing info lifecycle

- `CopyValidatorSigningInfo` handles key migration and infraction shaping:
  - `DOUBLE_SIGN`: tombstone + permanent jail on new key.
  - `DOWNTIME`: jail duration + reset counters/bit-array on new key.
  - `UNSPECIFIED`: migrate missed-block debt shape for key-rotation semantics.

### Freeze lifecycle

- Double-sign path freezes operator globally in x/operator.
- Frozen operators:
  - cannot re-enter active power paths,
  - cannot effectively unjail via dogfood staking bridge,
  - remain blocked for key/opt-in mutations by keeper checks.

## Key Findings and Fixes

### Finding FND-1: slash-success coupling (fixed)

- **Issue:** `CopyValidatorSigningInfo` could run even when operator economic slash did not commit.
- **Impact:** cross-module state drift (x/slashing migrated state without committed operator slash effects).
- **Fix:** introduced `SlashDogfoodInfraction(...)(attempted, err)` and gated copy on `attempted && err == nil`.
- **Safety:** prevents partial migration side effects while preserving double-sign safety behavior.

### Finding FND-2: repeated double-sign freeze panic risk (fixed)

- **Issue:** repeated direct double-sign slash could call `FreezeOperator` on already frozen operator and panic.
- **Fix:** guard freeze path with `IsOperatorFrozen` check before `FreezeOperator`.
- **Safety:** freeze path becomes idempotent for repeated inputs.

## Checklist Coverage (A–K)

All checklist items are closed for this scope. IDs are retained so tests and future reviews can reference them consistently.

### A. `CopyValidatorSigningInfo` and callers

- A1–A6: ordering, stale-key behavior, `UNSPECIFIED` semantics, and `!found` handling validated.

### B. Consensus-key prune and evidence window

- B0–B4: lifecycle ordering and prune semantics validated; B2 explicitly checks evidence window vs unbonding assumptions.

### C. Frozen operator behavior

- C1–C2: mutation-path checks and unjail no-op behavior validated/documented.

### D. Invariants around key changes

- D1–D5: rotation, same-epoch behavior, post-prune no-op slash, and slashing invariants validated.

### E. Slash orchestration and prune hooks

- E1–E6: freeze, unjail behavior, prune retry semantics, slash-record behavior, replay handling validated.

### F. Failure-mode and panic semantics

- F1–F3: freeze idempotency, panic boundary trace, and slash-attempt observability validated.

### G. Hook composition and ordering

- G1–G3: current wiring and retry/idempotency semantics validated for in-scope hooks.

### H. Config/upgrade safety

- H1–H3: in-scope param and lifecycle assumptions documented and traced.

### I. Oracle integration edges

- I1–I3: stale-key/rotation boundary and jail coherence semantics validated (trace + targeted tests).

### J. Stronger invariants

- J1–J3: deterministic randomized lifecycle and key invariants covered.

### K. Ops/observability

- K1–K3: structured logs and alert recommendations documented.

## Test Index (Checklist-ID -> Test)

- A3/B4/D2: `TestChecklist_A3_B4_KeyRotationSigningInfoInvariant`
- B2: `TestChecklist_B2_EvidenceWindowCoversDogfoodUnbonding`
- D3: `TestChecklist_D3_DoubleKeyRotationInvariants`
- D5/J2: `TestChecklist_D5_PostPruneReverseLookupAndSlashNoOp`
- Same-epoch replacement risk model: `TestSameEpochDoubleKeyRotation_SkipsSecondHookRisk`
- E1: `TestChecklist_E1_DoubleSignSlashFreezesOperator`
- E2: `TestChecklist_E2_UnjailNoOpWhenOperatorFrozen`
- E3/G1: `TestChecklist_E3_MultiDogfoodHooksAfterValidatorRemovedShortCircuits`
- E4: `TestChecklist_E4_DowntimeSlashRemovesImuaValidatorButRetainsSigningInfo`
- E5: `TestChecklist_E5_DoubleSignDogfoodSlashWritesOperatorSlashRecord`
- E6: `TestChecklist_E6_SecondEquivocationSameConsAddrIgnored`
- F1: `TestChecklist_F1_RepeatedDoubleSignSlashDoesNotPanicWhenAlreadyFrozen`
- I1: `TestChecklist_I1_RotationBoundaryUnspecifiedSlashMigratesToCurrentKey`
- I2: `TestChecklist_I2_JailOnOldKeyAppliesToCurrentKeyCoherently`
- J1: `TestChecklist_J1_RandomizedLifecycleInvariant`

## Recommended Verification Commands

```bash
go test ./x/dogfood/keeper/ -run 'TestKeyChangeEscapeTestSuite/TestChecklist_' -count=1
go test ./x/dogfood/keeper/ -run 'TestKeyChangeEscapeTestSuite/TestSameEpochDoubleKeyRotation_SkipsSecondHookRisk' -count=1
go test ./x/dogfood/types/ -run 'TestChecklist_E3_' -count=1
go test ./x/dogfood/keeper/ ./x/operator/keeper/ ./x/oracle/keeper/... ./x/dogfood/types/ -count=1
```

## Reviewer Notes

- MsgUnjail success at x/slashing does not imply operator unjailed when frozen; authoritative state is x/operator jail+freeze state.
- `alreadyRecorded` behavior is intentional for previous-key bookkeeping; same-context double replacement differs from normal msg+commit epoch flow.
