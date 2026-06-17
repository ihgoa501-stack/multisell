"""数据库模型定义"""

from sqlalchemy import Column, BigInteger, String, Integer, Numeric, DateTime, Text, JSON, ForeignKey, SmallInteger, func
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


# ========== AI Agent 系统模型 ==========


class AgentDecision(Base):
    """Agent决策日志 (Hermes Episodic Memory)"""
    __tablename__ = "agent_decision"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    user_id = Column(BigInteger, ForeignKey("user.id"), nullable=False, comment="用户ID")
    agent_id = Column(String(20), nullable=False, comment="Agent标识: A3/A4/A5/A6/A7/G1/G2/G3")
    decision_point = Column(String(50), nullable=False, comment="决策点: acos_adjustment/stock_alert/discount_check 等")

    context_json = Column(JSON, nullable=False, comment="决策上下文")
    agent_output = Column(JSON, nullable=False, comment="Agent原始输出")
    final_decision = Column(JSON, nullable=False, comment="最终执行的决策")

    user_action = Column(String(20), nullable=False, comment="用户操作: accepted/modified/rejected/ignored")
    user_overrides = Column(JSON, comment="用户修改内容")
    user_feedback = Column(Text, comment="用户显式反馈")

    rules_applied = Column(JSON, comment="应用的规则ID列表")
    rule_overrides = Column(Integer, default=0, comment="规则覆盖次数")

    evolution_stage = Column(String(20), nullable=False, comment="进化阶段: observation/suggestion/semi_autonomous/full_autonomous")
    confidence = Column(Numeric(4, 3), comment="置信度 0.000-1.000")

    response_time_ms = Column(Integer, comment="响应耗时(ms)")
    token_count = Column(Integer, comment="Token消耗")

    session_id = Column(String(100), nullable=False, comment="会话ID")
    episode_id = Column(BigInteger, comment="Episode批次ID")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")


class PersonalRule(Base):
    """个人规则库"""
    __tablename__ = "personal_rule"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    user_id = Column(BigInteger, ForeignKey("user.id"), nullable=False, comment="用户ID")
    agent_id = Column(String(20), nullable=False, comment="Agent标识")
    decision_point = Column(String(50), nullable=False, comment="决策点")

    rule_type = Column(String(20), nullable=False, comment="规则类型: threshold/strategy/style/veto")
    rule_name = Column(String(100), nullable=False, comment="规则名称")
    rule_condition = Column(JSON, nullable=False, comment="规则条件")
    rule_action = Column(JSON, nullable=False, comment="规则动作")
    priority = Column(Integer, default=100, comment="优先级")

    source = Column(String(20), nullable=False, comment="来源: manual/nudge/auto_extracted/template")
    source_decisions = Column(JSON, comment="来源决策ID列表")
    status = Column(String(20), default="active", comment="状态: active/shadow/paused/retired")
    confidence = Column(Numeric(4, 3), default=0, comment="置信度")

    times_applied = Column(Integer, default=0, comment="应用次数")
    times_overridden = Column(Integer, default=0, comment="被覆盖次数")
    last_applied_at = Column(DateTime(timezone=True), comment="最后应用时间")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class AgentEpisode(Base):
    """Agent Episode汇总 (每N个决策一批)"""
    __tablename__ = "agent_episode"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    user_id = Column(BigInteger, ForeignKey("user.id"), nullable=False, comment="用户ID")
    agent_id = Column(String(20), nullable=False, comment="Agent标识")

    episode_number = Column(Integer, nullable=False, comment="Episode序号")
    decision_count = Column(Integer, nullable=False, comment="决策数量")
    episode_summary = Column(Text, comment="Episode摘要")
    key_insights = Column(JSON, comment="关键洞察")
    improvement_suggestions = Column(JSON, comment="改进建议")

    acceptance_rate = Column(Numeric(4, 3), comment="采纳率")
    avg_confidence = Column(Numeric(4, 3), comment="平均置信度")
    avg_response_ms = Column(Integer, comment="平均响应耗时")
    total_tokens = Column(Integer, comment="总Token消耗")

    nudge_triggered = Column(Integer, default=0, comment="Nudge触发次数")
    nudge_topics = Column(JSON, comment="Nudge话题")
    nudge_response = Column(Text, comment="Nudge用户响应")

    started_at = Column(DateTime(timezone=True), nullable=False, comment="开始时间")
    ended_at = Column(DateTime(timezone=True), nullable=False, comment="结束时间")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")


class HonchoProfile(Base):
    """Honcho用户模型 (辩证建模)"""
    __tablename__ = "honcho_profile"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    user_id = Column(BigInteger, ForeignKey("user.id"), nullable=False, unique=True, comment="用户ID")

    risk_tolerance = Column(String(30), default="moderate", comment="风险容忍度: conservative/moderate/aggressive")
    communication_style = Column(String(20), default="balanced", comment="沟通风格: concise/balanced/detailed")
    notification_prefs = Column(JSON, comment="通知偏好")
    agent_profiles = Column(JSON, nullable=False, default=lambda: {}, comment="各Agent配置档案")

    hypothesis_count = Column(Integer, default=0, comment="假设数量")
    confirmed_count = Column(Integer, default=0, comment="已验证数量")
    last_dialectic_at = Column(DateTime(timezone=True), comment="上次辩证对话时间")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
    updated_at = Column(DateTime(timezone=True), server_default=func.now(), onupdate=func.now(), comment="更新时间")


class RuleConflict(Base):
    """规则冲突日志"""
    __tablename__ = "rule_conflict"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    decision_id = Column(BigInteger, ForeignKey("agent_decision.id"), nullable=False, comment="决策ID")
    conflicting_rules = Column(JSON, nullable=False, comment="冲突规则ID列表")
    winner_rule_id = Column(BigInteger, nullable=False, comment="胜出规则ID")
    resolution = Column(String(20), nullable=False, comment="解决方式: auto_priority/user_choice/latest_wins")
    nudge_sent = Column(Integer, default=0, comment="是否已发送Nudge")
    nudge_resolved = Column(Integer, default=0, comment="是否已解决")
    user_choice = Column(BigInteger, comment="用户选择的规则ID")
    created_at = Column(DateTime(timezone=True), server_default=func.now(), comment="创建时间")
