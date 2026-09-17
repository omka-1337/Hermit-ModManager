import { useEffect, useState } from "react";
import { InfoService } from "../bindings/bepinexmodmanager/internal/app";
import type { AppInfo } from "../bindings/bepinexmodmanager/internal/app";

type Page = "games" | "mods" | "settings";

const pages: { id: Page; label: string }[] = [
  { id: "games", label: "Games" },
  { id: "mods", label: "Mods" },
  { id: "settings", label: "Settings" },
];

function App() {
  const [page, setPage] = useState<Page>("games");
  const [info, setInfo] = useState<AppInfo | null>(null);

  useEffect(() => {
    InfoService.GetInfo().then(setInfo).catch(console.error);
  }, []);

  return (
    <div className="flex h-full">
      <aside className="flex w-56 flex-col border-r border-zinc-800 bg-zinc-900">
        <div className="px-4 py-5 text-lg font-semibold">BepInEx Mod Manager</div>
        <nav className="flex flex-col gap-1 px-2">
          {pages.map((p) => (
            <button
              key={p.id}
              onClick={() => setPage(p.id)}
              className={`rounded px-3 py-2 text-left text-sm ${
                page === p.id ? "bg-zinc-800 text-white" : "text-zinc-400 hover:bg-zinc-800/60"
              }`}
            >
              {p.label}
            </button>
          ))}
        </nav>
        <div className="mt-auto px-4 py-3 text-xs text-zinc-500">
          {info ? `v${info.version} · ${info.os}/${info.arch}` : "…"}
        </div>
      </aside>

      <main className="flex-1 overflow-auto p-6">
        <h1 className="mb-4 text-2xl font-semibold capitalize">{page}</h1>
        <p className="text-zinc-400">Coming soon.</p>
      </main>
    </div>
  );
}

export default App;
