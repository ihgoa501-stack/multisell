from sqlalchemy import (
    Column,
    Integer,
    String,
    BigInteger,
    DateTime,
    JSON,
    Float,
    ForeignKey,
    func,
)
from app.database import Base


class OrderImportBatch(Base):
    """订单导入批次"""

    __tablename__ = "order_import_batch"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    adapter_code = Column(String(50), nullable=False, comment="适配器代码")
    platform = Column(String(100), comment="平台名称")
    store_name = Column(String(200), comment="店铺名称")
    source_filename = Column(String(500), nullable=False, comment="源文件名")
    row_count = Column(Integer, default=0, comment="导入行数")
    created_order_count = Column(Integer, default=0, comment="创建订单数")
    skipped_duplicate_count = Column(Integer, default=0, comment="跳过重复数")
    failed_count = Column(Integer, default=0, comment="失败数")
    imported_by = Column(String(100), comment="导入人")
    chain_status = Column(
        String(50),
        server_default="chain_pending",
        comment="chain_pending/chain_processed/chain_failed",
    )
    ledger_rebuilt_count = Column(Integer, default=0, comment="已重建账本订单数")
    exception_generated_count = Column(Integer, default=0, comment="生成异常数")
    chain_failure_count = Column(Integer, default=0, comment="链路处理失败数")
    processed_at = Column(DateTime(timezone=True), comment="链路处理时间")
    created_at = Column(
        DateTime(timezone=True), server_default=func.now(), comment="创建时间"
    )
    updated_at = Column(
        DateTime(timezone=True),
        server_default=func.now(),
        onupdate=func.now(),
        comment="更新时间",
    )


class OrderImportItem(Base):
    """订单导入行"""

    __tablename__ = "order_import_item"

    id = Column(BigInteger, primary_key=True, autoincrement=True)
    batch_id = Column(
        BigInteger,
        ForeignKey("order_import_batch.id"),
        nullable=False,
        comment="批次ID",
    )
    row_number = Column(Integer, nullable=False, comment="原始行号")
    platform = Column(String(100), comment="平台名称")
    store_name = Column(String(200), comment="店铺名称")
    platform_order_no = Column(String(200), comment="平台订单号")
    order_no = Column(String(200), comment="系统订单号")
    order_id = Column(BigInteger, ForeignKey("sales_order.id"), comment="系统订单ID")
    sku_code = Column(String(100), nullable=False, comment="SKU编码")
    quantity = Column(Integer, nullable=False, comment="数量")
    unit_price = Column(Float, comment="单价")
    currency = Column(String(10), server_default="CNY", comment="币种")
    recipient_name = Column(String(100), comment="收件人")
    recipient_phone = Column(String(50), comment="联系电话")
    country_code = Column(String(10), comment="国家代码")
    shipping_address = Column(String(500), comment="收货地址")
    shipping_fee = Column(Float, default=0, comment="运费")
    tracking_number = Column(String(200), comment="追踪号")
    paid_at = Column(String(50), comment="支付时间/日期")
    status = Column(
        String(50),
        nullable=False,
        comment="imported/created_order/skipped_duplicate/failed",
    )
    failure_reason = Column(String(500), comment="失败原因")
    chain_status = Column(
        String(50),
        server_default="chain_pending",
        comment="chain_pending/ledger_rebuilt/exception_generated/chain_failed",
    )
    chain_failure_reason = Column(String(500), comment="链路处理失败原因")
    raw_payload = Column(JSON, comment="原始行数据")
    created_at = Column(
        DateTime(timezone=True), server_default=func.now(), comment="创建时间"
    )
