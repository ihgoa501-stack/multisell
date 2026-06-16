# CSV Order Import Smoke Checklist

This checklist verifies that imported external orders enter the operating chain:

CSV import -> sales order creation -> ledger rebuild -> exception generation -> visible processing result.

Use it after changes that touch order import, order creation, finance ledger, exception generation, permissions, or the order import frontend.

## Preconditions

- Backend is running against a migrated local database.
- Frontend is running against the same backend.
- A user can log in with these permissions:
  - `order_import:import`
  - `order_import:view`
  - `order_import:process`
  - `order:view`
  - `finance:ledger:rebuild`
  - `exception:view`
- Test SKUs exist with available inventory for `SKU-A` and `SKU-B`.
- Platform adapter `csv_order` exists and supports order import.

## Test CSV

Save this as a local CSV file, for example `/tmp/order-import-smoke.csv`.

```csv
platform,store_name,platform_order_no,order_no,sku_code,quantity,unit_price,currency,recipient_name,recipient_phone,country_code,shipping_address,shipping_fee,tracking_number,paid_at
amazon,US,AMZ-SMOKE-1001,,SKU-A,1,20,CNY,Alice,123,US,Street 1,5,TRK-SMOKE-1,2026-06-16
amazon,US,AMZ-SMOKE-1001,,SKU-B,2,15,CNY,Alice,123,US,Street 1,5,TRK-SMOKE-1,2026-06-16
amazon,US,AMZ-SMOKE-1002,,SKU-MISSING,1,10,CNY,Bob,456,US,Street 2,0,,2026-06-16
```

Expected import result:

- One system order is created for `AMZ-SMOKE-1001`.
- That order has two order items: `SKU-A` quantity `1`, and `SKU-B` quantity `2`.
- Both successful import rows reference the same `order_id`.
- The missing SKU row is marked `failed`.
- The batch remains `chain_pending` until chain processing is triggered.

## API Smoke Path

Run these requests with a token that has the required permissions.

1. Upload the CSV:

```bash
curl -X POST "http://localhost:8000/api/order-imports/csv" \
  -H "Authorization: Bearer $TOKEN" \
  -F "adapter_code=csv_order" \
  -F "file=@/tmp/order-import-smoke.csv"
```

Expected:

- HTTP 200.
- Response data includes `row_count=3`.
- Response data includes `created_order_count=1`.
- Response data includes `failed_count=1`.
- Response data includes `chain_status=chain_pending`.

2. Capture the returned batch id as `BATCH_ID`, then list items:

```bash
curl "http://localhost:8000/api/order-imports/$BATCH_ID/items" \
  -H "Authorization: Bearer $TOKEN"
```

Expected:

- Three import items are returned.
- The two `AMZ-SMOKE-1001` rows have the same non-empty `order_id`.
- The two `AMZ-SMOKE-1001` rows have `status=created_order` or `status=imported`.
- The `SKU-MISSING` row has `status=failed`.

3. Process the operating chain:

```bash
curl -X POST "http://localhost:8000/api/order-imports/$BATCH_ID/process-chain" \
  -H "Authorization: Bearer $TOKEN"
```

Expected:

- HTTP 200.
- Response data includes `ledger_rebuilt_count=1`.
- Response data includes `chain_failure_count=0`.
- `exception_generated_count` is present. The value depends on whether exception rules detect anything for the imported order.

4. Read the chain summary:

```bash
curl "http://localhost:8000/api/order-imports/$BATCH_ID/chain-summary" \
  -H "Authorization: Bearer $TOKEN"
```

Expected:

- Response data includes `chain_status=chain_processed`.
- Response data includes `ledger_rebuilt_count=1`.
- Response data includes item-level chain statuses.

## Frontend Smoke Path

1. Log in as a user with the required order import permissions.
2. Open the order import page.
3. Upload `/tmp/order-import-smoke.csv`.
4. Confirm the batch table shows:
   - Total rows: `3`.
   - Created orders: `1`.
   - Failed rows: `1`.
   - Chain status: `未处理`.
5. Open batch details.
6. Confirm the two `AMZ-SMOKE-1001` rows are visible and share the same system order id.
7. Click `处理链路`.
8. Confirm the batch table refreshes and shows:
   - Chain status: `已处理`.
   - Rebuilt ledger count: `1`.
   - Exception count column is populated.
9. Reopen batch details.
10. Confirm item chain status no longer shows all rows as `未处理`.

## Negative Checks

Run these when permissions or duplicate handling changed.

- Upload a non-CSV file. Expected: HTTP 400 with "仅支持 CSV 文件".
- Upload an empty CSV file. Expected: HTTP 400 with "空文件" or row validation failure.
- Re-upload the same `AMZ-SMOKE-1001` CSV. Expected: no duplicate sales order is created.
- Process a non-existent batch id. Expected: not found response.
- Process the batch without `order_import:process`. Expected: HTTP 403.

## Automated Verification

Run these commands before merging changes that affect this path:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test /Users/lc/multisell/backend/.venv/bin/python -m pytest tests/test_order_import_csv_adapter.py tests/test_order_import_operational_chain.py -q
```

For shared order, ledger, or exception changes, run the full backend suite:

```bash
cd backend
TEST_DATABASE_URL=postgresql+asyncpg://postgres:postgres@localhost:5432/product_management_test /Users/lc/multisell/backend/.venv/bin/python -m pytest -q
```

For frontend changes:

```bash
cd frontend
npm run build
```

## Sign-Off Record

Use this section when manually validating a release candidate.

```text
Date:
Branch / commit:
Tester:
Backend command result:
Frontend build result:
CSV upload result:
Chain processing result:
Known issues:
Decision: pass / fail
```
