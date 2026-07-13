import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';

vi.mock('@/lib/api-client', () => ({ default: { get: vi.fn(), post: vi.fn() } }));
import apiClient from '@/lib/api-client';
import PurchasePage from './page';

function renderPage(){const q=new QueryClient({defaultOptions:{queries:{retry:false},mutations:{retry:false}}});return render(<QueryClientProvider client={q}><PurchasePage/></QueryClientProvider>)}
const purchase={id:21,supplier_id:2,sku_mapping_id:3,cost_version_id:4,inventory_id:5,internal_sku_id:6,quantity:10,received_quantity:0,unit_amount_minor:1250,total_amount_minor:12500,currency:'CNY',status:'requested',request_sha256:'a'.repeat(64),created_at:'2026-07-12T00:00:00Z'};

describe('purchase authority owner page',()=>{
 beforeEach(()=>{vi.clearAllMocks();vi.mocked(apiClient.get).mockImplementation(async(path:string)=>({code:0,message:'ok',data:path.endsWith('/21')?{purchase,external_facts:[],inventory_ledger:[] }:[purchase]}));vi.mocked(apiClient.post).mockResolvedValue({code:0,message:'ok',data:purchase})});
 it('shows only authoritative minor-unit purchase status and truth boundary',async()=>{const user=userEvent.setup();renderPage();expect(await screen.findByText('待 Owner 决定')).toBeInTheDocument();expect(screen.getByText('125.00 CNY')).toBeInTheDocument();expect(screen.getByText(/内部状态不能冒充/)).toBeInTheDocument();await user.click(screen.getByRole('button',{name:'#21'}));expect(await screen.findByText('请求 SHA')).toBeInTheDocument();expect(screen.getByRole('button',{name:'绑定 selected Owner 决定'})).toBeInTheDocument();expect(screen.queryByText('记录真实收货')).not.toBeInTheDocument()});
 it('creates a request only from authority identifiers and idempotency key',async()=>{const user=userEvent.setup();renderPage();await screen.findByText('待 Owner 决定');await user.click(screen.getByRole('button',{name:'新建采购请求'}));expect(screen.getByLabelText('权威供应商 ID')).toBeInTheDocument();expect(screen.getByLabelText('Canonical SKU Mapping ID')).toBeInTheDocument();expect(screen.getByLabelText('精确成本版本 ID')).toBeInTheDocument();expect(screen.queryByLabelText('总金额')).not.toBeInTheDocument()});
 it('offers a deliberate manifest-bound selected Owner decision instead of asking for an opaque decision id',async()=>{const user=userEvent.setup();renderPage();await screen.findByText('待 Owner 决定');await user.click(screen.getByRole('button',{name:'#21'}));await screen.findByText('请求 SHA');await user.click(screen.getByRole('button',{name:'绑定 selected Owner 决定'}));expect(screen.getByLabelText('Owner 批准理由')).toBeInTheDocument();expect(screen.getByLabelText('本次决定幂等键')).toBeInTheDocument();expect(screen.queryByLabelText('Owner Decision ID')).not.toBeInTheDocument();expect(screen.getByText(/不会自动向供应商下单/)).toBeInTheDocument()});
});
