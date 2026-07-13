DROP TRIGGER IF EXISTS trg_cash_reconciliation_immutable ON cash_reconciliation;
DROP FUNCTION IF EXISTS guard_cash_reconciliation_immutable();
DROP TRIGGER IF EXISTS trg_cash_receipt_immutable ON cash_receipt;
DROP FUNCTION IF EXISTS guard_cash_receipt_immutable();
DROP TABLE IF EXISTS cash_reconciliation;
DROP TABLE IF EXISTS cash_receipt;
DROP INDEX IF EXISTS idx_finance_account_owner;
ALTER TABLE finance_account DROP COLUMN IF EXISTS owner_id;
