import { useCallback, useEffect, useState, type ReactNode } from "react";
import { api, on, shortcutLabel, soloKeys, type Download, type HistoryItem, type Mic, type Model, type Perms, type Settings } from "./api";

type Tab = "general" | "models" | "permissions" | "history";
const TABS: { id: Tab; label: string }[] = [
  { id: "general", label: "General" },
  { id: "models", label: "Models" },
  { id: "permissions", label: "Permissions" },
  { id: "history", label: "History" },
];

export default function SettingsWindow() {
  const [tab, setTab] = useState<Tab>("general");
  const [s, setS] = useState<Settings | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    api.getSettings().then(setS);
    return on<Settings>("sayki:settings", setS);
  }, []);

  const save = useCallback(async (patch: Partial<Settings>) => {
    if (!s) return;
    const next = { ...s, ...patch };
    try {
      await api.saveSettings(next);
      setS(next);
      setError("");
    } catch (e) {
      setError(String((e as Error)?.message ?? e));
    }
  }, [s]);

  if (!s) return null;
  return (
    <div className="flex h-full bg-neutral-50/60 text-[13px] text-neutral-900 dark:bg-neutral-900/60 dark:text-neutral-100">
      <aside className="drag flex w-48 shrink-0 flex-col gap-0.5 border-r border-black/5 px-3 pt-12 dark:border-white/5">
        <div className="mb-4 px-2">
          <div className="text-[15px] font-semibold tracking-tight">Sayki</div>
          <div className="text-xs text-neutral-500">Private voice typing</div>
        </div>
        {TABS.map((t) => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={`no-drag rounded-md px-2 py-1.5 text-left transition-colors ${
              tab === t.id ? "bg-black/[0.07] font-medium dark:bg-white/10" : "text-neutral-600 hover:bg-black/[0.04] dark:text-neutral-400 dark:hover:bg-white/5"
            }`}
          >
            {t.label}
          </button>
        ))}
        <div className="mt-auto mb-4 px-2 text-[11px] leading-4 text-neutral-500">
          Everything runs on this Mac. No audio or text ever leaves it.
        </div>
      </aside>
      <main className="flex-1 overflow-y-auto bg-white px-8 pt-10 pb-8 dark:bg-neutral-950">
        <div className="drag absolute inset-x-0 top-0 h-8" />
        {error && (
          <div className="mb-4 rounded-lg border border-amber-500/30 bg-amber-500/10 px-3 py-2 text-amber-700 dark:text-amber-300">{error}</div>
        )}
        {tab === "general" && <General s={s} save={save} />}
        {tab === "models" && <Models s={s} save={save} onError={setError} />}
        {tab === "permissions" && <Permissions />}
        {tab === "history" && <History s={s} save={save} />}
      </main>
    </div>
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="mb-7">
      <h2 className="mb-2 text-xs font-medium uppercase tracking-wide text-neutral-500">{title}</h2>
      <div className="divide-y divide-black/5 rounded-xl border border-black/[0.08] bg-neutral-50/50 dark:divide-white/5 dark:border-white/[0.08] dark:bg-white/[0.03]">
        {children}
      </div>
    </section>
  );
}

function Row({ label, hint, children }: { label: string; hint?: string; children: ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-6 px-4 py-3">
      <div>
        <div>{label}</div>
        {hint && <div className="mt-0.5 text-xs text-neutral-500">{hint}</div>}
      </div>
      <div className="shrink-0">{children}</div>
    </div>
  );
}

function Toggle({ on: checked, set }: { on: boolean; set: (v: boolean) => void }) {
  return (
    <button
      role="switch"
      aria-checked={checked}
      onClick={() => set(!checked)}
      className={`relative h-[22px] w-[38px] rounded-full transition-colors ${checked ? "bg-accent" : "bg-neutral-300 dark:bg-neutral-700"}`}
    >
      <span className={`absolute top-[2px] size-[18px] rounded-full bg-white shadow transition-[left] ${checked ? "left-[18px]" : "left-[2px]"}`} />
    </button>
  );
}

function Select<T extends string>({ value, options, set }: { value: T; options: [T, string][]; set: (v: T) => void }) {
  return (
    <select
      value={value}
      onChange={(e) => set(e.target.value as T)}
      className="max-w-60 rounded-md border border-black/10 bg-white px-2 py-1 dark:border-white/10 dark:bg-neutral-800"
    >
      {options.map(([v, l]) => <option key={v} value={v}>{l}</option>)}
    </select>
  );
}

function General({ s, save }: { s: Settings; save: (p: Partial<Settings>) => void }) {
  const [mics, setMics] = useState<Mic[]>([]);
  useEffect(() => { api.microphones().then(setMics).catch(() => setMics([])); }, []);
  return (
    <>
      <Section title="Dictation">
        <Row label="Shortcut" hint={shortcutHint(s.shortcut)}>
          <div className="flex gap-2">
            <ShortcutRecorder value={s.shortcut} set={(shortcut) => save({ shortcut })} />
            <Select
              value={soloKeys.some(([k]) => k === s.shortcut) ? s.shortcut : ""}
              set={(k) => k && save({ shortcut: k })}
              options={[["", "Single key…"], ...soloKeys]}
            />
          </div>
        </Row>
        <Row label="Mode" hint={s.mode === "push" ? "Hold the shortcut while you speak." : "Press once to start, again to insert."}>
          <Select value={s.mode} set={(mode) => save({ mode })} options={[["toggle", "Toggle"], ["push", "Push to talk"]]} />
        </Row>
        <Row label="Sounds" hint="A soft click when recording starts, stops or is cancelled.">
          <Toggle on={s.sounds} set={(sounds) => save({ sounds })} />
        </Row>
        <Row label="Microphone">
          <Select
            value={s.microphoneId}
            set={(microphoneId) => save({ microphoneId })}
            options={[["", "System default"], ...mics.map((m): [string, string] => [m.id, m.name])]}
          />
        </Row>
      </Section>
      <Section title="Inserting text">
        <Row
          label="Method"
          hint={s.injectMethod === "paste" ? "Fast and reliable. Your clipboard is restored afterwards." : "Types characters one by one. Never touches the clipboard."}
        >
          <Select value={s.injectMethod} set={(injectMethod) => save({ injectMethod })} options={[["paste", "Paste"], ["type", "Type"]]} />
        </Row>
        <Row label="Auto-punctuation" hint="English only; needs the punctuation model.">
          <Toggle on={s.punctuation} set={(punctuation) => save({ punctuation })} />
        </Row>
      </Section>
      <Section title="Appearance & startup">
        <Row label="Recording overlay">
          <Select value={s.overlay} set={(overlay) => save({ overlay })} options={[["bottom", "Bottom of screen"], ["top", "Top of screen"], ["hidden", "Hidden"]]} />
        </Row>
        <Row label="Launch at login">
          <Toggle on={s.launchAtLogin} set={(launchAtLogin) => save({ launchAtLogin })} />
        </Row>
      </Section>
    </>
  );
}

function shortcutHint(sc: string) {
  if (sc === "fn") return "Tap Fn to start and stop, or hold it. Set System Settings → Keyboard → “Press 🌐 key to” → Do Nothing, so macOS doesn't also react.";
  if (sc === "ctrl") return "Tap or hold Control alone. Careful: holding Control a moment before Ctrl+C or Ctrl+A will start dictation.";
  if (soloKeys.some(([k]) => k === sc)) return "Tap the key alone to start and stop, or hold it while you speak.";
  return "Works in any app. Or pick a single key, like Fn, as Wispr Flow does.";
}

function ShortcutRecorder({ value, set }: { value: string; set: (v: string) => void }) {
  const [recording, setRecording] = useState(false);
  useEffect(() => {
    if (!recording) return;
    api.suspendShortcut();
    const onKey = (e: KeyboardEvent) => {
      e.preventDefault();
      if (e.code === "Escape") return setRecording(false);
      if (/^(Shift|Control|Alt|Meta|OS|Fn)/.test(e.code)) return;
      const mods = [e.ctrlKey && "ctrl", e.altKey && "option", e.shiftKey && "shift", e.metaKey && "cmd"].filter(Boolean);
      set([...mods, e.code].join("+"));
      setRecording(false);
    };
    window.addEventListener("keydown", onKey);
    return () => { window.removeEventListener("keydown", onKey); api.resumeShortcut().catch(() => {}); };
  }, [recording, set]);
  return (
    <button
      onClick={() => setRecording(!recording)}
      className={`min-w-28 rounded-md border px-3 py-1 font-medium tracking-wide ${
        recording ? "border-accent text-accent" : "border-black/10 bg-white dark:border-white/10 dark:bg-neutral-800"
      }`}
    >
      {recording ? "Press keys…" : shortcutLabel(value)}
    </button>
  );
}

const mb = (b: number) => `${Math.round(b / 1e6)} MB`;

function Models({ s, save, onError }: { s: Settings; save: (p: Partial<Settings>) => void; onError: (e: string) => void }) {
  const [models, setModels] = useState<Model[]>([]);
  const [progress, setProgress] = useState<Record<string, Download>>({});
  const refresh = () => api.models().then(setModels);
  useEffect(() => {
    refresh();
    return on<Download>("sayki:download", (d) => {
      setProgress((p) => ({ ...p, [d.id]: d }));
      if (d.finished) {
        refresh();
        if (d.error && d.error !== "cancelled") onError(`Download failed: ${d.error}`);
      }
    });
  }, [onError]);

  const row = (m: Model) => {
    const p = progress[m.id];
    const busy = m.downloading || (p && !p.finished);
    const active = m.kind === "asr" ? s.modelId === m.id : s.punctuation;
    return (
      <div key={m.id} className="flex items-center gap-4 px-4 py-3">
        <div className="flex-1">
          <div className="flex items-center gap-2">
            {m.name}
            {m.installed && active && <span className="rounded bg-accent/15 px-1.5 text-[10px] font-semibold uppercase text-accent">In use</span>}
          </div>
          <div className="mt-0.5 text-xs text-neutral-500">
            {busy && p ? `Downloading… ${mb(p.done)} of ${mb(p.total)}`
              : m.engine === "apple" ? "Built into macOS · on-device, nothing to download"
              : m.installed ? "Installed · works offline" : `${mb(m.bytes)} download`}
          </div>
          {busy && p && (
            <div className="mt-1.5 h-1 overflow-hidden rounded-full bg-black/10 dark:bg-white/10">
              <div className="h-full bg-accent transition-[width]" style={{ width: `${(p.done / p.total) * 100}%` }} />
            </div>
          )}
        </div>
        {busy ? (
          <Btn onClick={() => api.cancelDownload(m.id)}>Cancel</Btn>
        ) : m.installed ? (
          <div className="flex gap-2">
            {m.kind === "asr" && s.modelId !== m.id && <Btn primary onClick={() => save({ modelId: m.id })}>Use</Btn>}
            {m.engine !== "apple" && <Btn onClick={() => api.deleteModel(m.id).then(refresh).catch((e) => onError(String(e?.message ?? e)))}>Remove</Btn>}
          </div>
        ) : (
          <Btn primary onClick={() => api.download(m.id).then(refresh)}>Download</Btn>
        )}
      </div>
    );
  };
  return (
    <>
      <p className="mb-5 max-w-lg text-neutral-500">
        Apple speech is built into macOS. Other models are downloaded once from the sherpa-onnx GitHub releases when you click Download. Either way, recognition runs on this Mac.
      </p>
      <Section title="Speech recognition">{models.filter((m) => m.kind === "asr").map(row)}</Section>
      <Section title="Punctuation">{models.filter((m) => m.kind === "punct").map(row)}</Section>
    </>
  );
}

function Btn({ primary, onClick, children }: { primary?: boolean; onClick: () => void; children: ReactNode }) {
  return (
    <button
      onClick={onClick}
      className={`rounded-md px-3 py-1 font-medium transition-opacity hover:opacity-85 ${
        primary ? "bg-accent text-white" : "border border-black/10 bg-white dark:border-white/10 dark:bg-neutral-800"
      }`}
    >
      {children}
    </button>
  );
}

function Permissions() {
  const [p, setP] = useState<Perms | null>(null);
  useEffect(() => {
    const load = () => api.permissions().then(setP);
    load();
    window.addEventListener("focus", load); // re-check after returning from System Settings
    return () => window.removeEventListener("focus", load);
  }, []);
  if (!p) return null;
  const item = (kind: "microphone" | "accessibility", title: string, why: string) => {
    const st = p[kind];
    return (
      <Row label={title} hint={why}>
        {st === "granted" ? (
          <span className="font-medium text-emerald-600 dark:text-emerald-400">✓ Allowed</span>
        ) : (
          <Btn primary onClick={async () => {
            await api.requestPermission(kind);
            if (st === "denied") await api.openPermissionSettings(kind);
          }}>
            {st === "undetermined" ? "Allow" : "Open Settings"}
          </Btn>
        )}
      </Row>
    );
  };
  return (
    <>
      <Section title="Required">
        {item("microphone", "Microphone", "Only used while you hold or toggle the shortcut.")}
        {item("accessibility", "Accessibility", "Lets Sayki paste text into the app you're typing in.")}
      </Section>
      <p className="max-w-lg text-xs leading-5 text-neutral-500">
        After enabling Accessibility, macOS may need Sayki to be restarted. Sayki will not type into password fields,
        and never presses Return — terminal commands are never run for you.
      </p>
    </>
  );
}

function History({ s, save }: { s: Settings; save: (p: Partial<Settings>) => void }) {
  const [items, setItems] = useState<HistoryItem[]>([]);
  useEffect(() => { api.history().then(setItems); }, [s.historyEnabled]);
  return (
    <>
      <Section title="History">
        <Row label="Keep transcription history" hint="Stored only on this Mac. Turning it off deletes it.">
          <Toggle on={s.historyEnabled} set={(historyEnabled) => save({ historyEnabled })} />
        </Row>
      </Section>
      {items.length > 0 && (
        <>
          <div className="mb-2 flex items-center justify-between">
            <h2 className="text-xs font-medium uppercase tracking-wide text-neutral-500">Recent</h2>
            <button className="text-xs text-accent" onClick={() => api.clearHistory().then(() => setItems([]))}>Delete all</button>
          </div>
          <ul className="space-y-2">
            {items.map((h) => (
              <li key={h.id} className="select-text rounded-lg border border-black/[0.06] px-3 py-2 dark:border-white/[0.06]">
                <div className="text-[11px] text-neutral-500">{new Date(h.createdAt).toLocaleString()}</div>
                <div className="mt-0.5">{h.text}</div>
              </li>
            ))}
          </ul>
        </>
      )}
    </>
  );
}
