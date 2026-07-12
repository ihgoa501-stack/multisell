'use client';
import { Alert, Button, Card, message, Space, Table, Tag, Typography } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import apiClient from '@/lib/api-client';
import PageContainer from '@/components/ui/PageContainer';
import type { DemandCase } from '@/types/demand-case';
const { Text } = Typography;
const verdictMeta:Record<string,{label:string;color:string}>={lead:{label:'线索',color:'blue'},evidence_missing:{label:'证据不足',color:'orange'},rejected:{label:'已淘汰',color:'red'},experiment_ready:{label:'可申请预检',color:'green'}};
export default function DemandCasesPage(){
 const router=useRouter(); const qc=useQueryClient(); const query=useQuery({queryKey:['demand-cases'],queryFn:()=>apiClient.getPage<DemandCase>('/v1/demand-cases',{page:'1',size:'100'})});
 const firstBatch=useMutation({mutationFn:()=>apiClient.post('/v1/demand-cases/research/first-public-batch',{}),onSuccess:()=>{message.success('仓库内置公开资料基线已导入');void qc.invalidateQueries({queryKey:['demand-cases']})},onError:(e:Error)=>message.error(e.message)});
 return <PageContainer title="候选市场" subtitle="AI 侦察、独立反证和数据现实审计形成的研究案件；这里没有平台默认答案。" loading={query.isLoading} error={query.isError} errorMsg={(query.error as Error|undefined)?.message} onRetry={()=>void query.refetch()} extra={<Button loading={firstBatch.isPending} onClick={()=>firstBatch.mutate()}>导入 2026-07-11 静态研究基线</Button>}>
  <Alert type="info" showIcon title="研究证据不等于真实付费需求" description="只有来源可追溯且反证独立的案件才可能进入只读数据预检；本页没有采购、发布或投放入口。" style={{marginBottom:16}} />
  <Card styles={{body:{padding:0}}}><Table rowKey="id" dataSource={query.data?.data??[]} pagination={false} onRow={r=>({onClick:()=>router.push(`/demand-cases/${r.id}`),style:{cursor:'pointer'}})} columns={[
   {title:'候选市场',render:(_,r)=><Space orientation="vertical" size={0}><Text strong>{r.region} × {r.consumer}</Text><Text type="secondary">{r.need_scenario}</Text></Space>},
   {title:'渠道 / 本地化',render:(_,r)=><Space orientation="vertical" size={0}><Text>{r.sales_channel}</Text><Text type="secondary">{r.target_locale || '待明确'}</Text></Space>},{title:'裁决',dataIndex:'status',render:(v:string)=>{const m=verdictMeta[v]??{label:v,color:'default'};return <Tag color={m.color}>{m.label}</Tag>}},
   {title:'停止线',dataIndex:'stop_condition',ellipsis:true}
  ]}/></Card>
 </PageContainer>;
}
