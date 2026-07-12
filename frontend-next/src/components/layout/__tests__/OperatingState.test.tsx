import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import ActivityFeed from '../ActivityFeed';
import { useAppStore } from '@/stores/app-store';

describe('operating state truthfulness', () => {
  beforeEach(() => useAppStore.setState({ activityFeedOpen: true, unseenCount: 3 }));

  it('shows an explicit empty state instead of fabricated activities', () => {
    render(<ActivityFeed />);
    expect(screen.getByText('暂无真实活动')).toBeInTheDocument();
    expect(screen.queryByText(/Shopee|Ozon|补货|价格监控/)).not.toBeInTheDocument();
  });
});
