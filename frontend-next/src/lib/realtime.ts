/**
 * WebSocket realtime client for AI command center.
 *
 * Connects to the backend WS hub (ws://localhost:8080/ws) with the stored JWT
 * token and dispatches SSEEvent messages to a callback. Auto-reconnects with
 * exponential backoff (1s -> 2s -> 4s -> 8s -> 16s -> 30s max), random jitter,
 * 25s heartbeat ping, and a max of 10 retry attempts.
 */

import { useEffect, useRef, useState, useCallback, startTransition } from 'react';
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

/** Connection status exposed to consumers. */
export type WSConnectionStatus =
  | 'connecting'
  | 'connected'
  | 'disconnected'
  | 'reconnecting';

/** Heartbeat interval in milliseconds. */
const HEARTBEAT_MS = 25_000;
// ponytail: pong timeout (10s) reserved for future heartbeat enforcement.

/** Maximum reconnection attempts before giving up entirely. */
const MAX_RETRIES = 10;

/**
 * Compute the delay (ms) for the n-th reconnection attempt.
 * Exponential backoff: 1s, 2s, 4s, 8s, 16s, capped at 30s.
 * Each delay includes 50% random jitter to prevent thundering herd.
 */
function getBackoffDelay(attempt: number): number {
  const base = Math.min(1000 * Math.pow(2, attempt), 30000);
  // Random jitter: 50%-100% of the base delay
  return base * (0.5 + Math.random() * 0.5);
}

/** Derive WS base from the same env var the API client uses. */
function getWSBase(): string {
  const apiBase =
    process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080/api';
  return apiBase.replace(/^http/, 'ws').replace(/\/api\/?$/, '');
}

export interface UseAIWebSocketReturn {
  status: WSConnectionStatus;
  close: () => void;
}

/**
 * Hook that maintains a persistent WebSocket connection to the realtime hub.
 * @param onEvent - called for every parsed JSON message from the server.
 * @param enabled - set to false to skip connecting (e.g. user not authed).
 */
export function useAIWebSocket(
  onEvent: (event: SSEEventData) => void,
  enabled = true,
): UseAIWebSocketReturn {
  const onEventRef = useRef(onEvent);
  const wsRef = useRef<WebSocket | null>(null);
  const heartbeatRef = useRef<ReturnType<typeof setInterval>>(undefined);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const mountedRef = useRef(true);
  const attemptRef = useRef(0);
  const enabledRef = useRef(enabled);

  const [status, setStatus] = useState<WSConnectionStatus>('disconnected');
  // Incrementing this key triggers a fresh connection attempt
  const [connectKey, setConnectKey] = useState(0);

  // Keep refs in sync via effects (safer than render-time assignment)
  useEffect(() => {
    onEventRef.current = onEvent;
  }, [onEvent]);
  useEffect(() => {
    enabledRef.current = enabled;
  }, [enabled]);

  // Connect effect
  useEffect(() => {
    if (!enabled) {
      startTransition(() => {
        setStatus('disconnected');
      });
      return;
    }

    const isReconnect = attemptRef.current > 0;
    setStatus(isReconnect ? 'reconnecting' : 'connecting');

    const token = getToken();
    if (!token) {
      setStatus('disconnected');
      return;
    }

    const base = getWSBase();
    const url = `${base}/ws?token=${encodeURIComponent(token)}`;

    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => {
      attemptRef.current = 0;
      setStatus('connected');

      // Start 25s heartbeat ping
      heartbeatRef.current = setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'ping' }));
        }
      }, HEARTBEAT_MS);
    };

    ws.onmessage = (e: MessageEvent) => {
      try {
        const data = JSON.parse(e.data) as SSEEventData;
        onEventRef.current(data);
      } catch {
        // non-JSON messages are ignored (e.g. pong frames)
      }
    };

    ws.onclose = () => {
      clearInterval(heartbeatRef.current);
      wsRef.current = null;

      if (mountedRef.current) {
        attemptRef.current++;
        if (attemptRef.current <= MAX_RETRIES) {
          setStatus('reconnecting');
          const delay = getBackoffDelay(attemptRef.current - 1);
          reconnectTimerRef.current = setTimeout(() => {
            if (mountedRef.current && enabledRef.current) {
              setConnectKey((k) => k + 1);
            }
          }, delay);
        } else {
          setStatus('disconnected');
        }
      }
    };

    ws.onerror = () => {
      ws.close();
    };

    mountedRef.current = true;

    return () => {
      mountedRef.current = false;
      clearInterval(heartbeatRef.current);
      clearTimeout(reconnectTimerRef.current);
      ws.close();
      wsRef.current = null;
    };
  }, [enabled, connectKey]);

  const close = useCallback(() => {
    clearInterval(heartbeatRef.current);
    clearTimeout(reconnectTimerRef.current);
    wsRef.current?.close();
    wsRef.current = null;
    attemptRef.current = 0;
    setStatus('disconnected');
  }, []);

  return { status, close };
}
