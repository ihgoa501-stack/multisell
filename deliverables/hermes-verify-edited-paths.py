#!/usr/bin/env python3
"""Ad-hoc verification for changed paths in multisell project.

Changed paths:
  - backend-go/internal/domain/owner/owner_test.go
  - backend-go/internal/domain/supplier/handler.go

Tests: build, vet, the two previously-failing tests, and full suite.
"""
import subprocess, json, sys

ROOT = "/Users/lc/multisell/backend-go"
checks = []

def check(name, cmd, cwd=ROOT, timeout=120):
    r = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, timeout=timeout)
    ok = r.returncode == 0
    err = r.stderr[:200] if not ok else ""
    checks.append({"check": name, "pass": ok, "detail": err} if err else {"check": name, "pass": ok})
    return ok

check("go build ./...", ["go", "build", "./..."])
check("go vet ./...", ["go", "vet", "./..."])
check("supplier: GetSupplierComparison", ["go", "test", "-v", "./internal/domain/supplier/",
    "-run", "TestHandler_GetSupplierComparison", "-count=1"])
check("owner: RecordFeedback_AdoptWithListingTask", ["go", "test", "-v", "./internal/domain/owner/",
    "-run", "TestService_RecordFeedback_AdoptWithListingTask", "-count=1"])
check("go test ./... (full suite)", ["go", "test", "./..."])

failed = [c for c in checks if not c["pass"]]
print(json.dumps(checks, indent=2))
print(f"\n{'ALL PASSED' if not failed else f'{len(failed)} FAILED'}")
sys.exit(1 if failed else 0)
