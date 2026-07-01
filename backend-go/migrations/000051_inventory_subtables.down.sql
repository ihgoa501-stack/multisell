-- ============================================================
-- Rollback migration 000051: Inventory subtables
-- Drops inventory_alert, inventory_alert_rule,
-- inventory_transfer, and bin_location tables.
-- ============================================================

DROP TABLE IF EXISTS bin_location;
DROP TABLE IF EXISTS inventory_transfer;
DROP TABLE IF EXISTS inventory_alert_rule;
DROP TABLE IF EXISTS inventory_alert;
