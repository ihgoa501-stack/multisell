"""add product_canvases

Revision ID: 21608065ff2f
Revises: 006883ae0fff
Create Date: 2026-06-18 15:41:38.176876
"""
from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects.postgresql import JSON


revision: str = '21608065ff2f'
down_revision: Union[str, None] = '006883ae0fff'
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    op.create_table('product_canvases',
        sa.Column('id', sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column('product_id', sa.BigInteger(), nullable=False, comment='关联商品ID'),
        sa.Column('name', sa.String(200), server_default='未命名画布', comment='画布名称'),
        sa.Column('layers', JSON, comment='Fabric.js 序列化图层数据'),
        sa.Column('thumbnail', sa.Text(), nullable=True, comment='缩略图URL'),
        sa.Column('created_by', sa.BigInteger(), comment='创建人'),
        sa.Column('created_at', sa.DateTime(timezone=True), server_default=sa.func.now(), comment='创建时间'),
        sa.Column('updated_at', sa.DateTime(timezone=True), server_default=sa.func.now(), comment='更新时间'),
        sa.PrimaryKeyConstraint('id'),
        sa.ForeignKeyConstraint(['product_id'], ['product.id']),
        sa.ForeignKeyConstraint(['created_by'], ['user.id']),
    )


def downgrade() -> None:
    op.drop_table('product_canvases')
