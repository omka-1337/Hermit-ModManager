const compact = new Intl.NumberFormat("en", { notation: "compact", maximumFractionDigits: 1 });

export function formatCount(n: number): string {
  return compact.format(n);
}

export function formatBytes(bytes: number): string {
  const units = ["B", "KB", "MB", "GB"];
  let i = 0;
  while (bytes >= 1024 && i < units.length - 1) {
    bytes /= 1024;
    i++;
  }
  return `${bytes.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

const relative = new Intl.RelativeTimeFormat("en", { numeric: "auto" });

export function formatAgo(iso: string): string {
  const seconds = (new Date(iso).getTime() - Date.now()) / 1000;
  const steps: [Intl.RelativeTimeFormatUnit, number][] = [
    ["year", 31536000],
    ["month", 2592000],
    ["day", 86400],
    ["hour", 3600],
    ["minute", 60],
  ];
  for (const [unit, size] of steps) {
    if (Math.abs(seconds) >= size) return relative.format(Math.round(seconds / size), unit);
  }
  return "just now";
}

// compareVersions compares dotted numeric versions like "5.4.2100".
export function compareVersions(a: string, b: string): number {
  const as = a.split(".").map(Number);
  const bs = b.split(".").map(Number);
  for (let i = 0; i < Math.max(as.length, bs.length); i++) {
    const d = (as[i] ?? 0) - (bs[i] ?? 0);
    if (d !== 0) return d;
  }
  return 0;
}

// packageLabel turns "<author>-<name>-<version>" into "Name v1.2.3". Authors may
// contain "-", package names may not.
export function packageLabel(full: string): string {
  const m = full.match(/^.*-([^-]+)-(\d+\.\d+\.\d+)$/);
  return m ? `${m[1].replace(/_/g, " ")} v${m[2]}` : full;
}

// progressText describes install progress, e.g. "Downloading 12/40 · LethalLib 45%".
export function progressText(p: {
  package: string;
  stage: string;
  done: number;
  total: number;
  step: number;
  steps: number;
}): string {
  const what = p.package.replace(/-\d+\.\d+\.\d+$/, "").replace(/^.*-/, "");
  const count = p.steps > 1 ? ` ${p.step}/${p.steps}` : "";
  if (p.stage === "download") {
    const size = p.total > 0 ? `${Math.floor((p.done / p.total) * 100)}%` : formatBytes(p.done);
    return `Downloading${count} · ${what} ${size}`;
  }
  return `Installing${count} · ${what}`;
}
