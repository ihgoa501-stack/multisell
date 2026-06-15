"""物流运费 - 服务层"""

import csv
import io
import math
from decimal import Decimal
from typing import Optional

from sqlalchemy import select, delete, and_
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload
import openpyxl

from app.models import (
    Product, Sku,
    ShippingProvider, ShippingChannel, ShippingZone, ShippingQuoteRule,
)
from app.shipping.schemas import (
    ProviderCreate, ProviderUpdate,
    ChannelCreate, ChannelUpdate,
    ZoneCreate, RuleCreate, RuleUpdate,
    CalculateRequest, CalculateResultItem, PackageInfo, CalculateResponse,
)


# ── Helpers ───────────────────────────────────────────────────────────────

def _money(value) -> float:
    return float(value or 0)


def _to_float(value) -> Optional[float]:
    if value is None:
        return None
    return float(value)


def _text(value) -> str:
    if value is None:
        return ""
    return str(value).strip()


def _float_or_default(value, default: float = 0) -> float:
    if value is None or value == "":
        return default
    return float(value)


def _int_or_none(value) -> Optional[int]:
    if value is None or value == "":
        return None
    return int(float(value))


def _split_list(value) -> list[str]:
    if value is None or value == "":
        return ["normal"]
    if isinstance(value, list):
        return [str(item).strip() for item in value if str(item).strip()]
    parts = str(value).replace("，", ",").split(",")
    result = [part.strip() for part in parts if part.strip()]
    return result or ["normal"]


# ── Provider CRUD ─────────────────────────────────────────────────────────

class ProviderService:

    @staticmethod
    async def list(db: AsyncSession) -> list[dict]:
        stmt = select(ShippingProvider).order_by(ShippingProvider.id)
        result = await db.execute(stmt)
        providers = result.scalars().all()
        return [_provider_to_dict(p) for p in providers]

    @staticmethod
    async def get_by_id(db: AsyncSession, provider_id: int) -> Optional[dict]:
        stmt = select(ShippingProvider).where(ShippingProvider.id == provider_id)
        result = await db.execute(stmt)
        provider = result.scalar_one_or_none()
        return _provider_to_dict(provider) if provider else None

    @staticmethod
    async def create(db: AsyncSession, data: ProviderCreate) -> dict:
        provider = ShippingProvider(
            name=data.name,
            code=data.code,
            contact=data.contact,
            phone=data.phone,
            remark=data.remark,
        )
        db.add(provider)
        await db.flush()
        await db.refresh(provider)
        return _provider_to_dict(provider)

    @staticmethod
    async def update(db: AsyncSession, provider_id: int, data: ProviderUpdate) -> Optional[dict]:
        stmt = select(ShippingProvider).where(ShippingProvider.id == provider_id)
        result = await db.execute(stmt)
        provider = result.scalar_one_or_none()
        if not provider:
            return None
        update_data = data.model_dump(exclude_unset=True)
        for k, v in update_data.items():
            setattr(provider, k, v)
        await db.flush()
        await db.refresh(provider)
        return _provider_to_dict(provider)

    @staticmethod
    async def delete(db: AsyncSession, provider_id: int) -> bool:
        stmt = select(ShippingProvider).where(ShippingProvider.id == provider_id)
        result = await db.execute(stmt)
        provider = result.scalar_one_or_none()
        if not provider:
            return False
        provider.status = 0
        await db.flush()
        await db.refresh(provider)
        return True


def _provider_to_dict(p: ShippingProvider) -> dict:
    return {
        "id": p.id,
        "name": p.name,
        "code": p.code,
        "contact": p.contact,
        "phone": p.phone,
        "remark": p.remark,
        "status": p.status or 0,
        "created_at": p.created_at,
        "updated_at": p.updated_at,
    }


# ── Channel CRUD ──────────────────────────────────────────────────────────

class ChannelService:

    @staticmethod
    async def list(db: AsyncSession, provider_id: Optional[int] = None) -> list[dict]:
        stmt = select(ShippingChannel).options(
            selectinload(ShippingChannel.provider)
        ).order_by(ShippingChannel.sort_order, ShippingChannel.id)
        if provider_id is not None:
            stmt = stmt.where(ShippingChannel.provider_id == provider_id)
        result = await db.execute(stmt)
        channels = result.scalars().all()
        return [_channel_to_dict(c) for c in channels]

    @staticmethod
    async def get_by_id(db: AsyncSession, channel_id: int) -> Optional[dict]:
        stmt = select(ShippingChannel).options(
            selectinload(ShippingChannel.provider)
        ).where(ShippingChannel.id == channel_id)
        result = await db.execute(stmt)
        channel = result.scalar_one_or_none()
        return _channel_to_dict(channel) if channel else None

    @staticmethod
    async def create(db: AsyncSession, data: ChannelCreate) -> dict:
        channel = ShippingChannel(
            provider_id=data.provider_id,
            name=data.name,
            code=data.code,
            volumetric_divisor=data.volumetric_divisor,
            cargo_types=data.cargo_types,
            estimated_delivery_min=data.estimated_delivery_min,
            estimated_delivery_max=data.estimated_delivery_max,
            currency=data.currency,
            sort_order=data.sort_order,
        )
        db.add(channel)
        await db.flush()
        # 加载 provider 关联
        await db.refresh(channel, ["provider"])
        return _channel_to_dict(channel)

    @staticmethod
    async def update(db: AsyncSession, channel_id: int, data: ChannelUpdate) -> Optional[dict]:
        stmt = select(ShippingChannel).options(
            selectinload(ShippingChannel.provider)
        ).where(ShippingChannel.id == channel_id)
        result = await db.execute(stmt)
        channel = result.scalar_one_or_none()
        if not channel:
            return None
        update_data = data.model_dump(exclude_unset=True)
        for k, v in update_data.items():
            setattr(channel, k, v)
        await db.flush()
        await db.refresh(channel)
        return _channel_to_dict(channel)

    @staticmethod
    async def delete(db: AsyncSession, channel_id: int) -> bool:
        stmt = select(ShippingChannel).where(ShippingChannel.id == channel_id)
        result = await db.execute(stmt)
        channel = result.scalar_one_or_none()
        if not channel:
            return False
        channel.status = 0
        await db.flush()
        return True


def _channel_to_dict(c: ShippingChannel) -> dict:
    return {
        "id": c.id,
        "provider_id": c.provider_id,
        "provider_name": c.provider.name if c.provider else None,
        "name": c.name,
        "code": c.code,
        "volumetric_divisor": c.volumetric_divisor,
        "cargo_types": c.cargo_types or [],
        "estimated_delivery_min": c.estimated_delivery_min,
        "estimated_delivery_max": c.estimated_delivery_max,
        "currency": c.currency or "CNY",
        "sort_order": c.sort_order or 0,
        "status": c.status or 0,
        "created_at": c.created_at,
        "updated_at": c.updated_at,
    }


# ── Zone CRUD ─────────────────────────────────────────────────────────────

class ZoneService:

    @staticmethod
    async def list_by_channel(db: AsyncSession, channel_id: int) -> list[dict]:
        stmt = select(ShippingZone).where(
            ShippingZone.channel_id == channel_id
        ).order_by(ShippingZone.country_code)
        result = await db.execute(stmt)
        zones = result.scalars().all()
        return [_zone_to_dict(z) for z in zones]

    @staticmethod
    async def create(db: AsyncSession, channel_id: int, data: ZoneCreate) -> dict:
        zone = ShippingZone(
            channel_id=channel_id,
            country_code=data.country_code.upper(),
            postal_code_from=data.postal_code_from,
            postal_code_to=data.postal_code_to,
        )
        db.add(zone)
        await db.flush()
        return _zone_to_dict(zone)

    @staticmethod
    async def delete(db: AsyncSession, zone_id: int) -> bool:
        stmt = select(ShippingZone).where(ShippingZone.id == zone_id)
        result = await db.execute(stmt)
        zone = result.scalar_one_or_none()
        if not zone:
            return False
        await db.delete(zone)
        await db.flush()
        return True


def _zone_to_dict(z: ShippingZone) -> dict:
    return {
        "id": z.id,
        "channel_id": z.channel_id,
        "country_code": z.country_code,
        "postal_code_from": z.postal_code_from,
        "postal_code_to": z.postal_code_to,
        "status": z.status or 0,
        "created_at": z.created_at,
        "updated_at": z.updated_at,
    }


# ── Rule CRUD ─────────────────────────────────────────────────────────────

class RuleService:

    @staticmethod
    async def list_by_channel(db: AsyncSession, channel_id: int) -> list[dict]:
        stmt = (
            select(ShippingQuoteRule)
            .options(selectinload(ShippingQuoteRule.zone))
            .where(ShippingQuoteRule.channel_id == channel_id)
            .order_by(ShippingQuoteRule.priority, ShippingQuoteRule.id)
        )
        result = await db.execute(stmt)
        rules = result.scalars().all()
        return [_rule_to_dict(r) for r in rules]

    @staticmethod
    async def create(db: AsyncSession, channel_id: int, data: RuleCreate) -> dict:
        rule = ShippingQuoteRule(
            channel_id=channel_id,
            zone_id=data.zone_id,
            rule_type=data.rule_type,
            priority=data.priority,
            min_weight_kg=data.min_weight_kg,
            max_weight_kg=data.max_weight_kg,
            first_kg=data.first_kg,
            first_price=data.first_price,
            additional_kg=data.additional_kg,
            additional_price=data.additional_price,
            fixed_fee=data.fixed_fee,
            per_kg_price=data.per_kg_price,
            minimum_charge=data.minimum_charge,
            tier_config=data.tier_config,
            surcharge_fixed=data.surcharge_fixed,
            fuel_surcharge_pct=data.fuel_surcharge_pct,
            rounding_increment=data.rounding_increment,
            remark=data.remark,
        )
        db.add(rule)
        await db.flush()
        return _rule_to_dict(rule)

    @staticmethod
    async def update(db: AsyncSession, rule_id: int, data: RuleUpdate) -> Optional[dict]:
        stmt = select(ShippingQuoteRule).where(ShippingQuoteRule.id == rule_id)
        result = await db.execute(stmt)
        rule = result.scalar_one_or_none()
        if not rule:
            return None
        update_data = data.model_dump(exclude_unset=True)
        for k, v in update_data.items():
            setattr(rule, k, v)
        await db.flush()
        await db.refresh(rule)
        return _rule_to_dict(rule)

    @staticmethod
    async def delete(db: AsyncSession, rule_id: int) -> bool:
        stmt = select(ShippingQuoteRule).where(ShippingQuoteRule.id == rule_id)
        result = await db.execute(stmt)
        rule = result.scalar_one_or_none()
        if not rule:
            return False
        await db.delete(rule)
        await db.flush()
        return True


def _rule_to_dict(r: ShippingQuoteRule) -> dict:
    zone = r.__dict__.get("zone")
    return {
        "id": r.id,
        "channel_id": r.channel_id,
        "zone_id": r.zone_id,
        "country_code": zone.country_code if zone else None,
        "rule_type": r.rule_type,
        "priority": r.priority or 0,
        "min_weight_kg": _to_float(r.min_weight_kg),
        "max_weight_kg": _to_float(r.max_weight_kg),
        "first_kg": _to_float(r.first_kg),
        "first_price": _money(r.first_price),
        "additional_kg": _to_float(r.additional_kg),
        "additional_price": _money(r.additional_price),
        "fixed_fee": _money(r.fixed_fee),
        "per_kg_price": _money(r.per_kg_price),
        "minimum_charge": _to_float(r.minimum_charge),
        "tier_config": r.tier_config,
        "surcharge_fixed": _money(r.surcharge_fixed),
        "fuel_surcharge_pct": _to_float(r.fuel_surcharge_pct),
        "rounding_increment": _to_float(r.rounding_increment),
        "remark": r.remark,
        "status": r.status or 0,
        "created_at": r.created_at,
        "updated_at": r.updated_at,
    }


# ── Import ────────────────────────────────────────────────────────────────

class ImportService:
    """物流报价表导入。"""

    REQUIRED_COLUMNS = {"provider_name", "channel_name", "country_code", "rule_type"}

    @staticmethod
    def parse_file(filename: str, content: bytes) -> list[dict]:
        lower_name = filename.lower()
        if lower_name.endswith(".csv"):
            return ImportService._parse_csv(content)
        if lower_name.endswith(".xlsx"):
            return ImportService._parse_xlsx(content)
        raise ValueError("仅支持 .xlsx 或 .csv 报价表")

    @staticmethod
    def _parse_xlsx(content: bytes) -> list[dict]:
        wb = openpyxl.load_workbook(io.BytesIO(content), data_only=True)
        ws = wb.active
        rows = list(ws.iter_rows(values_only=True))
        if not rows:
            return []
        headers = [ImportService._normalize_header(cell) for cell in rows[0]]
        parsed = []
        for index, values in enumerate(rows[1:], start=2):
            if not any(value not in (None, "") for value in values):
                continue
            parsed.append({
                "row_number": index,
                **{headers[i]: values[i] if i < len(values) else None for i in range(len(headers)) if headers[i]},
            })
        return parsed

    @staticmethod
    def _parse_csv(content: bytes) -> list[dict]:
        text = content.decode("utf-8-sig")
        reader = csv.DictReader(io.StringIO(text))
        rows = []
        for index, row in enumerate(reader, start=2):
            normalized = {
                ImportService._normalize_header(key): value
                for key, value in row.items()
            }
            if not any(value not in (None, "") for value in normalized.values()):
                continue
            rows.append({"row_number": index, **normalized})
        return rows

    @staticmethod
    def _normalize_header(value) -> str:
        return _text(value).lower().replace(" ", "_").replace("-", "_")

    @staticmethod
    async def import_rules(db: AsyncSession, filename: str, content: bytes) -> dict:
        rows = ImportService.parse_file(filename, content)
        summary = {
            "total_rows": len(rows),
            "imported_rows": 0,
            "error_rows": 0,
            "created_providers": 0,
            "created_channels": 0,
            "created_zones": 0,
            "created_rules": 0,
            "errors": [],
        }

        for row in rows:
            row_number = row.get("row_number")
            try:
                ImportService._validate_row(row)
                provider, provider_created = await ImportService._get_or_create_provider(db, row)
                channel, channel_created = await ImportService._get_or_create_channel(db, provider.id, row)
                zone, zone_created = await ImportService._get_or_create_zone(db, channel.id, row)
                await ImportService._create_rule(db, channel.id, zone.id, row)
                summary["imported_rows"] += 1
                summary["created_providers"] += 1 if provider_created else 0
                summary["created_channels"] += 1 if channel_created else 0
                summary["created_zones"] += 1 if zone_created else 0
                summary["created_rules"] += 1
            except Exception as exc:
                summary["error_rows"] += 1
                summary["errors"].append({"row": row_number, "message": str(exc)})

        await db.flush()
        return summary

    @staticmethod
    def _validate_row(row: dict) -> None:
        missing = [col for col in ImportService.REQUIRED_COLUMNS if not _text(row.get(col))]
        if missing:
            raise ValueError(f"缺少必填字段: {', '.join(sorted(missing))}")

    @staticmethod
    async def _get_or_create_provider(db: AsyncSession, row: dict) -> tuple[ShippingProvider, bool]:
        provider_code = _text(row.get("provider_code")) or None
        provider_name = _text(row.get("provider_name"))
        stmt = select(ShippingProvider)
        if provider_code:
            stmt = stmt.where(ShippingProvider.code == provider_code)
        else:
            stmt = stmt.where(ShippingProvider.name == provider_name)
        provider = await db.scalar(stmt.limit(1))
        if provider:
            return provider, False
        provider = ShippingProvider(name=provider_name, code=provider_code)
        db.add(provider)
        await db.flush()
        return provider, True

    @staticmethod
    async def _get_or_create_channel(db: AsyncSession, provider_id: int, row: dict) -> tuple[ShippingChannel, bool]:
        channel_code = _text(row.get("channel_code")) or None
        channel_name = _text(row.get("channel_name"))
        stmt = select(ShippingChannel).where(ShippingChannel.provider_id == provider_id)
        if channel_code:
            stmt = stmt.where(ShippingChannel.code == channel_code)
        else:
            stmt = stmt.where(ShippingChannel.name == channel_name)
        channel = await db.scalar(stmt.limit(1))
        if channel:
            return channel, False
        channel = ShippingChannel(
            provider_id=provider_id,
            name=channel_name,
            code=channel_code,
            volumetric_divisor=int(_float_or_default(row.get("volumetric_divisor"), 6000)),
            cargo_types=_split_list(row.get("cargo_types")),
            estimated_delivery_min=_int_or_none(row.get("estimated_delivery_min")),
            estimated_delivery_max=_int_or_none(row.get("estimated_delivery_max")),
            currency=_text(row.get("currency")) or "CNY",
        )
        db.add(channel)
        await db.flush()
        return channel, True

    @staticmethod
    async def _get_or_create_zone(db: AsyncSession, channel_id: int, row: dict) -> tuple[ShippingZone, bool]:
        country_code = _text(row.get("country_code")).upper()
        stmt = select(ShippingZone).where(
            ShippingZone.channel_id == channel_id,
            ShippingZone.country_code == country_code,
        )
        zone = await db.scalar(stmt.limit(1))
        if zone:
            return zone, False
        zone = ShippingZone(channel_id=channel_id, country_code=country_code)
        db.add(zone)
        await db.flush()
        return zone, True

    @staticmethod
    async def _create_rule(db: AsyncSession, channel_id: int, zone_id: int, row: dict) -> ShippingQuoteRule:
        rule = ShippingQuoteRule(
            channel_id=channel_id,
            zone_id=zone_id,
            rule_type=_text(row.get("rule_type")),
            priority=int(_float_or_default(row.get("priority"), 0)),
            min_weight_kg=Decimal(str(_float_or_default(row.get("min_weight_kg"), 0))),
            max_weight_kg=None if row.get("max_weight_kg") in (None, "") else Decimal(str(_float_or_default(row.get("max_weight_kg")))),
            first_kg=Decimal(str(_float_or_default(row.get("first_kg"), 0))),
            first_price=Decimal(str(_float_or_default(row.get("first_price"), 0))),
            additional_kg=Decimal(str(_float_or_default(row.get("additional_kg"), 0))),
            additional_price=Decimal(str(_float_or_default(row.get("additional_price"), 0))),
            fixed_fee=Decimal(str(_float_or_default(row.get("fixed_fee"), 0))),
            per_kg_price=Decimal(str(_float_or_default(row.get("per_kg_price"), 0))),
            minimum_charge=None if row.get("minimum_charge") in (None, "") else Decimal(str(_float_or_default(row.get("minimum_charge")))),
            surcharge_fixed=Decimal(str(_float_or_default(row.get("surcharge_fixed"), 0))),
            fuel_surcharge_pct=Decimal(str(_float_or_default(row.get("fuel_surcharge_pct"), 0))),
            rounding_increment=Decimal(str(_float_or_default(row.get("rounding_increment"), 0.1))),
        )
        db.add(rule)
        await db.flush()
        return rule


# ── Calculation ───────────────────────────────────────────────────────────

class CalculateService:

    @staticmethod
    async def calculate(
        db: AsyncSession,
        req: CalculateRequest,
    ) -> CalculateResponse:
        # Step 1: 解析包装数据
        pkg = await _resolve_calculation_package(db, req)
        if pkg is None:
            raise ValueError("物流数据不完整：商品包装尺寸或包装重量缺失")

        # Step 2: 实际重量和体积重
        actual_weight = pkg["weight_kg"] * req.quantity
        base_volume = pkg["length_cm"] * pkg["width_cm"] * pkg["height_cm"] * req.quantity

        # Step 3: 查找可用渠道
        channels = await _find_active_channels(db, req.destination_country, req.cargo_type)

        results = []
        for channel in channels:
            try:
                result = _calculate_channel(
                    channel_data=channel,
                    actual_weight=actual_weight,
                    base_volume=base_volume,
                )
                results.append(result)
            except ValueError:
                continue

        # Step 4: 按总升序排序
        results.sort(key=lambda r: r.total_shipping_fee)

        return CalculateResponse(
            mode=req.mode,
            sku_id=req.sku_id,
            quantity=req.quantity,
            destination_country=req.destination_country.upper(),
            package=PackageInfo(
                source=pkg["source"],
                length_cm=pkg["length_cm"],
                width_cm=pkg["width_cm"],
                height_cm=pkg["height_cm"],
                weight_kg=pkg["weight_kg"],
            ),
            results=results,
        )


async def _resolve_calculation_package(db: AsyncSession, req: CalculateRequest) -> Optional[dict]:
    if req.mode == "manual":
        if req.package is None:
            return None
        return {
            "source": "manual",
            "length_cm": float(req.package.length_cm),
            "width_cm": float(req.package.width_cm),
            "height_cm": float(req.package.height_cm),
            "weight_kg": float(req.package.weight_kg),
        }
    if req.sku_id is None:
        return None
    return await _resolve_package(db, req.sku_id)


async def _resolve_package(db: AsyncSession, sku_id: int) -> Optional[dict]:
    """解析 SKU 或商品的包装数据。返回包装信息或 None。"""
    stmt = select(Sku).where(Sku.id == sku_id)
    result = await db.execute(stmt)
    sku = result.scalar_one_or_none()
    if not sku:
        return None

    # 检查 SKU 覆盖字段
    if (sku.sku_length_cm is not None and sku.sku_width_cm is not None
            and sku.sku_height_cm is not None and sku.sku_weight_kg is not None
            and float(sku.sku_length_cm) > 0 and float(sku.sku_width_cm) > 0
            and float(sku.sku_height_cm) > 0 and float(sku.sku_weight_kg) > 0):
        return {
            "source": "sku",
            "length_cm": float(sku.sku_length_cm),
            "width_cm": float(sku.sku_width_cm),
            "height_cm": float(sku.sku_height_cm),
            "weight_kg": float(sku.sku_weight_kg),
        }

    # 回退到商品级包装字段
    stmt2 = select(Product).where(Product.id == sku.product_id)
    result2 = await db.execute(stmt2)
    product = result2.scalar_one_or_none()
    if not product:
        return None

    if (product.package_length_cm is not None and product.package_width_cm is not None
            and product.package_height_cm is not None and product.package_weight_kg is not None
            and float(product.package_length_cm) > 0 and float(product.package_width_cm) > 0
            and float(product.package_height_cm) > 0 and float(product.package_weight_kg) > 0):
        return {
            "source": "product",
            "length_cm": float(product.package_length_cm),
            "width_cm": float(product.package_width_cm),
            "height_cm": float(product.package_height_cm),
            "weight_kg": float(product.package_weight_kg),
        }

    return None


async def _find_active_channels(
    db: AsyncSession,
    destination_country: str,
    cargo_type: str,
) -> list[dict]:
    """查找可用的物流渠道（供应商启用 + 渠道启用 + 目的地匹配 + 货品类型匹配）。"""
    stmt = (
        select(ShippingChannel)
        .join(ShippingProvider, ShippingChannel.provider_id == ShippingProvider.id)
        .options(
            selectinload(ShippingChannel.provider),
            selectinload(ShippingChannel.zones),
            selectinload(ShippingChannel.rules),
        )
        .where(
            ShippingProvider.status == 1,
            ShippingChannel.status == 1,
        )
        .order_by(ShippingChannel.sort_order, ShippingChannel.id)
    )
    result = await db.execute(stmt)
    channels = result.scalars().all()

    matched = []
    for ch in channels:
        # 检查目的地
        matched_zone_id = None
        for z in ch.zones:
            if z.status != 1:
                continue
            if z.country_code.upper() == destination_country.upper():
                matched_zone_id = z.id
                break
        if matched_zone_id is None:
            continue

        # 检查货品类型
        supported = ch.cargo_types or []
        if cargo_type not in supported:
            continue

        # 检查是否有活跃的报价规则
        active_rules = [r for r in ch.rules if r.status == 1]
        zone_rules = [r for r in active_rules if r.zone_id == matched_zone_id]
        global_rules = [r for r in active_rules if r.zone_id is None]
        matched_rules = zone_rules or global_rules
        if not matched_rules:
            continue

        matched.append({
            "channel": ch,
            "rules": matched_rules,
        })

    return matched


def _calculate_channel(
    channel_data: dict,
    actual_weight: float,
    base_volume: float,
) -> CalculateResultItem:
    ch = channel_data["channel"]
    rules = channel_data["rules"]

    # 体积重
    volumetric_weight = base_volume / ch.volumetric_divisor

    # 取最优先的规则
    rule = rules[0]
    rounding_inc = float(rule.rounding_increment) if rule.rounding_increment else 0.1
    if rounding_inc <= 0:
        rounding_inc = 0.1

    # 计费重
    chargeable = max(actual_weight, volumetric_weight)
    # 向上取整
    rounded = math.ceil(chargeable / rounding_inc) * rounding_inc

    # 计算基础运费
    base_fee = _apply_rule(rule, rounded)

    # 最低收费
    minimum_applied = False
    if rule.minimum_charge is not None and float(rule.minimum_charge) > 0:
        min_charge = float(rule.minimum_charge)
        if base_fee < min_charge:
            base_fee = min_charge
            minimum_applied = True

    # 附加费
    surcharge = float(rule.surcharge_fixed) if rule.surcharge_fixed else 0

    # 燃油附加费
    fuel_pct = float(rule.fuel_surcharge_pct) if rule.fuel_surcharge_pct else 0
    fuel_fee = (base_fee + surcharge) * (fuel_pct / 100)

    total = base_fee + surcharge + fuel_fee

    # 计算明细
    detail = _build_detail(rule, rounded, base_fee, minimum_applied, surcharge, fuel_fee)

    provider = ch.provider

    return CalculateResultItem(
        provider_id=provider.id if provider else 0,
        provider_name=provider.name if provider else "",
        channel_id=ch.id,
        channel_name=ch.name,
        currency=ch.currency or "CNY",
        actual_weight_kg=round(actual_weight, 4),
        volumetric_weight_kg=round(volumetric_weight, 4),
        chargeable_weight_kg=round(rounded, 4),
        base_shipping_fee=round(base_fee, 2),
        minimum_applied=minimum_applied,
        surcharge_fee=round(surcharge, 2),
        fuel_surcharge_fee=round(fuel_fee, 2),
        total_shipping_fee=round(total, 2),
        estimated_delivery_min=ch.estimated_delivery_min,
        estimated_delivery_max=ch.estimated_delivery_max,
        calculation_detail=detail,
    )


def _apply_rule(rule: ShippingQuoteRule, chargeable_weight: float) -> float:
    """根据规则类型计算基础运费。"""
    if rule.rule_type == "fixed_plus_per_kg":
        fixed = float(rule.fixed_fee) if rule.fixed_fee else 0
        per_kg = float(rule.per_kg_price) if rule.per_kg_price else 0
        return fixed + (chargeable_weight * per_kg)

    elif rule.rule_type == "first_weight_plus_increment":
        first_kg = float(rule.first_kg) if rule.first_kg else 0
        first_price = float(rule.first_price) if rule.first_price else 0
        add_kg = float(rule.additional_kg) if rule.additional_kg else 0.1
        add_price = float(rule.additional_price) if rule.additional_price else 0

        if chargeable_weight <= first_kg:
            return first_price
        if add_kg <= 0:
            add_kg = 0.1
        # 浮点精度保护
        raw_units = (chargeable_weight - first_kg) / add_kg
        additional_units = math.ceil(raw_units - 1e-10)
        return first_price + (additional_units * add_price)

    elif rule.rule_type == "tiered_weight":
        tiers = rule.tier_config
        if not tiers:
            return 0
        for tier in tiers:
            min_kg = float(tier.get("min_kg", 0))
            max_kg = tier.get("max_kg")
            price = float(tier.get("price", 0))
            if max_kg is None:
                if chargeable_weight >= min_kg - 1e-10:
                    return price
            elif (min_kg - 1e-10) <= chargeable_weight < (float(max_kg) + 1e-10):
                return price
        # 如果超出所有区间，使用最后一个
        if tiers:
            return float(tiers[-1].get("price", 0))
        return 0

    return 0


def _build_detail(
    rule: ShippingQuoteRule,
    chargeable_weight: float,
    base_fee: float,
    minimum_applied: bool,
    surcharge: float,
    fuel_fee: float,
) -> str:
    """构建人类可读的计算说明。"""
    parts = []
    if rule.rule_type == "fixed_plus_per_kg":
        fixed = float(rule.fixed_fee) if rule.fixed_fee else 0
        per_kg = float(rule.per_kg_price) if rule.per_kg_price else 0
        parts.append(f"固定费{fixed:.1f} + 计费重{chargeable_weight:.2f}kg × {per_kg:.1f} = {base_fee:.1f}")
    elif rule.rule_type == "first_weight_plus_increment":
        first_kg = float(rule.first_kg) if rule.first_kg else 0
        first_price = float(rule.first_price) if rule.first_price else 0
        add_kg = float(rule.additional_kg) if rule.additional_kg else 0.1
        add_price = float(rule.additional_price) if rule.additional_price else 0
        if chargeable_weight <= first_kg:
            parts.append(f"首重{first_kg}kg={first_price:.1f}")
        else:
            raw = (chargeable_weight - first_kg) / add_kg if add_kg > 0 else 0
            add_units = math.ceil(raw - 1e-10)
            parts.append(f"首重{first_kg}kg={first_price:.1f} + 续重{add_units}单位×{add_price:.1f} = {base_fee:.1f}")
    elif rule.rule_type == "tiered_weight":
        parts.append(f"阶梯价: 计费重{chargeable_weight:.2f}kg → {base_fee:.1f}")

    if minimum_applied:
        parts.append("(最低收费)")
    if surcharge > 0:
        parts.append(f"附加费{surcharge:.1f}")
    if fuel_fee > 0:
        parts.append(f"燃油附加费{fuel_fee:.1f}")

    return " + ".join(parts)
