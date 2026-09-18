import GameIcon from "../components/GameIcon";
import Tooltip from "../components/Tooltip";
import { GearIcon, PlusIcon, ShellProps } from "./shell";

// DesktopShell is the mouse-and-keyboard chrome: a narrow icon sidebar. Adding
// a game happens in a dialog above the content, which Main renders.
export default function DesktopShell({ games, selectedId, page, onOpenGame, onAdd, onSettings, children }: ShellProps) {
  return (
    <div className="flex h-full">
      <aside className="flex w-[72px] shrink-0 flex-col items-center border-r border-zinc-800 bg-zinc-900 py-3">
        <nav className="flex w-full flex-1 flex-col items-center gap-2 overflow-y-auto py-1">
          {games.map((g) => {
            const active = g.id === selectedId && page !== "settings";
            return (
              <Tooltip key={g.id} label={g.name}>
                <button aria-label={g.name} onClick={() => onOpenGame(g.id)} className="group relative flex items-center">
                  <span
                    className={`absolute -left-3.5 h-8 w-1 rounded-r bg-white transition-opacity ${
                      active ? "opacity-100" : "opacity-0 group-hover:opacity-40"
                    }`}
                  />
                  <GameIcon
                    name={g.name}
                    steamAppId={g.steamAppId}
                    size={44}
                    className={`transition ${active ? "ring-2 ring-indigo-500" : "opacity-80 group-hover:opacity-100"}`}
                  />
                </button>
              </Tooltip>
            );
          })}
          <Tooltip label="Add game">
            <button
              aria-label="Add game"
              onClick={onAdd}
              className="flex h-11 w-11 items-center justify-center rounded-lg border border-dashed border-zinc-700 text-zinc-400 hover:border-zinc-500 hover:text-zinc-100"
            >
              <PlusIcon />
            </button>
          </Tooltip>
        </nav>
        <Tooltip label="Settings">
          <button
            aria-label="Settings"
            onClick={onSettings}
            className={`mt-2 flex h-11 w-11 items-center justify-center rounded-lg ${
              page === "settings" ? "bg-zinc-800 text-white" : "text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100"
            }`}
          >
            <GearIcon />
          </button>
        </Tooltip>
      </aside>

      <main className="min-w-0 flex-1 overflow-auto">{children}</main>
    </div>
  );
}
