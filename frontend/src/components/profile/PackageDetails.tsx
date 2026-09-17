import { useEffect, useState } from "react";
import { Browser } from "@wailsio/runtime";
import { BrowseService, errorMessage, isCancelled, PackageDetail } from "../../api";
import { formatAgo, formatBytes, formatCount } from "../../format";
import Markdown from "../Markdown";
import { Button, ErrorText, NsfwBadge } from "../ui";
import InstallButton from "./InstallButton";
import PackageIcon from "./PackageIcon";

type Props = {
  community: string;
  namespace: string;
  name: string;
  onClose: () => void;
  onOpenPackage: (namespace: string, name: string) => void;
};

export default function PackageDetails({ community, namespace, name, onClose, onOpenPackage }: Props) {
  const [pkg, setPkg] = useState<PackageDetail | null>(null);
  const [readme, setReadme] = useState<string | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    setPkg(null);
    setReadme(null);
    setError("");
    const detail = BrowseService.GetPackage(community, namespace, name);
    let readmeCall: ReturnType<typeof BrowseService.GetReadme> | null = null;
    detail
      .then((d) => {
        setPkg(d);
        readmeCall = BrowseService.GetReadme(namespace, name, d.latest_version_number);
        return readmeCall.then(setReadme);
      })
      .catch((err) => {
        if (!isCancelled(err)) setError(errorMessage(err));
      });
    return () => {
      detail.cancel();
      readmeCall?.cancel();
    };
  }, [community, namespace, name]);

  return (
    <div className="flex h-full flex-col">
      <div className="flex justify-end px-4 pt-3">
        <Button variant="ghost" onClick={onClose} aria-label="Close details">
          ✕
        </Button>
      </div>
      <div className="flex-1 overflow-y-auto px-6 pb-6">
        <ErrorText>{error}</ErrorText>
        {!pkg && !error && <p className="text-sm text-zinc-500">Loading…</p>}
        {pkg && (
          <div className="flex flex-col gap-5">
            <div className="flex gap-4">
              <PackageIcon url={pkg.icon_url} size={72} />
              <div className="flex min-w-0 flex-col gap-1">
                <div className="flex items-center gap-2">
                  <h2 className="truncate text-xl font-semibold">{pkg.name}</h2>
                  {pkg.is_nsfw && <NsfwBadge />}
                </div>
                <span className="text-sm text-zinc-400">by {pkg.namespace}</span>
                <span className="text-xs text-zinc-500">
                  v{pkg.latest_version_number} · updated {formatAgo(pkg.version_created)}
                </span>
              </div>
            </div>

            <p className="text-sm text-zinc-300">{pkg.description}</p>

            <InstallButton
              namespace={pkg.namespace}
              name={pkg.name}
              latestVersion={pkg.latest_version_number}
              modpack={(pkg.categories ?? []).some((c) => c.slug === "modpacks")}
            />

            <div className="-mt-2 flex items-center gap-2">
              <Button variant="ghost" onClick={() => Browser.OpenURL(pkg.page_url)}>
                Thunderstore ↗
              </Button>
              {pkg.website_url && (
                <Button variant="ghost" onClick={() => Browser.OpenURL(pkg.website_url)}>
                  Website ↗
                </Button>
              )}
            </div>

            <dl className="grid grid-cols-4 gap-3 text-sm">
              {[
                ["Downloads", formatCount(pkg.download_count)],
                ["Likes", formatCount(pkg.rating_count)],
                ["Size", formatBytes(pkg.size)],
                ["Dependants", formatCount(pkg.dependant_count)],
              ].map(([label, value]) => (
                <div key={label} className="rounded-md bg-zinc-800/60 px-3 py-2">
                  <dt className="text-xs text-zinc-500">{label}</dt>
                  <dd className="font-medium">{value}</dd>
                </div>
              ))}
            </dl>

            {!!pkg.categories?.length && (
              <div className="flex flex-wrap gap-1.5">
                {pkg.categories.map((c) => (
                  <span key={c.id} className="rounded bg-zinc-800 px-2 py-0.5 text-xs text-zinc-400">
                    {c.name}
                  </span>
                ))}
              </div>
            )}

            {!!pkg.dependencies?.length && (
              <section className="flex flex-col gap-2">
                <h3 className="text-xs font-semibold tracking-wide text-zinc-400 uppercase">
                  Dependencies ({pkg.dependencies.length})
                </h3>
                <ul className="flex flex-col gap-1">
                  {pkg.dependencies.map((d) => (
                    <li key={`${d.namespace}-${d.name}`}>
                      <button
                        className="flex w-full items-center gap-3 rounded-md px-2 py-1.5 text-left hover:bg-zinc-800"
                        onClick={() => onOpenPackage(d.namespace, d.name)}
                      >
                        <PackageIcon url={d.icon_url} size={32} />
                        <span className="min-w-0 flex-1 truncate text-sm">
                          {d.name} <span className="text-zinc-500">by {d.namespace}</span>
                        </span>
                        <span className="text-xs text-zinc-500">{d.version_number}</span>
                      </button>
                    </li>
                  ))}
                </ul>
              </section>
            )}

            <section className="border-t border-zinc-800 pt-2">
              {readme === null ? (
                <p className="py-3 text-sm text-zinc-500">Loading README…</p>
              ) : (
                <Markdown>{readme}</Markdown>
              )}
            </section>
          </div>
        )}
      </div>
    </div>
  );
}
