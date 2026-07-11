'use client';
import { Alert, Button, Card, Descriptions, Empty, Space, Table, Tag, Typography } from 'antd';
import { ArrowLeftOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { useParams, useRouter } from 'next/navigation';
import apiClient from '@/lib/api-client'; import PageContainer from '@/components/ui/PageContainer';
import type { DemandDetail, OwnerDecisionCard } from '@/types/demand-case';
const { Text }=Typography;
export default function DemandCaseDetailPage(){const {id=''}=useParams<{id:string}>();const router=useRouter();
 const detail=useQuery({queryKey:['demand-case',id],queryFn:async()=>(await apiClient.get<DemandDetail>(`/v1/demand-cases/${id}`)).data,enabled:!!id});
 const card=useQuery({queryKey:['demand-card',id],queryFn:async()=>(await apiClient.get<OwnerDecisionCard>(`/v1/demand-cases/${id}/decision-card`)).data,enabled:!!id}); const d=detail.data,c=card.data;
 return <PageContainer title={d?`${d.case.region} × ${d.case.consumer}`:'候选市场案件'} subtitle={d?.case.need_scenario} loading={detail.isLoading||card.isLoading} error={detail.isError||card.isError} errorMsg={(detail.error as Error|undefined)?.message||(card.error as Error|undefined)?.message} onRetry={()=>{void detail.refetch();void card.refetch()}} extra={<Button icon={<ArrowLeftOutlined/>} onClick={()=>router.push('/demand-cases')}>全部候选</Button>}>
  {c&&<Card title="Owner 六行决策卡" style={{marginBottom:16}}><Descriptions column={1} bordered items={[
   {key:'h',label:'我们怀疑什么',children:c.hypothesis},{key:'p',label:'已经证明什么',children:c.proven},{key:'n',label:'还不能证明什么',children:c.not_proven},{key:'x',label:'最强反证',children:c.strongest_counterevidence},{key:'a',label:'下一权限或成本',children:c.next_authority_or_cost},{key:'s',label:'停止线',children:c.stop_condition||'尚未冻结'}]}/></Card>}
  {d?.verdict?.blockers?.length?<Alert type="warning" showIcon title="当前未知与阻塞" description={<Space wrap>{d.verdict.blockers.map(x=><Tag key={x}>{x}</Tag>)}</Space>} style={{marginBottom:16}}/>:<Alert type="success" showIcon title="研究阶段未记录阻塞" style={{marginBottom:16}}/>}
  <Card title="证据与反证" style={{marginBottom:16}}><Table rowKey="id" pagination={false} dataSource={d?.evidence??[]} locale={{emptyText:<Empty description="尚无证据"/>}} columns={[{title:'作用',dataIndex:'kind',render:(v:string)=><Tag color={v==='counter'?'red':v==='conflict'?'orange':'blue'}>{v}</Tag>},{title:'维度',dataIndex:'dimension'},{title:'事实',dataIndex:'title'},{title:'真实性',dataIndex:'truth_status'},{title:'来源',render:(_,r)=><a href={r.source_uri} target="_blank" rel="noreferrer">查看原始来源</a>},{title:'Run',dataIndex:'run_id',render:(v:string)=><Text code>{v}</Text>} ]}/></Card>
  <Card title="不可变原始快照">{d?.snapshots?.length?<Descriptions column={1} items={d.snapshots.map(s=>({key:s.id,label:s.run_type,children:<Space><Text code>{s.run_id}</Text><a href={s.source_uri} target="_blank" rel="noreferrer">来源</a><Text type="secondary">SHA256 {s.raw_sha256.slice(0,12)}…</Text></Space>}))}/>:<Empty description="尚无研究快照"/>}</Card>
 </PageContainer>}
