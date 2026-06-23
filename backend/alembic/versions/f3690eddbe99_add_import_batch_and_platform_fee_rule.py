"""add_import_batch_and_platform_fee_rule

Revision ID: f3690eddbe99
Revises: 20260617_02
Create Date: 2026-06-17 11:54:36.983850
"""

from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa

revision: str = "f3690eddbe99"
down_revision: Union[str, None] = "20260617_02"
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    op.create_table(
        "import_batch",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column(
            "type",
            sa.String(length=30),
            nullable=False,
            comment="导入类型: product/sku/price/inventory",
        ),
        sa.Column(
            "file_name", sa.String(length=255), nullable=True, comment="原始文件名"
        ),
        sa.Column(
            "status",
            sa.String(length=20),
            nullable=False,
            server_default="pending",
            comment="pending/previewed/committed/failed",
        ),
        sa.Column("total_rows", sa.Integer(), default=0, comment="总行数"),
        sa.Column("success_count", sa.Integer(), default=0, comment="成功行数"),
        sa.Column("error_count", sa.Integer(), default=0, comment="失败行数"),
        sa.Column("error_summary", sa.Text(), nullable=True, comment="错误摘要"),
        sa.Column("created_by", sa.String(length=100), nullable=True, comment="操作人"),
        sa.Column(
            "created_at", sa.DateTime(timezone=True), server_default=sa.text("now()")
        ),
        sa.Column(
            "updated_at", sa.DateTime(timezone=True), server_default=sa.text("now()")
        ),
        sa.PrimaryKeyConstraint("id"),
    )
    op.create_table(
        "import_batch_row",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("batch_id", sa.BigInteger(), nullable=False),
        sa.Column("row_index", sa.Integer(), nullable=False, comment="Excel行号"),
        sa.Column(
            "status",
            sa.String(length=20),
            nullable=False,
            server_default="pending",
            comment="pending/success/error",
        ),
        sa.Column("raw_data", sa.JSON(), nullable=True, comment="原始行数据"),
        sa.Column("error_message", sa.Text(), nullable=True, comment="错误信息"),
        sa.Column(
            "created_at", sa.DateTime(timezone=True), server_default=sa.text("now()")
        ),
        sa.ForeignKeyConstraint(
            ["batch_id"],
            ["import_batch.id"],
        ),
        sa.PrimaryKeyConstraint("id"),
    )
    op.create_index(
        op.f("ix_import_batch_row_batch_id"),
        "import_batch_row",
        ["batch_id"],
        unique=False,
    )

    # platform_fee_rule may already exist from earlier code; drop & recreate cleanly
    op.execute("DROP TABLE IF EXISTS platform_fee_rule CASCADE")
    op.create_table(
        "platform_fee_rule",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("platform_id", sa.BigInteger(), nullable=False),
        sa.Column(
            "country_code",
            sa.String(length=10),
            nullable=True,
            comment="国家代码，null表示平台默认",
        ),
        sa.Column(
            "category_id",
            sa.BigInteger(),
            nullable=True,
            comment="类目ID，null表示类目默认",
        ),
        sa.Column(
            "fee_type",
            sa.String(length=30),
            nullable=False,
            comment="commission/fixed/payment/storage/other",
        ),
        sa.Column("fee_rate_pct", sa.Numeric(10, 4), default=0, comment="费率(%)"),
        sa.Column("fixed_amount", sa.Numeric(12, 2), default=0, comment="固定费用"),
        sa.Column("min_amount", sa.Numeric(12, 2), nullable=True, comment="最低费用"),
        sa.Column("max_amount", sa.Numeric(12, 2), nullable=True, comment="最高费用"),
        sa.Column("currency", sa.String(length=3), default="CNY", comment="币种"),
        sa.Column(
            "effective_from",
            sa.DateTime(timezone=True),
            nullable=True,
            comment="生效时间",
        ),
        sa.Column(
            "effective_to",
            sa.DateTime(timezone=True),
            nullable=True,
            comment="失效时间",
        ),
        sa.Column("priority", sa.Integer(), default=0, comment="优先级，小值优先"),
        sa.Column(
            "status", sa.String(length=20), default="active", comment="active/inactive"
        ),
        sa.Column("remark", sa.Text(), nullable=True, comment="备注"),
        sa.Column(
            "created_at", sa.DateTime(timezone=True), server_default=sa.text("now()")
        ),
        sa.Column(
            "updated_at", sa.DateTime(timezone=True), server_default=sa.text("now()")
        ),
        sa.ForeignKeyConstraint(
            ["platform_id"],
            ["platform.id"],
        ),
        sa.ForeignKeyConstraint(
            ["category_id"],
            ["category.id"],
        ),
        sa.PrimaryKeyConstraint("id"),
    )
    op.create_index(
        op.f("ix_platform_fee_rule_platform_id"),
        "platform_fee_rule",
        ["platform_id"],
        unique=False,
    )


def downgrade() -> None:
    op.drop_index(
        op.f("ix_platform_fee_rule_platform_id"), table_name="platform_fee_rule"
    )
    op.drop_table("platform_fee_rule")
    op.drop_index(op.f("ix_import_batch_row_batch_id"), table_name="import_batch_row")
    op.drop_table("import_batch_row")
    op.drop_table("import_batch")
