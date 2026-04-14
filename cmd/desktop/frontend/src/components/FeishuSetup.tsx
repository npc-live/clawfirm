import { useEffect, useState } from "react";
import { GetFeishuConfig, SaveFeishuConfig } from "../lib/wails-shim";

export function FeishuSetup() {
  const [appID, setAppID] = useState("");
  const [appSecret, setAppSecret] = useState("");
  const [secretMasked, setSecretMasked] = useState("");
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    GetFeishuConfig().then((cfg) => {
      setAppID(cfg.appId ?? "");
      setSecretMasked(cfg.appSecretMasked ?? "");
    }).catch(() => {});
  }, []);

  const configured = appID !== "" && secretMasked !== "";

  async function handleSave() {
    if (!appID.trim() || !appSecret.trim()) {
      setError("App ID 和 App Secret 不能为空");
      return;
    }
    setSaving(true);
    setError("");
    try {
      await SaveFeishuConfig(appID.trim(), appSecret.trim());
      setSecretMasked("••••••••");
      setAppSecret("");
      setEditing(false);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="bg-white rounded-2xl shadow-sm border border-gray-100 p-5">
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <span className="text-lg font-semibold text-gray-900">飞书</span>
          {configured && !editing && (
            <span className="text-xs font-medium px-2 py-0.5 rounded-full bg-green-100 text-green-700">
              已配置
            </span>
          )}
          {!configured && (
            <span className="text-xs font-medium px-2 py-0.5 rounded-full bg-gray-100 text-gray-500">
              未配置
            </span>
          )}
        </div>
        {configured && !editing && (
          <button
            onClick={() => setEditing(true)}
            className="text-sm text-blue-500 hover:text-blue-700 transition-colors"
          >
            修改
          </button>
        )}
      </div>

      {(!configured || editing) ? (
        <div className="space-y-3">
          <p className="text-xs text-gray-400">
            在飞书开放平台创建企业自建应用，开启消息事件订阅（长连接），填入下方凭证。
          </p>
          <div>
            <label className="block text-xs text-gray-500 mb-1">App ID</label>
            <input
              type="text"
              placeholder="cli_xxxxxxxxxxxx"
              value={appID}
              onChange={(e) => setAppID(e.target.value)}
              className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-300"
            />
          </div>
          <div>
            <label className="block text-xs text-gray-500 mb-1">App Secret</label>
            <input
              type="password"
              placeholder={secretMasked || "App Secret"}
              value={appSecret}
              onChange={(e) => setAppSecret(e.target.value)}
              className="w-full px-3 py-2 text-sm border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-300"
            />
          </div>
          {error && <p className="text-xs text-red-500">{error}</p>}
          <div className="flex gap-2">
            <button
              onClick={handleSave}
              disabled={saving}
              className="flex-1 py-2 rounded-xl bg-blue-600 text-white text-sm font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
            >
              {saving ? "保存中…" : "保存并连接"}
            </button>
            {editing && (
              <button
                onClick={() => { setEditing(false); setAppSecret(""); setError(""); }}
                className="px-4 py-2 rounded-xl border border-gray-200 text-sm text-gray-500 hover:bg-gray-50 transition-colors"
              >
                取消
              </button>
            )}
          </div>
        </div>
      ) : (
        <div className="text-sm text-gray-500">
          <p>App ID: <span className="font-mono text-gray-800">{appID}</span></p>
          <p className="mt-1 text-xs text-gray-400">
            通过 WebSocket 长连接接收消息，无需公网地址。
          </p>
        </div>
      )}
    </div>
  );
}
