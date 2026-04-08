## Validator Jailing & Slashing Fix Review

### Scope

**Commits in review scope (validator jailing/slashing corner cases)**  
- `d8dd3f60` – **fix: correctly jail and/or tombstone** (core behavior change)  
- `af55e433` – **fix: call AfterValidatorRemoved hook**  
- `48ff8054` – **fix: copy missed block array**  
- `4f4a2fa6` – **fix: exclude frozen operators**  
- `bc84b46c` – **fix(operator): block frozen operators power**  
- `5edaed39` – **fix: copy signing info correctly**  
- `81185b27` – **fix: reset signing info on maturity**  

Most work is in `x/dogfood/keeper/{impl_sdk.go,impl_operator_hooks.go,abci.go}`, `x/operator/keeper/{operator.go,abci.go}`, plus tests in `x/dogfood/keeper/key_change_escape_test.go`.

### High-level behavior changes

- **Double-signing now permanently freezes the operator and propagates tombstone to all future keys**
  - `Keeper.SlashWithInfractionReason` in `impl_sdk.go`:
    - On `INFRACTION_DOUBLE_SIGN`, calls `operatorKeeper.FreezeOperator` for the operator.
    - If the infraction key is not the current consensus key, it **copies signing info** from the old to the current key via `CopyValidatorSigningInfo`, ensuring the new key is also tombstoned and jailed forever.
- **Key changes now preserve slashing / downtime history across keys**
  - `OperatorHooksWrapper.AfterOperatorKeyReplaced`:
    - Calls `afterValidatorCreated` for the new key (creates a blank signing info in x/slashing).
    - Schedules pruning of the old consensus address at unbonding completion.
    - Calls `CopyValidatorSigningInfo(oldConsAddr, newConsAddr, INFRACTION_UNSPECIFIED)` to transplant missed-block state and other signing info from old to new key.
  - `CopyValidatorSigningInfo`:
    - For `INFRACTION_DOWNTIME`, copies metadata, jails the new key for downtime, resets missed-block counters and bit array for the new key.
    - For `INFRACTION_UNSPECIFIED` (plain key replacement), clears the **new key’s** missed-block bit array and then replays `MissedBlocksCounter` bits as missed; resets `IndexOffset` to 0 to make x/slashing re-count from scratch.
- **Frozen operators are globally excluded from active participation**
  - `impl_sdk.go::Unjail`: looks up the operator by cons addr and **refuses to unjail** if `IsOperatorFrozen` is true.
  - `operator/keeper/abci.go::UpdateVotingPower`:
    - When building `votingPowerSet`, only sets `ActiveUSDValue` and contributes to AVS voting power if:
      - `SelfUSDValue >= minimumSelfDelegation`, and  
      - `!IsOperatorFrozenStr(ctx, operator)`.
  - `operator/keeper/operator.go::IsOptedInAndNotJailed` includes `!IsOperatorFrozen(...)` in the predicate.
- **Consensus address cleanup now also clears signing info (unless tombstoned)**
  - `dogfood/keeper/abci.go::EndBlock`:
    - For each consensus address scheduled for pruning:
      - Calls `Hooks().AfterValidatorRemoved(...)` (x/slashing hook).
      - Deletes operator<->consensus mapping in x/operator.
      - Clears missed-block bit array.
      - Calls `ResetValidatorSigningInfo(consAddr)` to reset signing info, but with comment noting that tombstoned keys are effectively unaffected, preserving tombstone.
- **Reset on maturity / unbonding completion**
  - `81185b27` adjusts `abci.go` + hooks to **reset signing info when the key is fully out of the unbonding window**, avoiding stale mixed state (non-zero counters vs empty bit array).

### Key changes at suspicious / high‑risk places

#### 1. `ValidatorByConsAddr` status semantics

- In `impl_sdk.go`, `ValidatorByConsAddr` returns a validator from `operatorKeeper.ValidatorByConsAddrForChainID` and **intentionally leaves `Status` as unspecified** instead of trying to mark it unbonded/unbonding.
- Comments explain this is required so that:
  - Evidence / slashing modules accept the validator for equivocation handling (they reject UNBONDED), and
  - They rely only on jailed status and presence of the record.
- **Risk to verify**:  
  - Ensure there is no downstream logic (other modules, invariants, or ABCI) that assumes the staking status here is accurate beyond “not UNBONDED” and jailed flag.  
  - Check any invariants/tests that iterate validators via the staking interface.

#### 2. `CopyValidatorSigningInfo` – migration of missed-block and tombstone state

Summary: see **Deep dive: `CopyValidatorSigningInfo`** below for call sites, SDK ordering, comment meaning, and risks. At a high level, behavior depends on `infraction` (double-sign vs downtime vs unspecified migration), and **`UNSPECIFIED` is overloaded** (key-rotation hook vs other slash paths such as oracle).

#### 3. Key‑replacement hook ordering and pruning

- `AfterOperatorKeyReplaced` (dogfood hooks) does three things in order:
  1. `afterValidatorCreated` for the new key (registers in x/slashing, gives it a blank `ValidatorSigningInfo`).
  2. Schedules pruning of the **old** consensus addr at `GetUnbondingCompletionEpoch`.
  3. Calls `CopyValidatorSigningInfo(oldConsAddr, newConsAddr, INFRACTION_UNSPECIFIED)`.
- `EndBlock` uses the pending consensus addresses to:
  - Call `AfterValidatorRemoved`, delete operator <-> cons mapping, clear bit array, and reset signing info (unless tombstoned).
- **Suspicious points**:
  - The flow assumes unbonding completion ordering: pruning only after the unbonding epoch completes. Any bug in `GetUnbondingCompletionEpoch` or epoch scheduling would either:
    - Prune too early (lose ability to slash for misbehavior during the unbonding window), or
    - Prune too late (harmless but leaks state).
  - `ResetValidatorSigningInfo` being called for non-tombstoned keys is correct, but you want to confirm **tombstoned keys are not accidentally reset**, i.e. that `ResetValidatorSigningInfo` is implemented in the underlying slashing keeper in a tombstone-safe way (as the comment claims).
- **Review recommendation**:
  - Re-check tests in `key_change_escape_test.go` that cover:
    - Double-sign just before/after key changes.
    - Downtime accumulated across multiple epochs and key rotations.
    - Pruning behavior after unbonding completion.
  - **`alreadyRecorded` / multiple rotations in one epoch:** see **§4** (hooks may skip the second replacement on the same context; normal `Msg` + `Commit` flow often clears state in dogfood `EndBlock`).

#### 4. Same-epoch double key rotation (`alreadyRecorded`) and `CopyValidatorSigningInfo`

**Operator keeper** (`x/operator/keeper/consensus_keys.go`): the first consensus-key replacement in a dogfood epoch records a “previous” consensus key for Comet vote-power bookkeeping (including power‑0 updates). A flag (`alreadyRecorded` in the flow around `AfterOperatorKeyReplaced`) means **only the first replacement in that window** triggers **`AfterOperatorKeyReplaced`**. A **second** replacement **without** clearing that state **skips** the hook, so **x/slashing never runs `afterValidatorCreated` / `CopyValidatorSigningInfo` for the final key**. The forward map still updates to the latest key, but **`GetValidatorSigningInfo(finalConsAddr)` can be false**, which would be dangerous if that address were treated as an active signer.

**Why two `Msg` rotations in one epoch usually still run hooks twice:** each `Commit()` advances the app; dogfood **`EndBlock`** (when `ShouldUpdateValidatorSet` is true) runs **`ClearPreviousConsensusKeys`**, which clears the per-epoch “previous key” bookkeeping. The **second** `Msg` `SetConsKey` then often sees **`alreadyRecorded == false`** again, so hooks run a second time and **`CopyValidatorSigningInfo`** applies to the newest key. The risky shape is **two replacements on the same SDK context before that `EndBlock` path runs** (for example two direct **`OperatorKeeper.SetOperatorConsKeyForChainID`** calls with no intervening commit).

**Is `alreadyRecorded` “correct”?** For the **stated** operator-layer goal (one “previous key” slot per epoch for Comet updates), **yes**. It is **not** the same rule as **“every rotation must register the new key in x/slashing.”** If product policy is **at most one replacement per dogfood epoch** (or always wait for an epoch boundary / block that clears previous keys), the current coupling is consistent; if policy is **unbounded rotations per epoch with full slashing hook parity**, this path would need a design change (for example not gating the hook on `alreadyRecorded`, or resetting it in a way that preserves Comet invariants).

**Test:** `TestSameEpochDoubleKeyRotation_SkipsSecondHookRisk` in `key_change_escape_test.go` reproduces the skip by calling **`SetOperatorConsKeyForChainID` twice on the same context** and asserts **no `ValidatorSigningInfo` for the final consensus address**. Compare with **`TestChecklist_D3_DoubleKeyRotationInvariants`**, which uses an **epoch between** rotations so hooks run twice.

**Residual resolved:** **`SetOperatorConsKeyForChainID`** godoc in **`consensus_keys.go`** points integrators at **`alreadyRecorded`**, **`ClearPreviousConsensusKeys`**, and the two tests above.

#### 5. Frozen operator behavior and voting power

- `IsOptedInAndNotJailed` now also checks `!IsOperatorFrozen`.
  - This is used when determining active operators for vote power / AVS participation.
- `UpdateVotingPower` in `operator/keeper/abci.go`:
  - While recalculating voting power:
    - Filters operators that have opted out but not yet effective (current epoch).
    - Sets `ActiveUSDValue` only when self-delegation ≥ minimum AND operator is **not frozen**.
- `impl_sdk.go::Unjail`:
  - If the cons addr belongs to a frozen operator, unjail is silently ignored; logs **`reason=operator_frozen`** plus **`operator`** and **`cons_addr`** for indexers.
- **Suspicious points**:
  - There is an intentional divergence between **“can unjail”** and **“is active”**:
    - Frozen operators remain jailed and are invisible in active power.
  - Need to ensure there is **no API** (CLI / msg server) that allows editing operator metadata or other state in a way that conflicts with being frozen (some methods already check `IsOperatorFrozen`, others should be checked).
- **Review recommendation**:
  - Confirm all relevant operator‑mutating paths (commission changes, opt‑in/out, etc.) reject frozen operators.
  - Ensure monitoring/UX clarifies that unjail messages targeting a frozen operator will appear as no-ops (only log).

#### 6. `AfterValidatorRemoved` hook usage

- `af55e433` and `abci.go::EndBlock`:
  - For each consensus addr scheduled to prune, `AfterValidatorRemoved` is called in a **cache context** and only committed if no error occurs.
  - Then the local mappings and slashing data (missed block array + signing info) are cleared in the same cached context.
- **Suspicious points**:
  - The comment states x/slashing never returns an error from this hook; if this changes upstream, the code will now reschedule the consensus address for the next epoch when an error happens, which is probably fine but should be validated.
  - Any panic inside x/slashing’s `AfterValidatorRemoved` would now happen within dogfood’s `EndBlock`.
- **Review recommendation**:
  - Confirm the rescheduling logic (`AppendConsensusAddrToPrune`) for failures is correct and idempotent.

#### 7. Missed-block bit array copying fix

- `48ff8054` ensures the **missed block bit array** is also copied when transferring signing info between keys.
- Previously, only counters might have been copied, leading to inconsistent state (non-zero counters vs empty array).
- **Suspicious point**:
  - The current implementation sets bits for `[0, MissedBlocksCounter)` to “missed”, which compresses arbitrary missed-block distributions into a prefix. This is a deliberate trade‑off and is documented in comments, but it’s worth being comfortable that:
    - It doesn’t under‑account missed blocks in a way that lets operators evade downtime slashing.
    - It only over‑approximates missed time, which is conservative/punishing.

#### 8. Reset on “maturity” of unbonding / key lifetime

- `81185b27` adds logic in `abci.go` and hooks to:
  - Reset signing info when a key is “matured” (out of unbonding window / pruned).
  - Add tests in `key_change_escape_test.go` to cover lifecycle around maturity.
- **Suspicious points**:
  - The concept of “maturity” depends on AVS/epoch config. If this is mis‑aligned with slashing unbonding durations, there could be a window where evidence is still allowed but the signing info was reset.
- **Review recommendation**:
  - Cross-check unbonding durations in x/slashing vs the AVS/epoch unbonding model used here.
  - Ensure `GetUnbondingCompletionEpoch` and associated scheduling uses the **same effective unbonding window** as slashing evidence’s max age.

### Deep dive: `CopyValidatorSigningInfo`

Implementation: `x/dogfood/keeper/impl_sdk.go` (`CopyValidatorSigningInfo`, and the `slashingOldKey` branch in `SlashWithInfractionReason`).

#### Call sites (production)

There are exactly **two** ways this function is invoked in-repo:

1. **`dogfoodkeeper.SlashWithInfractionReason`** — when `slashingOldKey` is true: the infraction consensus address `addr` differs from the operator’s **current** key `currentConsAddr` (reverse lookup found the operator, forward lookup returned a newer key). The `infraction` argument is whatever the caller passed through (typically **`INFRACTION_DOWNTIME`** or **`INFRACTION_DOUBLE_SIGN`** from x/slashing / x/evidence; see oracle caveat below).

2. **`OperatorHooksWrapper.AfterOperatorKeyReplaced`** — after consensus key rotation on the dogfood chain. Always **`INFRACTION_UNSPECIFIED`**. Here `UNSPECIFIED` means “migrate signing state,” not “unknown misbehavior.”

#### Behavior by `infraction`

**Common prefix**

- Load `ValidatorSigningInfo` for `oldConsAddr`. If missing, initialize a **default** `ValidatorSigningInfo` for `newConsAddr` (current height, zero misses, etc.).

**`INFRACTION_DOUBLE_SIGN`**

- Sets `Tombstoned = true` and `JailedUntil = DoubleSignJailEndTime` on the **new** key, then `SetValidatorSigningInfo` for `newConsAddr`.
- **Why manual copy is needed:** x/evidence calls `slashingKeeper.SlashWithInfractionReason` (which calls staking/dogfood), then **`Jail` / `JailUntil` / `Tombstone` on the evidence consensus address** only (`x/evidence/keeper/infraction.go`). x/slashing’s `SlashWithInfractionReason` does **not** tombstone the key by itself; tombstoning happens **after** the staking slash returns. So the **new** key would not inherit tombstone without `CopyValidatorSigningInfo`. When `Copy` reads the **old** key, it is still **before** that later `Tombstone(infractionAddr)` for this event — ordering is consistent.

**`INFRACTION_DOWNTIME`**

- Sets `JailedUntil` on the new key to `now + DowntimeJailDuration`.
- Resets `MissedBlocksCounter` and `IndexOffset` to `0` and clears the missed-block bit array for **`newConsAddr`**.
- **SDK alignment:** In `x/slashing/keeper/infractions.go` `HandleValidatorSignature`, the sequence is: call **`sk.SlashWithInfractionReason`** (dogfood runs, including `Copy` when `slashingOldKey`), then **`Jail`**, then reset counter/offset and clear bit array on the **slashed** `consAddr`. So when `Copy` reads signing info for the **old** (infraction) key, it still sees **pre-reset** downtime accumulation; the new key gets the same post-slash shape slashing applies to the punished address (jail + clean counter/array on the key that must carry state forward).

**`INFRACTION_UNSPECIFIED` (key rotation hook path)**

- Clears the **new** key’s missed-block bit array.
- For `i in [0, MissedBlocksCounter)`, sets `SetValidatorMissedBlockBitArray(newConsAddr, i, true)`.
- Sets `IndexOffset = 0` on the new key; **`MissedBlocksCounter` is kept** from the copied `info` (not zeroed in this branch).
- In the Cosmos SDK, `MissedBlocksCounter` is maintained as the **sum of the bit array** (`HandleValidatorSignature` comments in `infractions.go`). Packing misses into indices `0 .. N-1` is a **lossy** approximation of the **spatial** pattern in the sliding window, but preserves **count** `N` and avoids scanning the full window (O(`MissedBlocksCounter`) writes vs O(window)).

#### What the in-function comments mean

- **Key A → B → A over a long time:** After `AfterValidatorCreated`, the new key may have a **stale** bit array. Clear the new array, then reapply the **old** key’s miss **count** so history is not mixed incorrectly.
- **“Reduce loop from O(window) to O(MissedBlocksCounter)”:** Deliberate trade-off: do not copy bit-by-bit across the full signed-blocks window; replay `N` misses at the front of the logical array.
- **“Two ways to game this currently”:** Partly documents **hypothetical** inconsistencies (e.g. new key still carrying `N` misses after a scenario where the old key’s counter was reset). The **`DOWNTIME`** branch of `Copy` **zeros** `MissedBlocksCounter` on the **new** key, which addresses the **downtime-slash + different current key** case; the **`UNSPECIFIED`** rotation branch does **not** zero the counter. The paragraph mixes rotation (`UNSPECIFIED`) and post-downtime copy (`DOWNTIME`) — read it as **design notes + known limitations**, not a tight formal proof.

#### Risk register (subtle / dangerous cases)

1. **`UNSPECIFIED` is overloaded**  
   Oracle malicious slashing uses **`SlashWithInfractionReason` with `INFRACTION_UNSPECIFIED`** (`x/oracle/keeper/feedermanagement/feedermanager.go`, `handleMalicious`). If **`slashingOldKey`** were ever true on that path, `Copy` would take the **`UNSPECIFIED`** branch (bit transplant + inherited `JailedUntil` / fields from the old info at read time), **not** the dedicated **`DOWNTIME`** or **`DOUBLE_SIGN`** shaping. Oracle then **`Jail` / `JailUntil` on the infraction `consAddr`** only. Worth tracing whether the oracle validator identity can diverge from dogfood “current” key the same way as Tendermint evidence.

2. **Packed misses vs true window geometry**  
   Rotation loses **where** misses sit in the window; usually this **concentrates** misses at low indices (often **more conservative** for hitting the threshold again). Under-accounting would be the worrying direction; the team assumes placement “does not matter as much” for their security model.

3. **Old signing info not found (`GetValidatorSigningInfo(oldConsAddr)` → `found == false`)** — **tied to consensus-key lifecycle and prune**  
   Consensus keys are **not permanent**: after replacement, the **old** address is scheduled for removal; dogfood **EndBlock** eventually runs **`AfterValidatorRemoved`**, clears bit arrays, and **`ResetValidatorSigningInfo`** on that address (tombstone aside). Once that happens, **`found == false`** for the old cons addr is **expected**. The security question is **ordering**: any code path that still needs **old** signing history (especially **`CopyValidatorSigningInfo` with `UNSPECIFIED` at rotation**) must run **before** that teardown; **late evidence or slash** after reverse lookup + signing info are gone becomes a **no-op or empty copy**, which may be intended post-maturity or a bug if it happens too early.  
   The function still runs the **`switch infraction`** (it is not skipped): defaults are a fresh `ValidatorSigningInfo` on **`newConsAddr`** (zero `MissedBlocksCounter`, etc.).
   - **`INFRACTION_DOUBLE_SIGN` / `INFRACTION_DOWNTIME`:** The infraction branch still applies tombstone/jail/downtime cleanup on the **new** key, so missing **old** history does not skip core punishment for those paths.
   - **`INFRACTION_UNSPECIFIED` (key rotation):** With `MissedBlocksCounter == 0`, the replay loop does **nothing** → **no transplanted liveness debt**. That is the **main worry** if the old key **should** still have had signing info at rotation time (hook ordering bug); after **prune**, `!found` is often **normal**, not a rotation bug. **`!found` can also be legitimate** when the old cons addr never gained x/slashing state (e.g. **opt-into-AVS then replace key** in the same epoch — see `impl_epochs_hooks_test.go` / `CopyValidatorSigningInfo` comment in `impl_sdk.go`). **Decision (resolved):** do **not** panic on `UNSPECIFIED` + `!found`; rely on **`TestChecklist_A3_B4_...`** for the active-key case and **`D5`** for post-prune behavior.

4. **`StartHeight` and other copied fields**  
   Rotation copies `StartHeight` from the old info, so downtime **minimum height** logic stays tied to the operator’s existing slashing clock rather than “new key = fresh clock.” Usually intentional; flag if product expectation differs.

5. **`GetOperatorAddressForChainIDAndConsAddr` returns `found == false` in `SlashWithInfractionReason`**  
   Slash becomes a no-op (`0` coins); see earlier review — mapping pruned or unknown key. Separate from `Copy` but same flow.

### Deep dive: other core functions (slash orchestration, prune teardown, jail / freeze)

These sit alongside **`CopyValidatorSigningInfo`**: they mutate **x/slashing**, **x/operator** mappings, or **operator opted-in state**, and interact with **consensus address lifecycle** (active key → previous key bookkeeping → scheduled prune).

#### Finding: Slash success coupling (fixed)

- **Finding:** In dogfood `SlashWithInfractionReason`, `CopyValidatorSigningInfo` could run on `slashingOldKey` even when operator economic slash did not commit (or was not truly entered), creating cross-module state drift.
- **Impact:** Explicit bad outcomes included migrated jail/tombstone/liveness debt on the new key without committed operator slash effects, confusing replay paths, and contradictory operator/indexer views.
- **Fix:** Added `operatorKeeper.SlashDogfoodInfraction(...)(attempted, err)`, gated `Copy` on `attempted && err == nil`, and kept double-sign freeze safety (`power > 0` and slash entered/failed).
- **Why safe:** We avoid partial migration side effects while preserving the consensus-safety guarantee that true double-sign faults still freeze operators.

#### 1. `dogfoodkeeper.SlashWithInfractionReason` (`impl_sdk.go`)

**Callers:** x/slashing (`HandleValidatorSignature`, slash paths), x/evidence (equivocation → slashing), and modules that use **staking** `SlashWithInfractionReason` with dogfood as **`StakingKeeper`** (e.g. oracle via embedded dogfood keeper).

**Flow (high level)**

1. **`GetOperatorAddressForChainIDAndConsAddr(chainID, addr)`** — if **`!found`**, return **0** (no economic slash, no `Copy`, no freeze). This is the **post-prune / unknown key** path aligned with checklist **D5**.
2. **`GetOperatorConsKeyForChainID(operator)`** — derive **current** consensus key; set **`slashingOldKey`** when **`currentConsAddr != addr`** (infraction is on a **stale** cons addr still in reverse map).
3. **`operatorKeeper.SlashDogfoodInfraction`** — AVS slash (`slash.go`); returns **`(attempted, err)`**. **`SlashWithInfractionReason`** on the operator keeper is a thin wrapper that logs and returns zero burn on failure (unchanged for other callers).
4. If **`slashingOldKey` && `attempted && err == nil`**: **`CopyValidatorSigningInfo(addr, currentConsAddr, infraction)`** — avoids writing x/slashing migration when the operator slash **aborted** (veto / **`SlashAssets`** error), which would desync signing info from committed operator state.
5. If **`INFRACTION_DOUBLE_SIGN`** and **`power > 0`** and **`(attempted || err != nil)`**: **`FreezeOperator(operatorAcc)`** — consensus fault: freeze even when the economic slash **failed**, so veto cannot leave the operator unfrozen; log when **`err != nil`**. If the slash path was **never entered** (`!attempted` and no error, e.g. invalid setup), **do not** freeze.

**Consensus-key lifecycle**

- **`Copy`** only runs when the **infraction address** still **reverse-resolves** to an operator and differs from the **forward** current key — i.e. **after rotation**, **before** reverse mapping for the old addr is pruned.
- After full prune, step 1 short-circuits; you do **not** reach **`Copy`** with a pruned addr.

**Invariant tests**

- **`TestChecklist_E1_DoubleSignSlashFreezesOperator`** — double-sign through dogfood sets **`IsOperatorFrozen`**.

#### 2. `dogfoodkeeper.EndBlock` — `ClearPreviousConsensusKeys` + consensus-addr prune queue (`abci.go`)

**Order (when `ShouldUpdateValidatorSet`)**

1. **`operatorKeeper.ClearPreviousConsensusKeys`** — clears **per-epoch “previous consensus key”** bookkeeping used for Comet 0-power updates; interacts with **`alreadyRecorded`** / second rotation in same context (see **§4**).
2. **Pending opt-outs** — `CompleteOperatorKeyRemovalForChainID` / reschedule on failure.
3. **Pending consensus addresses to prune** (from **`AppendConsensusAddrToPrune`** at rotation / hook time): for each **`consAddr`**:
   - **`CacheContext`**
   - **`Hooks().AfterValidatorRemoved(cc, consAddr, nil)`** — x/slashing removes cons-pubkey lookup, etc.
   - **`DeleteOperatorAddressForChainIDAndConsAddr`** — drops **reverse** operator mapping for that chain + addr.
   - **`ClearValidatorMissedBlockBitArray`**
   - **`ResetValidatorSigningInfo`** — deletes signing info if **not** tombstoned (tombstone retained).
   - **`writeFunc()`** — commit cache; on hook error, addr is **rescheduled** to next epoch.

**Why it matters**

- This is the **symmetric teardown** to **`CopyValidatorSigningInfo` + hook registration**: it removes the **old** addr from slashable / mappable state after the **unbonding epoch** scheduled at rotation.
- **Risk:** ordering or partial failure → stale mapping, or signing info / bit-array **desync** (comments justify **`ResetValidatorSigningInfo`** for non-tombstoned keys).
- **Hook error path (checklist E3):** On **`AfterValidatorRemoved` error**, the iteration **does not** call **`writeFunc`** — operator reverse map and x/slashing state for that addr are **unchanged**; the addr is **re-queued** for **`nextEpochNumber`**. With current **`app.go`** wiring, x/slashing’s hook **never** errors; **`MultiDogfoodHooks`** short-circuit behavior is covered by **`TestChecklist_E3_MultiDogfoodHooksAfterValidatorRemovedShortCircuits`**.

**Invariant tests**

- **`TestChecklist_D5_PostPruneReverseLookupAndSlashNoOp`** (and **B0–B4** narrative).
- **E3:** **`TestChecklist_E3_MultiDogfoodHooksAfterValidatorRemovedShortCircuits`** (`x/dogfood/types/hooks_test.go`).

#### 3. `operatorKeeper.Jail` / `Unjail` / `SetJailedState` (`slash.go`)

**Behavior:** Resolve **`(consAddr, chainID) → operator`**, then update **`OptedInfo.Jailed`** and **`JailToggleHeights`**, and call **`hooks.AfterJail`** with impacted AVS list.

**Dogfood bridge:** **`dogfoodkeeper.Jail` / `Unjail`** delegate to these using **`chainIDWithoutRevision(ctx.ChainID())`**.

**Lifecycle:** Jailed flag is **per AVS opt-in**, not the same store as x/slashing **`ValidatorSigningInfo.JailedUntil`**, but both must be consistent enough for **MsgUnjail** (slashing checks signing info + calls **`sk.Unjail(consAddr)`**).

#### 4. `dogfoodkeeper.Unjail` — frozen guard (`impl_sdk.go`)

**Behavior:** **`GetOperatorAddressForChainIDAndConsAddr`**; if operator **`IsOperatorFrozen`**, **return without** **`operatorKeeper.Unjail`** (log only). **StakingKeeper.Unjail** has **no error return**, so x/slashing **MsgUnjail** can still **return success** while the operator **stays jailed** in x/operator.

**Invariant tests**

- **`TestChecklist_E2_UnjailNoOpWhenOperatorFrozen`**.

#### 5. `operatorKeeper.FreezeOperator` (`freeze.go`)

**Behavior:** Sets a **permanent** KV flag **`KeyForOperatorFrozen`**; **`IsOperatorFrozen`** drives **dogfood `Unjail` no-op**, **`UpdateVotingPower`** (no active power), **`IsOptedInAndNotJailed`**, and **msg/keeper guards** (opt-in, key change, etc.). Second **`FreezeOperator`** returns **`ErrOperatorFrozenStateMismatch`** (dogfood double-sign path **panics** on freeze error).

**Replay / second double-sign (checklist E6):** x/evidence **`HandleEquivocationEvidence`** returns **before** **`SlashWithInfractionReason`** when the **evidence consensus address** is already **tombstoned**, so a **second** equivocation on that addr **does not** invoke dogfood slash or **`FreezeOperator`** again. **Invariant test:** **`TestChecklist_E6_SecondEquivocationSameConsAddrIgnored`**.

#### 6. `dogfoodkeeper.ApplyValidatorChanges` (`validators.go`) — Comet updates vs slashing teardown

When **power goes to 0**, **`DeleteImuachainValidator`** removes the **dogfood-local** validator record but **does not** call x/slashing removal immediately — comments state **slashing** lookup must stay until **unbonding** completes; actual teardown is **`EndBlock`** prune above.

**Grep / usage:** **`GetImuachainValidator`** appears in **`impl_operator_hooks.go`** (jail → validator-set update flag) and tests/queries — **no** branch treats “absent from Imuachain store” as “no **`ValidatorSigningInfo`**.” **Invariant test (checklist E4):** **`TestChecklist_E4_DowntimeSlashRemovesImuaValidatorButRetainsSigningInfo`**.

**Operator slash record (checklist E5):** **`SlashDogfoodInfraction`** / **`Slash`** persists **`OperatorSlashInfo`** keyed by **`GetSlashIDForDogfood(infraction, infractionHeight)`**. **Invariant test:** **`TestChecklist_E5_DoubleSignDogfoodSlashWritesOperatorSlashRecord`**. **Economic slash failure vs freeze:** resolved in **`impl_sdk.go`** ( **`Copy`** only after successful slash; **double-sign freeze** even if slash errors).

### Next steps checklist (trace-driven)

**Primary concern (signing info × lifecycle):** A replaced consensus key is eventually **pruned**: operator reverse mapping, x/slashing **`AfterValidatorRemoved`**, missed-bit array, and **`ResetValidatorSigningInfo`** can all drop state for the **old** address. The main review thread is whether **every path that reads `oldConsAddr`** (`CopyValidatorSigningInfo`, slash attribution, evidence) still sees consistent state **at the moment it runs**, and whether **B2–B4** line up with evidence max age and rotation timing. Checklist **A6** is the hook-specific slice of that story; **section B** is the **lifecycle backbone**—work B **first** if you want one ordered deep dive.

Use this as a concrete follow-up list; items below are updated after a trace + test pass (see **Checklist trace results**).

#### A. `CopyValidatorSigningInfo` and callers

- [x] **A1 — Downtime ordering:** In `x/slashing/keeper/infractions.go` `HandleValidatorSignature`, `sk.SlashWithInfractionReason` (→ `dogfoodkeeper.SlashWithInfractionReason`, including `Copy` when `slashingOldKey`) runs **before** `Jail` and **before** the reset of `MissedBlocksCounter` / `IndexOffset` and `ClearValidatorMissedBlockBitArray` on the **infraction** `consAddr`. So `Copy` reads **pre-reset** signing info on the old key; the `DOWNTIME` branch then applies jail + zeroed counter/array on the **new** key. **Verified by code trace** (Cosmos SDK / imuachain fork).

- [x] **A2 — Double-sign ordering:** In `x/evidence/keeper/infraction.go`, order is `slashingKeeper.SlashWithInfractionReason` (→ dogfood, including `Copy` when `slashingOldKey`) **then** `Jail`, `JailUntil`, **`Tombstone(consAddr)`** on the **evidence** consensus address. Tombstone on the infraction key happens **after** staking returns; `Copy` explicitly sets tombstone/jail on the **current** key. **Verified by code trace.**

- [x] **A3 — Key rotation:** `AfterOperatorKeyReplaced` runs `afterValidatorCreated` (new key in x/slashing), `AppendConsensusAddrToPrune` for **old** addr at `GetUnbondingCompletionEpoch`, then `CopyValidatorSigningInfo(..., UNSPECIFIED)`. **Invariant test added:** `TestChecklist_A3_B4_KeyRotationSigningInfoInvariant` in `key_change_escape_test.go` (run as `TestKeyChangeEscapeTestSuite/TestChecklist_A3_B4_KeyRotationSigningInfoInvariant`).

- [x] **A4 — Oracle × `UNSPECIFIED`:** **Resolved:** Malicious slash uses **current** oracle validator identities: quoting uses **`f.cs.GetValidators()`** (active-set cache); phase-2 bad raw data uses **`ConsAddrStrFromCreator(msg.Creator)`**. Comment in **`feedermanager.go`** documents alignment with dogfood; if **`consAddr` ≠ forward key**, **`slashingOldKey` + `Copy(..., UNSPECIFIED)`** matches Tendermint-style stale-key handling (not an oracle-specific gap).

- [x] **A5 — Paths with `UNSPECIFIED` to dogfood:** (1) `dogfoodkeeper.Slash` → `SlashWithInfractionReason(..., UNSPECIFIED)` (`impl_sdk.go`). (2) Oracle `handleMalicious` → `SlashWithInfractionReason(..., UNSPECIFIED)` (`feedermanager.go`). (3) x/slashing `Keeper.Slash` → `SlashWithInfractionReason(..., UNSPECIFIED)` on **staking** (`sk`); in production that is **dogfood** for this chain. **Downtime / double-sign** paths use explicit `DOWNTIME` / `DOUBLE_SIGN`. **Ongoing:** new modules calling **`sk.Slash`** should be reviewed in PR (same as any new staking hook consumer).

- [x] **A6 — Missing signing info on `oldConsAddr`:** **Resolved:** **No panic** on **`UNSPECIFIED` + `!found`** — legitimate for **opt-in then replace same epoch** (see `impl_sdk.go` comment on **`CopyValidatorSigningInfo`**). **Rotation with prior activity:** **`TestChecklist_A3_B4_...`** asserts **`found == true`** for old addr; **post-prune** **`!found`** is expected (**B3**).

#### B. Consensus-key lifecycle, mappings, pruning, evidence windows *(primary deep dive for “old key removed”)*

- [x] **B0 — Lifecycle diagram:** **Narrative (code-aligned):** `AfterValidatorCreated` / opt-in sets up signing info for a key → validator active → **`AfterOperatorKeyReplaced`** schedules **`AppendConsensusAddrToPrune(epoch, oldConsAddr)`** with `epoch = GetUnbondingCompletionEpoch` (`currentEpoch + EpochsUntilUnbonded`, `unbonding.go`) → on that epoch’s dogfood **`EndBlock`**, pending consensus addrs: **`AfterValidatorRemoved`** → **`DeleteOperatorAddressForChainIDAndConsAddr`** → **`ClearValidatorMissedBlockBitArray`** → **`ResetValidatorSigningInfo`** (`abci.go`). **Optional:** promote to a diagram in a design doc.

- [x] **B1 — Reverse vs forward map:** Forward: `(operator, chainID) → current cons key`. Reverse: `(chainID, consAddr) → operator` kept until **prune** (comments in `consensus_keys.go`, `impl_operator_hooks.go`). After rotation, **old** addr still reverses to operator until **EndBlock** prune completes.

- [x] **B2 — Pruning epoch vs evidence max age:** **`TestChecklist_B2_EvidenceWindowCoversDogfoodUnbonding`** compares consensus evidence limits to `EpochsUntilUnbonded` × dogfood epoch wall duration and × `testutil.TestBlockNumberPerEpoch` blocks. Default **`app/test_helpers.go`** evidence (`MaxAgeNumBlocks` 302400, `MaxAgeDuration` 504h) dwarfs test unbonding (e.g. 5 × 1 minute). **Resolved in-tree:** comment on **`DefaultConsensusParams.Evidence`** references **B2** and the need to co-change dogfood/epoch params if evidence limits are tightened on a **production** chain.

- [x] **B3 — `EndBlock` prune path:** **`ResetValidatorSigningInfo`** in imuachain Cosmos fork: if **tombstoned**, returns **without** deleting (tombstone retained); else **deletes** signing info key (`signing_info.go`). Matches comments in `abci.go`. **Post-prune, `GetValidatorSigningInfo(old)` → `found == false` is normal.**

- [x] **B4 — Ordering invariants:** (1) **Rotation + `Copy` run in the same tx as `SetConsKey`**, while prune runs at **scheduled epoch EndBlock** → **later**; **no** same-block teardown before hook. (2) **`Copy` is only invoked from** `AfterOperatorKeyReplaced` (old = replaced key) and `SlashWithInfractionReason` (old = infraction addr); neither uses an addr that has already gone through **this** prune path in the same step. (3) **After prune:** `GetOperatorAddressForChainIDAndConsAddr` false and/or no signing info → **`SlashWithInfractionReason` early-return** (`found == false`); **acceptable** if **B2** guarantees evidence cannot arrive that late. **(1) reinforced by** `TestChecklist_A3_B4_KeyRotationSigningInfoInvariant` (old addr still has signing info right after rotation).

#### C. Frozen operator and voting power

- [x] **C1 — Msg surface:** Sensitive paths go through keepers that check **`IsOperatorFrozen`**: `OptIn` / `OptOut` (`opt.go`), **`setOperatorConsKeyForChainID`** (`consensus_keys.go`), **`EditOperator` / `UpdateRewardCompoundingFlag`** (`operator.go`), **`ValidateAndUpdateCommissionRate`** (`commission.go`). **MsgServer** delegates to these. **Process:** new **`Msg` handlers** that mutate operator/dogfood state should go through these keepers or add an explicit **`IsOperatorFrozen`** check in code review.

- [x] **C2 — Unjail:** **`dogfoodkeeper.Unjail`** logs and returns early if frozen (`impl_sdk.go`) with structured fields **`reason=operator_frozen`**, **`operator`**, **`cons_addr`**, and notes that **MsgUnjail** may still succeed at x/slashing. Msg layer cannot return error without SDK interface change.

#### D. Tests and invariants

- [x] **D1 — Map tests to checklist:** `key_change_escape_test.go` covers **A1/A2-style** scenarios (`TestKeyChangeEscape`, `TestDowntimeDriftKeyReplacement`, tombstone tests) plus **`TestChecklist_*`**: **A3/B4/D2/D4(a)**, **B2**, **D3**, **D5**, **E1–E6**, and **`TestSameEpochDoubleKeyRotation_SkipsSecondHookRisk`** (**§4**). **E3** hook ordering lives in **`x/dogfood/types/hooks_test.go`**.

- [x] **D2 — Invariant:** **Covered** for **UNSPECIFIED** rotation path on **new** key: `MissedBlocksCounter == sumMissedBitsInWindow(newConsAddr)` in `TestChecklist_A3_B4_KeyRotationSigningInfoInvariant`. **DOWNTIME `Copy`:** **`TestDowntimeDriftKeyReplacement`** asserts non-negative / zero counter after slash + rotation path — accepted as sufficient unless a stricter bit-sum assertion is needed later.

- [x] **D3 — Stress (light):** **`TestChecklist_D3_DoubleKeyRotationInvariants`** — two `SetConsKey` rotations with an **epoch boundary between** them (required so `AfterOperatorKeyReplaced` runs again; otherwise `alreadyRecorded` skips hooks — see `setOperatorConsKeyForChainID`). Asserts forward key and **counter == bit-array sum** on the latest key. **`TestSameEpochDoubleKeyRotation_SkipsSecondHookRisk`** — two **`SetOperatorConsKeyForChainID`** calls on the **same context** (no intervening `Commit` / dogfood `EndBlock` that runs `ClearPreviousConsensusKeys`) asserts the **second** rotation **skips** hooks: **no `ValidatorSigningInfo` for the final key** while the forward map still points at it. This does **not** match two `Msg` rotations each followed by `Commit()` in the same epoch, which often **do** run hooks twice because **`EndBlock` clears** previous-key state (see **§4 Same-epoch double key rotation**).

- [x] **D4 — `UNSPECIFIED` + no old signing info:** **(a)** **At rotation:** `TestChecklist_A3_B4_KeyRotationSigningInfoInvariant` asserts **`found == true`** for old cons addr right after `SetConsKey`. **(b) After prune:** `Copy` is not invoked again; **D5** asserts pruned old addr has no reverse lookup / signing info.

- [x] **D5 — Prune vs late slash/evidence:** **`TestChecklist_D5_PostPruneReverseLookupAndSlashNoOp`** — rotate once, advance **`EpochsUntilUnbonded + 1`** epochs (same pattern as tombstone prune tests), then assert **`GetOperatorAddressForChainIDAndConsAddr`** false, **`GetValidatorSigningInfo`** false on old addr, and **`SlashWithInfractionReason`** returns **zero** burn.

#### E. Slash orchestration, `EndBlock` prune, jail / freeze (see **Deep dive: other core functions**)

- [x] **E1 — Double-sign → `FreezeOperator`:** Traced in **`dogfoodkeeper.SlashWithInfractionReason`** (`impl_sdk.go`). **Invariant test:** **`TestChecklist_E1_DoubleSignSlashFreezesOperator`**.

- [x] **E2 — Frozen operator vs `Unjail`:** **`dogfoodkeeper.Unjail`** returns early when **`IsOperatorFrozen`**; x/slashing **MsgUnjail** may still succeed. **Invariant test:** **`TestChecklist_E2_UnjailNoOpWhenOperatorFrozen`**.

- [x] **E3 — `EndBlock` prune hook failure / reschedule:** On hook error, **`writeFunc` is not called** — no partial prune; addr is **`AppendConsensusAddrToPrune(ctx, nextEpochNumber, addr)`**. **Production:** `app.go` wires only **`slashingKeeper.Hooks()`** into dogfood; **`AfterValidatorRemoved`** in x/slashing **always returns `nil`** (`deleteAddrPubkeyRelation` is a single **`store.Delete`**, idempotent on retry). **`MultiDogfoodHooks`** short-circuits on first error — unit test **`TestChecklist_E3_MultiDogfoodHooksAfterValidatorRemovedShortCircuits`** in **`x/dogfood/types/hooks_test.go`**. If more hooks are added later, reschedule behavior remains safe: reverse map and signing info stay until a full successful pass.

- [x] **E4 — `ApplyValidatorChanges` power 0 vs slashing state:** **`DeleteImuachainValidator`** does **not** remove x/slashing signing info. **`GetImuachainValidator`** is used in **`impl_operator_hooks.go`** only to decide whether a **jail** should **`MarkUpdateValidatorSetFlag`** — no “no imua validator ⇒ no signing info” assumption there. **Invariant test:** **`TestChecklist_E4_DowntimeSlashRemovesImuaValidatorButRetainsSigningInfo`**.

- [x] **E5 — Economic slash vs x/slashing / freeze:** **`SlashDogfoodInfraction`** (`slash.go`) implements the AVS slash; dogfood gates **`CopyValidatorSigningInfo`** on **`attempted && err == nil`**, and **double-sign** still **`FreezeOperator`** when **`power > 0`** and the slash was **entered or failed** (see **Deep dive §1**). **Invariant test:** **`TestChecklist_E5_DoubleSignDogfoodSlashWritesOperatorSlashRecord`**.

- [x] **E6 — Evidence / slash replay (already tombstoned):** **`HandleEquivocationEvidence`** returns **before** **`SlashWithInfractionReason`** if **`IsTombstoned(consAddr)`** (x/evidence). Second identical equivocation **does not** call dogfood slash or **`FreezeOperator` again — **no panic**. **Invariant test:** **`TestChecklist_E6_SecondEquivocationSameConsAddrIgnored`**.

### Checklist trace results (summary)

| Item | Status | Notes |
|------|--------|--------|
| A1, A2 | Done | SDK / evidence ordering traced against `dogfoodkeeper.SlashWithInfractionReason` + `CopyValidatorSigningInfo`. |
| A3, D2, D4(a), B4(1) | Done | New test `TestChecklist_A3_B4_KeyRotationSigningInfoInvariant`. |
| A4 | Done | Oracle malicious path documented in **`feedermanager.go`**; identities from active-set cache or tx creator; stale-key handling matches dogfood **`slashingOldKey`**. |
| A5 | Done | Grep + interface read (`KeeperOracle` / `KeeperDogfood`). |
| A6, B0, B1, B3, B4 | Done | Lifecycle + `ResetValidatorSigningInfo` tombstone behavior documented. |
| B2, D3, D5 | Done | `TestChecklist_B2_EvidenceWindowCoversDogfoodUnbonding`, `TestChecklist_D3_DoubleKeyRotationInvariants` (epoch between rotations: `alreadyRecorded` / hooks), `TestSameEpochDoubleKeyRotation_SkipsSecondHookRisk` (same-ctx double `SetOperatorConsKeyForChainID`: second hook skipped), `TestChecklist_D5_PostPruneReverseLookupAndSlashNoOp`. |
| C1, C2 | Done | Frozen checks on keeper paths; unjail no-op documented. |
| E1–E6 | Done | **E1/E2:** freeze + unjail no-op. **E3:** `x/dogfood/types` multi-hook short-circuit; slashing `AfterValidatorRemoved` always `nil`; reschedule + idempotent pubkey delete. **E4:** no imua validator, signing info retained after downtime. **E5:** operator slash record for double-sign. **E6:** second equivocation ignored (tombstone). |

**How to run checklist tests**

```bash
go test ./x/dogfood/keeper/ -run 'TestKeyChangeEscapeTestSuite/TestChecklist_A3_B4_KeyRotationSigningInfoInvariant' -count=1
go test ./x/dogfood/keeper/ -run 'TestKeyChangeEscapeTestSuite/TestChecklist_B2_EvidenceWindowCoversDogfoodUnbonding' -count=1
go test ./x/dogfood/keeper/ -run 'TestKeyChangeEscapeTestSuite/TestChecklist_D5_PostPruneReverseLookupAndSlashNoOp' -count=1
go test ./x/dogfood/keeper/ -run 'TestKeyChangeEscapeTestSuite/TestChecklist_D3_DoubleKeyRotationInvariants' -count=1
go test ./x/dogfood/keeper/ -run 'TestKeyChangeEscapeTestSuite/TestSameEpochDoubleKeyRotation_SkipsSecondHookRisk' -count=1
go test ./x/dogfood/keeper/ -run 'TestKeyChangeEscapeTestSuite/TestChecklist_E1_' -count=1
go test ./x/dogfood/keeper/ -run 'TestKeyChangeEscapeTestSuite/TestChecklist_E2_' -count=1
go test ./x/dogfood/keeper/ -run 'TestKeyChangeEscapeTestSuite/TestChecklist_E4_' -count=1
go test ./x/dogfood/keeper/ -run 'TestKeyChangeEscapeTestSuite/TestChecklist_E5_' -count=1
go test ./x/dogfood/keeper/ -run 'TestKeyChangeEscapeTestSuite/TestChecklist_E6_' -count=1
go test ./x/dogfood/types/ -run 'TestChecklist_E3_' -count=1
# Or all dogfood checklist-style tests in keeper:
go test ./x/dogfood/keeper/ -run 'TestKeyChangeEscapeTestSuite/TestChecklist_' -count=1
```

### Suggested focus areas for deeper review / testing

- **Key‑change escape tests** (`x/dogfood/keeper/key_change_escape_test.go`):
  - Verify coverage for:
    - Repeated rapid key changes with downtime.
    - Double-sign before/after key replacements.
    - Old key jailed/tombstoned state correctly “infecting” new key.
- **Unbonding window alignment**:
  - Confirm evidence age limits and AVS unbonding durations match the behavior of:
    - `AppendConsensusAddrToPrune` / `GetUnbondingCompletionEpoch`.
    - `EndBlock` pruning and `ResetValidatorSigningInfo`.
  - Treat this as the same thread as checklist **B2–B4** (old key removed only after it is safe).
- **Module invariants / slashing assumptions**:
  - Run invariants focusing on:
    - Consistency of `MissedBlocksCounter` and bit arrays after key changes.
    - **Lifecycle:** old consensus addr has `ValidatorSigningInfo` when rotation `Copy` runs; after prune it may legitimately have none—see **B0–B4**, **A6**, **D4–D5**.
    - No active/powered validators for frozen operators, including across epochs and opt‑in/out sequences.
- **Operational behavior**:
  - Attempt to:
    - Unjail a frozen operator (should be blocked, with logs only).
    - Reuse a tombstoned key (should be forbidden by `IsTombstoned` check in `AfterOperatorKeySet` and `AfterOperatorKeyReplaced`).
- **Same-epoch double rotation:** Read **§4** (`alreadyRecorded`, `ClearPreviousConsensusKeys`, `TestSameEpochDoubleKeyRotation_SkipsSecondHookRisk` vs `TestChecklist_D3_DoubleKeyRotationInvariants`) when reasoning about “two key changes in one epoch” on **Msg** vs **keeper** paths.
- **Slash / prune / freeze (checklist E):** Read **Deep dive: other core functions**; run **E1–E6** tests (`keeper` + **`x/dogfood/types`** for **E3**).

