import apiClient from '@/lib/api-client';
import type { CandidateFeedback, CandidateFeedbackInput, CreateCostEntryInput, CreateImageBudgetPolicyInput, CreateImageRuleSnapshotInput, CreateManualImportInput, CreateProductImageJobInput, CreateProductImageSetInput, CreateRightsGrantInput, DecideImageSetInput, FiveAxisReviewInput, ImageBudgetCharge, ImageBudgetPolicy, ImageBudgetReservation, ImageProcessorCapability, ImageReleaseAttestation, ImageRuleSnapshot, ImageSetDecision, IssueImageReleaseAttestationInput, ProductImageAsset, ProductImageCostEntry, ProductImageJob, ProductImageManualImport, ProductImageReview, ProductImageRightsGrant, ProductImageSet, RecipeSummary, ReconcileImageBudgetChargeInput, ReconcileImageBudgetNoChargeInput, SKUOption } from './types';

const ROOT = '/v1/product-images';

export function newImageIdempotencyKey(prefix: string): string {
  const suffix = globalThis.crypto?.randomUUID?.() ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`;
  return `${prefix}-${suffix}`;
}

export async function getImageProcessorCapabilities(): Promise<ImageProcessorCapability[]> {
  const result = await apiClient.get<ImageProcessorCapability[]>(`${ROOT}/capabilities`);
  if (!result.data) throw new Error('能力状态接口没有返回数据');
  return result.data;
}

export async function listImageJobs(): Promise<ProductImageJob[]> {
  const result = await apiClient.getPage<ProductImageJob>(`${ROOT}/tasks`);
  return (result.data ?? []).map((job) => job.output_url ? {
    ...job,
    outputs: [{ id: job.output_blob_id ?? `${job.id}-output`, asset_id: job.output_blob_id ?? '', url: job.output_url, sha256: job.output_blob_id }],
  } : job);
}

export async function listSKUOptions(): Promise<SKUOption[]> {
  const result = await apiClient.getPage<SKUOption>('/v1/skus', { page: '1', size: '100' });
  return result.data ?? [];
}

export async function uploadSourceImage(file: File): Promise<ProductImageAsset> {
  const form = new FormData();
  form.append('file', file);
  const result = await apiClient.upload<ProductImageAsset>(`${ROOT}/assets`, form);
  if (!result.data) throw new Error('图片上传成功但服务器没有返回资产');
  return result.data;
}

export async function createManualImport(input: CreateManualImportInput): Promise<ProductImageManualImport> {
  const form = new FormData();
  for (const [key, value] of Object.entries(input)) {
    if (value !== undefined) form.append(key, value instanceof File ? value : String(value));
  }
  const result = await apiClient.upload<ProductImageManualImport>(`${ROOT}/manual-imports`, form);
  if (!result.data) throw new Error('外部编辑结果已上传但服务器没有返回溯源记录');
  return result.data;
}

export async function listManualImports(): Promise<ProductImageManualImport[]> {
  const result = await apiClient.getPage<ProductImageManualImport>(`${ROOT}/manual-imports`);
  return result.data ?? [];
}

export async function createImageJob(input: CreateProductImageJobInput): Promise<ProductImageJob> {
  const result = await apiClient.post<ProductImageJob>(`${ROOT}/tasks`, input);
  if (!result.data) throw new Error('任务创建成功但服务器没有返回任务');
  return result.data;
}

export async function executeImageJob(id: number): Promise<unknown> {
  const idempotencyKey = newImageIdempotencyKey(`product-image-execute-${id}`);
  const result = await apiClient.post<unknown>(`${ROOT}/tasks/${encodeURIComponent(id)}/executions`, { idempotency_key: idempotencyKey });
  if (!result.data) throw new Error('任务已提交但服务器没有返回状态');
  return result.data;
}

export async function approveImageExecution(job: ProductImageJob): Promise<unknown> {
  const result = await apiClient.post<unknown>(`${ROOT}/tasks/${encodeURIComponent(job.id)}/execution-approvals`, {
    processor: job.processor,
    max_cost: job.max_cost,
    currency: job.currency,
    expected_version: job.version ?? 1,
  });
  if (!result.data) throw new Error('执行审批已创建但服务器没有返回记录');
  return result.data;
}

export async function fetchImageOutput(id: number): Promise<Blob> {
  return apiClient.getBlob(`${ROOT}/tasks/${encodeURIComponent(id)}/output/content`);
}

export async function createProductImageSet(input: CreateProductImageSetInput): Promise<ProductImageSet> {
  const result = await apiClient.post<ProductImageSet>(`${ROOT}/image-sets`, input);
  if (!result.data) throw new Error('图片集合已创建但服务器没有返回集合');
  return result.data;
}

export async function freezeProductImageSet(id: number): Promise<ProductImageSet> {
  const result = await apiClient.post<ProductImageSet>(`${ROOT}/image-sets/${encodeURIComponent(id)}/freeze`, {});
  if (!result.data) throw new Error('Owner 选择已提交但服务器没有返回冻结结果');
  return result.data;
}

export async function createRightsGrant(input: CreateRightsGrantInput): Promise<ProductImageRightsGrant> {
  const result = await apiClient.post<ProductImageRightsGrant>(`${ROOT}/rights-grants`, input);
  if (!result.data) throw new Error('权利授权已提交但服务器没有返回记录');
  return result.data;
}

export async function createFiveAxisReview(taskId: number, input: FiveAxisReviewInput): Promise<ProductImageReview> {
  const result = await apiClient.post<ProductImageReview>(`${ROOT}/tasks/${encodeURIComponent(taskId)}/reviews`, input);
  if (!result.data) throw new Error('五类审核已提交但服务器没有返回记录');
  return result.data;
}

export async function createCandidateFeedback(taskId: number, input: CandidateFeedbackInput): Promise<CandidateFeedback> {
  const result = await apiClient.post<CandidateFeedback>(`${ROOT}/tasks/${encodeURIComponent(taskId)}/feedback`, input);
  if (!result.data) throw new Error('候选反馈已提交但服务器没有返回记录');
  return result.data;
}

export async function getRecipeSummary(recipeKey: string, skuId: number): Promise<RecipeSummary> {
	const result = await apiClient.get<RecipeSummary>(`${ROOT}/recipes/${encodeURIComponent(recipeKey)}/summary`, { sku_id: String(skuId) });
  if (!result.data) throw new Error('作图配方统计接口没有返回数据');
  return result.data;
}

export async function createCostEntry(taskId: number, input: CreateCostEntryInput): Promise<ProductImageCostEntry> {
  const result = await apiClient.post<ProductImageCostEntry>(`${ROOT}/tasks/${encodeURIComponent(taskId)}/costs`, input);
  if (!result.data) throw new Error('成本记录已提交但服务器没有返回记录');
  return result.data;
}

export async function createImageRuleSnapshot(input: CreateImageRuleSnapshotInput): Promise<ImageRuleSnapshot> {
  const result = await apiClient.post<ImageRuleSnapshot>(`${ROOT}/rule-snapshots`, input);
  if (!result.data) throw new Error('渠道规则快照已提交但服务器没有返回记录');
  return result.data;
}

export async function decideImageSet(setId: number, input: DecideImageSetInput): Promise<ImageSetDecision> {
  const result = await apiClient.post<ImageSetDecision>(`${ROOT}/image-sets/${encodeURIComponent(setId)}/decisions`, input);
  if (!result.data) throw new Error('图片集合决定已提交但服务器没有返回记录');
  return result.data;
}

export async function issueImageReleaseAttestation(input: IssueImageReleaseAttestationInput): Promise<ImageReleaseAttestation> {
  const result = await apiClient.post<ImageReleaseAttestation>(`${ROOT}/release-attestations`, input);
  if (!result.data) throw new Error('发布证明签发请求成功但服务器没有返回证明');
  return result.data;
}

export async function getImageReleaseAttestation(id: number): Promise<ImageReleaseAttestation> {
  const result = await apiClient.get<ImageReleaseAttestation>(`${ROOT}/release-attestations/${encodeURIComponent(id)}`);
  if (!result.data) throw new Error('发布证明接口没有返回记录');
  return result.data;
}

export async function createImageBudgetPolicy(input: CreateImageBudgetPolicyInput): Promise<ImageBudgetPolicy> {
  const result = await apiClient.post<ImageBudgetPolicy>(`${ROOT}/budget-policies`, input);
  if (!result.data) throw new Error('预算政策已提交但服务器没有返回记录');
  return result.data;
}

export async function listImageBudgetPolicies(): Promise<ImageBudgetPolicy[]> {
  const result = await apiClient.get<ImageBudgetPolicy[]>(`${ROOT}/budget-policies`);
  return result.data ?? [];
}

export async function listImageBudgetReservations(): Promise<ImageBudgetReservation[]> {
  const result = await apiClient.get<ImageBudgetReservation[]>(`${ROOT}/budget-reservations`);
  return result.data ?? [];
}

export async function cancelImageBudgetReservation(id: number, reason: string): Promise<ImageBudgetReservation> {
  const result = await apiClient.post<ImageBudgetReservation>(`${ROOT}/budget-reservations/${encodeURIComponent(id)}/cancel`, { reason });
  if (!result.data) throw new Error('预算预占取消成功但服务器没有返回记录');
  return result.data;
}

export async function reconcileImageBudgetCharge(id: number, input: ReconcileImageBudgetChargeInput): Promise<ImageBudgetCharge> {
  const result = await apiClient.post<ImageBudgetCharge>(`${ROOT}/budget-reservations/${encodeURIComponent(id)}/charges`, input);
  if (!result.data) throw new Error('外部账单对账成功但服务器没有返回记录');
  return result.data;
}

export async function reconcileImageBudgetNoCharge(id: number, input: ReconcileImageBudgetNoChargeInput): Promise<ImageBudgetCharge> {
  const result = await apiClient.post<ImageBudgetCharge>(`${ROOT}/budget-reservations/${encodeURIComponent(id)}/no-charge-reconciliations`, input);
  if (!result.data) throw new Error('未扣费对账成功但服务器没有返回记录');
  return result.data;
}
