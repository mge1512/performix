# SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
# SPDX-License-Identifier: Apache-2.0

from __future__ import annotations

import json
import zipfile
from pathlib import Path
from types import SimpleNamespace

import pytest

import conftest as harness


def make_testcase(recipe: str) -> dict[str, str]:
    return {
        "id": "test_case_32",
        "recipe": recipe,
        "run_artifact": "test_case_32/latest.zip",
    }


def write_common_inputs(root: Path, recipe: str) -> Path:
    case_dir = root / "test_case_32"
    case_dir.mkdir(parents=True)
    archive = case_dir / "latest.zip"
    archive.write_bytes(b"run")
    (case_dir / "metadata.json").write_text(
        json.dumps({"recipe": recipe}), encoding="utf-8"
    )
    return case_dir


def write_metadata(case_dir: Path, metadata: dict[str, object]) -> None:
    (case_dir / "metadata.json").write_text(json.dumps(metadata), encoding="utf-8")


def write_source_archive(case_dir: Path) -> None:
    with zipfile.ZipFile(case_dir / "test_src.zip", "w") as archive:
        archive.writestr("test_src/main.cpp", "int main() {}")


def make_instruction_mix_testcase(mode: str) -> dict[str, str | list[str]]:
    test_case = make_testcase("instruction_mix")
    test_case["recipe_params"] = [f"mode={mode}"]
    return test_case


def test_system_utilization_does_not_require_source_archive(
    tmp_path: Path,
) -> None:
    write_common_inputs(tmp_path, "system_utilization")

    missing = harness._missing_run_artifacts_for_testcases(
        [make_testcase("system_utilization")], tmp_path
    )

    assert missing == []


def test_instruction_mix_requires_source_archive(
    tmp_path: Path,
) -> None:
    case_dir = write_common_inputs(tmp_path, "instruction_mix")

    missing = harness._missing_run_artifacts_for_testcases(
        [make_instruction_mix_testcase("dynamic")], tmp_path
    )

    assert missing == [
        ("test_case_32", "source archive", case_dir / "test_src.zip")
    ]


def test_static_instruction_mix_does_not_require_source_archive(tmp_path: Path) -> None:
    write_common_inputs(tmp_path, "instruction_mix")

    missing = harness._missing_run_artifacts_for_testcases(
        [make_instruction_mix_testcase("static")], tmp_path
    )

    assert missing == []


def test_code_hotspots_requires_source_archive(tmp_path: Path) -> None:
    case_dir = write_common_inputs(tmp_path, "code_hotspots")

    missing = harness._missing_run_artifacts_for_testcases(
        [make_testcase("code_hotspots")], tmp_path
    )

    assert missing == [
        ("test_case_32", "source archive", case_dir / "test_src.zip")
    ]


def test_cpu_microarchitecture_requires_source_archive(tmp_path: Path) -> None:
    case_dir = write_common_inputs(tmp_path, "cpu_microarchitecture")

    missing = harness._missing_run_artifacts_for_testcases(
        [make_testcase("cpu_microarchitecture")], tmp_path
    )

    assert missing == [
        ("test_case_32", "source archive", case_dir / "test_src.zip")
    ]


def test_system_utilization_does_not_download_source_archive(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    write_common_inputs(tmp_path, "system_utilization")
    downloads = []
    monkeypatch.setattr(
        harness,
        "_download_artifactory_input",
        lambda *args, **kwargs: downloads.append((args, kwargs)),
    )

    harness._download_missing_run_artifacts(
        [make_testcase("system_utilization")], tmp_path, "repository/path"
    )

    assert downloads == []


def test_metadata_recipe_must_match_manifest(tmp_path: Path) -> None:
    write_common_inputs(tmp_path, "code_hotspots")

    with pytest.raises(pytest.UsageError, match="recipe mismatch"):
        harness._validate_prerecorded_metadata(
            [make_testcase("system_utilization")], tmp_path
        )


def test_instruction_mix_metadata_mode_must_match_manifest(tmp_path: Path) -> None:
    case_dir = write_common_inputs(tmp_path, "instruction_mix")
    write_metadata(case_dir, {"recipe": "instruction_mix", "recipe_params": ["mode=static"]})

    with pytest.raises(pytest.UsageError, match="recipe params mismatch"):
        harness._validate_prerecorded_metadata(
            [make_instruction_mix_testcase("dynamic")], tmp_path
        )


def test_system_utilization_import_does_not_attach_source(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    write_common_inputs(tmp_path, "system_utilization")
    calls = []

    def run_cli(command, cwd):
        calls.append(command)
        return SimpleNamespace(
            stdout=json.dumps({"data": {"new_id": {"value": "run-123"}}})
        )

    class Cache:
        def __init__(self):
            self.value = None

        def get(self, key, default):
            return default

        def set(self, key, value):
            self.value = value

    cache = Cache()
    monkeypatch.setattr(harness, "run_cli", run_cli)

    harness._prepare_imported_runs(
        [make_testcase("system_utilization")],
        cache,
        tmp_path / "apx",
        tmp_path,
        tmp_path / "results",
    )

    assert len(calls) == 1
    assert calls[0][1:3] == ["run", "import"]
    assert cache.value is not None
    assert "source_root" not in cache.value


@pytest.mark.parametrize(
    "recipe", ["code_hotspots", "cpu_microarchitecture", "instruction_mix"]
)
def test_recipe_with_source_archive_import_attaches_extracted_source(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch, recipe: str
) -> None:
    case_dir = write_common_inputs(tmp_path, recipe)
    write_source_archive(case_dir)
    source_sha = harness.sha256_file(case_dir / "test_src.zip")
    source_root = (
        tmp_path / "results" / "imported_sources" / "test_case_32" / source_sha / "test_src"
    )
    source_root.mkdir(parents=True)
    calls = []

    def run_cli(command, cwd):
        calls.append(command)
        return SimpleNamespace(
            stdout=json.dumps({"data": {"new_id": {"value": "run-123"}}})
        )

    class Cache:
        def get(self, key, default):
            return default

        def set(self, key, value):
            self.value = value

    cache = Cache()
    monkeypatch.setattr(harness, "run_cli", run_cli)

    harness._prepare_imported_runs(
        [
            make_instruction_mix_testcase("dynamic")
            if recipe == "instruction_mix"
            else make_testcase(recipe)
        ],
        cache,
        tmp_path / "apx",
        tmp_path,
        tmp_path / "results",
    )

    assert len(calls) == 2
    assert calls[1][-2:] == ["--source", str(source_root)]
    assert cache.value["source_root"] == str(source_root)


def test_static_instruction_mix_import_does_not_attach_source(
    tmp_path: Path, monkeypatch: pytest.MonkeyPatch
) -> None:
    write_common_inputs(tmp_path, "instruction_mix")
    calls = []

    def run_cli(command, cwd):
        calls.append(command)
        return SimpleNamespace(
            stdout=json.dumps({"data": {"new_id": {"value": "run-123"}}})
        )

    class Cache:
        def __init__(self):
            self.value = None

        def get(self, key, default):
            return default

        def set(self, key, value):
            self.value = value

    cache = Cache()
    monkeypatch.setattr(harness, "run_cli", run_cli)

    harness._prepare_imported_runs(
        [make_instruction_mix_testcase("static")],
        cache,
        tmp_path / "apx",
        tmp_path,
        tmp_path / "results",
    )

    assert len(calls) == 1
    assert calls[0][1:3] == ["run", "import"]
    assert cache.value is not None
    assert "source_root" not in cache.value
