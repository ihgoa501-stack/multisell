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
    stock = Column(Integer, default=0, comment="库存")
    lock_stock = Column(Integer, default=0, comment="锁定库存")
    warning_stock = Column(Integer, default=0, comment="安全库存预警")
    weight = Column(Numeric(10, 2), default=0, comment="重量(kg)")
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
