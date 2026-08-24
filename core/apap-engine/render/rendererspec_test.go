// SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Arm-Debug/apap-cli/apap-engine/message"
)

var (
	idA = "renderer-a"
	idB = "renderer-b"
	idC = "renderer-c"
	idD = "renderer-d"
	idE = "renderer-e"

	configs = RendererConfigList{
		{Name: "A", ID: &idA},
		{Name: "B", ID: &idB},
		{Name: "C", ID: &idC},
		{Name: "D", ID: &idD},
		{Name: "E", ID: &idE},
	}
)

func TestNewRendererSpec(t *testing.T) {
	t.Run("creates graph from data sources correctly", func(t *testing.T) {
		// Flip ordering of indexes 0 and 1 to test that order doesn't matter
		localConfigs := RendererConfigList{configs[1], configs[0], configs[2]}
		dependencies := []Dependencies{
			{
				Tables: DataSourcesMap{
					"tables": {
						&OutputTableRef{RendererID: idA, Output: "out-a", ContentIndex: 0},
					},
				},
			},
			{
				Tables: DataSourcesMap{
					"tables": {
						&TableRefSource{Name: "independent_table"},
					},
				},
			},
			{
				Tables: DataSourcesMap{
					"tables": {
						&OutputTableRef{RendererID: idA, Output: "out-a", ContentIndex: 0},
						&OutputTableRef{RendererID: idB, Output: "out-b", ContentIndex: 0},
					},
				},
			},
		}

		spec, errs, err := NewRendererSpec(localConfigs, dependencies)
		assert.NoError(t, err)

		expectedGraph := expectedRendererSuccessorGraph(t, localConfigs, map[string][]*string{
			idA: {localConfigs[0].ID, localConfigs[2].ID},
			idB: {localConfigs[2].ID},
		})

		assert.Equal(t, expectedGraph, spec.Graph)
		assert.Equal(t, localConfigs, spec.Configs)
		assert.Equal(t, dependencies, spec.Dependencies)
		assert.Equal(t, make([]error, len(localConfigs)), errs)
	})
	t.Run("handles invalid renderer id referenced in data source", func(t *testing.T) {
		localConfigs := RendererConfigList{configs[0], configs[1], configs[2], configs[3]}
		dependencies := []Dependencies{
			{
				Tables: DataSourcesMap{
					"tables": {
						&TableRefSource{Name: "independent_table"},
					},
				},
			},
			{
				Tables: DataSourcesMap{
					"tables": {
						&OutputTableRef{RendererID: idA, Output: "out-a", ContentIndex: 0},
					},
				},
			},
			{
				Tables: DataSourcesMap{
					"tables": {
						&OutputTableRef{RendererID: "missing", Output: "out-a", ContentIndex: 0},
						&OutputTableRef{RendererID: idB, Output: "out-b", ContentIndex: 0},
					},
				},
			},
			{
				Tables: DataSourcesMap{
					"tables": {
						&OutputTableRef{RendererID: idA, Output: "out-a", ContentIndex: 0},
						&OutputTableRef{RendererID: idC, Output: "out-c", ContentIndex: 0},
					},
				},
			},
		}

		spec, errs, err := NewRendererSpec(localConfigs, dependencies)
		assert.NoError(t, err)

		// There are 4 links in the graph as that's how many valid inter-renderer dependencies we have
		// (given that "missing" doesn't exist).
		expectedGraph := expectedRendererSuccessorGraph(t, localConfigs, map[string][]*string{
			idA: {localConfigs[1].ID, localConfigs[3].ID},
			idB: {localConfigs[2].ID},
			idC: {localConfigs[3].ID},
		})
		assert.Equal(t, expectedGraph, spec.Graph)
		assert.Equal(t, localConfigs, spec.Configs)
		assert.Equal(t, dependencies, spec.Dependencies)
		// Configuration error should have been added for renderer with invalid data sources
		assert.Error(t, errs[2])
	})
	t.Run("creates graph from direct renderer dependencies correctly", func(t *testing.T) {
		localConfigs := RendererConfigList{configs[0], configs[1], configs[2]}
		dependencies := []Dependencies{
			{},
			{
				Renderers: []RendererDependency{{RendererID: idA}},
			},
			{
				Tables: DataSourcesMap{
					"tables": {
						&OutputTableRef{RendererID: idB, Output: "out-b", ContentIndex: 0},
					},
				},
				Renderers: []RendererDependency{{RendererID: idA}},
			},
		}

		spec, errs, err := NewRendererSpec(localConfigs, dependencies)
		assert.NoError(t, err)

		expectedGraph := expectedRendererSuccessorGraph(t, localConfigs, map[string][]*string{
			idA: {localConfigs[1].ID, localConfigs[2].ID},
			idB: {localConfigs[2].ID},
		})

		assert.Equal(t, expectedGraph, spec.Graph)
		assert.Equal(t, localConfigs, spec.Configs)
		assert.Equal(t, dependencies, spec.Dependencies)
		assert.Equal(t, make([]error, len(localConfigs)), errs)
	})
	t.Run("handles invalid renderer id referenced in direct renderer dependency", func(t *testing.T) {
		localConfigs := RendererConfigList{configs[0], configs[1]}
		dependencies := []Dependencies{
			{},
			{
				Renderers: []RendererDependency{{RendererID: "missing"}},
			},
		}

		spec, errs, err := NewRendererSpec(localConfigs, dependencies)
		assert.NoError(t, err)

		expectedGraph := expectedRendererSuccessorGraph(t, localConfigs, map[string][]*string{})

		assert.Equal(t, expectedGraph, spec.Graph)
		assert.Equal(t, localConfigs, spec.Configs)
		assert.Equal(t, dependencies, spec.Dependencies)
		assert.NoError(t, errs[0])
		assert.Error(t, errs[1])
	})
	t.Run("sets config errs for all renderers that form part of a cycle", func(t *testing.T) {
		localConfigs := RendererConfigList{configs[0], configs[1], configs[2], configs[3], configs[4]}
		// A -> B -> C -> D -> B, E -> E (2 discrete cycles)
		dependencies := []Dependencies{
			{
				Tables: DataSourcesMap{
					"tables": {},
				},
			},
			{
				Tables: DataSourcesMap{
					"tables": {
						&OutputTableRef{RendererID: idA, Output: "out-a", ContentIndex: 0},
					},
				},
				Renderers: []RendererDependency{{RendererID: idD}},
			},
			{
				Tables: DataSourcesMap{
					"tables": {
						&OutputTableRef{RendererID: idB, Output: "out-b", ContentIndex: 0},
					},
				},
			},
			{
				Renderers: []RendererDependency{{RendererID: idC}},
			},
			{
				Renderers: []RendererDependency{{RendererID: idE}},
			},
		}

		spec, errs, err := NewRendererSpec(localConfigs, dependencies)
		assert.NoError(t, err)

		expectedGraph := expectedRendererSuccessorGraph(t, localConfigs, map[string][]*string{
			idA: {localConfigs[1].ID},
			idB: {localConfigs[2].ID},
			idC: {localConfigs[3].ID},
			idD: {localConfigs[1].ID},
			idE: {localConfigs[4].ID},
		})

		expectedCycle1 := "B (id: `renderer-b`) -> C (id: `renderer-c`) -> D (id: `renderer-d`) -> B (id: `renderer-b`)"
		expectedCycle2 := "E (id: `renderer-e`) -> E (id: `renderer-e`)"
		expectedErrB := message.New(message.EngineRenderRendererspecRendererDependencyCycle).WithMetadata(map[string]string{
			"type":  "B",
			"id":    "renderer-b",
			"cycle": expectedCycle1,
		})
		expectedErrC := message.New(message.EngineRenderRendererspecRendererDependencyCycle).WithMetadata(map[string]string{
			"type":  "C",
			"id":    "renderer-c",
			"cycle": expectedCycle1,
		})
		expectedErrD := message.New(message.EngineRenderRendererspecRendererDependencyCycle).WithMetadata(map[string]string{
			"type":  "D",
			"id":    "renderer-d",
			"cycle": expectedCycle1,
		})
		expectedErrE := message.New(message.EngineRenderRendererspecRendererDependencyCycle).WithMetadata(map[string]string{
			"type":  "E",
			"id":    "renderer-e",
			"cycle": expectedCycle2,
		})

		assert.Equal(t, expectedGraph, spec.Graph)
		assert.Equal(t, localConfigs, spec.Configs)
		assert.Equal(t, dependencies, spec.Dependencies)
		assert.NoError(t, errs[0])
		assert.Equal(t, expectedErrB, errs[1])
		assert.Equal(t, expectedErrC, errs[2])
		assert.Equal(t, expectedErrD, errs[3])
		assert.Equal(t, expectedErrE, errs[4])
		assert.NoError(t, message.ValidateMetadataPlaceholders(errs[1]))
		assert.NoError(t, message.ValidateMetadataPlaceholders(errs[2]))
		assert.NoError(t, message.ValidateMetadataPlaceholders(errs[3]))
		assert.NoError(t, message.ValidateMetadataPlaceholders(errs[4]))
	})
}

func TestMerge(t *testing.T) {
	existing0 := errors.New("existing 0")
	existing1 := errors.New("existing 1")
	existing2 := errors.New("existing 2")
	new0 := errors.New("new 0")
	new1 := errors.New("new 1")
	new2 := errors.New("new 2")

	tests := []struct {
		name     string
		existing []error
		new      []error
		expected []error
	}{
		{
			name:     "keeps existing errors and fills gaps from new errors",
			existing: []error{existing0, nil, existing2, nil},
			new:      []error{new0, new1, new2, nil},
			expected: []error{existing0, new1, existing2, nil},
		},
		{
			name:     "preserves trailing errors from longer existing slice",
			existing: []error{nil, existing1, existing2},
			new:      []error{new0},
			expected: []error{new0, existing1, existing2},
		},
		{
			name:     "preserves trailing errors from longer new slice",
			existing: []error{existing0},
			new:      []error{new0, new1, new2},
			expected: []error{existing0, new1, new2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.expected, merge(tt.existing, tt.new))
		})
	}
}

func expectedRendererSuccessorGraph(t *testing.T, configs RendererConfigList, successorIDsByRendererID map[string][]*string) Digraph {
	t.Helper()

	nodes := make(NodeSet, len(configs))
	for i := range configs {
		nodes[NodeID(i)] = struct{}{}
	}

	successors := map[NodeID]NodeSet{}
	for rendererID, successorIDs := range successorIDsByRendererID {
		rendererIndex := requireRendererIndexByID(t, configs, rendererID)
		successors[rendererIndex] = NodeSet{}
		for _, successorID := range successorIDs {
			require.NotNil(t, successorID)
			successorIndex := requireRendererIndexByID(t, configs, *successorID)
			successors[rendererIndex][successorIndex] = struct{}{}
		}
	}

	return Digraph{
		nodes:      nodes,
		successors: successors,
	}
}

func requireRendererIndexByID(t *testing.T, configs RendererConfigList, id string) NodeID {
	t.Helper()
	for i, config := range configs {
		if config.ID != nil && *config.ID == id {
			return NodeID(i)
		}
	}

	require.Failf(t, "renderer ID not found", "renderer ID %q was not found in configs", id)
	return 0
}
