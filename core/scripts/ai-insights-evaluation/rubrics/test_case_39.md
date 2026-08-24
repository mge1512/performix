<!--
SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
SPDX-License-Identifier: Apache-2.0
-->

# Test Case 39: Twin Transforms With Mismatched Optimization

## Problem Summary
- Insight target: dynamic Instruction Mix run with two related local hot
  functions that perform the same floating-point transform work but do not have
  the same measured instruction profile.

## ID
- `test_case_39`

## Public Intent (safe summary)
- Run a deterministic local C++ workload over two halves of the same input.
- Keep the measured work concentrated in two similarly named local transform
  functions and report stable runtime output.

## What's Wrong In Current Implementation
- The case is designed to exercise comparative Instruction Mix reasoning rather
  than a single obvious defect.
- Two closely related transform functions should both appear in the dynamic
  evidence, but their measured mix should not be described as identical.
- One transform is intentionally built with reduced optimization, so the model
  should be open to a compiler-configuration or code-generation explanation
  instead of treating the result as a pure algorithm issue.

## What The LLM Should Suggest
- Identify the two related hot functions or clearly describe the hot work as
  split across a matched pair of transform functions.
- Describe the dynamic evidence comparatively, for example noting that one side
  looks less SIMD-led or shows more scalar/supporting work than the other.
- Suggest cautious follow-up that fits the evidence, such as checking compiler
  options, vectorization behaviour, or disassembly/source for the two
  functions.
- Avoid claiming a precise root cause that is not directly established by the
  Instruction Mix evidence alone.

## Expected Profiling Characteristics
- Dynamic function-level attribution should identify two important local
  functions with related names and meaningful self-sample coverage.
- The two hot functions should have the same overall algorithmic role but
  measurably different instruction-mix composition.
- At least one of the two functions should look more SIMD-friendly or more
  SIMD-heavy than the other.

## Scoring Guidance
- Pass:
  - Identifies the hot work as split across two related transform functions or
    an equivalent clearly paired hot path.
  - Correctly treats the run as comparative rather than as one single dominant
    hotspot with no meaningful contrast.
  - Gives at least one cautious, evidence-backed next step consistent with a
    possible optimization or code-generation difference.
- Fail:
  - Describes the two hot functions as effectively identical without caveats.
  - Invents an unrelated bottleneck unsupported by the Instruction Mix
    evidence.
  - Claims a certain root cause, such as a specific compiler bug, without
    treating it as follow-up to verify.
  - Treats the run as static-only evidence rather than dynamic function-level
    attribution.
