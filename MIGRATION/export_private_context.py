"""Export a Codex session JSONL while removing credentials and later turns."""

from __future__ import annotations

import argparse
import json
import re
from pathlib import Path
from typing import Any


SECRET_KEYS = {
    "access_token",
    "authorization",
    "cookie",
    "password",
    "passwd",
    "privatekey",
    "private_key",
    "refresh_token",
    "secret",
    "token",
}

TOKEN_PATTERNS = (
    (re.compile(r"\bgh[opsu]_[A-Za-z0-9_]{12,}\b"), "<GITHUB_TOKEN_REDACTED>"),
    (re.compile(r"\bsk-[A-Za-z0-9._-]{16,}\b"), "<OPENAI_LIKE_KEY_REDACTED>"),
    (re.compile(r"\bre_[A-Za-z0-9_]{16,}\b"), "<SERVICE_KEY_REDACTED>"),
    (re.compile(r"\bark-[A-Za-z0-9._-]{16,}\b", re.IGNORECASE), "<SERVICE_KEY_REDACTED>"),
    (re.compile(r"\bAIza[A-Za-z0-9_-]{20,}\b"), "<GOOGLE_KEY_REDACTED>"),
    (re.compile(r"\bxox[baprs]-[A-Za-z0-9-]{12,}\b"), "<SLACK_TOKEN_REDACTED>"),
)

PRIVATE_KEY_BLOCK = re.compile(
    r"-----BEGIN (?:OPENSSH|RSA|EC|DSA) PRIVATE KEY-----.*?"
    r"-----END (?:OPENSSH|RSA|EC|DSA) PRIVATE KEY-----",
    re.IGNORECASE | re.DOTALL,
)
BEARER = re.compile(r"(?i)(\bBearer\s+)[A-Za-z0-9._~+/-]{12,}=*")
URL_USERINFO = re.compile(r"(?i)\b((?:https?|socks5?)://)[^\s/:@]+:[^\s/@]+@")
JSON_SECRET_VALUE = re.compile(
    r"(?i)([\"'](?:password|passwd|token|secret|privateKey|private_key)[\"']\s*:\s*[\"'])"
    r"(?!<[^>]+>)[^\"'\r\n]+([\"'])"
)
ASSIGNMENT_SECRET = re.compile(
    r"(?i)\b(API[_-]?KEY|ACCESS[_-]?TOKEN|REFRESH[_-]?TOKEN|PASSWORD|PASSWD|SECRET)"
    r"(\s*[=:]\s*)(?!<[^>]+>)[^\s,;]+"
)
SSH_PRIVATE_KEY_PATH = re.compile(
    r"(?i)[A-Za-z]:(?:\\+|/+)Users(?:\\+|/+)[^\\/\s\"']+"
    r"(?:\\+|/+)AppData(?:\\+|/+)Local(?:\\+|/+)LCapi"
    r"(?:\\+|/+)ssh(?:\\+|/+)[^\\/\s\"']+"
)


def redact_text(value: str) -> str:
    value = PRIVATE_KEY_BLOCK.sub("<PRIVATE_KEY_BLOCK_REDACTED>", value)
    for pattern, replacement in TOKEN_PATTERNS:
        value = pattern.sub(replacement, value)
    value = BEARER.sub(r"\1<BEARER_TOKEN_REDACTED>", value)
    value = URL_USERINFO.sub(r"\1<URL_USERINFO_REDACTED>@", value)
    value = JSON_SECRET_VALUE.sub(r"\1<SECRET_VALUE_REDACTED>\2", value)
    value = ASSIGNMENT_SECRET.sub(r"\1\2<SECRET_VALUE_REDACTED>", value)
    value = SSH_PRIVATE_KEY_PATH.sub("<SSH_PRIVATE_KEY_PATH_REDACTED>", value)
    return value


def redact(value: Any, key: str | None = None) -> Any:
    if key and key.lower() == "encrypted_content":
        return "<ENCRYPTED_CONTENT_REMOVED>"
    if key and key.lower() in SECRET_KEYS and isinstance(value, str) and value:
        return "<SECRET_VALUE_REDACTED>"
    if isinstance(value, str):
        return redact_text(value)
    if isinstance(value, list):
        return [redact(item) for item in value]
    if isinstance(value, dict):
        return {item_key: redact(item_value, item_key) for item_key, item_value in value.items()}
    return value


def contains_text(value: Any, marker: str) -> bool:
    if not marker:
        return False
    if isinstance(value, str):
        return marker in value
    if isinstance(value, list):
        return any(contains_text(item, marker) for item in value)
    if isinstance(value, dict):
        return any(contains_text(item, marker) for item in value.values())
    return False


def contains_text_casefold(value: Any, marker: str) -> bool:
    if not marker:
        return False
    if isinstance(value, str):
        if value.startswith("data:") and ";base64," in value[:128]:
            return False
        return marker.casefold() in value.casefold()
    if isinstance(value, list):
        return any(contains_text_casefold(item, marker) for item in value)
    if isinstance(value, dict):
        return any(contains_text_casefold(item, marker) for item in value.values())
    return False


def export_session(
    source: Path, destination: Path, stop_before: str, exclude_text: list[str]
) -> dict[str, Any]:
    destination.parent.mkdir(parents=True, exist_ok=True)
    read_count = 0
    written_count = 0
    excluded_count = 0
    stopped = False

    with source.open("r", encoding="utf-8") as reader, destination.open(
        "w", encoding="utf-8", newline="\n"
    ) as writer:
        for raw_line in reader:
            if not raw_line.strip():
                continue
            read_count += 1
            record = json.loads(raw_line)
            if stop_before and contains_text(record, stop_before):
                stopped = True
                break
            if any(contains_text_casefold(record, marker) for marker in exclude_text):
                excluded_count += 1
                continue
            writer.write(json.dumps(redact(record), ensure_ascii=False, separators=(",", ":")))
            writer.write("\n")
            written_count += 1

    return {
        "source": str(source),
        "destination": str(destination),
        "records_read": read_count,
        "records_written": written_count,
        "records_excluded": excluded_count,
        "stopped_before_marker": stopped,
    }


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument("--source", required=True, type=Path)
    parser.add_argument("--destination", required=True, type=Path)
    parser.add_argument("--stop-before", default="")
    parser.add_argument("--exclude-text", action="append", default=[])
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    if not args.source.is_file():
        raise FileNotFoundError(args.source)
    summary = export_session(args.source, args.destination, args.stop_before, args.exclude_text)
    print(json.dumps(summary, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
