// SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
// SPDX-License-Identifier: Apache-2.0

package recipejstest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Arm-Debug/apap-cli/apap-engine/cdf"
	"github.com/Arm-Debug/apap-cli/apap-engine/recipe"
	"github.com/Arm-Debug/apap-cli/apap-engine/render"
	"github.com/Arm-Debug/apap-cli/apap-engine/run"
)

func TestCodeHotspotsTimelineVisibilityRules(t *testing.T) {
	t.Run("includes timeline for single run with provisional parquet components", func(t *testing.T) {
		runRoot := t.TempDir()
		model := newRunComponentPresenceModel(t, runRoot, []timelineComponentFixture{
			counterCapabilityFixture(t, "counter.key_type_0.series_4", map[string]any{
				"title":       "Cycles: CPU Cycles",
				"description": "CPU cycle count.",
				"units":       "cycles",
				"key_type":    0,
				"series_id":   4,
			}),
			counterCapabilityFixture(t, "counter.key_type_0.series_6", map[string]any{
				"title":       "Instructions: Executed",
				"description": "Executed instruction count.",
				"units":       "instructions",
				"key_type":    0,
				"series_id":   6,
			}),
			{
				RelativePath: "tool/neoprof/0/output/parquet/timeline/counter_series_files.parquet",
				ComponentType: cdf.ComponentType{
					Name:          "timeline-counter-series-files-metadata",
					SchemaVersion: "1.0",
				},
			},
			{
				RelativePath: "tool/neoprof/0/output/parquet/timeline/key_type=0/series_id=4/bin_duration=10000/counter.parquet",
				ComponentType: cdf.ComponentType{
					Name:          "hotspots-provisional-parquet",
					SchemaVersion: "1.0",
				},
			},
			{
				RelativePath: "tool/neoprof/0/output/parquet/timeline/key_type=0/series_id=6/bin_duration=10000/counter.parquet",
				ComponentType: cdf.ComponentType{
					Name:          "hotspots-provisional-parquet",
					SchemaVersion: "1.0",
				},
			},
		})

		output, err := executeCodeHotspotsRenderStage(
			t,
			[]*run.RunDescription{{ToolsUsed: []cdf.ToolUsed{{Tool: "neoprof"}}}},
			[]cdf.ModelView{model},
		)
		require.NoError(t, err)

		timelineWidget := requireTimelineWidget(t, output)
		require.Equal(t, "timeline", timelineWidget.Type)
		require.NotEmpty(t, timelineWidget.RendererID)
		require.NotNil(t, timelineWidget.Config)
		require.Contains(t, timelineWidget.Config, "groups")

		groups, ok := timelineWidget.Config["groups"].(map[string]any)
		require.True(t, ok)
		require.Contains(t, groups, "key_0_series_4_10000")
		require.Contains(t, groups, "key_0_series_6_10000")
	})

	t.Run("distinguishes counters with the same series ID", func(t *testing.T) {
		runRoot := t.TempDir()
		model := newRunComponentPresenceModel(t, runRoot, []timelineComponentFixture{
			counterCapabilityFixture(t, "counter.key_type_8.series_4", map[string]any{
				"title":       "CPU Cycles",
				"description": "CPU cycle count.",
				"units":       "cycles",
				"key_type":    8,
				"series_id":   4,
			}),
			counterCapabilityFixture(t, "counter.key_type_9.series_4", map[string]any{
				"title":       "GPU Cycles",
				"description": "GPU cycle count.",
				"units":       "cycles",
				"key_type":    9,
				"series_id":   4,
			}),
			{
				RelativePath: "tool/neoprof/0/output/parquet/timeline/key_type=8/series_id=4/bin_duration=10000/counter.parquet",
				ComponentType: cdf.ComponentType{
					Name:          "hotspots-provisional-parquet",
					SchemaVersion: "1.0",
				},
			},
			{
				RelativePath: "tool/neoprof/0/output/parquet/timeline/key_type=9/series_id=4/bin_duration=10000/counter.parquet",
				ComponentType: cdf.ComponentType{
					Name:          "hotspots-provisional-parquet",
					SchemaVersion: "1.0",
				},
			},
		})

		output, err := executeCodeHotspotsRenderStage(
			t,
			[]*run.RunDescription{{ToolsUsed: []cdf.ToolUsed{{Tool: "neoprof"}}}},
			[]cdf.ModelView{model},
		)
		require.NoError(t, err)

		groups := requireTimelineGroups(t, requireTimelineWidget(t, output))
		require.Equal(t, "CPU Cycles", groups["key_8_series_4_10000"].(map[string]any)["title"])
		require.Equal(t, "GPU Cycles", groups["key_9_series_4_10000"].(map[string]any)["title"])
	})

	t.Run("does not include timeline without required parquet inputs", func(t *testing.T) {
		runRoot := t.TempDir()
		model := newRunComponentPresenceModel(t, runRoot, []timelineComponentFixture{{
			RelativePath: "tool/neoprof/0/output/parquet/timeline/hotspots_timeline.csv",
			ComponentType: cdf.ComponentType{
				Name:          "hotspots-provisional-csv",
				SchemaVersion: "1.0",
			},
		}})

		output, err := executeCodeHotspotsRenderStage(
			t,
			[]*run.RunDescription{{ToolsUsed: []cdf.ToolUsed{{Tool: "neoprof"}}}},
			[]cdf.ModelView{model},
		)
		require.NoError(t, err)

		requireNoTimelineWidget(t, output)
	})

	t.Run("does not include timeline for comparison runs", func(t *testing.T) {
		runRoot := t.TempDir()
		model := newRunComponentPresenceModel(t, runRoot, []timelineComponentFixture{
			{
				RelativePath: "tool/neoprof/0/output/parquet/timeline/counter_series_files.parquet",
				ComponentType: cdf.ComponentType{
					Name:          "timeline-counter-series-files-metadata",
					SchemaVersion: "1.0",
				},
			},
			{
				RelativePath: "tool/neoprof/0/output/parquet/timeline/key_type=0/series_id=1/bin_duration=1000000/counter.parquet",
				ComponentType: cdf.ComponentType{
					Name:          "hotspots-provisional-parquet",
					SchemaVersion: "1.0",
				},
			},
		})

		output, err := executeCodeHotspotsRenderStage(
			t,
			[]*run.RunDescription{
				{ToolsUsed: []cdf.ToolUsed{{Tool: "neoprof"}}},
				{ToolsUsed: []cdf.ToolUsed{{Tool: "neoprof"}}},
			},
			[]cdf.ModelView{model, model},
		)
		require.NoError(t, err)

		requireNoTimelineWidget(t, output)
	})
}

func TestCodeHotspotsTimelineUsesCounterCapabilityMetadata(t *testing.T) {
	runRoot := t.TempDir()
	model := newRunComponentPresenceModel(t, runRoot, []timelineComponentFixture{
		counterCapabilityFixture(t, "counter.instructions.executed", map[string]any{
			"title":       "Instructions (Executed): All",
			"description": "The counter increments for every executed instruction.",
			"units":       "instructions",
			"key_type":    0,
			"series_id":   16,
		}),
		counterCapabilityFixture(t, "counter.branch.mispredictions", map[string]any{
			"title":       "Branch Predictor: Mispredictions",
			"description": "The counter increments for every misprediction.",
			"units":       "",
			"key_type":    0,
			"series_id":   18,
		}),
		{
			RelativePath: "tool/neoprof/0/capabilities/unrelated.json",
			ComponentType: cdf.ComponentType{
				Name:          "tool_capabilities/unrelated",
				SchemaVersion: "1.0",
			},
			Contents: []byte(`{"state":"collected","payload":null}`),
		},
		counterParquetComponentFixture(16, 20_000),
		counterParquetComponentFixture(18, 10_000),
		counterParquetComponentFixture(16, 10_000),
	})

	output, err := executeCodeHotspotsRenderStage(
		t,
		[]*run.RunDescription{{ToolsUsed: []cdf.ToolUsed{{Tool: "neoprof"}}}},
		[]cdf.ModelView{model},
	)
	require.NoError(t, err)

	timelineWidget := requireTimelineWidget(t, output)
	require.Equal(t, "s", timelineWidget.Config["xAxisUnit"])
	groups := requireTimelineGroups(t, timelineWidget)
	series16At10k := requireTimelineGroup(t, groups, "key_0_series_16_10000")
	series16At20k := requireTimelineGroup(t, groups, "key_0_series_16_20000")
	require.Equal(t, "Instructions (Executed): All", series16At10k["title"])
	require.Equal(t, "Instructions (Executed): All", series16At20k["title"])
	require.Equal(t, "The counter increments for every executed instruction.", series16At10k["description"])
	series16Config := requireTimelineGroupConfig(t, series16At10k)
	require.Equal(t, "Time (s)", series16Config["xAxisTitle"])
	require.Equal(t, "instructions", series16Config["yAxisUnit"])

	series18 := requireTimelineGroup(t, groups, "key_0_series_18_10000")
	require.Equal(t, "Branch Predictor: Mispredictions", series18["title"])
	require.NotContains(t, requireTimelineGroupConfig(t, series18), "yAxisUnit")
}

func TestCodeHotspotsTimelineUsesGenericPresentationForRunWithoutCapabilities(t *testing.T) {
	runRoot := t.TempDir()
	model := newRunComponentPresenceModel(t, runRoot, []timelineComponentFixture{
		counterParquetComponentFixture(10, 100_000_000),
	})

	output, err := executeCodeHotspotsRenderStage(
		t,
		[]*run.RunDescription{{ToolsUsed: []cdf.ToolUsed{{Tool: "neoprof"}}}},
		[]cdf.ModelView{model},
	)
	require.NoError(t, err)

	groups := requireTimelineGroups(t, requireTimelineWidget(t, output))
	series10 := requireTimelineGroup(t, groups, "key_0_series_10_100000000")
	require.Equal(t, "Key 0, Series 10", series10["title"])
	require.Equal(
		t,
		"Provisional timeline series Key 0, Series 10 at 100000000 ns resolution.",
		series10["description"],
	)
	require.NotContains(t, requireTimelineGroupConfig(t, series10), "yAxisUnit")
}

func TestCodeHotspotsTimelineUsesGenericPresentationForSeriesWithoutMetadata(t *testing.T) {
	runRoot := t.TempDir()
	model := newRunComponentPresenceModel(t, runRoot, []timelineComponentFixture{
		counterCapabilityFixture(t, "counter.instructions", map[string]any{
			"title":       "Instructions: Executed",
			"description": "Executed instruction count.",
			"units":       "instructions",
			"key_type":    0,
			"series_id":   6,
		}),
		counterParquetComponentFixture(4, 10_000),
		counterParquetComponentFixture(6, 10_000),
	})

	output, err := executeCodeHotspotsRenderStage(
		t,
		[]*run.RunDescription{{ToolsUsed: []cdf.ToolUsed{{Tool: "neoprof"}}}},
		[]cdf.ModelView{model},
	)
	require.NoError(t, err)

	groups := requireTimelineGroups(t, requireTimelineWidget(t, output))
	series4 := requireTimelineGroup(t, groups, "key_0_series_4_10000")
	require.Equal(t, "Key 0, Series 4", series4["title"])
	require.Equal(
		t,
		"Provisional timeline series Key 0, Series 4 at 10000 ns resolution.",
		series4["description"],
	)
	require.NotContains(t, requireTimelineGroupConfig(t, series4), "yAxisUnit")

	series6 := requireTimelineGroup(t, groups, "key_0_series_6_10000")
	require.Equal(t, "Instructions: Executed", series6["title"])
	require.Equal(t, "Executed instruction count.", series6["description"])
	require.Equal(
		t,
		"instructions",
		requireTimelineGroupConfig(t, series6)["yAxisUnit"],
	)
}

func TestCodeHotspotsTimelineRejectsInvalidCounterCapabilityMetadata(t *testing.T) {
	tests := []struct {
		name          string
		capabilities  []timelineComponentFixture
		expectedError string
	}{
		{
			name: "invalid series id",
			capabilities: []timelineComponentFixture{counterCapabilityFixture(t, "counter.invalid", map[string]any{
				"title":       "Cycles: CPU Cycles",
				"description": "CPU cycle count.",
				"units":       "cycles",
				"key_type":    0,
				"series_id":   -1,
			})},
			expectedError: "series_id must be a non-negative safe integer",
		},
		{
			name: "duplicate counter identity",
			capabilities: []timelineComponentFixture{
				counterCapabilityFixture(t, "counter.cycles", map[string]any{
					"title":       "Cycles: CPU Cycles",
					"description": "CPU cycle count.",
					"units":       "cycles",
					"key_type":    0,
					"series_id":   4,
				}),
				counterCapabilityFixture(t, "counter.other_cycles", map[string]any{
					"title":       "Cycles: Other Cycles",
					"description": "Other cycle count.",
					"units":       "cycles",
					"key_type":    0,
					"series_id":   4,
				}),
			},
			expectedError: "Duplicate timeline counter metadata for key_0_series_4",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			runRoot := t.TempDir()
			components := append(
				[]timelineComponentFixture{counterParquetComponentFixture(4, 10_000)},
				test.capabilities...,
			)
			model := newRunComponentPresenceModel(t, runRoot, components)

			_, err := executeCodeHotspotsRenderStage(
				t,
				[]*run.RunDescription{{ToolsUsed: []cdf.ToolUsed{{Tool: "neoprof"}}}},
				[]cdf.ModelView{model},
			)
			require.ErrorContains(t, err, test.expectedError)
		})
	}
}

func TestCodeHotspotsTimelinePresentationQuerySumsDevicesAndThreadsAndZeroFillsGaps(t *testing.T) {
	runRoot := t.TempDir()
	fixture := writeTimelineBinnedDeltaParquetFixture(t, runRoot, []timelineCounterSeriesFixture{
		{
			SeriesID:    4,
			BinDuration: 10_000,
			CounterRows: []timelineCounterRowFixture{
				{
					StartTimestamp: 1_000_000_000,
					EndTimestamp:   1_000_020_000,
					DeviceNo:       7,
					Thread:         23,
					Value:          3,
				},
				{
					StartTimestamp: 1_000_000_000,
					EndTimestamp:   1_000_010_000,
					DeviceNo:       7,
					Thread:         31,
					Value:          6,
				},
				{
					StartTimestamp: 1_000_000_000,
					EndTimestamp:   1_000_010_000,
					DeviceNo:       8,
					Thread:         41,
					Value:          9,
				},
				{
					StartTimestamp: 1_000_030_000,
					EndTimestamp:   1_000_040_000,
					DeviceNo:       7,
					Thread:         31,
					Value:          6,
				},
			},
		},
		{
			SeriesID:    6,
			BinDuration: 10_000,
			CounterRows: []timelineCounterRowFixture{{
				StartTimestamp: 1_000_000_000,
				EndTimestamp:   1_000_010_000,
				DeviceNo:       9,
				Thread:         41,
				Value:          99,
			}},
		},
	})
	model := newCodeHotspotsTimelineFixtureModel(t, runRoot, fixture)

	output, err := executeCodeHotspotsRenderStage(
		t,
		[]*run.RunDescription{{ToolsUsed: []cdf.ToolUsed{{Tool: "neoprof"}}}},
		[]cdf.ModelView{model},
	)
	require.NoError(t, err)

	timelineWidget := requireTimelineWidget(t, output)
	session := newRenderSession(t, "timeline-code-hotspots-run", runRoot, model)
	renderers := initializeRenderers(t, session, output, func(rendererConfig recipe.RendererConfig) bool {
		return rendererConfig.Type == "SQL" && strings.HasPrefix(rendererConfig.ID, "timeline_")
	})

	query := resolveTimelineGroupQuery(t, session, renderers, timelineWidget, "key_0_series_4_10000")
	rows, err := session.Database().Conn.QueryContext(context.Background(), query)
	require.NoError(t, err)
	defer rows.Close()

	columns, err := rows.Columns()
	require.NoError(t, err)
	require.Equal(t, []string{"x_start", "value"}, columns)

	require.True(t, rows.Next())
	var row1XStart float64
	var row1Value float64
	require.NoError(t, rows.Scan(&row1XStart, &row1Value))
	require.InDelta(t, 1.0, row1XStart, 1e-12)
	require.Equal(t, 18.0, row1Value)

	require.True(t, rows.Next())
	var row2XStart float64
	var row2Value float64
	require.NoError(t, rows.Scan(&row2XStart, &row2Value))
	require.InDelta(t, 1.00001, row2XStart, 1e-12)
	require.Equal(t, 3.0, row2Value)

	require.True(t, rows.Next())
	var row3XStart float64
	var row3Value float64
	require.NoError(t, rows.Scan(&row3XStart, &row3Value))
	require.InDelta(t, 1.00002, row3XStart, 1e-12)
	require.Equal(t, 0.0, row3Value)

	require.True(t, rows.Next())
	var row4XStart float64
	var row4Value float64
	require.NoError(t, rows.Scan(&row4XStart, &row4Value))
	require.InDelta(t, 1.00003, row4XStart, 1e-12)
	require.Equal(t, 6.0, row4Value)

	require.False(t, rows.Next())
	require.NoError(t, rows.Err())
}

type timelineComponentFixture struct {
	RelativePath  string
	ComponentType cdf.ComponentType
	Contents      []byte
}

func counterParquetComponentFixture(seriesID, binDuration int64) timelineComponentFixture {
	return timelineComponentFixture{
		RelativePath: fmt.Sprintf(
			"tool/neoprof/0/output/parquet/timeline/key_type=0/series_id=%d/bin_duration=%d/counter.parquet",
			seriesID,
			binDuration,
		),
		ComponentType: cdf.ComponentType{
			Name:          "hotspots-provisional-parquet",
			SchemaVersion: "1.0",
		},
	}
}

func counterCapabilityFixture(
	t *testing.T,
	capabilityID string,
	payload map[string]any,
) timelineComponentFixture {
	t.Helper()

	contents, err := json.Marshal(map[string]any{
		"state":   "collected",
		"payload": payload,
	})
	require.NoError(t, err)

	return timelineComponentFixture{
		RelativePath: fmt.Sprintf(
			"tool/neoprof/0/capabilities/%s.json",
			capabilityID,
		),
		ComponentType: cdf.ComponentType{
			Name:          "tool_capabilities/counter",
			SchemaVersion: "1.0",
		},
		Contents: contents,
	}
}

func executeCodeHotspotsRenderStage(
	t *testing.T,
	runDescriptions []*run.RunDescription,
	runModels []cdf.ModelView,
) (recipe.RenderOutput, error) {
	t.Helper()

	return executeRenderStage(
		t,
		parseRecipeFile(t, "code_hotspots.js"),
		runDescriptions,
		runModels,
		map[string]any{
			"filter_pid":           nil,
			"filter_tid":           nil,
			"filter_start_time_ns": nil,
			"filter_end_time_ns":   nil,
		},
		renderStageOptions{
			neoprofTimelineEnabled: true,
		},
	)
}

func requireTimelineWidget(t *testing.T, output recipe.RenderOutput) *recipe.WidgetConfig {
	t.Helper()

	for i := range output.Widgets {
		if output.Widgets[i].ID == "timeline" {
			return &output.Widgets[i]
		}
	}

	t.Fatal("timeline widget not found")
	return nil
}

func requireNoTimelineWidget(t *testing.T, output recipe.RenderOutput) {
	t.Helper()

	for _, widget := range output.Widgets {
		require.NotEqual(t, "timeline", widget.ID)
	}
}

func requireTimelineGroups(t *testing.T, widget *recipe.WidgetConfig) map[string]any {
	t.Helper()
	groups, ok := widget.Config["groups"].(map[string]any)
	require.True(t, ok)
	return groups
}

func requireTimelineGroup(t *testing.T, groups map[string]any, key string) map[string]any {
	t.Helper()
	group, ok := groups[key].(map[string]any)
	require.True(t, ok)
	return group
}

func requireTimelineGroupConfig(t *testing.T, group map[string]any) map[string]any {
	t.Helper()
	config, ok := group["config"].(map[string]any)
	require.True(t, ok)
	return config
}

func writeTimelineComponentFixture(t *testing.T, runRoot string, component timelineComponentFixture) {
	t.Helper()
	absPath := filepath.Join(runRoot, filepath.FromSlash(component.RelativePath))
	require.NoError(t, os.MkdirAll(filepath.Dir(absPath), 0o755))
	contents := component.Contents
	if contents == nil {
		contents = []byte("fixture")
	}
	require.NoError(t, os.WriteFile(absPath, contents, 0o644))
}

func newRunComponentPresenceModel(
	t *testing.T,
	runRoot string,
	components []timelineComponentFixture,
) cdf.ModelView {
	t.Helper()

	manifestEntries := make([]cdf.ManifestEntry, 0, len(components))
	for _, component := range components {
		writeTimelineComponentFixture(t, runRoot, component)

		manifestEntries = append(manifestEntries, cdf.ManifestEntry{
			Path:          component.RelativePath,
			ComponentType: component.ComponentType,
		})
	}

	return cdf.NewOnDiskModel(runRoot, &cdf.Manifest{Entries: manifestEntries}, cdf.Metadata{})
}

func resolveTimelineGroupQuery(
	t *testing.T,
	session render.Session,
	renderers render.RendererList,
	timelineWidget *recipe.WidgetConfig,
	groupKey string,
) string {
	t.Helper()

	groups, ok := timelineWidget.Config["groups"].(map[string]any)
	require.True(t, ok)
	group, ok := groups[groupKey].(map[string]any)
	require.True(t, ok)
	config, ok := group["config"].(map[string]any)
	require.True(t, ok)
	customQuery, ok := config["customQuery"].(map[string]any)
	require.True(t, ok)

	timelineConfigJSON, err := json.Marshal(timelineWidget.Config)
	require.NoError(t, err)

	parsedDataSources, err := render.ParseDataSourcesFromConfig(string(timelineConfigJSON))
	require.NoError(t, err)

	resolvedDataSources, err := render.ResolveDataSources(session, parsedDataSources, renderers)
	require.NoError(t, err)

	groupTables, ok := resolvedDataSources[groupKey]
	require.True(t, ok)
	require.Len(t, groupTables, 1)

	return strings.ReplaceAll(
		customQuery["query"].(string),
		customQuery["tableNamePlaceholder"].(string),
		fmt.Sprintf(`"%s"`, groupTables[0].Name),
	)
}
