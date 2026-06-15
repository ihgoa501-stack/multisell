"""add logistics attributes

Revision ID: 20260614_01
Revises: 20260613_01
Create Date: 2026-06-14 11:30:00
"""

from alembic import op
import sqlalchemy as sa


revision = "20260614_01"
down_revision = "20260613_01"
branch_labels = None
depends_on = None


def upgrade() -> None:
    op.add_column("product", sa.Column("product_length_cm", sa.Numeric(10, 2), nullable=True))
    op.add_column("product", sa.Column("product_width_cm", sa.Numeric(10, 2), nullable=True))
    op.add_column("product", sa.Column("product_height_cm", sa.Numeric(10, 2), nullable=True))
    op.add_column("product", sa.Column("product_weight_kg", sa.Numeric(10, 2), nullable=True))
    op.add_column("product", sa.Column("package_length_cm", sa.Numeric(10, 2), nullable=True))
    op.add_column("product", sa.Column("package_width_cm", sa.Numeric(10, 2), nullable=True))
    op.add_column("product", sa.Column("package_height_cm", sa.Numeric(10, 2), nullable=True))
    op.add_column("product", sa.Column("package_weight_kg", sa.Numeric(10, 2), nullable=True))
    op.add_column(
        "product",
        sa.Column("cargo_type", sa.String(length=50), nullable=True, server_default="normal"),
    )
    op.add_column("sku", sa.Column("sku_length_cm", sa.Numeric(10, 2), nullable=True))
    op.add_column("sku", sa.Column("sku_width_cm", sa.Numeric(10, 2), nullable=True))
    op.add_column("sku", sa.Column("sku_height_cm", sa.Numeric(10, 2), nullable=True))
    op.add_column("sku", sa.Column("sku_weight_kg", sa.Numeric(10, 2), nullable=True))


def downgrade() -> None:
    op.drop_column("sku", "sku_weight_kg")
    op.drop_column("sku", "sku_height_cm")
    op.drop_column("sku", "sku_width_cm")
    op.drop_column("sku", "sku_length_cm")
    op.drop_column("product", "cargo_type")
    op.drop_column("product", "package_weight_kg")
    op.drop_column("product", "package_height_cm")
    op.drop_column("product", "package_width_cm")
    op.drop_column("product", "package_length_cm")
    op.drop_column("product", "product_weight_kg")
    op.drop_column("product", "product_height_cm")
    op.drop_column("product", "product_width_cm")
    op.drop_column("product", "product_length_cm")
