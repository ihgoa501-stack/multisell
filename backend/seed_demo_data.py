#!/usr/bin/env python3
"""
演示数据补充脚本 — 用原始 SQL 创建物流、订单、库存等演示数据。
"""
import asyncio, sys, os, random
sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from datetime import datetime, timedelta, timezone
from sqlalchemy import text
from app.config import settings
from sqlalchemy.ext.asyncio import create_async_engine


async def run():
    engine = create_async_engine(settings.DATABASE_URL)
    async with engine.begin() as conn:
        now = datetime.now(timezone.utc)

        # ── 物流供应商 ──
        print("📦 物流供应商...")
        providers = {}
        for code, name in (("russian_post","俄罗斯邮政"),("cdek","CDEK"),("yuntu","云途物流"),("yanwen","燕文物流"),("china_post","中国邮政")):
            r = await conn.execute(text(f"INSERT INTO shipping_provider (name, code) SELECT '{name}', '{code}' WHERE NOT EXISTS (SELECT 1 FROM shipping_provider WHERE code='{code}') RETURNING id"))
            row = r.fetchone()
            if row: providers[code] = row[0]
        for row in (await conn.execute(text("SELECT id, code FROM shipping_provider"))).fetchall():
            if row[1] not in providers: providers[row[1]] = row[0]

        # ── 物流渠道 ──
        print("📦 物流渠道...")
        chs = {}
        for prov, name, code, cargo, dmin, dmax in [
            ("russian_post","经济小包","rpe",'["normal"]',7,20),
            ("russian_post","标准包","rps",'["normal","battery"]',10,18),
            ("cdek","标准快递","cdek_s",'["normal","battery"]',5,12),
            ("cdek","经济快递","cdek_e",'["normal"]',8,20),
            ("yuntu","俄罗斯专线","yuntu_r",'["normal"]',7,15),
            ("yanwen","俄罗斯专线","yanwen_r",'["normal","battery"]',10,22),
        ]:
            pid = providers.get(prov)
            if not pid: continue
            r = await conn.execute(text(
                f"INSERT INTO shipping_channel (provider_id,name,code,volumetric_divisor,cargo_types,estimated_delivery_min,estimated_delivery_max,currency) "
                f"SELECT {pid},'{name}','{code}',6000,'{cargo}'::jsonb,{dmin},{dmax},'USD' "
                f"WHERE NOT EXISTS (SELECT 1 FROM shipping_channel WHERE code='{code}') RETURNING id"
            ))
            row = r.fetchone()
            if row: chs[code] = row[0]

        # ── 区域和报价规则 ──
        print("📦 报价规则...")
        for ch, cc, rtype, fkg, fpr, akg, apr in [
            ("rpe","RU","first_weight_plus_increment",0.5,10,1.0,8),
            ("rps","RU","first_weight_plus_increment",0.5,15,0.5,10),
            ("cdek_s","RU","first_weight_plus_increment",0.5,12,0.5,8),
            ("cdek_e","RU","first_weight_plus_increment",0.5,8,1.0,5),
            ("yuntu_r","RU","first_weight_plus_increment",0.5,13,0.5,7),
            ("yanwen_r","RU","first_weight_plus_increment",0.5,9,1.0,6),
        ]:
            cid = chs.get(ch)
            if not cid: continue
            # 创建区域
            zr = await conn.execute(text(f"INSERT INTO shipping_zone (channel_id, country_code) SELECT {cid}, '{cc}' WHERE NOT EXISTS (SELECT 1 FROM shipping_zone WHERE channel_id={cid} AND country_code='{cc}') RETURNING id"))
            zrow = zr.fetchone()
            zid = zrow[0] if zrow else (await conn.execute(text(f"SELECT id FROM shipping_zone WHERE channel_id={cid} AND country_code='{cc}'"))).fetchone()[0]
            # 创建规则
            await conn.execute(text(
                f"INSERT INTO shipping_quote_rule (channel_id,zone_id,rule_type,min_weight_kg,max_weight_kg,first_kg,first_price,additional_kg,additional_price,priority) "
                f"SELECT {cid},{zid},'{rtype}',0.1,30.0,{fkg},{fpr},{akg},{apr},10 "
                f"WHERE NOT EXISTS (SELECT 1 FROM shipping_quote_rule WHERE channel_id={cid} AND zone_id={zid})"
            ))
        print("  ✅ 6 条报价规则")

        # ── 异常 ──
        print("📋 异常数据...")
        for i, (t, d, s, m) in enumerate([
            ("SKU-001 库存低于安全线","蓝牙耳机库存余5件","critical","inventory"),
            ("订单 ORD-20260615-001 物流异常","CDEK追踪号7天无更新","high","shipping"),
            ("Ozon 6月对账差异","结算差额 $128.50","error","settlement"),
            ("广告花费超预算15%","Shopee ACOS从22%升至37%","warning","listing"),
        ]):
            ts = (now - timedelta(hours=i*3)).isoformat()
            await conn.execute(text(f"INSERT INTO exception_item (title,description,severity,source_module,status,created_at) SELECT '{t}','{d}','{s}','{m}','open','{ts}' WHERE NOT EXISTS (SELECT 1 FROM exception_item WHERE title='{t}')"))

        # ── 通知 ──
        print("📋 通知...")
        for i, (t, c, a, s) in enumerate([
            ("补货提醒","NatureHome保温杯需补货","inventory_low_stock","warning"),
            ("每日经营报告","昨日订单3单 ¥1,280","daily_report","info"),
            ("利润预警","SKU-002利润率将至7.5%","margin_warning","warning"),
            ("合规提醒","EAC认证30天后到期","compliance_reminder","warning"),
        ]):
            ts = (now - timedelta(hours=i*2)).isoformat()
            await conn.execute(text(f"INSERT INTO notification (user_id,title,content,alert_type,severity,is_read,source_id,created_at) SELECT 1,'{t}','{c}','{a}','{s}',0,'demo-{i}','{ts}' WHERE NOT EXISTS (SELECT 1 FROM notification WHERE title='{t}')"))

        # ── 仓库 ──
        print("🏭 仓库...")
        for name, code in [("广州总仓","GZ"),("义乌分仓","YW"),("深圳跨境仓","SZ")]:
            await conn.execute(text(f"INSERT INTO warehouse (name, code) SELECT '{name}','{code}' WHERE NOT EXISTS (SELECT 1 FROM warehouse WHERE code='{code}')"))

        # ── 库存 (Inventory: unique on sku_id) ──
        print("🏭 库存...")
        skus = (await conn.execute(text("SELECT id FROM sku LIMIT 4"))).fetchall()
        for sid, in skus:
            qty = random.randint(50, 300)
            locked = random.randint(0, qty//5)
            await conn.execute(text(
                f"INSERT INTO inventory (sku_id, warehouse, quantity, locked_quantity, safety_stock) "
                f"SELECT {sid}, '广州总仓', {qty}, {locked}, 20 "
                f"WHERE NOT EXISTS (SELECT 1 FROM inventory WHERE sku_id={sid})"
            ))
            await conn.execute(text(f"INSERT INTO inventory_log (sku_id, change_type, change_qty, after_qty, remark) VALUES ({sid}, 'in', {qty}, {qty}, '演示初始化')"))

        # ── 订单 ──
        print("📋 订单...")
        products = (await conn.execute(text("SELECT p.id, p.name, COALESCE(s.price, 100) FROM product p LEFT JOIN sku s ON s.product_id=p.id AND s.id=(SELECT MIN(id) FROM sku WHERE product_id=p.id) LIMIT 5"))).fetchall()
        for oid, name, addr, status, days, items in [
            ("ORD-20260601-001","Ivan Petrov","莫斯科列宁大街100号","paid",5,[(0,2),(2,1)]),
            ("ORD-20260602-001","Nguyen Van A","圣彼得堡涅瓦大街50号","shipped",3,[(1,1)]),
            ("ORD-20260605-001","Anna Sokolova","新西伯利亚红色大街30号","pending",1,[(3,3),(4,1)]),
        ]:
            if (await conn.execute(text(f"SELECT 1 FROM sales_order WHERE order_no='{oid}'"))).fetchone(): continue
            ts = (now - timedelta(days=days)).isoformat()
            r = await conn.execute(text(f"INSERT INTO sales_order (order_no,recipient_name,shipping_address,status,total_amount,shipping_fee,pay_amount,created_at) VALUES ('{oid}','{name}','{addr}','{status}',0,0,0,'{ts}') RETURNING id"))
            dbid = r.fetchone()[0]
            total = 0.0
            for pi, qty in items:
                pid, pname, price = products[pi]
                uprice = round(float(price or 100)*1.15,2) if price else round(random.uniform(50,300),2)
                sub = round(uprice*qty,2)
                await conn.execute(text(f"INSERT INTO sales_order_item (order_id,sku_id,product_id,product_name,unit_price,quantity,subtotal) VALUES ({dbid},1,{pid},'{pname}',{uprice},{qty},{sub})"))
                total += sub
            sfee = round(total*0.1,2)
            await conn.execute(text(f"UPDATE sales_order SET total_amount={round(total,2)},shipping_fee={sfee},pay_amount={round(total*1.1,2)} WHERE id={dbid}"))
            await conn.execute(text(f"INSERT INTO finance_ledger_entry (entry_type,amount,currency,order_id,cost_layer,description,created_at) VALUES ('revenue',{round(total,2)},'CNY',{dbid},'product','订单{oid}收入','{ts}')"))
            print(f"  ✅ {oid} {name} ¥{round(total,2)}")

    await engine.dispose()
    print("\n✅ 演示数据全部完成！")

asyncio.run(run())
