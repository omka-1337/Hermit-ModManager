export default function PackageIcon({ url, size }: { url: string; size: number }) {
  return url ? (
    <img src={url} alt="" loading="lazy" className="shrink-0 rounded-md bg-zinc-800" style={{ width: size, height: size }} />
  ) : (
    <div className="shrink-0 rounded-md bg-zinc-800" style={{ width: size, height: size }} />
  );
}

// PackageCover is the icon as a card's header: square and full width.
export function PackageCover({ url }: { url: string }) {
  return (
    <div className="aspect-square w-full bg-zinc-800">
      {url && <img src={url} alt="" loading="lazy" className="h-full w-full object-cover" />}
    </div>
  );
}
