import { describe, expect, it } from 'vitest';

import { menuGroups } from './menu';

describe('formal navigation', () => {
  it('does not expose the retired Owner mock workspace', () => {
    const paths = menuGroups.flatMap((group) => group.items.map((item) => item.key));

    expect(paths).not.toContain('/owner');
    expect(paths).toContain('/platform-truth');
  });
});
