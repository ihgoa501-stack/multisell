#!/usr/bin/env python3
"""Demo Acceptance — API-level verification against localhost:8000."""

import json
import os
import subprocess
import urllib.request
import io
import uuid

BASE = "http://localhost:8000/api"
REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
CSV_DIR = os.path.join(REPO_ROOT, "docs", "demo-data")
FRONTEND_DIR = os.path.join(REPO_ROOT, "frontend")

def req(method, path, data=None, files=None, form=None, token=None):
    url = f"{BASE}/{path.lstrip('/')}"
    headers = {"Authorization": f"Bearer {token}"} if token else {}
    if files:
        boundary = uuid.uuid4().hex
        body = io.BytesIO()
        for key, value in (form or {}).items():
            body.write(f"--{boundary}\r\n".encode())
            body.write(f'Content-Disposition: form-data; name="{key}"\r\n\r\n'.encode())
            body.write(str(value).encode())
            body.write(b"\r\n")
        for key, (filename, content, mime) in files.items():
            body.write(f"--{boundary}\r\n".encode())
            body.write(f'Content-Disposition: form-data; name="{key}"; filename="{filename}"\r\n'.encode())
            body.write(f"Content-Type: {mime}\r\n\r\n".encode())
            if isinstance(content, str): body.write(content.encode())
            else: body.write(content)
            body.write(b"\r\n")
        body.write(f"--{boundary}--\r\n".encode())
        headers["Content-Type"] = f"multipart/form-data; boundary={boundary}"
        post_data = body.getvalue()
    elif data:
        post_data = json.dumps(data).encode()
        headers["Content-Type"] = "application/json"
    else:
        post_data = None
    r = urllib.request.Request(url, data=post_data, method=method, headers=headers)
    try:
        resp = urllib.request.urlopen(r)
        return resp.status, json.loads(resp.read().decode())
    except urllib.error.HTTPError as e:
        body = e.read().decode()
        try: return e.code, json.loads(body)
        except Exception:
            return e.code, {"error": body}

def main():
    results = []
    print("=" * 65)
    print("  LingMirror Demo Acceptance (API) Report")
    print("=" * 65)

    # 0 — Login
    print("\n[0] Login...")
    c, d = req("POST", "/auth/login", {"username": "demo", "password": "demo123"})
    if c == 200:
        tok = d["data"]["access_token"]
        print(f"  + OK — role={d['data']['user']['role']}")
    else:
        print(f"  ! FAIL: {d}")
        return

    # 1 — Products
    c, d = req("GET", "/products", token=tok)
    prods = d.get("data", d).get("items", d.get("records", []))
    n = len(prods)
    results.append(("Products visible", "PASS" if n >= 5 else "FAIL", f"{n} products"))
    print(f"  {'+' if n>=5 else '!'} Products: {n}")

    # 2 — SKUs
    first_product_id = prods[0]["id"] if prods else None
    c, d = req("GET", f"/products/{first_product_id}/skus", token=tok) if first_product_id else (0, {})
    skus = d.get("data", []) if c == 200 else []
    n = len(skus)
    first_sku_id = skus[0]["id"] if skus else None
    results.append(("SKUs visible", "PASS" if n >= 1 else "FAIL", f"{n} SKUs for product {first_product_id}"))

    # 3 — Shipping Calculator
    c, _ = req("POST", "/shipping/calculate", {
        "mode": "manual",
        "destination_country": "RU",
        "cargo_type": "normal",
        "quantity": 1,
        "package": {"length_cm": 30, "width_cm": 20, "height_cm": 10, "weight_kg": 0.5},
    }, token=tok)
    results.append(("Shipping Calculator", "PASS" if c == 200 else "FAIL", ""))

    # 4 — Decision
    c, d = req("POST", "/decisions/prelisting", {
        "sku_id": first_sku_id or 1,
        "target_sale_price": 299,
        "destination_country": "RU",
        "platform_fee_pct": 10,
        "payment_fee_pct": 3,
    }, token=tok) if first_sku_id else (0, {})
    ok = c == 200 and d.get("code") == 200
    results.append(("Pre-listing Decision", "PASS" if ok else "FAIL", ""))

    # 5 — Order Import
    print("\n[5] Order Import...")
    csv_path = os.path.join(CSV_DIR, "order_import_demo.csv")
    with open(csv_path, "rb") as f: csv_content = f.read()
    c, d = req("POST", "/order-imports/csv",
        form={"adapter_code": "csv_order"},
        files={"file": ("orders.csv", csv_content, "text/csv")},
        token=tok)
    if c == 200:
        b = d["data"]
        bid = b["id"]
        rc, coc, fc, _cs = b["row_count"], b["created_order_count"], b["failed_count"], b["chain_status"]
        ok = rc >= 6 and coc >= 3
        results.append(("Order Import", "PASS" if ok else "FAIL", f"rows={rc}, orders={coc}, fail={fc}"))
        print(f"  {'+' if ok else '!'} rows={rc}, orders={coc}, fail={fc}")
    else:
        results.append(("Order Import", "FAIL", str(d)))
        print(f"  ! FAIL: {d.get('message','')}")
        bid = None

    # 6 — Multi-SKU
    print("\n[6] Multi-SKU...")
    if bid:
        c, d = req("GET", f"/order-imports/{bid}/items", token=tok)
        if c == 200:
            items = d.get("data", [])
            oids = {it["order_id"] for it in items if it.get("order_id")}
            po_map = {}
            for it in items:
                po = it.get("platform_order_no")
                if po: po_map.setdefault(po, []).append(it)
            multi = [po for po, its in po_map.items() if len(its) > 1]
            ok = len(oids) >= 3
            results.append(("Multi-SKU merge", "PASS" if ok else "FAIL", f"{len(oids)} orders, multi={multi}"))
            print(f"  {'+' if ok else '!'} {len(oids)} orders, multi-SKU: {multi}")
        else:
            results.append(("Multi-SKU", "FAIL", ""))

    # 7 — Process Chain
    print("\n[7] Process Chain...")
    if bid:
        c, d = req("POST", f"/order-imports/{bid}/process-chain", token=tok)
        if c == 200:
            r = d["data"]
            loc, exc, cf = r["ledger_rebuilt_count"], r["exception_generated_count"], r["chain_failure_count"]
            ok = loc >= 1
            results.append(("Process Chain", "PASS" if ok else "FAIL", f"ledger={loc}, exc={exc}, fail={cf}"))
            print(f"  {'+' if ok else '!'} ledger={loc}, exc={exc}, fail={cf}")
        else:
            results.append(("Process Chain", "FAIL", ""))

    # 8 — Shipping Bill Import
    print("\n[8] Shipping Bill Import...")
    csv_path = os.path.join(CSV_DIR, "shipping_bill_demo.csv")
    with open(csv_path, "rb") as f: csv_content = f.read()
    c, d = req("POST", "/shipping/bills/import",
        files={"file": ("bills.csv", csv_content, "text/csv")}, token=tok)
    if c == 200:
        r = d["data"]
        bid2 = r.get("batch_id")
        results.append(("Shipping Bill Import", "PASS", f"rows={r['total_rows']}"))
        print(f"  + rows={r['total_rows']}")
    else:
        results.append(("Shipping Bill Import", "FAIL", str(d)))
        print("  ! FAIL")
        bid2 = None

    # 9 — Shipping Reconcile
    print("\n[9] Shipping Reconcile...")
    if bid2:
        c, d = req("POST", f"/shipping/bills/{bid2}/reconcile", token=tok)
        if c == 200:
            r = d["data"]
            classified = (r["matched_count"] + r["mismatch_count"] + r["unmatched_count"]) >= 1
            status = "PASS" if r["matched_count"] >= 1 else ("WARN" if classified else "FAIL")
            results.append(("Shipping Reconcile", status,
                f"matched={r['matched_count']}, mismatch={r['mismatch_count']}, unmatched={r['unmatched_count']}"))
            print(f"  {'+' if status == 'PASS' else '?'} matched={r['matched_count']}, mismatch={r['mismatch_count']}, unmatched={r['unmatched_count']}")
        else:
            results.append(("Shipping Reconcile", "FAIL", ""))

    # 10 — Settlement Import
    print("\n[10] Settlement Import...")
    csv_path = os.path.join(CSV_DIR, "platform_settlement_demo.csv")
    with open(csv_path, "rb") as f: csv_content = f.read()
    c, d = req("POST", "/settlements/import",
        files={"file": ("settlements.csv", csv_content, "text/csv")}, token=tok)
    if c == 200:
        r = d["data"]
        results.append(("Settlement Import", "PASS", f"rows={r['total_rows']}"))
        print(f"  + rows={r['total_rows']}")
    else:
        results.append(("Settlement Import", "FAIL", str(d)))
        print("  ! FAIL")

    # 11 — Exceptions
    print("\n[11] Exceptions...")
    c, d = req("POST", "/exceptions/generate", token=tok)
    if c == 200:
        r = d["data"]
        ok = r["created_count"] >= 1
        results.append(("Exception Generate", "PASS" if ok else "FAIL", f"created={r['created_count']}"))
        print(f"  {'+' if ok else '!'} created={r['created_count']}")
    else:
        results.append(("Exception Generate", "FAIL", ""))
    c, d = req("GET", "/exceptions", token=tok)
    exc_list = d.get("data", d.get("items", d.get("records", [])))
    results.append(("Exception List", "PASS" if len(exc_list) >= 1 else "WARN", f"{len(exc_list)} items"))

    # 12 — Profit Dashboard
    print("\n[12] Profit Dashboard...")
    c, d = req("GET", "/finance/reports/profit-summary", token=tok)
    if c == 200:
        s = d.get("data", {})
        rev = s.get("total_revenue", 0)
        ok = rev > 0
        results.append(("Profit Summary", "PASS" if ok else "WARN", f"revenue={rev}"))
        print(f"  {'+' if ok else '!'} revenue={rev}")
    else:
        results.append(("Profit Summary", "FAIL", ""))
    
    c, d = req("GET", "/finance/reports/negative-profit-orders", token=tok)
    neg = d.get("data", d.get("items", d.get("records", [])))
    results.append(("Negative Profit Orders", "PASS" if len(neg) >= 1 else "WARN", f"{len(neg)} orders"))

    # 13 — Frontend build
    print("\n[13] Frontend Build...")
    r = subprocess.run(["npm", "run", "build"], capture_output=True, cwd=FRONTEND_DIR)
    ok = r.returncode == 0
    # Check for warnings
    out = r.stdout.decode() + r.stderr.decode()
    has_warnings = "warn" in out.lower() or "WARN" in out
    results.append(("Frontend Build", "PASS" if ok else "FAIL", "warnings detected" if has_warnings else "clean"))
    print(f"  {'+' if ok else '!'} build {'OK' if ok else 'FAILED'}")

    # Summary
    print("\n" + "=" * 65)
    print("  ACCEPTANCE SUMMARY")
    print("=" * 65)
    passed = sum(1 for r in results if r[1] == "PASS")
    failed = sum(1 for r in results if r[1] == "FAIL")
    warned = sum(1 for r in results if r[1] == "WARN")
    print(f"\n  PASS: {passed}  FAIL: {failed}  WARN: {warned}  TOTAL: {len(results)}\n")
    for name, status, detail in results:
        icon = {"PASS": "+", "FAIL": "!", "WARN": "?"}[status]
        print(f"  [{status:5s}] {icon} {name}")
        if detail: print(f"          {detail}")

if __name__ == "__main__":
    main()
