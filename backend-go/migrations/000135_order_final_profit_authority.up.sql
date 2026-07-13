CREATE TABLE order_product_cost_allocation (
 id BIGSERIAL PRIMARY KEY, owner_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT,
 order_id BIGINT NOT NULL REFERENCES sales_order(id) ON DELETE RESTRICT, order_item_id BIGINT NOT NULL REFERENCES sales_order_item(id) ON DELETE RESTRICT,
 sku_id BIGINT NOT NULL REFERENCES sku(id) ON DELETE RESTRICT, sourcing_cost_version_id BIGINT NOT NULL REFERENCES sourcing_cost_version(id) ON DELETE RESTRICT,
 amount_minor BIGINT NOT NULL CHECK(amount_minor>=0), currency CHAR(3) NOT NULL CHECK(currency ~ '^[A-Z]{3}$'), content_sha256 CHAR(64) NOT NULL CHECK(content_sha256 ~ '^[0-9a-f]{64}$'), created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(owner_id,order_item_id)
);
CREATE TABLE order_final_profit_version (
 id BIGSERIAL PRIMARY KEY, owner_id BIGINT NOT NULL REFERENCES "user"(id) ON DELETE RESTRICT, order_id BIGINT NOT NULL REFERENCES sales_order(id) ON DELETE RESTRICT,
 version BIGINT NOT NULL CHECK(version>0), currency CHAR(3) NOT NULL CHECK(currency ~ '^[A-Z]{3}$'),
 revenue_minor BIGINT NOT NULL CHECK(revenue_minor>0), product_cost_minor BIGINT NOT NULL CHECK(product_cost_minor>=0), settlement_fee_minor BIGINT NOT NULL CHECK(settlement_fee_minor>=0), fulfillment_fee_minor BIGINT NOT NULL CHECK(fulfillment_fee_minor>=0), refund_minor BIGINT NOT NULL CHECK(refund_minor>=0), total_cost_minor BIGINT NOT NULL CHECK(total_cost_minor>=0), profit_minor BIGINT NOT NULL,
 source_manifest_sha256 CHAR(64) NOT NULL CHECK(source_manifest_sha256 ~ '^[0-9a-f]{64}$'), finalized_at TIMESTAMPTZ NOT NULL,
 UNIQUE(owner_id,order_id,version), UNIQUE(owner_id,order_id,source_manifest_sha256),
 CHECK(total_cost_minor=product_cost_minor+settlement_fee_minor+fulfillment_fee_minor+refund_minor), CHECK(profit_minor=revenue_minor-total_cost_minor)
);
CREATE FUNCTION enforce_order_product_cost_allocation_authority() RETURNS trigger AS $$
DECLARE expected_minor BIGINT; expected_currency CHAR(3); line_sku BIGINT; invalid_lines BIGINT;
BEGIN
 IF (SELECT count(*) FROM platform_order_ingest WHERE owner_id=NEW.owner_id AND normalized_order_id=NEW.order_id AND event_action='reserve' AND truth_status='external_observed' AND processing_status='applied') <> 1 THEN RAISE EXCEPTION 'cost allocation lacks one external Owner order'; END IF;
 SELECT sku_id INTO line_sku FROM sales_order_item WHERE id=NEW.order_item_id AND order_id=NEW.order_id;
 IF line_sku IS NULL OR line_sku<>NEW.sku_id THEN RAISE EXCEPTION 'cost allocation order line mismatch'; END IF;
 SELECT cv.total_minor*oi.quantity,cv.target_currency INTO expected_minor,expected_currency FROM sourcing_cost_version cv JOIN sourcing_sku_mapping sm ON sm.id=cv.sku_mapping_id AND sm.owner_id=cv.owner_id JOIN sales_order_item oi ON oi.id=NEW.order_item_id WHERE cv.id=NEW.sourcing_cost_version_id AND cv.owner_id=NEW.owner_id AND sm.internal_sku_id=NEW.sku_id;
 IF expected_minor IS NULL OR expected_minor<>NEW.amount_minor OR expected_currency<>NEW.currency THEN RAISE EXCEPTION 'cost allocation is not exact'; END IF;
 SELECT count(*) INTO invalid_lines FROM sourcing_cost_line WHERE cost_version_id=NEW.sourcing_cost_version_id AND truth_status<>'actual';
 IF invalid_lines<>0 OR (SELECT count(*) FROM sourcing_cost_line WHERE cost_version_id=NEW.sourcing_cost_version_id)=0 THEN RAISE EXCEPTION 'final cost requires all actual evidence'; END IF;
 RETURN NEW;
END; $$ LANGUAGE plpgsql;
CREATE TRIGGER trg_order_product_cost_allocation_authority BEFORE INSERT ON order_product_cost_allocation FOR EACH ROW EXECUTE FUNCTION enforce_order_product_cost_allocation_authority();
CREATE FUNCTION enforce_order_final_profit_authority() RETURNS trigger AS $$
DECLARE line_count BIGINT; allocation_count BIGINT; sale_total BIGINT; fee_total BIGINT; fulfillment_total BIGINT; settlement_refund BIGINT; receipt_refund BIGINT; unresolved BIGINT; settlement_currency_count BIGINT;
BEGIN
 IF (SELECT count(*) FROM platform_order_ingest WHERE owner_id=NEW.owner_id AND normalized_order_id=NEW.order_id AND event_action='reserve' AND truth_status='external_observed' AND processing_status='applied') <> 1 THEN RAISE EXCEPTION 'final profit lacks one external Owner order'; END IF;
 SELECT count(*) INTO line_count FROM sales_order_item WHERE order_id=NEW.order_id;
 SELECT count(*) INTO allocation_count FROM order_product_cost_allocation WHERE owner_id=NEW.owner_id AND order_id=NEW.order_id AND currency=NEW.currency;
 IF line_count=0 OR allocation_count<>line_count OR (SELECT COALESCE(sum(amount_minor),0) FROM order_product_cost_allocation WHERE owner_id=NEW.owner_id AND order_id=NEW.order_id)<>NEW.product_cost_minor THEN RAISE EXCEPTION 'final profit cost allocation incomplete'; END IF;
 SELECT COALESCE(sum(CASE WHEN l.kind='sale' THEN l.amount_minor ELSE 0 END),0), COALESCE(sum(CASE WHEN l.kind IN('fee','commission') AND l.fee_code<>'fulfillment_fee' THEN l.amount_minor ELSE 0 END),0), COALESCE(sum(CASE WHEN l.kind IN('fee','commission') AND l.fee_code='fulfillment_fee' THEN l.amount_minor ELSE 0 END),0), COALESCE(sum(CASE WHEN l.kind='refund' THEN l.amount_minor ELSE 0 END),0), count(DISTINCT l.currency)
 INTO sale_total,fee_total,fulfillment_total,settlement_refund,settlement_currency_count
 FROM platform_settlement_fact_line l JOIN platform_settlement_ingest i ON i.id=l.ingest_id WHERE i.owner_id=NEW.owner_id AND i.truth_status='external_observed' AND l.order_id=NEW.order_id;
 IF settlement_currency_count<>1 OR NOT EXISTS(SELECT 1 FROM platform_settlement_fact_line l JOIN platform_settlement_ingest i ON i.id=l.ingest_id WHERE i.owner_id=NEW.owner_id AND i.truth_status='external_observed' AND l.order_id=NEW.order_id AND l.kind IN('fee','commission') AND l.fee_code='fulfillment_fee') OR sale_total<>NEW.revenue_minor OR fee_total<>NEW.settlement_fee_minor OR fulfillment_total<>NEW.fulfillment_fee_minor THEN RAISE EXCEPTION 'final profit settlement facts mismatch'; END IF;
 SELECT count(*) INTO unresolved FROM aftersales_resolution_case WHERE owner_id=NEW.owner_id AND order_id=NEW.order_id AND status NOT IN('succeeded','failed','rejected');
 SELECT COALESCE(sum(r.actual_minor),0) INTO receipt_refund FROM aftersales_resolution_case c JOIN aftersales_resolution_receipt r ON r.resolution_id=c.id AND r.owner_id=c.owner_id WHERE c.owner_id=NEW.owner_id AND c.order_id=NEW.order_id AND c.status='succeeded' AND r.outcome='succeeded' AND r.currency=NEW.currency AND r.source_type IN('platform_receipt','controlled_reconciliation');
 IF unresolved<>0 OR receipt_refund<>settlement_refund OR receipt_refund<>NEW.refund_minor THEN RAISE EXCEPTION 'final profit refund terminal facts mismatch'; END IF;
 RETURN NEW;
END; $$ LANGUAGE plpgsql;
CREATE TRIGGER trg_order_final_profit_authority BEFORE INSERT ON order_final_profit_version FOR EACH ROW EXECUTE FUNCTION enforce_order_final_profit_authority();
CREATE FUNCTION reject_order_profit_authority_mutation() RETURNS trigger AS $$ BEGIN RAISE EXCEPTION 'order final profit authority is immutable'; END; $$ LANGUAGE plpgsql;
CREATE TRIGGER trg_order_product_cost_allocation_immutable BEFORE UPDATE OR DELETE ON order_product_cost_allocation FOR EACH ROW EXECUTE FUNCTION reject_order_profit_authority_mutation();
CREATE TRIGGER trg_order_final_profit_version_immutable BEFORE UPDATE OR DELETE ON order_final_profit_version FOR EACH ROW EXECUTE FUNCTION reject_order_profit_authority_mutation();
