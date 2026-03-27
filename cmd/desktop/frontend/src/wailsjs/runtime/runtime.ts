// Wails runtime shim — replaced by the real wailsjs/runtime in Wails dev/build mode.
// In a browser dev environment (e.g., `vite dev` outside Wails), these are no-ops.

declare global {
  interface Window {
    // Wails v2 bridge: window.go.<package>.<StructName>.<Method>(..args)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    go?: Record<string, Record<string, Record<string, (...args: unknown[]) => Promise<unknown>>>>;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    runtime?: Record<string, (...args: unknown[]) => unknown>;
  }
}

/**
 * Call a Go method bound by Wails.
 * Wails v2 bridge path: window.go[pkg][structName][method]
 * e.g. callGo("app", "App", "GetVersion")
 */
export async function callGo<T>(
  pkg: string,
  structName: string,
  method: string,
  ...args: unknown[]
): Promise<T> {
  const fn = window.go?.[pkg]?.[structName]?.[method];
  if (typeof fn === "function") {
    return fn(...args) as Promise<T>;
  }
  console.warn(`[wails] ${pkg}.${structName}.${method} not available`);
  return undefined as unknown as T;
}

/** Register a Wails event listener. Returns an unsubscribe function. */
export function EventsOn(
  event: string,
  callback: (...args: unknown[]) => void
): () => void {
  const fn = window.runtime?.["EventsOn"];
  if (typeof fn === "function") {
    fn(event, callback);
  }
  return () => {
    const off = window.runtime?.["EventsOff"];
    if (typeof off === "function") {
      off(event);
    }
  };
}

/** Emit a Wails event from the frontend to Go. */
export function EventsEmit(event: string, ...args: unknown[]): void {
  const fn = window.runtime?.["EventsEmit"];
  if (typeof fn === "function") {
    fn(event, ...args);
  }
}
