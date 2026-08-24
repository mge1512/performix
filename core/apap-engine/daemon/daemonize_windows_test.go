// SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
// SPDX-License-Identifier: Apache-2.0

package daemon

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestIsolateDaemonProcess(t *testing.T) {
	cmd := exec.Command("unused")

	isolateDaemonProcess(cmd)

	assert.Equal(t, uint32(syscall.CREATE_NEW_PROCESS_GROUP), cmd.SysProcAttr.CreationFlags)
}

func TestStart(t *testing.T) {
	daemon := NewDaemon()

	t.Run("Does not block", func(t *testing.T) {
		start := func() {
			_, err := daemon.Start("powershell", []string{"-command", "Start-Sleep", "-s", "1"})
			assert.NoError(t, err)
		}
		// Arbitrarily big timeout for the sake of CI. Locally, this executes much faster.
		// Timeout should be between daemon.WaitForExitStatusDuration and -s defined above.
		assertExecutedInLessThan(t, start, 750*time.Millisecond)
	})

	t.Run("Returns spawned commands process pid", func(t *testing.T) {
		pid, err := daemon.Start("powershell", []string{"-command", "Start-Sleep", "-s", "1"})
		assert.NoError(t, err)
		assert.NotZero(t, pid) // Best effort check for pid value
	})

	t.Run("Starts command in the background", func(t *testing.T) {
		// Use a long sleep time to minimise the chance of the pid being recycled before kill
		pid, err := daemon.Start("powershell", []string{"-command", "Start-Sleep", "-s", "30"})
		assert.NoError(t, err)
		err = daemon.killPid(pid)
		assert.NoError(t, err)
	})

	t.Run("Returns error when command exists with nonzero status code", func(t *testing.T) {
		_, err := daemon.Start("not-a-command", []string{})
		assert.Error(t, err)
	})

	t.Run("KillAndWait returns after command exits successfully during startup check", func(t *testing.T) {
		process, err := daemon.StartProcess("powershell", []string{"-command", "exit 0"})
		assert.NoError(t, err)

		assertExecutedInLessThan(t, func() {
			assert.NoError(t, process.KillAndWait())
		}, 750*time.Millisecond)
	})

	t.Run("KillAndWait treats successfully killed process as successful cleanup", func(t *testing.T) {
		process, err := daemon.StartProcess("powershell", []string{"-command", "Start-Sleep", "-s", "10"})
		assert.NoError(t, err)

		assert.NoError(t, process.KillAndWait())
	})

	t.Run("Logs a debug message when command was run succesfully", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		restoreLogOutputAndLevel := setLogOutputAndLevel(buffer, log.DebugLevel)
		defer restoreLogOutputAndLevel()

		pid, err := daemon.Start("powershell", []string{"-command", "Start-Sleep", "-s", "1"})
		assert.NoError(t, err)

		got := buffer.String()
		assert.Contains(t, got, "Daemon process started")
		assert.Contains(t, got, fmt.Sprintf("%d", pid))
	})

}

func TestStartAttachedForwardsChildStderr(t *testing.T) {
	readStderr, writeStderr, err := os.Pipe()
	assert.NoError(t, err)
	defer readStderr.Close()

	originalStderr := os.Stderr
	os.Stderr = writeStderr
	defer func() { os.Stderr = originalStderr }()

	daemon := NewDaemon()
	_, err = daemon.StartAttached("powershell", []string{
		"-command",
		"[Console]::Error.Write('child-stderr')",
	})
	assert.NoError(t, err)

	os.Stderr = originalStderr
	assert.NoError(t, writeStderr.Close())
	output, err := io.ReadAll(readStderr)
	assert.NoError(t, err)
	assert.Equal(t, "child-stderr", string(output))
}

func TestLogAttachedProcessExitStatus(t *testing.T) {
	buffer := &bytes.Buffer{}
	restoreLogOutputAndLevel := setLogOutputAndLevel(buffer, log.ErrorLevel)
	defer restoreLogOutputAndLevel()

	cmd := exec.Command("powershell", "-command", "exit 7")
	err := cmd.Run()
	assert.Error(t, err)

	logAttachedProcessExit(completedStartedProcess(cmd, err))

	got := buffer.String()
	assert.Contains(t, got, "Attached daemon process exited with an error")
	assert.Contains(t, got, fmt.Sprintf("PID=%d", cmd.Process.Pid))
	assert.Contains(t, got, "exit-code=7")
}

func TestLogAttachedProcessSuccessfulExit(t *testing.T) {
	buffer := &bytes.Buffer{}
	restoreLogOutputAndLevel := setLogOutputAndLevel(buffer, log.DebugLevel)
	defer restoreLogOutputAndLevel()

	cmd := exec.Command("powershell", "-command", "exit 0")
	err := cmd.Run()
	assert.NoError(t, err)

	logAttachedProcessExit(completedStartedProcess(cmd, err))

	got := buffer.String()
	assert.Contains(t, got, "Attached daemon process exited")
	assert.Contains(t, got, "exit-code=0")
	assert.NotContains(t, got, "error")
}
