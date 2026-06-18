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
    conn = op.get_bind()
    columns = [
        ("product", "product_length_cm", "NUMERIC(10, 2)"),
        ("product", "product_width_cm", "NUMERIC(10, 2)"),
        ("product", "product_height_cm", "NUMERIC(10, 2)"),
        ("product", "product_weight_kg", "NUMERIC(10, 2)"),
        ("product", "package_length_cm", "NUMERIC(10, 2)"),
        ("product", "package_width_cm", "NUMERIC(10, 2)"),
        ("product", "package_height_cm", "NUMERIC(10, 2)"),
        ("product", "package_weight_kg", "NUMERIC(10, 2)"),
        ("product", "cargo_type", "VARCHAR(50) DEFAULT 'normal'"),
        ("sku", "sku_length_cm", "NUMERIC(10, 2)"),
        ("sku", "sku_width_cm", "NUMERIC(10, 2)"),
        ("sku", "sku_height_cm", "NUMERIC(10, 2)"),
        ("sku", "sku_weight_kg", "NUMERIC(10, 2)"),
    ]
    for table, col, dtype in columns:
        conn.execute(sa.text(f"ALTER TABLE {table} ADD COLUMN IF NOT EXISTS {col} {dtype}"))


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
