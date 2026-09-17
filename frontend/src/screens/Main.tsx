import { useEffect, useState } from "react";
import { AppInfo, Game, InfoService } from "../api";
import GamePicker from "../components/GamePicker";
import GameView from "../components/GameView";
import ProfileView from "../components/profile/ProfileView";
import { Button, Modal } from "../components/ui";

type Props = {
  games: Game[];
  selectedId: string | null;
  onSelect: (id: string | null) => void;
  onGamesChanged: (selectId?: string | null) => Promise<void>;
};

export default function Main({ games, selectedId, onSelect, onGamesChanged }: Props) {
  const [adding, setAdding] = useState(false);
  const [info, setInfo] = useState<AppInfo | null>(null);
  const [openProfile, setOpenProfile] = useState<{ gameId: string; profileId: string } | null>(null);
  const selected = games.find((g) => g.id === selectedId) ?? null;

  useEffect(() => {
    InfoService.GetInfo().then(setInfo).catch(console.error);
  }, []);

  return (
    <div className="flex h-full">
      <aside className="flex w-60 flex-col border-r border-zinc-800 bg-zinc-900">
        <div className="px-4 py-5 text-lg font-semibold">BepInEx Mod Manager</div>
        <div className="px-4 pb-2 text-xs font-semibold tracking-wide text-zinc-500 uppercase">Games</div>
        <nav className="flex flex-col gap-0.5 overflow-y-auto px-2">
          {games.map((g) => (
            <button
              key={g.id}
              onClick={() => {
                setOpenProfile(null);
                onSelect(g.id);
              }}
              className={`truncate rounded-md px-3 py-2 text-left text-sm ${
                g.id === selectedId ? "bg-zinc-800 text-white" : "text-zinc-400 hover:bg-zinc-800/60"
              }`}
            >
              {g.name}
            </button>
          ))}
        </nav>
        <div className="px-2 pt-2">
          <Button variant="ghost" className="w-full text-left" onClick={() => setAdding(true)}>
            + Add game
          </Button>
        </div>
        <div className="mt-auto px-4 py-3 text-xs text-zinc-500">
          {info ? `v${info.version} · ${info.os}/${info.arch}` : "…"}
        </div>
      </aside>

      <main className="min-w-0 flex-1 overflow-auto">
        {selected && openProfile?.gameId === selected.id ? (
          <ProfileView
            key={openProfile.profileId}
            game={selected}
            profileId={openProfile.profileId}
            onBack={() => setOpenProfile(null)}
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
        <Modal title="Add game" wide onClose={() => setAdding(false)}>
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
