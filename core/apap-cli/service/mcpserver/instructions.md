<!--
SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
SPDX-License-Identifier: Apache-2.0
-->

# Performix MCP Usage

## Terminology & Core Concepts

Arm Performix is a performance analysis toolkit for profiling an application and turning captured performance data into actionable insights. Through MCP, you can manage targets, run recipes against workloads, and analyze existing runs without requiring the user to switch to the CLI or GUI. Prefer accessing Performix via the MCP server wherever the relevant functionality is exposed via MCP.

| Term | Definition |
| --- | --- |
| Recipe | A performance analysis pathway that defines tools, parameters, execution stages and data processing. A recipe is selected by you or the user and is run against a target and workload to produce a run. |
| Target | The system on which the workload to be profiled is already running, or will be run. A target may be localhost or remote, is usually saved with a friendly name, and must be specified when running a recipe via MCP. |
| Run | The recorded result of executing a recipe against a target and workload. A run has a run ID and stores the recipe metadata, captured performance data, and artifacts needed for data rendering or producing AI-generated insights. Performix persists an archive of previous runs. |


## Overall Goals & Capabilities

This MCP server exposes Performix functionality, with the following overall goals:
1. Managing the set of targets available to Performix
2. Enabling users to generate new Performix runs against their application directly via the MCP — see the [Recipe Run Playbook](#recipe-run-playbook) section below.
3. Providing users with AI-generated insights into their application's performance, based on profiling data from a Performix run (which may have originated from the MCP, CLI or GUI).


## Recipe Run Playbook

### Existing Runs
Do not assume that you always need to do a new live recipe run, as running a recipe may take several minutes, depending on the user's workload and the chosen settings (e.g. timeout).
Take into account the following:
- Whether the user has explicitly requested a new run
- If necessary, you can use the `list_runs` tool to see pre-existing runs that are already available on the user's system


### Choosing a Recipe
Default to `code_hotspots` when a user asks to profile a workload without naming a recipe.

Use the `list_recipes` tool to check which recipes are currently available. Recipe availability is controlled by the engine recipe catalogue and configuration, so a recipe that is disabled or failed to load will not be runnable via MCP.

Use the `recipe_info` tool before running a recipe, when you need to understand recipe-specific parameters, or when the target may affect valid parameter choices. If you already know the target, pass it to `recipe_info` so target compatibility with this recipe can be validated.

Usage guidance for various recipes is provided below.

| Recipe | Usage guidance |
| --- | --- |
| `code_hotspots` | Use this as the default general-purpose profiling recipe. It is the fastest way to answer "what code is spending CPU time?" for a workload run. |
| `cpu_microarchitecture` | Use this after hotspots when the next question is why the hot code is underperforming at the microarchitectural level. |
| `memory_access` | Use this when the workload looks memory-bound or when code hotspots suggest cache or latency issues. |
| `instruction_mix` | Use this when you need a breakdown of instruction categories, compiler output, or ISA usage. |
| `asct` | Use this for Arm system characterization scenarios rather than as a first-pass workload profiling recipe. |
| `system_utilization` | Use this to see how CPU, memory, disk, and network resources are utilized while the workload runs. It may help you find saturated system resources, understand utilization trends over time, and correlate workload behavior with broader system activity. |
| `cache_sharing` | Use this when you need to understand cache line sharing, cache-to-cache transfers, or false-sharing style effects. |
| `cmn_analysis` | Use this for CMN mesh and interconnect analysis rather than as a first-pass workload profiling recipe. |
| `syscall_trace_summary` | Use this when syscall tracing data is needed to summarize operating system call behaviour during a workload run. |


### Managing Targets
The `run_recipe` MCP tool requires you to specify which `target` to run against — it does not assume fallback to the default.
Use the `list_targets` tool first to discover the available targets, including which one is the default, then pass the chosen target's name explicitly. If several targets exist and the user has not indicated a preference, confirm which one to use before running. If the user has never added a target to Performix before, the pre-configured localhost target will be the default.
New targets can be added directly via MCP using the `add_target` tool, or alternatively the user can add MCP-visible targets via the CLI or GUI. The MCP `add_target` tool enforces strict host key checking, so the target's SSH host key must already be present in the user's known_hosts file before connecting to the target.


### Workloads & Running a Recipe
You can run a recipe live using the `run_recipe` tool, which generates a new run and returns the run's ID among other relevant details.
For initial profiling runs, omit the timeout to use the MCP default of 10 seconds unless the user asks for a longer collection. Set timeout to 0 only when the user explicitly wants no collection timeout.


### Generating Insights
Use the `generate_ai_insights` tool to retrieve key data from a run to help you generate performance analysis.
