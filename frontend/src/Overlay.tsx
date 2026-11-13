import { useEffect, useRef, useState } from "react";
import { api, on, type Status } from "./api";

// History length; the waveform mirrors it so the newest level sits in the middle.
const HISTORY = 22;

export default function Overlay() {
  const [status, setStatus] = useState<Status>({ state: "idle", language: "" });
  const [text, setText] = useState("");
  const [now, setNow] = useState(Date.now());
  const levels = useRef<number[]>(Array(HISTORY).fill(0));
  const [, redraw] = useState(0);

  useEffect(() => {
    api.getStatus().then(setStatus);
    const offs = [
      on<Status>("sayki:state", (s) => {
        setStatus(s);
        if (s.state === "recording") levels.current = Array(HISTORY).fill(0);
      }),
      on<string>("sayki:transcript", setText),
      on<number>("sayki:level", (rms) => {
        // dB scale: -60 dBFS (room noise) → 0, -15 dBFS (loud speech) → 1
        const v = Math.min(1, Math.max(0, (20 * Math.log10(rms + 1e-9) + 60) / 45));
        levels.current = [...levels.current.slice(1), v];
        redraw((n) => n + 1);
      }),
    ];
    const t = setInterval(() => setNow(Date.now()), 250);
    return () => { offs.forEach((off) => off()); clearInterval(t); };
  }, []);

  const recording = status.state === "recording";
  const secs = recording && status.startedAt ? Math.max(0, Math.floor((now - status.startedAt) / 1000)) : 0;
  const clock = `${Math.floor(secs / 60)}:${String(secs % 60).padStart(2, "0")}`;

  let caption: string = text;
  if (status.state === "loading") caption = "Loading speech model…";
  else if (status.state === "finishing") caption = text || "Transcribing…";
  else if (status.state === "error") caption = status.error ?? "Something went wrong";
  else if (!text && recording) caption = "Listening…";
  const muted = !text || status.state === "loading";

  // Newest level in the centre, older ones spreading outwards on both sides.
  const wave = [...levels.current].reverse().concat(levels.current);

  return (
    // Fixed geometry: app.go makes the top overlayDragH px of this 96px window a native
    // drag area (the waveform row); the caption row with the buttons stays clickable.
    <div className="h-full p-2">
      <div
        className="h-20 w-full animate-[pop_220ms_cubic-bezier(.2,.9,.3,1.2)] rounded-[20px] px-4 py-3 text-white
          shadow-[0_4px_8px_rgba(0,0,0,0.35),inset_0_0_0_1px_rgba(255,255,255,0.08)]"
        style={{ background: "linear-gradient(180deg, rgba(255,255,255,0.07), rgba(255,255,255,0) 45%), rgba(16,16,18,0.94)" }}
      >
        <div className="flex h-7 cursor-grab items-center gap-3 active:cursor-grabbing">
          <Indicator state={status.state} />
          <div className="flex h-6 flex-1 items-center justify-between" aria-hidden>
            {wave.map((v, i) => (
              <span
                key={i}
                className="w-[2.5px] rounded-full transition-[height] duration-100 ease-out"
                style={{
                  height: `${Math.max(14, (recording ? v : 0) * 100)}%`,
                  background: recording ? `hsl(${352 + (Math.abs(i - HISTORY + 0.5) / HISTORY) * -30} 100% 64%)` : "rgba(255,255,255,0.16)",
                }}
              />
            ))}
          </div>
          <span className="w-9 text-right text-xs font-medium tabular-nums text-white/50">
            {recording ? clock : ""}
          </span>
          {status.language && status.language !== "en" && (
            <span className="rounded-full bg-white/10 px-1.5 py-px text-[10px] font-semibold uppercase tracking-wide text-white/60">
              {status.language}
            </span>
          )}
        </div>
        <div className="mt-1.5 flex h-5 items-center gap-2">
          <p
            className={`min-w-0 flex-1 truncate text-[13px] leading-5 [direction:rtl] text-left ${
              status.state === "error" ? "text-amber-300" : muted ? "text-white/40" : "text-white/90"
            }`}
            title={caption}
          >
            {/* rtl + bdi keeps the newest words visible when the line overflows */}
            <bdi>{caption || " "}</bdi>
          </p>
          {recording && (
            <>
              <button onClick={() => api.stop()} title="Stop and insert" aria-label="Stop and insert"
                className="h-5 rounded-full bg-white px-2.5 text-[11px] font-semibold text-neutral-900 transition hover:bg-white/85 active:scale-95">
                Done
              </button>
              <button onClick={() => api.cancel()} title="Cancel" aria-label="Cancel"
                className="grid size-5 place-items-center rounded-full bg-white/10 text-white/60 transition hover:bg-white/20 hover:text-white active:scale-95">
                <svg viewBox="0 0 10 10" className="size-2" aria-hidden>
                  <path d="M1 1l8 8M9 1l-8 8" stroke="currentColor" strokeWidth="1.6" strokeLinecap="round" />
                </svg>
              </button>
            </>
          )}
        </div>
      </div>
      <style>{`@keyframes pop{from{opacity:0;transform:translateY(8px) scale(.96)}to{opacity:1;transform:none}}`}</style>
    </div>
  );
}

function Indicator({ state }: { state: Status["state"] }) {
  if (state === "error")
    return <span className="grid size-5 place-items-center rounded-full bg-amber-400/15 text-[11px] font-bold text-amber-300">!</span>;
  if (state === "loading" || state === "finishing")
    return <span className="size-5 animate-spin rounded-full border-2 border-white/15 border-t-white/80" />;
  return (
    <span className="relative grid size-5 place-items-center">
      <span className="absolute inset-0 animate-ping rounded-full bg-accent/25 [animation-duration:1.6s]" />
      <span className="size-2.5 rounded-full bg-accent shadow-[0_0_10px_2px_rgba(255,90,79,0.55)]" />
    </span>
  );
}
