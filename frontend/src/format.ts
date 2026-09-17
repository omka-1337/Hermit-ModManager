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
