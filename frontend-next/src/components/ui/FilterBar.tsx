'use client';

import { useState, useCallback, ReactNode } from 'react';
import { Input, Select, Button } from 'antd';
import { ReloadOutlined, FilterOutlined } from '@ant-design/icons';

export interface FilterOption {
  key: string;
  label: string;
  options: { label: string; value: string | number | null }[];
  /** Placeholder for the select. */
  placeholder?: string;
  /** Width of the select control. */
  width?: number;
}

export interface FilterBarProps {
  /** Search value. */
  search?: string;
  /** Fired when the search text changes. */
  onSearch?: (value: string) => void;
  /** Search input placeholder. */
  searchPlaceholder?: string;
  /** Multi-dimension filter configs. */
  filters?: FilterOption[];
  /** Current filter values keyed by filter key. */
  filterValues?: Record<string, string | number | null | undefined>;
  /** Fired when any filter changes. */
  onFilterChange?: (key: string, value: string | number | null) => void;
  /** Fired when all filters and search are reset. */
  onReset?: () => void;
  /** Extra actions rendered after the filters. */
  extra?: ReactNode;
  /** Whether to show the filter icon. */
  showIcon?: boolean;
}

export default function FilterBar({
  search,
  onSearch,
  searchPlaceholder = '搜索...',
  filters,
  filterValues,
  onFilterChange,
  onReset,
  extra,
  showIcon = true,
}: FilterBarProps) {
  const [localSearch, setLocalSearch] = useState(search ?? '');

  const handleSearch = useCallback(
    (value: string) => {
      setLocalSearch(value);
      onSearch?.(value);
    },
    [onSearch]
  );

  const hasActiveFilters =
    (search && search.length > 0) ||
    (filters && filterValues && Object.values(filterValues).some((v) => v !== undefined && v !== null));

  return (
    <div
      style={{
        display: 'flex',
        gap: 12,
        alignItems: 'center',
        flexWrap: 'wrap',
        marginBottom: 16,
      }}
    >
      {showIcon && <FilterOutlined style={{ color: '#999', fontSize: 16 }} />}

      <Input.Search
        placeholder={searchPlaceholder}
        value={localSearch}
        onChange={(e) => setLocalSearch(e.target.value)}
        onSearch={handleSearch}
        allowClear
        style={{ maxWidth: 360, minWidth: 200 }}
        size="middle"
      />

      {filters?.map((filter) => (
        <Select
          key={filter.key}
          placeholder={filter.placeholder ?? `选择${filter.label}`}
          value={filterValues?.[filter.key] ?? undefined}
          onChange={(value) => onFilterChange?.(filter.key, value)}
          allowClear
          options={filter.options}
          style={{ width: filter.width ?? 140 }}
          size="middle"
        />
      ))}

      {extra}

      {hasActiveFilters && onReset && (
        <Button
          icon={<ReloadOutlined />}
          size="small"
          onClick={() => {
            setLocalSearch('');
            onReset();
          }}
        >
          重置
        </Button>
      )}
    </div>
  );
}
