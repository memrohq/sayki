import { Call, Events } from "@wailsio/runtime";

export type Settings = {
  shortcut: string;
  mode: "push" | "toggle";
  microphoneId: string;
  modelId: string;
  punctuation: boolean;
  injectMethod: "paste" | "type";
  overlay: "bottom" | "top" | "hidden";
  launchAtLogin: boolean;
  historyEnabled: boolean;
  sounds: boolean;
};
export type Status = {
  state: "idle" | "loading" | "recording" | "finishing" | "error";
  error?: string;
  language: string;
  startedAt?: number;
};
export type Mic = { id: string; name: string; isDefault: boolean };
export type Model = {
  id: string; name: string; language: string; kind: "asr" | "punct";
  bytes: number; installed: boolean; downloading: boolean; engine: "" | "parakeet" | "apple";
};
export type Download = { id: string; done: number; total: number; finished: boolean; error?: string };
export type Perms = { microphone: string; accessibility: string };
export type HistoryItem = { id: number; text: string; createdAt: string };

const call = <T,>(method: string, ...args: unknown[]): Promise<T> =>
  Call.ByName(`github.com/memrohq/sayki/internal/app.App.${method}`, ...args) as Promise<T>;

export const api = {
  getSettings: () => call<Settings>("GetSettings"),
  saveSettings: (s: Settings) => call<void>("SaveSettings", s),
  getStatus: () => call<Status>("GetStatus"),
  microphones: () => call<Mic[]>("ListMicrophones"),
  models: () => call<Model[]>("ListModels"),
  download: (id: string) => call<void>("DownloadModel", id),
  cancelDownload: (id: string) => call<void>("CancelDownload", id),
  deleteModel: (id: string) => call<void>("DeleteModel", id),
  permissions: () => call<Perms>("Permissions"),
  requestPermission: (kind: string) => call<void>("RequestPermission", kind),
  openPermissionSettings: (kind: string) => call<void>("OpenPermissionSettings", kind),
  history: () => call<HistoryItem[]>("History"),
  clearHistory: () => call<void>("ClearHistory"),
  toggle: () => call<void>("ToggleRecording"),
  stop: () => call<void>("Stop"),
  cancel: () => call<void>("Cancel"),
  suspendShortcut: () => call<void>("SuspendShortcut"),
  resumeShortcut: () => call<void>("ResumeShortcut"),
};

export function on<T>(name: string, fn: (data: T) => void): () => void {
  return Events.On(name, (e: { data: T }) => fn(e.data));
}

// Keys usable alone; must match shortcuts.SoloKeys in Go.
export const soloKeys: [string, string][] = [
  ["fn", "🌐 Fn"], ["ctrl", "⌃ Control"], ["rightoption", "Right ⌥ Option"], ["rightcmd", "Right ⌘ Command"],
];

export const shortcutLabel = (s: string) =>
  soloKeys.find(([k]) => k === s)?.[1] ??
  s.replace("ctrl+", "⌃").replace("option+", "⌥").replace("shift+", "⇧").replace("cmd+", "⌘")
    .replace("Key", "").replace("Digit", "");
