// Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
//
// SPDX-License-Identifier: Apache-2.0

package stages

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/Arm-Debug/apap-cli/apap-engine/agent"
	"github.com/Arm-Debug/apap-cli/apap-engine/conductor"
	"github.com/Arm-Debug/apap-cli/apap-engine/deploymentsupport"
	"github.com/Arm-Debug/apap-cli/apap-engine/message"
	"github.com/Arm-Debug/apap-cli/apap-engine/recipe"
	"github.com/Arm-Debug/apap-cli/apap-engine/run"
	"github.com/Arm-Debug/apap-cli/apap-engine/targetsession"
	targetsessionmocks "github.com/Arm-Debug/apap-cli/apap-engine/targetsession/mocks"
	"github.com/Arm-Debug/apap-cli/apap-engine/terminology"
	"github.com/Arm-Debug/apap-cli/atperf-version/versions"
)

func TestConnectingToHostAgentStage(t *testing.T) {
	targetBundle := deploymentsupport.ToolBundleInfo{
		Name:     "sl-record",
		Version:  "2.1.0",
		Locality: deploymentsupport.DeploymentLocalityTarget,
	}
	hostBundle := deploymentsupport.ToolBundleInfo{
		Name:     "sl-analyze",
		Version:  "2.1.0",
		Locality: deploymentsupport.DeploymentLocalityHost,
	}
	hostBundles := func() []deploymentsupport.ToolBundleInfo {
		return []deploymentsupport.ToolBundleInfo{targetBundle, hostBundle}
	}

	t.Run("skips the host when no host bundles are resolved", func(t *testing.T) {
		stage := NewConnectingToHostAgentStage(
			func() []deploymentsupport.ToolBundleInfo {
				return []deploymentsupport.ToolBundleInfo{{Locality: deploymentsupport.DeploymentLocalityTarget}}
			},
			nil,
			nil,
			nil,
			nil,
			nil,
			func() targetsession.TargetSession { panic("host session should not be requested") },
		)

		cleanup, err := stage.Execute(&recipe.StageContext{Context: context.Background()})
		require.NoError(t, err)
		require.Nil(t, cleanup)
		require.Nil(t, stage.Agent)
	})

	t.Run("connects to the host agent", func(t *testing.T) {
		hostAgent := agent.NewAgentConn(nil)
		hostSession := &targetsessionmocks.MockTargetSession{}
		hostSession.On("TargetAgent", mock.Anything).Return(hostAgent, nil).Once()
		stage := NewConnectingToHostAgentStage(
			hostBundles,
			nil,
			nil,
			nil,
			nil,
			nil,
			func() targetsession.TargetSession { return hostSession },
		)

		cleanup, err := stage.Execute(&recipe.StageContext{Context: context.Background()})
		require.NoError(t, err)
		require.Nil(t, cleanup)
		require.Same(t, hostAgent, stage.Agent)
		hostSession.AssertExpectations(t)
	})

	t.Run("reports a missing host agent as a tool to deploy", func(t *testing.T) {
		connectionErr := message.New(message.EngineAgentConnectionCreatorHostAgentNotDeployed)
		hostSession := &targetsessionmocks.MockTargetSession{}
		hostSession.On("TargetAgent", mock.Anything).Return((*agent.AgentConn)(nil), connectionErr).Once()
		hostSession.On("ResolveToolsDir").Return("/host/tools").Once()
		hostPlatform := conductor.PlatformConfiguration{
			OS:           conductor.Darwin,
			Architecture: conductor.AArch64,
		}
		hostFilesystem := conductor.NewAferoTargetFilesystem(afero.NewMemMapFs())
		targetPlatform := conductor.PlatformConfiguration{
			OS:           conductor.Android,
			Architecture: conductor.AArch64,
		}
		targetFilesystem := conductor.NewAferoTargetFilesystem(afero.NewMemMapFs())
		targetSession := &targetsessionmocks.MockTargetSession{}
		targetSession.On("ResolveToolsDir").Return("/target/tools").Once()
		notifier := &MockReadinessNotifier{}
		var readinessOutput recipe.ReadyOutput
		notifier.On("OnReadinessProbed", mock.Anything).Run(func(args mock.Arguments) {
			readinessOutput = args.Get(0).(recipe.ReadyOutput)
		}).Once()
		stage := NewConnectingToHostAgentStage(
			hostBundles,
			func() conductor.PlatformConfiguration { return targetPlatform },
			func() conductor.TargetFilesystem { return targetFilesystem },
			func() targetsession.TargetSession { return targetSession },
			func() conductor.PlatformConfiguration { return hostPlatform },
			func() conductor.TargetFilesystem { return hostFilesystem },
			func() targetsession.TargetSession { return hostSession },
		)

		cleanup, err := stage.Execute(&recipe.StageContext{
			Context:           context.Background(),
			ReadinessNotifier: notifier,
		})
		require.ErrorIs(t, err, connectionErr)
		returnedMessage := message.IsMessage(err)
		require.NotNil(t, returnedMessage)
		require.Equal(t, message.ToolIntegrationsCommonToolNotDeployed, returnedMessage.Code())
		require.Equal(t, map[string]string{
			"tool": terminology.GetAgentBinaryName(),
			"deployPath": filepath.ToSlash(filepath.Join(
				"/host/tools",
				terminology.GetAgentBinaryName(),
				versions.GetVersion(),
			)),
			"locality": string(deploymentsupport.DeploymentLocalityHost),
		}, returnedMessage.Metadata())
		require.Nil(t, cleanup)
		require.Nil(t, stage.Agent)
		require.Len(t, readinessOutput.Advice, 3)
		require.Equal(t, terminology.GetAgentBinaryName(), readinessOutput.Advice[0].ToolName)
		require.Equal(t, message.ToolIntegrationsCommonToolNotDeployed, readinessOutput.Advice[0].AdviceMessage.Code())
		require.Equal(t, map[string]string{
			"tool": terminology.GetAgentBinaryName(),
			"deployPath": filepath.ToSlash(filepath.Join(
				"/host/tools",
				terminology.GetAgentBinaryName(),
				versions.GetVersion(),
			)),
			"locality": string(deploymentsupport.DeploymentLocalityHost),
		}, readinessOutput.Advice[0].AdviceMessage.Metadata())
		require.Equal(t, targetBundle.Name, readinessOutput.Advice[1].ToolName)
		require.Equal(t, message.ToolIntegrationsCommonToolNotDeployed, readinessOutput.Advice[1].AdviceMessage.Code())
		require.Equal(t, map[string]string{
			"tool":       targetBundle.Name,
			"deployPath": filepath.ToSlash(filepath.Join("/target/tools", targetBundle.Name, targetBundle.Version)),
			"locality":   string(deploymentsupport.DeploymentLocalityTarget),
		}, readinessOutput.Advice[1].AdviceMessage.Metadata())
		require.Equal(t, hostBundle.Name, readinessOutput.Advice[2].ToolName)
		require.Equal(t, message.ToolIntegrationsCommonToolNotDeployed, readinessOutput.Advice[2].AdviceMessage.Code())
		require.Equal(t, map[string]string{
			"tool":       hostBundle.Name,
			"deployPath": filepath.ToSlash(filepath.Join("/host/tools", hostBundle.Name, hostBundle.Version)),
			"locality":   string(deploymentsupport.DeploymentLocalityHost),
		}, readinessOutput.Advice[2].AdviceMessage.Metadata())
		targetSession.AssertExpectations(t)
		hostSession.AssertExpectations(t)
		notifier.AssertExpectations(t)
	})

	t.Run("reports a host agent connection failure", func(t *testing.T) {
		connectionErr := errors.New("host agent unavailable")
		hostSession := &targetsessionmocks.MockTargetSession{}
		hostSession.On("TargetAgent", mock.Anything).Return((*agent.AgentConn)(nil), connectionErr).Once()
		notifier := &MockReadinessNotifier{}
		var readinessOutput recipe.ReadyOutput
		notifier.On("OnReadinessProbed", mock.Anything).Run(func(args mock.Arguments) {
			readinessOutput = args.Get(0).(recipe.ReadyOutput)
		}).Once()
		stage := NewConnectingToHostAgentStage(
			hostBundles,
			nil,
			nil,
			nil,
			nil,
			nil,
			func() targetsession.TargetSession { return hostSession },
		)

		cleanup, err := stage.Execute(&recipe.StageContext{
			Context:           context.Background(),
			ReadinessNotifier: notifier,
		})
		require.ErrorIs(t, err, connectionErr)
		require.Nil(t, cleanup)
		require.Nil(t, stage.Agent)
		require.Len(t, readinessOutput.Advice, 1)
		require.Equal(t, terminology.GetAgentBinaryName(), readinessOutput.Advice[0].ToolName)
		require.Equal(t, message.EngineRecipeStagesHostAgentConnectionFailed, readinessOutput.Advice[0].AdviceMessage.Code())
		catalogMessage, lookupErr := message.LookupMessage(readinessOutput.Advice[0].AdviceMessage)
		require.NoError(t, lookupErr)
		require.Contains(t, catalogMessage.Message, "on the host")
		require.NotContains(t, catalogMessage.Message, "on the target")
		hostSession.AssertExpectations(t)
		notifier.AssertExpectations(t)
	})
}

func TestConnectingToHostAgentStageAPI(t *testing.T) {
	stage := &ConnectingToHostAgentStage{}
	require.Equal(t, "Connecting to host agent", stage.Name())
	require.Equal(t, run.RecipeFailureConnectAgent, stage.ErrorType())
	require.False(t, stage.AlwaysExecute())
}

func TestNewHostAgentDeploymentErrorUsesReportedWorkingDirectory(t *testing.T) {
	deployPath := "/host/tools/apx-agent/version"
	connectionErr := message.New(message.EngineAgentConnectionCreatorHostAgentNotDeployed).
		WithMetadata(map[string]string{"workingDir": deployPath})
	hostConnectionErr := message.New(message.EngineRecipeStagesHostAgentConnectionFailed).
		WithCause(connectionErr)

	deploymentErr := newHostAgentDeploymentError(
		connectionErr,
		hostConnectionErr,
		func() string { panic("tools root should not be resolved") },
	)

	require.Equal(t, message.ToolIntegrationsCommonToolNotDeployed, deploymentErr.Code())
	require.Equal(t, map[string]string{
		"tool":       terminology.GetAgentBinaryName(),
		"deployPath": deployPath,
		"locality":   string(deploymentsupport.DeploymentLocalityHost),
	}, deploymentErr.Metadata())
	require.ErrorIs(t, deploymentErr, connectionErr)
}
