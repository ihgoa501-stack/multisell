"""add order shipping snapshot and profit fields

Revision ID: 20260614_03
Revises: 20260614_02
Create Date: 2026-06-14
"""

from alembic import op
import sqlalchemy as sa


revision = "20260614_03"
down_revision = "20260614_02"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.add_column(
        "sales_order",
        sa.Column(
            "platform_fee",
            sa.Numeric(10, 2),
            server_default="0",
            nullable=True,
            comment="平台佣金/平台费",
        ),
    )
    op.add_column(
        "sales_order",
        sa.Column(
            "payment_fee",
            sa.Numeric(10, 2),
            server_default="0",
            nullable=True,
            comment="支付手续费",
        ),
    )
    op.add_column(
        "sales_order",
        sa.Column(
            "other_fee",
            sa.Numeric(10, 2),
            server_default="0",
            nullable=True,
            comment="其他费用",
        ),
    )
    op.add_column(
        "sales_order",
        sa.Column(
            "product_cost",
            sa.Numeric(10, 2),
            server_default="0",
            nullable=True,
            comment="商品成本",
        ),
    )
    op.add_column(
        "sales_order",
        sa.Column(
            "profit_amount",
            sa.Numeric(10, 2),
            server_default="0",
            nullable=True,
            comment="订单利润",
        ),
    )
    op.add_column(
        "sales_order",
        sa.Column(
            "profit_margin",
            sa.Numeric(10, 4),
            server_default="0",
            nullable=True,
            comment="利润率百分比",
        ),
    )

    op.create_table(
        "sales_order_shipping_snapshot",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("order_id", sa.BigInteger(), nullable=False, comment="订单ID"),
        sa.Column("sku_id", sa.BigInteger(), nullable=False, comment="试算SKU ID"),
        sa.Column("quantity", sa.Integer(), nullable=False, comment="试算数量"),
        sa.Column(
            "destination_country",
            sa.String(length=10),
            nullable=False,
            comment="目的地国家",
        ),
        sa.Column(
            "postal_code", sa.String(length=20), nullable=True, comment="目的地邮编"
        ),
        sa.Column(
            "cargo_type",
            sa.String(length=50),
            server_default="normal",
            nullable=True,
            comment="货品类型",
        ),
        sa.Column(
            "package_source",
            sa.String(length=20),
            nullable=True,
            comment="包装数据来源",
        ),
        sa.Column(
            "package_length_cm", sa.Numeric(10, 2), nullable=False, comment="包装长cm"
        ),
        sa.Column(
            "package_width_cm", sa.Numeric(10, 2), nullable=False, comment="包装宽cm"
        ),
        sa.Column(
            "package_height_cm", sa.Numeric(10, 2), nullable=False, comment="包装高cm"
        ),
        sa.Column(
            "package_weight_kg",
            sa.Numeric(10, 3),
            nullable=False,
            comment="单件包装重量kg",
        ),
        sa.Column(
            "provider_id", sa.BigInteger(), nullable=False, comment="物流供应商ID"
        ),
        sa.Column(
            "provider_name",
            sa.String(length=200),
            nullable=False,
            comment="物流供应商名称快照",
        ),
        sa.Column("channel_id", sa.BigInteger(), nullable=False, comment="物流渠道ID"),
        sa.Column(
            "channel_name",
            sa.String(length=200),
            nullable=False,
            comment="物流渠道名称快照",
        ),
        sa.Column(
            "currency",
            sa.String(length=10),
            server_default="CNY",
            nullable=True,
            comment="币种",
        ),
        sa.Column(
            "actual_weight_kg", sa.Numeric(10, 4), nullable=False, comment="实际重量kg"
        ),
        sa.Column(
            "volumetric_weight_kg",
            sa.Numeric(10, 4),
            nullable=False,
            comment="体积重量kg",
        ),
        sa.Column(
            "chargeable_weight_kg",
            sa.Numeric(10, 4),
            nullable=False,
            comment="计费重量kg",
        ),
        sa.Column(
            "base_shipping_fee", sa.Numeric(10, 2), nullable=False, comment="基础运费"
        ),
        sa.Column(
            "surcharge_fee",
            sa.Numeric(10, 2),
            server_default="0",
            nullable=True,
            comment="固定附加费",
        ),
        sa.Column(
            "fuel_surcharge_fee",
            sa.Numeric(10, 2),
            server_default="0",
            nullable=True,
            comment="燃油附加费",
        ),
        sa.Column(
            "total_shipping_fee", sa.Numeric(10, 2), nullable=False, comment="总运费"
        ),
        sa.Column("calculation_detail", sa.Text(), nullable=True, comment="计算说明"),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
            comment="创建时间",
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
            comment="更新时间",
        ),
        sa.ForeignKeyConstraint(["channel_id"], ["shipping_channel.id"]),
        sa.ForeignKeyConstraint(["order_id"], ["sales_order.id"]),
        sa.ForeignKeyConstraint(["provider_id"], ["shipping_provider.id"]),
        sa.ForeignKeyConstraint(["sku_id"], ["sku.id"]),
        sa.PrimaryKeyConstraint("id"),
        sa.UniqueConstraint(
            "order_id", name="uq_sales_order_shipping_snapshot_order_id"
        ),
    )
    op.create_index(
        "ix_order_shipping_snapshot_order_id",
        "sales_order_shipping_snapshot",
        ["order_id"],
    )


def downgrade() -> None:
    op.drop_index(
        "ix_order_shipping_snapshot_order_id",
        table_name="sales_order_shipping_snapshot",
    )
    op.drop_table("sales_order_shipping_snapshot")
    op.drop_column("sales_order", "profit_margin")
    op.drop_column("sales_order", "profit_amount")
    op.drop_column("sales_order", "product_cost")
    op.drop_column("sales_order", "other_fee")
    op.drop_column("sales_order", "payment_fee")
    op.drop_column("sales_order", "platform_fee")
