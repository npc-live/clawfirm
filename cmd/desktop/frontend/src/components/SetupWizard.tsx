import React, { useState } from "react";
import {
  SaveAPIKey,
  GetWebhookBaseURL,
  SaveChannelConfig,
  StartOAuthLogin,
} from "../wailsjs/go/app/App";
import { EventsOn } from "../wailsjs/runtime/runtime";

interface Props {
  onComplete: () => void;
}

const PROVIDERS = [
  // Popular cloud providers
  { id: "anthropic",   label: "Anthropic",        icon: "🤖", requiresKey: true  },
  { id: "openai",      label: "OpenAI",            icon: "🧠", requiresKey: true  },
  { id: "gemini",      label: "Google Gemini",     icon: "♊", requiresKey: true  },
  { id: "deepseek",    label: "DeepSeek",          icon: "🔍", requiresKey: true  },
  { id: "zenmux",      label: "ZenMux",            icon: "⚡", requiresKey: true  },
  // Chinese providers
  { id: "minimax",     label: "MiniMax",           icon: "🎯", requiresKey: true  },
  { id: "moonshot",    label: "Moonshot (Kimi)",   icon: "🌙", requiresKey: true  },
  { id: "volcengine",  label: "Volcengine (Doubao)",icon: "🌋", requiresKey: true  },
  { id: "modelstudio", label: "Model Studio (阿里)", icon: "☁️", requiresKey: true },
  { id: "glm",         label: "GLM / Z.AI",        icon: "🔬", requiresKey: true  },
  // Global providers
  { id: "groq",        label: "Groq",              icon: "⚡", requiresKey: true  },
  { id: "openrouter",  label: "OpenRouter",        icon: "🔀", requiresKey: true  },
  { id: "mistral",     label: "Mistral",           icon: "🇫🇷", requiresKey: true  },
  { id: "together",    label: "Together AI",       icon: "🤝", requiresKey: true  },
  { id: "xai",         label: "xAI (Grok)",        icon: "𝕏",  requiresKey: true  },
  { id: "perplexity",  label: "Perplexity",        icon: "🔎", requiresKey: true  },
  { id: "nvidia",      label: "NVIDIA",            icon: "🎮", requiresKey: true  },
  { id: "venice",      label: "Venice AI",         icon: "🏛️", requiresKey: true  },
  { id: "huggingface", label: "Hugging Face",      icon: "🤗", requiresKey: true  },
  { id: "xiaomi",      label: "Xiaomi (MiMo)",     icon: "📱", requiresKey: true  },
  // Local / self-hosted
  { id: "ollama",      label: "Ollama",            icon: "🦙", requiresKey: false },
  { id: "litellm",     label: "LiteLLM",           icon: "🔁", requiresKey: false },
  { id: "vllm",        label: "vLLM",              icon: "🚀", requiresKey: false },
  { id: "sglang",      label: "SGLang",            icon: "⚙️", requiresKey: false },
];

const DEFAULT_MODELS: Record<string, string> = {
  anthropic:   "claude-haiku-4-5-20251001",
  openai:      "gpt-4o-mini",
  gemini:      "gemini-2.0-flash",
  deepseek:    "deepseek-chat",
  zenmux:      "anthropic/claude-haiku-4-5",
  minimax:     "MiniMax-M1",
  moonshot:    "moonshot-v1-8k",
  volcengine:  "doubao-lite-4k",
  modelstudio: "qwen-turbo",
  glm:         "glm-4-flash",
  groq:        "llama-3.1-8b-instant",
  openrouter:  "openai/gpt-4o-mini",
  mistral:     "mistral-small-latest",
  together:    "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo",
  xai:         "grok-3-mini",
  perplexity:  "sonar",
  nvidia:      "meta/llama-3.1-405b-instruct",
  venice:      "llama-3.3-70b",
  huggingface: "meta-llama/Llama-3.3-70B-Instruct",
  xiaomi:      "mimo-v2-flash",
  ollama:      "llama3.2",
  litellm:     "",
  vllm:        "",
  sglang:      "",
};

export function SetupWizard({ onComplete }: Props) {
  const [step, setStep] = useState<1 | 2 | 3>(1);
  const [selectedProvider, setSelectedProvider] = useState<string | null>(null);
  const [apiKey, setApiKey] = useState("");
  const [webhookURL, setWebhookURL] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleSaveKey() {
    if (!selectedProvider) return;
    setLoading(true);
    setError(null);
    try {
      await SaveAPIKey(selectedProvider, apiKey);
      // Create a default agent using this provider.
      await SaveChannelConfig({
        name: "default",
        provider: selectedProvider,
        model: DEFAULT_MODELS[selectedProvider] ?? "",
        system_prompt: "You are a helpful assistant.",
        max_tokens: 4096,
      });
      const url = await GetWebhookBaseURL();
      setWebhookURL(url);
      setStep(3);
    } catch (e) {
      setError(String(e));
    } finally {
      setLoading(false);
    }
  }

  function handleOAuth() {
    if (!selectedProvider) return;
    // Listen for oauth:callback event from Go.
    const unsub = EventsOn("oauth:callback", () => {
      unsub();
      setStep(3);
    });
    // Trigger OAuth flow (Go opens browser).
    StartOAuthLogin(selectedProvider);
  }

  const provider = PROVIDERS.find((p) => p.id === selectedProvider);

  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center p-6">
      <div className="bg-white rounded-2xl shadow-lg w-full max-w-lg p-8">
        {/* Progress */}
        <div className="flex items-center gap-2 mb-8">
          {[1, 2, 3].map((n) => (
            <React.Fragment key={n}>
              <div
                className={`w-8 h-8 rounded-full flex items-center justify-center text-sm font-medium transition-colors ${
                  step >= n
                    ? "bg-blue-600 text-white"
                    : "bg-gray-200 text-gray-500"
                }`}
              >
                {n}
              </div>
              {n < 3 && (
                <div
                  className={`flex-1 h-1 rounded transition-colors ${
                    step > n ? "bg-blue-600" : "bg-gray-200"
                  }`}
                />
              )}
            </React.Fragment>
          ))}
        </div>

        {/* Step 1 — Choose provider */}
        {step === 1 && (
          <div>
            <h2 className="text-2xl font-semibold mb-2">Choose a Provider</h2>
            <p className="text-gray-500 mb-6">
              Select the AI provider you'd like to use.
            </p>
            <div className="grid grid-cols-2 gap-2 max-h-72 overflow-y-auto pr-1">
              {PROVIDERS.map((p) => (
                <button
                  key={p.id}
                  onClick={() => setSelectedProvider(p.id)}
                  className={`p-4 rounded-xl border-2 text-left transition-all ${
                    selectedProvider === p.id
                      ? "border-blue-600 bg-blue-50"
                      : "border-gray-200 hover:border-gray-300"
                  }`}
                >
                  <div className="text-2xl mb-1">{p.icon}</div>
                  <div className="font-medium">{p.label}</div>
                </button>
              ))}
            </div>
            <button
              disabled={!selectedProvider}
              onClick={() => setStep(2)}
              className="mt-6 w-full py-3 bg-blue-600 text-white rounded-xl font-medium disabled:opacity-40 hover:bg-blue-700 transition-colors"
            >
              Continue
            </button>
          </div>
        )}

        {/* Step 2 — API key / OAuth */}
        {step === 2 && provider && (
          <div>
            <h2 className="text-2xl font-semibold mb-2">
              Connect {provider.label}
            </h2>
            {provider.requiresKey ? (
              <>
                <p className="text-gray-500 mb-4">
                  Enter your {provider.label} API key.
                </p>
                <input
                  type="password"
                  placeholder="sk-..."
                  value={apiKey}
                  onChange={(e) => setApiKey(e.target.value)}
                  className="w-full border border-gray-300 rounded-lg px-4 py-3 mb-4 focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </>
            ) : (
              <p className="text-gray-500 mb-4">
                Ollama runs locally — no API key needed.
              </p>
            )}
            {error && (
              <p className="text-red-600 text-sm mb-3">{error}</p>
            )}
            <button
              disabled={loading || (provider.requiresKey && !apiKey)}
              onClick={provider.requiresKey ? handleSaveKey : handleSaveKey}
              className="w-full py-3 bg-blue-600 text-white rounded-xl font-medium disabled:opacity-40 hover:bg-blue-700 transition-colors"
            >
              {loading ? "Saving…" : "Save & Continue"}
            </button>
            {/* OAuth option for providers that support it */}
            {provider.id === "google" && (
              <button
                onClick={handleOAuth}
                className="mt-3 w-full py-3 border border-gray-300 rounded-xl font-medium hover:bg-gray-50 transition-colors"
              >
                Sign in with Google
              </button>
            )}
            <button
              onClick={() => setStep(1)}
              className="mt-3 w-full py-2 text-gray-500 text-sm hover:text-gray-700"
            >
              ← Back
            </button>
          </div>
        )}

        {/* Step 3 — Done */}
        {step === 3 && (
          <div>
            <div className="text-5xl mb-4">🎉</div>
            <h2 className="text-2xl font-semibold mb-2">You're all set!</h2>
            <p className="text-gray-500 mb-6">
              Your gateway is running. Use the URL below as your webhook
              endpoint.
            </p>
            {webhookURL && (
              <div className="bg-gray-100 rounded-lg px-4 py-3 font-mono text-sm break-all mb-6">
                {webhookURL}/ws/default/&lt;sessionID&gt;
              </div>
            )}
            <button
              onClick={onComplete}
              className="w-full py-3 bg-blue-600 text-white rounded-xl font-medium hover:bg-blue-700 transition-colors"
            >
              Open Dashboard →
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
