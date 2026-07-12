import subprocess
import sys
import unittest
from pathlib import Path


SCRIPT = Path(__file__).with_name("verify_pr.py")


class VerifyPRTest(unittest.TestCase):
    def run_verify(self, body: str) -> subprocess.CompletedProcess[str]:
        return subprocess.run(
            [sys.executable, str(SCRIPT)],
            input=body,
            text=True,
            capture_output=True,
            check=False,
        )

    def test_accepts_unchecked_test_with_explicit_command_reason(self):
        body = """## 测试
- [ ] `npm run e2e` 通过

## 验收状态
- [ ] Beta Accepted
- [x] 本 PR 不声称业务验收或 Beta 验收完成

## 文档同步
- [ ] `docs/KNOWN_ISSUES.md` 已更新

## 风险
- [ ] 涉及敏感操作
- [ ] 高风险动作已证明

## 未通过 / 未运行项
- NOT RUN: `npm run e2e`，本地没有浏览器运行环境。
"""
        result = self.run_verify(body)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_rejects_unchecked_test_without_matching_reason(self):
        body = """## 测试
- [ ] `npm run e2e` 通过

## 验收状态
- [x] 本 PR 不声称业务验收或 Beta 验收完成

## 文档同步
- [ ] `docs/KNOWN_ISSUES.md` 已更新

## 风险
- [ ] 涉及敏感操作
- [ ] 高风险动作已证明

## 未通过 / 未运行项
- NOT RUN: another check was not run.
"""
        result = self.run_verify(body)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Test checkbox not checked", result.stdout)


if __name__ == "__main__":
    unittest.main()
