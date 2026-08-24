<!--
SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
SPDX-License-Identifier: Apache-2.0
-->

# Test Case 46: Too Much Generated Handler Code

## Problem Summary
- CPU category and cause: Frontend Bound — instruction-fetch pressure from an
  oversized code footprint. The CPU is often waiting for the next instructions
  because the generated handler code does not fit well in its nearby instruction
  cache or instruction address-translation cache.

## ID
- `test_case_46`

## Public Intent (safe summary)
- Process a fixed, warmed-up sequence of specialised request handlers.
- Each handler repeats common parsing and validation work, with small
  handler-specific differences.

## What's Wrong In Current Implementation
- The generated handlers duplicate too much code. The CPU cannot keep all the
  frequently used instructions in its nearby instruction caches.
- The program calls the handlers in a fixed order, so incorrect branch
  predictions are not the main cause.

## What The LLM Should Suggest
- Factor common parsing and validation out of specialisations.
- Reduce unnecessary specialisation and inlining. Keeping the most frequently
  used code close together is also acceptable.
- Do not recommend branch hints or changes to data layout as the main fix.

## Expected Profiling Characteristics
- Frontend Bound is the largest top-level CPU category and leads the next
  category by at least 15 percentage points.
- Bad Speculation, which measures work discarded after events such as incorrect
  branch predictions, remains below 10%.
- Level 1 instruction-cache (L1I) misses or instruction address translation
  cache (ITLB) misses show that the CPU has trouble fetching the handler code.
- Samples point to the generated request handlers.

## Scoring Guidance
- Pass:
  - Identifies instruction fetching as the main cause of lost CPU time.
  - Uses L1I or ITLB measurements and connects them to the handler code.
  - Recommends reducing duplicated generated code or unnecessary
    specialisation, or keeping frequently used code closer together.
- Fail:
  - Treats incorrect branch predictions as the main cause.
  - Gives only generic cache advice without connecting it to instruction
    fetching.
  - Recommends changing input order without reducing the amount of handler code.
