"""recreate missing migration stub for f22e62fe526b

The DB applied this revision but the .py file was deleted.
This is a no-op stub that depends on 4ad5419fa71c (current head).

Revision ID: f22e62fe526b
Revises: 4ad5419fa71c
Create Date: 2026-06-22 20:50:00.000000
"""
from typing import Sequence, Union



# revision identifiers, used by Alembic.
revision: str = "f22e62fe526b"
down_revision: Union[str, None] = "4ad5419fa71c"
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    pass


def downgrade() -> None:
    pass
