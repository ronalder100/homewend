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
const { fontFamily: sansFamily } = loadSans("normal", { weights: ["400"], subsets: ["latin"] });

export const fps = 30;
export const width = 1000;
export const height = 560;
// The longest line, 68 columns, fits the wide shot whole.
const fontSize = 22;
const lineHeight = 1.45;
const rows = 13;

const sec = (s: number) => Math.round(s * fps);

// Characters a second, as a person types.
const typingSpeed = 60;

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
const print = (...lines: string[]) => {
  for (const text of lines) log.push({ at: sec(t), text });
};
const wait = (s: number) => {
  t += s;
};
const show = (s: number, view: Status["view"]) => {
  status.push({ from: sec(t), to: sec(t + s), view });
  t += s;
};

type("homewend login");
print("a browser window is open: sign in to Google there");
wait(0.6);
browser.from = sec(t);
wait(5);
browser.to = sec(t);
print("signed in");
wait(1);
print("");
type("homewend get --year 2025 --library ~/Pictures/Homewend");
print("asking Google Takeout for an export");
show(2, (p, f) => spinning(f, "asking Google for the export", 70 * p));
print(
  "Google is preparing the export: this can take hours.",
  "Leave this open, the download starts when it is ready.",
);
wait(1.2);
waiting.from = sec(t);
show(4, (_, f) => <Pulse frame={f}>waiting for Google</Pulse>);
waiting.to = sec(t);
print("", "the export is ready: downloading 2 parts, 2.0 GiB");
wait(0.8);

// One part and the manifest: the smallest real export, so the run is not a
// list of the same three lines.
const parts = [2 * GiB];
const seconds = [5];
parts.forEach((total, i) => {
  show(seconds[i], (p) => downloading(i + 1, parts.length + 1, total * p, total, 41 * MiB));
  print(`[${i + 1}/${parts.length + 1}] downloaded, ${ibytes(total)}`);
  print(`[${i + 1}/${parts.length + 1}] unpacking`);
});
wait(0.6);
print("[2/2] downloaded, 61 KiB");
show(3, (p) => placing(Math.round(742 * p), 742));
print("placed 742 photos, 2.0 GiB: 2 duplicates, 0 undated, 87 in albums");
wait(0.8);
print("declared 742, on disk 742, missing 0", "  2025  742 of 742", "", "download complete, congratulations 🎉");
wait(1.5);
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
      lines.push(line.text || " ");
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
  const fx = textLeft + 15 * charPx;
  const fy = textTop + (lines - 0.5) * linePx;
  return {
    transformOrigin: `${fx - margin}px ${fy - margin}px`,
    transform: `translate(${(width / 2 - fx) * ease}px, ${(height / 2 - fy) * ease}px) scale(${1 + 1.2 * ease})`,
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
        fontFamily: sansFamily,
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
                display: "flex",
                alignItems: "center",
                boxSizing: "border-box",
                height: 54,
                padding: "0 16px",
                border: `2px solid ${typed ? "#1a73e8" : "#dadce0"}`,
                borderRadius: 6,
                fontSize: 18,
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
