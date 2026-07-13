import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
const root=`${process.cwd()}/src/app/(main)/business-decisions`;
const list=readFileSync(`${root}/page.tsx`,'utf8');
const detail=readFileSync(`${root}/[id]/page.tsx`,'utf8');
describe('Owner business decision recovery chain',()=>{
 it('lists and creates cases from Owner-scoped authoritative options',()=>{expect(list).toContain("'/v1/business-decisions'");expect(list).toContain('/v1/business-decisions/fact-options');expect(list).toContain('冻结事实并创建案卷');expect(list).not.toContain('type="number"')});
 it('persists exact payload and restores action execution after refresh',()=>{expect(detail).toContain('input_payload');expect(detail).toContain('stableJSONStringify');expect(detail).toContain("'/v1/business-feedback/actions'");expect(detail).toContain('/execute');expect(detail).toContain('刷新可恢复')});
 it('keeps authoritative observations and inferred next recommendations separate',()=>{expect(detail).toContain('/observations');expect(detail).toContain('/next-recommendations');expect(detail).toContain('保存权威观测');expect(detail).toContain('保存inferred建议')});
});
