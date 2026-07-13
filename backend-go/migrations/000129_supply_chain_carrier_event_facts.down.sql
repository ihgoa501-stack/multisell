DROP TRIGGER IF EXISTS trg_carrier_event_owner_immutable ON supply_chain_carrier_event;
DROP FUNCTION IF EXISTS enforce_carrier_event_owner_and_immutability();
DROP TABLE IF EXISTS supply_chain_carrier_event;
DROP TRIGGER IF EXISTS trg_supply_chain_tracking_owner_immutable ON supply_chain_tracking;
DROP FUNCTION IF EXISTS protect_supply_chain_tracking_owner();
DROP INDEX IF EXISTS idx_supply_chain_tracking_owner;
ALTER TABLE supply_chain_tracking DROP COLUMN IF EXISTS owner_id;
