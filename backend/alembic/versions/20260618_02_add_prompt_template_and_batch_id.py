"""add_prompt_template_and_batch_id

Revision ID: 20260618_02
Revises: 20260618_01
Create Date: 2026-06-18 11:00:00.000000
"""

from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision: str = "20260618_02"
down_revision: Union[str, None] = "20260618_01"
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    # prompt_template 表
    op.create_table(
        "prompt_template",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("name", sa.String(length=200), nullable=False, comment="模板名称"),
        sa.Column(
            "description", sa.String(length=500), nullable=True, comment="模板描述"
        ),
        sa.Column(
            "prompt", sa.String(length=2000), nullable=False, comment="正向提示词"
        ),
        sa.Column(
            "negative_prompt",
            sa.String(length=1000),
            nullable=True,
            server_default="",
            comment="反向提示词",
        ),
        sa.Column(
            "style",
            sa.String(length=50),
            nullable=True,
            server_default="product_white",
            comment="风格预设",
        ),
        sa.Column(
            "size",
            sa.String(length=20),
            nullable=True,
            server_default="1024x1024",
            comment="图片尺寸",
        ),
        sa.Column(
            "platform_code",
            sa.String(length=50),
            nullable=True,
            comment="关联平台（为空则通用）",
        ),
        sa.Column(
            "is_shared",
            sa.SmallInteger(),
            nullable=True,
            server_default=sa.text("1"),
            comment="是否团队共享: 0-私有, 1-共享",
        ),
        sa.Column(
            "usage_count",
            sa.Integer(),
            nullable=True,
            server_default=sa.text("0"),
            comment="使用次数",
        ),
        sa.Column("created_by", sa.BigInteger(), nullable=True, comment="创建人"),
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
    # product_image_gen 加 batch_id 字段
    op.add_column(
        "product_image_gen",
        sa.Column(
            "batch_id",
            sa.String(length=36),
            nullable=True,
            comment="批量生图的批次标识 (UUID)",
        ),
    )
    op.create_index("idx_product_image_gen_batch_id", "product_image_gen", ["batch_id"])
    op.create_index(
        "idx_prompt_template_platform", "prompt_template", ["platform_code"]
    )
    op.create_index("idx_prompt_template_created_by", "prompt_template", ["created_by"])


def downgrade() -> None:
    op.drop_index("idx_prompt_template_created_by", table_name="prompt_template")
    op.drop_index("idx_prompt_template_platform", table_name="prompt_template")
    op.drop_index("idx_product_image_gen_batch_id", table_name="product_image_gen")
    op.drop_column("product_image_gen", "batch_id")
    op.drop_table("prompt_template")
