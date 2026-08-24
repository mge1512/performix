<!--
SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
SPDX-License-Identifier: Apache-2.0
-->

# Test Case 35: Transient CPU Spike Control

## Problem Summary
- Insight target: one short CPU spike followed by recovery does not establish a
  sustained resource bottleneck.

## ID
- `test_case_35`

## Public Intent
- Record an otherwise idle system with one short period of CPU activity.
- Test whether analysis distinguishes a transient maximum from sustained
  pressure.

## What's Wrong In Current Implementation
- No sustained performance problem is present in the recording.
- Treating the isolated CPU spike as a capacity problem would overstate the
  available evidence.

## What The LLM Should Suggest
- Identify the short CPU peak and subsequent recovery with measured time
  ranges.
- State that the data does not support a sustained CPU, memory, storage or
  network bottleneck.
- Avoid unnecessary optimisation advice.
- Suggest another representative recording only if application work was
  expected during the idle periods.

## Expected Profiling Characteristics
- An extended idle baseline precedes a short system-wide CPU peak.
- CPU utilisation returns to idle for the remainder of the recording.
- Memory, storage and network metrics do not show sustained pressure.

## Scoring Guidance
- Pass:
  - Identifies and quantifies the short CPU peak and recovery.
  - Distinguishes the isolated peak from sustained pressure.
  - States that no sustained resource bottleneck is supported by the data.
  - Avoids unnecessary optimisation advice.
- Fail:
  - Invents a bottleneck from the transient maximum.
  - Recommends CPU, storage or memory changes without supporting evidence.
  - Identifies or guesses the hidden workload implementation.
