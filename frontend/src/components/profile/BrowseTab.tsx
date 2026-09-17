import { useEffect, useRef, useState } from "react";
import {
  BrowseService,
  Community,
  errorMessage,
  Filters,
  Game,
  isCancelled,
  Ordering,
  PackageSummary,
} from "../../api";
import { formatAgo, formatCount } from "../../format";
import { ErrorText, inputClass, NsfwBadge } from "../ui";
import PackageDetails from "./PackageDetails";
import PackageIcon from "./PackageIcon";

const orderings: { value: Ordering; label: string }[] = [
  { value: Ordering.OrderMostDownloaded, label: "Most downloaded" },
  { value: Ordering.OrderTopRated, label: "Top rated" },
  { value: Ordering.OrderLastUpdated, label: "Last updated" },
  { value: Ordering.OrderNewest, label: "Newest" },
];

type Selected = { namespace: string; name: string };

export default function BrowseTab({ game }: { game: Game }) {
  const [community, setCommunity] = useState<Community | null>(null);
  const [filters, setFilters] = useState<Filters | null>(null);
  const [fatal, setFatal] = useState("");

  const [query, setQuery] = useState("");
  const [debouncedQuery, setDebouncedQuery] = useState("");
  const [ordering, setOrdering] = useState<Ordering>(Ordering.OrderMostDownloaded);
  const [section, setSection] = useState("");

  const [packages, setPackages] = useState<PackageSummary[]>([]);
  const [page, setPage] = useState(1);
  const [hasMore, setHasMore] = useState(false);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [selected, setSelected] = useState<Selected | null>(null);

  const listRef = useRef<HTMLDivElement>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const call = BrowseService.GetCommunity(game.id);
    call
      .then((c) => {
        setCommunity(c);
        return BrowseService.GetFilters(c.id).then(setFilters);
      })
      .catch((err) => {
        if (!isCancelled(err)) setFatal(errorMessage(err));
      });
    return () => {
      call.cancel();
    };
  }, [game.id]);

  useEffect(() => {
    const t = setTimeout(() => setDebouncedQuery(query.trim()), 300);
    return () => clearTimeout(t);
  }, [query]);

  // Any filter change restarts the listing from the first page.
  useEffect(() => {
    setPage(1);
    listRef.current?.scrollTo({ top: 0 });
  }, [debouncedQuery, ordering, section]);

  useEffect(() => {
    if (!community) return;
    setLoading(true);
    setError("");
    const call = BrowseService.ListPackages(community.id, { query: debouncedQuery, ordering, section, page });
    call
      .then((list) => {
        const items = list.packages ?? [];
        setPackages((prev) => (page === 1 ? items : [...prev, ...items]));
        setHasMore(list.has_more);
        setTotal(list.count);
        setLoading(false);
      })
      .catch((err) => {
        if (isCancelled(err)) return;
        setError(errorMessage(err));
        setLoading(false);
      });
    return () => {
      call.cancel();
    };
  }, [community, debouncedQuery, ordering, section, page]);

  // Load the next page when the bottom of the list scrolls into view.
  useEffect(() => {
    const el = sentinelRef.current;
    if (!el || !hasMore || loading) return;
    const observer = new IntersectionObserver(
      (entries) => entries[0].isIntersecting && setPage((p) => p + 1),
      { root: listRef.current, rootMargin: "400px" },
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [hasMore, loading, packages]);

  if (fatal) {
    return <div className="p-8 text-sm text-zinc-400">{fatal}</div>;
  }
  if (!community) {
    return <div className="p-8 text-sm text-zinc-500">Connecting to Thunderstore…</div>;
  }

  const sections = [...(filters?.sections ?? [])].sort((a, b) => b.priority - a.priority);

  return (
    <div className="flex h-full min-h-0">
      <div className="flex min-w-0 flex-1 flex-col">
        <div className="flex items-center gap-2 border-b border-zinc-800 px-6 py-3">
          <input
            className={inputClass}
            placeholder={`Search ${community.name} mods`}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          <select
            className={`${inputClass} w-auto`}
            value={section}
            onChange={(e) => setSection(e.target.value)}
          >
            <option value="">All</option>
            {sections.map((s) => (
              <option key={s.uuid} value={s.uuid}>
                {s.name}
              </option>
            ))}
          </select>
          <select
            className={`${inputClass} w-auto`}
            value={ordering}
            onChange={(e) => setOrdering(e.target.value as Ordering)}
          >
            {orderings.map((o) => (
              <option key={o.value} value={o.value}>
                {o.label}
              </option>
            ))}
          </select>
        </div>

        <div ref={listRef} className="flex-1 overflow-y-auto">
          <div className="px-6 py-2 text-xs text-zinc-500">{formatCount(total)} packages</div>
          <ul className="flex flex-col px-3 pb-3">
            {packages.map((p) => {
              const active = selected?.namespace === p.namespace && selected?.name === p.name;
              return (
                <li key={`${p.namespace}-${p.name}`}>
                  <button
                    onClick={() => setSelected({ namespace: p.namespace, name: p.name })}
                    className={`flex w-full gap-3 rounded-md px-3 py-2.5 text-left ${
                      active ? "bg-zinc-800" : "hover:bg-zinc-800/50"
                    }`}
                  >
                    <PackageIcon url={p.icon_url} size={48} />
                    <div className="flex min-w-0 flex-1 flex-col">
                      <div className="flex items-baseline gap-2">
                        <span className="truncate text-sm font-medium">{p.name}</span>
                        <span className="truncate text-xs text-zinc-500">by {p.namespace}</span>
                        {p.is_nsfw && <NsfwBadge />}
                      </div>
                      <p className="line-clamp-2 text-xs text-zinc-400">{p.description}</p>
                      <div className="mt-1 flex gap-3 text-xs text-zinc-500">
                        <span>↓ {formatCount(p.download_count)}</span>
                        <span>♥ {formatCount(p.rating_count)}</span>
                        <span>{formatAgo(p.last_updated)}</span>
                      </div>
                    </div>
                  </button>
                </li>
              );
            })}
          </ul>
          <div ref={sentinelRef} />
          <div className="px-6 pb-6">
            <ErrorText>{error}</ErrorText>
            {loading && <p className="text-center text-sm text-zinc-500">Loading…</p>}
            {!loading && !error && packages.length === 0 && (
              <p className="text-center text-sm text-zinc-500">No packages found.</p>
            )}
          </div>
        </div>
      </div>

      {selected && (
        <aside className="w-[440px] shrink-0 border-l border-zinc-800 bg-zinc-900">
          <PackageDetails
            key={`${selected.namespace}-${selected.name}`}
            community={community.id}
            namespace={selected.namespace}
            name={selected.name}
            onClose={() => setSelected(null)}
            onOpenPackage={(namespace, name) => setSelected({ namespace, name })}
          />
        </aside>
      )}
    </div>
  );
}
