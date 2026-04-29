#!/usr/bin/env python3
"""
Refresh integration test fixtures from the SuprSend staging API.

Usage:
    python3 refresh_fixtures.py                      # refresh all fixtures
    python3 refresh_fixtures.py preference_category  # refresh one fixture by name

Credentials are read from the repo's .vscode/launch.json (staging config).
"""

import json
import re
import subprocess
import sys
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parents[6]
FIXTURES_DIR = Path(__file__).parent

# ---------------------------------------------------------------------------
# Fixture registry
# Each entry: name -> {endpoint, file, params (optional), description}
# {workspace} is substituted with the workspace from launch.json at runtime.
# ---------------------------------------------------------------------------
FIXTURES = {
    "preference_category": {
        "file": "preference_category.json",
        "endpoint": "/v1/{workspace}/preference_category/",
        "params": {"mode": "live"},
        "description": "Live preference category tree for the workspace",
    },
    "preference_category_draft": {
        "file": "preference_category_draft.json",
        "endpoint": "/v1/{workspace}/preference_category/",
        "params": {"mode": "draft"},
        "description": "Draft preference category tree for the workspace",
    },
    "translation_locales": {
        "file": "translation_locales.json",
        "endpoint": "/v1/{workspace}/preference_category/translation/locale",
        "description": "Available translation locales for preference categories",
    },
    "translation_content_en_NA": {
        "file": "translation_content_en_NA.json",
        "endpoint": "/v1/{workspace}/preference_category/translation/content/en-NA",
        "description": "Preference category translations for en-NA locale",
    },
    "translation_content_es": {
        "file": "translation_content_es.json",
        "endpoint": "/v1/{workspace}/preference_category/translation/content/es",
        "description": "Preference category translations for es locale",
    },
    "translation_content_es_AR": {
        "file": "translation_content_es_AR.json",
        "endpoint": "/v1/{workspace}/preference_category/translation/content/es-AR",
        "description": "Preference category translations for es-AR locale",
    },
    "translation_content_es_BO": {
        "file": "translation_content_es_BO.json",
        "endpoint": "/v1/{workspace}/preference_category/translation/content/es-BO",
        "description": "Preference category translations for es-BO locale",
    },
}


def load_credentials():
    launch_json = REPO_ROOT / ".vscode" / "launch.json"
    raw = launch_json.read_text()
    # launch.json is JSONC — extract fields directly to avoid comment/trailing-comma issues
    def extract(key):
        m = re.search(rf'"{key}"\s*:\s*"([^"]+)"', raw)
        if not m:
            raise RuntimeError(f"{key} not found in launch.json")
        return m.group(1)

    token = extract("SUPRSEND_SERVICE_TOKEN")
    mgmnt_url = extract("SUPRSEND_MGMNT_URL").rstrip("/")
    return token, mgmnt_url, "staging"


def fetch(url, token):
    result = subprocess.run(
        ["curl", "-sf", url, "-H", f"Authorization: ServiceToken {token}"],
        capture_output=True, text=True,
    )
    if result.returncode != 0:
        raise RuntimeError(f"curl failed (exit {result.returncode}): {result.stderr.strip()}")
    return json.loads(result.stdout)


def refresh_fixture(name, token, mgmnt_url, workspace):
    spec = FIXTURES[name]
    endpoint = spec["endpoint"].format(workspace=workspace)
    params = spec.get("params", {})
    query = "&".join(f"{k}={v}" for k, v in params.items())
    url = f"{mgmnt_url}{endpoint}" + (f"?{query}" if query else "")

    outpath = FIXTURES_DIR / spec["file"]
    outpath.parent.mkdir(parents=True, exist_ok=True)

    data = fetch(url, token)
    outpath.write_text(json.dumps(data, indent=4))
    print(f"  ✓  {name:40s}  →  {spec['file']}")


def main():
    token, mgmnt_url, workspace = load_credentials()

    requested = sys.argv[1:]
    if requested:
        unknown = [n for n in requested if n not in FIXTURES]
        if unknown:
            print(f"Unknown fixture(s): {', '.join(unknown)}")
            print(f"Available: {', '.join(FIXTURES)}")
            sys.exit(1)
        targets = requested
    else:
        targets = list(FIXTURES)

    print(f"Refreshing {len(targets)} fixture(s) from {mgmnt_url} (workspace: {workspace})\n")
    errors = []
    for name in targets:
        try:
            refresh_fixture(name, token, mgmnt_url, workspace)
        except Exception as e:
            print(f"  ✗  {name}: {e}")
            errors.append(name)

    print()
    if errors:
        print(f"Failed: {', '.join(errors)}")
        sys.exit(1)
    else:
        print("Done.")


if __name__ == "__main__":
    main()
