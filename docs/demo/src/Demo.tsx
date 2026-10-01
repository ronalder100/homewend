// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// The terminal animation at the top of the README. Every line is what
// homewend prints (cmd/homewend/strings.go), in the order it prints it; the
// status line under them is the one cmd/homewend/live.go draws. The sizes and
// counts are those of one ordinary year, and the hours Google takes are
// fast-forwarded: a real run is recorded once, not on every change.

import { loadFont } from "@remotion/google-fonts/JetBrainsMono";
import { loadFont as loadSans } from "@remotion/google-fonts/Roboto";
import type { ReactNode } from "react";
import { AbsoluteFill, Easing, interpolate, useCurrentFrame } from "remotion";

const { fontFamily } = loadFont("normal", { weights: ["400"], subsets: ["latin"] });
const { fontFamily: sansFamily } = loadSans("normal", { weights: ["400", "700"], subsets: ["latin"] });

export const fps = 30;
export const width = 1100;
export const height = 560;
const fontSize = 22;
const lineHeight = 1.45;
const rows = 14;
// Columns of the terminal: a longer line wraps, at a space, as the CLI's
// sentences are meant to be read.
const cols = 76;

const sec = (s: number) => Math.round(s * fps);

// Characters a second, as a person types.
const typingSpeed = 60;

const color = {
  background: "#171717",
  text: "#dddddd",
  dim: "#8a8a8a",
  prompt: "#7571F9",
  // The CLI's own: done in green, what to type in blue.
  ok: "#9ECE6A",
  ask: "#7AA2F7",
  // homewend.app's dark-theme --faint and --accent: the notice runs from one to the other.
  noticeFrom: [0x56, 0x5f, 0x89],
  noticeTo: [0x7a, 0xa2, 0xf7],
  empty: "#606060",
  blendStart: [0x5a, 0x56, 0xe0],
  blendEnd: [0xee, 0x6f, 0xf8],
};

const GiB = 1024 ** 3;
const MiB = 1024 ** 2;

// A line of the log: typed at a prompt, or printed.
type Line = {
  at: number;
  text: string;
  typed?: boolean;
  notice?: boolean;
  color?: string;
  // An answer typed after the line, from a later frame.
  answer?: { text: string; at: number };
};

// The status line under the log, from one frame to another.
type Status = { from: number; to: number; view: (progress: number, frame: number) => ReactNode };


const log: Line[] = [];
const status: Status[] = [];

// The browser window login opens, from one frame to another.
const browser = { from: 0, to: 0 };

// The wait for Google, which the camera moves in on: it is the moment a
// first-time user must not miss.
const waiting = { from: 0, to: 0 };
let t = 0.4;

const type = (text: string) => {
  log.push({ at: sec(t), text, typed: true });
  t += text.length / typingSpeed + 0.8;
};
// wrap breaks a line at the last space before the terminal's edge.
function wrap(text: string): string[] {
  const rows: string[] = [];
  let rest = text;
  while (rest.length > cols) {
    const cut = rest.lastIndexOf(" ", cols);
    rows.push(rest.slice(0, cut));
    rest = rest.slice(cut + 1);
  }
  return [...rows, rest];
}
const print = (...lines: string[]) => {
  for (const line of lines) for (const text of wrap(line)) log.push({ at: sec(t), text });
};
const colored = (c: string, text: string) => {
  log.push({ at: sec(t), text, color: c });
};
// The one line a person must not skim past, in the colour the CLI gives it.
const notice = (text: string) => {
  log.push({ at: sec(t), text, notice: true });
};
// A question, and the answer typed after a moment's thought.
const ask = (question: string, answer: string) => {
  const at = sec(t + 0.9);
  log.push({ at: sec(t), text: question, color: color.ask, answer: { text: answer, at } });
  t += 0.9 + answer.length / typingSpeed + 0.5;
};
const wait = (s: number) => {
  t += s;
};
const show = (s: number, view: Status["view"]) => {
  status.push({ from: sec(t), to: sec(t + s), view });
  t += s;
};

type("homewend login");
print("a small browser window is open: sign in to Google there");
wait(0.6);
browser.from = sec(t);
wait(6);
browser.to = sec(t);
print("signed in to Google");
show(2, (p, f) => spinning(f, "getting Takeout ready, about half a minute", 2 + 26 * p));
colored(color.ok, "signed in");
wait(1);
print("");
type("homewend get --year 2025 --library ~/Pictures/Homewend");
print(
  "Homewend is about to ask Google Takeout for an export of your photos of 2025.",
  "",
  "Google takes its time to prepare it, often hours. You do not have to wait here: close this window whenever you like, and run the same command again later. It picks up where it left off, and never asks Google twice.",
  "",
);
wait(2.5);
ask("Continue? [y/N] ", "y");
print("asking Google Takeout for an export");
show(2, (p, f) => spinning(f, "asking Google for the export, a minute or two", 70 * p));
print(
  "Google is preparing the export: this can take hours.",
  "Leave this open and the download starts when it is ready, or close it and run the same command later.",
);
notice("If the computer restarts, run the same command again.");
wait(1.2);
waiting.from = sec(t);
show(4, (_, f) => <Pulse frame={f}>waiting for Google, it can take a few hours</Pulse>);
waiting.to = sec(t);
print("", "the export is ready: downloading 2 parts, 2.0 GiB");
wait(0.8);

// One part and the manifest: the smallest real export, so the run is not a
// list of the same three lines.
const parts = [2 * GiB];
const seconds = [3.8];
parts.forEach((total, i) => {
  show(seconds[i], (p) => downloading(i + 1, parts.length + 1, total * skipped(p), total, 41 * MiB));
  print(`[${i + 1}/${parts.length + 1}] downloaded, ${ibytes(total)}`);
  print(`[${i + 1}/${parts.length + 1}] unpacking`);
});
wait(0.6);
print("[2/2] downloaded, 61 KiB");
show(3, (p) => placing(Math.round(742 * p), 742));
print("placed 742 photos, 2.0 GiB: 2 duplicates, 0 undated, 87 in albums");
wait(0.8);
print("declared 742, on disk 742, missing 0", "  2025  742 of 742", "");
colored(color.ok, "download complete, congratulations 🎉");
wait(1.5);
const end = sec(t);
export const duration = sec(t + 3);

// A cut in the download, from half to three quarters: the bar moves at its
// real pace, and the film skips a quarter of it.
function skipped(p: number) {
  const x = p * 0.75;
  return x < 0.5 ? x : x + 0.25;
}

// The notice, letter by letter from grey to blue, as the CLI draws it.
function blended(text: string) {
  const letters = [...text];
  return letters.map((letter, i) => {
    const k = letters.length > 1 ? i / (letters.length - 1) : 0;
    const [r, g, b] = color.noticeFrom.map((c, j) => Math.round(c + (color.noticeTo[j] - c) * k));
    return (
      <span key={i} style={{ color: `rgb(${r},${g},${b})` }}>
        {letter}
      </span>
    );
  });
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

const spinnerFrames = ["⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"];

function spinning(frame: number, what: string, elapsed: number) {
  return (
    <>
      <span style={{ color: color.prompt }}>{spinnerFrames[Math.floor(frame / (fps / 10)) % spinnerFrames.length]}</span>
      {` ${what} · ${goDuration(elapsed)}`}
    </>
  );
}

// A slow fade in and out, so a screen where nothing moves still reads as alive.
const Pulse = ({ frame, children }: { frame: number; children: ReactNode }) => (
  <span style={{ opacity: 0.6 + 0.4 * Math.cos((frame / fps) * Math.PI * 1.2) }}>{children}</span>
);

// The bubbles progress bar: 13 cells blended purple to pink, then the percent.
function bar(fraction: number) {
  const cells = 13;
  const full = Math.round(cells * fraction);
  const blend = (i: number) => {
    const k = cells > 1 ? i / (cells - 1) : 0;
    const [r, g, b] = color.blendStart.map((c, j) => Math.round(c + (color.blendEnd[j] - c) * k));
    return `rgb(${r},${g},${b})`;
  };
  return (
    <>
      {Array.from({ length: cells }, (_, i) => (
        // Drawn, not typed: the font has no block characters, and a
        // fallback font's are wider than a column.
        <span
          key={i}
          style={{
            display: "inline-block",
            width: "1ch",
            height: "0.8em",
            verticalAlign: "-0.05em",
            background: i < full ? blend(i) : color.empty,
            opacity: i < full ? 1 : 0.35,
          }}
        />
      ))}
      {` ${String(Math.round(fraction * 100)).padStart(3)}%`}
    </>
  );
}

function downloading(n: number, of: number, done: number, total: number, rate: number) {
  return (
  <>
    {`[${n}/${of}] `}
    {bar(done / total)}
    {` · ${ibytes(done)} of ${ibytes(total)} · ${ibytes(rate)}/s · ${goDuration((total - done) / rate)} left`}
  </>
  );
}

function placing(n: number, of: number) {
  return (
  <>
    {bar(n / of)}
    {`  placing ${n} of ${of}`}
  </>
  );
}

const Prompt = ({ children, cursor }: { children: ReactNode; cursor?: boolean }) => (
  <>
    <span style={{ color: color.prompt }}>{"> "}</span>
    {children}
    {cursor ? <span style={{ background: color.text }}>{" "}</span> : null}
  </>
);

// The terminal's lines at a frame, the one being written last.
function linesAt(frame: number) {
  const lines: ReactNode[] = [];
  log.forEach((line) => {
    if (frame < line.at) return;
    if (!line.typed) {
      // A blank line still takes its height.
      if (line.notice) lines.push(blended(line.text));
      else if (line.color) {
        const typed = line.answer && frame >= line.answer.at ? line.answer.text.slice(0, Math.floor(((frame - line.answer.at) / fps) * typingSpeed) + 1) : "";
        lines.push(
          <>
            <span style={{ color: line.color }}>{line.text}</span>
            {typed}
          </>,
        );
      } else lines.push(line.text || " ");
      return;
    }
    const shown = Math.floor(((frame - line.at) / fps) * typingSpeed);
    lines.push(<Prompt cursor={shown < line.text.length}>{line.text.slice(0, shown)}</Prompt>);
  });
  const live = status.find((s) => frame >= s.from && frame < s.to);
  if (live) {
    lines.push(live.view(interpolate(frame, [live.from, live.to - 1], [0, 1], { extrapolateRight: "clamp" }), frame));
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

// The camera moves in on the waiting line, centred and close, and back out,
// each in a fifth of a second.
function camera(frame: number, lines: number) {
  const cut = sec(0.2);
  const k = Math.min(
    interpolate(frame, [waiting.from, waiting.from + cut], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp" }),
    interpolate(frame, [waiting.to - cut, waiting.to], [1, 0], { extrapolateLeft: "clamp", extrapolateRight: "clamp" }),
  );
  if (k === 0) return {};
  const ease = Easing.inOut(Easing.cubic)(k);
  const fx = textLeft + 21 * charPx;
  const fy = textTop + (lines - 0.5) * linePx;
  return {
    transformOrigin: `${fx - margin}px ${fy - margin}px`,
    transform: `translate(${(width / 2 - fx) * ease}px, ${(height / 2 - fy) * ease}px) scale(${1 + 0.7 * ease})`,
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

// The site's dark theme, as the sign-in page takes it.
const site = { bg: "#1A1B26", fg: "#C0CAF5", muted: "#9AA5CE", accent: "#7AA2F7", ok: "#9ECE6A" };

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
      <div style={{ display: "flex", alignItems: "center", gap: 7, padding: "9px 12px", background: ours ? "#16161e" : "#e8eaed" }}>
        {["#ff5f57", "#febc2e", "#28c840"].map((c) => (
          <div key={c} style={{ width: 11, height: 11, borderRadius: "50%", background: c }} />
        ))}
        <div
          style={{
            flex: 1,
            marginLeft: 10,
            padding: "4px 12px",
            borderRadius: 12,
            background: ours ? "#24283b" : "#ffffff",
            color: ours ? site.muted : "#3c4043",
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
