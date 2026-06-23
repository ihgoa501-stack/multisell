"""add_product_image_gen

Revision ID: 20260618_01
Revises: a44be0e405da
Create Date: 2026-06-18 10:00:00.000000
"""

from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision: str = "20260618_01"
down_revision: Union[str, None] = "a44be0e405da"
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    op.create_table(
        "product_image_gen",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column(
            "product_id",
            sa.BigInteger(),
            sa.ForeignKey("product.id"),
            nullable=False,
            comment="商品ID",
        ),
        sa.Column(
            "prompt", sa.String(length=2000), nullable=False, comment="用户提示词"
        ),
        sa.Column(
            "style",
            sa.String(length=50),
            nullable=True,
            server_default="product_white",
            comment="风格预设",
        ),
        sa.Column(
            "negative_prompt",
            sa.String(length=1000),
            nullable=True,
            server_default="",
            comment="反向提示词",
        ),
        sa.Column(
            "size",
            sa.String(length=20),
            nullable=True,
            server_default="1024x1024",
            comment="图片尺寸",
        ),
        sa.Column(
            "requested_count",
            sa.Integer(),
            nullable=True,
            server_default=sa.text("1"),
            comment="请求生成数量",
        ),
        sa.Column(
            "status",
            sa.String(length=20),
            nullable=True,
            server_default="pending",
            comment="生成状态: pending/done/failed",
        ),
        sa.Column("image_urls", sa.JSON(), nullable=True, comment="生成的图片URL列表"),
        sa.Column(
            "error_message", sa.String(length=1000), nullable=True, comment="失败原因"
        ),
        sa.Column("created_by", sa.BigInteger(), nullable=True, comment="操作人"),
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
        sa.PrimaryKeyConstraint("id"),
    )
    op.create_index(
        "idx_product_image_gen_product_id", "product_image_gen", ["product_id"]
    )
    op.create_index(
        "idx_product_image_gen_created_at", "product_image_gen", ["created_at"]
    )


def downgrade() -> None:
    op.drop_index("idx_product_image_gen_created_at", table_name="product_image_gen")
    op.drop_index("idx_product_image_gen_product_id", table_name="product_image_gen")
    op.drop_table("product_image_gen")
