"""数据库模型定义"""

from sqlalchemy import Column, BigInteger, String, Integer, Numeric, DateTime, Text, JSON, ForeignKey, SmallInteger, func, UniqueConstraint as sa_UniqueConstraint
from sqlalchemy.orm import relationship
from app.database import Base


class Category(Base):
    """商品分类"""
    __tablename__ = "category"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    name = Column(String(100), nullable=False, comment="分类名称")
    parent_id = Column(BigInteger, default=0, comment="父分类ID, 0表示根分类")
    level = Column(Integer, default=0, comment="层级")
    sort_order = Column(Integer, default=0, comment="排序")
    status = Column(SmallInteger, default=1, comment="状态: 0-禁用, 1-启用")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class Product(Base):
    """商品"""
    __tablename__ = "product"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    name = Column(String(200), nullable=False, comment="商品名称")
    subtitle = Column(String(500), comment="副标题")
    description = Column(Text, comment="商品描述")
    brand_id = Column(BigInteger, default=0, comment="品牌ID")
    category_id = Column(BigInteger, ForeignKey("category.id"), comment="分类ID")
    unit = Column(String(20), default="件", comment="单位")
    status = Column(SmallInteger, default=0, comment="状态: 0-草稿, 1-上架, 2-下架")
    main_image = Column(String(500), comment="主图URL")
    images = Column(JSON, comment="图片列表")
    product_length_cm = Column(Numeric(10, 2), comment="商品长(cm)")
    product_width_cm = Column(Numeric(10, 2), comment="商品宽(cm)")
    product_height_cm = Column(Numeric(10, 2), comment="商品高(cm)")
    product_weight_kg = Column(Numeric(10, 2), comment="商品重量(kg)")
    package_length_cm = Column(Numeric(10, 2), comment="包装长(cm)")
    package_width_cm = Column(Numeric(10, 2), comment="包装宽(cm)")
    package_height_cm = Column(Numeric(10, 2), comment="包装高(cm)")
    package_weight_kg = Column(Numeric(10, 2), comment="包装重量(kg)")
    cargo_type = Column(String(50), default="normal", comment="货品类型")
    # AI 辅助字段
    ai_title = Column(String(500), comment="AI生成的优化标题")
    ai_description = Column(Text, comment="AI生成的优化描述")
    seo_keywords = Column(JSON, comment="SEO关键词（AI建议的+用户自定义）")
    ai_status = Column(String(50), default="pending", comment="AI处理状态: pending/completed/failed")
    # 多平台状态概览
    platform_statuses = Column(JSON, comment="各平台发布状态概览")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")

    category = relationship("Category", lazy="selectin")


class SpecName(Base):
    """规格名称"""
    __tablename__ = "spec_name"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    product_id = Column(BigInteger, ForeignKey("product.id"), nullable=False, comment="商品ID")
    name = Column(String(100), nullable=False, comment="规格名称（如：颜色、尺寸）")
    sort_order = Column(Integer, default=0, comment="排序")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")

    values = relationship("SpecValue", backref="spec_name", lazy="selectin", cascade="all, delete-orphan",
                          order_by="SpecValue.sort_order")


class SpecValue(Base):
    """规格值"""
    __tablename__ = "spec_value"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    spec_name_id = Column(BigInteger, ForeignKey("spec_name.id"), nullable=False, comment="规格名称ID")
    product_id = Column(BigInteger, ForeignKey("product.id"), nullable=False, comment="商品ID")
    value = Column(String(100), nullable=False, comment="规格值（如：红色、S码）")
    sort_order = Column(Integer, default=0, comment="排序")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")


class Sku(Base):
    """SKU（库存量单位）"""
    __tablename__ = "sku"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    product_id = Column(BigInteger, ForeignKey("product.id"), nullable=False, comment="商品ID")
    code = Column(String(100), comment="SKU编码")
    barcode = Column(String(100), comment="条形码")
    spec_desc = Column(String(500), comment="规格描述（如：红色-S）")
    spec_values = Column(JSON, comment='规格值JSON(如: {"颜色":"红色","尺寸":"S"})')
    price = Column(Numeric(10, 2), default=0, comment="销售价")
    cost_price = Column(Numeric(10, 2), default=0, comment="成本价")
    market_price = Column(Numeric(10, 2), default=0, comment="市场价")
    stock = Column(Integer, default=0, comment="库存（已弃用，请使用 Inventory.quantity）")
    lock_stock = Column(Integer, default=0, comment="锁定库存")
    warning_stock = Column(Integer, default=0, comment="安全库存预警")
    weight = Column(Numeric(10, 2), default=0, comment="重量(kg，历史字段，不表示包装重量)")
    sku_length_cm = Column(Numeric(10, 2), comment="SKU包装长(cm)")
    sku_width_cm = Column(Numeric(10, 2), comment="SKU包装宽(cm)")
    sku_height_cm = Column(Numeric(10, 2), comment="SKU包装高(cm)")
    sku_weight_kg = Column(Numeric(10, 2), comment="SKU包装重量(kg)")
    image = Column(String(500), comment="SKU图片")
    status = Column(SmallInteger, default=1, comment="状态: 0-禁用, 1-启用")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class Price(Base):
    """价格"""
    __tablename__ = "price"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    sku_id = Column(BigInteger, ForeignKey("sku.id"), nullable=False, comment="SKU ID")
    price_type = Column(String(50), nullable=False, comment="价格类型: sale_price/market_price/cost_price/vip_price/wholesale_price")
    price = Column(Numeric(10, 2), nullable=False, comment="价格")
    start_time = Column(DateTime(timezone=True), comment="生效时间")
    end_time = Column(DateTime(timezone=True), comment="失效时间")
    status = Column(SmallInteger, default=1, comment="状态: 0-禁用, 1-启用")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class PriceChangeLog(Base):
    """调价记录"""
    __tablename__ = "price_change_log"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    sku_id = Column(BigInteger, ForeignKey("sku.id"), nullable=False, comment="SKU ID")
    old_price = Column(Numeric(10, 2), comment="旧价格")
    new_price = Column(Numeric(10, 2), comment="新价格")
    price_type = Column(String(50), comment="价格类型")
    change_type = Column(String(50), comment="变更类型: manual/batch")
    operator = Column(String(100), comment="操作人")
    remark = Column(String(500), comment="备注")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")


class Inventory(Base):
    """库存"""
    __tablename__ = "inventory"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    sku_id = Column(BigInteger, ForeignKey("sku.id"), nullable=False, unique=True, comment="SKU ID")
    warehouse = Column(String(100), default="默认仓库", comment="仓库")
    location = Column(String(200), comment="货位")
    quantity = Column(Integer, default=0, comment="当前库存")
    locked_quantity = Column(Integer, default=0, nullable=False, comment="锁定库存")
    safety_stock = Column(Integer, default=0, comment="安全库存")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class InventoryLog(Base):
    """库存变动记录"""
    __tablename__ = "inventory_log"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    sku_id = Column(BigInteger, ForeignKey("sku.id"), nullable=False, comment="SKU ID")
    change_type = Column(String(50), nullable=False, comment="变动类型: in/out/adjust")
    change_qty = Column(Integer, nullable=False, comment="变动数量")
    before_qty = Column(Integer, comment="变动前数量")
    after_qty = Column(Integer, comment="变动后数量")
    remark = Column(String(500), comment="备注")
    operator = Column(String(100), comment="操作人")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")


class Supplier(Base):
    """供应商"""
    __tablename__ = "supplier"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    name = Column(String(200), nullable=False, comment="供应商名称")
    contact_person = Column(String(100), comment="联系人")
    contact_phone = Column(String(50), comment="联系电话")
    email = Column(String(200), comment="邮箱")
    address = Column(String(500), comment="地址")
    status = Column(SmallInteger, default=1, comment="状态: 0-禁用, 1-启用")
    remark = Column(Text, comment="备注")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class ProductSupplier(Base):
    """商品-供应商关联"""
    __tablename__ = "product_supplier"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    product_id = Column(BigInteger, ForeignKey("product.id"), nullable=False, comment="商品ID")
    supplier_id = Column(BigInteger, ForeignKey("supplier.id"), nullable=False, comment="供应商ID")
    supply_price = Column(Numeric(10, 2), comment="供货价")
    min_order_qty = Column(Integer, default=1, comment="最小起订量")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")


class Sourcing1688Product(Base):
    """1688 货源采集候选池"""
    __tablename__ = "sourcing_1688_product"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    source_url = Column(String(1000), nullable=False, unique=True, comment="1688 商品链接")
    title = Column(String(500), comment="采集标题")
    price = Column(Numeric(10, 2), comment="采集供货价")
    moq = Column(Integer, default=1, comment="最小起订量")
    supplier_name = Column(String(200), comment="供应商名称")
    shop_url = Column(String(1000), comment="1688 店铺链接")
    shop_location = Column(String(200), comment="店铺地区")
    images = Column(JSON, comment="图片列表")
    attributes = Column(JSON, comment="属性列表")
    sku_variants = Column(JSON, comment="SKU 变体")
    description = Column(Text, comment="描述")
    package_length_cm = Column(Numeric(10, 2), comment="包装长(cm)")
    package_width_cm = Column(Numeric(10, 2), comment="包装宽(cm)")
    package_height_cm = Column(Numeric(10, 2), comment="包装高(cm)")
    package_weight_kg = Column(Numeric(10, 2), comment="包装重量(kg)")
    raw_data = Column(JSON, comment="完整原始 payload")
    status = Column(String(50), default="collected", comment="状态: collected/imported/rejected")
    product_id = Column(BigInteger, ForeignKey("product.id"), comment="导入后的商品 ID")
    supplier_id = Column(BigInteger, ForeignKey("supplier.id"), comment="导入后的供应商 ID")
    collected_by = Column(String(100), comment="采集人")
    imported_by = Column(String(100), comment="导入人")
    imported_at = Column(DateTime(timezone=True), comment="导入时间")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class ShippingProvider(Base):
    """物流供应商"""
    __tablename__ = "shipping_provider"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    name = Column(String(200), nullable=False, comment="供应商名称")
    code = Column(String(50), unique=True, comment="编码")
    contact = Column(String(100), comment="联系人")
    phone = Column(String(50), comment="联系电话")
    remark = Column(Text, comment="备注")
    status = Column(SmallInteger, default=1, comment="状态: 0-禁用, 1-启用")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class ShippingChannel(Base):
    """物流渠道（供应商的物流产品）"""
    __tablename__ = "shipping_channel"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    provider_id = Column(BigInteger, ForeignKey("shipping_provider.id"), nullable=False, comment="物流供应商ID")
    name = Column(String(200), nullable=False, comment="渠道名称")
    code = Column(String(50), comment="渠道编码")
    volumetric_divisor = Column(Integer, nullable=False, default=6000, comment="抛重系数")
    cargo_types = Column(JSON, comment="支持货品类型: [\"normal\",\"battery\",\"liquid\",\"sensitive\"]")
    estimated_delivery_min = Column(Integer, comment="最短时效(天)")
    estimated_delivery_max = Column(Integer, comment="最长时效(天)")
    currency = Column(String(10), default="CNY", comment="报价币种")
    sort_order = Column(Integer, default=0, comment="排序")
    status = Column(SmallInteger, default=1, comment="状态: 0-禁用, 1-启用")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")

    provider = relationship("ShippingProvider", lazy="selectin")
    zones = relationship("ShippingZone", lazy="selectin", cascade="all, delete-orphan")
    rules = relationship("ShippingQuoteRule", lazy="selectin", cascade="all, delete-orphan",
                         order_by="ShippingQuoteRule.priority, ShippingQuoteRule.id")


class ShippingZone(Base):
    """物流区域（渠道覆盖的目的地国家）"""
    __tablename__ = "shipping_zone"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    channel_id = Column(BigInteger, ForeignKey("shipping_channel.id"), nullable=False, comment="物流渠道ID")
    country_code = Column(String(10), nullable=False, comment="国家代码 ISO 3166-1 alpha-2")
    postal_code_from = Column(String(20), comment="邮编范围起始")
    postal_code_to = Column(String(20), comment="邮编范围截止")
    status = Column(SmallInteger, default=1, comment="状态: 0-禁用, 1-启用")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")

    channel_rel = relationship("ShippingChannel", viewonly=True, overlaps="zones")


class ShippingQuoteRule(Base):
    """报价规则"""
    __tablename__ = "shipping_quote_rule"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    channel_id = Column(BigInteger, ForeignKey("shipping_channel.id"), nullable=False, comment="物流渠道ID")
    zone_id = Column(BigInteger, ForeignKey("shipping_zone.id"), comment="物流区域ID，空表示渠道全局规则")
    rule_type = Column(String(50), nullable=False, comment="规则类型: fixed_plus_per_kg/first_weight_plus_increment/tiered_weight")
    priority = Column(Integer, default=0, comment="优先级，值小优先")
    min_weight_kg = Column(Numeric(10, 3), default=0, comment="适用最小重量(kg)")
    max_weight_kg = Column(Numeric(10, 3), comment="适用最大重量(kg)，NULL无上限")
    first_kg = Column(Numeric(10, 3), default=0, comment="首重(kg)")
    first_price = Column(Numeric(10, 2), default=0, comment="首重价格")
    additional_kg = Column(Numeric(10, 3), default=0, comment="续重单位(kg)")
    additional_price = Column(Numeric(10, 2), default=0, comment="续重单价")
    fixed_fee = Column(Numeric(10, 2), default=0, comment="固定费")
    per_kg_price = Column(Numeric(10, 2), default=0, comment="每公斤价格")
    minimum_charge = Column(Numeric(10, 2), comment="最低收费")
    tier_config = Column(JSON, comment="阶梯配置: [{min_kg, max_kg, price}]")
    surcharge_fixed = Column(Numeric(10, 2), default=0, comment="附加费(固定)")
    fuel_surcharge_pct = Column(Numeric(5, 2), default=0, comment="燃油附加费百分比")
    rounding_increment = Column(Numeric(10, 3), default=0.1, comment="计费重向上取整增量(kg)")
    remark = Column(Text, comment="备注")
    status = Column(SmallInteger, default=1, comment="状态: 0-禁用, 1-启用")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")

    channel_rel = relationship("ShippingChannel", viewonly=True, overlaps="rules")
    zone = relationship("ShippingZone", lazy="selectin", foreign_keys=[zone_id])


class OperationLog(Base):
    """操作日志"""
    __tablename__ = "operation_log"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    module = Column(String(50), comment="模块")
    action = Column(String(50), comment="操作")
    resource_id = Column(String(100), comment="资源ID")
    content = Column(Text, comment="操作内容")
    operator = Column(String(100), comment="操作人")
    ip = Column(String(50), comment="IP地址")
    duration = Column(Integer, comment="耗时(ms)")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")


class Brand(Base):
    """品牌"""
    __tablename__ = "brand"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    name = Column(String(200), nullable=False, unique=True, comment="品牌名称")
    logo = Column(String(500), comment="品牌Logo URL")
    description = Column(Text, comment="品牌描述")
    status = Column(SmallInteger, default=1, comment="状态: 0-禁用, 1-启用")
    sort_order = Column(Integer, default=0, comment="排序")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class Platform(Base):
    """电商平台配置"""
    __tablename__ = "platform"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    name = Column(String(100), nullable=False, comment="平台名称（如: Ozon, Shopee, Wildberries）")
    code = Column(String(50), nullable=False, unique=True, comment="平台代码（ozon, shopee, wb）")
    api_base_url = Column(String(500), comment="API基础地址")
    api_key = Column(String(500), comment="API密钥（加密存储）")
    client_id = Column(String(200), comment="Client ID")
    extra_config = Column(JSON, comment="额外配置")
    status = Column(SmallInteger, default=1, comment="状态: 0-禁用, 1-启用")
    sort_order = Column(Integer, default=0, comment="排序")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class ProductListing(Base):
    """商品在各平台的发布记录"""
    __tablename__ = "product_listing"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    product_id = Column(BigInteger, ForeignKey("product.id"), nullable=False, comment="商品ID")
    platform_id = Column(BigInteger, ForeignKey("platform.id"), nullable=False, comment="平台ID")
    platform_product_id = Column(String(200), comment="平台上的商品ID")
    platform_sku = Column(String(200), comment="平台上的SKU")
    status = Column(String(50), default="draft", comment="发布状态: draft/pending/synced/failed")
    platform_url = Column(String(500), comment="平台商品链接")
    sync_message = Column(Text, comment="同步失败的错误信息")
    published_data = Column(JSON, comment="发布到平台的原始数据")
    last_sync_at = Column(DateTime(timezone=True), comment="上次同步时间")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class ListingTask(Base):
    """上架任务队列"""
    __tablename__ = "listing_task"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    product_id = Column(BigInteger, ForeignKey("product.id"), nullable=False, comment="商品ID")
    platform_id = Column(BigInteger, ForeignKey("platform.id"), nullable=False, comment="平台ID")
    sku_id = Column(BigInteger, ForeignKey("sku.id"), comment="来源SKU ID")
    product_listing_id = Column(BigInteger, ForeignKey("product_listing.id"), comment="发布记录ID")
    source_type = Column(String(50), default="decision", nullable=False, comment="来源: decision/manual")
    source_item_key = Column(String(100), comment="来源行标识")
    status = Column(String(50), default="blocked", nullable=False, comment="ready/blocked/published/failed/cancelled")
    missing_requirements = Column(JSON, default=list, nullable=False, comment="阻塞发布的缺失项")
    decision_snapshot = Column(JSON, comment="决策结果快照")
    target_sale_price = Column(Numeric(12, 2), comment="决策目标售价")
    target_profit_margin = Column(Numeric(8, 2), comment="决策利润率")
    destination_country = Column(String(10), comment="目的国")
    last_error = Column(Text, comment="最近错误")
    created_by = Column(String(100), comment="创建人")
    updated_by = Column(String(100), comment="更新人")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")

    product = relationship("Product", lazy="selectin")
    platform = relationship("Platform", lazy="selectin")
    sku = relationship("Sku", lazy="selectin")
    product_listing = relationship("ProductListing", lazy="selectin")


class ShippingBillBatch(Base):
    """运费账单批次"""
    __tablename__ = "shipping_bill_batch"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    provider_id = Column(BigInteger, ForeignKey("shipping_provider.id"), comment="物流供应商ID")
    source_filename = Column(String(500), nullable=False, comment="源文件名")
    currency = Column(String(10), server_default="CNY", comment="默认币种")
    row_count = Column(Integer, default=0, comment="导入行数")
    matched_count = Column(Integer, default=0, comment="对账成功")
    mismatch_count = Column(Integer, default=0, comment="金额/币种不符")
    unmatched_count = Column(Integer, default=0, comment="无匹配订单")
    status = Column(String(30), server_default="imported", comment="imported/reconciled/failed")
    created_by = Column(String(100), comment="创建人")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class ShippingBillItem(Base):
    """运费账单行"""
    __tablename__ = "shipping_bill_item"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    batch_id = Column(BigInteger, ForeignKey("shipping_bill_batch.id"), nullable=False, comment="批次ID")
    row_number = Column(Integer, nullable=False, comment="原始行号")
    reconciliation_status = Column(String(30), server_default="unmatched_bill", comment="matched/unmatched_bill/missing_snapshot/amount_mismatch/currency_mismatch")

    # 运单信息
    tracking_number = Column(String(200), comment="运单号")
    order_no = Column(String(200), comment="订单号")
    provider_name = Column(String(200), comment="物流商名称")
    channel_name = Column(String(200), comment="渠道名称")
    destination_country = Column(String(10), comment="目的国")

    # 账单金额
    billed_weight_kg = Column(Numeric(10, 3), comment="账单计费重(kg)")
    currency = Column(String(10), server_default="CNY", comment="币种")
    actual_shipping_fee = Column(Numeric(12, 2), comment="实际运费")
    surcharge_fee = Column(Numeric(12, 2), default=0, comment="附加费")
    total_actual_fee = Column(Numeric(12, 2), comment="实际总运费(含附加费)")
    billed_at = Column(DateTime(timezone=True), comment="账单日期")

    # 匹配到的系统数据
    matched_order_id = Column(BigInteger, ForeignKey("sales_order.id"), comment="匹配订单ID")
    matched_snapshot_id = Column(BigInteger, ForeignKey("sales_order_shipping_snapshot.id"), comment="匹配快照ID")
    snapshot_shipping_fee = Column(Numeric(12, 2), comment="快照运费")
    variance_amount = Column(Numeric(12, 2), comment="差异金额(账单-快照)")

    # 原始数据
    raw_payload = Column(JSON, comment="原始CSV行数据")

    # 差错处理
    note = Column(Text, comment="备注/手动解决说明")
    resolved_by = Column(String(100), comment="解决人")
    resolved_at = Column(DateTime(timezone=True), comment="解决时间")

    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")

    batch = relationship("ShippingBillBatch", lazy="selectin")
    matched_order = relationship("Order", lazy="selectin")
    matched_snapshot = relationship("OrderShippingSnapshot", lazy="selectin")


ALLOWED_TRANSACTION_TYPES = [
    "sale", "platform_fee", "payment_fee", "refund", "adjustment", "payout", "tax", "other",
]


class PlatformSettlementBatch(Base):
    """平台结算批次"""
    __tablename__ = "platform_settlement_batch"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    platform_name = Column(String(100), comment="结算单平台名称")
    filename = Column(String(500), nullable=False, comment="源文件名")
    row_count = Column(Integer, default=0, comment="导入行数")
    matched_count = Column(Integer, default=0, comment="已匹配行")
    unmatched_count = Column(Integer, default=0, comment="未匹配行")
    import_status = Column(String(30), server_default="imported", comment="imported/partial/complete")
    status = Column(String(30), server_default="imported", comment="imported/matched")
    created_by = Column(String(100), comment="创建人")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class PlatformSettlementItem(Base):
    """平台结算行"""
    __tablename__ = "platform_settlement_item"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    batch_id = Column(BigInteger, ForeignKey("platform_settlement_batch.id"), nullable=False, comment="批次ID")
    row_number = Column(Integer, nullable=False, comment="原始行号")

    platform = Column(String(100), comment="平台名称（导入时填写）")
    store_name = Column(String(200), comment="店铺名称")
    platform_order_no = Column(String(200), comment="平台订单号")
    order_no = Column(String(200), comment="系统订单号")
    transaction_type = Column(String(50), nullable=False, comment="交易类型: sale/platform_fee/payment_fee/refund/adjustment/payout/tax/other")
    currency = Column(String(10), server_default="CNY", comment="币种")
    amount = Column(Numeric(14, 2), default=0, comment="金额")
    settled_at = Column(DateTime(timezone=True), comment="结算日期")
    description = Column(Text, comment="描述")

    match_status = Column(String(30), server_default="unmatched", comment="matched/unmatched/manual")
    matched_order_id = Column(BigInteger, ForeignKey("sales_order.id"), comment="匹配订单ID")

    raw_payload = Column(JSON, comment="原始CSV行数据")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")

    batch = relationship("PlatformSettlementBatch", lazy="selectin")
    matched_order = relationship("Order", lazy="selectin")


class FinanceLedgerEntry(Base):
    """财务账本条目"""
    __tablename__ = "finance_ledger_entry"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    order_id = Column(BigInteger, ForeignKey("sales_order.id"), nullable=False, comment="订单ID")
    entry_type = Column(String(50), nullable=False, comment="条目类型: revenue/product_cost/shipping_cost/platform_fee/payment_fee/refund/adjustment/other_fee")
    amount = Column(Numeric(14, 2), nullable=False, comment="金额（正数为收入，负数为成本/费用）")
    currency = Column(String(10), server_default="CNY", comment="币种")
    cost_layer = Column(String(30), nullable=False, comment="成本层: estimated/snapshot/actual/allocated")
    source_type = Column(String(50), comment="来源类型: order/shipping_snapshot/shipping_bill_row/settlement_row")
    source_id = Column(BigInteger, comment="来源ID")
    description = Column(String(500), comment="描述说明")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")

    order = relationship("Order", lazy="selectin")


EXCEPTION_SEVERITY_CHOICES = ["low", "medium", "high", "critical"]
EXCEPTION_STATUS_CHOICES = ["open", "assigned", "resolved", "ignored"]


class ExceptionItem(Base):
    """异常工作台条目"""
    __tablename__ = "exception_item"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    source_module = Column(String(50), nullable=False, comment="来源模块: listing/shipping/settlement/finance")
    source_type = Column(String(50), comment="来源类型")
    source_id = Column(BigInteger, comment="来源ID")
    severity = Column(String(20), default="medium", comment="严重程度: low/medium/high/critical")
    status = Column(String(20), default="open", comment="状态: open/assigned/resolved/ignored")
    title = Column(String(300), nullable=False, comment="异常标题")
    description = Column(Text, comment="异常描述")
    recommended_action = Column(String(500), comment="建议操作")
    assigned_to = Column(String(100), comment="分配给")
    resolved_at = Column(DateTime(timezone=True), comment="解决时间")
    resolved_by = Column(String(100), comment="解决人")
    note = Column(Text, comment="备注")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


AGENT_ACTION_STATUS_FLOW = {
    "proposed": {"approved", "rejected"},
    "approved": {"executed"},
    "rejected": set(),
    "executed": set(),
}


class AgentAction(Base):
    """Agent 动作提案与审批"""
    __tablename__ = "agent_action"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    source_module = Column(String(50), comment="来源模块")
    source_type = Column(String(50), comment="来源类型")
    source_id = Column(BigInteger, comment="来源ID")
    exception_id = Column(BigInteger, ForeignKey("exception_item.id"), comment="关联异常ID")
    action_type = Column(String(100), nullable=False, comment="动作类型")
    title = Column(String(300), nullable=False, comment="动作标题")
    description = Column(Text, comment="动作描述")
    proposed_payload = Column(JSON, comment="动作提议参数")
    before_snapshot = Column(JSON, comment="执行前状态快照")
    after_snapshot = Column(JSON, comment="执行后状态快照")
    status = Column(String(30), default="proposed", comment="proposed/approved/rejected/executed")
    proposed_by = Column(String(100), comment="提议人")
    approved_by = Column(String(100), comment="审批人")
    approved_at = Column(DateTime(timezone=True), comment="审批时间")
    rejected_by = Column(String(100), comment="驳回人")
    rejected_at = Column(DateTime(timezone=True), comment="驳回时间")
    rejection_reason = Column(Text, comment="驳回原因")
    executed_by = Column(String(100), comment="执行人")
    executed_at = Column(DateTime(timezone=True), comment="执行时间")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


AgentActionProposal = AgentAction


class PlatformIntegrationAccount(Base):
    """平台账号/连接配置"""
    __tablename__ = "platform_integration_account"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    platform_id = Column(BigInteger, ForeignKey("platform.id"), nullable=False, comment="平台ID")
    adapter_code = Column(String(50), nullable=False, comment="adapter 代码")
    account_name = Column(String(200), nullable=False, comment="账号名称")
    status = Column(String(30), default="draft", comment="draft/active/disabled")
    credential_metadata = Column(JSON, default=dict, nullable=False, comment="密钥元信息: [{key: 'api_key', masked: '***...abcd'}]，不存明文")
    created_by = Column(String(100), comment="创建人")
    updated_by = Column(String(100), comment="更新人")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")

    platform = relationship("Platform", lazy="selectin")


class PlatformCategoryMapping(Base):
    """平台类目映射"""
    __tablename__ = "platform_category_mapping"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    platform_id = Column(BigInteger, ForeignKey("platform.id"), nullable=False, comment="平台ID")
    adapter_code = Column(String(50), nullable=False, comment="适配器代码")
    local_category_id = Column(BigInteger, ForeignKey("category.id"), nullable=False, comment="本地类目ID")
    platform_category_id = Column(String(200), nullable=False, comment="平台类目ID")
    platform_category_name = Column(String(500), comment="平台类目名称")
    platform_category_path = Column(String(1000), comment="平台类目路径")
    created_by = Column(String(100), comment="创建人")
    updated_by = Column(String(100), comment="更新人")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class PlatformAttributeMapping(Base):
    """平台属性映射"""
    __tablename__ = "platform_attribute_mapping"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    platform_id = Column(BigInteger, ForeignKey("platform.id"), nullable=False, comment="平台ID")
    adapter_code = Column(String(50), nullable=False, comment="适配器代码")
    local_attribute = Column(String(100), nullable=False, comment="本地属性名")
    platform_attribute = Column(String(200), nullable=False, comment="平台属性名")
    default_value = Column(String(500), comment="默认值")
    created_by = Column(String(100), comment="创建人")
    updated_by = Column(String(100), comment="更新人")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


ALLOWED_ALLOCATION_TYPES = ["first_leg", "fba", "overseas_warehouse", "other"]
ALLOWED_ALLOCATION_METHODS = ["quantity", "weight", "volume", "value"]


class CostAllocationBatch(Base):
    """费用分摊批次"""
    __tablename__ = "cost_allocation_batch"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    allocation_type = Column(String(50), nullable=False, comment="first_leg/fba/overseas_warehouse/other")
    allocation_method = Column(String(30), nullable=False, comment="quantity/weight/volume/value")
    total_amount = Column(Numeric(14, 2), nullable=False, comment="分摊总金额")
    currency = Column(String(10), server_default="CNY", comment="币种")
    source_filename = Column(String(500), comment="源文件名")
    row_count = Column(Integer, default=0, comment="行数")
    status = Column(String(30), server_default="imported", comment="imported/calculated/posted")
    posted_count = Column(Integer, default=0, comment="已入账行数")
    created_by = Column(String(100), comment="创建人")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class CostAllocationItem(Base):
    """费用分摊明细"""
    __tablename__ = "cost_allocation_item"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    batch_id = Column(BigInteger, ForeignKey("cost_allocation_batch.id"), nullable=False, comment="批次ID")
    row_number = Column(Integer, nullable=False, comment="原始行号")

    sku_id = Column(BigInteger, ForeignKey("sku.id"), comment="SKU ID")
    sku_code = Column(String(100), comment="SKU编码")
    order_id = Column(BigInteger, ForeignKey("sales_order.id"), comment="订单ID")
    quantity = Column(Integer, default=0, comment="数量")
    weight_kg = Column(Numeric(10, 3), comment="重量(kg)")
    volume_m3 = Column(Numeric(10, 4), comment="体积(m³)")
    item_value = Column(Numeric(14, 2), comment="货值")

    allocation_factor = Column(Numeric(14, 4), comment="分摊因子")
    allocated_amount = Column(Numeric(14, 2), default=0, comment="分摊金额")
    cost_layer = Column(String(30), server_default="allocated", comment="成本层")

    posted_to_ledger = Column(Integer, default=0, comment="是否已入账")
    raw_payload = Column(JSON, comment="原始CSV行数据")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")

    batch = relationship("CostAllocationBatch", lazy="selectin")
    sku = relationship("Sku", lazy="selectin")
    matched_order = relationship("Order", lazy="selectin")


class User(Base):
    """用户"""
    __tablename__ = "user"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    username = Column(String(100), nullable=False, unique=True, comment="用户名")
    password_hash = Column(String(500), nullable=False, comment="密码哈希")
    display_name = Column(String(200), comment="显示名称")
    role = Column(String(50), default="user", comment="角色: admin/user")
    email = Column(String(200), comment="邮箱")
    status = Column(SmallInteger, default=1, comment="状态: 0-禁用, 1-启用")
    last_login_at = Column(DateTime(timezone=True), comment="最后登录时间")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")

    roles = relationship("Role", secondary="user_role", backref="users", lazy="selectin")


class Role(Base):
    """角色"""
    __tablename__ = "role"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    name = Column(String(100), nullable=False, comment="角色名称")
    code = Column(String(100), nullable=False, unique=True, comment="角色代码")
    description = Column(String(500), comment="角色描述")
    status = Column(SmallInteger, default=1, comment="状态: 0-禁用, 1-启用")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")

    permissions = relationship("Permission", secondary="role_permission", backref="roles", lazy="selectin")


class Permission(Base):
    """权限"""
    __tablename__ = "permission"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    name = Column(String(100), nullable=False, comment="权限名称")
    code = Column(String(100), nullable=False, unique=True, comment="权限代码")
    description = Column(String(500), comment="权限描述")
    module = Column(String(100), comment="所属模块")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")


class UserRole(Base):
    """用户-角色关联"""
    __tablename__ = "user_role"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    user_id = Column(BigInteger, ForeignKey("user.id"), nullable=False, comment="用户ID")
    role_id = Column(BigInteger, ForeignKey("role.id"), nullable=False, comment="角色ID")


class RolePermission(Base):
    """角色-权限关联"""
    __tablename__ = "role_permission"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    role_id = Column(BigInteger, ForeignKey("role.id"), nullable=False, comment="角色ID")
    permission_id = Column(BigInteger, ForeignKey("permission.id"), nullable=False, comment="权限ID")


class Order(Base):
    """销售订单"""
    __tablename__ = "sales_order"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    order_no = Column(String(100), nullable=False, unique=True, comment="订单号")
    status = Column(String(50), default="pending", comment="状态: pending/paid/shipped/delivered/completed/cancelled")
    tracking_number = Column(String(200), comment="运单号/追踪号")
    recipient_name = Column(String(100), comment="收件人")
    recipient_phone = Column(String(50), comment="联系电话")
    shipping_address = Column(String(500), comment="收货地址")
    total_amount = Column(Numeric(10, 2), default=0, comment="商品总额")
    shipping_fee = Column(Numeric(10, 2), default=0, comment="运费")
    pay_amount = Column(Numeric(10, 2), default=0, comment="实付金额")
    platform_fee = Column(Numeric(10, 2), default=0, comment="平台佣金/平台费")
    payment_fee = Column(Numeric(10, 2), default=0, comment="支付手续费")
    other_fee = Column(Numeric(10, 2), default=0, comment="其他费用")
    product_cost = Column(Numeric(10, 2), default=0, comment="商品成本")
    profit_amount = Column(Numeric(10, 2), default=0, comment="订单利润")
    profit_margin = Column(Numeric(10, 4), default=0, comment="利润率百分比")
    payment_method = Column(String(50), comment="支付方式")
    remark = Column(Text, comment="备注")
    paid_at = Column(DateTime(timezone=True), comment="支付时间")
    shipped_at = Column(DateTime(timezone=True), comment="发货时间")
    delivered_at = Column(DateTime(timezone=True), comment="签收时间")
    cancelled_at = Column(DateTime(timezone=True), comment="取消时间")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class OrderItem(Base):
    """销售订单明细"""
    __tablename__ = "sales_order_item"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    order_id = Column(BigInteger, ForeignKey("sales_order.id"), nullable=False, comment="订单ID")
    sku_id = Column(BigInteger, ForeignKey("sku.id"), nullable=False, comment="SKU ID")
    product_id = Column(BigInteger, ForeignKey("product.id"), nullable=False, comment="商品ID")
    product_name = Column(String(200), nullable=False, comment="商品名称快照")
    sku_code = Column(String(100), comment="SKU编码快照")
    spec_desc = Column(String(500), comment="规格描述快照")
    unit_price = Column(Numeric(10, 2), nullable=False, comment="单价")
    quantity = Column(Integer, nullable=False, comment="数量")
    subtotal = Column(Numeric(10, 2), nullable=False, comment="小计")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")


class OrderStatusLog(Base):
    """订单状态流转记录"""
    __tablename__ = "sales_order_status_log"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    order_id = Column(BigInteger, ForeignKey("sales_order.id"), nullable=False, comment="订单ID")
    from_status = Column(String(50), comment="原状态")
    to_status = Column(String(50), nullable=False, comment="新状态")
    operator = Column(String(100), comment="操作人")
    remark = Column(String(500), comment="备注")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")


class OrderShippingSnapshot(Base):
    """订单运费快照"""
    __tablename__ = "sales_order_shipping_snapshot"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    order_id = Column(BigInteger, ForeignKey("sales_order.id"), nullable=False, unique=True, comment="订单ID")
    sku_id = Column(BigInteger, ForeignKey("sku.id"), nullable=False, comment="试算SKU ID")
    quantity = Column(Integer, nullable=False, comment="试算数量")
    destination_country = Column(String(10), nullable=False, comment="目的地国家")
    postal_code = Column(String(20), comment="目的地邮编")
    cargo_type = Column(String(50), default="normal", comment="货品类型")
    package_source = Column(String(20), comment="包装数据来源: product/sku")
    package_length_cm = Column(Numeric(10, 2), nullable=False, comment="包装长cm")
    package_width_cm = Column(Numeric(10, 2), nullable=False, comment="包装宽cm")
    package_height_cm = Column(Numeric(10, 2), nullable=False, comment="包装高cm")
    package_weight_kg = Column(Numeric(10, 3), nullable=False, comment="单件包装重量kg")
    provider_id = Column(BigInteger, ForeignKey("shipping_provider.id"), nullable=False, comment="物流供应商ID")
    provider_name = Column(String(200), nullable=False, comment="物流供应商名称快照")
    channel_id = Column(BigInteger, ForeignKey("shipping_channel.id"), nullable=False, comment="物流渠道ID")
    channel_name = Column(String(200), nullable=False, comment="物流渠道名称快照")
    currency = Column(String(10), default="CNY", comment="币种")
    actual_weight_kg = Column(Numeric(10, 4), nullable=False, comment="实际重量kg")
    volumetric_weight_kg = Column(Numeric(10, 4), nullable=False, comment="体积重量kg")
    chargeable_weight_kg = Column(Numeric(10, 4), nullable=False, comment="计费重量kg")
    base_shipping_fee = Column(Numeric(10, 2), nullable=False, comment="基础运费")
    surcharge_fee = Column(Numeric(10, 2), default=0, comment="固定附加费")
    fuel_surcharge_fee = Column(Numeric(10, 2), default=0, comment="燃油附加费")
    total_shipping_fee = Column(Numeric(10, 2), nullable=False, comment="总运费")
    calculation_detail = Column(Text, comment="计算说明")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class ImportBatch(Base):
    """导入批次"""
    __tablename__ = "import_batch"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    type = Column(String(30), nullable=False, comment="导入类型: product/sku/price/inventory")
    file_name = Column(String(255), comment="原始文件名")
    status = Column(String(20), nullable=False, default="pending", comment="pending/previewed/committed/failed")
    total_rows = Column(Integer, default=0, comment="总行数")
    success_count = Column(Integer, default=0, comment="成功行数")
    error_count = Column(Integer, default=0, comment="失败行数")
    error_summary = Column(Text, comment="错误摘要")
    created_by = Column(String(100), comment="操作人")
    created_at = Column(DateTime(timezone=True), server_default=func.now())
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())


class ImportBatchRow(Base):
    """导入行结果"""
    __tablename__ = "import_batch_row"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    batch_id = Column(BigInteger, ForeignKey("import_batch.id"), nullable=False, index=True)
    row_index = Column(Integer, nullable=False, comment="Excel行号")
    status = Column(String(20), nullable=False, default="pending", comment="pending/success/error")
    raw_data = Column(JSON, comment="原始行数据")
    error_message = Column(Text, comment="错误信息")
    created_at = Column(DateTime(timezone=True), server_default=func.now())

    batch = relationship("ImportBatch", backref="rows")


class PlatformFeeRule(Base):
    """平台费用规则"""
    __tablename__ = "platform_fee_rule"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    platform_id = Column(BigInteger, ForeignKey("platform.id"), nullable=False, index=True)
    country_code = Column(String(10), comment="国家代码，null表示平台默认")
    category_id = Column(BigInteger, ForeignKey("category.id"), comment="类目ID，null表示类目默认")
    fee_type = Column(String(30), nullable=False, comment="commission/fixed/payment/storage/other")
    fee_rate_pct = Column(Numeric(10, 4), default=0, comment="费率(%)")
    fixed_amount = Column(Numeric(12, 2), default=0, comment="固定费用")
    min_amount = Column(Numeric(12, 2), comment="最低费用")
    max_amount = Column(Numeric(12, 2), comment="最高费用")
    currency = Column(String(3), default="CNY", comment="币种")
    effective_from = Column(DateTime(timezone=True), comment="生效时间")
    effective_to = Column(DateTime(timezone=True), comment="失效时间")
    priority = Column(Integer, default=0, comment="优先级，小值优先")
    status = Column(String(20), default="active", comment="active/inactive")
    remark = Column(Text, comment="备注")
    created_at = Column(DateTime(timezone=True), server_default=func.now())
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now())

    platform = relationship("Platform", backref="fee_rules")
    category = relationship("Category", backref="fee_rules")


class Settlement(Base):
    """结算单 — 从平台导入的结算报告"""
    __tablename__ = "settlement"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    platform_id = Column(BigInteger, ForeignKey("platform.id"), nullable=False, comment="平台ID")
    settlement_no = Column(String(100), nullable=False, comment="结算单号")
    period_start = Column(DateTime(timezone=True), comment="结算周期开始")
    period_end = Column(DateTime(timezone=True), comment="结算周期结束")
    currency = Column(String(3), default="CNY", comment="币种")

    total_revenue = Column(Numeric(12, 2), default=0, comment="总收入")
    total_fee = Column(Numeric(12, 2), default=0, comment="总费用")
    total_refund = Column(Numeric(12, 2), default=0, comment="总退款")
    total_net = Column(Numeric(12, 2), default=0, comment="净收入")

    status = Column(String(20), default="pending", comment="状态: pending/reconciling/reconciled/closed")
    raw_data = Column(JSON, comment="原始数据(JSON)")

    imported_at = Column(DateTime(timezone=True), server_default=func.now(), comment="导入时间")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")

    platform = relationship("Platform", lazy="selectin")
    items = relationship("SettlementItem", back_populates="settlement_rel", cascade="all, delete-orphan", lazy="selectin")


class SettlementItem(Base):
    """结算明细 — 结算单中的每笔交易"""
    __tablename__ = "settlement_item"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    settlement_id = Column(BigInteger, ForeignKey("settlement.id"), nullable=False, index=True, comment="结算单ID")

    transaction_type = Column(String(30), nullable=False, comment="交易类型: order_sale/refund/shipping_fee/platform_fee/payment_fee/other")
    transaction_id = Column(String(100), comment="平台交易ID")

    order_no = Column(String(100), comment="关联订单号")
    order_id = Column(BigInteger, ForeignKey("sales_order.id"), comment="内部订单ID")
    sku_id = Column(BigInteger, ForeignKey("sku.id"), comment="SKU ID")

    amount = Column(Numeric(12, 2), default=0, comment="金额")
    fee = Column(Numeric(12, 2), default=0, comment="费用")
    net = Column(Numeric(12, 2), default=0, comment="净额")

    quantity = Column(Integer, default=0, comment="数量")

    occurred_at = Column(DateTime(timezone=True), comment="交易发生时间")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")

    # 对账状态
    reconciliation_status = Column(String(20), default="pending", comment="对账状态: pending/matched/unmatched/discrepancy")
    reconciliation_note = Column(Text, comment="对账备注")
    reconciled_at = Column(DateTime(timezone=True), comment="对账时间")
    reconciled_by = Column(String(100), comment="对账人")

    settlement_rel = relationship("Settlement", back_populates="items", lazy="selectin")


class OrderImport(Base):
    """订单导入记录"""
    __tablename__ = "order_import"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    platform_id = Column(BigInteger, ForeignKey("platform.id"), comment="平台ID")
    source_type = Column(String(50), nullable=False, comment="来源: ozon/shopee/wb/manual")
    file_name = Column(String(255), comment="文件名")

    total_rows = Column(Integer, default=0, comment="总行数")
    success_count = Column(Integer, default=0, comment="成功数")
    error_count = Column(Integer, default=0, comment="失败数")
    error_detail = Column(JSON, comment="错误详情")

    status = Column(String(20), default="pending", comment="状态: pending/processing/completed/failed")
    created_by = Column(String(100), comment="导入人")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class Warehouse(Base):
    """仓库"""
    __tablename__ = "warehouse"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    name = Column(String(200), nullable=False, comment="仓库名称")
    code = Column(String(50), unique=True, comment="仓库编码")
    address = Column(String(500), comment="地址")
    contact = Column(String(100), comment="联系人")
    phone = Column(String(50), comment="联系电话")
    is_default = Column(SmallInteger, default=0, comment="是否默认仓库")
    status = Column(SmallInteger, default=1, comment="状态: 0-禁用, 1-启用")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class AllocationRule(Base):
    """库存分配规则"""
    __tablename__ = "allocation_rule"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    name = Column(String(200), nullable=False, comment="规则名称")
    priority = Column(Integer, default=0, comment="优先级")
    rule_type = Column(String(50), nullable=False, comment="规则类型: percentage/fixed/priority")
    warehouse_id = Column(BigInteger, ForeignKey("warehouse.id"), nullable=False, comment="仓库ID")
    allocation_pct = Column(Numeric(5, 2), default=100, comment="分配百分比")
    allocation_qty = Column(Integer, default=0, comment="固定分配数量")
    status = Column(SmallInteger, default=1, comment="状态: 0-禁用, 1-启用")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class InventoryWarehouse(Base):
    """仓库库存（SKU在各仓库的库存）"""
    __tablename__ = "inventory_warehouse"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    sku_id = Column(BigInteger, ForeignKey("sku.id"), nullable=False, comment="SKU ID")
    warehouse_id = Column(BigInteger, ForeignKey("warehouse.id"), nullable=False, comment="仓库ID")
    quantity = Column(Integer, default=0, comment="库存数量")
    locked_quantity = Column(Integer, default=0, comment="锁定库存")
    safety_stock = Column(Integer, default=0, comment="安全库存")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")

    __table_args__ = (sa_UniqueConstraint("sku_id", "warehouse_id", name="uq_sku_warehouse"),)


class FinanceAccount(Base):
    """财务账户"""
    __tablename__ = "finance_account"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    name = Column(String(200), nullable=False, comment="账户名称")
    account_type = Column(String(50), nullable=False, comment="账户类型: platform/payment/bank/cash")
    platform_id = Column(BigInteger, ForeignKey("platform.id"), comment="关联平台ID")
    currency = Column(String(3), default="CNY", comment="币种")
    balance = Column(Numeric(14, 2), default=0, comment="余额")
    status = Column(SmallInteger, default=1, comment="状态: 0-禁用, 1-启用")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class FinanceTransaction(Base):
    """财务流水"""
    __tablename__ = "finance_transaction"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    account_id = Column(BigInteger, ForeignKey("finance_account.id"), nullable=False, comment="账户ID")
    transaction_type = Column(String(50), nullable=False, comment="类型: revenue/cost/fee/refund/transfer")
    amount = Column(Numeric(14, 2), nullable=False, comment="金额（正收入/负支出）")
    currency = Column(String(3), default="CNY", comment="币种")

    order_id = Column(BigInteger, ForeignKey("sales_order.id"), comment="关联订单ID")
    settlement_id = Column(BigInteger, ForeignKey("settlement.id"), comment="关联结算单ID")
    platform_id = Column(BigInteger, ForeignKey("platform.id"), comment="关联平台ID")

    description = Column(String(500), comment="描述")
    transaction_date = Column(DateTime(timezone=True), comment="交易日期")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")


class Notification(Base):
    """通知消息"""
    __tablename__ = "notification"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    user_id = Column(BigInteger, ForeignKey("user.id"), nullable=False, index=True, comment="接收用户ID")
    alert_type = Column(String(50), nullable=False, comment="类型: inventory_low_stock/inventory_out_of_stock/settlement_pending/settlement_discrepancy/listing_failed/order_pending/system")
    title = Column(String(200), nullable=False, comment="标题")
    content = Column(Text, comment="内容")
    link_url = Column(String(500), comment="跳转链接")
    severity = Column(String(20), default="info", comment="严重程度: info/warning/error/critical")
    is_read = Column(SmallInteger, default=0, comment="是否已读: 0-未读, 1-已读")
    source_id = Column(String(100), comment="来源ID (如 sku=123)")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")


class AlertRule(Base):
    """预警规则配置"""
    __tablename__ = "alert_rule"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    name = Column(String(200), nullable=False, comment="规则名称")
    alert_type = Column(String(50), nullable=False, unique=True, comment="预警类型")
    enabled = Column(SmallInteger, default=1, comment="是否启用: 0-禁用, 1-启用")
    config = Column(JSON, comment="配置 (如阈值、检查间隔等)")
    description = Column(String(500), comment="说明")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class ListingTaskItem(Base):
    """上架任务条目（每个产品×平台组合）"""
    __tablename__ = "listing_task_item"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    task_id = Column(BigInteger, ForeignKey("listing_task.id"), nullable=False, comment="任务ID")
    product_id = Column(BigInteger, ForeignKey("product.id"), nullable=False, comment="商品ID")
    platform_id = Column(BigInteger, ForeignKey("platform.id"), nullable=False, comment="平台ID")
    status = Column(String(50), default="pending", comment="状态: pending/in_progress/success/failed")
    result = Column(JSON, comment="发布结果（平台商品ID、URL等）")
    error_message = Column(Text, comment="错误信息")
    retry_count = Column(Integer, default=0, comment="重试次数")
    executed_at = Column(DateTime(timezone=True), comment="执行时间")

    __table_args__ = (
        sa_UniqueConstraint("task_id", "product_id", "platform_id", name="uq_task_product_platform"),
    )


class ProductImageGen(Base):
    """AI 生图生成记录"""
    __tablename__ = "product_image_gen"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    product_id = Column(BigInteger, ForeignKey("product.id"), nullable=False, comment="商品ID")
    prompt = Column(String(2000), nullable=False, comment="用户提示词")
    style = Column(String(50), default="product_white", comment="风格预设")
    negative_prompt = Column(String(1000), default="", comment="反向提示词")
    size = Column(String(20), default="1024x1024", comment="图片尺寸")
    requested_count = Column(Integer, default=1, comment="请求生成数量")
    status = Column(String(20), default="pending", comment="生成状态: pending/done/failed")
    image_urls = Column(JSON, comment="生成的图片URL列表")
    error_message = Column(String(1000), comment="失败原因")
    created_by = Column(BigInteger, comment="操作人")
    batch_id = Column(String(36), comment="批量生图的批次标识 (UUID)，便于一次批量结果查询")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class PromptTemplate(Base):
    """生图 Prompt 模板"""
    __tablename__ = "prompt_template"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    name = Column(String(200), nullable=False, comment="模板名称")
    description = Column(String(500), comment="模板描述")
    prompt = Column(String(2000), nullable=False, comment="正向提示词")
    negative_prompt = Column(String(1000), default="", comment="反向提示词")
    style = Column(String(50), default="product_white", comment="风格预设")
    size = Column(String(20), default="1024x1024", comment="图片尺寸")
    platform_code = Column(String(50), comment="关联平台（为空则通用）")
    is_shared = Column(SmallInteger, default=1, comment="是否团队共享: 0-私有, 1-共享")
    usage_count = Column(Integer, default=0, comment="使用次数")
    created_by = Column(BigInteger, comment="创建人")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class ProductCanvas(Base):
    """AI 生图画布"""
    __tablename__ = "product_canvases"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    product_id = Column(BigInteger, ForeignKey("product.id"), nullable=False, comment="关联商品ID")
    name = Column(String(200), default="未命名画布", comment="画布名称")
    layers = Column(JSON, comment="Fabric.js 序列化图层数据")
    thumbnail = Column(Text, nullable=True, comment="缩略图URL")
    created_by = Column(BigInteger, ForeignKey("user.id"), comment="创建人")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


# ========== 向后兼容导入（Agent 模型已移至 app/agent/models.py） ==========
from app.agent.models import (  # noqa: E402, F401
    AgentAction,
    AgentDecision,
    AgentEpisode,
    AgentEvolutionConfig,
    AgentNudge,
    HonchoProfile,
    PersonalRule,
    RuleConflict,
    RuleMarkChange,
    SpcControlLimit,
    SystemConfig,
)
