<!--
SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
SPDX-License-Identifier: Apache-2.0
-->

# Test Case 40: Static SIMD-Heavy Whole-Binary Instruction Mix

## Problem Summary
- Insight target: static-only Instruction Mix run where the primary evidence is
  whole-binary instruction-category composition rather than dynamic sampled hot
  functions.

## ID
- `test_case_40`

## Public Intent (safe summary)
- Analyze a deterministic local Linux Arm64 assembly executable whose body is
  designed to be dominated by synthetic SIMD-heavy code.
- Exercise static Instruction Mix reasoning without requiring a source bundle
  or dynamic hotspot attribution.

## What's Wrong In Current Implementation
- The case is not about proving a concrete performance defect.
- Instead, it checks whether the model can interpret static Instruction Mix
  evidence correctly, especially that this run should be described as
  whole-binary composition rather than runtime hotspot behaviour.
- The response should not invent dynamic attribution, source-backed diagnosis,
  or disassembly follow-up that is unavailable in static-only mode.

## What The LLM Should Suggest
- Describe the binary as SIMD-heavy or vector-heavy from the static
  instruction-category evidence.
- Keep the conclusion scoped to binary composition rather than sampled runtime
  cost.
- If suggesting follow-up, keep it cautious and appropriate for static mode,
  such as validating whether the binary matches the intended ISA usage or using
  a dynamic recipe next if runtime behaviour is needed.
- Avoid claiming that the binary is definitely the runtime bottleneck solely
  from static composition.

## Expected Profiling Characteristics
- Static `flat_table` evidence should show a strong SIMD-oriented instruction
  presence in the binary.
- The run should not require dynamic function-level attribution to reach the
  core conclusion.
- Source-tree follow-up and dynamic hotspot claims should be unnecessary and
  inappropriate for a correct first-pass answer.

## Scoring Guidance
- Pass:
  - Correctly identifies the run as static Instruction Mix evidence.
  - Correctly describes the binary as SIMD-heavy or strongly vector-oriented.
  - Keeps recommendations cautious and consistent with the limits of static
    evidence.
- Fail:
  - Invents a dominant runtime hot function or sampled hotspot that the static
    run does not provide.
  - Treats the binary as mostly scalar or branch-dominated without matching
    evidence.
  - Claims a definite runtime bottleneck solely from static composition.
  - Recommends source-loading or dynamic disassembly follow-up as if those were
    available by default for this static-only run.
