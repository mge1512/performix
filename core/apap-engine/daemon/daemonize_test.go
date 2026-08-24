// SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

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

	assert.Equal(t, &syscall.SysProcAttr{Setpgid: true}, cmd.SysProcAttr)
}

func TestStart(t *testing.T) {
	daemon := NewDaemon()

	t.Run("Does not block", func(t *testing.T) {
		start := func() {
			_, err := daemon.Start("/bin/sh", []string{"-c", "sleep 10"})
			assert.NoError(t, err)
		}
		// Arbitrarily big timeout for the sake of CI. Locally, this executes much faster.
		// Timeout should be between daemon.WaitForExitStatusDuration and sleep defined above.
		assertExecutedInLessThan(t, start, 750*time.Millisecond)
	})

	t.Run("Returns spawned commands process pid", func(t *testing.T) {
		pid, err := daemon.Start("/bin/sh", []string{"-c", "sleep 10"})
		assert.NoError(t, err)
		assert.NotZero(t, pid) // Best effort check for pid value
	})

	t.Run("Command keeps running in the background", func(t *testing.T) {
		pid, err := daemon.Start("/bin/sh", []string{"-c", "sleep 0.5"})
		assert.NoError(t, err)
		// Killing a dead process would fail https://stackoverflow.com/a/15210305
		process, _ := os.FindProcess(pid)
		err = process.Signal(syscall.Signal(0))
		assert.NoError(t, err)
	})

	t.Run("Returns error when command exits with nonzero status code", func(t *testing.T) {
		_, err := daemon.Start("/bin/sh", []string{"-c", "exit 1"})
		assert.Error(t, err)
	})

	t.Run("KillAndWait returns after command exits successfully during startup check", func(t *testing.T) {
		process, err := daemon.StartProcess("/bin/sh", []string{"-c", "exit 0"})
		assert.NoError(t, err)

		assertExecutedInLessThan(t, func() {
			assert.NoError(t, process.KillAndWait())
		}, 750*time.Millisecond)
	})

	t.Run("KillAndWait treats successfully killed process as successful cleanup", func(t *testing.T) {
		process, err := daemon.StartProcess("/bin/sh", []string{"-c", "sleep 10"})
		assert.NoError(t, err)

		assert.NoError(t, process.KillAndWait())
	})

	t.Run("Logs a debug message when command was run succesfully", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		restoreLogOutputAndLevel := setLogOutputAndLevel(buffer, log.DebugLevel)
		defer restoreLogOutputAndLevel()

		pid, err := daemon.Start("/bin/sh", []string{"-c", "sleep 10"})
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
	_, err = daemon.StartAttached("/bin/sh", []string{"-c", "printf child-stderr >&2"})
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

	cmd := exec.Command("/bin/sh", "-c", "exit 7")
	err := cmd.Run()
	assert.Error(t, err)

	logAttachedProcessExit(completedStartedProcess(cmd, err))

	got := buffer.String()
	assert.Contains(t, got, "Attached daemon process exited with an error")
	assert.Contains(t, got, fmt.Sprintf("PID=%d", cmd.Process.Pid))
	assert.Contains(t, got, "exit-code=7")
	assert.Contains(t, got, "error=\"exit status 7\"")
}

func TestLogAttachedProcessSuccessfulExit(t *testing.T) {
	buffer := &bytes.Buffer{}
	restoreLogOutputAndLevel := setLogOutputAndLevel(buffer, log.DebugLevel)
	defer restoreLogOutputAndLevel()

	cmd := exec.Command("/bin/sh", "-c", "exit 0")
	err := cmd.Run()
	assert.NoError(t, err)

	logAttachedProcessExit(completedStartedProcess(cmd, err))

	got := buffer.String()
	assert.Contains(t, got, "Attached daemon process exited")
	assert.Contains(t, got, "exit-code=0")
	assert.NotContains(t, got, "error")
}
