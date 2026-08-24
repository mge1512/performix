// SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
// SPDX-License-Identifier: Apache-2.0

package recipejstest

import (
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Arm-Debug/apap-cli/apap-engine/message"
	"github.com/Arm-Debug/apap-cli/apap-engine/tool"
)

type controlledProcessResult struct {
	exitCode int
	err      error
}

type controlledProcessHandle struct {
	pid            int
	completion     chan controlledProcessResult
	completeOne    sync.Once
	signalMu       sync.Mutex
	killCalls      int
	interruptCalls int
	killErr        error
	interruptErr   error
	onInterrupt    func()
	onKill         func()
	onWait         func()
}

func newControlledProcessHandle() *controlledProcessHandle {
	return &controlledProcessHandle{
		pid:        1234,
		completion: make(chan controlledProcessResult, 1),
	}
}

func (h *controlledProcessHandle) complete(exitCode int, err error) {
	h.completeOne.Do(func() {
		h.completion <- controlledProcessResult{exitCode: exitCode, err: err}
	})
}

func (h *controlledProcessHandle) PID() int { return h.pid }

func (h *controlledProcessHandle) Kill() error {
	h.signalMu.Lock()
	h.killCalls++
	h.signalMu.Unlock()
	if h.onKill != nil {
		h.onKill()
	}
	return h.killErr
}

func (h *controlledProcessHandle) Interrupt() error {
	h.signalMu.Lock()
	h.interruptCalls++
	h.signalMu.Unlock()
	if h.onInterrupt != nil {
		h.onInterrupt()
	}
	return h.interruptErr
}

func (h *controlledProcessHandle) signalCallCounts() (kill, interrupt int) {
	h.signalMu.Lock()
	defer h.signalMu.Unlock()
	return h.killCalls, h.interruptCalls
}

func (h *controlledProcessHandle) Wait() (int, error) {
	if h.onWait != nil {
		h.onWait()
	}
	result := <-h.completion
	return result.exitCode, result.err
}

func (h *controlledProcessHandle) Stdout() io.Reader { return nil }
func (h *controlledProcessHandle) Stderr() io.Reader { return nil }
func (h *controlledProcessHandle) WriteStdin(string) error {
	return nil
}

type eventRecorder struct {
	mu     sync.Mutex
	events []string
}

func (r *eventRecorder) record(event string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, event)
}

func (r *eventRecorder) snapshot() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.events...)
}

type workloadLifecycleOptions struct {
	workload           *controlledProcessHandle
	workloadStartError error
	timeout            uint32
}

type workloadLifecycleFactory func(
	t *testing.T,
	options workloadLifecycleOptions,
) tool.ToolIntegration

type workloadLifecycleSpec struct {
	workloadName   string
	newIntegration workloadLifecycleFactory
}

func validateToolWorkloadLifecycle(t *testing.T, spec workloadLifecycleSpec) {
	t.Helper()
	require.NotEmpty(t, spec.workloadName)
	require.NotNil(t, spec.newIntegration)

	t.Run("successful workload", func(t *testing.T) {
		workload := newControlledProcessHandle()
		workload.complete(0, nil)
		integration := spec.newIntegration(t, workloadLifecycleOptions{workload: workload})
		assert.True(t, integration.Properties().SupportsWorkloadLaunch)

		withToolRuntime(t, integration, func() {
			require.NoError(t, integration.Run())
		})
	})

	t.Run("workload start failure", func(t *testing.T) {
		startError := errors.New("workload unavailable")
		integration := spec.newIntegration(t, workloadLifecycleOptions{workloadStartError: startError})
		withToolRuntime(t, integration, func() {
			err := integration.Run()
			assertMessageError(t, err, "tool_integrations.common.WORKLOAD_START_FAILED", map[string]string{
				"workload": spec.workloadName,
				"reason":   startError.Error(),
			})
		})
	})

	t.Run("non-zero workload exit", func(t *testing.T) {
		workload := newControlledProcessHandle()
		workload.complete(23, nil)
		integration := spec.newIntegration(t, workloadLifecycleOptions{workload: workload})
		withToolRuntime(t, integration, func() {
			err := integration.Run()
			assertMessageError(t, err, "tool_integrations.common.WORKLOAD_RUNTIME_FAILED", map[string]string{
				"workload": spec.workloadName,
				"exitCode": "23",
				"reason":   "non_zero_exit",
			})
		})
	})

	t.Run("stop interrupts workload", func(t *testing.T) {
		workload := newControlledProcessHandle()
		waitStarted := make(chan struct{})
		events := &eventRecorder{}
		workload.onWait = func() { close(waitStarted) }
		workload.onInterrupt = func() {
			events.record("interrupt")
			workload.complete(0, nil)
		}
		integration := spec.newIntegration(t, workloadLifecycleOptions{workload: workload})

		withToolRuntime(t, integration, func() {
			runResult := runToolIntegrationAsync(integration)
			waitForProcessWait(t, waitStarted)
			require.NoError(t, integration.Stop())
			require.NoError(t, waitForRunResult(t, runResult))
		})
		assert.Contains(t, events.snapshot(), "interrupt")
	})

	t.Run("stop kills workload after interrupt grace period", func(t *testing.T) {
		workload := newControlledProcessHandle()
		waitStarted := make(chan struct{})
		events := &eventRecorder{}
		workload.onWait = func() { close(waitStarted) }
		workload.onInterrupt = func() { events.record("interrupt") }
		workload.onKill = func() {
			events.record("kill")
			workload.complete(0, nil)
		}
		integration := spec.newIntegration(t, workloadLifecycleOptions{workload: workload})

		withToolRuntime(t, integration, func() {
			runResult := runToolIntegrationAsync(integration)
			waitForProcessWait(t, waitStarted)
			require.NoError(t, integration.Stop())
			require.NoError(t, waitForRunResult(t, runResult))
		})
		assert.Equal(t, []string{"interrupt", "kill"}, events.snapshot())
	})

	t.Run("stop returns after kill without waiting for process exit", func(t *testing.T) {
		workload := newControlledProcessHandle()
		waitStarted := make(chan struct{})
		workload.onWait = func() { close(waitStarted) }
		integration := spec.newIntegration(t, workloadLifecycleOptions{workload: workload})

		withToolRuntime(t, integration, func() {
			runResult := runToolIntegrationAsync(integration)
			waitForProcessWait(t, waitStarted)
			stopResult := make(chan error, 1)
			go func() { stopResult <- integration.Stop() }()
			select {
			case err := <-stopResult:
				require.NoError(t, err)
			case <-time.After(2 * time.Second):
				t.Fatal("timed out waiting for tool integration to stop")
			}

			kills, interrupts := workload.signalCallCounts()
			assert.Equal(t, 1, interrupts)
			assert.Equal(t, 1, kills)

			workload.complete(0, nil)
			require.NoError(t, waitForRunResult(t, runResult))
		})
	})

	t.Run("cancel kills workload", func(t *testing.T) {
		workload := newControlledProcessHandle()
		waitStarted := make(chan struct{})
		events := &eventRecorder{}
		workload.onWait = func() { close(waitStarted) }
		workload.onKill = func() {
			events.record("kill")
			workload.complete(0, nil)
		}
		integration := spec.newIntegration(t, workloadLifecycleOptions{workload: workload})

		withToolRuntime(t, integration, func() {
			runResult := runToolIntegrationAsync(integration)
			waitForProcessWait(t, waitStarted)
			require.NoError(t, integration.Cancel())
			require.NoError(t, waitForRunResult(t, runResult))
		})
		assert.Contains(t, events.snapshot(), "kill")
	})

	t.Run("timeout interrupts workload", func(t *testing.T) {
		workload := newControlledProcessHandle()
		events := &eventRecorder{}
		workload.onInterrupt = func() {
			events.record("interrupt")
			workload.complete(0, nil)
		}
		integration := spec.newIntegration(t, workloadLifecycleOptions{
			workload: workload,
			timeout:  1,
		})

		withToolRuntime(t, integration, func() {
			require.NoError(t, integration.Run())
		})
		assert.Contains(t, events.snapshot(), "interrupt")
	})
}

func withToolRuntime(t *testing.T, integration tool.ToolIntegration, run func()) {
	t.Helper()
	cleanup, err := integration.StartRuntime()
	require.NoError(t, err)
	defer cleanup()
	run()
}

func runToolIntegrationAsync(integration tool.ToolIntegration) <-chan error {
	result := make(chan error, 1)
	go func() {
		result <- integration.Run()
	}()
	return result
}

func waitForProcessWait(t *testing.T, waitStarted <-chan struct{}) {
	t.Helper()
	select {
	case <-waitStarted:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for process wait to start")
	}
}

func waitForRunResult(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for tool integration to finish")
		return nil
	}
}

func assertMessageError(
	t *testing.T,
	err error,
	wantCode message.MessageCode,
	wantMetadata map[string]string,
) {
	t.Helper()
	var messageError *message.MessageImpl
	require.ErrorAs(t, err, &messageError)
	assert.Equal(t, wantCode, messageError.Code())
	for key, value := range wantMetadata {
		assert.Equal(t, value, messageError.Metadata()[key])
	}
}

var _ tool.ProcessHandle = (*controlledProcessHandle)(nil)
