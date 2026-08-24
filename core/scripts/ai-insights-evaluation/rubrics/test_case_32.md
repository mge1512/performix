<!--
SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
SPDX-License-Identifier: Apache-2.0
-->

# Test Case 32: Sustained CPU Oversubscription

## Problem Summary
- Insight target: sustained CPU oversubscription after a brief baseline.

## ID
- `test_case_32`

## Public Intent
- Exercise a long-lived CPU-bound service with more runnable work than the
  available CPU capacity can execute.
- Record the sustained system-level effect of that CPU demand.

## What's Wrong In Current Implementation
- The workload keeps more CPU-bound workers runnable than the available cores
  can execute.
- It remains CPU-saturated after the brief startup baseline.

## What The LLM Should Suggest
- Identify the sustained CPU-pressure interval and quantify its time range or
  proportion of the run.
- Use both per-core utilisation and system-wide runnable demand to show that
  CPU-bound concurrency exceeds available CPU capacity.
- Give an actionable next step supported by the evidence, such as capping
  CPU-bound workers near the available core count, applying backpressure,
  reducing CPU work, adding CPU capacity or using Code Hotspots to locate
  expensive code.

## Expected Profiling Characteristics
- The short initial baseline is comparatively idle.
- Most cores are saturated for a sustained interval after startup.
- System-wide runnable demand remains above the online core count during that
  interval.

## Scoring Guidance
- Pass:
  - Identifies sustained saturation across most available cores and quantifies
    its time range or proportion of the run.
  - Uses runnable-demand evidence, not only an average CPU percentage, to
    conclude that CPU-bound concurrency exceeds available CPU capacity.
  - Gives an appropriate next step supported by the evidence, such as a
    qualified concurrency or capacity recommendation, or targeted profiling to
    locate the CPU-intensive work.
- Fail:
  - Gives only generic high-CPU advice without the runnable-demand evidence.
  - Treats storage or memory as the primary bottleneck.
  - Invents a function, algorithm or workload generator.
  - Recommends additional concurrency despite the sustained oversubscription.
