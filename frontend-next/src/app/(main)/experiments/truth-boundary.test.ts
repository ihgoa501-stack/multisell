import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const pageRoot = `${process.cwd()}/src/app/(main)/experiments`;
const listSource = readFileSync(`${pageRoot}/page.tsx`, 'utf8');
const detailSource = readFileSync(`${pageRoot}/[experimentId]/page.tsx`, 'utf8');
const reportsSource = readFileSync(`${process.cwd()}/src/app/(main)/reports/page.tsx`, 'utf8');

describe('legacy experiment truth boundary', () => {
  it('labels the module as a fact-verification dossier rather than an experiment loop', () => {
    expect(listSource).toContain('经营事实核验案卷');
    expect(listSource).toContain('不证明因果或反馈闭环');
    expect(detailSource).toContain('非因果、非反馈闭环');
  });

  it('does not expose the legacy final-decision completion mutation', () => {
    expect(detailSource).not.toContain('finishCase');
    expect(detailSource).not.toContain('final_decision: decision');
    expect(detailSource).toContain('历史决策阶段已冻结');
    expect(detailSource).toContain('案卷在此停止');
  });

  it('does not present legacy experiment or report amounts as profit and cash authority', () => {
    expect(listSource).not.toContain("dataIndex: 'final_profit_status'");
    expect(listSource).not.toContain("dataIndex: 'cash_recovery_status'");
    expect(detailSource).not.toContain("overrides.final_profit_status = 'final'");
    expect(detailSource).not.toContain("overrides.cash_recovery_status = 'recovered'");
    expect(reportsSource).not.toContain("key: 'profit', label: '利润报表'");
    expect(reportsSource).not.toContain('title="利润总额"');
    expect(reportsSource).not.toContain('title="利润" value={dailyData.profit}');
  });
});
