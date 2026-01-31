#!/usr/bin/env python3
import argparse
import json
import os
import sys
import urllib.error
import urllib.request


def post_json(url, payload, headers=None):
    data = json.dumps(payload).encode("utf-8")
    final_headers = {"Content-Type": "application/json"}
    if headers:
        final_headers.update(headers)
    req = urllib.request.Request(url, data=data, headers=final_headers, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            body = resp.read().decode("utf-8")
            return resp.status, body
    except urllib.error.HTTPError as e:
        body = e.read().decode("utf-8") if e.fp else ""
        return e.code, body


def rpc_call(base_url, container_id, rpc_id, method, params=None, headers=None):
    url = f"{base_url.rstrip('/')}/mcp/fs/{container_id}"
    payload = {"jsonrpc": "2.0", "id": rpc_id, "method": method}
    if params is not None:
        payload["params"] = params
    status, body = post_json(url, payload, headers=headers)
    return status, body


def print_response(title, status, body):
    print(f"\n== {title} ==")
    print(f"HTTP {status}")
    if body:
        print(body)
    else:
        print("<empty body>")


def parse_json(body):
    try:
        return json.loads(body)
    except Exception:
        return None


def get_tool_result(payload):
    if not isinstance(payload, dict):
        return None
    return payload.get("result")


def get_structured_content(result):
    if not isinstance(result, dict):
        return None
    if "structuredContent" in result:
        return result.get("structuredContent")
    content = result.get("content") or []
    if not content:
        return None
    text = content[0].get("text")
    if not text:
        return None
    return parse_json(text)


def expect_ok(title, status, body, failures):
    if status != 200:
        failures.append(f"{title}: expected HTTP 200, got {status}")
        return None
    payload = parse_json(body)
    result = get_tool_result(payload)
    if not isinstance(result, dict) or result.get("isError"):
        failures.append(f"{title}: expected isError=false")
        return result
    return result


def expect_error(title, status, body, failures):
    if status != 200:
        failures.append(f"{title}: expected HTTP 200, got {status}")
        return None
    payload = parse_json(body)
    result = get_tool_result(payload)
    if not isinstance(result, dict) or not result.get("isError"):
        failures.append(f"{title}: expected isError=true")
        return result
    return result


def main():
    parser = argparse.ArgumentParser(description="Test MCP fs JSON-RPC endpoint")
    parser.add_argument(
        "--base-url",
        default="http://127.0.0.1:8080",
        help="API base URL (default: http://127.0.0.1:8080)",
    )
    parser.add_argument(
        "--container-id",
        default="test-create-1769798787",
        help="Container ID to target",
    )
    parser.add_argument(
        "--path",
        default="notes.txt",
        help="Relative path used in examples",
    )
    parser.add_argument(
        "--token",
        default="",
        help="Bearer token (or set MCP_TOKEN env var)",
    )
    args = parser.parse_args()
    token = args.token or os.getenv("MCP_TOKEN", "")
    headers = {"Authorization": f"Bearer {token}"} if token else None

    failures = []
    rpc_id = 1

    def call(title, method, params=None):
        nonlocal rpc_id
        status, body = rpc_call(
            args.base_url,
            args.container_id,
            rpc_id,
            method,
            params=params,
            headers=headers,
        )
        print_response(title, status, body)
        rpc_id += 1
        return status, body

    status, body = call("tools/list", "tools/list")
    expect_ok("tools/list", status, body, failures)

    files = [
        ("alpha.txt", "alpha"),
        ("dir1/beta.txt", "beta"),
        ("dir1/dir2/gamma.txt", "gamma"),
    ]

    for path, content in files:
        status, body = call(
            f"fs.write {path}",
            "tools/call",
            {"name": "fs.write", "arguments": {"path": path, "content": content}},
        )
        expect_ok(f"fs.write {path}", status, body, failures)

    for path, content in files:
        status, body = call(
            f"fs.read {path}",
            "tools/call",
            {"name": "fs.read", "arguments": {"path": path}},
        )
        result = expect_ok(f"fs.read {path}", status, body, failures)
        sc = get_structured_content(result) if result else None
        if not sc or sc.get("content") != content:
            failures.append(f"fs.read {path}: content mismatch")

    status, body = call(
        "fs.list (non-recursive)",
        "tools/call",
        {"name": "fs.list", "arguments": {"path": "", "recursive": False}},
    )
    expect_ok("fs.list (non-recursive)", status, body, failures)

    status, body = call(
        "fs.list (recursive)",
        "tools/call",
        {"name": "fs.list", "arguments": {"path": "", "recursive": True}},
    )
    result = expect_ok("fs.list (recursive)", status, body, failures)
    sc = get_structured_content(result) if result else None
    if sc and "entries" in sc:
        listed = {e.get("path") for e in sc.get("entries", [])}
        for path, _ in files:
            if path not in listed:
                failures.append(f"fs.list (recursive): missing {path}")

    for path, _ in files:
        status, body = call(
            f"fs.stat {path}",
            "tools/call",
            {"name": "fs.stat", "arguments": {"path": path}},
        )
        expect_ok(f"fs.stat {path}", status, body, failures)

    patch = "@@ -1,5 +1,5 @@\n-alpha\n+alpha-patched"
    status, body = call(
        "fs.apply_patch alpha.txt",
        "tools/call",
        {"name": "fs.apply_patch", "arguments": {"path": "alpha.txt", "patch": patch}},
    )
    expect_ok("fs.apply_patch alpha.txt", status, body, failures)

    status, body = call(
        "fs.read alpha.txt (after patch)",
        "tools/call",
        {"name": "fs.read", "arguments": {"path": "alpha.txt"}},
    )
    result = expect_ok("fs.read alpha.txt (after patch)", status, body, failures)
    sc = get_structured_content(result) if result else None
    if not sc or sc.get("content") != "alpha-patched":
        failures.append("fs.read alpha.txt (after patch): content mismatch")

    status, body = call(
        "fs.delete dir1",
        "tools/call",
        {"name": "fs.delete", "arguments": {"path": "dir1"}},
    )
    expect_ok("fs.delete dir1", status, body, failures)

    status, body = call(
        "fs.read dir1/beta.txt (after delete)",
        "tools/call",
        {"name": "fs.read", "arguments": {"path": "dir1/beta.txt"}},
    )
    expect_error("fs.read dir1/beta.txt (after delete)", status, body, failures)

    status, body = call(
        "fs.read ../escape (invalid path)",
        "tools/call",
        {"name": "fs.read", "arguments": {"path": "../escape"}},
    )
    expect_error("fs.read ../escape (invalid path)", status, body, failures)

    if failures:
        print("\n== SUMMARY ==")
        for item in failures:
            print(f"- FAIL: {item}")
        return 1

    print("\n== SUMMARY ==\nAll checks passed.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
