// SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
// SPDX-License-Identifier: Apache-2.0

package insights

import (
	"context"
	"encoding/json"
	"slices"

	"github.com/Arm-Debug/apap-cli/apap-engine/message"
	"github.com/Arm-Debug/apap-cli/apap-engine/render"
	run "github.com/Arm-Debug/apap-cli/apap-engine/run"
)

type RunSummary struct {
	Name           string
	PromptFragment string
	Payload        string
}

// UnbudgetedRunSummarizer should be used for summarizers that cannot be truncated to fit a byte budget.
type UnbudgetedRunSummarizer struct {
	Name      string
	Summarize func(context.Context, *run.RunDescription, render.Session) (RunSummary, error)
}

// BudgetedRunSummarizer should be used for summarizers that can be truncated to fit a byte budget.
// When the summarizer is called, it will be passed its total byte limit based on the bundle budget
// and the configured weight for the summarizer.
type BudgetedRunSummarizer struct {
	Name      string
	Summarize func(context.Context, *run.RunDescription, render.Session, int) (RunSummary, error)
}

type RecipeRunSummarizers struct {
	Unbudgeted []UnbudgetedRunSummarizer
	Budgeted   []BudgetedRunSummarizerConfig
}

type BudgetedRunSummarizerConfig struct {
	BudgetedRunSummarizer
	Weight int
}

// additionalSummarizersByRecipe maps recipe names to the recipe-specific
// summarizers applied after the common run details summary.
var additionalSummarizersByRecipe = map[string]RecipeRunSummarizers{
	"code_hotspots": {
		Budgeted: []BudgetedRunSummarizerConfig{
			{BudgetedRunSummarizer: HotFunctionsSummarizer, Weight: 1},
			{BudgetedRunSummarizer: CallTreeSummarizer, Weight: 1},
			{BudgetedRunSummarizer: DisassemblyWindowsSummarizer, Weight: 1},
			{BudgetedRunSummarizer: SourceWindowsSummarizer, Weight: 1},
		},
	},
}

// SummarizersForRecipe returns common run details plus any recipe-specific
// summarizers. Summary generation does not decide whether a recipe supports
// Dynamic Insights.
func SummarizersForRecipe(recipeName string) RecipeRunSummarizers {
	additional := additionalSummarizersByRecipe[recipeName]
	return RecipeRunSummarizers{
		Unbudgeted: append([]UnbudgetedRunSummarizer{RunDetailsSummarizer}, additional.Unbudgeted...),
		Budgeted:   slices.Clone(additional.Budgeted),
	}
}

// NewRunSummary creates a RunSummary with the given name, prompt fragment, and payload, marshaling the payload to JSON.
func NewRunSummary(name, promptFragment string, payload any) (RunSummary, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return RunSummary{}, message.New(message.EngineInsightsRunSummaryMarshalFailed).WithCause(err)
	}

	return RunSummary{
		Name:           name,
		PromptFragment: promptFragment,
		Payload:        string(payloadBytes),
	}, nil
}

func runSummarySizeBytes(summary RunSummary) int {
	return len(summary.Name) + len(summary.PromptFragment) + len(summary.Payload)
}
