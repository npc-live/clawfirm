import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import "./index.css";
import { EventsOn } from "./wailsjs/runtime/runtime";

// EvalJS bridge: listen for "app:eval" events from Go backend and execute JS.
EventsOn("app:eval", (...args: unknown[]) => {
  const { id, script } = args[0] as { id: number; script: string };
  try {
    // eslint-disable-next-line no-eval
    const result = eval(script);
    // Convert result to JSON-serializable string
    const resultStr = result === undefined ? "undefined" : String(result);
    // Call back to Go with the result
    if (window.go?.app?.App?.EvalResult) {
      window.go.app.App.EvalResult(id, resultStr).catch(() => {});
    }
  } catch (e) {
    if (window.go?.app?.App?.EvalResult) {
      window.go.app.App.EvalResult(id, `Error: ${e}`).catch(() => {});
    }
  }
});

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
