"""add zone scoped shipping quote rules

Revision ID: 20260615_01
Revises: 20260614_03
Create Date: 2026-06-15
"""

from alembic import op
import sqlalchemy as sa


revision = "20260615_01"
down_revision = "20260614_03"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.add_column(
        "shipping_quote_rule",
        sa.Column(
            "zone_id",
            sa.BigInteger(),
            nullable=True,
            comment="物流区域ID，空表示渠道全局规则",
        ),
    )
    op.create_foreign_key(
        "fk_shipping_quote_rule_zone_id",
        "shipping_quote_rule",
        "shipping_zone",
        ["zone_id"],
        ["id"],
    )
    op.create_index(
        "ix_shipping_quote_rule_zone_id", "shipping_quote_rule", ["zone_id"]
    )


def downgrade() -> None:
    op.drop_index("ix_shipping_quote_rule_zone_id", table_name="shipping_quote_rule")
    op.drop_constraint(
        "fk_shipping_quote_rule_zone_id", "shipping_quote_rule", type_="foreignkey"
    )
    op.drop_column("shipping_quote_rule", "zone_id")
