# SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
# SPDX-License-Identifier: Apache-2.0

import sys
import tempfile
import types
import unittest
from pathlib import Path
from unittest.mock import Mock, patch

import pytest

import conftest


class AiInsightsGraphReportTests(unittest.TestCase):
    def test_identifies_ai_insights_collection(self):
        self.assertTrue(
            conftest._is_ai_insights_collection(
                ["test_ai_insights_evaluation.py::test_ai_insights[test_case_01-rest]"]
            )
        )
        self.assertFalse(
            conftest._is_ai_insights_collection(
                ["test_conftest_graph_report.py::test_unrelated"]
            )
        )

    def test_generates_graphs_from_junit_report(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            junit_xml = root / "ai-insights-evaluation.xml"
            junit_xml.write_text(
                "<testsuites><testsuite><testcase><properties>"
                '<property name="ai_mode" value="rest" />'
                "</properties></testcase></testsuite></testsuites>",
                encoding="utf-8",
            )
            results_dir = root / "results"
            config = types.SimpleNamespace(
                option=types.SimpleNamespace(xmlpath=str(junit_xml)),
                getoption=Mock(return_value=str(results_dir)),
            )
            plot_module = types.ModuleType("plot_ai_insights_junit")
            plot_module.load_observations = Mock(return_value=["observation"])
            plot_module.generate_graphs = Mock(
                return_value=[results_dir / "reporting" / "graphs" / "index.html"]
            )

            with patch.dict(sys.modules, {"plot_ai_insights_junit": plot_module}):
                report = conftest._generate_graph_report(config)

        resolved_junit_xml = junit_xml.resolve()
        output_dir = (results_dir / "reporting" / "graphs").resolve()
        self.assertEqual(
            {"output_dir": str(output_dir), "file_count": 1},
            report,
        )
        plot_module.load_observations.assert_called_once_with(resolved_junit_xml)
        plot_module.generate_graphs.assert_called_once_with(["observation"], output_dir)

    def test_skips_graphs_without_junit_reporting(self):
        config = types.SimpleNamespace(option=types.SimpleNamespace(xmlpath=None))

        self.assertIsNone(conftest._generate_graph_report(config))

    def test_skips_graphs_for_junit_without_ai_insights_results(self):
        with tempfile.TemporaryDirectory() as tmp:
            junit_xml = Path(tmp) / "results.xml"
            junit_xml.write_text(
                "<testsuites><testsuite><testcase /></testsuite></testsuites>",
                encoding="utf-8",
            )
            config = types.SimpleNamespace(
                option=types.SimpleNamespace(xmlpath=str(junit_xml)),
            )

            self.assertIsNone(conftest._generate_graph_report(config))

    def test_graph_failure_fails_an_otherwise_successful_session(self):
        config = types.SimpleNamespace(
            option=types.SimpleNamespace(collectonly=False),
        )
        session = types.SimpleNamespace(config=config, exitstatus=pytest.ExitCode.OK)

        with patch.object(
            conftest,
            "_generate_graph_report",
            side_effect=ValueError("no graphable observations"),
        ):
            conftest.pytest_sessionfinish(session, pytest.ExitCode.OK)

        self.assertEqual(pytest.ExitCode.TESTS_FAILED, session.exitstatus)
        self.assertEqual(
            {"error": "no graphable observations"},
            getattr(config, conftest.AI_GRAPH_REPORT_ATTR),
        )

    def test_unexpected_graph_failure_preserves_existing_failure(self):
        config = types.SimpleNamespace(
            option=types.SimpleNamespace(collectonly=False),
        )
        session = types.SimpleNamespace(
            config=config,
            exitstatus=pytest.ExitCode.TESTS_FAILED,
        )

        with patch.object(
            conftest,
            "_generate_graph_report",
            side_effect=RuntimeError("matplotlib failed"),
        ):
            conftest.pytest_sessionfinish(session, pytest.ExitCode.TESTS_FAILED)

        self.assertEqual(pytest.ExitCode.TESTS_FAILED, session.exitstatus)
        self.assertEqual(
            {"error": "matplotlib failed"},
            getattr(config, conftest.AI_GRAPH_REPORT_ATTR),
        )


if __name__ == "__main__":
    unittest.main()
