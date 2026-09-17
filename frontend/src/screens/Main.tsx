import { useState } from "react";
import { Game } from "../api";
import GameIcon from "../components/GameIcon";
import GamePicker from "../components/GamePicker";
import GameView from "../components/GameView";
import ProfileView from "../components/profile/ProfileView";
import SettingsView from "./SettingsView";
import Tooltip from "../components/Tooltip";
import { Button, Modal } from "../components/ui";

type Props = {
  games: Game[];
  selectedId: string | null;
  onSelect: (id: string | null) => void;
  onGamesChanged: (selectId?: string | null) => Promise<void>;
};

export default function Main({ games, selectedId, onSelect, onGamesChanged }: Props) {
  const [adding, setAdding] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [openProfile, setOpenProfile] = useState<{ gameId: string; profileId: string } | null>(null);
  const selected = games.find((g) => g.id === selectedId) ?? null;

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
                  onClick={() => {
                    setOpenProfile(null);
                    setShowSettings(false);
                    onSelect(g.id);
                  }}
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
              <svg viewBox="0 0 24 24" className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M12 5v14M5 12h14" strokeLinecap="round" />
              </svg>
            </button>
          </Tooltip>
        </nav>
        <Tooltip label="Settings">
          <button
            aria-label="Settings"
            onClick={() => setShowSettings(true)}
            className={`mt-2 flex h-11 w-11 items-center justify-center rounded-lg ${
              showSettings ? "bg-zinc-800 text-white" : "text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100"
            }`}
          >
            <svg viewBox="0 0 24 24" className="h-5 w-5" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z" />
              <circle cx="12" cy="12" r="3" />
            </svg>
          </button>
        </Tooltip>
      </aside>

      <main className="min-w-0 flex-1 overflow-auto">
        {showSettings ? (
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
        )}
      </main>

      {adding && (
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
      )}
    </div>
  );
}
