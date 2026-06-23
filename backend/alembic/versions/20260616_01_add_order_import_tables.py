"""add_order_import_tables

Revision ID: 20260616_01_add_order_import_tables
Revises: 20260615_10_add_cost_allocation
Create Date: 2026-06-16

"""

from alembic import op
import sqlalchemy as sa

# revision identifiers, used by Alembic.
revision = "20260616_01_add_order_import_tables"
down_revision = "20260615_10"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.create_table(
        "order_import_batch",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("adapter_code", sa.String(50), nullable=False, comment="适配器代码"),
        sa.Column("platform", sa.String(100), comment="平台名称"),
        sa.Column("store_name", sa.String(200), comment="店铺名称"),
        sa.Column(
            "source_filename", sa.String(500), nullable=False, comment="源文件名"
        ),
        sa.Column("row_count", sa.Integer(), server_default="0", comment="导入行数"),
        sa.Column(
            "created_order_count",
            sa.Integer(),
            server_default="0",
            comment="创建订单数",
        ),
        sa.Column(
            "skipped_duplicate_count",
            sa.Integer(),
            server_default="0",
            comment="跳过重复数",
        ),
        sa.Column("failed_count", sa.Integer(), server_default="0", comment="失败数"),
        sa.Column("imported_by", sa.String(100), comment="导入人"),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            comment="创建时间",
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            comment="更新时间",
        ),
        sa.PrimaryKeyConstraint("id"),
    )

    op.create_table(
        "order_import_item",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column(
            "batch_id",
            sa.BigInteger(),
            sa.ForeignKey("order_import_batch.id"),
            nullable=False,
            comment="批次ID",
        ),
        sa.Column("row_number", sa.Integer(), nullable=False, comment="原始行号"),
        sa.Column("platform", sa.String(100), comment="平台名称"),
        sa.Column("store_name", sa.String(200), comment="店铺名称"),
        sa.Column("platform_order_no", sa.String(200), comment="平台订单号"),
        sa.Column("order_no", sa.String(200), comment="系统订单号"),
        sa.Column(
            "order_id",
            sa.BigInteger(),
            sa.ForeignKey("sales_order.id"),
            comment="系统订单ID",
        ),
        sa.Column("sku_code", sa.String(100), nullable=False, comment="SKU编码"),
        sa.Column("quantity", sa.Integer(), nullable=False, comment="数量"),
        sa.Column("unit_price", sa.Float(), comment="单价"),
        sa.Column("currency", sa.String(10), server_default="CNY", comment="币种"),
        sa.Column("recipient_name", sa.String(100), comment="收件人"),
        sa.Column("recipient_phone", sa.String(50), comment="联系电话"),
        sa.Column("country_code", sa.String(10), comment="国家代码"),
        sa.Column("shipping_address", sa.String(500), comment="收货地址"),
        sa.Column("shipping_fee", sa.Float(), server_default="0", comment="运费"),
        sa.Column("tracking_number", sa.String(200), comment="追踪号"),
        sa.Column("paid_at", sa.String(50), comment="支付时间/日期"),
        sa.Column(
            "status",
            sa.String(50),
            nullable=False,
            comment="imported/created_order/skipped_duplicate/failed",
        ),
        sa.Column("failure_reason", sa.String(500), comment="失败原因"),
        sa.Column("raw_payload", sa.JSON(), comment="原始行数据"),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.func.now(),
            comment="创建时间",
        ),
        sa.PrimaryKeyConstraint("id"),
    )
    op.create_index("ix_order_import_item_batch_id", "order_import_item", ["batch_id"])


def downgrade() -> None:
    op.drop_index("ix_order_import_item_batch_id", table_name="order_import_item")
    op.drop_table("order_import_item")
    op.drop_table("order_import_batch")
