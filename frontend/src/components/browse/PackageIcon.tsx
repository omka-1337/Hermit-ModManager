export default function PackageIcon({ url, size }: { url: string; size: number }) {
  return url ? (
    <img src={url} alt="" loading="lazy" className="shrink-0 rounded-md bg-zinc-800" style={{ width: size, height: size }} />
  ) : (
    <div className="shrink-0 rounded-md bg-zinc-800" style={{ width: size, height: size }} />
  );
}
