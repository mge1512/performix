<!--
SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
SPDX-License-Identifier: Apache-2.0
-->

# Test Case 34: Memory Saturation

## Problem Summary
- Insight target: a recurring high-memory phase exceeds physical memory and
  causes paging and storage wait.

## ID
- `test_case_34`

## Public Intent
- Alternate between low memory use and several short periods where the
  aggregate worker working set is larger than physical memory.
- Repeat the high-memory period so its temporal pattern is visible across the
  recording.

## What's Wrong In Current Implementation
- A recurring operation temporarily activates more memory than the machine can
  hold in RAM.
- During each event, active pages are read from and written to swap, making
  storage rather than CPU compute capacity limit progress.

## What The LLM Should Suggest
- Identify that low-memory-headroom and paging occur repeatedly rather than as
  a single event or a continuous condition. Exact interval timing or cadence
  is useful supporting evidence but is not required.
- Correlate swap occupancy and major page faults with storage activity, I/O
  wait or blocked work before diagnosing active paging.
- Note that CPU saturation is not the primary limitation when compute capacity
  remains available during the paging interval.
- Recommend reducing or bounding the active working set, for example through
  worker concurrency, per-worker heaps or caches, batching or shared data, or
  selecting a machine with more physical memory.
- If suggesting a controlled experiment with swap disabled, explain that it
  can replace paging with allocation failure or OOM termination and does not
  resolve the underlying working-set pressure.
- Do not require a numerical worker count, per-worker limit or machine size:
  the System Utilisation recording does not expose the configuration needed to
  calculate one.

## Expected Profiling Characteristics
- Available memory falls during each high-memory event and recovers between
  events.
- Swap occupancy, major page faults, storage reads and writes, I/O wait or
  blocked work rise together during several separate events.
- CPU compute capacity remains available despite slow progress.
- Memory, fault and storage pressure recover during the low-memory intervals.

## Scoring Guidance
- Pass:
  - Identifies repeated paging events rather than reporting only one peak or a
    continuous run-wide condition. The exact period does not need to be
    quantified.
  - Uses correlated major-fault and storage or wait evidence, not swap
    occupancy alone.
  - Recognises physical-memory or working-set pressure and quantifies material
    intervals or measured resource levels.
  - Gives an actionable working-set or physical-memory capacity
    recommendation without inventing unsupported configuration values.
- Fail:
  - Claims active paging or swap thrashing from swap occupancy alone.
  - Misses the recurrence and describes the condition as continuous across the
    complete run.
  - Diagnoses CPU saturation as the primary problem.
  - Definitively attributes the problem to a leak, allocator, garbage
    collector or specific process.
  - Recommends disabling swap as a remedy, or suggests testing with swap
    disabled without noting the risk of allocation failure or OOM termination.
  - Recommends only faster storage or collecting another profile without
    addressing working-set or physical-memory capacity.
