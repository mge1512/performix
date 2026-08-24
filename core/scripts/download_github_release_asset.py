#!/usr/bin/env python3

# SPDX-FileCopyrightText: Copyright 2026 Arm Limited and/or its affiliates <open-source-office@arm.com>
# SPDX-License-Identifier: Apache-2.0

"""
This script is used to download a GitHub release asset. Set GITHUB_TOKEN for authentication.
"""

import argparse
import certifi
import json
import os
import sys
import ssl
import time
from typing import Dict, List
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

# Use certifi's CA bundle for SSL verification, not system defaults which may be outdated or missing (e.g. on Windows).
ctx = ssl.create_default_context(cafile=certifi.where())

ACCEPT_JSON = "application/vnd.github+json"
ACCEPT_BINARY = "application/octet-stream"
CHUNK_SIZE = 1024 * 256
MAX_RETRIES = 3
INITIAL_BACKOFF_SECONDS = 1
RETRYABLE_HTTP_CODES = {408, 429, 500, 502, 503, 504}


def with_retries(operation, description: str):
    for retry in range(MAX_RETRIES + 1):
        try:
            return operation()
        except HTTPError as exc:
            if exc.code not in RETRYABLE_HTTP_CODES or retry == MAX_RETRIES:
                raise
            reason = f"HTTP {exc.code} {exc.reason}"
        except (URLError, TimeoutError, ConnectionError) as exc:
            if retry == MAX_RETRIES:
                raise
            reason = getattr(exc, "reason", exc)

        delay = INITIAL_BACKOFF_SECONDS * (2**retry)
        print(
            f"{description} failed ({reason}); retrying in {delay}s "
            f"({retry + 1}/{MAX_RETRIES})",
            file=sys.stderr,
        )
        time.sleep(delay)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Download a GitHub release asset. Set GITHUB_TOKEN for authentication."
    )
    parser.add_argument("--repo", required=True, help="The GitHub repository.")
    parser.add_argument("--tag", required=True, help="The Release tag.")
    parser.add_argument(
        "--asset-name", required=True, help="The asset filename to download."
    )
    parser.add_argument("--out", required=True, help="The output directory.")
    return parser.parse_args()


def headers(token: str, accept: str) -> dict:
    h = {"Accept": accept, "User-Agent": "gh-release-asset-downloader/0.1"}
    if token:
        h["Authorization"] = f"Bearer {token}"
    return h


def fetch_release(repo: str, tag: str, token: str) -> dict:
    url = f"https://api.github.com/repos/{repo}/releases/tags/{tag}"
    req = Request(url, headers=headers(token, ACCEPT_JSON))

    def fetch():
        with urlopen(req, context=ctx) as resp:
            charset = resp.headers.get_content_charset() or "utf-8"
            data = resp.read().decode(charset)
            return json.loads(data)

    try:
        return with_retries(fetch, "GitHub API request")
    except HTTPError as exc:
        if exc.code == 403:
            sys.exit(
                "Error 403 (forbidden): is your token authorised for Arm-Debug SSO?"
            )
        sys.exit(f"GitHub API request failed: {exc.code} {exc.reason}")
    except (URLError, TimeoutError, ConnectionError) as exc:
        sys.exit(f"Unable to reach GitHub: {getattr(exc, 'reason', exc)}")


def pick_asset(assets: List[Dict], name: str) -> Dict:
    if not assets:
        sys.exit("This release has no assets.")
    for a in assets:
        if a.get("name") == name:
            return a
    available = ", ".join(a.get("name", "?") for a in assets) or "(no assets)"
    sys.exit(f"Asset {name!r} not found. Available: {available}")


def download_asset(asset: dict, token: str, out_dir: str) -> str:
    req = Request(asset["url"], headers=headers(token, ACCEPT_BINARY), method="GET")
    target = os.path.join(out_dir, asset["name"])
    target_dir = os.path.dirname(target)
    if target_dir:
        os.makedirs(target_dir, exist_ok=True)

    def download():
        with urlopen(req, context=ctx) as resp, open(target, "wb") as fh:
            while True:
                chunk = resp.read(CHUNK_SIZE)
                if not chunk:
                    break
                fh.write(chunk)

    try:
        with_retries(download, "Asset download")
    except HTTPError as exc:
        if exc.code == 403:
            sys.exit(
                "Error 403 (forbidden): is your token authorised for Arm-Debug SSO?"
            )
        sys.exit(f"Asset download failed: {exc.code} {exc.reason}")
    except (URLError, TimeoutError, ConnectionError) as exc:
        sys.exit(f"Unable to download asset: {getattr(exc, 'reason', exc)}")

    return target


def main():
    args = parse_args()

    out_dir = args.out
    if os.path.exists(out_dir) and not os.path.isdir(out_dir):
        sys.exit("--out must be a directory.")

    token = os.environ.get("GITHUB_TOKEN")
    if not token:
        sys.exit(
            "GITHUB_TOKEN env var is not set. Set this to a GitHub personal access token with permissions for the Arm-Debug organisation."
        )

    release = fetch_release(args.repo, args.tag, token)
    assets = release.get("assets", [])

    asset = pick_asset(assets, args.asset_name)
    dest = download_asset(asset, token, out_dir)
    print(f"Downloaded: {dest}")


if __name__ == "__main__":
    main()
