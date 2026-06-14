// format.js — small, pure helpers for turning raw numbers into readable strings.
// Everything here treats missing/zero data as "—" so the UI never shows
// misleading zeros when a value isn't really available.

const DASH = '—';

// lapTime formats seconds as M:SS.mmm (e.g. 1:23.456).
export function lapTime(s) {
  if (!s || s <= 0) return DASH;
  const m = Math.floor(s / 60);
  const sec = s - m * 60;
  return `${m}:${sec.toFixed(3).padStart(6, '0')}`;
}

// secs formats a short duration like a sector time (e.g. 23.456).
export function secs(s, digits = 3) {
  if (!s || s <= 0) return DASH;
  return s.toFixed(digits);
}

// gap formats a time gap with a leading sign (e.g. +2.1). Zero reads as "—".
export function gap(s) {
  if (s == null || s === 0) return DASH;
  return `${s > 0 ? '+' : ''}${s.toFixed(1)}`;
}

// clock formats a duration as H:MM:SS (or M:SS under an hour) for session time.
export function clock(s) {
  if (s == null || s < 0) return DASH;
  s = Math.floor(s);
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  const mm = String(m).padStart(2, '0');
  const ss = String(sec).padStart(2, '0');
  return h > 0 ? `${h}:${mm}:${ss}` : `${m}:${ss}`;
}

// num formats a plain number to fixed digits, with "—" for missing values.
export function num(v, digits = 0) {
  if (v == null || Number.isNaN(v)) return DASH;
  return v.toFixed(digits);
}

// pct formats a 0..1 fraction as a whole-number percentage (e.g. 0.62 -> "62%").
export function pct(v) {
  if (v == null || Number.isNaN(v)) return DASH;
  return `${Math.round(v * 100)}%`;
}

// gear formats the gear number (0 = N, -1 = R).
export function gear(g) {
  if (g == null) return DASH;
  if (g === 0) return 'N';
  if (g < 0) return 'R';
  return String(g);
}

// titleCase capitalizes a lowercase enum-ish string (e.g. "race" -> "Race").
export function titleCase(s) {
  if (!s) return DASH;
  return s
    .replace(/_/g, ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

// tempColor returns a CSS variable name suggesting how hot a tire/brake is.
export function tempClass(c) {
  if (c == null) return '';
  if (c < 70) return 'cold';
  if (c > 105) return 'hot';
  return 'ok';
}
