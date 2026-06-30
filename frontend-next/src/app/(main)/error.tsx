'use client';

import { useEffect } from 'react';

/**
 * Error boundary for the authenticated `(main)` route group.
 * Catches all page-level crashes and shows a friendly message with a retry button.
 */
export default function MainError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    // Log the error for debugging (Sentry will also pick this up)
    console.error('Main layout error:', error);
  }, [error]);

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        height: '100vh',
        padding: 24,
        textAlign: 'center',
        background: 'var(--bg)',
        color: 'var(--t1)',
      }}
    >
      <div
        style={{
          fontSize: '3rem',
          marginBottom: 16,
          lineHeight: 1,
          fontFamily: 'var(--mono)',
        }}
      >
        :(
      </div>
      <h2
        style={{
          fontSize: '1.1rem',
          fontWeight: 600,
          marginBottom: 8,
          color: 'var(--t1)',
        }}
      >
        页面出错了
      </h2>
      <p
        style={{
          fontSize: '0.85rem',
          color: 'var(--t3)',
          marginBottom: 24,
          maxWidth: 400,
          lineHeight: 1.5,
        }}
      >
        {error.message
          ? `错误信息: ${error.message}`
          : '发生了未知错误，请稍后重试。'}
      </p>
      <button
        onClick={() => reset()}
        style={{
          padding: '8px 24px',
          fontSize: '0.85rem',
          fontWeight: 500,
          borderRadius: 6,
          border: '1px solid var(--bd)',
          background: 'var(--s1)',
          color: 'var(--t1)',
          cursor: 'pointer',
          transition: 'background var(--dur-micro)',
        }}
        onMouseEnter={(e) => {
          (e.currentTarget as HTMLElement).style.background = 'var(--s2)';
        }}
        onMouseLeave={(e) => {
          (e.currentTarget as HTMLElement).style.background = 'var(--s1)';
        }}
      >
        重试
      </button>
    </div>
  );
}
