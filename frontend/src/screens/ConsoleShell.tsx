import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { Game } from "../api";
import GameIcon from "../components/GameIcon";
import { useGamepad } from "../gamepad";
import { useModalOpen } from "../components/ui";
import { GearIcon, PlusIcon, ShellProps } from "./shell";

// Entry is a stop in the game bar: a game, or one of the two actions after them.
type Entry = { kind: "game"; game: Game } | { kind: "add" } | { kind: "settings" };

// ConsoleShell is the Steam Deck chrome: games in a row along the top, one page
// per entry, switched with the shoulder buttons like Steam's Big Picture.
export default function ConsoleShell({
  games,
  selectedId,
  page,
  canGoBack,
  onOpenGame,
  onAdd,
  onSettings,
  onBack,
  children,
}: ShellProps) {
  const entries: Entry[] = [
    ...games.map((game): Entry => ({ kind: "game", game })),
    { kind: "add" },
    { kind: "settings" },
  ];
  const current =
    page === "settings"
      ? entries.length - 1
      : page === "add"
        ? entries.length - 2
        : entries.findIndex((e) => e.kind === "game" && e.game.id === selectedId);
  const [index, setIndex] = useState(Math.max(current, 0));

  // Follow page changes made elsewhere, e.g. right after a game is added.
  useEffect(() => {
    if (current >= 0) setIndex(current);
  }, [current]);

  // Every entry is a page, so moving the highlight switches the view at once.
  const goto = (to: number) => {
    const wrapped = (to + entries.length) % entries.length;
    setIndex(wrapped);
    const entry = entries[wrapped];
    if (entry.kind === "game") onOpenGame(entry.game.id);
    if (entry.kind === "add") onAdd();
    if (entry.kind === "settings") onSettings();
  };

  // A dialog takes over the buttons while it is open.
  const modalOpen = useModalOpen();
  useGamepad(
    {
      onPrev: () => goto(index - 1),
      onNext: () => goto(index + 1),
      onBack,
    },
    !modalOpen,
  );

  return (
    <div className="flex h-full flex-col">
      <GameBar entries={entries} index={index} onPick={goto} />
      <main className="min-h-0 flex-1 overflow-auto">{children}</main>
      <HintBar canGoBack={canGoBack} />
    </div>
  );
}

type GameBarProps = {
  entries: Entry[];
  index: number;
  onPick: (index: number) => void;
};

// GameBar keeps the selected entry in the middle: the strip slides under it.
function GameBar({ entries, index, onPick }: GameBarProps) {
  const viewport = useRef<HTMLDivElement>(null);
  const strip = useRef<HTMLDivElement>(null);
  const [offset, setOffset] = useState(0);

  useLayoutEffect(() => {
    const center = () => {
      const box = viewport.current;
      const tile = strip.current?.children[index] as HTMLElement | undefined;
      if (!box || !tile) return;
      setOffset(box.clientWidth / 2 - (tile.offsetLeft + tile.offsetWidth / 2));
    };
    center();
    // The window can be resized (and the deck rotates its own scale).
    const observer = new ResizeObserver(center);
    if (viewport.current) observer.observe(viewport.current);
    return () => observer.disconnect();
  }, [index, entries.length]);

  return (
    <header className="flex items-center gap-3 border-b border-zinc-800 bg-zinc-900 px-3 py-2">
      <ShoulderHint label="LB" />
      <div ref={viewport} className="relative min-w-0 flex-1 overflow-hidden">
        <div
          ref={strip}
          className="flex gap-2 transition-transform duration-200 ease-out"
          style={{ transform: `translateX(${offset}px)` }}
        >
          {entries.map((entry, i) => {
            const active = i === index;
            const tile = `flex h-[5rem] w-28 shrink-0 flex-col items-center justify-center gap-1 rounded-xl px-2 text-xs transition ${
              active ? "bg-zinc-800 text-white ring-2 ring-indigo-500" : "text-zinc-400 opacity-70"
            }`;
            if (entry.kind === "game") {
              return (
                <button key={entry.game.id} onClick={() => onPick(i)} className={tile}>
                  <GameIcon name={entry.game.name} steamAppId={entry.game.steamAppId} size={36} />
                  <span className="line-clamp-2 w-full text-center leading-tight break-words">{entry.game.name}</span>
                </button>
              );
            }
            if (entry.kind === "add") {
              return (
                <button key="add" onClick={() => onPick(i)} className={`${tile} border border-dashed border-zinc-700`}>
                  <PlusIcon />
                  Add game
                </button>
              );
            }
            return (
              <button key="settings" onClick={() => onPick(i)} className={tile}>
                <GearIcon />
                Settings
              </button>
            );
          })}
        </div>
      </div>
      <ShoulderHint label="RB" />
    </header>
  );
}

function ShoulderHint({ label }: { label: string }) {
  return (
    <span className="shrink-0 rounded-md border border-zinc-700 px-2 py-1 text-xs font-semibold text-zinc-400">
      {label}
    </span>
  );
}

function HintBar({ canGoBack }: { canGoBack: boolean }) {
  const hints = ["LB / RB — switch", canGoBack ? "B — back" : null].filter(Boolean);
  return (
    <footer className="flex gap-4 border-t border-zinc-800 bg-zinc-900 px-4 py-1.5 text-xs text-zinc-500">
      {hints.map((hint) => (
        <span key={hint}>{hint}</span>
      ))}
    </footer>
  );
}
