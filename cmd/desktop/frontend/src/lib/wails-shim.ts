// Wails shim: when running inside Wails (window.go exists), delegate to Wails bindings.
// When running as standalone Web UI, delegate to HTTP API.

import * as wails from "../wailsjs/go/app/App";
import * as api from "./api";

const isWails = typeof window !== "undefined" && !!window.go;

export const IsFirstRun: typeof wails.IsFirstRun = (...args) =>
  isWails ? wails.IsFirstRun(...args) : api.IsFirstRun(...args);

export const GetChannels: typeof wails.GetChannels = (...args) =>
  isWails ? wails.GetChannels(...args) : api.GetChannels(...args);

export const GetChatSessions: typeof wails.GetChatSessions = (...args) =>
  isWails ? wails.GetChatSessions(...args) : api.GetChatSessions(...args);

export const GetHistory: typeof wails.GetHistory = (...args) =>
  isWails ? wails.GetHistory(...args) : api.GetHistory(...args);

export const GetWebhookBaseURL: typeof wails.GetWebhookBaseURL = (...args) =>
  isWails ? wails.GetWebhookBaseURL(...args) : api.GetWebhookBaseURL(...args);

export const GetVersion: typeof wails.GetVersion = (...args) =>
  isWails ? wails.GetVersion(...args) : api.GetVersion(...args);

// Re-export everything else directly from Wails bindings.
// In Web mode these will return undefined (the Wails shim's fallback),
// which components should handle gracefully.
export {
  GetConfig,
  SaveConfig,
  GetProviders,
  SaveAPIKey,
  StartOAuthLogin,
  GetModels,
  TestProviderConnection,
  SaveChannelConfig,
  DeleteChannelConfig,
  TestChannelConnection,
  SendMessage,
  AbortCurrentTurn,
  OpenLogsFolder,
  GetFeishuConfig,
  SaveFeishuConfig,
  GetWhatsAppStatus,
  GetWhatsAppQR,
  LogoutWhatsApp,
  GetConfigRaw,
  SaveConfigRaw,
  GetToolExecutionState,
  GetAllSkills,
  GetSkillContent,
  GetAgentSkills,
  AddSkillPath,
  RemoveSkillPath,
  SearchRemoteSkills,
  InstallRemoteSkill,
  ListCronJobs,
  AddCronJob,
  UpdateCronJob,
  DeleteCronJob,
  ToggleCronJob,
  GetCronJobHistory,
  GetCronJobHistoryAll,
  TriggerCronJob,
  ListMemoryFiles,
  GetMemoryFileContent,
  SaveMemoryFileContent,
  DeleteMemoryFile,
  CreateMemoryFile,
  SearchMemory,
  SyncMemory,
  GetMemoryDir,
  GetSoulPrompt,
  SaveSoulPrompt,
  ListWhipFiles,
  GetWhipFileContent,
  BrowserTestCDP,
  BrowserLaunchChrome,
  BrowserListShortcuts,
  BrowserRunShortcut,
  EnableRemote,
  DisableRemote,
  EnableNgrok,
  DisableNgrok,
  GetRemoteStatus,
  GetVault,
  SetVaultEntry,
  DeleteVaultEntry,
  ListCanvasFiles,
  ReadCanvasFile,
  WriteCanvasFile,
} from "../wailsjs/go/app/App";

// Re-export types
export type {
  ProviderInfo,
  ChannelInfo,
  AgentConfig,
  WhipflowConfig,
  Config,
  HistoryMessage,
  VaultEntry,
  ToolExecutionInfo,
  SkillInfo,
  RemoteSkillInfo,
  ScheduleData,
  CronJob,
  CronJobHistory,
  MemoryFile,
  MemorySearchResult,
  BrowserStatus,
  ShortcutInfo,
  RemoteStatus,
} from "../wailsjs/go/app/App";
