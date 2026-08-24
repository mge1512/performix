# SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
# SPDX-License-Identifier: Apache-2.0

"""Robot keywords for exercising an MCP server over stdio."""

import asyncio
import os
from pathlib import Path
import platform
import signal
import tempfile
import time

from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client
import psutil
from robot.api.deco import keyword, library


@library(scope="TEST", auto_keywords=False)
class MCPClient:
    """Call MCP tools through a real ``apx`` process."""

    @keyword("Run MCP And Verify Engine Lifecycle")
    def run_mcp_and_verify_engine_lifecycle(
        self,
        apx_binary: str,
        tool_name: str | None = None,
        interrupt_cleanup: bool = False,
    ):
        """Run MCP and verify that its engine stops with the session."""
        with tempfile.TemporaryDirectory(prefix="apx-mcp-robot-") as directory:
            test_root = Path(directory)
            environment, state_directory = self._isolated_environment(test_root)
            stderr_path = test_root / "mcp-stderr.log"

            try:
                with stderr_path.open("w", encoding="utf-8") as stderr:
                    return asyncio.run(
                        self._run_mcp(
                            apx_binary,
                            tool_name,
                            environment,
                            state_directory,
                            stderr,
                            interrupt_cleanup,
                        )
                    )
            except Exception as error:
                stderr = stderr_path.read_text(encoding="utf-8")
                raise AssertionError(f"{error}\nMCP stderr:\n{stderr}") from error

    async def _run_mcp(
        self,
        apx_binary: str,
        tool_name: str | None,
        environment: dict,
        state_directory: Path,
        stderr,
        interrupt_cleanup: bool,
    ):
        parameters = StdioServerParameters(
            command=apx_binary,
            args=["mcp", "start"],
            env=environment,
        )
        engine_process = None
        try:
            async with stdio_client(parameters, errlog=stderr) as (read, write):
                async with ClientSession(read, write) as session:
                    await session.initialize()
                    pid_files = await self._wait_for_pid_file_count(
                        state_directory, 1
                    )
                    if pid_files[0].name.endswith("_9000.pid"):
                        raise AssertionError(
                            f"MCP engine used the default daemon PID file: {pid_files[0]}"
                        )
                    engine_process = self._running_engine_process(pid_files[0])
                    if interrupt_cleanup:
                        mcp_process = self._running_mcp_process(apx_binary)
                        # Stop MCP before signalling its group so it cannot run
                        # its deferred engine shutdown. The engine must stop
                        # without help from MCP.
                        os.kill(mcp_process.pid, signal.SIGSTOP)
                        os.killpg(os.getpgid(mcp_process.pid), signal.SIGTERM)
                        os.kill(mcp_process.pid, signal.SIGKILL)
                        result = True
                    else:
                        if tool_name is None:
                            raise AssertionError("MCP tool name is required")
                        result = await session.call_tool(tool_name, arguments={})
                        if result.isError:
                            raise AssertionError(
                                f"MCP tool {tool_name} returned an error"
                            )

            await self._wait_for_process_exit(engine_process)
            await self._wait_for_pid_file_count(state_directory, 0)
            if interrupt_cleanup:
                return result
            return result.structuredContent
        finally:
            if engine_process is not None and engine_process.is_running():
                engine_process.terminate()
                try:
                    engine_process.wait(timeout=5)
                except psutil.TimeoutExpired:
                    engine_process.kill()

    @staticmethod
    def _running_mcp_process(apx_binary: str):
        expected_binary = Path(apx_binary).resolve()
        for process in psutil.Process().children():
            try:
                command = process.cmdline()
            except psutil.Error:
                continue
            if (
                len(command) >= 3
                and Path(command[0]).resolve() == expected_binary
                and command[1:3] == ["mcp", "start"]
            ):
                return process
        raise AssertionError("No running MCP process was found")

    @staticmethod
    def _isolated_environment(test_root: Path):
        state_home = test_root / "state"
        config_home = test_root / "config"
        data_home = test_root / "data"
        for directory in (state_home, config_home, data_home):
            directory.mkdir(parents=True)

        environment = os.environ.copy()
        environment.update(
            {
                "HOME": str(test_root),
                "USERPROFILE": str(test_root),
                "XDG_STATE_HOME": str(state_home),
                "APXD_CONFIG_DIR": str(config_home),
                "APXD_DATA_DIR": str(data_home),
                "APXD_LOG_FILE": str(state_home / "apxd.log"),
            }
        )

        if platform.system() == "Windows":
            state_directory = test_root / "AppData" / "Local" / "apxd"
        else:
            state_directory = state_home / "apxd"
        return environment, state_directory

    @staticmethod
    def _running_engine_process(pid_file: Path):
        try:
            pid = int(pid_file.read_text(encoding="utf-8"))
            process = psutil.Process(pid)
            command = process.cmdline()
        except (OSError, ValueError, psutil.Error) as error:
            raise AssertionError(
                f"No running process was found for MCP engine PID file {pid_file}"
            ) from error

        if not process.is_running():
            raise AssertionError(f"MCP engine process {pid} is not running")
        if command[1:4] != ["daemon", "start", "--block"]:
            raise AssertionError(
                f"MCP engine PID {pid} has unexpected command line: {command}"
            )
        return process

    @staticmethod
    async def _wait_for_process_exit(process: psutil.Process):
        deadline = time.monotonic() + 5
        while time.monotonic() < deadline:
            if not process.is_running():
                return
            await asyncio.sleep(0.05)
        raise AssertionError(f"MCP engine process {process.pid} is still running")

    @staticmethod
    async def _wait_for_pid_file_count(state_directory: Path, expected: int):
        deadline = time.monotonic() + 5
        while time.monotonic() < deadline:
            pid_files = sorted(state_directory.glob("*.pid"))
            if len(pid_files) == expected:
                return pid_files
            await asyncio.sleep(0.05)
        pid_files = sorted(state_directory.glob("*.pid"))
        raise AssertionError(
            f"Expected {expected} MCP engine PID files, found {pid_files}"
        )
