# SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
# SPDX-License-Identifier: Apache-2.0

"""Immediately retry failed Robot Framework tests."""

import json
from pathlib import Path

from robot.api import ExecutionResult, logger, ResultWriter


class RetryFailed:
    """Retry failed tests in place while keeping one final Robot result."""

    ROBOT_LISTENER_API_VERSION = 3
    REPORT_NAME = "flaky-robot-report.json"
    WILL_RETRY_TAG = "performix:will-retry"

    def __init__(self, retry_count=1, fail_fast=False):
        self.retry_count = int(retry_count)
        if self.retry_count < 1:
            raise ValueError("retry_count must be at least 1")
        self.fail_fast = str(fail_fast).lower() in ("1", "true", "yes", "on")
        self._attempts = {}
        self._recovered = []
        self._stop = False
        self._rewriting_output = False
        self._fatal_failure = False

    def start_test(self, _data, result):
        """Skip tests encountered after an exhausted fail-fast retry."""
        self._fatal_failure = False
        if self._stop:
            result.tags.add("robot:skip")

    def end_keyword(self, data, result):
        """Recognize Robot's built-in fatal error keyword."""
        normalized_name = data.name.replace(" ", "").replace("_", "").lower()
        if normalized_name == "fatalerror" and result.status == "FAIL":
            self._fatal_failure = True

    def end_test(self, data, result):
        """Record the attempt and schedule an immediate retry when applicable."""
        test_id = data.id
        attempts = self._attempts.setdefault(test_id, [])
        attempts.append(self._attempt(result, len(attempts) + 1))

        automatic_failure = self._fatal_failure or self._is_automatic_failure(result)
        if result.status == "FAIL" and not automatic_failure:
            retries_used = len(attempts) - 1
            if retries_used < self.retry_count:
                attempt_number = len(attempts)
                result.tags.add(self.WILL_RETRY_TAG)
                self._insert_after_current_attempt(data, attempt_number)
                logger.console(
                    f"[RETRY] {result.longname}: retrying after attempt {attempt_number} failed"
                )
                return

        if result.status == "PASS" and len(attempts) > 1:
            result.message = (
                f"[RETRY] Passed on attempt {len(attempts)} after "
                f"{len(attempts) - 1} failed attempt(s)."
            )
            self._recovered.append(
                {
                    "title": result.name,
                    "longName": result.longname,
                    "file": str(result.source) if result.source else None,
                    "line": result.lineno,
                    "attempts": attempts,
                }
            )
        elif result.status == "FAIL" and len(attempts) > 1:
            result.message = (
                f"[RETRY] Failed after {len(attempts)} attempts.\n\n{result.message}"
            )
            self._stop_after_failure(data)
        elif result.status == "FAIL":
            self._stop_after_failure(data)
        elif result.status == "SKIP" and len(attempts) > 1:
            details = "\n\n".join(
                f"Attempt {attempt['attempt']} [{attempt['status'].upper()}]:\n"
                f"{attempt['message']}"
                for attempt in attempts
            )
            result.status = "FAIL"
            result.message = f"[RETRY] No attempt passed.\n\n{details}"
            self._stop_after_failure(data)

    def end_suite(self, data, result):
        """Remove intermediate attempts from Robot's in-memory result model."""
        self._keep_final_tests(data.tests)
        retained_results = [
            test for test in result.tests if self.WILL_RETRY_TAG not in test.tags
        ]
        result.tests.clear()
        result.tests.extend(retained_results)

    def output_file(self, path):
        """Normalize output.xml and write the recovered-flake report."""
        if self._rewriting_output:
            return
        if any(len(attempts) > 1 for attempts in self._attempts.values()):
            self._rewriting_output = True
            try:
                self._remove_retry_attempts_from_output(path)
            finally:
                self._rewriting_output = False
        report_path = Path(path).with_name(self.REPORT_NAME)
        if self._recovered:
            report_path.write_text(
                json.dumps({"testResults": self._recovered}, indent=2) + "\n",
                encoding="utf-8",
            )
        elif report_path.exists():
            report_path.unlink()

    @staticmethod
    def _attempt(result, attempt_number):
        return {
            "attempt": attempt_number,
            "status": result.status.lower(),
            "message": result.message,
            "elapsedMilliseconds": round(result.elapsed_time.total_seconds() * 1000),
        }

    def _insert_after_current_attempt(self, test, attempt_number):
        tests = test.parent.tests
        occurrences = [index for index, item in enumerate(tests) if item is test]
        current_index = occurrences[min(attempt_number - 1, len(occurrences) - 1)]
        tests.insert(current_index + 1, test)

    def _stop_after_failure(self, test):
        """Tag unstarted tests early enough to suppress later suite setups."""
        if not self.fail_fast:
            return
        self._stop = True
        root = test.parent
        while root.parent:
            root = root.parent
        for remaining in root.all_tests:
            if remaining.id not in self._attempts:
                remaining.tags.add("robot:skip")

    @staticmethod
    def _is_automatic_failure(result):
        message = result.message or ""
        return message.startswith(
            (
                "Parent suite setup failed:",
                "Test execution stopped due to a fatal error.",
                "Failure occurred and exit-on-failure mode is in use.",
            )
        )

    @staticmethod
    def _keep_final_tests(tests):
        final_by_id = {test.id: test for test in tests}
        unique = []
        seen = set()
        for test in tests:
            if test.id not in seen:
                unique.append(final_by_id[test.id])
                seen.add(test.id)
        tests.clear()
        tests.extend(unique)

    @classmethod
    def _remove_retry_attempts_from_output(cls, path):
        execution_result = ExecutionResult(path)

        def remove_from(suite):
            retained = [
                test for test in suite.tests if cls.WILL_RETRY_TAG not in test.tags
            ]
            suite.tests.clear()
            suite.tests.extend(retained)
            for child in suite.suites:
                remove_from(child)

        remove_from(execution_result.suite)
        ResultWriter(execution_result).write_results(
            output=path, log=None, report=None
        )
