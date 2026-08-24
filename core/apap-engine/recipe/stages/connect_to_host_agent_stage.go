// Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
//
// SPDX-License-Identifier: Apache-2.0

package stages

import (
	"errors"
	"path/filepath"

	"github.com/Arm-Debug/apap-cli/apap-engine/agent"
	"github.com/Arm-Debug/apap-cli/apap-engine/conductor"
	"github.com/Arm-Debug/apap-cli/apap-engine/deploymentsupport"
	"github.com/Arm-Debug/apap-cli/apap-engine/message"
	"github.com/Arm-Debug/apap-cli/apap-engine/recipe"
	"github.com/Arm-Debug/apap-cli/apap-engine/run"
	"github.com/Arm-Debug/apap-cli/apap-engine/terminology"
	"github.com/Arm-Debug/apap-cli/apap-engine/tool"
	"github.com/Arm-Debug/apap-cli/apap-engine/tool/deployer"
	"github.com/Arm-Debug/apap-cli/atperf-version/versions"
)

// ConnectingToHostAgentStage connects to the host agent only when the resolved
// recipe deployment contains host-locality tool bundles.
type ConnectingToHostAgentStage struct {
	ToolBundlesSupplier      recipe.ToolBundlesSupplier
	TargetPlatformSupplier   recipe.PlatformConfigurationSupplier
	TargetFilesystemSupplier recipe.TargetFilesystemSupplier
	TargetSessionSupplier    recipe.TargetSessionSupplier
	HostPlatformSupplier     recipe.PlatformConfigurationSupplier
	HostFilesystemSupplier   recipe.TargetFilesystemSupplier
	HostSessionSupplier      recipe.TargetSessionSupplier
	Agent                    *agent.AgentConn
}

func NewConnectingToHostAgentStage(
	toolBundlesSupplier recipe.ToolBundlesSupplier,
	targetPlatformSupplier recipe.PlatformConfigurationSupplier,
	targetFilesystemSupplier recipe.TargetFilesystemSupplier,
	targetSessionSupplier recipe.TargetSessionSupplier,
	hostPlatformSupplier recipe.PlatformConfigurationSupplier,
	hostFilesystemSupplier recipe.TargetFilesystemSupplier,
	hostSessionSupplier recipe.TargetSessionSupplier,
) *ConnectingToHostAgentStage {
	return &ConnectingToHostAgentStage{
		ToolBundlesSupplier:      toolBundlesSupplier,
		TargetPlatformSupplier:   targetPlatformSupplier,
		TargetFilesystemSupplier: targetFilesystemSupplier,
		TargetSessionSupplier:    targetSessionSupplier,
		HostPlatformSupplier:     hostPlatformSupplier,
		HostFilesystemSupplier:   hostFilesystemSupplier,
		HostSessionSupplier:      hostSessionSupplier,
	}
}

func (s *ConnectingToHostAgentStage) Name() string {
	return "Connecting to host agent"
}

func (s *ConnectingToHostAgentStage) ErrorType() run.RunResult {
	return run.RecipeFailureConnectAgent
}

func (s *ConnectingToHostAgentStage) AlwaysExecute() bool {
	return false
}

func (s *ConnectingToHostAgentStage) Execute(ctx *recipe.StageContext) (func(), error) {
	if !deploymentsupport.ContainsHostToolBundles(s.ToolBundlesSupplier()) {
		return nil, nil
	}

	hostSession := s.HostSessionSupplier()
	hostAgent, err := hostSession.TargetAgent(ctx.Context)
	if err != nil {
		hostConnectionErr := message.New(message.EngineRecipeStagesHostAgentConnectionFailed).WithCause(err)
		var missingToolAdvice []recipe.ReadyAdvice
		if errors.Is(err, message.New(message.EngineAgentConnectionCreatorHostAgentNotDeployed)) {
			toolsRoot := hostSession.ResolveToolsDir()
			hostConnectionErr = newHostAgentDeploymentError(
				err,
				hostConnectionErr,
				func() string { return toolsRoot },
			)
			// This error stops the remaining readiness stages, including the
			// target and host tool probes, so report their missing bundles here.
			missingToolAdvice = append(
				s.missingToolAdvice(
					deploymentsupport.DeploymentLocalityTarget,
					s.TargetSessionSupplier().ResolveToolsDir(),
					s.TargetPlatformSupplier(),
					s.TargetFilesystemSupplier(),
				),
				s.missingToolAdvice(
					deploymentsupport.DeploymentLocalityHost,
					toolsRoot,
					s.HostPlatformSupplier(),
					s.HostFilesystemSupplier(),
				)...,
			)
		}
		notifyAgentReadinessFailure(
			ctx,
			hostConnectionErr,
			terminology.GetAgentBinaryName(),
			missingToolAdvice...,
		)
		return nil, hostConnectionErr
	}
	s.Agent = hostAgent
	return nil, nil
}

func newHostAgentDeploymentError(
	err error,
	hostConnectionErr error,
	hostToolsRootSupplier func() string,
) *message.MessageImpl {
	deployPath := ""
	if hostAgentNotDeployed := message.IsMessage(err); hostAgentNotDeployed != nil {
		deployPath = hostAgentNotDeployed.Metadata()["workingDir"]
	}
	if deployPath == "" {
		deployPath = filepath.ToSlash(filepath.Join(
			hostToolsRootSupplier(),
			terminology.GetAgentBinaryName(),
			versions.GetVersion(),
		))
	}
	return message.New(message.ToolIntegrationsCommonToolNotDeployed).
		WithMetadata(map[string]string{
			"tool":       terminology.GetAgentBinaryName(),
			"deployPath": deployPath,
			"locality":   string(deploymentsupport.DeploymentLocalityHost),
		}).
		WithCause(hostConnectionErr)
}

func (s *ConnectingToHostAgentStage) missingToolAdvice(
	locality deploymentsupport.DeploymentLocality,
	toolsRoot string,
	platform conductor.PlatformConfiguration,
	filesystem conductor.TargetFilesystem,
) []recipe.ReadyAdvice {
	agentTool := tool.ToolInfo{
		Name:    terminology.GetAgentBinaryName(),
		Version: versions.GetVersion(),
	}

	var advice []recipe.ReadyAdvice
	for _, toolInfo := range requiredToolsForLocality(
		s.ToolBundlesSupplier(),
		locality,
	) {
		if toolInfo == agentTool {
			continue
		}
		if deployer.GetToolPath(
			toolInfo,
			platform,
			deployer.BaseToolDeploymentPaths{DeployedToolsDirectory: toolsRoot},
			filesystem,
		) != "" {
			continue
		}
		advice = append(advice, recipe.ReadyAdvice{
			ToolName:       toolInfo.Name,
			AdviceSeverity: recipe.AdviceSeverityError,
			AdviceMessage: message.New(message.ToolIntegrationsCommonToolNotDeployed).
				WithMetadata(map[string]string{
					"tool":       toolInfo.Name,
					"deployPath": filepath.ToSlash(filepath.Join(toolsRoot, toolInfo.Name, toolInfo.Version)),
					"locality":   string(locality),
				}),
		})
	}
	return advice
}
