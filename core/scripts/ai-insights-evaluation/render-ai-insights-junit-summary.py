#!/usr/bin/env python3

# SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
# SPDX-License-Identifier: Apache-2.0

"""Render a GitHub Markdown summary from an AI Insights JUnit XML file.

The input is the JUnit XML written by pytest with the AI Insights `ai_*`
properties. The output is a Markdown heading and HTML table suitable for
`GITHUB_STEP_SUMMARY`.

Example:
    render-ai-insights-junit-summary.py ai-insights-evaluation.xml
"""

from __future__ import annotations

import argparse
from pathlib import Path
import xml.etree.ElementTree as ET

from evaluation_summary import render_markdown_summary


AI_PROPERTY_PREFIX = "ai_"


def main() -> int:
    args = parse_args()
    attempts = attempts_from_junit(args.junit_xml)
    if not attempts:
        return 0
    if args.title:
        print(f"## {args.title}")
        print()
    print(render_markdown_summary(attempts), end="")
    return 0


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Render AI Insights recorded pytest properties as a Markdown table.",
    )
    parser.add_argument(
        "junit_xml",
        type=Path,
        help="pytest JUnit XML file from the AI Insights evaluation suite.",
    )
    parser.add_argument(
        "--title",
        default="AI Insights evaluation",
        help="Markdown heading to print before the table. Use an empty string to omit it.",
    )
    return parser.parse_args()


def attempts_from_junit(junit_xml: Path) -> list[dict[str, str]]:
    """Read AI Insights recorded properties from pytest JUnit XML."""
    root = ET.parse(junit_xml).getroot()
    attempts = []
    for testcase in root.iter():
        if _local_name(testcase.tag) != "testcase":
            continue
        properties = _junit_properties(testcase)
        if not properties:
            continue
        properties["pytest_outcome"] = _junit_outcome(testcase)
        attempts.append(properties)
    return attempts


def _junit_properties(testcase: ET.Element) -> dict[str, str]:
    properties: dict[str, str] = {}
    for child in testcase:
        if _local_name(child.tag) != "properties":
            continue
        for prop in child:
            if _local_name(prop.tag) != "property":
                continue
            name = prop.attrib.get("name")
            if not name or not name.startswith(AI_PROPERTY_PREFIX):
                continue
            properties[name] = prop.attrib.get("value", "")
    return properties


def _junit_outcome(testcase: ET.Element) -> str:
    for child in testcase:
        name = _local_name(child.tag)
        if name == "failure":
            return "failed"
        if name == "error":
            return "error"
        if name == "skipped":
            return "skipped"
    return "passed"


def _local_name(tag: str) -> str:
    return tag.rsplit("}", 1)[-1]


if __name__ == "__main__":
    raise SystemExit(main())
