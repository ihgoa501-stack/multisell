import apiClient from '@/lib/api-client';
import type { PlatformTruth } from './types';

export async function getPlatformTruth(): Promise<PlatformTruth> {
  const result = await apiClient.get<PlatformTruth>('/v1/platform-truth');
  if (!result.data) throw new Error('平台真相合同为空');
  return result.data;
}
