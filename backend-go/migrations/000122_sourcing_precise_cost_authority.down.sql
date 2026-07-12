DROP TRIGGER IF EXISTS trg_sourcing_cost_version_complete ON sourcing_cost_version;
DROP FUNCTION IF EXISTS validate_sourcing_cost_version_complete();
DROP TRIGGER IF EXISTS trg_sourcing_cost_line_immutable ON sourcing_cost_line;
DROP TRIGGER IF EXISTS trg_sourcing_cost_version_immutable ON sourcing_cost_version;
DROP FUNCTION IF EXISTS reject_sourcing_cost_authority_mutation();
DROP TRIGGER IF EXISTS trg_sourcing_cost_line_currency ON sourcing_cost_line;
DROP FUNCTION IF EXISTS validate_sourcing_cost_line_currency();
DROP TABLE IF EXISTS sourcing_cost_line;
DROP TABLE IF EXISTS sourcing_cost_version;
