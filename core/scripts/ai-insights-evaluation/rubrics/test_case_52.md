<!--
SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
SPDX-License-Identifier: Apache-2.0
-->

# Test Case 52: Three Different CPU Bottlenecks

## Problem Summary
- CPU categories and causes: Frontend Bound from instruction-fetch pressure,
  Backend Bound from D-cache misses, and Bad Speculation from branch
  mispredictions. Each category and cause must be connected to the correct
  function.

## ID
- `test_case_52`

## Public Intent (safe summary)
- Link the unchanged generated request handler, analytics record and mixed
  packet type processing code into one application.
- Adjust only how often each part runs so that the recording contains enough
  samples from all three.

## What's Wrong In Current Implementation
- The application contains three independent performance causes. A run-wide
  summary of the top-level CPU categories cannot show which function has each
  cause.

## What The LLM Should Suggest
- For the request handlers, reduce duplicated specialised code or keep the
  frequently used code closer together.
- For the analytics records, store the frequently read values in a separate
  packed array, or split frequently and rarely used fields.
- For the packets, batch or separate packet types, or otherwise avoid the
  unpredictable branch for every packet.

## Expected Profiling Characteristics
- All three processing paths appear in the function or call-tree results.
- Each path receives at least 10% and no more than 60% of regular samples.
- The case 46 path is mainly Frontend Bound (waiting for instructions),
  supported by Level 1 instruction-cache or instruction address translation
  misses.
- The case 47 path is mainly Backend Bound (waiting for data or an execution
  unit), supported by data-cache measurements.
- The case 49 path is mainly Bad Speculation (discarding work), supported by
  incorrect branch prediction measurements.

## Scoring Guidance
- Pass:
  - Identifies all three distinct causes.
  - Uses the top-level CPU categories and supporting measurements for each path
    to connect every cause to the correct code.
  - Gives the corresponding code change for each path.
- Fail:
  - Reports only the run-wide dominant category.
  - Connects the cache, instruction-fetch or branch evidence to the wrong path.
  - Omits one cause or gives one generic recommendation for the whole process.
