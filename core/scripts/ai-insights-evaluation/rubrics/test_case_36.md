<!--
SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
SPDX-License-Identifier: Apache-2.0
-->

# Test Case 36: Multi-Hotspot Pipeline Requiring Function-Level Attribution

## Problem Summary
- Insight target: Instruction Mix run with multiple meaningful local hot
  functions whose measured mixes differ enough that root totals alone are not
  sufficient for a good answer.

## ID
- `test_case_36`

## Public Intent (safe summary)
- Run a deterministic local C++ pipeline over array-backed data.
- Keep the measured work concentrated in a small set of named local functions
  and report stable runtime output.

## What's Wrong In Current Implementation
- The case is designed to exercise function-level Instruction Mix reasoning
  rather than a single obvious defect.
- Multiple local functions should appear in the dynamic evidence with
  materially different instruction-mix profiles, with a branch-heavy
  classification path, a scalar multi-pass finalization path, and a smaller
  SIMD-led transform stage.
- The model should not flatten those stages into one generic global diagnosis,
  and should keep any static composition observations separate from the dynamic
  hotspot attribution.

## What The LLM Should Suggest
- Identify multiple important hot functions from the dynamic function-level
  evidence rather than relying only on top-level totals.
- Describe at least two of those functions as doing different kinds of work,
  such as branch-heavy classification, scalar multi-pass finalization, or
  SIMD-led numeric transform work.
- Suggest at least one cautious, evidence-backed improvement or follow-up that
  is tied to the right function, for example simplifying control flow in the
  branch-heavy stage or reducing repeated passes and memory traffic in the
  finalization stage, while not recommending “add SIMD” to a stage that already
  looks SIMD-led.
- If discussing the SIMD-led transform, treat it as a lower-priority
  optimization target when it has clearly smaller sample share than the
  classification and finalization paths.
- Keep dynamic hotspot attribution and any static composition comments clearly
  separated if both evidence streams are mentioned.

## Expected Profiling Characteristics
- Dynamic function-level attribution should identify multiple important local
  functions with meaningful self-sample coverage.
- `finalize_records_36` should appear as the largest single sampled function
  with mixed predominantly scalar integer/load/floating-point work and
  substantially less sampled SIMD than `transform_records_36`.
- `classify_records_36` and `classify_tag` should appear as a second major
  branch/integer-heavy path with lower IPC than the overall run.
- `transform_records_36` should appear as a materially smaller but clearly
  SIMD-led function.
- The run should provide enough `samples.count.self` coverage across the hot
  set for comparative attribution to be meaningful.

## Scoring Guidance
- Pass:
  - Identifies multiple important hot functions or an equivalent clearly split
    hot path from dynamic evidence.
  - Correctly attributes different instruction-mix characteristics to the
    relevant functions rather than flattening the run into one generic
    description.
  - Gives at least one cautious, evidence-backed follow-up or improvement tied
    to the right function.
  - If it discusses the SIMD-led transform stage, correctly treats it as
    already vectorized or lower priority when its sample share is materially
    smaller than the main hotspots.
  - Does not need to mention `transform_records_36` when it correctly
    distinguishes at least two major function-level paths. Omission of the
    smaller transform stage alone is not a reason to fail the response.
  - Does not confuse static composition with dynamic hotspot attribution.
- Fail:
  - Relies only on root totals or describes the workload as one generic loop
    without using function-level attribution.
  - Applies the same optimization suggestion globally without regard to which
    function is actually branch-heavy, SIMD-led, or load/store-heavy.
  - Recommends adding SIMD to a stage that already looks SIMD-led, or treats a
    branch-heavy stage as primarily a vectorization issue without caveats.
  - Invents an unrelated bottleneck that is not supported by the Instruction
    Mix evidence.
