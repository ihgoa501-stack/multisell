import type { ExistingPrivateCollectionSummary, PageData } from "./protocol.js";
import { getApiBaseUrl } from "./auth.js";

export interface PrivateCollectionPayload {
	schema_version: string;
	page_offer_id: string;
	price_model: PageData["price_model"];
  request_id: string;
  source_url: string;
  observed_at: string;
  parser_version: string;
  extension_version: string;
  raw_payload: PageData;
  title: string;
  price?: number;
  moq?: number;
  supplier_name?: string;
  supplier_business_id?: string;
  images?: string[];
  sku_variants?: PageData["spec_variants"];
  attributes?: Record<string, string>;
	field_statuses: PageData["field_statuses"];
	observation_intent?: "save_new_observation";
}

export interface SavedPrivateCollection {
  status: "saved";
  recordId: number;
  snapshotId: number;
  requestId: string;
  idempotentReplay: boolean;
  newObservation: boolean;
}

export class DuplicatePrivateCollectionError extends Error {
	constructor(
		public readonly recordId: number,
		public readonly snapshotId: number,
		public readonly existing: ExistingPrivateCollectionSummary,
	) {
		super("该1688商品已在私人采集箱");
		this.name = "DuplicatePrivateCollectionError";
	}
}

export function duplicateComparisonLines(page: PageData, existing: ExistingPrivateCollectionSummary): string[] {
	const same = (current: unknown, previous: unknown) => current === previous ? "相同" : "有变化";
	const text = (value: string | null | undefined) => value?.trim() || "未取得";
	const number = (value: number | null | undefined, suffix = "") =>
		Number.isFinite(value) && (value as number) > 0 ? `${value}${suffix}` : "未取得";
	const currentPrice = page.price_1688 > 0 ? page.price_1688 : null;
	const currentMOQ = page.min_order_qty > 0 ? page.min_order_qty : null;
	const currentSupplier = page.supplier_name?.trim() || "";
	const rows: Array<[string, unknown, unknown, string, string]> = [
		["标题", page.title.trim(), existing.title?.trim() || "", text(page.title), text(existing.title)],
		["价格", currentPrice, existing.price, number(currentPrice, "元"), number(existing.price, "元")],
		["起订量", currentMOQ, existing.moq, number(currentMOQ, "件"), number(existing.moq, "件")],
		["供应商", currentSupplier, existing.supplier_name?.trim() || "", text(currentSupplier), text(existing.supplier_name)],
		["SKU数", page.spec_variants?.length || 0, existing.sku_count, String(page.spec_variants?.length || 0), String(existing.sku_count)],
		["图片数", page.images?.length || 0, existing.image_count, String(page.images?.length || 0), String(existing.image_count)],
	];
	return [
		"本次页面 vs 已有观察：",
		...rows.map(([label, current, previous, currentText, previousText]) =>
			`${label}：本次 ${currentText}｜已有 ${previousText}（${same(current, previous)}）`),
		`已有观察时间：${existing.observed_at || "未取得"}`,
	];
}

export function isTrustedPrivateCollectionSource(senderURL: string, page: PageData): boolean {
	const senderOffer = senderURL.match(/^https:\/\/detail\.1688\.com\/offer\/(\d+)\.html(?:[?#].*)?$/i)?.[1];
	const listSender = /^https:\/\/(?!(?:detail)\.)[a-zA-Z0-9_-]+\.1688\.com\//i.test(senderURL);
	const payloadOffer = page.source_url.match(/^https:\/\/detail\.1688\.com\/offer\/(\d+)\.html(?:[?#].*)?$/i)?.[1];
	const identityMatches = Boolean(payloadOffer)
		&& page.offer_id_url === payloadOffer
		&& page.offer_id_page === payloadOffer;
	const trustedDetail = Boolean(senderOffer && senderOffer === payloadOffer);
	const trustedVisibleList = listSender && page.driver === "chrome_extension_list_visible"
		&& page.parser_version === "1688-list-visible-v1";
	return identityMatches && (trustedDetail || trustedVisibleList);
}

export interface PrivateCaptureFailureInput {
  requestId: string;
  sourceUrl: string;
  errorCode: "invalid_source_url" | "title_parse_failed" | "sku_parse_failed" | "invalid_payload";
  schemaVersion: string;
  extensionVersion: string;
  parserVersion: string;
  occurredAt: string;
}

export async function submitPrivateCaptureFailure(
  input: PrivateCaptureFailureInput,
  token: string,
  serverUrl: string,
  fetcher: typeof fetch = fetch,
): Promise<void> {
  const response = await fetcher(`${getApiBaseUrl(serverUrl)}/extension/sourcing-1688/private-collections/failures`, {
    method: "POST",
    headers: { "Authorization": `Bearer ${token}`, "Content-Type": "application/json" },
    body: JSON.stringify({
      request_id: input.requestId,
      source_url: input.sourceUrl,
      error_code: input.errorCode,
      schema_version: input.schemaVersion,
      extension_version: input.extensionVersion,
      parser_version: input.parserVersion,
      occurred_at: input.occurredAt,
    }),
  });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(typeof body?.message === "string" ? body.message : "采集失败记录未送达凌镜");
  }
}

/**
 * The only collection state allowed to survive a service-worker/browser
 * restart. Page contents deliberately stay in the active tab and are never
 * copied into extension storage.
 */
export interface PendingCollectionMarker {
  requestId: string;
  apiOrigin: string;
  createdAt: string;
  tabId?: number;
  sourceSummary?: string;
}

export type PendingCollectionMap = Record<string, PendingCollectionMarker>;

export type CollectionReconciliationResult =
  | { status: "saved"; requestId: string; recordId: number; snapshotId: number; sourceSummary?: string; tabId?: number }
  | { status: "not_saved"; requestId: string; message?: string; sourceSummary?: string; tabId?: number }
  | { status: "reconcile_required"; requestId: string; message: string; sourceSummary?: string; tabId?: number };

export type StoredCollectionRecovery = CollectionReconciliationResult & { recoveredAt: string };

export function mergeCollectionRecoveryHistory(
  prior: StoredCollectionRecovery[],
  result: CollectionReconciliationResult,
  marker: PendingCollectionMarker | undefined,
  recoveredAt: string,
  limit = 10,
): StoredCollectionRecovery[] {
  const recovered = {
    ...result,
    sourceSummary: marker?.sourceSummary,
    tabId: marker?.tabId,
    recoveredAt,
  } as StoredCollectionRecovery;
  return [recovered, ...prior.filter((item) => item.requestId !== result.requestId)].slice(0, limit);
}

export function describeCollectionRecovery(result: CollectionReconciliationResult): string {
  const identity = result.sourceSummary || `请求 ${result.requestId.slice(0, 18)}`;
  if (result.status === "saved") {
    return `已恢复 ${identity}：已保存到凌镜私人采集箱\n记录编号：#${result.recordId}`;
  }
  if (result.status === "not_saved") {
    return `已恢复 ${identity}：服务器确认未保存\n${result.message || "可以重新打开商品页采集"}`;
  }
  return `${identity} 的采集结果仍待确认\n请勿重复采集；联网后插件会继续核对`;
}

export function buildPendingCollectionMarker(
  requestId: string,
  serverUrl: string,
  createdAt: string,
  source?: { tabId?: number; sourceUrl?: string },
): PendingCollectionMarker {
  const marker: PendingCollectionMarker = {
    requestId,
    apiOrigin: new URL(getApiBaseUrl(serverUrl)).origin,
    createdAt,
  };
  if (Number.isInteger(source?.tabId) && (source?.tabId ?? -1) >= 0) marker.tabId = source!.tabId;
  if (source?.sourceUrl) {
    try {
      const url = new URL(source.sourceUrl);
      const offer = url.pathname.match(/\/offer\/(\d+)\.html/i)?.[1];
      marker.sourceSummary = offer ? `${url.hostname}/offer/${offer}` : url.hostname;
    } catch { /* source identity is optional */ }
  }
  return marker;
}

export function addPendingCollection(
  pending: PendingCollectionMap,
  marker: PendingCollectionMarker,
): PendingCollectionMap {
  return { ...pending, [marker.requestId]: marker };
}

export function removePendingCollection(
  pending: PendingCollectionMap,
  requestId: string,
): PendingCollectionMap {
  const next = { ...pending };
  delete next[requestId];
  return next;
}

export function buildPrivateCollectionPayload(
	page: PageData,
	requestId: string,
	extensionVersion: string,
	observationIntent?: "save_new_observation",
): PrivateCollectionPayload {
  const payload: PrivateCollectionPayload = {
	schema_version: page.schema_version,
	page_offer_id: page.offer_id_page,
	price_model: page.price_model,
    request_id: requestId,
    source_url: page.source_url,
    observed_at: page.collected_at,
    parser_version: page.parser_version,
    extension_version: extensionVersion,
    raw_payload: page,
    title: page.title,
	field_statuses: page.field_statuses,
	};
	if (observationIntent) payload.observation_intent = observationIntent;
  if (Number.isFinite(page.price_1688) && page.price_1688 > 0) payload.price = page.price_1688;
  if (Number.isInteger(page.min_order_qty) && page.min_order_qty >= 0) payload.moq = page.min_order_qty;
  if (page.supplier_name) payload.supplier_name = page.supplier_name;
  if (page.supplier_business_id) payload.supplier_business_id = page.supplier_business_id;
  if (page.images?.length) payload.images = page.images;
  if (page.spec_variants?.length) payload.sku_variants = page.spec_variants;
  if (page.attributes && Object.keys(page.attributes).length) payload.attributes = page.attributes;
  return payload;
}

export function parsePrivateCollectionResponse(value: unknown, expectedRequestId?: string): SavedPrivateCollection {
  const envelope = value as {
    code?: number;
    message?: string;
    data?: {
      status?: string;
      record_id?: number;
      snapshot_id?: number;
      request_id?: string;
      idempotent_replay?: boolean;
      new_observation?: boolean;
    };
  };
  const data = envelope?.data;
  if (envelope?.code !== 0 || data?.status !== "saved" || !data.record_id || !data.request_id) {
    throw new Error(envelope?.message || "凌镜没有返回有效记录编号，本次不能视为已保存");
  }
  if (expectedRequestId && data.request_id !== expectedRequestId) {
    throw new Error("凌镜返回了其他采集请求的结果，本次结果必须继续对账");
  }
  return {
    status: "saved",
    recordId: data.record_id,
    snapshotId: data.snapshot_id || 0,
    requestId: data.request_id,
    idempotentReplay: Boolean(data.idempotent_replay),
    newObservation: Boolean(data.new_observation),
  };
}

export async function submitPrivateCollection(
  page: PageData,
  requestId: string,
  extensionVersion: string,
  token: string,
	serverUrl: string,
	fetcher: typeof fetch = fetch,
	observationIntent?: "save_new_observation",
): Promise<SavedPrivateCollection> {
  const response = await fetcher(`${getApiBaseUrl(serverUrl)}/extension/sourcing-1688/private-collections`, {
    method: "POST",
    headers: {
      "Authorization": `Bearer ${token}`,
      "Content-Type": "application/json",
    },
		body: JSON.stringify(buildPrivateCollectionPayload(page, requestId, extensionVersion, observationIntent)),
	});
	const body = await response.json().catch(() => ({}));
	if (!response.ok) {
			if (response.status === 409 && body?.data?.status === "duplicate_requires_choice" && body.data.record_id && body.data.existing) {
				throw new DuplicatePrivateCollectionError(body.data.record_id, body.data.snapshot_id || 0, body.data.existing);
		}
    const message = typeof body?.message === "string" ? body.message : `凌镜保存失败（HTTP ${response.status}）`;
    throw new Error(message);
  }
  return parsePrivateCollectionResponse(body, requestId);
}

export async function queryPrivateCollectionRequest(
  requestId: string, token: string, serverUrl: string, fetcher: typeof fetch = fetch,
): Promise<CollectionReconciliationResult | null> {
  const response = await fetcher(`${getApiBaseUrl(serverUrl)}/extension/sourcing-1688/private-collections/requests/${encodeURIComponent(requestId)}`, {
    headers: { "Authorization": `Bearer ${token}` },
  });
  if (response.status === 404) return null;
  const body = await response.json().catch(() => ({}));
  if (!response.ok) throw new Error(typeof body?.message === "string" ? body.message : "无法确认采集结果");
  const data = body?.data;
  if (body?.code !== 0 || data?.request_id !== requestId) {
    throw new Error("凌镜返回了其他采集请求的状态，本次结果必须继续对账");
  }
  if (data.status === "saved") {
    const saved = parsePrivateCollectionResponse(body, requestId);
    return { status: "saved", requestId, recordId: saved.recordId, snapshotId: saved.snapshotId };
  }
  if (data.status === "not_saved") return {
    status: "not_saved", requestId,
    message: typeof data.safe_message === "string" && data.safe_message ? data.safe_message : "服务器确认本次没有保存",
  };
  if (data.status === "receiving" || data.status === "reconcile_required") {
    return {
      status: "reconcile_required",
      requestId,
      message: typeof data.safe_message === "string" && data.safe_message
        ? data.safe_message
        : data.status === "receiving" ? "服务器仍在处理本次采集，请稍后继续对账" : "服务器尚不能确认本次采集结果",
    };
  }
  throw new Error("凌镜返回了未知的采集状态，本次结果必须继续对账");
}

/**
 * Resolve an uncertain POST with a small, bounded read-after-write window.
 * A first 404 is not authoritative: the POST may have committed just before a
 * client timeout while the following lookup races request processing.
 */
export async function reconcilePrivateCollectionRequest(
  requestId: string,
  token: string,
  serverUrl: string,
  options: {
    fetcher?: typeof fetch;
    delaysMs?: readonly number[];
    wait?: (milliseconds: number) => Promise<void>;
  } = {},
): Promise<CollectionReconciliationResult> {
  const fetcher = options.fetcher ?? fetch;
  const delays = options.delaysMs ?? [250, 750, 1500];
  const wait = options.wait ?? ((milliseconds) => new Promise((resolve) => setTimeout(resolve, milliseconds)));

  for (let attempt = 0; attempt <= delays.length; attempt += 1) {
    try {
      const state = await queryPrivateCollectionRequest(requestId, token, serverUrl, fetcher);
      if (state) return state;
    } catch (error) {
      return {
        status: "reconcile_required",
        requestId,
        message: error instanceof Error ? error.message : "无法连接凌镜确认采集结果",
      };
    }
    if (attempt < delays.length) await wait(delays[attempt]);
  }
  return { status: "reconcile_required", requestId, message: "服务器尚未出现本次请求记录；不能据此确认未保存，请稍后继续对账" };
}
