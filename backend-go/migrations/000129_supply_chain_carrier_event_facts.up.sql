ALTER TABLE supply_chain_tracking
    ADD COLUMN IF NOT EXISTS owner_id BIGINT REFERENCES "user"(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_supply_chain_tracking_owner
    ON supply_chain_tracking(owner_id, created_at DESC);

CREATE OR REPLACE FUNCTION protect_supply_chain_tracking_owner()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.owner_id IS NOT NULL AND NEW.owner_id IS DISTINCT FROM OLD.owner_id THEN
        RAISE EXCEPTION 'tracking owner is immutable';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
CREATE TRIGGER trg_supply_chain_tracking_owner_immutable
BEFORE UPDATE OF owner_id ON supply_chain_tracking
FOR EACH ROW EXECUTE FUNCTION protect_supply_chain_tracking_owner();

CREATE TABLE IF NOT EXISTS supply_chain_carrier_event (
    id BIGSERIAL PRIMARY KEY,
    owner_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
    tracking_id UUID NOT NULL REFERENCES supply_chain_tracking(id) ON DELETE RESTRICT,
    source_system VARCHAR(80) NOT NULL,
    external_event_id VARCHAR(200) NOT NULL,
    status VARCHAR(30) NOT NULL CHECK (status IN ('pending','picked_up','outbound','transit','customs','last_mile','delivered','exception')),
    occurred_at TIMESTAMPTZ NOT NULL,
    observed_at TIMESTAMPTZ NOT NULL CHECK (observed_at >= occurred_at),
    location VARCHAR(300) NOT NULL DEFAULT '',
    message TEXT NOT NULL DEFAULT '',
    raw_payload JSONB NOT NULL,
    payload_sha256 CHAR(64) NOT NULL CHECK (payload_sha256 ~ '^[0-9a-f]{64}$'),
    truth_status VARCHAR(30) NOT NULL CHECK (truth_status = 'external_observed'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (owner_id, source_system, external_event_id)
);

CREATE INDEX IF NOT EXISTS idx_carrier_event_owner_tracking_time
    ON supply_chain_carrier_event(owner_id, tracking_id, occurred_at, id);

CREATE OR REPLACE FUNCTION enforce_carrier_event_owner_and_immutability()
RETURNS TRIGGER AS $$
DECLARE tracking_owner BIGINT;
BEGIN
    IF TG_OP <> 'INSERT' THEN
        RAISE EXCEPTION 'carrier events are immutable';
    END IF;
    SELECT owner_id INTO tracking_owner FROM supply_chain_tracking WHERE id = NEW.tracking_id;
    IF tracking_owner IS NULL OR tracking_owner <> NEW.owner_id THEN
        RAISE EXCEPTION 'carrier event owner does not match tracking owner';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_carrier_event_owner_immutable ON supply_chain_carrier_event;
CREATE TRIGGER trg_carrier_event_owner_immutable
BEFORE INSERT OR UPDATE OR DELETE ON supply_chain_carrier_event
FOR EACH ROW EXECUTE FUNCTION enforce_carrier_event_owner_and_immutability();
