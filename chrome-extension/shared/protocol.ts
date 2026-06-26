// Shared protocol types for Chrome Extension <-> Backend WebSocket communication.

/** Structured product data extracted from a 1688 product page. */
export interface PageData {
  source_url: string;
  collected_at: string;
  driver: string;

  title: string;
  price_1688: number;
  price_min?: number | null;
  price_max?: number | null;
  currency: string;
  min_order_qty: number;

  images: string[];

  spec_variants?: SpecVariant[];

  supplier_name: string;
  supplier_id_1688: string;
  supplier_score?: number | null;

  description?: string;
  attributes?: Record<string, string>;

  package_weight_kg?: number | null;
  package_length_cm?: number | null;
  package_width_cm?: number | null;
  package_height_cm?: number | null;
  freight_cny?: number | null;
}

/** A single spec variant (e.g. "color:red; size:L"). */
export interface SpecVariant {
  spec: string;
  price: number;
  stock: number;
  image_url?: string;
}

// ─── WebSocket messages (Extension <-> Backend) ────────────────────────────

/** Backend requests the extension to fetch a product page. */
export interface FetchProductMessage {
  type: "fetch_product";
  id: string;
  payload: { url: string };
}

/** Extension sends extracted product data back to the backend. */
export interface FetchProductResult {
  type: "fetch_product_result";
  id: string;
  payload: { status: string; data: PageData };
}

/** Extension reports an error while fetching a product. */
export interface FetchProductError {
  type: "fetch_product_error";
  id: string;
  payload: { code: string; message: string };
}

/** Extension heartbeat ping. */
export interface PingMessage {
  type: "ping";
}

/** Backend heartbeat pong. */
export interface PongMessage {
  type: "pong";
}

/** Union of all messages the backend can send to the extension. */
export type WSIncomingMessage = FetchProductMessage | PongMessage;

/** Union of all messages the extension can send to the backend. */
export type WSOutgoingMessage = FetchProductResult | FetchProductError | PingMessage;

// ─── Internal extension messaging (Content Script <-> Background) ────────

/** Background requests the content script to extract product data from the current page. */
export interface ContentScriptFetchRequest {
  type: "fetch_product_from_page";
  requestId: string;
}

/** Content script responds with extraction result. */
export interface ContentScriptFetchResult {
  type: "fetch_product_from_page_result";
  requestId: string;
  payload:
    | { status: "ok"; data: PageData }
    | { code: string; message: string };
}

/** Union of all messages exchanged between content script and background. */
export type ExtensionMessage =
  | ContentScriptFetchRequest
  | ContentScriptFetchResult;

// ─── Popup <-> Background messaging ────────────────────────────────────────

export interface StatusRequest {
  type: "get_status";
}

export interface StatusResponse {
  type: "connection_status";
  status: "connected" | "disconnected" | "no_token" | "error";
}

export type PopupMessage = StatusRequest | StatusResponse;
