import { useState } from "react";
import { Game } from "../api";
import GameIcon from "../components/GameIcon";
import GamePicker from "../components/GamePicker";
import GameView from "../components/GameView";
import ProfileView from "../components/profile/ProfileView";
import SettingsView from "./SettingsView";
import Tooltip from "../components/Tooltip";
import { useLayout } from "../uimode";
import { Button, Modal } from "../components/ui";

type Props = {
  games: Game[];
  selectedId: string | null;
  onSelect: (id: string | null) => void;
  onGamesChanged: (selectId?: string | null) => Promise<void>;
};

export default function Main({ games, selectedId, onSelect, onGamesChanged }: Props) {
  const deck = useLayout() === "deck";
  const [adding, setAdding] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [openProfile, setOpenProfile] = useState<{ gameId: string; profileId: string } | null>(null);
  const selected = games.find((g) => g.id === selectedId) ?? null;

  return (
    <div className="flex h-full">
      <aside
        className={`flex shrink-0 flex-col border-r border-zinc-800 bg-zinc-900 py-3 ${
          deck ? "w-56" : "w-[72px] items-center"
        }`}
      >
        <nav className={`flex w-full flex-1 flex-col gap-2 overflow-y-auto py-1 ${deck ? "px-2" : "items-center"}`}>
          {games.map((g) => {
            const active = g.id === selectedId && !showSettings;
            const open = () => {
              setOpenProfile(null);
              setShowSettings(false);
              onSelect(g.id);
            };
            if (deck) {
              return (
                <button
                  key={g.id}
                  onClick={open}
                  className={`flex items-center gap-3 rounded-lg px-2 py-2 text-left ${
                    active ? "bg-zinc-800 text-white" : "text-zinc-300 active:bg-zinc-800/60"
                  }`}
                >
                  <GameIcon name={g.name} steamAppId={g.steamAppId} size={40} />
                  <span className="min-w-0 flex-1 truncate text-sm">{g.name}</span>
                </button>
              );
            }
            return (
              <Tooltip key={g.id} label={g.name}>
                <button aria-label={g.name} onClick={open} className="group relative flex items-center">
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
          <SidebarButton deck={deck} label="Add game" onClick={() => setAdding(true)} dashed>
            <path d="M12 5v14M5 12h14" strokeLinecap="round" />
          </SidebarButton>
        </nav>
        <div className={deck ? "px-2" : ""}>
          <SidebarButton
            deck={deck}
            label="Settings"
            active={showSettings}
            onClick={() => setShowSettings(true)}
          >
            <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z" />
            <circle cx="12" cy="12" r="3" />
          </SidebarButton>
        </div>
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

type SidebarButtonProps = {
  deck: boolean;
  label: string;
  active?: boolean;
  dashed?: boolean;
  onClick: () => void;
  children: React.ReactNode;
};

// SidebarButton is an icon with a tooltip on the desktop and an icon with a
// label on the Deck, where hovering does not exist.
function SidebarButton({ deck, label, active, dashed, onClick, children }: SidebarButtonProps) {
  const icon = (
    <svg
      viewBox="0 0 24 24"
      className="h-5 w-5 shrink-0"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      {children}
    </svg>
  );
  if (deck) {
    return (
      <button
        onClick={onClick}
        className={`flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-left text-sm ${
          dashed ? "border border-dashed border-zinc-700 text-zinc-400" : ""
        } ${active ? "bg-zinc-800 text-white" : "text-zinc-300 active:bg-zinc-800/60"}`}
      >
        {icon}
        {label}
      </button>
    );
  }
  return (
    <Tooltip label={label}>
      <button
        aria-label={label}
        onClick={onClick}
        className={`flex h-11 w-11 items-center justify-center rounded-lg ${
          dashed
            ? "border border-dashed border-zinc-700 text-zinc-400 hover:border-zinc-500 hover:text-zinc-100"
            : active
              ? "bg-zinc-800 text-white"
              : "text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100"
        }`}
      >
        {icon}
      </button>
    </Tooltip>
  );
}
