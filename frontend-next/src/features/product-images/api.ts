import apiClient from '@/lib/api-client';
import type { CreateCostEntryInput, CreateProductImageJobInput, CreateProductImageSetInput, CreateRightsGrantInput, FiveAxisReviewInput, ImageProcessorCapability, ProductImageAsset, ProductImageCostEntry, ProductImageJob, ProductImageReview, ProductImageRightsGrant, ProductImageSet } from './types';

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

export async function uploadSourceImage(file: File): Promise<ProductImageAsset> {
  const form = new FormData();
  form.append('file', file);
  const result = await apiClient.upload<ProductImageAsset>(`${ROOT}/assets`, form);
  if (!result.data) throw new Error('图片上传成功但服务器没有返回资产');
  return result.data;
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

export async function createCostEntry(taskId: number, input: CreateCostEntryInput): Promise<ProductImageCostEntry> {
  const result = await apiClient.post<ProductImageCostEntry>(`${ROOT}/tasks/${encodeURIComponent(taskId)}/costs`, input);
  if (!result.data) throw new Error('成本记录已提交但服务器没有返回记录');
  return result.data;
}
