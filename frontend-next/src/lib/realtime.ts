/**
 * WebSocket realtime client for AI command center.
 *
 * Connects to the backend WS hub (ws://localhost:8080/ws) with the stored JWT
 * token and dispatches SSEEvent messages to a callback. Auto-reconnects on
 * disconnect with a 3s delay.
 */

import { useEffect, useRef, useState } from 'react';
import { getToken } from '@/lib/auth';

/** Wire format from backend (SSEEvent). */
export interface SSEEventData {
  event: string;
  trace_id?: string;
  agent_id?: string;
  seq?: number;
  data?: Record<string, unknown>;
  ts?: string;
}

/** Derive WS base from the same env var the API client uses. */
function getWSBase(): string {
  const apiBase =
    process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';
  return apiBase.replace(/^http/, 'ws').replace(/\/api\/?$/, '');
}

/**
 * Hook that maintains a persistent WebSocket connection to the realtime hub.
 * @param onEvent — called for every parsed JSON message from the server.
 * @param enabled — set to false to skip connecting (e.g. user not authed).
 */
export function useAIWebSocket(
  onEvent: (event: SSEEventData) => void,
  enabled = true,
) {
  const onEventRef = useRef(onEvent);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout>>(undefined);
  const wsRef = useRef<WebSocket | null>(null);
  const mounted = useRef(true);
  const enabledRef = useRef(enabled);

  // Keep refs in sync via effects (safer than render-time assignment)
  useEffect(() => {
    onEventRef.current = onEvent;
  }, [onEvent]);
  useEffect(() => {
    enabledRef.current = enabled;
  }, [enabled]);

  // Reconnect counter — incrementing triggers a fresh connection
  const [reconnectKey, setReconnectKey] = useState(0);

  // Connect effect
  useEffect(() => {
    if (!enabled) return;

    const token = getToken();
    if (!token) return;

    const base = getWSBase();
    const url = `${base}/ws?token=${encodeURIComponent(token)}`;

    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onmessage = (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data) as SSEEventData;
        onEventRef.current(data);
      } catch {
        // non-JSON messages are ignored (e.g. pong frames)
      }
    };

    ws.onclose = () => {
      wsRef.current = null;
      if (mounted.current) {
        reconnectTimer.current = setTimeout(() => {
          if (mounted.current && enabledRef.current) {
            setReconnectKey((k) => k + 1);
          }
        }, 3000);
      }
    };

    ws.onerror = () => {
      ws.close();
    };

    mounted.current = true;

    return () => {
      mounted.current = false;
      clearTimeout(reconnectTimer.current);
      wsRef.current?.close();
      wsRef.current = null;
    };
  }, [enabled, reconnectKey]);
}
