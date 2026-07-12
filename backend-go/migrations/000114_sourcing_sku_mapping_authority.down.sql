DROP TRIGGER IF EXISTS trg_sourcing_sku_mapping_immutable ON sourcing_sku_mapping;
DROP FUNCTION IF EXISTS reject_sourcing_sku_mapping_mutation();
DROP TRIGGER IF EXISTS trg_validate_sourcing_sku_mapping_binding ON sourcing_sku_mapping;
DROP FUNCTION IF EXISTS validate_sourcing_sku_mapping_binding();
DROP TABLE IF EXISTS sourcing_sku_mapping;
