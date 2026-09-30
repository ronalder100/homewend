// homewend — Copyright (C) 2026 Ron Alder
// SPDX-License-Identifier: AGPL-3.0-or-later

// The terminal animation at the top of the README. Every line is what
// homewend prints (cmd/homewend/strings.go), in the order it prints it; the
// status line under them is the one cmd/homewend/live.go draws. The sizes and
// counts are those of one ordinary year, and the hours Google takes are
// fast-forwarded: a real run is recorded once, not on every change.

import { loadFont } from "@remotion/google-fonts/JetBrainsMono";
import type { ReactNode } from "react";
import { AbsoluteFill, Easing, interpolate, useCurrentFrame } from "remotion";

const { fontFamily } = loadFont("normal", { weights: ["400"], subsets: ["latin"] });

export const fps = 30;
export const width = 1000;
export const height = 560;
const fontSize = 23;
const lineHeight = 1.45;
const rows = 13;

const sec = (s: number) => Math.round(s * fps);

const color = {
  background: "#171717",
  text: "#dddddd",
  dim: "#8a8a8a",
  prompt: "#7571F9",
  empty: "#606060",
  blendStart: [0x5a, 0x56, 0xe0],
  blendEnd: [0xee, 0x6f, 0xf8],
};

const GiB = 1024 ** 3;
const MiB = 1024 ** 2;

// A line of the log: typed at a prompt, or printed.
type Line = { at: number; text: string; typed?: boolean };

// The status line under the log, from one frame to another.
type Status = { from: number; to: number; view: (progress: number, frame: number) => ReactNode };

// How close the camera is, from a moment on. It follows the line being
// written, and eases from one closeness to the next.
type Shot = { at: number; scale: number };

const log: Line[] = [];
const status: Status[] = [];
const shots: Shot[] = [];

// The browser window login opens, from one frame to another.
const browser = { from: 0, to: 0 };
let t = 0.4;

const type = (text: string) => {
  log.push({ at: sec(t), text, typed: true });
  t += text.length / 26 + 0.5;
};
const print = (...lines: string[]) => {
  for (const text of lines) log.push({ at: sec(t), text });
};
const wait = (s: number) => {
  t += s;
};
const zoom = (scale: number) => {
  shots.push({ at: sec(t), scale });
};
const show = (s: number, view: Status["view"]) => {
  status.push({ from: sec(t), to: sec(t + s), view });
  t += s;
};

zoom(1.7);
type("homewend login");
print("a browser window is open: sign in to Google there");
wait(0.4);
browser.from = sec(t);
wait(3.2);
browser.to = sec(t);
print("signed in");
wait(0.5);
zoom(1.25);
type("homewend get --year 2025 --library ~/Pictures/Homewend");
print("asking Google Takeout for an export");
show(1.4, (p, f) => spinning(f, "asking Google for the export", 70 * p));
print(
  "Google is preparing the export: this can take hours.",
  "Leave this open, the download starts when it is ready.",
);
zoom(1.8);
show(3.5, (p, f) => spinning(f, "waiting for Google", 47 * 60 * p));
zoom(1);

const parts = [2 * GiB, 2 * GiB, 1.1 * GiB];
const seconds = [3, 2.2, 1.8];
parts.forEach((total, i) => {
  show(seconds[i], (p) => downloading(i + 1, parts.length + 1, total * p, total, 41 * MiB));
  print(`[${i + 1}/${parts.length + 1}] downloaded, ${ibytes(total)}`);
  print(`[${i + 1}/${parts.length + 1}] unpacking`);
});
wait(0.3);
print("[4/4] downloaded, 61 KiB");
show(2, (p) => placing(Math.round(1812 * p), 1812));
print("placed 1812 photos, 5.1 GiB: 4 duplicates, 0 undated, 212 in albums");
zoom(1.3);
wait(0.4);
print("declared 1812, on disk 1812, missing 0", "  2025  1812 of 1812");
wait(1.6);
zoom(1);
const end = sec(t);
export const duration = sec(t + 3);

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
      lines.push(line.text);
      return;
    }
    const shown = Math.floor(((frame - line.at) / fps) * 26);
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

const transition = sec(0.9);

function scaleAt(frame: number) {
  let scale = shots[0].scale;
  for (const [i, shot] of shots.entries()) {
    if (i === 0 || frame < shot.at) continue;
    scale = interpolate(frame, [shot.at, shot.at + transition], [shots[i - 1].scale, shot.scale], {
      easing: Easing.inOut(Easing.cubic),
      extrapolateRight: "clamp",
    });
  }
  return scale;
}

// Where the text starts inside the terminal window, and how tall a line is.
const margin = 16;
const textLeft = 28;
const textTop = 20 + 13 + 14;
const linePx = fontSize * lineHeight;

// The camera: scaled about the text's left edge, and moved so the line being
// written sits a little below the middle, never past the window's edges.
function camera(frame: number) {
  const scale = scaleAt(frame);
  // The last line jumps as lines arrive; half a second of it, averaged, glides.
  const recent = Array.from({ length: sec(0.5) }, (_, k) => linesAt(Math.max(0, frame - k)).length - 1);
  const active = recent.reduce((a, b) => a + b, 0) / recent.length;
  const y = textTop + (active + 0.5) * linePx;
  const closeness = Math.min(1, (scale - 1) / 0.3);
  const target = margin + y + (height * 0.58 - margin - y) * closeness;
  const inner = height - 2 * margin;
  const ty = Math.min(0, Math.max(inner - inner * scale, target - margin - y * scale));
  const tx = textLeft - textLeft * scale;
  return `translate(${tx}px, ${ty}px) scale(${scale})`;
}

export const Demo = () => {
  const frame = useCurrentFrame();
  const visible = linesAt(frame);
  return (
    <AbsoluteFill style={{ background: "#0d0d0d", padding: 16, overflow: "hidden" }}>
      <div
        style={{
          flex: 1,
          transform: camera(frame),
          transformOrigin: "0 0",
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

// The browser window of homewend login, over the terminal: it opens, the
// person signs in, it closes by itself.
const Browser = ({ frame }: { frame: number }) => {
  if (frame < browser.from || frame >= browser.to) return null;
  const p = (frame - browser.from) / (browser.to - browser.from);
  const at = (from: number, to: number) =>
    interpolate(p, [from, to], [0, 1], { extrapolateLeft: "clamp", extrapolateRight: "clamp", easing: Easing.out(Easing.cubic) });
  const shown = at(0, 0.1) * (1 - at(0.92, 1));
  const email = "you@gmail.com";
  const typed = email.slice(0, Math.round(email.length * at(0.18, 0.5)));
  const pressed = p > 0.6 && p < 0.66;
  const done = p >= 0.7;
  return (
    <div
      style={{
        position: "absolute",
        left: "50%",
        top: "52%",
        width: 560,
        height: 380,
        transform: `translate(-50%, -50%) scale(${0.85 + 0.15 * shown})`,
        opacity: shown,
        borderRadius: 12,
        overflow: "hidden",
        background: "#ffffff",
        boxShadow: "0 30px 80px rgba(0,0,0,0.6)",
        fontFamily: "Helvetica, Arial, sans-serif",
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: 8, padding: "10px 14px", background: "#e8eaed" }}>
        {["#ff5f57", "#febc2e", "#28c840"].map((c) => (
          <div key={c} style={{ width: 12, height: 12, borderRadius: "50%", background: c }} />
        ))}
        <div
          style={{
            flex: 1,
            marginLeft: 12,
            padding: "5px 14px",
            borderRadius: 14,
            background: "#ffffff",
            color: "#3c4043",
            fontSize: 14,
          }}
        >
          accounts.google.com
        </div>
      </div>
      <div style={{ padding: "34px 48px", color: "#202124" }}>
        {done ? (
          <div style={{ textAlign: "center", marginTop: 70 }}>
            <div style={{ fontSize: 56, color: "#1e8e3e", transform: `scale(${0.6 + 0.4 * at(0.7, 0.78)})` }}>✓</div>
            <div style={{ fontSize: 24, marginTop: 8 }}>Signed in</div>
          </div>
        ) : (
          <>
            <div style={{ fontSize: 30 }}>Sign in</div>
            <div style={{ fontSize: 16, marginTop: 8, color: "#5f6368" }}>with your Google Account</div>
            <div
              style={{
                marginTop: 34,
                padding: "14px 16px",
                border: `2px solid ${typed ? "#1a73e8" : "#dadce0"}`,
                borderRadius: 6,
                fontSize: 18,
                height: 24,
                lineHeight: "24px",
              }}
            >
              {typed || <span style={{ color: "#80868b" }}>Email or phone</span>}
            </div>
            <div style={{ display: "flex", justifyContent: "flex-end", marginTop: 30 }}>
              <div
                style={{
                  padding: "10px 26px",
                  borderRadius: 20,
                  background: pressed ? "#1557b0" : "#1a73e8",
                  color: "#ffffff",
                  fontSize: 16,
                }}
              >
                Next
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  );
};
