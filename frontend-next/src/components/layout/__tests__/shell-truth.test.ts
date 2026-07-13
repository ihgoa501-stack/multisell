import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';

const root = `${process.cwd()}/src/components/layout`;
const sidebar = readFileSync(`${root}/AppSidebar.tsx`, 'utf8');
const tools = readFileSync(`${root}/ToolPanel.tsx`, 'utf8');
const header = readFileSync(`${root}/AppHeader.tsx`, 'utf8');

describe('global shell truth boundaries', () => {
  it('does not present fixed platform work or invented operating counts as current facts', () => {
    for (const source of [sidebar, tools, header]) {
      expect(source).not.toContain('2,847');
      expect(source).not.toContain('8 待处理');
      expect(source).not.toContain('127 次决策');
      expect(source).not.toContain('3 Agents Online');
      expect(source).not.toContain('补货 Shopee');
      expect(source).not.toContain('Ozon 价格对比');
    }
  });

  it('labels unavailable shell metrics as unverified', () => {
    expect(sidebar).toContain('运行任务事实');
    expect(sidebar).toContain('未接入权威信任事实');
    expect(tools).toContain('未核验');
    expect(header).toContain('Agent 状态未核验');
  });
});
