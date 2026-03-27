import { useEffect, useRef, useCallback } from "react";

export interface WSMessage {
  type: "delta" | "done" | "error" | "tool_start" | "tool_update" | "tool_end";
  content?: string;
  stop_reason?: string;
  // Tool event fields
  tool_call_id?: string;
  tool_name?: string;
  tool_args?: any;
  tool_result?: any;
  tool_is_error?: boolean;
  partial_result?: any;
}

export interface UseWebSocketOptions {
  url: string | null;
  onMessage: (msg: WSMessage) => void;
  onOpen?: () => void;
  onClose?: () => void;
}

/**
 * Manages a single stable WebSocket connection.
 * Reconnects ONLY when `url` changes, never on callback identity changes.
 * Callbacks are stored in refs so they stay current without triggering re-connects.
 */
export function useWebSocket({ url, onMessage, onOpen, onClose }: UseWebSocketOptions) {
  const wsRef = useRef<WebSocket | null>(null);

  // Keep callback refs up to date without triggering the effect.
  const onMessageRef = useRef(onMessage);
  const onOpenRef = useRef(onOpen);
  const onCloseRef = useRef(onClose);
  useEffect(() => { onMessageRef.current = onMessage; });
  useEffect(() => { onOpenRef.current = onOpen; });
  useEffect(() => { onCloseRef.current = onClose; });

  // Connect / disconnect ONLY when url changes.
  useEffect(() => {
    if (!url) return;

    console.log("[ws] connecting to", url);
    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => {
      console.log("[ws] open");
      onOpenRef.current?.();
    };

    ws.onmessage = (ev) => {
      try {
        const msg: WSMessage = JSON.parse(ev.data as string);
        onMessageRef.current(msg);
      } catch {
        onMessageRef.current({ type: "delta", content: ev.data as string });
      }
    };

    ws.onerror = (e) => {
      console.error("[ws] error", e);
      onMessageRef.current({ type: "error", content: "WebSocket error" });
    };

    ws.onclose = (e) => {
      console.log("[ws] closed", e.code, e.reason);
      wsRef.current = null;
      onCloseRef.current?.();
    };

    return () => {
      console.log("[ws] cleanup, closing");
      ws.close();
      wsRef.current = null;
    };
  }, [url]); // ← ONLY url, never the callbacks

  const send = useCallback((data: string) => {
    const ws = wsRef.current;
    if (ws && ws.readyState === WebSocket.OPEN) {
      console.log("[ws] send", data);
      ws.send(data);
    } else {
      console.warn("[ws] send skipped, readyState=", ws?.readyState);
    }
  }, []);

  return { send };
}
