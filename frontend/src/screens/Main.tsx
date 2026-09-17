import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { Game } from "../api";
import GameIcon from "../components/GameIcon";
import GamePicker from "../components/GamePicker";
import GameView from "../components/GameView";
import ProfileView from "../components/profile/ProfileView";
import SettingsView from "./SettingsView";
import Tooltip from "../components/Tooltip";
import { useGamepad } from "../gamepad";
import { useLayout } from "../uimode";
import { Button, Modal } from "../components/ui";

type Props = {
  games: Game[];
  selectedId: string | null;
  onSelect: (id: string | null) => void;
  onGamesChanged: (selectId?: string | null) => Promise<void>;
};

// Entry is a stop in the console layout's top bar: a game, or one of the two
// actions at its end.
type Entry = { kind: "game"; game: Game } | { kind: "add" } | { kind: "settings" };

export default function Main({ games, selectedId, onSelect, onGamesChanged }: Props) {
  const deck = useLayout() === "deck";
  const [adding, setAdding] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [openProfile, setOpenProfile] = useState<{ gameId: string; profileId: string } | null>(null);
  const selected = games.find((g) => g.id === selectedId) ?? null;

  const openGame = (id: string) => {
    setOpenProfile(null);
    setShowSettings(false);
    onSelect(id);
  };

  const entries: Entry[] = [
    ...games.map((game): Entry => ({ kind: "game", game })),
    { kind: "add" },
    { kind: "settings" },
  ];
  const currentEntry = showSettings
    ? entries.length - 1
    : entries.findIndex((e) => e.kind === "game" && e.game.id === selectedId);
  const [barIndex, setBarIndex] = useState(Math.max(currentEntry, 0));

  // Follow selections made elsewhere, e.g. after adding a game.
  useEffect(() => {
    if (currentEntry >= 0) setBarIndex(currentEntry);
  }, [currentEntry]);

  // Moving onto a game or onto Settings switches the view right away; "add"
  // only highlights, because opening a dialog while browsing would be rude.
  const goto = (index: number) => {
    const wrapped = (index + entries.length) % entries.length;
    setBarIndex(wrapped);
    const entry = entries[wrapped];
    if (entry.kind === "game") openGame(entry.game.id);
    if (entry.kind === "settings") {
      setOpenProfile(null);
      setShowSettings(true);
    }
  };

  useGamepad(
    {
      onPrev: () => goto(barIndex - 1),
      onNext: () => goto(barIndex + 1),
      onAccept: () => {
        if (entries[barIndex]?.kind === "add") setAdding(true);
      },
      onBack: () => {
        if (openProfile) setOpenProfile(null);
        else if (showSettings && selected) openGame(selected.id);
      },
    },
    deck && !adding,
  );

  const content = showSettings ? (
    <SettingsView />
  ) : selected && openProfile?.gameId === selected.id ? (
    <ProfileView
      key={openProfile.profileId}
      game={selected}
      profileId={openProfile.profileId}
      onBack={() => setOpenProfile(null)}
      onGameChanged={() => onGamesChanged(selected.id)}
      onOpenProfile={(profileId) => setOpenProfile({ gameId: selected.id, profileId })}
    />
  ) : selected ? (
    <GameView
      key={selected.id}
      game={selected}
      onChanged={(g) => onGamesChanged(g.id)}
      onRemoved={() => onGamesChanged(null)}
      onOpenProfile={(profileId) => setOpenProfile({ gameId: selected.id, profileId })}
    />
  ) : (
    <div className="flex h-full flex-col items-center justify-center gap-3 text-center">
      <p className="text-zinc-400">{games.length ? "Select a game" : "No games added yet"}</p>
      {!games.length && (
        <Button variant="primary" onClick={() => setAdding(true)}>
          Add game
        </Button>
      )}
    </div>
  );

  const addGameModal = adding && (
    <Modal title="Add game" size="lg" onClose={() => setAdding(false)}>
      <GamePicker
        onAdded={async (g) => {
          setAdding(false);
          await onGamesChanged(g.id);
        }}
        actions={
          <Button variant="ghost" onClick={() => setAdding(false)}>
            Close
          </Button>
        }
      />
    </Modal>
  );

  if (deck) {
    return (
      <div className="flex h-full flex-col">
        <GameBar entries={entries} index={barIndex} onPick={goto} onAdd={() => setAdding(true)} />
        <main className="min-h-0 flex-1 overflow-auto">{content}</main>
        <HintBar entry={entries[barIndex]} inProfile={openProfile !== null} />
        {addGameModal}
      </div>
    );
  }

  return (
    <div className="flex h-full">
      <aside className="flex w-[72px] shrink-0 flex-col items-center border-r border-zinc-800 bg-zinc-900 py-3">
        <nav className="flex w-full flex-1 flex-col items-center gap-2 overflow-y-auto py-1">
          {games.map((g) => {
            const active = g.id === selectedId && !showSettings;
            return (
              <Tooltip key={g.id} label={g.name}>
                <button
                  aria-label={g.name}
                  onClick={() => openGame(g.id)}
                  className="group relative flex items-center"
                >
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
              onClick={() => setAdding(true)}
              className="flex h-11 w-11 items-center justify-center rounded-lg border border-dashed border-zinc-700 text-zinc-400 hover:border-zinc-500 hover:text-zinc-100"
            >
              <PlusIcon />
            </button>
          </Tooltip>
        </nav>
        <Tooltip label="Settings">
          <button
            aria-label="Settings"
            onClick={() => {
              setOpenProfile(null);
              setShowSettings(true);
            }}
            className={`mt-2 flex h-11 w-11 items-center justify-center rounded-lg ${
              showSettings ? "bg-zinc-800 text-white" : "text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100"
            }`}
          >
            <GearIcon />
          </button>
        </Tooltip>
      </aside>

      <main className="min-w-0 flex-1 overflow-auto">{content}</main>
      {addGameModal}
    </div>
  );
}

type GameBarProps = {
  entries: Entry[];
  index: number;
  onPick: (index: number) => void;
  onAdd: () => void;
};

// GameBar is the console layout's row of games, switched with LB/RB. The
// selected entry stays in the middle: the strip slides under it.
function GameBar({ entries, index, onPick, onAdd }: GameBarProps) {
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
                <button
                  key="add"
                  onClick={() => {
                    onPick(i);
                    onAdd();
                  }}
                  className={`${tile} border border-dashed border-zinc-700`}
                >
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

function HintBar({ entry, inProfile }: { entry: Entry | undefined; inProfile: boolean }) {
  const hints = [
    "LB / RB — switch",
    entry?.kind === "add" ? "A — add game" : null,
    inProfile ? "B — back to game" : null,
  ].filter(Boolean);
  return (
    <footer className="flex gap-4 border-t border-zinc-800 bg-zinc-900 px-4 py-1.5 text-xs text-zinc-500">
      {hints.map((hint) => (
        <span key={hint}>{hint}</span>
      ))}
    </footer>
  );
}

function PlusIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-5 w-5 shrink-0" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M12 5v14M5 12h14" strokeLinecap="round" />
    </svg>
  );
}

function GearIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      className="h-5 w-5 shrink-0"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
}
