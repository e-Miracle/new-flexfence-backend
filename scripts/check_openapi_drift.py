#!/usr/bin/env python3
"""Fail CI when implemented HTTP handlers drift from OpenAPI paths/methods."""

from __future__ import annotations

import re
import sys
from pathlib import Path


HTTP_METHODS = {"get", "post", "put", "patch", "delete", "options", "head"}


def implemented_endpoints() -> set[tuple[str, str]]:
    # Keep this list aligned with router handlers.
    return {
        ("GET", "/health"),
        ("POST", "/v1/auth/business/register"),
        ("POST", "/v1/auth/business/login"),
        ("POST", "/v1/auth/business/otp/verify"),
        ("POST", "/v1/auth/business/otp/resend"),
        ("POST", "/v1/auth/business/oauth/google"),
        ("POST", "/v1/auth/user/register"),
        ("POST", "/v1/auth/user/login"),
        ("POST", "/v1/auth/user/password-reset/request"),
        ("POST", "/v1/auth/user/password-reset/resend"),
        ("POST", "/v1/auth/user/password-reset/confirm"),
        ("POST", "/v1/auth/user/oauth/google"),
        ("GET", "/v1/me/event-joins"),
        ("GET", "/v1/me/subscribed-events"),
        ("DELETE", "/v1/me/event-joins/{joinId}"),
        ("DELETE", "/v1/me"),
        ("POST", "/v1/me/change-password"),
        ("GET", "/v1/consent-field-recommendations"),
        ("GET", "/v1/events"),
        ("POST", "/v1/events"),
        ("GET", "/v1/events/{eventId}"),
        ("POST", "/v1/events/{eventId}/fences"),
        ("POST", "/v1/events/{eventId}/join-by-qr"),
        ("POST", "/v1/events/{eventId}/attendance/mark-present"),
        ("GET", "/v1/events/{eventId}/fences"),
        ("DELETE", "/v1/events/{eventId}/fences/{fenceId}"),
        ("GET", "/v1/events/{eventId}/attendance"),
        ("GET", "/v1/events/{eventId}/analytics"),
        ("GET", "/v1/events/{eventId}/fences/{fenceId}/analytics"),
        ("GET", "/v1/events/{eventId}/share"),
        ("POST", "/v1/events/{eventId}/share/regenerate"),
        ("GET", "/v1/events/{eventId}/consent-template"),
        ("PUT", "/v1/events/{eventId}/consent-template"),
    }


def parse_openapi_endpoints(openapi_path: Path) -> set[tuple[str, str]]:
    if not openapi_path.exists():
        raise FileNotFoundError(f"OpenAPI file missing: {openapi_path}")

    lines = openapi_path.read_text(encoding="utf-8").splitlines()
    in_paths = False
    current_path = None
    parsed: set[tuple[str, str]] = set()

    path_line = re.compile(r"^\s{2}(/[^:]*):\s*$")
    method_line = re.compile(r"^\s{4}([a-z]+):\s*$")

    for line in lines:
        stripped = line.strip()
        if stripped == "paths:":
            in_paths = True
            current_path = None
            continue

        # Stop parsing when leaving paths section.
        if in_paths and re.match(r"^[a-zA-Z_][a-zA-Z0-9_]*:\s*$", line):
            break

        if not in_paths:
            continue

        p_match = path_line.match(line)
        if p_match:
            current_path = p_match.group(1)
            continue

        m_match = method_line.match(line)
        if m_match and current_path:
            method = m_match.group(1).lower()
            if method in HTTP_METHODS:
                parsed.add((method.upper(), current_path))

    return parsed


def format_endpoints(items: set[tuple[str, str]]) -> str:
    return "\n".join(f"  - {m} {p}" for m, p in sorted(items))


def main() -> int:
    repo_root = Path(__file__).resolve().parents[1]
    openapi_file = repo_root / "openapi" / "openapi.yaml"

    impl = implemented_endpoints()
    spec = parse_openapi_endpoints(openapi_file)

    missing_in_spec = impl - spec
    extra_in_spec = spec - impl

    if missing_in_spec or extra_in_spec:
        print("OpenAPI drift detected between handlers and spec.")
        if missing_in_spec:
            print("\nImplemented but missing in OpenAPI:")
            print(format_endpoints(missing_in_spec))
        if extra_in_spec:
            print("\nPresent in OpenAPI but not implemented:")
            print(format_endpoints(extra_in_spec))
        return 1

    print("OpenAPI drift check passed.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
