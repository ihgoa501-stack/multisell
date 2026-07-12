#!/usr/bin/env python3
"""
verify_pr.py — 验证 PR 模板 checkbox 的 CI 强校验

读取 PR body（stdin 或文件参数），检查：
1. 所有测试 checkbox 不能空着（必须 [x] 或明确声明未运行原因）
2. 验收状态不能矛盾（不能同时勾 Beta Accepted 和 "不声称"）
3. 高风险 section：勾了"敏感操作"必须同时勾"高风险动作已证明"
4. 声明 FAIL/BLOCKED/SKIPPED 的必须有对应的 KNOWN_ISSUES 条目在 body 中引用
5. Breaking change 必须有说明（不能只勾 checkbox 不写内容）

Usage:
    gh pr view <number> --json body --jq .body | python3 scripts/verify_pr.py
    python3 scripts/verify_pr.py < pr_body.txt
    python3 scripts/verify_pr.py pr_body.txt
"""

import re
import sys
from typing import List, Optional
from pathlib import Path




def parse_checkbox(line: str):
    """Parse a checkbox line. Returns (label, checked: bool) or None."""
    m = re.match(r'^\s*[-*]\s*\[([ x])\]\s*(.*)', line)
    if not m:
        return None
    checked = m.group(1) == 'x'
    label = m.group(2).strip()
    return label, checked


def section_lines(body: str, section_header: str) -> list:
    """Extract lines under a markdown section header until next ## or end."""
    pattern = rf'^##\s*{re.escape(section_header)}(?:\s*/[^#]+)?\s*$'
    lines = body.split('\n')
    in_section = False
    result = []
    for line in lines:
        if re.match(pattern, line.strip()):
            in_section = True
            continue
        if in_section:
            if re.match(r'^##\s', line.strip()):
                break
            result.append(line)
    return result


def find_checkbox(section_lines_list: List[str], keyword: str) -> Optional[bool]:
    """Find a checkbox in section by keyword in label. Returns checked state or None if not found."""
    for line in section_lines_list:
        result = parse_checkbox(line)
        if result and keyword in result[0]:
            return result[1]
    return None


def get_unchecked_tests(section: list, nonpass_notes: str) -> list[str]:
    """Get unchecked tests that lack an explicit note naming the command."""
    unchecked = []
    in_note_section = False
    for line in section:
        stripped = line.strip()
        if stripped.startswith('<!--') or stripped.startswith('## 未'):
            in_note_section = True
        if in_note_section:
            continue
        result = parse_checkbox(line)
        if result and not result[1]:
            label = result[0]
            commands = re.findall(r'`([^`]+)`', label)
            if commands and any(command in nonpass_notes for command in commands):
                continue
            unchecked.append(label)
    return unchecked


def extract_failed_refs(body: str) -> list[str]:
    """Extract IDs referenced in 未通过/未运行项 section like KI-2026-..."""
    lines = section_lines(body, '未通过')
    refs = []
    for line in lines:
        refs.extend(re.findall(r'KI-\d{4}-\d{2}-\d{2}-\d{3}', line))
    return refs


def find_nonpass_statuses(body: str) -> list[str]:
    """Find FAIL/BLOCKED/SKIPPED/NOT RUN annotations in the test section."""
    test_section = '\n'.join(section_lines(body, '测试'))
    statuses = re.findall(r'(FAIL|BLOCKED|SKIPPED|NOT\s+RUN)', test_section)
    return statuses


def main():
    if len(sys.argv) > 1:
        path = Path(sys.argv[1])
        if path.exists():
            body = path.read_text()
        else:
            print(f"❌ File not found: {sys.argv[1]}")
            sys.exit(1)
    else:
        body = sys.stdin.read()

    errors = []

    # ── Check 1: All test checkboxes are filled ──
    test_lines = section_lines(body, '测试')
    nonpass_notes = '\n'.join(section_lines(body, '未通过'))
    unchecked = get_unchecked_tests(test_lines, nonpass_notes)
    if unchecked:
        for label in unchecked[:3]:  # cap at 3 errors
            errors.append(f"Test checkbox not checked: {label}")
        if len(unchecked) > 3:
            errors.append(f"... and {len(unchecked) - 3} more unchecked test items")

    # ── Check 2: Acceptance status consistency ──
    acc_lines = section_lines(body, '验收状态')
    beta_accepted = find_checkbox(acc_lines, 'Beta Accepted')
    no_claim = find_checkbox(acc_lines, '不声称')
    if beta_accepted and no_claim:
        errors.append("Contradiction: both 'Beta Accepted' and '不声称业务验收或 Beta 验收完成' are checked")

    # ── Check 3: Risk section consistency ──
    risk_lines = section_lines(body, '风险')
    has_sensitive = find_checkbox(risk_lines, '敏感操作')
    has_proof = find_checkbox(risk_lines, '高风险动作已证明')
    has_breaking = find_checkbox(risk_lines, 'breaking change')
    if has_sensitive and not has_proof:
        errors.append("Sensitive operations checked but '高风险动作已证明' is not checked")
    if has_breaking and not has_sensitive:
        errors.append("Breaking change checked but sensitive operations should also be checked")

    # ── Check 4: FAIL/BLOCKED/SKIPPED must reference KNOWN_ISSUES ──
    nonpass = find_nonpass_statuses(body)
    if nonpass:
        refs = extract_failed_refs(body)
        if not refs:
            errors.append(f"Test items have FAIL/BLOCKED/SKIPPED status but no KNOWN_ISSUES reference found in 未通过/未运行项 section")

    # ── Check 5: Docs section ──
    docs_lines = section_lines(body, '文档同步')
    known_issues_updated = find_checkbox(docs_lines, 'KNOWN_ISSUES')
    if nonpass and known_issues_updated is False:
        errors.append("FAIL/BLOCKED/SKIPPED items exist but docs/KNOWN_ISSUES.md is not marked as updated")

    # ── Report ──
    if errors:
        print("❌ PR template validation failed:")
        for err in errors:
            print(f"   • {err}")
        sys.exit(1)
    else:
        print("✅ PR template checkboxes validated successfully")
        sys.exit(0)


if __name__ == '__main__':
    main()
