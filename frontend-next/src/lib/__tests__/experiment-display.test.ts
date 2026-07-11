import { describe, expect, it } from 'vitest';
import { formatBlocker, gateMeta, nextExperimentStage, stageGateCodes, truthMeta } from '../experiment-display';

describe('experiment display semantics', () => {
  it('keeps simulated, inferred and unknown evidence visibly untrusted', () => {
    expect(truthMeta.mock.trustedForHighRisk).toBe(false);
    expect(truthMeta.inferred.trustedForHighRisk).toBe(false);
    expect(truthMeta.unknown.trustedForHighRisk).toBe(false);
    expect(truthMeta.actual.trustedForHighRisk).toBe(true);
  });

  it('does not describe return or expired gates as passed', () => {
    expect(gateMeta.return.label).toBe('退回补证');
    expect(gateMeta.expired.label).toBe('证据过期');
    expect(formatBlocker('supplier_ready:return')).toContain('退回补证');
  });

  it('fixes one canonical gate per stage and advances without skipping', () => {
    expect(stageGateCodes.supply).toBe('supply_ready');
    expect(stageGateCodes.cash).toBe('cash_recovered');
    expect(nextExperimentStage('profit')).toBe('cash');
    expect(nextExperimentStage('decision')).toBeNull();
  });
});
