DROP TRIGGER IF EXISTS trg_validate_sourcing_compliance_authority ON sourcing_compliance_evidence;
DROP FUNCTION IF EXISTS validate_sourcing_compliance_authority();
DROP TRIGGER IF EXISTS trg_protect_sourcing_compliance_evidence ON sourcing_compliance_evidence;
DROP FUNCTION IF EXISTS protect_sourcing_compliance_evidence();
DROP TABLE IF EXISTS sourcing_compliance_evidence;
