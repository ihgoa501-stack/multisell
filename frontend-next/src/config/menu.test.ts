import { describe, expect, it } from 'vitest';

import { menuGroups } from './menu';

describe('formal navigation', () => {
  it('does not expose the retired Owner mock workspace', () => {
    const paths = menuGroups.flatMap((group) => group.items.map((item) => item.key));

    expect(paths).not.toContain('/owner');
    expect(paths).not.toContain('/candidates');
    expect(paths).not.toContain('/sandbox-listing');
    expect(paths).not.toContain('/fulfillment');
    expect(paths).not.toContain('/design-system');
    expect(menuGroups.flatMap((group) => group.items)).not.toContainEqual(expect.objectContaining({ status: expect.any(String) }));
    expect(paths).toContain('/platform-truth');
    expect(paths).toContain('/approval');
    expect(paths).toContain('/platform-integrations');
  });
});
