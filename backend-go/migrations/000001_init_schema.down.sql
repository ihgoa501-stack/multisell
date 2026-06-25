-- Reverse of 000001_init_schema.up.sql
-- Drop tables in reverse dependency order (children first, parents last)

-- Section 5 children
DROP TABLE IF EXISTS agentos_outcome_review;
DROP TABLE IF EXISTS agentos_command_execution;
DROP TABLE IF EXISTS agentos_approval_request;
DROP TABLE IF EXISTS listing_task_item;
DROP TABLE IF EXISTS order_import_item;
DROP TABLE IF EXISTS import_batch_row;
DROP TABLE IF EXISTS shipping_bill_item;
DROP TABLE IF EXISTS platform_settlement_item;
DROP TABLE IF EXISTS cost_allocation_item;
DROP TABLE IF EXISTS finance_transaction;
DROP TABLE IF EXISTS finance_ledger_entry;
DROP TABLE IF EXISTS settlement_item;
DROP TABLE IF EXISTS after_sales_order;
DROP TABLE IF EXISTS sales_order_shipping_snapshot;
DROP TABLE IF EXISTS sales_order_status_log;
DROP TABLE IF EXISTS sales_order_item;

-- Section 4
DROP TABLE IF EXISTS listing_task;
DROP TABLE IF EXISTS sales_order;
DROP TABLE IF EXISTS shipping_quote_rule;
DROP TABLE IF EXISTS inventory_warehouse;
DROP TABLE IF EXISTS inventory_log;
DROP TABLE IF EXISTS inventory;
DROP TABLE IF EXISTS price_change_log;
DROP TABLE IF EXISTS price;
DROP TABLE IF EXISTS spec_value;

-- Section 3
DROP TABLE IF EXISTS rule_conflict;
DROP TABLE IF EXISTS agent_pending_action;
DROP TABLE IF EXISTS shipping_zone;
DROP TABLE IF EXISTS product_canvases;
DROP TABLE IF EXISTS product_image_gen;
DROP TABLE IF EXISTS product_listing;
DROP TABLE IF EXISTS sourcing_1688_product;
DROP TABLE IF EXISTS product_supplier;
DROP TABLE IF EXISTS sku;
DROP TABLE IF EXISTS spec_name;

-- Section 2
DROP TABLE IF EXISTS allocation_rule;
DROP TABLE IF EXISTS agent_action;
DROP TABLE IF EXISTS shipping_bill_batch;
DROP TABLE IF EXISTS settlement;
DROP TABLE IF EXISTS order_import;
DROP TABLE IF EXISTS platform_fee_rule;
DROP TABLE IF EXISTS platform_attribute_mapping;
DROP TABLE IF EXISTS platform_category_mapping;
DROP TABLE IF EXISTS platform_integration_account;
DROP TABLE IF EXISTS role_permission;
DROP TABLE IF EXISTS user_role;
DROP TABLE IF EXISTS stores;
DROP TABLE IF EXISTS notification;
DROP TABLE IF EXISTS agent_nudge;
DROP TABLE IF EXISTS agent_evolution_config;
DROP TABLE IF EXISTS spc_control_limit;
DROP TABLE IF EXISTS personal_rule;
DROP TABLE IF EXISTS agent_decision;
DROP TABLE IF EXISTS honcho_profile;
DROP TABLE IF EXISTS shipping_channel;
DROP TABLE IF EXISTS product;

-- Section 1
DROP TABLE IF EXISTS agentos_action_proposal;
DROP TABLE IF EXISTS prompt_template;
DROP TABLE IF EXISTS order_import_batch;
DROP TABLE IF EXISTS import_batch;
DROP TABLE IF EXISTS cost_allocation_batch;
DROP TABLE IF EXISTS platform_settlement_batch;
DROP TABLE IF EXISTS agentos_operation_log;
DROP TABLE IF EXISTS rule_mark_change;
DROP TABLE IF EXISTS agent_episode;
DROP TABLE IF EXISTS exception_item;
DROP TABLE IF EXISTS operation_log;
DROP TABLE IF EXISTS exchange_rate;
DROP TABLE IF EXISTS system_config;
DROP TABLE IF EXISTS alert_rule;
DROP TABLE IF EXISTS finance_account;
DROP TABLE IF EXISTS permission;
DROP TABLE IF EXISTS role;
DROP TABLE IF EXISTS "user";
DROP TABLE IF EXISTS warehouse;
DROP TABLE IF EXISTS supplier;
DROP TABLE IF EXISTS shipping_provider;
DROP TABLE IF EXISTS platform;
DROP TABLE IF EXISTS brand;
DROP TABLE IF EXISTS category;

-- End of init schema downgrade
