<!--
SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
SPDX-License-Identifier: Apache-2.0
-->

# Test Case 37: Auto-Vectorized Floating-Point Loop With Strong SIMD Contribution

## Problem Summary
- Insight target: dynamic Instruction Mix run with one dominant local hot
  function whose measured instruction mix shows a strong SIMD contribution,
  alongside substantial load activity and supporting integer work.

## ID
- `test_case_37`

## Public Intent (safe summary)
- Run a deterministic local C++ floating-point transform over array-backed
  data.
- Keep the measured work in one dominant function and report stable runtime
  output.

## What's Wrong In Current Implementation
- The case is not primarily about proving a defect in the current
  implementation.
- Instead, it is meant to test whether the model can read dynamic Instruction
  Mix evidence carefully and recognise an auto-vectorized hot loop without
  overstating the result.
- The hot path is not purely SIMD instructions; measured load and integer
  activity still matter.

## What The LLM Should Suggest
- Identify the dominant hot function or clearly dominant hot path from dynamic
  evidence.
- Describe the hot path as vectorized or SIMD-led, while acknowledging the
  measured load and address/update work around it.
- If suggesting follow-up analysis, keep it evidence-backed and lightweight,
  such as checking source or disassembly to confirm the vectorized loop
  structure or looking for remaining non-SIMD overhead in the hot path.
- Avoid inventing a major optimization problem when the run mainly shows
  composition.

## Expected Profiling Characteristics
- Dynamic function-level attribution should identify one clearly dominant local
  function.
- The hot function should show meaningful Advanced SIMD share, plus visible
  load activity and supporting integer work.
- The run should provide enough `samples.count.self` coverage for the dominant
  function to make attribution meaningful.

## Scoring Guidance
- Pass:
  - Identifies the dominant hot function or clearly dominant hot path from the
    dynamic evidence.
  - Correctly describes the run as vectorized or SIMD-led rather than mostly
    scalar.
  - Does not require the SIMD share to be nearly exclusive, and does not
    ignore the measured load activity.
  - Keeps any follow-up recommendation cautious and evidence-backed.
- Fail:
  - Treats the run as mostly scalar or branch-dominated.
  - Claims that the hot path is almost entirely SIMD without acknowledging the
    mixed instruction profile.
  - Invents an unrelated bottleneck that is not supported by the Instruction
    Mix evidence.
  - Forces a strong “fix this with SIMD” diagnosis when the run already shows
    meaningful SIMD usage.
