"""add shipping tables

Revision ID: 20260614_02
Revises: 20260614_01
Create Date: 2026-06-14 14:00:00
"""

from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects.postgresql import JSON


revision = "20260614_02"
down_revision = "20260614_01"
branch_labels = None
depends_on = None


def upgrade() -> None:
    # shipping_provider
    op.create_table(
        "shipping_provider",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column("name", sa.String(200), nullable=False),
        sa.Column("code", sa.String(50), nullable=True, unique=True),
        sa.Column("contact", sa.String(100), nullable=True),
        sa.Column("phone", sa.String(50), nullable=True),
        sa.Column("remark", sa.Text(), nullable=True),
        sa.Column(
            "status", sa.SmallInteger(), server_default=sa.text("1"), nullable=True
        ),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
        ),
        sa.PrimaryKeyConstraint("id"),
    )
    # shipping_channel
    op.create_table(
        "shipping_channel",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column(
            "provider_id",
            sa.BigInteger(),
            sa.ForeignKey("shipping_provider.id"),
            nullable=False,
        ),
        sa.Column("name", sa.String(200), nullable=False),
        sa.Column("code", sa.String(50), nullable=True),
        sa.Column(
            "volumetric_divisor",
            sa.Integer(),
            nullable=False,
            server_default=sa.text("6000"),
        ),
        sa.Column("cargo_types", JSON(), nullable=True),
        sa.Column("estimated_delivery_min", sa.Integer(), nullable=True),
        sa.Column("estimated_delivery_max", sa.Integer(), nullable=True),
        sa.Column(
            "currency", sa.String(10), server_default=sa.text("'CNY'"), nullable=True
        ),
        sa.Column(
            "sort_order", sa.Integer(), server_default=sa.text("0"), nullable=True
        ),
        sa.Column(
            "status", sa.SmallInteger(), server_default=sa.text("1"), nullable=True
        ),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
        ),
        sa.PrimaryKeyConstraint("id"),
    )
    # shipping_zone
    op.create_table(
        "shipping_zone",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column(
            "channel_id",
            sa.BigInteger(),
            sa.ForeignKey("shipping_channel.id"),
            nullable=False,
        ),
        sa.Column("country_code", sa.String(10), nullable=False),
        sa.Column("postal_code_from", sa.String(20), nullable=True),
        sa.Column("postal_code_to", sa.String(20), nullable=True),
        sa.Column(
            "status", sa.SmallInteger(), server_default=sa.text("1"), nullable=True
        ),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
        ),
        sa.PrimaryKeyConstraint("id"),
    )
    # shipping_quote_rule
    op.create_table(
        "shipping_quote_rule",
        sa.Column("id", sa.BigInteger(), autoincrement=True, nullable=False),
        sa.Column(
            "channel_id",
            sa.BigInteger(),
            sa.ForeignKey("shipping_channel.id"),
            nullable=False,
        ),
        sa.Column("rule_type", sa.String(50), nullable=False),
        sa.Column("priority", sa.Integer(), server_default=sa.text("0"), nullable=True),
        sa.Column(
            "min_weight_kg",
            sa.Numeric(10, 3),
            server_default=sa.text("0"),
            nullable=True,
        ),
        sa.Column("max_weight_kg", sa.Numeric(10, 3), nullable=True),
        sa.Column(
            "first_kg", sa.Numeric(10, 3), server_default=sa.text("0"), nullable=True
        ),
        sa.Column(
            "first_price", sa.Numeric(10, 2), server_default=sa.text("0"), nullable=True
        ),
        sa.Column(
            "additional_kg",
            sa.Numeric(10, 3),
            server_default=sa.text("0"),
            nullable=True,
        ),
        sa.Column(
            "additional_price",
            sa.Numeric(10, 2),
            server_default=sa.text("0"),
            nullable=True,
        ),
        sa.Column(
            "fixed_fee", sa.Numeric(10, 2), server_default=sa.text("0"), nullable=True
        ),
        sa.Column(
            "per_kg_price",
            sa.Numeric(10, 2),
            server_default=sa.text("0"),
            nullable=True,
        ),
        sa.Column("minimum_charge", sa.Numeric(10, 2), nullable=True),
        sa.Column("tier_config", JSON(), nullable=True),
        sa.Column(
            "surcharge_fixed",
            sa.Numeric(10, 2),
            server_default=sa.text("0"),
            nullable=True,
        ),
        sa.Column(
            "fuel_surcharge_pct",
            sa.Numeric(5, 2),
            server_default=sa.text("0"),
            nullable=True,
        ),
        sa.Column(
            "rounding_increment",
            sa.Numeric(10, 3),
            server_default=sa.text("0.1"),
            nullable=True,
        ),
        sa.Column("remark", sa.Text(), nullable=True),
        sa.Column(
            "status", sa.SmallInteger(), server_default=sa.text("1"), nullable=True
        ),
        sa.Column(
            "created_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
        ),
        sa.Column(
            "updated_at",
            sa.DateTime(timezone=True),
            server_default=sa.text("now()"),
            nullable=True,
        ),
        sa.PrimaryKeyConstraint("id"),
    )


def downgrade() -> None:
    op.drop_table("shipping_quote_rule")
    op.drop_table("shipping_zone")
    op.drop_table("shipping_channel")
    op.drop_table("shipping_provider")
