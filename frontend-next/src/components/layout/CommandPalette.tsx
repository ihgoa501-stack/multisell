'use client';

import { useEffect, useCallback } from 'react';
import { useRouter } from 'next/navigation';
import { Command } from 'cmdk';
import { useAppStore } from '@/stores/app-store';
import { menuGroups } from '@/config/menu';

export default function CommandPalette() {
  const router = useRouter();
  const { commandPaletteOpen, setCommandPaletteOpen } = useAppStore();

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setCommandPaletteOpen(!commandPaletteOpen);
      }
    },
    [commandPaletteOpen, setCommandPaletteOpen]
  );

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);

  const handleSelect = (value: string) => {
    router.push(value);
    setCommandPaletteOpen(false);
  };

  if (!commandPaletteOpen) return null;

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 1000,
        background: 'rgba(0,0,0,0.5)',
        display: 'flex',
        alignItems: 'flex-start',
        justifyContent: 'center',
        paddingTop: 120,
      }}
      onClick={() => setCommandPaletteOpen(false)}
    >
      <div
        style={{
          width: 560,
          maxHeight: 420,
          background: '#fff',
          borderRadius: 8,
          boxShadow: '0 8px 30px rgba(0,0,0,0.12)',
          overflow: 'hidden',
        }}
        onClick={(e) => e.stopPropagation()}
      >
        <Command
          style={{ padding: 8 }}
          onKeyDown={(e) => {
            if (e.key === 'Escape') {
              setCommandPaletteOpen(false);
            }
          }}
        >
          <Command.Input
            placeholder="Search pages and actions..."
            style={{
              width: '100%',
              padding: '12px 16px',
              fontSize: 16,
              border: 'none',
              borderBottom: '1px solid #f0f0f0',
              outline: 'none',
            }}
          />
          <Command.List
            style={{
              maxHeight: 340,
              overflow: 'auto',
              padding: '8px 0',
            }}
          >
            <Command.Empty style={{ padding: 16, color: '#999' }}>
              No results found.
            </Command.Empty>
            {menuGroups.map((group) => (
              <Command.Group
                key={group.label}
                heading={group.label}
                style={{ padding: '4px 0' }}
              >
                {group.items.map((item) => (
                  <Command.Item
                    key={item.key}
                    value={item.key}
                    onSelect={handleSelect}
                    style={{
                      padding: '8px 16px',
                      cursor: 'pointer',
                      borderRadius: 4,
                      margin: '0 4px',
                    }}
                  >
                    {item.label}
                  </Command.Item>
                ))}
              </Command.Group>
            ))}
          </Command.List>
        </Command>
      </div>
    </div>
  );
}
