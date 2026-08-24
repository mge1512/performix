<!--
SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
SPDX-License-Identifier: Apache-2.0
-->

# Test Case 33: Periodic Synchronous Write Pressure

## Problem Summary
- Insight target: recurring write pressure causes periodic I/O wait or blocked
  work while compute capacity remains available.

## ID
- `test_case_33`

## Public Intent
- Exercise repeated storage-active and idle intervals.
- Record the recurring system-level effect of the storage work.

## What's Wrong In Current Implementation
- Small synchronous writes are issued in bursts rather than being batched or
  smoothed.
- The recurring bursts make work wait on storage even though the machine
  retains CPU compute capacity.

## What The LLM Should Suggest
- Identify and quantify the recurring active and idle pattern.
- Correlate write activity with elevated I/O wait or blocked work.
- Use write-operation and bandwidth rates together when characterising write
  size. Explicit bytes-per-operation or a definitive small-write diagnosis is
  useful supporting evidence but is not required.
- Recommend a useful next step focused on the write path. This may be a direct
  mitigation such as batching writes or reducing persistence frequency, or a
  targeted investigation of batching, synchronous writes, storage latency or
  workload attribution before changing the system.

## Expected Profiling Characteristics
- Four write-active periods are separated by idle intervals.
- Write-operation rate rises strongly during each active period, with low
  bandwidth relative to the number of operations.
- I/O wait or blocked work rises with the write bursts.
- CPU compute capacity remains available.

## Scoring Guidance
- Pass:
  - Identifies the recurring burst-and-idle pattern rather than describing a
    steady storage load.
  - Connects write activity to I/O wait or blocked work.
  - Notes that CPU saturation is not the primary problem and gives a useful
    write-path recommendation or targeted follow-up. A definitive small-write
    diagnosis or specific remediation is not required.
- Fail:
  - Diagnoses CPU saturation as the primary problem.
  - Reports storage throughput without connecting it to waiting work.
  - Averages away the periodic stalls.
  - Asserts measured latency, a specific system call or physical-device
    saturation without evidence.
  - Gives only generic advice to use a faster disk.
