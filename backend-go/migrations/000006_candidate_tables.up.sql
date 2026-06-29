CREATE TABLE IF NOT EXISTS candidate_product (
    id BIGSERIAL PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    description TEXT DEFAULT '',
    main_image VARCHAR(500) DEFAULT '',
    images JSONB DEFAULT '[]',
    category_id BIGINT REFERENCES category(id) ON DELETE SET NULL,
    brand_id BIGINT REFERENCES brand(id) ON DELETE SET NULL,
    spec_json JSONB DEFAULT '{}',
    supplier_id BIGINT REFERENCES supplier(id) ON DELETE SET NULL,
    purchase_price NUMERIC(12,2) NOT NULL DEFAULT 0,
    purchase_currency VARCHAR(3) NOT NULL DEFAULT 'CNY',
    package_weight_kg NUMERIC(8,3) NOT NULL DEFAULT 0,
    package_length_cm NUMERIC(8,2) NOT NULL DEFAULT 0,
    package_width_cm NUMERIC(8,2) NOT NULL DEFAULT 0,
    package_height_cm NUMERIC(8,2) NOT NULL DEFAULT 0,
    hs_code VARCHAR(20) DEFAULT '',
    origin_country VARCHAR(3) NOT NULL DEFAULT 'CN',
    target_sale_price NUMERIC(12,2) NOT NULL DEFAULT 0,
    target_currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    target_platform_id BIGINT REFERENCES platform(id) ON DELETE SET NULL,
    destination_country VARCHAR(3) NOT NULL DEFAULT 'US',
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    is_seed_data BOOLEAN NOT NULL DEFAULT false,
    created_by VARCHAR(100) DEFAULT '',
    updated_by VARCHAR(100) DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_candidate_product_status ON candidate_product(status);
CREATE INDEX IF NOT EXISTS idx_candidate_product_seed ON candidate_product(is_seed_data);

CREATE TABLE IF NOT EXISTS completeness_check (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL,
    score NUMERIC(5,2) NOT NULL DEFAULT 0,
    missing_items TEXT DEFAULT '[]',
    score_breakdown TEXT DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'incomplete',
    triggered_by VARCHAR(100) DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_completeness_check_product ON completeness_check(product_id);
CREATE INDEX IF NOT EXISTS idx_completeness_check_status ON completeness_check(status);

CREATE TABLE IF NOT EXISTS profit_summary (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL,
    purchase_cost NUMERIC(12,2) NOT NULL DEFAULT 0,
    shipping_cost NUMERIC(12,2) NOT NULL DEFAULT 0,
    platform_fee NUMERIC(12,2) NOT NULL DEFAULT 0,
    tariff_cost NUMERIC(12,2) NOT NULL DEFAULT 0,
    other_cost NUMERIC(12,2) NOT NULL DEFAULT 0,
    total_cost NUMERIC(12,2) NOT NULL DEFAULT 0,
    target_revenue NUMERIC(12,2) NOT NULL DEFAULT 0,
    estimated_profit NUMERIC(12,2) NOT NULL DEFAULT 0,
    profit_margin NUMERIC(8,2) NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'unknown',
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    calculated_by VARCHAR(100) DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_profit_summary_product ON profit_summary(product_id);
CREATE INDEX IF NOT EXISTS idx_profit_summary_status ON profit_summary(status);

CREATE TABLE IF NOT EXISTS listing_recommendation (
    id BIGSERIAL PRIMARY KEY,
    product_id BIGINT NOT NULL,
    completeness_score NUMERIC(5,2) NOT NULL DEFAULT 0,
    profit_margin NUMERIC(8,2) NOT NULL DEFAULT 0,
    estimated_profit NUMERIC(12,2) NOT NULL DEFAULT 0,
    decision VARCHAR(20) NOT NULL,
    confidence NUMERIC(5,4) NOT NULL DEFAULT 0,
    reason TEXT DEFAULT '',
    risk_flags TEXT DEFAULT '[]',
    created_listing_task_id BIGINT,
    triggered_by VARCHAR(100) DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_listing_recommendation_product ON listing_recommendation(product_id);
CREATE INDEX IF NOT EXISTS idx_listing_recommendation_decision ON listing_recommendation(decision);

INSERT INTO candidate_product (title, description, main_image, purchase_price, purchase_currency, package_weight_kg, package_length_cm, package_width_cm, package_height_cm, hs_code, target_sale_price, target_currency, destination_country, status, is_seed_data)
VALUES
('Wireless Bluetooth Earbuds TWS Noise Cancelling', 'High-quality wireless Bluetooth 5.3 earbuds with active noise cancellation, IPX5 waterproof, 24h battery life, comfortable ergonomic fit for sports and daily use.', '/seeds/earbuds.jpg', 35.00, 'CNY', 0.08, 10, 8, 4, '85183000', 19.99, 'USD', 'US', 'draft', true),
('Smart Watch Fitness Tracker with Heart Rate Monitor', '1.69-inch HD color touchscreen smart watch with heart rate, blood oxygen, sleep monitoring, IP68 waterproof, 14 sports modes, 7-day battery life. Compatible with iOS and Android.', '/seeds/smartwatch.jpg', 68.00, 'CNY', 0.12, 12, 10, 6, '91021200', 39.99, 'USD', 'US', 'draft', true),
('Portable LED Desk Lamp with Touch Control', 'Eye-caring LED desk lamp with 3 color temperatures, 10 brightness levels, USB charging, flexible gooseneck, touch control, memory function. Perfect for reading, studying, working.', '/seeds/desklamp.jpg', 28.00, 'CNY', 0.35, 20, 12, 6, '94052190', 15.99, 'USD', 'US', 'draft', true),
('Stainless Steel Vacuum Insulated Water Bottle 500ml', 'Double-wall vacuum insulated stainless steel water bottle, keeps drinks cold 24h / hot 12h. BPA-free, leak-proof, wide mouth, suitable for outdoor sports, gym, travel. 500ml capacity.', '/seeds/bottle.jpg', 22.00, 'CNY', 0.25, 25, 7, 7, '73239300', 12.99, 'USD', 'US', 'draft', true),
('Organic Cotton T-Shirt Men Casual Crew Neck', '100% organic cotton men t-shirt, soft breathable fabric, regular fit, crew neck, short sleeves. Pre-shrunk, machine washable. Available in 6 colors. Size S-XXL.', '/seeds/tshirt.jpg', 18.00, 'CNY', 0.20, 30, 20, 2, '61091000', 9.99, 'USD', 'US', 'draft', true),
('Wireless Charging Pad Fast Charger 15W', 'Qi-compatible wireless charging pad, 15W fast charging, supports iPhone, Samsung, Huawei. Ultra-thin 5mm design, LED indicator, smart temperature control, overcharge protection.', '/seeds/charger.jpg', 15.00, 'CNY', 0.06, 10, 10, 1, '85044019', 8.99, 'USD', 'US', 'draft', true),
('Portable External SSD 1TB USB-C', 'Ultra-fast portable SSD 1TB, USB 3.2 Gen2 Type-C, read speed up to 1050MB/s. Shockproof, dustproof, IP55. Pocket-sized 50g. Compatible with PC, Mac, PS5, Xbox.', '/seeds/ssd.jpg', 260.00, 'CNY', 0.04, 8, 5, 1, '84717020', 49.99, 'USD', 'US', 'draft', true),
('Natural Bamboo Cutting Board Set', 'Set of 3 natural organic bamboo cutting boards in different sizes. Antibacterial, knife-friendly, double-sided use. Includes juice groove. Eco-friendly packaging.', '/seeds/cuttingboard.jpg', 32.00, 'CNY', 0.60, 40, 25, 2, '44219090', 14.99, 'USD', 'US', 'draft', true),
('Yoga Mat Premium Non-Slip Extra Thick 10mm', 'Premium TPE yoga mat, 183x68x1cm, non-slip on both sides, excellent cushioning for joints. Lightweight 0.8kg, includes carrying strap. Non-toxic, eco-friendly material.', '/seeds/yogamat.jpg', 45.00, 'CNY', 0.80, 68, 18, 18, '95069110', 22.99, 'USD', 'US', 'draft', true),
('Rechargeable Handheld Mini Fan Portable', 'Portable mini handheld fan, USB rechargeable, 3 adjustable speeds, 4000mAh battery, up to 12h runtime. Foldable, lightweight 150g. Quiet operation <40dB.', '/seeds/fan.jpg', 20.00, 'CNY', 0.15, 10, 5, 3, '84145990', 9.99, 'USD', 'US', 'draft', true),
('Silicone Baking Mat Set Non-Stick 2-Pack', 'Set of 2 non-stick silicone baking mats, 43x30cm, suitable for standard baking sheets. FDA-approved, BPA-free, temperature range -40°C to 230°C. Reusable, easy to clean.', '/seeds/bakingmat.jpg', 12.00, 'CNY', 0.15, 30, 5, 5, '39241000', 6.99, 'USD', 'US', 'draft', true),
('Dog Nail Grinder Rechargeable Pet Nail Trimmer', 'Low-noise electric pet nail grinder with 2 speeds, 2 grinding ports, LED light. USB rechargeable, 8h battery. Safe for small and large dogs, cats. Includes 2 replacement heads.', '/seeds/nailgrinder.jpg', 38.00, 'CNY', 0.18, 15, 8, 4, '85098090', 16.99, 'USD', 'US', 'draft', true),
('Collapsible Silicone Food Storage Containers Set', 'Set of 4 collapsible silicone food storage containers, 500ml-1500ml. BPA-free, leak-proof, microwave/dishwasher safe. Collapses flat for space-saving storage. Assorted colors.', '/seeds/containers.jpg', 25.00, 'CNY', 0.30, 20, 15, 5, '39241000', 11.99, 'USD', 'US', 'draft', true),
('LED Grow Light Full Spectrum Plant Lamp', 'Full spectrum LED grow light for indoor plants, 40W equivalent, 3 switchable modes (red/blue/full), 6 brightness levels, auto timer 3/6/12h. Flexible gooseneck clamp design.', '/seeds/growlight.jpg', 42.00, 'CNY', 0.22, 18, 10, 5, '94054290', 18.99, 'USD', 'US', 'draft', true),
('Digital Kitchen Scale Precision 0.1g', 'High-precision digital kitchen scale, 2000g capacity / 0.1g accuracy. Stainless steel platform, tare function, auto-off, battery included. Unit conversion g/kg/oz/lb.', '/seeds/scale.jpg', 16.00, 'CNY', 0.20, 18, 14, 3, '84231000', 7.99, 'USD', 'US', 'draft', true),
('Travel Makeup Organizer Cosmetic Bag Large Capacity', 'Large capacity waterproof travel cosmetic bag, 30x20x15cm. Multiple compartments for makeup brushes, skincare, toiletries. Detachable hanging hook. Durable nylon material.', '/seeds/makeupbag.jpg', 28.00, 'CNY', 0.22, 30, 20, 3, '42021290', 12.99, 'USD', 'US', 'draft', true),
('Bluetooth Speaker Waterproof Portable IPX7', 'Portable waterproof Bluetooth speaker, IPX7 rated, 20W stereo sound, TWS pairing, 24h playtime. Built-in mic, TF card slot, USB-C charging. Perfect for outdoor, shower, beach.', '/seeds/speaker.jpg', 55.00, 'CNY', 0.35, 18, 8, 8, '85182200', 29.99, 'USD', 'US', 'draft', true),
('Linen Tablecloth Wrinkle Free 60x84 inch', 'Premium linen blend tablecloth, 60x84 inch rectangular, fits 6-8 seat table. Wrinkle-resistant fabric, machine washable. Available in 8 colors. Elegant for dinner parties or daily use.', '/seeds/tablecloth.jpg', 35.00, 'CNY', 0.40, 28, 20, 2, '63025990', 14.99, 'USD', 'US', 'draft', true),
('Electric Wine Opener Rechargeable', 'Rechargeable electric wine opener set, opens up to 30 bottles on full charge. Includes foil cutter, vacuum stopper. LED indicator, one-button operation. Comes in gift box.', '/seeds/wineopener.jpg', 48.00, 'CNY', 0.50, 25, 8, 8, '85098090', 21.99, 'USD', 'US', 'draft', true),
('Plant-Based Protein Powder Vegan 500g', 'Organic plant-based protein powder, pea and rice protein blend, 25g protein per serving. Unflavored, no artificial sweeteners. 500g bag. Suitable for vegans, gluten-free.', '/seeds/protein.jpg', 55.00, 'CNY', 0.55, 18, 12, 5, '21061000', 16.99, 'USD', 'US', 'draft', true)
ON CONFLICT DO NOTHING;
