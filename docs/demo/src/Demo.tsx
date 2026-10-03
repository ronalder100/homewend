// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// The terminal animation at the top of the README. Every line is what
// homewend prints (cmd/homewend/strings.go), drawn as cmd/homewend/look.go
// draws it, in the order it prints it; the line going on at the bottom is the
// one cmd/homewend/live.go redraws. The sizes and counts are those of one
// ordinary year, and the hours Google takes are fast-forwarded: a real run is
// recorded once, not on every change.

import { loadFont } from "@remotion/google-fonts/SourceCodePro";
import { loadFont as loadSans } from "@remotion/google-fonts/Roboto";
import type { ReactNode } from "react";
import { AbsoluteFill, Easing, interpolate, useCurrentFrame } from "remotion";

const { fontFamily } = loadFont("normal", { weights: ["400", "700"], subsets: ["latin"] });
const { fontFamily: sansFamily } = loadSans("normal", { weights: ["400", "700"], subsets: ["latin"] });

export const fps = 30;
export const width = 1100;
export const height = 620;
const fontSize = 22;
const lineHeight = 1.45;
const rows = 16;

const sec = (s: number) => Math.round(s * fps);

// Characters a second, as a person types.
const typingSpeed = 60;

// cmd/homewend/palette.go: Tokyo Night, its night variant.
const color = {
  background: "#1A1B26",
  text: "#C0CAF5",
  title: "#737AA2",
  muted: "#A9B1D6",
  faint: "#565F89",
  line: "#292E42",
  ok: "#9ECE6A",
  err: "#F7768E",
  warn: "#E0AF68",
  accent: "#7AA2F7",
};

// A part of a line: its text, its colour, bold or not.
type Seg = [string, string?, boolean?];

// A line of the screen, from a frame on: printed, or typed at the prompt.
type Line = { at: number; segs: Seg[]; typed?: string };

// The line going on at the bottom, from one frame to another.
type Live = { from: number; to: number; view: (progress: number, frame: number) => ReactNode };

const log: Line[] = [];
const live: Live[] = [];
// Frames at which a command clears the screen: what came before is gone.
const clears: number[] = [];

// The browser window login opens, from one frame to another.
const browser = { from: 0, to: 0 };

// The wait for Google, which the camera moves in on: it is the moment a
// first-time user must not miss.
const waiting = { from: 0, to: 0 };
let t = 0.4;

const type = (text: string) => {
  log.push({ at: sec(t), segs: [], typed: text });
  t += text.length / typingSpeed + 0.8;
};
const clear = () => clears.push(sec(t));
const print = (...segs: Seg[]) => log.push({ at: sec(t), segs });
const blank = () => print([" "]);
const wait = (s: number) => {
  t += s;
};
const show = (s: number, view: Live["view"]) => {
  live.push({ from: sec(t), to: sec(t + s), view });
  t += s;
};

const B = true;
const header = (cmd: string) => {
  clear();
  print([cmd, color.title, B]);
  blank();
};
const done = (...segs: Seg[]) => print(["✓ ", color.ok], ...segs);
const param = (name: string, value: string, details = "") =>
  print(["  "], [name.padEnd(10), color.text, B], [value], [details, color.muted]);

type("homewend login");
header("homewend login");
show(0.6, (_, f) => going(f, [["Waiting for you to sign in to Google in the new window"]]));
browser.from = sec(t);
show(6, (_, f) => going(f, [["Waiting for you to sign in to Google in the new window"]]));
browser.to = sec(t);
show(1.2, (_, f) => going(f, [["Setting up your Google account"]]));
done(["Signed in as "], ["you@gmail.com", color.text, B]);
blank();
print(["hint:", color.accent, B], [" to bring your 2025 photos home, run: "], ["homewend takeout 2025", color.text, B]);
wait(1.4);
type("homewend takeout 2025");
header("homewend takeout 2025");
show(0.8, (_, f) => going(f, [["Checking your Google sign-in"]]));
show(1.2, (_, f) => going(f, [["Looking for an export of your "], ["2025", color.text, B], [" photos on Google Takeout"]]));
// The question, answered with the arrows: Yes is under the cursor.
show(2.4, () => question("Ask Google to export your 2025 photos?", "Google takes hours to prepare the export. --yes skips this question.", ["Yes", "No"], 0));
param("Google", "you@gmail.com");
param("Takeout", "2025 photos");
param("Library", "~/Pictures/Homewend");
blank();
show(1.6, (p, f) => going(f, [["Asking Google for an export"]], ` · ${goDuration(1 + 70 * p)}`));
done(["Asked Google for an export"], [" at 10:12", color.muted]);
waiting.from = sec(t);
show(4, (_, f) => (
  <>
    <Breath frame={f} />
    {" Google is preparing the export"}
    {"\n"}
    <span style={{ color: color.muted }}>{"  This takes hours. Ctrl-C is safe: run the same command to resume."}</span>
  </>
));
waiting.to = sec(t);
done(["Found export 4184685c"], [" · 2 parts · 2.0 GiB", color.muted]);
show(3.8, (p, f) => going(f, [["Downloading part 1 of 2  "]], "", skipped(p), `  ${ibytes(2 * GiB * skipped(p))} of 2.0 GiB · ${goDuration((2 * GiB * (1 - skipped(p))) / (41 * MiB))} left`));
done(["Downloaded part "], ["1 of 2", color.text, B], [" · 2.0 GiB in 52s", color.muted]);
show(0.6, (_, f) => going(f, [["Downloading part 2 of 2  "]], "", 1, "  61 KiB of 61 KiB"));
done(["Downloaded part "], ["2 of 2", color.text, B], [" · 61 KiB in 0s", color.muted]);
show(3, (p, f) => going(f, [["Sorting photos  "]], "", p, `  ${Math.round(742 * p)} of 742`));
done(["Sorted "], ["742 photos", color.text, B], [" in 9s · 2 duplicates · 0 undated · 87 in albums", color.muted]);
done(["Checked "], ["742 of 742", color.text, B], [" against the export's list", color.muted]);
blank();
print(["✓ ", color.ok, B], ["All 742 photos are there", color.text, B]);
wait(1.5);
const end = sec(t);
export const duration = sec(t + 3);

const GiB = 1024 ** 3;
const MiB = 1024 ** 2;

// A cut in the download, from half to three quarters: the bar moves at its
// real pace, and the film skips a quarter of it.
function skipped(p: number) {
  const x = p * 0.75;
  return x < 0.5 ? x : x + 0.25;
}

// Go's Duration.String, rounded to the second, as live.go prints it.
function goDuration(seconds: number) {
  const s = Math.round(seconds);
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const r = s % 60;
  if (h > 0) return `${h}h${m}m${r}s`;
  if (m > 0) return `${m}m${r}s`;
  return `${r}s`;
}

// go-humanize's IBytes.
function ibytes(n: number) {
  if (n < 1024) return `${n} B`;
  const units = ["KiB", "MiB", "GiB", "TiB"];
  let value = n / 1024;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit++;
  }
  return `${value < 10 ? value.toFixed(1) : value.toFixed(0)} ${units[unit]}`;
}

// bubbles' MiniDot, the spinner live.go turns ten times a second.
const spinnerFrames = ["⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"];

// The dot of a long wait, breathing from all but off to the accent and back,
// as live.go draws it: no time, which over hours reads as a hang.
const Breath = ({ frame }: { frame: number }) => {
  const k = (1 - Math.cos((frame / fps / 2.5) * 2 * Math.PI)) / 2;
  const [a, b] = [[0x29, 0x2e, 0x42], [0x7a, 0xa2, 0xf7]];
  const [r, g, bl] = a.map((c, i) => Math.round(c + (b[i] - c) * k));
  return <span style={{ color: `rgb(${r},${g},${bl})` }}>●</span>;
};

const segs = (list: Seg[]) => list.map(([text, c, b], i) => (
  <span key={i} style={{ color: c ?? color.text, fontWeight: b ? 700 : 400 }}>{text}</span>
));

// The line going on: the spinner, what is said, a bar of 20 cells when there
// is something to count, and details.
function going(frame: number, said: Seg[], details = "", bar?: number, after = "") {
  const cells = 20;
  const full = bar === undefined ? 0 : Math.round(cells * bar);
  return (
    <>
      <span style={{ color: color.accent }}>{spinnerFrames[Math.floor(frame / (fps / 10)) % spinnerFrames.length]}</span>{" "}
      {segs(said)}
      {bar === undefined ? null : (
        <>
          <span style={{ color: color.accent }}>{"━".repeat(full)}</span>
          <span style={{ color: color.line }}>{"━".repeat(cells - full)}</span>
        </>
      )}
      <span style={{ color: color.muted }}>{details + after}</span>
    </>
  );
}

// A question in the flow: ? and the question bold, its detail, the choices
// with › on the one under the cursor, and the keys.
function question(said: string, detail: string, choices: string[], at: number) {
  return (
    <>
      <span style={{ color: color.accent, fontWeight: 700 }}>?</span> <b>{said}</b>
      {"\n"}
      <span style={{ color: color.muted }}>{"  " + detail}</span>
      {"\n\n"}
      {choices.map((c, i) => (
        <span key={c}>
          {i === at ? (
            <>
              {"  "}
              <span style={{ color: color.accent, fontWeight: 700 }}>›</span> <b>{c}</b>
            </>
          ) : (
            <span style={{ color: color.muted }}>{"    " + c}</span>
          )}
          {"\n"}
        </span>
      ))}
      {"\n  "}
      <span style={{ color: color.muted }}>↑/↓</span>
      <span style={{ color: color.faint }}> move · </span>
      <span style={{ color: color.muted }}>enter</span>
      <span style={{ color: color.faint }}> select · </span>
      <span style={{ color: color.muted }}>ctrl+c</span>
      <span style={{ color: color.faint }}> quit</span>
    </>
  );
}

const Prompt = ({ children, cursor }: { children: ReactNode; cursor?: boolean }) => (
  <>
    <span style={{ color: color.faint }}>{"$ "}</span>
    {children}
    {cursor ? <span style={{ background: color.text }}>{" "}</span> : null}
  </>
);

// The screen's lines at a frame, the line going on last.
function linesAt(frame: number) {
  const since = Math.max(0, ...clears.filter((c) => c <= frame));
  const lines: ReactNode[] = [];
  log.forEach((line) => {
    if (frame < line.at || line.at < since) return;
    if (line.typed !== undefined) {
      const shown = Math.floor(((frame - line.at) / fps) * typingSpeed);
      lines.push(<Prompt cursor={shown < line.typed.length}>{line.typed.slice(0, shown)}</Prompt>);
      return;
    }
    lines.push(<>{segs(line.segs)}</>);
  });
  const now = live.find((s) => frame >= s.from && frame < s.to);
  if (now) {
    lines.push(now.view(interpolate(frame, [now.from, now.to - 1], [0, 1], { extrapolateRight: "clamp" }), frame));
  }
  if (frame >= end) {
    lines.push(<Prompt cursor={Math.floor(frame / (fps / 2)) % 2 === 0}>{""}</Prompt>);
  }
  return lines.slice(-rows);
}

// Where the text sits in the frame.
const margin = 16;
const textLeft = margin + 28;
const textTop = margin + 20 + 13 + 14;
const linePx = fontSize * lineHeight;
const charPx = fontSize * 0.6;

// The camera moves in on the wait for Google, centred and close, and back
// out, each in a fifth of a second.
function camera(frame: number, lines: number) {
  const cut = sec(0.2);
  const k = Math.min(
    interpolate(frame, [waiting.from, waiting.from + cut], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp" }),
    interpolate(frame, [waiting.to - cut, waiting.to], [1, 0], { extrapolateLeft: "clamp", extrapolateRight: "clamp" }),
  );
  if (k === 0) return {};
  const ease = Easing.inOut(Easing.cubic)(k);
  const fx = textLeft + 34 * charPx;
  const fy = textTop + (lines - 0.5) * linePx;
  return {
    transformOrigin: `${fx - margin}px ${fy - margin}px`,
    transform: `translate(${(width / 2 - fx) * ease}px, ${(height / 2 - fy) * ease}px) scale(${1 + 0.15 * ease})`,
  };
}

export const Demo = () => {
  const frame = useCurrentFrame();
  const visible = linesAt(frame);
  return (
    <AbsoluteFill style={{ background: "#0d0d0d", padding: margin, overflow: "hidden" }}>
      <div
        style={{
          ...camera(frame, visible.length),
          flex: 1,
          background: color.background,
          borderRadius: 12,
          border: "1px solid #2a2a2a",
          padding: "20px 28px",
          fontFamily,
          fontSize,
          lineHeight,
          color: color.text,
          whiteSpace: "pre",
          overflow: "hidden",
        }}
      >
        <div style={{ display: "flex", gap: 8, marginBottom: 14 }}>
          {["#ff5f57", "#febc2e", "#28c840"].map((c) => (
            <div key={c} style={{ width: 13, height: 13, borderRadius: "50%", background: c }} />
          ))}
        </div>
        {visible.map((line, i) => (
          <div key={i}>{line}</div>
        ))}
      </div>
      <Browser frame={frame} />
    </AbsoluteFill>
  );
};

// The sign-in window of homewend login, over the terminal: a small popup, the
// address read-only, Google's page and then Homewend's own (internal/signin),
// which closes itself.
const logo =
  "M69.58 128.26895c-12.845-1.99499-20.72-4.65501-20.72-11.09502 0-6.36999 8.365-9.86999 21.45502-14.55999 34.64998-12.215 50.95999-22.4 50.95999-46.305l0-5.11 24.64 0c2.55499-0.07 4.655-2.13499 4.54999-4.655l-0.14-6.825c-0.07-1.61001-0.83998-3.08-2.13498-4.025l-44.41501-34.755c-1.785-1.33-4.30501-1.225-6.02001 0.175l-44.13499 35.385c-1.40001 1.085-2.03001 2.765-1.96001 4.515l0.07001 6.405c0 2.59 2.06499 4.41 4.585 4.34001l20.50999-0.21001 0 5.355c0 10.39499-7.62999 14.21-25.19999 20.57999-23.31001 8.57501-45.88501 18.795-45.88501 42.38501 0 27.545 28.735 34.79 56.84 40.04 23.80001 4.41001 41.19501 7.525 41.19501 16.87001 0 13.93-34.61501 16.69499-80.53501 20.615-13.65 1.22499-23.24 10.98998-23.24 22.53999 0 12.00501 10.57 20.965 23.275 20.79001 34.26501-1.19 131.355-11.515 131.355-62.16 0-41.51001-51.69499-44.76502-85.05-50.295z";

// The site's light theme, as the sign-in page takes it (internal/signin).
const site = { bg: "#FFFFFF", fg: "#0A0A0A", muted: "#6B6B6B", accent: "#2F5BFF", ok: "#1E9E5A" };

const Browser = ({ frame }: { frame: number }) => {
  if (frame < browser.from || frame >= browser.to) return null;
  const p = (frame - browser.from) / (browser.to - browser.from);
  const at = (from: number, to: number) =>
    interpolate(p, [from, to], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp", easing: Easing.out(Easing.cubic) });
  const shown = at(0, 0.08) * (1 - at(0.93, 1));
  const email = "you@gmail.com";
  const typed = email.slice(0, Math.round(email.length * at(0.12, 0.38)));
  const pressed = p > 0.44 && p < 0.5;
  const ours = p >= 0.52;
  const drawn = at(0.52, 0.66);
  const words = at(0.62, 0.72);
  return (
    <div
      style={{
        position: "absolute",
        left: "50%",
        top: "50%",
        width: 420,
        height: 440,
        transform: `translate(-50%, -50%) scale(${0.9 + 0.1 * shown})`,
        opacity: shown,
        borderRadius: 10,
        overflow: "hidden",
        background: ours ? site.bg : "#ffffff",
        boxShadow: "0 30px 80px rgba(0,0,0,0.6)",
        fontFamily: sansFamily,
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 7, padding: "9px 12px", background: "#e8eaed" }}>
        {["#ff5f57", "#febc2e", "#28c840"].map((c) => (
          <div key={c} style={{ width: 11, height: 11, borderRadius: "50%", background: c }} />
        ))}
        <div
          style={{
            flex: 1,
            marginLeft: 10,
            padding: "4px 12px",
            borderRadius: 12,
            background: "#ffffff",
            color: "#3c4043",
            fontSize: 13,
          }}
        >
          {ours ? "127.0.0.1" : "accounts.google.com"}
        </div>
      </div>
      {ours ? (
        <div style={{ height: 400, display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", gap: 26 }}>
          <div style={{ display: "flex", alignItems: "center", gap: 8, color: site.fg, fontSize: 18, fontWeight: 700 }}>
            <svg viewBox="-44 -1 243 243" style={{ width: 15, height: 24 }}>
              <path d={logo} fill={site.accent} />
            </svg>
            Homewend
          </div>
          <svg viewBox="0 0 96 96" style={{ width: 84, height: 84 }}>
            <circle cx="48" cy="48" r="42" fill="none" stroke={site.ok} strokeWidth={5} strokeDasharray={264} strokeDashoffset={264 * (1 - drawn)} />
            <path d="M30 49 l12 12 l24 -26" fill="none" stroke={site.ok} strokeWidth={5} strokeLinecap="round" strokeLinejoin="round" strokeDasharray={80} strokeDashoffset={80 * (1 - at(0.6, 0.66))} />
          </svg>
          <div style={{ textAlign: "center", opacity: words, transform: `translateY(${8 * (1 - words)}px)` }}>
            <div style={{ color: site.fg, fontSize: 28, fontWeight: 700, marginBottom: 6 }}>Signed in.</div>
            <div style={{ color: site.muted, fontSize: 15 }}>You're all set. This window closes by itself.</div>
          </div>
        </div>
      ) : (
        <div style={{ padding: "34px 36px", color: "#202124" }}>
          <div style={{ fontSize: 26 }}>Sign in</div>
          <div style={{ fontSize: 15, marginTop: 8, color: "#5f6368" }}>to continue to Homewend</div>
          <div
            style={{
              marginTop: 32,
              display: "flex",
              alignItems: "center",
              boxSizing: "border-box",
              height: 52,
              padding: "0 14px",
              border: `2px solid ${typed ? "#1a73e8" : "#dadce0"}`,
              borderRadius: 6,
              fontSize: 17,
            }}
          >
            {typed || <span style={{ color: "#80868b" }}>Email or phone</span>}
          </div>
          <div style={{ display: "flex", justifyContent: "flex-end", marginTop: 30 }}>
            <div
              style={{
                padding: "10px 24px",
                borderRadius: 20,
                background: pressed ? "#1557b0" : "#1a73e8",
                color: "#ffffff",
                fontSize: 15,
              }}
            >
              Next
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
