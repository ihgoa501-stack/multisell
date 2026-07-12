'use client';
import { useState, type Key } from 'react';
import { Alert, Button, Card, message, Modal, Space, Table, Tag, Typography } from 'antd';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useRouter } from 'next/navigation';
import apiClient from '@/lib/api-client';
import PageContainer from '@/components/ui/PageContainer';
import type { DemandCase } from '@/types/demand-case';
import ComparisonMatrix, { DEMAND_DIMENSIONS, type ComparisonCandidate } from '@/features/demandcase/ComparisonMatrix';
const { Text } = Typography;
const verdictMeta:Record<string,{label:string;color:string}>={lead:{label:'线索',color:'blue'},evidence_missing:{label:'证据不足',color:'orange'},rejected:{label:'已淘汰',color:'red'},experiment_ready:{label:'可申请预检',color:'green'}};
export default function DemandCasesPage(){
 const [selected,setSelected]=useState<Key[]>([]);const [compareOpen,setCompareOpen]=useState(false);
 const router=useRouter(); const qc=useQueryClient(); const query=useQuery({queryKey:['demand-cases'],queryFn:()=>apiClient.getPage<DemandCase>('/v1/demand-cases',{page:'1',size:'100'})});
 const comparison=useQuery({queryKey:['demand-comparison',selected],queryFn:async()=>(await apiClient.get<BackendComparison>('/v1/demand-cases/comparison',{ids:selected.join(',')})).data,enabled:compareOpen&&selected.length>=2&&selected.length<=4});
 const firstBatch=useMutation({mutationFn:()=>apiClient.post('/v1/demand-cases/research/first-public-batch',{}),onSuccess:()=>{message.success('仓库内置公开资料基线已导入');void qc.invalidateQueries({queryKey:['demand-cases']})},onError:(e:Error)=>message.error(e.message)});
 return <PageContainer title="候选市场" subtitle="AI 侦察、独立反证和数据现实审计形成的研究案件；这里没有平台默认答案。" loading={query.isLoading} error={query.isError} errorMsg={(query.error as Error|undefined)?.message} onRetry={()=>void query.refetch()} extra={<Button loading={firstBatch.isPending} onClick={()=>firstBatch.mutate()}>导入 2026-07-11 静态研究基线</Button>}>
  <Alert type="info" showIcon title="研究证据不等于真实付费需求" description="只有来源可追溯且反证独立的案件才可能进入只读数据预检；本页没有采购、发布或投放入口。" style={{marginBottom:16}} />
  <Space style={{marginBottom:12}}><Button type="primary" disabled={selected.length<2||selected.length>4} onClick={()=>setCompareOpen(true)}>同框比较（{selected.length}）</Button><Text type="secondary">请选择 2–4 个候选；选择不会改变任何状态。</Text></Space>
  <Card styles={{body:{padding:0}}}><Table rowKey="id" rowSelection={{selectedRowKeys:selected,onChange:setSelected}} dataSource={query.data?.data??[]} pagination={false} onRow={r=>({onDoubleClick:()=>router.push(`/demand-cases/${r.id}`),style:{cursor:'pointer'}})} columns={[
   {title:'候选市场',render:(_,r)=><Space orientation="vertical" size={0}><Text strong>{r.region} × {r.consumer}</Text><Text type="secondary">{r.need_scenario}</Text></Space>},
   {title:'渠道 / 本地化',render:(_,r)=><Space orientation="vertical" size={0}><Text>{r.sales_channel}</Text><Text type="secondary">{r.target_locale || '待明确'}</Text></Space>},{title:'裁决',dataIndex:'status',render:(v:string)=>{const m=verdictMeta[v]??{label:v,color:'default'};return <Tag color={m.color}>{m.label}</Tag>}},
   {title:'停止线',dataIndex:'stop_condition',ellipsis:true}
  ]}/></Card>
  <Modal title="候选市场八维同框比较" open={compareOpen} onCancel={()=>setCompareOpen(false)} footer={<Button onClick={()=>setCompareOpen(false)}>关闭</Button>} width="96vw">{comparison.isLoading?<Text>正在加载比较证据…</Text>:comparison.error?<Alert type="error" title="比较加载失败" description={(comparison.error as Error).message}/>:comparison.data?<ComparisonMatrix candidates={adaptComparison(comparison.data)}/>:null}</Modal>
 </PageContainer>;
}

type BackendEvidence={id:number;dimension:string;kind:'support'|'counter'|'conflict';truth_status:string;title:string;source_uri?:string;observed_at?:string};
type BackendComparison={dimensions:string[];candidates:Array<{case:DemandCase;evidence_by_dimension:Record<string,BackendEvidence[]>;strongest_counterevidence:string;unknowns:string[]}>};
function adaptComparison(data:BackendComparison):ComparisonCandidate[]{return data.candidates.map(item=>({id:item.case.id,region:item.case.region,consumer:item.case.consumer,needScenario:item.case.need_scenario,salesChannel:item.case.sales_channel,strongestCounterevidence:item.strongest_counterevidence,unknowns:item.unknowns,stopCondition:item.case.stop_condition,dimensions:DEMAND_DIMENSIONS.map(({key})=>({dimension:key,evidence:(item.evidence_by_dimension[key]??[]).map(e=>({id:e.id,role:e.kind,truth:e.truth_status,summary:e.title,sourceUri:e.source_uri,observedAt:e.observed_at})),unknowns:item.unknowns.filter(x=>x.includes(key))}))}))}
