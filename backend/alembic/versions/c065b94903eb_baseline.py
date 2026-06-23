"""baseline

Revision ID: c065b94903eb
Revises:
Create Date: 2026-06-13 11:03:24.363800
"""

from alembic import op
from typing import Sequence, Union


# revision identifiers, used by Alembic.
revision: str = "c065b94903eb"
down_revision: Union[str, None] = None
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    from app.database import Base
    import app.models  # noqa: F401

    bind = op.get_bind()
    Base.metadata.create_all(bind=bind)


def downgrade() -> None:
    from app.database import Base
    import app.models  # noqa: F401

    bind = op.get_bind()
    Base.metadata.drop_all(bind=bind)
