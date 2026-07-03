/** Product hub types — mirrors backend domain/producthub/model.go */

export interface ProductMaster {
  id: number;
  name: string;
  description: string;
  category_id: number;
  brand_id: number;
  lifecycle_status: string;
  default_image: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  [key: string]: unknown;
}

export interface ProductVariant {
  id: number;
  product_master_id: number;
  sku_code: string;
  attributes: string;
  created_at: string;
}

export interface SupplierOffer {
  id: number;
  product_master_id: number;
  supplier_id: number;
  supply_price: number;
  min_order_qty: number;
  created_at: string;
}

export interface SampleRequest {
  id: number;
  product_master_id: number;
  supplier_id: number;
  status: string;
  created_at: string;
}

export interface CostVersion {
  id: number;
  product_master_id: number;
  total_cost: number;
  version: number;
  status: string;
  created_at: string;
}
