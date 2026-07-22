// SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
// SPDX-License-Identifier: Apache-2.0

package recipeparser

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Arm-Debug/apap-cli/apap-engine/conductor"
	"github.com/Arm-Debug/apap-cli/apap-engine/deploymentsupport"
	"github.com/Arm-Debug/apap-cli/apap-engine/parameters"
	"github.com/Arm-Debug/apap-cli/apap-engine/recipe"
	"github.com/Arm-Debug/apap-cli/apap-engine/recipe/runtime"
	"github.com/Arm-Debug/apap-cli/apap-engine/recipe/stages"
	"github.com/Arm-Debug/apap-cli/apap-engine/run"
	"github.com/Arm-Debug/apap-cli/apap-engine/target"
	tool_goja "github.com/Arm-Debug/apap-cli/apap-engine/tool/goja"
)

func TestCodeHotspotsResolvesAndroidSlRecordBundle(t *testing.T) {
	recipePath := filepath.Join("..", "..", "..", "core", "apap-cli", "recipes", "code_hotspots.js")
	recipeData, err := os.ReadFile(recipePath)
	require.NoError(t, err)

	parser := RecipeParserJS{APIFactory: CreateConcreteAPI}
	parsedRecipe, err := parser.ParseRecipe(string(recipeData))
	require.NoError(t, err)

	toolPath := filepath.Join("..", "..", "..", "core", "apap-cli", "tool-integrations", "neoprof.js")
	toolData, err := os.ReadFile(toolPath)
	require.NoError(t, err)
	neoprof, err := tool_goja.LoadFromSource(string(toolData), toolPath)
	require.NoError(t, err)

	bundles, err := deploymentsupport.ResolveToolBundles(
		context.Background(),
		conductor.PlatformConfiguration{OS: conductor.Android, Architecture: conductor.AArch64},
		nil,
		parsedRecipe.Deployments,
		func(name, version string) ([]deploymentsupport.DeploymentDeclaration, error) {
			assert.Equal(t, "neoprof", name)
			assert.Equal(t, "1.1.0", version)
			return neoprof.Deployments(), nil
		},
	)
	require.NoError(t, err)
	assert.Equal(t, []deploymentsupport.ToolBundleInfo{
		{Name: "sl-record", Version: "2.1.0-build-1", Locality: deploymentsupport.DeploymentLocalityTarget},
	}, bundles)
}

func TestCPUMicroarchitectureOptionsFallBackToN1Telemetry(t *testing.T) {
	recipeData, err := os.ReadFile(filepath.Join("..", "..", "..", "core/apap-cli/recipes/cpu_microarchitecture.js"))
	require.NoError(t, err)

	parser := RecipeParserJS{APIFactory: CreateConcreteAPI}
	recipeDefinition, err := parser.ParseRecipe(string(recipeData))
	require.NoError(t, err)
	require.Len(t, recipeDefinition.ParameterOptionsStages, 1)

	stageContext := &recipe.StageContext{
		ParameterOptions: recipe.ParameterOptions{MultiSelectOptions: make([][]parameters.ParameterOption, 1)},
	}
	recipeStage := &stages.CustomRecipeStage{
		ScriptedStage: recipeDefinition.ParameterOptionsStages[0],
		Ctx: &recipe.RunExecutionContext{
			RecipeCtx: &recipe.RecipeCtx{
				ParamValues: parameters.BoundParameters{Parameters: recipeDefinition.Parameters},
			},
			TargetInfoSupplier: func() *target.Description {
				return &target.Description{PrimaryCPUName: "Cortex-A76"}
			},
		},
	}

	_, err = recipeStage.Execute(stageContext)
	require.NoError(t, err)
	assert.Contains(t, stageContext.ParameterOptions.MultiSelectOptions[0], parameters.ParameterOption{
		Value: "cycle_accounting",
		Label: "cycle_accounting",
	})
}

func TestRerenderCapableRecipesWireTimeFilterParameters(t *testing.T) {
	tests := []struct {
		name                  string
		recipeFile            string
		timeFilteredWidgetIDs []string
	}{
		{
			name:                  "code hotspots",
			recipeFile:            "core/apap-cli/recipes/code_hotspots.js",
			timeFilteredWidgetIDs: []string{"flame_graph", "functions", "call_stack"},
		},
		{
			name:                  "cpu microarchitecture",
			recipeFile:            "core/apap-cli/recipes/cpu_microarchitecture.js",
			timeFilteredWidgetIDs: []string{"node_graph", "functions", "call_stack"},
		},
		{
			name:                  "instruction mix",
			recipeFile:            "core/apap-cli/recipes/instruction_mix.js",
			timeFilteredWidgetIDs: []string{"instruction_mix", "functions"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("declares start and end time render parameters", func(t *testing.T) {
				recipeData, err := os.ReadFile(filepath.Join("..", "..", "..", tt.recipeFile))
				require.NoError(t, err)
				parser := RecipeParserJS{APIFactory: CreateConcreteAPI}
				recipeProp, err := parser.ParseRecipe(string(recipeData))
				require.NoError(t, err)

				renderParameterTypes := make(map[string]parameters.RenderParameterValueType, len(recipeProp.RenderParameters))
				for _, renderParameter := range recipeProp.RenderParameters {
					renderParameterTypes[renderParameter.ID] = renderParameter.Type
				}
				assert.Equal(t, parameters.RenderParameterValueTypeNumber, renderParameterTypes["filter_start_time_ns"])
				assert.Equal(t, parameters.RenderParameterValueTypeNumber, renderParameterTypes["filter_end_time_ns"])
			})

			t.Run("adds time range renderer only when rerendering is enabled", func(t *testing.T) {
				recipeData, err := os.ReadFile(filepath.Join("..", "..", "..", tt.recipeFile))
				require.NoError(t, err)
				parser := RecipeParserJS{APIFactory: CreateConcreteAPI}
				recipeProp, err := parser.ParseRecipe(string(recipeData))
				require.NoError(t, err)

				rerenderOutput := executeTimeFilterRenderStage(t, recipeProp, true, map[string]any{
					"filter_pid":           nil,
					"filter_tid":           nil,
					"filter_start_time_ns": float64(1000),
					"filter_end_time_ns":   float64(2000),
				})

				renderersByID := make(map[string]recipe.RendererConfig, len(rerenderOutput.Renderers))
				for _, renderer := range rerenderOutput.Renderers {
					renderersByID[renderer.ID] = renderer
				}
				assert.Equal(t, "TimeRangeParser", renderersByID["time_range"].Type)
				require.Equal(t, "SlAnalyzeRenderer", renderersByID["sl_analyze"].Type)
				assert.Equal(t, int64(1000), renderersByID["sl_analyze"].Config["filter_start_time_ns"])
				assert.Equal(t, int64(2000), renderersByID["sl_analyze"].Config["filter_end_time_ns"])

				widgetsByID := make(map[string]recipe.WidgetConfig, len(rerenderOutput.Widgets))
				for _, widget := range rerenderOutput.Widgets {
					widgetsByID[widget.ID] = widget
				}
				timeRangeWidget, ok := widgetsByID["time_range"]
				require.True(t, ok)
				assert.Equal(t, "time_range_filter", timeRangeWidget.Type)
				assert.Equal(t, "time_range", timeRangeWidget.RendererID)
				assert.Equal(t, "top_bar_filters", timeRangeWidget.Placement)
				assert.Equal(t, map[string]string{
					"filter_start_time": "filter_start_time_ns",
					"filter_end_time":   "filter_end_time_ns",
				}, timeRangeWidget.ParameterBindings)

				const noDataMessage = "No samples match the selected time range. Try widening or clearing the time range filter."
				for _, widgetID := range tt.timeFilteredWidgetIDs {
					assert.Equal(t, noDataMessage, widgetsByID[widgetID].Config["noDataMessage"])
				}

				nonRerenderOutput := executeTimeFilterRenderStage(t, recipeProp, false, map[string]any{
					"filter_pid":           nil,
					"filter_tid":           nil,
					"filter_start_time_ns": nil,
					"filter_end_time_ns":   nil,
				})
				for _, renderer := range nonRerenderOutput.Renderers {
					assert.False(t, renderer.Type == "TimeRangeParser" && renderer.ID == "time_range")
				}
				nonRerenderWidgetsByID := make(map[string]recipe.WidgetConfig, len(nonRerenderOutput.Widgets))
				for _, widget := range nonRerenderOutput.Widgets {
					nonRerenderWidgetsByID[widget.ID] = widget
					assert.False(t, widget.ID == "time_range" && widget.Placement == "top_bar_filters")
				}
				for _, widgetID := range tt.timeFilteredWidgetIDs {
					assert.NotContains(t, nonRerenderWidgetsByID[widgetID].Config, "noDataMessage")
				}
			})

		})
	}
}

func executeTimeFilterRenderStage(
	t *testing.T,
	recipeProp recipe.Recipe,
	rerenderingEnabled bool,
	renderParams map[string]any,
) recipe.RenderOutput {
	t.Helper()

	renderNotifier := &runtime.RendererStageCollector{}
	stageContext := &recipe.StageContext{RendererNotifier: renderNotifier}
	recipeStage := &stages.CustomRecipeStage{
		StageName:     recipeProp.RenderStages[0].Name(),
		ScriptedStage: recipeProp.RenderStages[0],
		Ctx: &recipe.RunExecutionContext{
			RecipeCtx: &recipe.RecipeCtx{
				RenderParamValues: renderParams,
			},
			RunDescriptions: []*run.RunDescription{
				{Parameters: map[string]any{"mode": "dynamic"}},
			},
			RerenderingEnabled: rerenderingEnabled,
		},
	}

	_, err := recipeStage.Execute(stageContext)
	require.NoError(t, err)
	return renderNotifier.Output
}
