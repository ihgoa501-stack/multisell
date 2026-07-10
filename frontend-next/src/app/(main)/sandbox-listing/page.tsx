'use client';

import { useEffect } from 'react';
import { Steps, Card } from 'antd';
import { useSandboxListingStore } from './store';
import StepProductEntry from './steps/StepProductEntry';
import StepFieldCompletion from './steps/StepFieldCompletion';
import StepEvidenceCard from './steps/StepEvidenceCard';
import StepRecommendation from './steps/StepRecommendation';
import StepApproval from './steps/StepApproval';
import StepExecution from './steps/StepExecution';
import PageContainer from '@/components/ui/PageContainer';
import { useSearchParams } from 'next/navigation';

const stepTitles = ['录入商品', '补齐字段', '利润证据卡', '上架建议', '审批沙箱任务', '执行与复盘'];

export default function SandboxListingPage() {
  const { currentStep, setStep, setCandidateId, candidateId } = useSandboxListingStore();
  const searchParams = useSearchParams();

  // URL restore: /sandbox-listing?candidate_id=123
  useEffect(() => {
    const cid = searchParams.get('candidate_id');
    if (cid && !candidateId) {
      setCandidateId(Number(cid));
      // TODO: auto-detect current step based on data state (Task 5-7 scope)
      setStep(2);
    }
  }, [searchParams, candidateId, setCandidateId, setStep]);

  const renderStep = () => {
    switch (currentStep) {
      case 1: return <StepProductEntry />;
      case 2: return <StepFieldCompletion />;
      case 3: return <StepEvidenceCard />;
      case 4: return <StepRecommendation />;
      case 5: return <StepApproval />;
      case 6: return <StepExecution />;
    }
  };

  return (
    <PageContainer title="真实商品沙箱上架" subtitle="Sandbox Mode — 不会真实发布">
      <Card>
        <Steps current={currentStep - 1} size="small" style={{ marginBottom: 32 }} items={stepTitles.map((t) => ({ title: t }))} />
        <div style={{ minHeight: 400 }}>{renderStep()}</div>
      </Card>
    </PageContainer>
  );
}
