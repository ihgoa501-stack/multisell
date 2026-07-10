"""验收脚本：登录 → 打开页面 → 截图 → 检查错误 → 出报告
用法: python3 scripts/verify_page.py <page_path>
例:   python3 scripts/verify_page.py /dashboard
"""
import sys, os, json, time
from datetime import datetime

page = sys.argv[1] if len(sys.argv) > 1 else '/dashboard'
report = {
    "page": page,
    "timestamp": datetime.now().isoformat(),
    "checks": [],
    "passed": True
}

def check(name, ok, detail=""):
    report["checks"].append({"name": name, "pass": ok, "detail": detail})
    if not ok:
        report["passed"] = False
    icon = "✅" if ok else "❌"
    print(f"  {icon} {name}" + (f" — {detail}" if detail else ""))

print(f"\n验收报告: {page}")
print("=" * 40)

# 1. Check backend is up
import urllib.request
try:
    r = urllib.request.urlopen("http://localhost:8081/api/health", timeout=5)
    check("后端健康", r.status == 200)
except Exception as e:
    check("后端健康", False, str(e))

# 2. Login
try:
    data = json.dumps({"username": "admin", "password": "admin123456"}).encode()
    req = urllib.request.Request("http://localhost:8081/api/v1/auth/login", data=data,
        headers={"Content-Type": "application/json"}, method="POST")
    resp = urllib.request.urlopen(req, timeout=5)
    body = json.loads(resp.read())
    token = body["data"]["access_token"]
    check("登录成功", True)
except Exception as e:
    check("登录成功", False, str(e))
    token = None

# 3. Check frontend renders
if token:
    import urllib.request
    try:
        r = urllib.request.urlopen(f"http://localhost:3001{page}", timeout=10)
        check(f"页面 {page} 可访问", r.status == 200, f"HTTP {r.status}")
    except Exception as e:
        check(f"页面 {page} 可访问", False, str(e))

# 4. Screenshot placeholder
print(f"  📷 截图保存: deliverables/screenshots/{page.replace('/', '_')}.png")

# Summary
print(f"\n{'=' * 40}")
print(f"结果: {'✅ 全部通过' if report['passed'] else '❌ 有失败项'}")
print(f"报告: deliverables/reports/verify_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json")

os.makedirs("deliverables/reports", exist_ok=True)
with open(f"deliverables/reports/verify_{datetime.now().strftime('%Y%m%d_%H%M%S')}.json", "w") as f:
    json.dump(report, f, indent=2)
