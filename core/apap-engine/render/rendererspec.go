// SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Arm-Debug/apap-cli/apap-engine/message"
	"github.com/Arm-Debug/apap-cli/apap-engine/util"
)

type RendererSpec struct {
	Graph        Digraph
	Configs      RendererConfigList
	Dependencies []Dependencies
}

type RendererDependency struct {
	RendererID string `json:"renderer_id"`
}

type Dependencies struct {
	Tables    DataSourcesMap
	Renderers []RendererDependency
}

func NewRendererSpec(configs RendererConfigList, dependencies []Dependencies) (RendererSpec, []error, error) {
	nodes := make(NodeSet, len(configs))
	errs := make([]error, len(configs))
	successors := map[NodeID]NodeSet{}
	for i := range configs {
		nodes[NodeID(i)] = struct{}{}
		errs[i] = addSuccessorsFromDependencies(successors, dependencies[i], configs, i)
	}

	digraph, err := NewDigraph(nodes, successors)
	if err != nil {
		return RendererSpec{}, nil, err
	}
	propagationErrs, err := propagateCycleErrs(digraph, configs, errs)
	if err != nil {
		return RendererSpec{}, nil, err
	}
	errs = merge(errs, propagationErrs)

	spec := RendererSpec{
		Graph:        digraph,
		Configs:      configs,
		Dependencies: dependencies,
	}
	return spec, errs, nil
}

func addSuccessorsFromDependencies(successors map[NodeID]NodeSet, dependencies Dependencies, configs RendererConfigList, dependeeIndex int) error {
	var errs []error
	if err := addSuccessorFromTableDependencies(successors, dependencies.Tables, configs, dependeeIndex); err != nil {
		errs = append(errs, err)
	}
	if err := addSuccessorsFromRendererDependencies(successors, dependencies.Renderers, configs, dependeeIndex); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func addSuccessorFromTableDependencies(successors map[NodeID]NodeSet, dataSources map[string][]DataSource, configs RendererConfigList, dependeeIndex int) error {
	if dataSources == nil {
		return nil
	}
	var err error
	for _, sources := range dataSources {
		for _, ds := range sources {
			otr, ok := ds.(*OutputTableRef)
			if !ok {
				continue
			}

			if successorErr := addSuccessor(successors, otr.RendererID, configs, dependeeIndex); successorErr != nil && err == nil {
				err = successorErr
			}
		}
	}

	return err
}

func addSuccessorsFromRendererDependencies(successors map[NodeID]NodeSet, dependencies []RendererDependency, configs RendererConfigList, dependeeIndex int) error {
	var err error
	for _, dependency := range dependencies {
		if successorErr := addSuccessor(successors, dependency.RendererID, configs, dependeeIndex); successorErr != nil && err == nil {
			err = successorErr
		}
	}
	return err
}

func addSuccessor(successors map[NodeID]NodeSet, rendererID string, configs RendererConfigList, dependeeIndex int) error {
	dependencyIndex := rendererIndexFromID(rendererID, configs)
	if dependencyIndex == -1 {
		return fmt.Errorf("unknown renderer ID %v", rendererID)
	}

	dependency := successors[NodeID(dependencyIndex)]
	if dependency == nil {
		successors[NodeID(dependencyIndex)] = NodeSet{NodeID(dependeeIndex): {}}
		return nil
	}
	dependency[NodeID(dependeeIndex)] = struct{}{}
	return nil
}

func rendererIndexFromID(id string, configs RendererConfigList) int {
	for i, conf := range configs {
		if conf.ID != nil && *conf.ID == id {
			return i
		}
	}

	return -1
}

// propagateCycleErrs identifies any cycles in the provided digraph, and propagates an appropriate
// error to all renderers corresponding to nodes which are elements of a cycle
func propagateCycleErrs(digraph Digraph, configs RendererConfigList, configErrs []error) ([]error, error) {
	propagationErrs := make([]error, len(configErrs))
	currentGraph := digraph
	cycleFound, cycle := currentGraph.ContainsCycle()
	// While this graph contains a cycle, remove all nodes in that cycle from the graph copy
	// and recheck for cycles
	for cycleFound {
		stringCycle := util.Map(cycle, func(i NodeID) string {
			return fmt.Sprintf("%v (id: `%v`)", configs[i].Name, configs[i].GetDisplayID())
		})
		metadata := map[string]string{"cycle": strings.Join(stringCycle, " -> ")}
		toRemove := NodeSet{}
		for _, rendererIndex := range cycle {
			if configErrs[rendererIndex] == nil {
				metadata["type"] = configs[rendererIndex].Name
				metadata["id"] = configs[rendererIndex].GetDisplayID()
				propagationErrs[rendererIndex] = message.New(message.EngineRenderRendererspecRendererDependencyCycle).WithMetadata(metadata)
			}

			toRemove[rendererIndex] = struct{}{}
		}
		// Remove all nodes in this cycle and recompute
		updatedGraph, err := currentGraph.RemoveAllNodes(toRemove)
		if err != nil {
			return nil, err
		}
		currentGraph = updatedGraph
		cycleFound, cycle = currentGraph.ContainsCycle()
	}
	return propagationErrs, nil
}

func merge(existing []error, new []error) []error {
	lens := []int{len(existing), len(new)}
	slices.Sort(lens)

	// Make a new slice of the longer of the two lengths
	mergedErrs := make([]error, lens[1])
	for i := range mergedErrs {
		if i < len(existing) && existing[i] != nil {
			mergedErrs[i] = existing[i]
			continue
		}
		if i < len(new) {
			mergedErrs[i] = new[i]
		}
	}

	return mergedErrs
}
