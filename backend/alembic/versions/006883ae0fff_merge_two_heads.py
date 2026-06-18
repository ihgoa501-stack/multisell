"""merge two heads

Revision ID: 006883ae0fff
Revises: 20260616_02_add_order_import_chain_fields, f7f1f102ba7b
Create Date: 2026-06-18 15:41:31.595193
"""
from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision: str = '006883ae0fff'
down_revision: Union[str, None] = ('20260616_02_add_order_import_chain_fields', 'f7f1f102ba7b')
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    pass


def downgrade() -> None:
    pass
