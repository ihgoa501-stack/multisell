'use client';

import { useEffect, useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert, Button, Card, Checkbox, Col, Descriptions, Empty, Form, Image, Input, InputNumber, List, Modal, Row, Select, Space, Spin, Tag, Typography, Upload,
} from 'antd';
import type { UploadProps } from 'antd';
import { CloudUploadOutlined, PlayCircleOutlined, ReloadOutlined } from '@ant-design/icons';
import PageHeader from '@/components/ui/PageHeader';
import {
  approveImageExecution, cancelImageBudgetReservation, createCandidateFeedback, createCostEntry, createFiveAxisReview, createImageBudgetPolicy, createImageJob, createImageRuleSnapshot, createManualImport, createProductImageSet, createRightsGrant, decideImageSet, executeImageJob, fetchImageOutput, freezeProductImageSet, getImageProcessorCapabilities, getImageReleaseAttestation, getRecipeSummary, issueImageReleaseAttestation, listImageBudgetPolicies, listImageBudgetReservations, listImageJobs, listManualImports, listSKUOptions, newImageIdempotencyKey, reconcileImageBudgetCharge, reconcileImageBudgetNoCharge, uploadSourceImage,
} from './api';
import type { CandidateFeedbackInput, CandidateReasonCode, CreateCostEntryInput, CreateImageBudgetPolicyInput, CreateManualImportInput, CreateProductImageJobInput, CreateRightsGrantInput, FiveAxisReviewInput, ImageBudgetReservation, ImageGateStatus, ImageProcessorCapability, ImageReleaseAttestation, ImageRuleSnapshot, ImageSetDecision, ProductImageAsset, ProductImageJob, ProductImageJobStatus, ProductImageRole, ProductImageSet, ReconcileImageBudgetChargeInput } from './types';

type TaskScope = Pick<CreateProductImageJobInput, 'processor' | 'operation' | 'purpose' | 'channel' | 'region'>;
type TaskFormValues = TaskScope & {
  sku_id: number; recipe_key: string; recipe_version: number; parent_task_id?: number; candidate_round: number;
  scene_structure: string; prompt?: string; negative_prompt?: string; model: string; model_version: string;
  parameters_json: string; must_not_change: string;
  max_cost?: string;
};
type FeedbackFormValues = Omit<CandidateFeedbackInput, 'asset_sha256' | 'idempotency_key' | 'expected_version' | 'error_regions'> & { error_regions_json: string };
type InputRightsFields = Pick<CreateRightsGrantInput, 'can_copy' | 'can_modify' | 'can_third_party_ai' | 'can_cross_border' | 'grantor' | 'rights_chain' | 'evidence_sha256' | 'owner_verified'>;

const photoroomOperations = [
  { value: 'PHOTOROOM_REMOVE_BACKGROUND_SANDBOX', label: '移除背景' },
  { value: 'PHOTOROOM_WHITE_BACKGROUND_SANDBOX', label: '生成纯白背景' },
  { value: 'PHOTOROOM_AI_SHADOW_SANDBOX', label: '添加 AI 阴影' },
] as const;

const statusPresentation: Record<ProductImageJobStatus, { label: string; color: string }> = {
  pending: { label: '待执行', color: 'default' }, queued: { label: '排队中', color: 'processing' },
  created: { label: '待执行', color: 'default' },
  running: { label: '处理中', color: 'processing' }, completed: { label: '已完成', color: 'success' },
  succeeded: { label: '已完成', color: 'success' }, failed: { label: '失败', color: 'error' }, reconcile_required: { label: '待对账', color: 'warning' },
  QUEUED: { label: '待执行', color: 'default' }, RUNNING: { label: '处理中', color: 'processing' }, READY: { label: '已完成', color: 'success' }, FAILED: { label: '失败', color: 'error' }, RECONCILE_REQUIRED: { label: '待对账', color: 'warning' },
};

function completePaidExecutionContext(job?: ProductImageJob): boolean {
  const recipe = job?.recipe_manifest;
  return Boolean(job?.version && job.sku_id && job.recipe_key && job.recipe_version && job.recipe_hash && job.manifest_hash
    && job.operation && job.processor && job.region && job.max_cost && moneyUnits(job.max_cost) && moneyUnits(job.max_cost)! > BigInt(0)
    && recipe?.reference_asset_ids?.length && recipe.scene_structure?.trim() && recipe.prompt?.trim()
    && recipe.model?.trim() && recipe.model_version?.trim() && recipe.must_not_change?.length);
}

function unresolvedPaidOutcome(job?: ProductImageJob): boolean {
  return Boolean(job && ['RECONCILE_REQUIRED', 'reconcile_required', 'FAILED', 'failed'].includes(job.status)
    && !['NO_CHARGE_CONFIRMED', 'CHARGED_OUTPUT_UNRECOVERABLE'].includes(job.error_code ?? ''));
}

function errorText(error: unknown) {
  if (error instanceof Error) return error.message;
  if (error && typeof error === 'object' && 'errorFields' in error) {
    const fields = (error as { errorFields?: Array<{ errors?: string[] }> }).errorFields ?? [];
    return fields.flatMap((field) => field.errors ?? []).join('；') || '表单校验未通过';
  }
  return '发生未知错误';
}

function moneyUnits(value: string): bigint | undefined {
  if (!/^(0|[1-9][0-9]*)(\.[0-9]{1,4})?$/.test(value)) return undefined;
  const [whole, fraction = ''] = value.split('.');
  return BigInt(whole) * BigInt(10000) + BigInt(fraction.padEnd(4, '0'));
}

function formatMoney(units: bigint): string {
  const zero = BigInt(0);
  const scale = BigInt(10000);
  const sign = units < zero ? '-' : '';
  const absolute = units < zero ? -units : units;
  const fraction = String(absolute % scale).padStart(4, '0').replace(/0+$/, '');
  return `${sign}${absolute / scale}${fraction ? `.${fraction}` : ''}`;
}

function ProtectedOutput({ jobId }: { jobId: number }) {
  const output = useQuery({ queryKey: ['product-images', 'output', jobId], queryFn: () => fetchImageOutput(jobId), staleTime: Infinity });
  const [url, setURL] = useState<string>();
  useEffect(() => { if (!output.data) return; const next = URL.createObjectURL(output.data); setURL(next); return () => URL.revokeObjectURL(next); }, [output.data]);
  if (output.isLoading) return <Spin />;
  if (output.error || !url) return <Alert type="error" message="候选图片读取失败" description={errorText(output.error)} />;
  return <Image src={url} alt="图片处理候选" width="100%" />;
}

export default function ProductImagesWorkspace() {
  const client = useQueryClient();
  const defaultRecipeKey = useMemo(() => newImageIdempotencyKey('recipe'), []);
  const [pageOpenedAt] = useState(() => Date.now());
  const [source, setSource] = useState<ProductImageAsset>();
  const [inputRightsFingerprint, setInputRightsFingerprint] = useState<string>();
  const [selected, setSelected] = useState<number[]>([]);
  const [roles, setRoles] = useState<Record<number, ProductImageRole>>({});
  const [imageSet, setImageSet] = useState<ProductImageSet>();
  const [governanceJobId, setGovernanceJobId] = useState<number>();
  const [rightsRecorded, setRightsRecorded] = useState(false);
  const [reviewRecorded, setReviewRecorded] = useState(false);
  const [manualFile, setManualFile] = useState<File>();
  const [ruleSnapshot, setRuleSnapshot] = useState<ImageRuleSnapshot>();
  const [setDecision, setSetDecision] = useState<ImageSetDecision>();
  const [attestation, setAttestation] = useState<ImageReleaseAttestation>();
  const [attestationLookupId, setAttestationLookupId] = useState<number>();
  const [budgetReservationId, setBudgetReservationId] = useState<number>();
  const [billEvidenceConfirmed, setBillEvidenceConfirmed] = useState(false);
  const [chargedNoOutputConfirmed, setChargedNoOutputConfirmed] = useState(false);
  const [noChargeEvidenceConfirmed, setNoChargeEvidenceConfirmed] = useState(false);
  const [paidExecutionJob, setPaidExecutionJob] = useState<ProductImageJob>();
  const [setForm] = Form.useForm<{ listing_id: number; channel: string; locale: string }>();
  const [taskForm] = Form.useForm<TaskFormValues>();
  const [inputRightsForm] = Form.useForm<InputRightsFields>();
  const [rightsForm] = Form.useForm<CreateRightsGrantInput>();
  const [reviewForm] = Form.useForm<FiveAxisReviewInput>();
  const [costForm] = Form.useForm<CreateCostEntryInput>();
  const [manualForm] = Form.useForm<Omit<CreateManualImportInput, 'file' | 'parent_asset_id' | 'parent_asset_sha256' | 'source_observed_at' | 'idempotency_key'>>();
  const [ruleForm] = Form.useForm<{ channel: string; site: string; locale: string; category_id: number; rules_json: string; expires_at?: string }>();
  const [decisionForm] = Form.useForm<{ decision: 'approved' | 'rejected'; reason: string }>();
  const [attestationForm] = Form.useForm<{ platform_account_id: number; site: string; ttl_seconds: number }>();
  const [budgetPolicyForm] = Form.useForm<Omit<CreateImageBudgetPolicyInput, 'idempotency_key'>>();
  const [budgetCancelForm] = Form.useForm<{ reason: string }>();
  const [budgetChargeForm] = Form.useForm<Omit<ReconcileImageBudgetChargeInput, 'idempotency_key'>>();
  const [budgetNoChargeForm] = Form.useForm<{ evidence_sha256: string; observed_at: string; reason: string }>();
  const [feedbackForm] = Form.useForm<FeedbackFormValues>();
  const jobs = useQuery({
    queryKey: ['product-images', 'jobs'], queryFn: listImageJobs, retry: false,
    refetchInterval: (query) => query.state.data?.some((job) => ['queued', 'running', 'RUNNING'].includes(job.status)) ? 2000 : false,
  });
  const capabilities = useQuery({
    queryKey: ['product-images', 'capabilities'], queryFn: getImageProcessorCapabilities, retry: false,
  });
  const manualImports = useQuery({ queryKey: ['product-images', 'manual-imports'], queryFn: listManualImports, retry: false });
  const skuOptions = useQuery({ queryKey: ['product-images', 'skus'], queryFn: listSKUOptions, retry: false });
  useEffect(() => {
    if (skuOptions.data?.length === 1 && !taskForm.getFieldValue('sku_id')) taskForm.setFieldValue('sku_id', skuOptions.data[0].id);
  }, [skuOptions.data, taskForm]);
  const budgetPolicies = useQuery({ queryKey: ['product-images', 'budget-policies'], queryFn: listImageBudgetPolicies, retry: false });
  const budgetReservations = useQuery({ queryKey: ['product-images', 'budget-reservations'], queryFn: listImageBudgetReservations, retry: false });
  const deterministic = capabilities.data?.find((capability) => capability.code === 'deterministic');
  const deterministicAvailable = deterministic?.configured === true;
  const photoroom = capabilities.data?.find((capability) => capability.code === 'photoroom');
  const photoroomCanaryAvailable = photoroom?.configured === true
    && photoroom.availability === 'available'
    && photoroom.safety_level === 'sandbox_only'
    && photoroom.provider_environment === 'sandbox'
    && photoroom.watermarked === true
    && photoroom.non_publishable === true
    && photoroom.quota_available === true
    && (photoroom.quota_remaining ?? 0) > 0;
  const openai = capabilities.data?.find((capability) => capability.code === 'openai');
  const openAIAvailable = openai?.configured === true
    && openai.availability === 'available'
    && openai.safety_level === 'production_paid'
    && openai.provider_environment === 'production'
    && openai.region === 'us'
    && openai.watermarked !== true
    && openai.non_publishable !== true;
  const selectedProcessor = Form.useWatch('processor', taskForm) ?? 'deterministic';
  const parentTaskID = Form.useWatch('parent_task_id', taskForm);
  const feedbackOutcome = Form.useWatch('outcome', feedbackForm) ?? 'selected';
  const isPhotoroomScope = selectedProcessor === 'photoroom';
  const isOpenAIScope = selectedProcessor === 'openai';
  const isExternalScope = isPhotoroomScope || isOpenAIScope;
  const selectedProcessorAvailable = isPhotoroomScope ? photoroomCanaryAvailable : isOpenAIScope ? openAIAvailable : deterministicAvailable;
  const refresh = () => client.invalidateQueries({ queryKey: ['product-images', 'jobs'] });
  const upload = useMutation({ mutationFn: uploadSourceImage, onSuccess: (asset) => { setSource(asset); setInputRightsFingerprint(undefined); } });
  const taskScopeFingerprint = (asset: ProductImageAsset, scope: TaskScope) => [asset.id, asset.sha256, scope.processor, scope.purpose.trim().toLowerCase(), scope.channel.trim().toLowerCase(), scope.region].join(':');
  const inputRights = useMutation({ mutationFn: async () => {
    if (!source) throw new Error('请先上传原图');
    const scope = await taskForm.validateFields();
    const values = await inputRightsForm.validateFields();
    const grant = await createRightsGrant({
      asset_id: source.id, asset_sha256: source.sha256, ...values,
      can_third_party_ai: values.can_third_party_ai, can_cross_border: values.can_cross_border, can_commercial_publish: false,
      can_platform_sublicense: false, trademark_cleared: false, likeness_cleared: false,
      purpose: scope.purpose, jurisdiction: isExternalScope ? 'us' : 'local', channel: scope.channel,
      provider: scope.processor, region: scope.region, valid_from: new Date().toISOString(),
      idempotency_key: newImageIdempotencyKey(`image-input-rights-${source.id}`), expected_version: 1,
    });
    return { grant, fingerprint: taskScopeFingerprint(source, scope) };
  }, onSuccess: ({ fingerprint }) => setInputRightsFingerprint(fingerprint) });
  const create = useMutation({
    mutationFn: async () => {
      const values = await taskForm.validateFields();
      const parent = values.parent_task_id ? (jobs.data ?? []).find((job) => job.id === values.parent_task_id) : undefined;
      const assetId = parent?.asset_id ?? source?.id;
      if (!assetId) throw new Error('请先上传原图，或从已有候选载入返工配方');
      if (!parent && (!source || inputRightsFingerprint !== taskScopeFingerprint(source, values))) throw new Error('必须先保存与当前原图及处理范围完全一致的复制、修改权利');
      let parameters: Record<string, unknown>;
      try { parameters = JSON.parse(values.parameters_json || '{}') as Record<string, unknown>; } catch { throw new Error('模型参数必须是有效 JSON 对象'); }
      if (!parameters || Array.isArray(parameters) || typeof parameters !== 'object') throw new Error('模型参数必须是 JSON 对象');
      const mustNotChange = values.must_not_change.split('\n').map((item) => item.trim()).filter(Boolean);
      return createImageJob({
        asset_id: assetId, sku_id: values.sku_id, recipe_key: values.recipe_key,
        recipe_version: values.recipe_version, parent_task_id: values.parent_task_id, candidate_round: values.candidate_round,
        recipe: { reference_asset_ids: [assetId], scene_structure: values.scene_structure, prompt: values.prompt, negative_prompt: values.negative_prompt, model: values.model, model_version: values.model_version, parameters, must_not_change: mustNotChange },
        idempotency_key: newImageIdempotencyKey(`product-image-create-${assetId}`),
        processor: values.processor, operation: values.operation, purpose: values.purpose, channel: values.channel, region: values.region,
        width: values.processor === 'openai' ? 1024 : 1200, height: values.processor === 'openai' ? 1024 : 1200, format: 'png',
        ...(values.processor === 'photoroom' ? { max_cost: '0', currency: 'USD' as const } : {}),
        ...(values.processor === 'openai' ? { max_cost: values.max_cost, currency: 'USD' as const } : {}),
      });
    },
    onSuccess: refresh,
  });
  const execute = useMutation({ mutationFn: executeImageJob, onSuccess: refresh });
  const approveAndExecute = useMutation({ mutationFn: async (job: ProductImageJob) => { await approveImageExecution(job); return executeImageJob(job.id); }, onSuccess: () => setPaidExecutionJob(undefined), onSettled: refresh });
  const customRequest: UploadProps['customRequest'] = ({ file, onError, onSuccess }) => {
    upload.mutate(file as File, { onSuccess: (asset) => onSuccess?.(asset), onError: (error) => onError?.(error) });
  };
  const completed = useMemo(() => (jobs.data ?? []).filter((job) => job.status === 'READY' && job.output_blob_id), [jobs.data]);
  const releaseEligible = useMemo(() => completed.filter((job) => job.processor !== 'photoroom' && !job.sandbox && !job.watermarked && !job.non_publishable), [completed]);
  const governanceJob = completed.find((job) => job.id === governanceJobId);
  const recipeSummary = useQuery({
	queryKey: ['product-images', 'recipe-summary', governanceJob?.sku_id, governanceJob?.recipe_key],
	queryFn: () => getRecipeSummary(governanceJob!.recipe_key!, governanceJob!.sku_id!), enabled: Boolean(governanceJob?.recipe_key && governanceJob?.sku_id), retry: false,
  });
  useEffect(() => {
    if (!governanceJobId && completed[0]) setGovernanceJobId(completed[0].id);
  }, [completed, governanceJobId]);
  useEffect(() => {
    if (!governanceJob) return;
    feedbackForm.setFieldsValue({ outcome: governanceJob.sandbox || governanceJob.watermarked || governanceJob.non_publishable ? 'rejected' : 'selected', reason_codes: [], error_regions_json: '[]', review_seconds: 0 });
  }, [feedbackForm, governanceJob]);
  const rights = useMutation({ mutationFn: async () => {
    const values = await rightsForm.validateFields();
    return createRightsGrant({ ...values, asset_sha256: governanceJob!.output_blob_id!, valid_from: new Date().toISOString(), idempotency_key: newImageIdempotencyKey(`image-rights-${governanceJob!.id}`), expected_version: 1 });
  }, onSuccess: () => setRightsRecorded(true) });
  const review = useMutation({ mutationFn: async () => {
    const values = await reviewForm.validateFields();
    return createFiveAxisReview(governanceJob!.id, { ...values, asset_sha256: governanceJob!.output_blob_id!, idempotency_key: newImageIdempotencyKey(`image-review-${governanceJob!.id}`), expected_version: governanceJob!.version ?? 1 });
  }, onSuccess: () => setReviewRecorded(true) });
  const cost = useMutation({ mutationFn: async () => {
    const values = await costForm.validateFields();
    return createCostEntry(governanceJob!.id, { ...values, observed_at: new Date().toISOString(), idempotency_key: newImageIdempotencyKey(`image-cost-${governanceJob!.id}`), expected_version: governanceJob!.version ?? 1 });
  } });
  const feedback = useMutation({ mutationFn: async () => {
    if (!governanceJob?.output_blob_id) throw new Error('请先选择已完成候选');
    const values = await feedbackForm.validateFields();
    let regions: Array<Record<string, unknown>>;
    try { regions = JSON.parse(values.error_regions_json || '[]') as Array<Record<string, unknown>>; } catch { throw new Error('错误区域必须是有效 JSON 数组'); }
    if (!Array.isArray(regions)) throw new Error('错误区域必须是 JSON 数组');
    return createCandidateFeedback(governanceJob.id, {
      ...values, error_regions: regions, asset_sha256: governanceJob.output_blob_id,
      idempotency_key: newImageIdempotencyKey(`image-feedback-${governanceJob.id}`), expected_version: governanceJob.version ?? 1,
    });
  }, onSuccess: async () => { feedbackForm.resetFields(); await recipeSummary.refetch(); } });
  const manualImport = useMutation({ mutationFn: async () => {
    const values = await manualForm.validateFields();
    if (!source || !manualFile) throw new Error('请先上传父原图并选择外部编辑结果文件');
    return createManualImport({ ...values, file: manualFile, parent_asset_id: source.id, parent_asset_sha256: source.sha256, source_observed_at: new Date().toISOString(), idempotency_key: newImageIdempotencyKey(`manual-import-${source.id}`) });
  }, onSuccess: () => { setManualFile(undefined); manualImports.refetch(); } });
  const createSet = useMutation({
    mutationFn: async () => {
      const scope = await setForm.validateFields();
      const ordered = releaseEligible.filter((job) => selected.includes(job.id));
      return createProductImageSet({ ...scope, items: ordered.map((job, index) => ({ task_id: job.id, role: roles[job.id] ?? (index === 0 ? 'main' : 'gallery'), ordinal: index + 1 })) });
    },
    onSuccess: setImageSet,
  });
  const freezeSet = useMutation({ mutationFn: freezeProductImageSet, onSuccess: setImageSet });
  const createRule = useMutation({ mutationFn: async () => {
    const values = await ruleForm.validateFields();
    let rules: Record<string, unknown>;
    try { rules = JSON.parse(values.rules_json) as Record<string, unknown>; } catch { throw new Error('渠道规则必须是有效 JSON'); }
    if (!rules || Array.isArray(rules) || typeof rules !== 'object') throw new Error('渠道规则必须是 JSON 对象');
    return createImageRuleSnapshot({ channel: values.channel, site: values.site, locale: values.locale, category_id: values.category_id, rules, effective_at: new Date().toISOString(), expires_at: values.expires_at ? new Date(values.expires_at).toISOString() : undefined, idempotency_key: newImageIdempotencyKey('image-rule') });
  }, onSuccess: setRuleSnapshot });
  const decideSet = useMutation({ mutationFn: async () => {
    if (!imageSet || imageSet.status !== 'frozen') throw new Error('只能审批已冻结的图片集合');
    const values = await decisionForm.validateFields();
    return decideImageSet(imageSet.id, { ...values, expected_version: imageSet.version, idempotency_key: newImageIdempotencyKey(`image-set-decision-${imageSet.id}`) });
  }, onSuccess: setSetDecision });
  const issueAttestation = useMutation({ mutationFn: async () => {
    if (!imageSet || !ruleSnapshot || setDecision?.decision !== 'approved') throw new Error('签发前必须完成规则快照和图片集合批准');
    const values = await attestationForm.validateFields();
    return issueImageReleaseAttestation({ image_set_id: imageSet.id, rule_snapshot_id: ruleSnapshot.id, ...values, idempotency_key: newImageIdempotencyKey(`image-attestation-${imageSet.id}`) });
  }, onSuccess: (value) => { setAttestation(value); setAttestationLookupId(value.id); } });
  const lookupAttestation = useMutation({ mutationFn: async () => {
    if (!attestationLookupId) throw new Error('请输入发布证明 ID');
    return getImageReleaseAttestation(attestationLookupId);
  }, onSuccess: setAttestation });
  const refreshBudget = async () => { await Promise.all([budgetPolicies.refetch(), budgetReservations.refetch()]); };
  const createBudgetPolicy = useMutation({ mutationFn: async () => {
    const values = await budgetPolicyForm.validateFields();
    return createImageBudgetPolicy({ ...values, period_start: new Date(values.period_start).toISOString(), period_end: new Date(values.period_end).toISOString(), idempotency_key: newImageIdempotencyKey('image-budget-policy') });
  }, onSuccess: refreshBudget });
  const cancelBudget = useMutation({ mutationFn: async () => {
    if (!budgetReservationId) throw new Error('请选择预算预占');
    const { reason } = await budgetCancelForm.validateFields();
    return cancelImageBudgetReservation(budgetReservationId, reason);
  }, onSuccess: refreshBudget });
  const reconcileBudget = useMutation({ mutationFn: async () => {
    if (!budgetReservationId || !billEvidenceConfirmed) throw new Error('必须选择预占并确认已有可信外部账单证据');
    const values = await budgetChargeForm.validateFields();
    return reconcileImageBudgetCharge(budgetReservationId, { ...values, ...(chargedNoOutputConfirmed ? { resolution: 'charged_no_output' as const } : {}), observed_at: new Date(values.observed_at).toISOString(), idempotency_key: newImageIdempotencyKey(`image-budget-charge-${budgetReservationId}`) });
  }, onSuccess: async () => { setBillEvidenceConfirmed(false); setChargedNoOutputConfirmed(false); await Promise.all([refreshBudget(), refresh()]); } });
  const reconcileNoCharge = useMutation({ mutationFn: async () => {
    if (!budgetReservationId || !noChargeEvidenceConfirmed) throw new Error('必须选择待对账预占并确认已核对可信未扣费证据');
    const values = await budgetNoChargeForm.validateFields();
    return reconcileImageBudgetNoCharge(budgetReservationId, { ...values, observed_at: new Date(values.observed_at).toISOString(), idempotency_key: newImageIdempotencyKey(`image-budget-no-charge-${budgetReservationId}`) });
  }, onSuccess: async () => { setNoChargeEvidenceConfirmed(false); budgetNoChargeForm.resetFields(); await Promise.all([refreshBudget(), refresh()]); } });
  const activeBudget = useMemo(() => {
    return (budgetPolicies.data ?? []).filter((policy) => Date.parse(policy.period_start) <= pageOpenedAt && pageOpenedAt < Date.parse(policy.period_end));
  }, [budgetPolicies.data, pageOpenedAt]);
  const budgetSummary = (policyId: number, total: string) => {
    const totalUnits = moneyUnits(total) ?? BigInt(0);
    const reservedUnits = (budgetReservations.data ?? []).filter((item) => item.policy_id === policyId && (item.state === 'reserved' || item.state === 'claimed')).reduce((sum, item) => sum + (moneyUnits(item.reserved_amount) ?? BigInt(0)), BigInt(0));
    return { reserved: formatMoney(reservedUnits), remaining: formatMoney(totalUnits - reservedUnits) };
  };
  const selectedReservation = (budgetReservations.data ?? []).find((item) => item.id === budgetReservationId);
  const selectedReservationTask = (jobs.data ?? []).find((job) => job.id === selectedReservation?.task_id);
  const reconcileJobs = (jobs.data ?? []).filter((job) => job.status === 'reconcile_required' || job.status === 'RECONCILE_REQUIRED');
  const mutationError = upload.error ?? inputRights.error ?? create.error ?? execute.error ?? approveAndExecute.error ?? manualImport.error ?? rights.error ?? review.error ?? cost.error ?? feedback.error ?? createSet.error ?? freezeSet.error ?? createRule.error ?? decideSet.error ?? issueAttestation.error ?? lookupAttestation.error ?? createBudgetPolicy.error ?? cancelBudget.error ?? reconcileBudget.error ?? reconcileNoCharge.error;

  const capabilityLabel = (item: ImageProcessorCapability) => {
    if (item.code === 'photoroom') return photoroomCanaryAvailable
      ? `${item.name} · sandbox-only · 剩余 ${item.quota_remaining ?? 0} 次 canary`
      : `${item.name} · sandbox canary 不可用：${item.reason || ((item.quota_remaining ?? 0) <= 0 ? '一次 canary 配额不可用' : '后端安全条件不完整')}`;
    if (item.code === 'openai') return openAIAvailable
      ? `${item.name} · 可用 · 真实付费，执行前需 Owner 批准与预算预占`
      : `${item.name} · 付费执行未启用：${item.reason || '后端生产安全条件不完整'}`;
    return `${item.name} · ${item.configured ? '可用' : '未配置'}`;
  };

  return <div style={{ padding: '16px 20px', minHeight: '100%' }}>
    <PageHeader title="商品图片工作室" subtitle="上传真实商品原图，生成候选图片；完成并不代表图片已获准发布" />
    <Space orientation="vertical" size={16} style={{ width: '100%' }}>
      <Alert type="info" showIcon title={deterministicAvailable ? '当前可用闭环：确定性尺寸处理' : '图片处理能力尚未可用'} description="能力状态来自凌镜后端。外部 Provider 还必须满足配置、预算和 Owner 批准门禁，不会用模拟图片冒充结果。" />
      {(jobs.error || mutationError) && <Alert type="error" showIcon title="图片工作台操作失败" description={errorText(jobs.error ?? mutationError)} />}
      {reconcileJobs.length > 0 && <Alert type="warning" showIcon title={`有 ${reconcileJobs.length} 个真实付费任务等待对账，禁止重试`} description="外部结果不确定时不要重复批准或创建同一付费意图。先核对 Provider 结果和账单：已扣费就录入真实账单；已确认未扣费时，选择对应 claimed 预占并提交未扣费证据；仍不确定就保持待对账。" />}
      <Card title="处理方案">
        {capabilities.isLoading ? <Space><Spin size="small" /><Typography.Text type="secondary">正在读取真实能力状态…</Typography.Text></Space>
          : capabilities.error ? <Alert type="error" showIcon title="能力状态读取失败" description={`${errorText(capabilities.error)}；为避免误执行，所有处理入口已关闭。`} action={<Button size="small" onClick={() => capabilities.refetch()}>重试</Button>} />
            : <Space wrap>{(capabilities.data ?? []).map((item) => <Tag key={item.code} color={(item.code === 'openai' ? openAIAvailable : item.configured) ? 'success' : 'default'}>
              {capabilityLabel(item)}
            </Tag>)}</Space>}
        {!capabilities.isLoading && !capabilities.error && !deterministicAvailable && <Alert style={{ marginTop: 12 }} type="warning" showIcon title="确定性处理未配置" description={deterministic?.reason || '请先配置并启动 Image Service；在后端确认可用前，不能创建或执行图片任务。'} />}
      </Card>
      <Card title="付费图片预算与预占">
        <Space orientation="vertical" size={16} style={{ width: '100%' }}>
          <Alert type="info" showIcon title="付费批准前先看可用额度" description="estimated 是事前估算；reserved 是批准后锁定、尚未等同真实扣费的额度；spent 只能来自外部账单对账；over_budget 表示账单金额超过预占或预算。页面不会把预占伪装成已花费。" />
          <Form form={budgetPolicyForm} layout="inline" initialValues={{ currency: 'USD', total_amount: '100' }}>
            <Form.Item name="currency" label="币种" rules={[{ required: true }]}><Select style={{ width: 90 }} options={['USD', 'EUR', 'CNY', 'GBP', 'JPY'].map((value) => ({ value, label: value }))} /></Form.Item>
            <Form.Item name="period_start" label="周期开始" rules={[{ required: true }]}><Input type="datetime-local" /></Form.Item>
            <Form.Item name="period_end" label="周期结束" rules={[{ required: true }]}><Input type="datetime-local" /></Form.Item>
            <Form.Item name="total_amount" label="总额" rules={[{ required: true }, { pattern: /^(0|[1-9][0-9]{0,9})(\.[0-9]{1,4})?$/ }]}><Input style={{ width: 110 }} /></Form.Item>
            <Button type="primary" loading={createBudgetPolicy.isPending} onClick={() => createBudgetPolicy.mutate()}>创建不可变预算政策</Button>
          </Form>
          {activeBudget.length === 0 ? <Alert type="warning" showIcon title="当前没有生效中的预算政策" description="付费批准应保持关闭；先为对应币种和周期创建预算政策。" /> : <List size="small" header="当前生效政策与付费批准前剩余额度" dataSource={activeBudget} renderItem={(policy) => {
            const summary = budgetSummary(policy.id, policy.total_amount);
            return <List.Item><List.Item.Meta title={`政策 #${policy.id} · ${policy.currency} ${policy.total_amount}`} description={`reserved（已预占）${summary.reserved} · 预占口径剩余 ${summary.remaining} ${policy.currency} · spent（已花费）须以外部账单对账记录为准，本列表不推断`} /></List.Item>;
          }} />}
          <List<ImageBudgetReservation> size="small" header="预算预占记录" locale={{ emptyText: '暂无预占；预占只会由受控付费批准流程产生' }} dataSource={budgetReservations.data ?? []} renderItem={(item) => <List.Item><List.Item.Meta title={`预占 #${item.id} · ${item.reserved_amount} ${item.currency}`} description={`任务 ${item.task_id} · ${item.provider} · 状态 ${item.state}${item.state === 'claimed' ? '（已claim，不可取消）' : ''}`} /></List.Item>} />
          <Row gutter={[16, 16]}>
            <Col xs={24} xl={8}><Form.Item label="选择预占"><Select<number> aria-label="选择预占" value={budgetReservationId} onChange={(value) => { setBudgetReservationId(value); setBillEvidenceConfirmed(false); setChargedNoOutputConfirmed(false); setNoChargeEvidenceConfirmed(false); }} style={{ width: '100%' }} options={(budgetReservations.data ?? []).map((item) => ({ value: item.id, label: `#${item.id} · ${item.state} · ${item.reserved_amount} ${item.currency}` }))} /></Form.Item></Col>
            <Col xs={24} xl={16}><Form name="product-image-budget-cancel" form={budgetCancelForm} layout="inline"><Form.Item name="reason" label="取消原因" rules={[{ required: true }]}><Input style={{ width: 260 }} /></Form.Item><Button danger disabled={!selectedReservation || selectedReservation.state !== 'reserved'} loading={cancelBudget.isPending} onClick={() => cancelBudget.mutate()}>取消未claim预占</Button></Form></Col>
          </Row>
          <Card size="small" title="外部账单对账（受控录入）">
            <Alert type="warning" showIcon title="没有可信账单证据就不要操作" description="对账不是生成账单。只有从 Provider/支付方取得账单并计算其文件 SHA-256 后才可提交；系统会追加记录，不覆盖历史。" />
            <Form name="product-image-budget-charge" form={budgetChargeForm} layout="inline" initialValues={{ currency: selectedReservation?.currency ?? 'USD' }} style={{ marginTop: 12 }}>
              <Form.Item name="amount" label="账单累计金额" rules={[{ required: true }, { pattern: /^(0|[1-9][0-9]{0,9})(\.[0-9]{1,4})?$/ }]}><Input style={{ width: 120 }} /></Form.Item>
              <Form.Item name="currency" label="币种" rules={[{ required: true }]}><Input style={{ width: 80 }} /></Form.Item>
              <Form.Item name="evidence_sha256" label="账单证据 SHA-256" rules={[{ required: true }, { pattern: /^[0-9a-f]{64}$/ }]}><Input style={{ width: 300 }} /></Form.Item>
              <Form.Item name="observed_at" label="账单观察时间" rules={[{ required: true }]}><Input type="datetime-local" /></Form.Item>
            </Form>
            <Checkbox checked={billEvidenceConfirmed} onChange={(event) => setBillEvidenceConfirmed(event.target.checked)}>我已取得并核对真实外部账单，SHA-256 指向该证据</Checkbox>
            <Checkbox style={{ marginLeft: 12 }} checked={chargedNoOutputConfirmed} disabled={!selectedReservation || selectedReservation.state !== 'claimed' || !unresolvedPaidOutcome(selectedReservationTask)} onChange={(event) => setChargedNoOutputConfirmed(event.target.checked)}>该证据同时确认已扣费且没有可恢复输出，结束待对账状态</Checkbox>
            <Button style={{ marginLeft: 12 }} disabled={!selectedReservation || !billEvidenceConfirmed} loading={reconcileBudget.isPending} onClick={() => reconcileBudget.mutate()}>追加账单对账</Button>
          </Card>
          <Card size="small" title="确认未扣费（只用于待对账任务）">
            <Alert type="warning" showIcon title="系统会先原子停止排队任务，再释放预占" description="仅当 Provider 控制台或账单明确证明该任务没有请求、没有费用时使用。若任务已运行、已有输出或无法证明不会继续执行，系统会拒绝释放。" />
            <Form name="product-image-budget-no-charge" form={budgetNoChargeForm} layout="inline" style={{ marginTop: 12 }}>
              <Form.Item name="evidence_sha256" label="未扣费证据 SHA-256" rules={[{ required: true }, { pattern: /^[0-9a-f]{64}$/ }]}><Input style={{ width: 300 }} /></Form.Item>
              <Form.Item name="observed_at" label="证据观察时间" rules={[{ required: true }]}><Input type="datetime-local" /></Form.Item>
              <Form.Item name="reason" label="核对说明" rules={[{ required: true }]}><Input style={{ width: 280 }} /></Form.Item>
            </Form>
            <Checkbox checked={noChargeEvidenceConfirmed} onChange={(event) => setNoChargeEvidenceConfirmed(event.target.checked)}>我已核对可信 Provider 证据，确认该精确任务未扣费</Checkbox>
            <Button style={{ marginLeft: 12 }} danger disabled={!selectedReservation || selectedReservation.state !== 'claimed' || !unresolvedPaidOutcome(selectedReservationTask) || !noChargeEvidenceConfirmed} loading={reconcileNoCharge.isPending} onClick={() => reconcileNoCharge.mutate()}>确认未扣费并安全释放预占</Button>
          </Card>
        </Space>
      </Card>
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={9}><Card title="1. 上传真实原图">
          <Upload accept="image/png,image/jpeg" maxCount={1} showUploadList={false} customRequest={customRequest}>
            <Button icon={<CloudUploadOutlined />} loading={upload.isPending}>选择并上传图片</Button>
          </Upload>
          {source && <Space orientation="vertical" style={{ marginTop: 16 }}>
            <Typography.Text strong>{source.filename}</Typography.Text>
            <Typography.Text type="secondary">SHA-256：{source.sha256}</Typography.Text>
          </Space>}
        </Card></Col>
        <Col xs={24} lg={15}><Card title="2. 核对原图权利并创建处理任务">
          <Alert type="warning" showIcon title="先授权，后处理" description={isPhotoroomScope ? 'Photoroom 会把原图发送到美国区域的第三方 AI。必须逐项确认复制、修改、第三方 AI 和跨境处理权利；结果带水印、不可发布，且只允许一次 sandbox canary。' : isOpenAIScope ? 'OpenAI 图片编辑会把原图与配方提示词发送到美国区域并产生真实费用。必须确认第三方 AI 与跨境处理权，并由 Owner 对准确任务版本设置单次最高费用后再执行。' : '复制权和修改权必须绑定当前原图 SHA-256、处理器、用途、渠道和处理地区。这里的授权只允许这次确定性处理，不代表允许跨境、商业发布或平台转授权。'} />
          <Form form={taskForm} layout="vertical" initialValues={{ processor: 'deterministic', operation: 'DETERMINISTIC_RESIZE', region: 'local', recipe_key: defaultRecipeKey, recipe_version: 1, candidate_round: 1, scene_structure: '商品主体保持不变，置于简洁、真实且不喧宾夺主的使用场景', model: 'deterministic', model_version: '1', parameters_json: '{}', must_not_change: '商品结构\n颜色与材质\nLogo与文字\n数量与配件' }} onValuesChange={(changed) => {
            setInputRightsFingerprint(undefined);
            if (changed.processor === 'photoroom') taskForm.setFieldsValue({ operation: 'PHOTOROOM_REMOVE_BACKGROUND_SANDBOX', region: 'us', model: 'photoroom', model_version: 'sandbox' });
            if (changed.processor === 'openai') taskForm.setFieldsValue({ operation: 'OPENAI_IMAGE_EDIT', region: 'us', model: 'gpt-image-2', model_version: 'current', max_cost: '0.20' });
            if (changed.processor === 'deterministic') taskForm.setFieldsValue({ operation: 'DETERMINISTIC_RESIZE', region: 'local', model: 'deterministic', model_version: '1' });
          }} style={{ marginTop: 12 }}>
            <Row gutter={8}>
              <Col xs={12} md={5}><Form.Item name="processor" label="处理器" rules={[{ required: true }]}><Select options={[{ value: 'deterministic', label: 'deterministic（本地）' }, { value: 'photoroom', label: 'Photoroom（sandbox-only）', disabled: !photoroomCanaryAvailable }, { value: 'openai', label: 'OpenAI Images（真实付费）', disabled: !openAIAvailable }]} /></Form.Item></Col>
              <Col xs={12} md={5}><Form.Item name="operation" label="显式操作" rules={[{ required: true }]}><Select options={isPhotoroomScope ? [...photoroomOperations] : isOpenAIScope ? [{ value: 'OPENAI_IMAGE_EDIT', label: 'OpenAI 场景编辑' }] : [{ value: 'DETERMINISTIC_RESIZE', label: '确定性尺寸处理' }]} /></Form.Item></Col>
              <Col xs={12} md={6}><Form.Item name="purpose" label="用途" rules={[{ required: true, message: '请明确填写用途' }]}><Input placeholder="例如 listing_main" /></Form.Item></Col>
              <Col xs={12} md={4}><Form.Item name="channel" label="销售渠道" rules={[{ required: true, message: '请明确填写渠道' }]}><Input placeholder="例如 ozon" /></Form.Item></Col>
              <Col xs={12} md={4}><Form.Item name="region" label="处理地区" rules={[{ required: true }]}><Select disabled options={isExternalScope ? [{ value: 'us', label: 'US' }] : [{ value: 'local', label: 'local（凌镜本地）' }]} /></Form.Item></Col>
            </Row>
            <Row gutter={8}>
              <Col xs={24} md={8}><Form.Item name="sku_id" label="准确 SKU" rules={[{ required: true, message: '请选择真实 SKU' }]}><Select showSearch optionFilterProp="label" loading={skuOptions.isLoading} options={(skuOptions.data ?? []).map((sku) => ({ value: sku.id, label: `${sku.code || `SKU #${sku.id}`} · ${sku.spec_desc || '无规格说明'}` }))} /></Form.Item></Col>
              <Col xs={24} md={8}><Form.Item name="recipe_key" label="配方编号" rules={[{ required: true }]}><Input disabled={Boolean(parentTaskID)} /></Form.Item></Col>
              <Col xs={12} md={4}><Form.Item name="recipe_version" label="配方版本" rules={[{ required: true }]}><InputNumber min={1} precision={0} disabled /></Form.Item></Col>
              <Col xs={12} md={4}><Form.Item name="candidate_round" label="候选轮次" rules={[{ required: true }]}><InputNumber min={1} precision={0} disabled /></Form.Item></Col>
            </Row>
            <Form.Item name="scene_structure" label="场景结构" rules={[{ required: true, message: '请说明商品放在什么场景和构图中' }]}><Input.TextArea rows={2} /></Form.Item>
            <Row gutter={8}>
              <Col xs={24} md={12}><Form.Item name="prompt" label="Provider 提示词（确定性处理可留空）"><Input.TextArea rows={3} /></Form.Item></Col>
              <Col xs={24} md={12}><Form.Item name="negative_prompt" label="负面约束"><Input.TextArea rows={3} /></Form.Item></Col>
              <Col xs={12} md={6}><Form.Item name="model" label="模型" rules={[{ required: true }]}><Input /></Form.Item></Col>
              <Col xs={12} md={6}><Form.Item name="model_version" label="模型版本" rules={[{ required: true }]}><Input /></Form.Item></Col>
              <Col xs={24} md={12}><Form.Item name="parameters_json" label="模型参数 JSON" rules={[{ required: true }]}><Input /></Form.Item></Col>
            </Row>
            <Form.Item name="must_not_change" label="商品绝对不能改变的事实（每行一项）" rules={[{ required: true }]}><Input.TextArea rows={4} /></Form.Item>
            {isOpenAIScope && <Form.Item name="max_cost" label="本次预算预占上限（USD，非供应商硬封顶）" rules={[{ required: true }, { validator: (_, value) => typeof value === 'string' && moneyUnits(value) && moneyUnits(value)! > BigInt(0) ? Promise.resolve() : Promise.reject(new Error('请输入大于 0、最多四位小数的金额')) }]}><Input aria-label="OpenAI 本次预算预占上限" /></Form.Item>}
            <Form.Item name="parent_task_id" hidden><InputNumber /></Form.Item>
            {parentTaskID && <Alert style={{ marginBottom: 12 }} type="info" showIcon title={`正在基于任务 #${parentTaskID} 创建返工版本`} description="配方版本和候选轮次已递增；修改场景或提示词后创建新任务，旧记录不会被覆盖。" />}
          </Form>
          {isPhotoroomScope && <Alert style={{ marginBottom: 12 }} type={photoroomCanaryAvailable ? 'info' : 'warning'} showIcon title="Photoroom sandbox canary" description={photoroomCanaryAvailable ? `provider_environment=sandbox · region=US · max_cost=0 USD · 带水印 · 不可发布 · 剩余 ${photoroom?.quota_remaining ?? 0} 次` : `入口已关闭：${photoroom?.reason || '后端未明确报告 available、sandbox_only 或剩余配额'}`} />}
          <Form form={inputRightsForm} layout="vertical" initialValues={{ can_copy: false, can_modify: false, can_third_party_ai: false, can_cross_border: false, owner_verified: false }}>
            <Row gutter={8}>
              <Col xs={24} md={8}><Form.Item name="grantor" label="授权人/权利人" rules={[{ required: true }]}><Input /></Form.Item></Col>
              <Col xs={24} md={8}><Form.Item name="rights_chain" label="权利来源说明" rules={[{ required: true }]}><Input /></Form.Item></Col>
              <Col xs={24} md={8}><Form.Item name="evidence_sha256" label="权利证据 SHA-256" rules={[{ required: true }, { pattern: /^[0-9a-f]{64}$/, message: '必须是64位小写SHA-256' }]}><Input aria-label="原图权利证据 SHA-256" /></Form.Item></Col>
            </Row>
            <Space wrap>
              <Form.Item name="can_copy" valuePropName="checked" rules={[{ validator: (_, value) => value ? Promise.resolve() : Promise.reject(new Error('必须确认复制权')) }]}><Checkbox>确认拥有复制权 can_copy</Checkbox></Form.Item>
              <Form.Item name="can_modify" valuePropName="checked" rules={[{ validator: (_, value) => value ? Promise.resolve() : Promise.reject(new Error('必须确认修改权')) }]}><Checkbox>确认拥有修改权 can_modify</Checkbox></Form.Item>
              {isExternalScope && <Form.Item name="can_third_party_ai" valuePropName="checked" rules={[{ validator: (_, value) => value ? Promise.resolve() : Promise.reject(new Error('必须确认第三方 AI 处理权')) }]}><Checkbox>允许第三方 AI can_third_party_ai</Checkbox></Form.Item>}
              {isExternalScope && <Form.Item name="can_cross_border" valuePropName="checked" rules={[{ validator: (_, value) => value ? Promise.resolve() : Promise.reject(new Error('必须确认跨境处理权')) }]}><Checkbox>允许跨境至 US can_cross_border</Checkbox></Form.Item>}
              <Form.Item name="owner_verified" valuePropName="checked" rules={[{ validator: (_, value) => value ? Promise.resolve() : Promise.reject(new Error('必须由 Owner 核验')) }]}><Checkbox>Owner 已核对精确证据</Checkbox></Form.Item>
            </Space>
          </Form>
          <Space wrap>
            <Button disabled={!source || !selectedProcessorAvailable} loading={inputRights.isPending} onClick={() => inputRights.mutate()}>保存原图处理权利</Button>
            <Button type="primary" disabled={(!source && !parentTaskID) || !selectedProcessorAvailable || (!parentTaskID && !inputRightsFingerprint)} loading={create.isPending} onClick={() => create.mutate()}>{parentTaskID ? '创建返工候选' : isPhotoroomScope ? '创建一次 Photoroom sandbox canary' : isOpenAIScope ? '创建 OpenAI 场景候选' : '创建确定性图片任务'}</Button>
            {inputRightsFingerprint && <Tag color="success">当前原图与处理范围的权利已记录</Tag>}
          </Space>
          <Typography.Paragraph type="secondary" style={{ marginTop: 8 }}>确定性与 Photoroom 输出为 1200 × 1200 PNG；OpenAI 场景候选为 1024 × 1024 PNG。处理器、模型、操作、用途、渠道和地区均写入冻结配方；页面不接收 Provider 密钥。</Typography.Paragraph>
        </Card></Col>
      </Row>
      <Card title="外部编辑结果导入（不调用外部工具）">
        <Alert type="warning" showIcon title="导入不等于获准发布" description="结果会经 Image Service 清洗并绑定当前原图 SHA-256；真实性默认 unknown，仍须完成图片权利和五类审核。" />
        <Form form={manualForm} layout="vertical" initialValues={{ import_kind: 'manual_import', tool: 'Photoshop', operation: '', fee_amount: '0', fee_currency: 'USD', model: 'unknown', model_version: 'unknown', channel_restriction: '*' }} style={{ marginTop: 12 }}>
          <Row gutter={12}>
            <Col xs={24} md={8}><Form.Item name="import_kind" label="导入类型" rules={[{ required: true }]}><Select options={[{ value: 'manual_import', label: '通用外部编辑' }, { value: 'channel_native_import', label: '渠道内置编辑' }]} /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item name="tool" label="工具" rules={[{ required: true }]}><Input /></Form.Item></Col>
            <Col xs={24} md={8}><Form.Item name="operation" label="实际操作" rules={[{ required: true }]}><Input placeholder="例如：去灰尘、校正曝光" /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item name="fee_amount" label="实际费用" rules={[{ required: true }, { pattern: /^(0|[1-9][0-9]{0,9})(\.[0-9]{1,4})?$/ }]}><Input /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item name="fee_currency" label="币种"><Select options={['USD', 'EUR', 'CNY', 'GBP', 'JPY'].map((value) => ({ value, label: value }))} /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item name="model" label="模型（未知填 unknown）" rules={[{ required: true }]}><Input /></Form.Item></Col>
            <Col xs={12} md={6}><Form.Item name="model_version" label="版本（未知填 unknown）" rules={[{ required: true }]}><Input /></Form.Item></Col>
            <Col xs={12}><Form.Item name="original_channel" label="原渠道（渠道内置时必填）"><Input placeholder="例如 shopify" /></Form.Item></Col>
            <Col xs={12}><Form.Item name="channel_restriction" label="仅可使用渠道" rules={[{ required: true }]}><Input placeholder="通用填 *；渠道内置必须与原渠道相同" /></Form.Item></Col>
          </Row>
          <Space wrap>
            <Upload accept="image/png,image/jpeg" maxCount={1} beforeUpload={(file) => { setManualFile(file); return false; }} onRemove={() => { setManualFile(undefined); }}><Button>选择编辑结果文件</Button></Upload>
            <Button type="primary" disabled={!source || !manualFile} loading={manualImport.isPending} onClick={() => manualImport.mutate()}>保存外部编辑结果与溯源</Button>
          </Space>
        </Form>
        <List size="small" style={{ marginTop: 12 }} dataSource={manualImports.data ?? []} locale={{ emptyText: '暂无外部编辑导入' }} renderItem={(item) => <List.Item><List.Item.Meta title={`${item.tool} · ${item.operation}`} description={`结果 ${item.asset_sha256} · 父图 ${item.parent_asset_sha256} · ${item.fee_amount} ${item.fee_currency} · 渠道限制 ${item.channel_restriction} · truth=${item.truth}`} /></List.Item>} />
      </Card>
      <Card title="任务" extra={<Button icon={<ReloadOutlined />} loading={jobs.isFetching} onClick={() => jobs.refetch()}>刷新</Button>}>
        {jobs.isLoading ? <Spin /> : !(jobs.data?.length) ? <Empty description="还没有图片处理任务" /> : <List dataSource={jobs.data} renderItem={(job: ProductImageJob) => {
          const status = statusPresentation[job.status] ?? { label: job.status, color: 'default' };
          const executable = job.processor === 'photoroom' ? photoroomCanaryAvailable : job.processor === 'openai' ? openAIAvailable : deterministicAvailable;
          return <List.Item actions={['pending', 'created', 'QUEUED'].includes(job.status) ? [job.processor === 'openai' ? <Button key="approve-execute" disabled={!executable} onClick={() => setPaidExecutionJob(job)}>查看配方并批准付费执行（{job.max_cost} USD）</Button> : <Button key="execute" disabled={!executable} icon={<PlayCircleOutlined />} loading={execute.isPending && execute.variables === job.id} onClick={() => execute.mutate(job.id)}>执行</Button>] : []}>
            <List.Item.Meta title={<Space><Typography.Text code>{job.id}</Typography.Text><Tag color={status.color}>{status.label}</Tag></Space>} description={<Space orientation="vertical" size={0} style={{ width: '100%' }}><span>{job.operation} · {job.width} × {job.height} {job.format}</span><span>{job.processor} · 用途 {job.purpose} · 渠道 {job.channel} · 地区 {job.region}</span>{job.error_code && <Typography.Text type="danger">错误代码：{job.error_code}</Typography.Text>}{job.recipe_manifest && <details><summary>查看冻结作图配方</summary><Typography.Paragraph style={{ marginBottom: 0 }}>配方 {job.recipe_key} v{job.recipe_version} · recipe SHA-256：<Typography.Text code>{job.recipe_hash}</Typography.Text><br />manifest SHA-256：<Typography.Text code>{job.manifest_hash}</Typography.Text><br />参考图：{job.recipe_manifest.reference_asset_ids?.join(', ') || '缺失'}<br />场景：{job.recipe_manifest.scene_structure}<br />提示词：{job.recipe_manifest.prompt || '缺失'}<br />负面约束：{job.recipe_manifest.negative_prompt || '无'}<br />模型：{job.recipe_manifest.model} / {job.recipe_manifest.model_version}<br />参数：{JSON.stringify(job.recipe_manifest.parameters)}<br />绝不能改变：{job.recipe_manifest.must_not_change.join('；')}</Typography.Paragraph></details>}</Space>} />
          </List.Item>;
        }} />}
      </Card>
      <Card title="3. 权利、五类审核与成本">
        {!completed.length ? <Empty description="处理成功并取得受控输出后才能登记权利与审核" /> : <Space orientation="vertical" size={16} style={{ width: '100%' }}>
          <Alert type="warning" showIcon title="这些记录是发布前门禁，不是自动判断" description="必须绑定精确输出 SHA-256。只有 Owner 明确核对权利证据和五类检查后，图片集合才允许冻结。" />
          <Select<number> value={governanceJobId} style={{ width: '100%' }} onChange={(id) => { setGovernanceJobId(id); setRightsRecorded(false); setReviewRecorded(false); }} options={completed.map((job) => ({ value: job.id, label: `任务 ${job.id} · ${job.output_blob_id}` }))} />
          {governanceJob && <>
            <Typography.Text code>输出 SHA-256：{governanceJob.output_blob_id}</Typography.Text>
            <Row gutter={[16, 16]}>
              <Col xs={24} xl={12}><Card size="small" title="A. 精确图片权利授权">
                <Form form={rightsForm} layout="vertical" initialValues={{ purpose: 'listing_main', jurisdiction: '*', channel: 'ozon', provider: 'deterministic', region: 'local', can_copy: false, can_modify: false, can_third_party_ai: false, can_cross_border: false, can_commercial_publish: false, can_platform_sublicense: false, trademark_cleared: false, likeness_cleared: false, owner_verified: false }}>
                  <Row gutter={8}><Col span={12}><Form.Item name="purpose" label="用途" rules={[{ required: true }]}><Input /></Form.Item></Col><Col span={12}><Form.Item name="channel" label="渠道" rules={[{ required: true }]}><Input /></Form.Item></Col></Row>
                  <Row gutter={8}><Col span={12}><Form.Item name="jurisdiction" label="法域" rules={[{ required: true }]}><Input /></Form.Item></Col><Col span={12}><Form.Item name="region" label="处理地区" rules={[{ required: true }]}><Input /></Form.Item></Col></Row>
                  <Row gutter={8}><Col span={12}><Form.Item name="provider" label="处理方" rules={[{ required: true }]}><Input /></Form.Item></Col><Col span={12}><Form.Item name="grantor" label="授权人" rules={[{ required: true }]}><Input /></Form.Item></Col></Row>
                  <Form.Item name="rights_chain" label="权利链说明" rules={[{ required: true }]}><Input.TextArea rows={2} /></Form.Item>
                  <Form.Item name="evidence_sha256" label="权利证据 SHA-256" rules={[{ required: true }, { pattern: /^[0-9a-f]{64}$/, message: '必须是64位小写SHA-256' }]}><Input /></Form.Item>
                  <Space wrap>{['can_copy', 'can_modify', 'can_third_party_ai', 'can_cross_border', 'can_commercial_publish', 'can_platform_sublicense', 'trademark_cleared', 'likeness_cleared'].map((name) => <Form.Item key={name} name={name} valuePropName="checked" noStyle><Checkbox>{name}</Checkbox></Form.Item>)}</Space>
                  <Form.Item name="owner_verified" valuePropName="checked" rules={[{ validator: (_, value) => value ? Promise.resolve() : Promise.reject(new Error('必须由Owner明确核验')) }]}><Checkbox>我已核对上述授权和精确证据</Checkbox></Form.Item>
                  <Button type="primary" disabled={!governanceJob} loading={rights.isPending} onClick={() => rights.mutate()}>保存权利授权</Button>
                  {rightsRecorded && <Tag color="success">权利记录已保存</Tag>}
                </Form>
              </Card></Col>
              <Col xs={24} xl={12}><Card size="small" title="B. 五类逐项审核">
                <Form form={reviewForm} layout="vertical" initialValues={{ purpose: 'listing_main', channel: 'ozon', evidence_truth: 'quoted', product_authenticity: 'unknown', rights: 'unknown', channel_rules: 'unknown', claims_scene: 'unknown', technical_visual: 'unknown' }}>
                  <Row gutter={8}><Col span={12}><Form.Item name="purpose" label="用途" rules={[{ required: true }]}><Input /></Form.Item></Col><Col span={12}><Form.Item name="channel" label="渠道" rules={[{ required: true }]}><Input /></Form.Item></Col></Row>
                  {([['product_authenticity', '商品真实性'], ['rights', '图片权利'], ['channel_rules', '渠道规则'], ['claims_scene', '声明与场景'], ['technical_visual', '技术视觉']] as const).map(([name, label]) => <Form.Item key={name} name={name} label={label} rules={[{ required: true }]}><Select<ImageGateStatus> options={['passed', 'blocked', 'unknown'].map((value) => ({ value, label: value }))} /></Form.Item>)}
                  <Form.Item name="evidence_sha256" label="审核证据 SHA-256" rules={[{ required: true }, { pattern: /^[0-9a-f]{64}$/, message: '必须是64位小写SHA-256' }]}><Input /></Form.Item>
                  <Form.Item name="evidence_truth" label="证据真实性"><Select options={['quoted', 'inferred', 'unknown'].map((value) => ({ value, label: value }))} /></Form.Item>
                  <Form.Item name="notes" label="说明"><Input.TextArea rows={2} /></Form.Item>
                  <Button type="primary" disabled={!governanceJob} loading={review.isPending} onClick={() => review.mutate()}>保存五类审核</Button>
                  {reviewRecorded && <Tag color="success">五类审核已保存</Tag>}
                </Form>
              </Card></Col>
            </Row>
            <Card size="small" title="C. 成本记录（确定性处理可选，外部付费处理必填）">
              <Form form={costForm} layout="inline" initialValues={{ kind: 'estimated', category: 'provider', provider: governanceJob.processor ?? 'deterministic', amount: '0', currency: 'USD', exchange_rate: '1', exchange_rate_source: 'Owner entered', billing_status: 'estimated' }}>
                <Form.Item name="kind" label="类型"><Select style={{ width: 110 }} options={['estimated', 'actual'].map((value) => ({ value, label: value }))} /></Form.Item>
                <Form.Item name="category" label="类别" rules={[{ required: true }]}><Input style={{ width: 120 }} /></Form.Item>
                <Form.Item name="provider" label="处理方" rules={[{ required: true }]}><Input style={{ width: 120 }} /></Form.Item>
                <Form.Item name="amount" label="金额" rules={[{ required: true }, { pattern: /^(0|[1-9][0-9]{0,9})(\.[0-9]{1,4})?$/ }]}><Input style={{ width: 100 }} /></Form.Item>
                <Form.Item name="currency" label="币种"><Select style={{ width: 90 }} options={['USD', 'EUR', 'CNY', 'GBP', 'JPY'].map((value) => ({ value, label: value }))} /></Form.Item>
                <Form.Item name="exchange_rate" label="汇率" rules={[{ required: true }]}><Input style={{ width: 90 }} /></Form.Item>
                <Form.Item name="exchange_rate_source" label="汇率来源" rules={[{ required: true }]}><Input style={{ width: 140 }} /></Form.Item>
                <Form.Item name="billing_status" label="账单状态"><Select style={{ width: 120 }} options={['estimated', 'pending', 'invoiced', 'paid', 'reconciled', 'unknown'].map((value) => ({ value, label: value }))} /></Form.Item>
                <Button loading={cost.isPending} onClick={() => cost.mutate()}>保存成本</Button>
              </Form>
            </Card>
            <Card size="small" title="D. 候选选择、拒绝与返工">
              <Form form={feedbackForm} layout="vertical" initialValues={{ outcome: 'selected', reason_codes: [], error_regions_json: '[]', review_seconds: 0 }}>
                <Row gutter={8}>
                  <Col xs={24} md={8}><Form.Item name="outcome" label="Owner 结论" rules={[{ required: true }]}><Select options={[{ value: 'selected', label: '选为最终候选', disabled: Boolean(governanceJob.sandbox || governanceJob.watermarked || governanceJob.non_publishable) }, { value: 'rejected', label: '拒绝' }, { value: 'rework_requested', label: '要求返工' }]} /></Form.Item></Col>
                  <Col xs={24} md={8}><Form.Item name="reason_codes" label="原因" rules={feedbackOutcome === 'selected' ? [] : [{ required: true, message: '拒绝或返工必须记录原因' }]}><Select<CandidateReasonCode[]> mode="multiple" options={[
                    ['product_structure', '商品结构错误'], ['color', '颜色/材质错误'], ['text_logo', '文字或 Logo 错误'], ['quantity_accessories', '数量或配件错误'], ['scene', '场景不合适'], ['visual_quality', '视觉质量不好'], ['other', '其他'],
                  ].map(([value, label]) => ({ value, label }))} /></Form.Item></Col>
                  <Col xs={24} md={8}><Form.Item name="review_seconds" label="本次人工审核秒数" rules={[{ required: true }]}><InputNumber min={0} max={86400} precision={0} style={{ width: '100%' }} /></Form.Item></Col>
                </Row>
                <Form.Item name="error_regions_json" label="错误区域 JSON（可选）"><Input.TextArea rows={2} placeholder='例如 [{"x":120,"y":80,"width":200,"height":100,"label":"Logo错误"}]' /></Form.Item>
                {feedbackOutcome === 'rework_requested' && <Form.Item name="rework_instruction" label="返工要求" rules={[{ required: true }]}><Input.TextArea rows={2} /></Form.Item>}
                <Form.Item name="notes" label="补充说明"><Input.TextArea rows={2} /></Form.Item>
                <Space wrap>
                  <Button type="primary" danger={feedbackOutcome !== 'selected'} loading={feedback.isPending} onClick={() => feedback.mutate()}>保存不可变反馈</Button>
                  <Button disabled={!governanceJob.recipe_manifest} onClick={() => {
                    const recipe = governanceJob.recipe_manifest;
                    if (!recipe) return;
                    taskForm.setFieldsValue({
                      sku_id: governanceJob.sku_id, recipe_key: governanceJob.recipe_key, recipe_version: (governanceJob.recipe_version ?? 0) + 1,
                      parent_task_id: governanceJob.id, candidate_round: (governanceJob.candidate_round ?? 0) + 1,
                      processor: governanceJob.processor as TaskFormValues['processor'], operation: governanceJob.operation as TaskFormValues['operation'], purpose: governanceJob.purpose!, channel: governanceJob.channel!, region: governanceJob.region as TaskFormValues['region'],
                      scene_structure: recipe.scene_structure, prompt: recipe.prompt, negative_prompt: recipe.negative_prompt,
                      model: recipe.model, model_version: recipe.model_version, parameters_json: JSON.stringify(recipe.parameters), must_not_change: recipe.must_not_change.join('\n'),
                    });
                    setInputRightsFingerprint(undefined);
                  }}>载入为返工配方</Button>
                </Space>
              </Form>
              {governanceJob.sandbox && <Alert style={{ marginTop: 12 }} type="warning" showIcon title="Sandbox 候选只能拒绝或要求返工" description="带水印或不可发布输出不能选为最终图片，但反馈仍会进入配方记录。" />}
            </Card>
            {recipeSummary.data && <Card size="small" title="E. 作图配方表现">
              <Space wrap>
                <Tag>SKU #{recipeSummary.data.sku_id}</Tag><Tag>候选 {recipeSummary.data.candidates}</Tag><Tag color="success">采用 {recipeSummary.data.selected}</Tag><Tag color="error">拒绝 {recipeSummary.data.rejected}</Tag><Tag color="warning">返工 {recipeSummary.data.rework_requested}</Tag>
                <Tag>通过率 {(recipeSummary.data.acceptance_rate * 100).toFixed(0)}%</Tag><Tag>返工轮次 {recipeSummary.data.rework_rounds}</Tag><Tag>人工审核 {recipeSummary.data.review_seconds} 秒</Tag><Tag>生产耗时 {recipeSummary.data.production_seconds} 秒</Tag><Tag>实际成本 {recipeSummary.data.actual_cost} {recipeSummary.data.currency || ''}</Tag>
              </Space>
            </Card>}
          </>}
        </Space>}
      </Card>
      <Card title="候选图片">
        {!releaseEligible.length ? <Empty description="只有 READY、已有受控输出字节且可发布的非 sandbox 任务才能成为候选；Photoroom canary 永不进入图片集合或发布证明" /> : <Space orientation="vertical" size={16} style={{ width: '100%' }}>
          <Alert type="info" showIcon title="创建 Listing 图片集合" description="必须填写真实 Listing ID；集合创建后仍是草稿，需再次冻结 Owner 选择。" />
          <Form form={setForm} layout="vertical" initialValues={{ channel: 'ozon', locale: 'ru-RU' }}>
            <Row gutter={12}>
              <Col xs={24} md={8}><Form.Item name="listing_id" label="真实 Listing ID" rules={[{ required: true, message: '请填写真实 Listing ID，不会自动生成或使用 mock' }]}><InputNumber min={1} precision={0} style={{ width: '100%' }} placeholder="必须填写" /></Form.Item></Col>
              <Col xs={24} md={8}><Form.Item name="channel" label="渠道" rules={[{ required: true }]}><Input placeholder="例如 ozon" /></Form.Item></Col>
              <Col xs={24} md={8}><Form.Item name="locale" label="语言/地区" rules={[{ required: true }]}><Input placeholder="例如 ru-RU" /></Form.Item></Col>
            </Row>
          </Form>
          <Row gutter={[12, 12]}>{releaseEligible.map((job) => {
            const checked = selected.includes(job.id);
            const order = selected.indexOf(job.id);
            return <Col key={job.id} xs={24} sm={12} lg={8}><Card size="small" title={<Checkbox checked={checked} onChange={(event) => setSelected((current) => event.target.checked ? [...current, job.id] : current.filter((id) => id !== job.id))}>任务 {job.id}</Checkbox>}>
              <ProtectedOutput jobId={job.id} />
              <Space orientation="vertical" style={{ width: '100%', marginTop: 8 }}>
                <Select<ProductImageRole> disabled={!checked} value={roles[job.id] ?? (order === 0 ? 'main' : 'gallery')} style={{ width: '100%' }} onChange={(role) => setRoles((current) => ({ ...current, [job.id]: role }))} options={['main', 'gallery', 'detail', 'size', 'packaging', 'ad_cover'].map((value) => ({ value, label: value }))} />
                <Typography.Text type="secondary">{job.width} × {job.height} {job.format} · 顺序 {order >= 0 ? order + 1 : '未选择'}</Typography.Text>
              </Space>
            </Card></Col>;
          })}</Row>
          <Space>
            <Button type="primary" disabled={!selected.length || Boolean(imageSet)} loading={createSet.isPending} onClick={() => createSet.mutate()}>创建图片集合</Button>
            <Button danger disabled={!imageSet || imageSet.status === 'frozen'} loading={freezeSet.isPending} onClick={() => imageSet && freezeSet.mutate(imageSet.id)}>冻结 Owner 选择</Button>
          </Space>
          {imageSet && <Alert type={imageSet.status === 'frozen' ? 'success' : 'warning'} showIcon title={`图片集合 #${imageSet.id} · ${imageSet.status === 'frozen' ? '已冻结' : '草稿'}`} description={imageSet.status === 'frozen' ? `最终字节清单 SHA-256：${imageSet.manifest_sha256}` : '尚未冻结，不得作为发布放行依据。'} />}
        </Space>}
      </Card>
      <Card title="发布证明管理">
        <Space orientation="vertical" size={16} style={{ width: '100%' }}>
          <Alert type="warning" showIcon title="签发证明不等于已经发布" description="发布证明只证明当前 Listing、冻结图片字节、渠道规则和 Owner 决定在签发时一致。真正发布仍由独立发布流程校验并一次性消费证明；本页面不提供消费按钮。" />
          <Row gutter={[16, 16]}>
            <Col xs={24} xl={12}><Card size="small" title="1. 创建不可变渠道规则快照">
              <Form form={ruleForm} layout="vertical" initialValues={{ channel: imageSet?.channel ?? 'ozon', site: 'ru', locale: imageSet?.locale ?? 'ru-RU', rules_json: '{\n  "main_image_background": "white",\n  "minimum_width": 1200\n}' }}>
                <Row gutter={8}>
                  <Col span={12}><Form.Item name="channel" label="渠道" rules={[{ required: true }]}><Input /></Form.Item></Col>
                  <Col span={12}><Form.Item name="site" label="站点" rules={[{ required: true }]}><Input placeholder="例如 ru" /></Form.Item></Col>
                  <Col span={12}><Form.Item name="locale" label="语言/地区" rules={[{ required: true }]}><Input /></Form.Item></Col>
                  <Col span={12}><Form.Item name="category_id" label="真实渠道类目 ID" rules={[{ required: true }]}><InputNumber min={1} precision={0} style={{ width: '100%' }} /></Form.Item></Col>
                </Row>
                <Form.Item name="rules_json" label="已核验渠道规则（JSON）" rules={[{ required: true }]}><Input.TextArea rows={5} /></Form.Item>
                <Form.Item name="expires_at" label="失效时间（可选）"><Input type="datetime-local" /></Form.Item>
                <Button type="primary" loading={createRule.isPending} onClick={() => createRule.mutate()}>冻结规则快照</Button>
              </Form>
              {ruleSnapshot && <Alert style={{ marginTop: 12 }} type="success" showIcon title={`规则快照 #${ruleSnapshot.id} · 版本 ${ruleSnapshot.version}`} description={`规则 SHA-256：${ruleSnapshot.rules_sha256}`} />}
            </Card></Col>
            <Col xs={24} xl={12}><Card size="small" title="2. 决定已冻结图片集合">
              {!imageSet || imageSet.status !== 'frozen' ? <Empty description="先在上方创建并冻结图片集合" /> : <>
                <Typography.Paragraph>集合 #{imageSet.id} · 版本 {imageSet.version}<br />清单 SHA-256：<Typography.Text code>{imageSet.manifest_sha256}</Typography.Text></Typography.Paragraph>
                <Form form={decisionForm} layout="vertical" initialValues={{ decision: 'approved' }}>
                  <Form.Item name="decision" label="Owner 决定" rules={[{ required: true }]}><Select options={[{ value: 'approved', label: '批准这组精确图片' }, { value: 'rejected', label: '拒绝这组图片' }]} /></Form.Item>
                  <Form.Item name="reason" label="决定理由" rules={[{ required: true, message: '请记录批准或拒绝理由' }]}><Input.TextArea rows={3} /></Form.Item>
                  <Button type="primary" danger={decisionForm.getFieldValue('decision') === 'rejected'} loading={decideSet.isPending} onClick={() => decideSet.mutate()}>保存不可变决定</Button>
                </Form>
                {setDecision && <Alert style={{ marginTop: 12 }} type={setDecision.decision === 'approved' ? 'success' : 'error'} showIcon title={setDecision.decision === 'approved' ? 'Owner 已批准精确图片集合' : 'Owner 已拒绝精确图片集合'} description={setDecision.reason} />}
              </>}
            </Card></Col>
          </Row>
          <Card size="small" title="3. 签发并查看发布证明">
            <Form form={attestationForm} layout="inline" initialValues={{ site: 'ru', ttl_seconds: 900 }}>
              <Form.Item name="platform_account_id" label="真实平台账号 ID" rules={[{ required: true }]}><InputNumber min={1} precision={0} /></Form.Item>
              <Form.Item name="site" label="站点" rules={[{ required: true }]}><Input /></Form.Item>
              <Form.Item name="ttl_seconds" label="有效秒数" rules={[{ required: true }]}><InputNumber min={60} max={3600} precision={0} /></Form.Item>
              <Button type="primary" disabled={!imageSet || !ruleSnapshot || setDecision?.decision !== 'approved'} loading={issueAttestation.isPending} onClick={() => issueAttestation.mutate()}>签发发布证明（不会发布）</Button>
            </Form>
            <Space wrap style={{ marginTop: 16 }}>
              <InputNumber min={1} precision={0} value={attestationLookupId} onChange={(value) => setAttestationLookupId(value ?? undefined)} placeholder="发布证明 ID" />
              <Button loading={lookupAttestation.isPending} onClick={() => lookupAttestation.mutate()}>查询证明状态</Button>
            </Space>
            {attestation && <Alert style={{ marginTop: 12 }} type={attestation.status === 'issued' ? 'success' : attestation.status === 'consumed' ? 'info' : 'error'} showIcon title={`发布证明 #${attestation.id} · ${attestation.status === 'issued' ? '已签发、尚未消费' : attestation.status === 'consumed' ? '已由发布流程消费' : '已撤销'}`} description={<Space orientation="vertical" size={0}><span>Listing #{attestation.listing_id} · 图片集合 #{attestation.image_set_id} · 规则快照 #{attestation.rule_snapshot_id}</span><span>签发：{attestation.issued_at} · 到期：{attestation.expires_at}</span><strong>这条状态不代表渠道已经发布成功。</strong></Space>} />}
          </Card>
        </Space>
      </Card>
    </Space>
    <Modal title="确认真实付费执行" open={Boolean(paidExecutionJob)} width={760} okText="确认批准、预占并执行" cancelText="返回检查" confirmLoading={approveAndExecute.isPending} okButtonProps={{ danger: true, disabled: !completePaidExecutionContext(paidExecutionJob) }} cancelButtonProps={{ disabled: approveAndExecute.isPending }} closable={!approveAndExecute.isPending} mask={{ closable: false }} onCancel={() => setPaidExecutionJob(undefined)} onOk={() => paidExecutionJob && approveAndExecute.mutateAsync(paidExecutionJob)}>
      {!completePaidExecutionContext(paidExecutionJob) ? <Alert type="error" showIcon title="任务上下文不完整，禁止付费批准" description="缺少精确任务版本、SKU、配方哈希、参考图、场景、完整提示词、模型版本、不可改变项或预算预占上限。请创建完整的新任务版本。" /> : paidExecutionJob && <Space orientation="vertical" size={12} style={{ width: '100%' }}>
        <Alert type="warning" showIcon title="这是单次真实付费调用，预算预占不是 Provider 硬封顶" description="OpenAI Images 未提供凌镜可验证的供应商侧幂等查询。凌镜不会自动重试；若响应丢失，可能已扣费但没有可恢复输出，任务会进入待对账。" />
        <Descriptions bordered size="small" column={1}>
          <Descriptions.Item label="精确任务">#{paidExecutionJob.id} / version {paidExecutionJob.version} / SKU #{paidExecutionJob.sku_id}</Descriptions.Item>
          <Descriptions.Item label="配方">{paidExecutionJob.recipe_key} v{paidExecutionJob.recipe_version}<br />recipe SHA-256：<Typography.Text code>{paidExecutionJob.recipe_hash}</Typography.Text><br />manifest SHA-256：<Typography.Text code>{paidExecutionJob.manifest_hash}</Typography.Text></Descriptions.Item>
          <Descriptions.Item label="参考图资产">{paidExecutionJob.recipe_manifest?.reference_asset_ids?.join(', ')}</Descriptions.Item>
          <Descriptions.Item label="场景结构">{paidExecutionJob.recipe_manifest?.scene_structure}</Descriptions.Item>
          <Descriptions.Item label="完整提示词"><Typography.Paragraph copyable style={{ whiteSpace: 'pre-wrap', marginBottom: 0 }}>{paidExecutionJob.recipe_manifest?.prompt}</Typography.Paragraph></Descriptions.Item>
          <Descriptions.Item label="负面约束">{paidExecutionJob.recipe_manifest?.negative_prompt || '无'}</Descriptions.Item>
          <Descriptions.Item label="模型与参数">{paidExecutionJob.recipe_manifest?.model} / {paidExecutionJob.recipe_manifest?.model_version}<br />{JSON.stringify(paidExecutionJob.recipe_manifest?.parameters)}</Descriptions.Item>
          <Descriptions.Item label="绝不能改变">{paidExecutionJob.recipe_manifest?.must_not_change.join('；')}</Descriptions.Item>
          <Descriptions.Item label="外部处理">{paidExecutionJob.processor} / {paidExecutionJob.operation} / region={paidExecutionJob.region}</Descriptions.Item>
          <Descriptions.Item label="预算预占上限">{paidExecutionJob.max_cost} {paidExecutionJob.currency}（非供应商硬封顶）</Descriptions.Item>
        </Descriptions>
      </Space>}
    </Modal>
  </div>;
}
