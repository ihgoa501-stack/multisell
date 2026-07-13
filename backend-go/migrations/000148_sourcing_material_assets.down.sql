DROP TRIGGER IF EXISTS sourcing_material_rights_content_immutable ON sourcing_material_rights_evidence;
DROP FUNCTION IF EXISTS protect_sourcing_material_rights_content();
DROP TRIGGER IF EXISTS sourcing_material_source_immutable ON sourcing_material_asset;
DROP FUNCTION IF EXISTS protect_sourcing_material_source();
DROP TABLE IF EXISTS sourcing_material_rendition;
DROP TABLE IF EXISTS sourcing_material_rights_evidence;
DROP TABLE IF EXISTS sourcing_material_asset;
