# Module: `order`

Package: `backend-go/internal/domain/order/`

**Base mount prefix:** `/api/v1`
**Required permission:** `order.read`

## API Routes

| Method | Path | Handler |
|--------|------|--------|
| `GET` | `/api/v1/order` | `h.List` |
| `POST` | `/api/v1/order` | `h.Create` |
| `DELETE` | `/api/v1/order/:id` | `h.Delete` |
| `GET` | `/api/v1/order/:id` | `h.Get` |
| `PUT` | `/api/v1/order/:id` | `h.Update` |
| `POST` | `/api/v1/order/:id/status` | `h.UpdateStatus` |
| `GET` | `/api/v1/order/summary` | `h.Summary` |

## Models

### `Order`
**DB table:** `sales_order`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `OrderNo` | `string` | `order_no` | `order_no` |  |
| `PlatformID` | `*int64` | `platform_id,omitempty` | `platform_id` |  |
| `Status` | `string` | `status` | `status` | default:pending |
| `TrackingNumber` | `string` | `tracking_number` | `tracking_number` |  |
| `RecipientName` | `string` | `recipient_name` | `recipient_name` |  |
| `RecipientPhone` | `string` | `recipient_phone` | `recipient_phone` |  |
| `ShippingAddress` | `string` | `shipping_address` | `shipping_address` |  |
| `TotalAmount` | `float64` | `total_amount` | `total_amount` | default:0 |
| `ShippingFee` | `float64` | `shipping_fee` | `shipping_fee` | default:0 |
| `PayAmount` | `float64` | `pay_amount` | `pay_amount` | default:0 |
| `PlatformFee` | `float64` | `platform_fee` | `platform_fee` | default:0 |
| `PaymentFee` | `float64` | `payment_fee` | `payment_fee` | default:0 |
| `OtherFee` | `float64` | `other_fee` | `other_fee` | default:0 |
| `ProductCost` | `float64` | `product_cost` | `product_cost` | default:0 |
| `ProfitAmount` | `float64` | `profit_amount` | `profit_amount` | default:0 |
| `ProfitMargin` | `float64` | `profit_margin` | `profit_margin` | default:0 |
| `PaymentMethod` | `string` | `payment_method` | `payment_method` |  |
| `Remark` | `string` | `remark` | `remark` |  |
| `PaidAt` | `*time.Time` | `paid_at,omitempty` | `paid_at` |  |
| `ShippedAt` | `*time.Time` | `shipped_at,omitempty` | `shipped_at` |  |
| `DeliveredAt` | `*time.Time` | `delivered_at,omitempty` | `delivered_at` |  |
| `CancelledAt` | `*time.Time` | `cancelled_at,omitempty` | `cancelled_at` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `updated_at` |  |

### `OrderItem`
**DB table:** `sales_order_item`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `OrderID` | `int64` | `order_id` | `order_id` | NOT NULL |
| `SkuID` | `int64` | `sku_id` | `sku_id` | NOT NULL |
| `ProductID` | `int64` | `product_id` | `product_id` | NOT NULL |
| `ProductName` | `string` | `product_name` | `product_name` | NOT NULL |
| `SkuCode` | `string` | `sku_code` | `sku_code` |  |
| `SpecDesc` | `string` | `spec_desc` | `spec_desc` |  |
| `UnitPrice` | `float64` | `unit_price` | `unit_price` | NOT NULL |
| `Quantity` | `int` | `quantity` | `quantity` | NOT NULL |
| `Subtotal` | `float64` | `subtotal` | `subtotal` | NOT NULL |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `OrderStatusLog`
**DB table:** `sales_order_status_log`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `id` | PK |
| `OrderID` | `int64` | `order_id` | `order_id` | NOT NULL |
| `FromStatus` | `string` | `from_status` | `from_status` |  |
| `ToStatus` | `string` | `to_status` | `to_status` | NOT NULL |
| `Operator` | `string` | `operator` | `operator` |  |
| `Remark` | `string` | `remark` | `remark` |  |
| `CreatedAt` | `time.Time` | `created_at` | `created_at` |  |

### `OrderDetail`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Order` | `Order` | `order` | `—` |  |
| `Items` | `[]OrderItem` | `items` | `—` |  |
| `StatusLogs` | `[]OrderStatusLog` | `status_logs` | `—` |  |

### `OrderResponse`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `ID` | `int64` | `id` | `—` |  |
| `OrderNo` | `string` | `order_no` | `—` |  |
| `PlatformID` | `*int64` | `platform_id,omitempty` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `TrackingNumber` | `string` | `tracking_number` | `—` |  |
| `RecipientName` | `string` | `recipient_name` | `—` |  |
| `RecipientPhone` | `string` | `recipient_phone` | `—` |  |
| `ShippingAddress` | `string` | `shipping_address` | `—` |  |
| `TotalAmount` | `float64` | `total_amount` | `—` |  |
| `ShippingFee` | `float64` | `shipping_fee` | `—` |  |
| `PayAmount` | `float64` | `pay_amount` | `—` |  |
| `PaymentMethod` | `string` | `payment_method` | `—` |  |
| `Remark` | `string` | `remark` | `—` |  |
| `PaidAt` | `*time.Time` | `paid_at,omitempty` | `—` |  |
| `ShippedAt` | `*time.Time` | `shipped_at,omitempty` | `—` |  |
| `DeliveredAt` | `*time.Time` | `delivered_at,omitempty` | `—` |  |
| `CancelledAt` | `*time.Time` | `cancelled_at,omitempty` | `—` |  |
| `CreatedAt` | `time.Time` | `created_at` | `—` |  |
| `UpdatedAt` | `time.Time` | `updated_at` | `—` |  |

### `OrderDetailResponse`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Order` | `OrderResponse` | `order` | `—` |  |
| `Items` | `[]OrderItem` | `items` | `—` |  |
| `StatusLogs` | `[]OrderStatusLog` | `status_logs` | `—` |  |

### `CreateOrderInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `OrderNo` | `string` | `order_no` | `—` |  |
| `PlatformID` | `*int64` | `platform_id` | `—` |  |
| `Status` | `string` | `status` | `—` |  |
| `TrackingNumber` | `string` | `tracking_number` | `—` |  |
| `RecipientName` | `string` | `recipient_name` | `—` |  |
| `RecipientPhone` | `string` | `recipient_phone` | `—` |  |
| `ShippingAddress` | `string` | `shipping_address` | `—` |  |
| `TotalAmount` | `*float64` | `total_amount` | `—` |  |
| `ShippingFee` | `*float64` | `shipping_fee` | `—` |  |
| `PayAmount` | `*float64` | `pay_amount` | `—` |  |
| `PaymentMethod` | `string` | `payment_method` | `—` |  |
| `Remark` | `string` | `remark` | `—` |  |
| `Items` | `[]OrderItemInput` | `items` | `—` |  |

### `OrderItemInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `SkuID` | `int64` | `sku_id` | `—` |  |
| `ProductID` | `int64` | `product_id` | `—` |  |
| `ProductName` | `string` | `product_name` | `—` |  |
| `SkuCode` | `string` | `sku_code` | `—` |  |
| `SpecDesc` | `string` | `spec_desc` | `—` |  |
| `UnitPrice` | `float64` | `unit_price` | `—` |  |
| `Quantity` | `int` | `quantity` | `—` |  |

### `UpdateOrderInput`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Status` | `*string` | `status` | `—` |  |
| `TrackingNumber` | `*string` | `tracking_number` | `—` |  |
| `RecipientName` | `*string` | `recipient_name` | `—` |  |
| `RecipientPhone` | `*string` | `recipient_phone` | `—` |  |
| `ShippingAddress` | `*string` | `shipping_address` | `—` |  |
| `PaymentMethod` | `*string` | `payment_method` | `—` |  |
| `Remark` | `*string` | `remark` | `—` |  |

### `OrderListFilter`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Search` | `string` | `` | `—` |  |
| `Status` | `string` | `` | `—` |  |
| `PlatformID` | `*int64` | `` | `—` |  |

### `OrderSummary`
**DB table:** `—`

| Field | Type | JSON | Column | Constraints |
|-------|------|------|--------|-------------|
| `Total` | `int64` | `total` | `—` |  |
| `ByStatus` | `map[string]int64` | `by_status` | `—` |  |
| `TotalRevenue` | `float64` | `total_revenue` | `—` |  |
| `TotalProfit` | `float64` | `total_profit` | `—` |  |

---
_Auto-generated by `docgen`. Do not edit manually._
