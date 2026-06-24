-- ============================================================
-- LingMirror data migration: legacy schema → new schema
-- Source: existing multisell production DB (legacy Python stack)
-- Target: new schema defined in 000001_init_schema.up.sql
-- Usage: psql -d multisell_new -f 000003_data_migration.up.sql
--
-- This migration assumes:
--   1. The new schema (000001 + 000002) has been applied.
--   2. The legacy tables are still present in the same DB OR have been
--      dumped into this DB under their original names. Column names match
--      the legacy SQLAlchemy models (app/models.py).
--   3. Where table names changed (e.g. none in this refactor — we kept the
--      legacy names like `sales_order`, `after_sales_order`, `sourcing_1688_product`),
--      rows are inserted directly.
--
-- Strategy: INSERT ... ON CONFLICT (id) DO NOTHING so the script is idempotent.
-- Validation happens in a separate step (see validate.sql).
-- ============================================================

BEGIN;

-- Reference / root tables (no FK dependencies)
INSERT INTO category (id, name, parent_id, level, sort_order, status, created_at, updated_at)
SELECT id, name, parent_id, level, sort_order, status, created_at, updated_at
FROM legacy_category
ON CONFLICT (id) DO NOTHING;

INSERT INTO brand (id, name, logo, status, sort_order, created_at, updated_at)
SELECT id, name, logo, status, sort_order, created_at, updated_at
FROM legacy_brand
ON CONFLICT (id) DO NOTHING;

INSERT INTO platform (id, name, code, extra_config, status, created_at, updated_at)
SELECT id, name, code, extra_config, status, created_at, updated_at
FROM legacy_platform
ON CONFLICT (id) DO NOTHING;

INSERT INTO "user" (id, email, username, display_name, password_hash, status, created_at, updated_at)
SELECT id, email, username, display_name, password_hash, status, created_at, updated_at
FROM legacy_user
ON CONFLICT (id) DO NOTHING;

INSERT INTO warehouse (id, name, code, address, contact, phone, is_default, status, created_at, updated_at)
SELECT id, name, code, address, contact, phone, is_default, status, created_at, updated_at
FROM legacy_warehouse
ON CONFLICT (id) DO NOTHING;

-- Products + specs
INSERT INTO product (id, name, subtitle, description, brand_id, category_id, unit, status,
    main_image, images, product_length_cm, product_width_cm, product_height_cm, product_weight_kg,
    package_length_cm, package_width_cm, package_height_cm, package_weight_kg, cargo_type,
    ai_title, ai_description, seo_keywords, ai_status, platform_statuses, created_at, updated_at)
SELECT id, name, subtitle, description, brand_id, category_id, unit, status,
    main_image, images, product_length_cm, product_width_cm, product_height_cm, product_weight_kg,
    package_length_cm, package_width_cm, package_height_cm, package_weight_kg, cargo_type,
    ai_title, ai_description, seo_keywords, ai_status, platform_statuses, created_at, updated_at
FROM legacy_product
ON CONFLICT (id) DO NOTHING;

INSERT INTO spec_name (id, product_id, name, sort_order, created_at)
SELECT id, product_id, name, sort_order, created_at FROM legacy_spec_name
ON CONFLICT (id) DO NOTHING;

INSERT INTO spec_value (id, spec_name_id, product_id, value, sort_order, created_at)
SELECT id, spec_name_id, product_id, value, sort_order, created_at FROM legacy_spec_value
ON CONFLICT (id) DO NOTHING;

-- SKU + price + inventory
INSERT INTO sku (id, product_id, code, barcode, spec_desc, spec_values, price, cost_price, market_price,
    stock, lock_stock, warning_stock, weight, sku_length_cm, sku_width_cm, sku_height_cm, sku_weight_kg,
    image, status, created_at, updated_at)
SELECT id, product_id, code, barcode, spec_desc, spec_values, price, cost_price, market_price,
    stock, lock_stock, warning_stock, weight, sku_length_cm, sku_width_cm, sku_height_cm, sku_weight_kg,
    image, status, created_at, updated_at
FROM legacy_sku
ON CONFLICT (id) DO NOTHING;

INSERT INTO price (id, sku_id, price_type, price, start_time, end_time, status, created_at, updated_at)
SELECT id, sku_id, price_type, price, start_time, end_time, status, created_at, updated_at
FROM legacy_price
ON CONFLICT (id) DO NOTHING;

INSERT INTO inventory (id, sku_id, warehouse, location, quantity, locked_quantity, safety_stock, created_at, updated_at)
SELECT id, sku_id, warehouse, location, quantity, locked_quantity, safety_stock, created_at, updated_at
FROM legacy_inventory
ON CONFLICT (id) DO NOTHING;

-- Supplier + sourcing
INSERT INTO supplier (id, name, contact_person, contact_phone, email, address, remark, status, created_at, updated_at)
SELECT id, name, contact_person, contact_phone, email, address, remark, status, created_at, updated_at
FROM legacy_supplier
ON CONFLICT (id) DO NOTHING;

INSERT INTO sourcing_1688_product (id, source_url, title, price, moq, supplier_name, shop_url, shop_location, images, attributes, sku_variants, description, package_length_cm, package_width_cm, package_height_cm, package_weight_kg, raw_data, status, product_id, created_at, updated_at)
SELECT id, source_url, title, price, moq, supplier_name, shop_url, shop_location, images, attributes, sku_variants, description, package_length_cm, package_width_cm, package_height_cm, package_weight_kg, raw_data, status, product_id, created_at, updated_at
FROM legacy_sourcing_1688_product
ON CONFLICT (id) DO NOTHING;

-- Shipping
INSERT INTO shipping_provider (id, name, code, contact, phone, remark, status, created_at, updated_at)
SELECT id, name, code, contact, phone, remark, status, created_at, updated_at
FROM legacy_shipping_provider
ON CONFLICT (id) DO NOTHING;

INSERT INTO shipping_channel (id, provider_id, name, code, volumetric_divisor, cargo_types,
    estimated_delivery_min, estimated_delivery_max, currency, sort_order, status, created_at, updated_at)
SELECT id, provider_id, name, code, volumetric_divisor, cargo_types,
    estimated_delivery_min, estimated_delivery_max, currency, sort_order, status, created_at, updated_at
FROM legacy_shipping_channel
ON CONFLICT (id) DO NOTHING;

INSERT INTO shipping_zone (id, channel_id, country_code, postal_code_from, postal_code_to, status, created_at, updated_at)
SELECT id, channel_id, country_code, postal_code_from, postal_code_to, status, created_at, updated_at
FROM legacy_shipping_zone
ON CONFLICT (id) DO NOTHING;

INSERT INTO shipping_quote_rule (id, channel_id, zone_id, rule_type, priority, min_weight_kg, max_weight_kg,
    first_kg, first_price, additional_kg, additional_price, fixed_fee, per_kg_price, minimum_charge,
    tier_config, surcharge_fixed, fuel_surcharge_pct, rounding_increment, remark, status, created_at, updated_at)
SELECT id, channel_id, zone_id, rule_type, priority, min_weight_kg, max_weight_kg,
    first_kg, first_price, additional_kg, additional_price, fixed_fee, per_kg_price, minimum_charge,
    tier_config, surcharge_fixed, fuel_surcharge_pct, rounding_increment, remark, status, created_at, updated_at
FROM legacy_shipping_quote_rule
ON CONFLICT (id) DO NOTHING;

-- Orders + items + logs + snapshots
INSERT INTO sales_order (id, order_no, platform_id, status, tracking_number, recipient_name, recipient_phone,
    shipping_address, total_amount, shipping_fee, pay_amount, platform_fee, payment_fee, other_fee,
    product_cost, profit_amount, profit_margin, payment_method, remark, paid_at, shipped_at, delivered_at,
    cancelled_at, created_at, updated_at)
SELECT id, order_no, platform_id, status, tracking_number, recipient_name, recipient_phone,
    shipping_address, total_amount, shipping_fee, pay_amount, platform_fee, payment_fee, other_fee,
    product_cost, profit_amount, profit_margin, payment_method, remark, paid_at, shipped_at, delivered_at,
    cancelled_at, created_at, updated_at
FROM legacy_sales_order
ON CONFLICT (id) DO NOTHING;

INSERT INTO sales_order_item (id, order_id, sku_id, product_id, product_name, sku_code, spec_desc,
    unit_price, quantity, subtotal, created_at)
SELECT id, order_id, sku_id, product_id, product_name, sku_code, spec_desc,
    unit_price, quantity, subtotal, created_at
FROM legacy_sales_order_item
ON CONFLICT (id) DO NOTHING;

INSERT INTO sales_order_status_log (id, order_id, from_status, to_status, operator, remark, created_at)
SELECT id, order_id, from_status, to_status, operator, remark, created_at
FROM legacy_sales_order_status_log
ON CONFLICT (id) DO NOTHING;

INSERT INTO sales_order_shipping_snapshot (id, order_id, sku_id, quantity, destination_country, postal_code,
    cargo_type, package_source, package_length_cm, package_width_cm, package_height_cm, package_weight_kg,
    provider_id, provider_name, channel_id, channel_name, chargeable_weight_kg, base_shipping_fee,
    surcharge_fee, fuel_surcharge_fee, total_shipping_fee, currency, calculation_detail, created_at)
SELECT id, order_id, sku_id, quantity, destination_country, postal_code,
    cargo_type, package_source, package_length_cm, package_width_cm, package_height_cm, package_weight_kg,
    provider_id, provider_name, channel_id, channel_name, chargeable_weight_kg, base_shipping_fee,
    surcharge_fee, fuel_surcharge_fee, total_shipping_fee, currency, calculation_detail, created_at
FROM legacy_sales_order_shipping_snapshot
ON CONFLICT (id) DO NOTHING;

-- Listing + tasks
INSERT INTO product_listing (id, product_id, platform_id, platform_product_id, platform_sku, status, platform_url, sync_message, published_data, last_sync_at, created_at, updated_at)
SELECT id, product_id, platform_id, platform_product_id, platform_sku, status, platform_url, sync_message, published_data, last_sync_at, created_at, updated_at
FROM legacy_product_listing
ON CONFLICT (id) DO NOTHING;

INSERT INTO listing_task (id, product_id, platform_id, sku_id, product_listing_id, source_type, source_item_key, status, missing_requirements, decision_snapshot, target_sale_price, target_profit_margin, destination_country, last_error, created_by, updated_by, created_at, updated_at)
SELECT id, product_id, platform_id, sku_id, product_listing_id, source_type, source_item_key, status, missing_requirements, decision_snapshot, target_sale_price, target_profit_margin, destination_country, last_error, created_by, updated_by, created_at, updated_at
FROM legacy_listing_task
ON CONFLICT (id) DO NOTHING;

-- Settlement + items
INSERT INTO settlement (id, platform_id, settlement_no, period_start, period_end, currency,
    total_revenue, total_fee, total_refund, total_net, status, raw_data, imported_at, created_at, updated_at)
SELECT id, platform_id, settlement_no, period_start, period_end, currency,
    total_revenue, total_fee, total_refund, total_net, status, raw_data, imported_at, created_at, updated_at
FROM legacy_settlement
ON CONFLICT (id) DO NOTHING;

INSERT INTO settlement_item (id, settlement_id, transaction_type, transaction_id, order_no, order_id, sku_id,
    amount, fee, net, quantity, occurred_at, created_at, reconciliation_status, reconciliation_note,
    reconciled_at, reconciled_by)
SELECT id, settlement_id, transaction_type, transaction_id, order_no, order_id, sku_id,
    amount, fee, net, quantity, occurred_at, created_at, reconciliation_status, reconciliation_note,
    reconciled_at, reconciled_by
FROM legacy_settlement_item
ON CONFLICT (id) DO NOTHING;

-- Finance
INSERT INTO finance_account (id, name, account_type, platform_id, currency, balance, status, created_at, updated_at)
SELECT id, name, account_type, platform_id, currency, balance, status, created_at, updated_at
FROM legacy_finance_account
ON CONFLICT (id) DO NOTHING;

INSERT INTO finance_transaction (id, account_id, transaction_type, amount, currency, order_id, settlement_id,
    platform_id, description, transaction_date, created_at)
SELECT id, account_id, transaction_type, amount, currency, order_id, settlement_id,
    platform_id, description, transaction_date, created_at
FROM legacy_finance_transaction
ON CONFLICT (id) DO NOTHING;

INSERT INTO finance_ledger_entry (id, order_id, entry_type, amount, currency, cost_layer, source_type,
    source_id, description, created_at)
SELECT id, order_id, entry_type, amount, currency, cost_layer, source_type,
    source_id, description, created_at
FROM legacy_finance_ledger_entry
ON CONFLICT (id) DO NOTHING;

-- Platform fee rules
INSERT INTO platform_fee_rule (id, platform_id, country_code, category_id, fee_type, fee_rate_pct,
    fixed_amount, min_amount, max_amount, currency, effective_from, effective_to, priority, status,
    remark, created_at, updated_at)
SELECT id, platform_id, country_code, category_id, fee_type, fee_rate_pct,
    fixed_amount, min_amount, max_amount, currency, effective_from, effective_to, priority, status,
    remark, created_at, updated_at
FROM legacy_platform_fee_rule
ON CONFLICT (id) DO NOTHING;

-- Aftersales
INSERT INTO after_sales_order (id, order_id, item_id, sku_id, return_quantity, reason, status,
    refund_amount, inspection_result, rejection_reason, created_by, approved_by, approved_at,
    rejected_by, rejected_at, received_by, received_at, refunded_by, refunded_at, created_at, updated_at)
SELECT id, order_id, item_id, sku_id, return_quantity, reason, status,
    refund_amount, inspection_result, rejection_reason, created_by, approved_by, approved_at,
    rejected_by, rejected_at, received_by, received_at, refunded_by, refunded_at, created_at, updated_at
FROM legacy_after_sales_order
ON CONFLICT (id) DO NOTHING;

-- Import batches + rows
INSERT INTO import_batch (id, type, file_name, total_rows, success_count, error_count, error_summary,
    status, created_by, created_at, updated_at)
SELECT id, type, file_name, total_rows, success_count, error_count, error_summary,
    status, created_by, created_at, updated_at
FROM legacy_import_batch
ON CONFLICT (id) DO NOTHING;

INSERT INTO order_import (id, platform_id, source_type, file_name, total_rows, success_count, error_count,
    error_detail, status, created_by, created_at, updated_at)
SELECT id, platform_id, source_type, file_name, total_rows, success_count, error_count,
    error_detail, status, created_by, created_at, updated_at
FROM legacy_order_import
ON CONFLICT (id) DO NOTHING;

-- Notifications + alert rules
INSERT INTO notification (id, user_id, alert_type, title, content, link_url, severity, is_read, source_id, created_at)
SELECT id, user_id, alert_type, title, content, link_url, severity, is_read, source_id, created_at
FROM legacy_notification
ON CONFLICT (id) DO NOTHING;

INSERT INTO alert_rule (id, name, alert_type, enabled, config, description, created_at, updated_at)
SELECT id, name, alert_type, enabled, config, description, created_at, updated_at
FROM legacy_alert_rule
ON CONFLICT (id) DO NOTHING;

-- Exceptions
INSERT INTO exception_item (id, source_module, source_type, source_id, severity, status, title, description,
    recommended_action, assigned_to, resolved_at, resolved_by, note, created_at, updated_at)
SELECT id, source_module, source_type, source_id, severity, status, title, description,
    recommended_action, assigned_to, resolved_at, resolved_by, note, created_at, updated_at
FROM legacy_exception_item
ON CONFLICT (id) DO NOTHING;

-- Operation logs (read-only history; migrate as-is)
INSERT INTO operation_log (id, module, action, resource_id, content, operator, ip, duration, created_at)
SELECT id, module, action, resource_id, content, operator, ip, duration, created_at
FROM legacy_operation_log
ON CONFLICT (id) DO NOTHING;

-- Platform integrations + mappings
INSERT INTO platform_integration_account (id, platform_id, adapter_code, account_name, status, credential_metadata, created_by, updated_by, created_at, updated_at)
SELECT id, platform_id, adapter_code, account_name, status, credential_metadata, created_by, updated_by, created_at, updated_at
FROM legacy_platform_integration_account
ON CONFLICT (id) DO NOTHING;

-- Allocation
INSERT INTO allocation_rule (id, name, priority, rule_type, warehouse_id, allocation_pct, allocation_qty, status, created_at, updated_at)
SELECT id, name, priority, rule_type, warehouse_id, allocation_pct, allocation_qty, status, created_at, updated_at
FROM legacy_allocation_rule
ON CONFLICT (id) DO NOTHING;

INSERT INTO cost_allocation_batch (id, allocation_type, allocation_method, total_amount, currency, source_filename, row_count, status, posted_count, created_by, created_at, updated_at)
SELECT id, allocation_type, allocation_method, total_amount, currency, source_filename, row_count, status, posted_count, created_by, created_at, updated_at
FROM legacy_cost_allocation_batch
ON CONFLICT (id) DO NOTHING;

-- Image gen
INSERT INTO product_image_gen (id, product_id, prompt, style, negative_prompt, size, requested_count, status, image_urls, error_message, created_by, batch_id, created_at, updated_at)
SELECT id, product_id, prompt, style, negative_prompt, size, requested_count, status, image_urls, error_message, created_by, batch_id, created_at, updated_at
FROM legacy_product_image_gen
ON CONFLICT (id) DO NOTHING;

COMMIT;
