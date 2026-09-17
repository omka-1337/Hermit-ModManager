import { useCallback, useEffect, useState } from "react";
import { confirmDanger, errorMessage, Game, Library, Profile } from "../api";
import EditableName from "./EditableName";
import { LaunchSetup, PlayButton } from "./launch";
import { ImportModal } from "./share";
import { BackendBadge, Button, ErrorText, inputClass, RuntimeBadge } from "./ui";

type Props = {
  game: Game;
  onChanged: (game: Game) => void;
  onRemoved: () => void;
  onOpenProfile: (profileId: string) => void;
};

export default function GameView({ game, onChanged, onRemoved, onOpenProfile }: Props) {
  const [profiles, setProfiles] = useState<Profile[]>([]);
  const [newProfile, setNewProfile] = useState("");
  const [error, setError] = useState("");
  const [importing, setImporting] = useState(false);

  const run = async (action: () => Promise<void>) => {
    setError("");
    try {
      await action();
    } catch (err) {
      setError(errorMessage(err));
    }
  };

  const loadProfiles = useCallback(async () => {
    setProfiles((await Library.ListProfiles(game.id)) ?? []);
  }, [game.id]);

  useEffect(() => {
    loadProfiles().catch((err) => setError(errorMessage(err)));
  }, [loadProfiles]);

  const refreshGame = async () => onChanged(await Library.GetGame(game.id));

  const removeGame = () =>
    run(async () => {
      const ok = await confirmDanger(
        "Remove game",
        `Remove "${game.name}" with all its profiles and installed mods? The game itself will not be touched.`,
        "Remove",
      );
      if (!ok) return;
      await Library.RemoveGame(game.id);
      onRemoved();
    });

  const createProfile = (e: React.FormEvent) => {
    e.preventDefault();
    run(async () => {
      await Library.CreateProfile(game.id, newProfile.trim());
      setNewProfile("");
      await loadProfiles();
    });
  };

  const removeProfile = (p: Profile) =>
    run(async () => {
      const ok = await confirmDanger(
        "Remove profile",
        `Remove profile "${p.name}" with all its installed mods and configs?`,
        "Remove",
      );
      if (!ok) return;
      await Library.RemoveProfile(game.id, p.id);
      await Promise.all([loadProfiles(), refreshGame()]);
    });

  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-6 p-8">
      <header className="flex items-start justify-between gap-4">
        <div className="flex min-w-0 flex-col gap-1">
          <EditableName
            value={game.name}
            className="text-2xl font-semibold"
            onSave={(name) => run(async () => onChanged(await Library.RenameGame(game.id, name)))}
          />
          <div className="flex items-center gap-2 text-sm text-zinc-500">
            <RuntimeBadge runtime={game.runtime} />
            <BackendBadge backend={game.backend} />
            <span className="truncate select-text">{game.path}</span>
          </div>
        </div>
        <div className="flex items-start gap-2">
          <Button variant="danger" onClick={removeGame}>
            Remove game
          </Button>
          <PlayButton game={game} profileId={game.activeProfile} onPlayed={refreshGame} />
        </div>
      </header>

      <ErrorText>{error}</ErrorText>

      <section className="flex flex-col gap-3">
        <h2 className="text-sm font-semibold tracking-wide text-zinc-400 uppercase">Launch</h2>
        <LaunchSetup game={game} />
      </section>

      <section className="flex flex-col gap-3">
        <h2 className="text-sm font-semibold tracking-wide text-zinc-400 uppercase">Profiles</h2>
        <ul className="divide-y divide-zinc-800 rounded-lg border border-zinc-800">
          {profiles.map((p) => {
            const active = p.id === game.activeProfile;
            const modCount = p.mods?.length ?? 0;
            return (
              <li key={p.id} className="flex items-center gap-3 px-4 py-2.5">
                <EditableName
                  value={p.name}
                  className="flex-1"
                  onSave={(name) =>
                    run(async () => {
                      await Library.RenameProfile(game.id, p.id, name);
                      await loadProfiles();
                    })
                  }
                />
                <span className="text-xs text-zinc-500">
                  {modCount} {modCount === 1 ? "mod" : "mods"}
                </span>
                <Button variant="secondary" onClick={() => onOpenProfile(p.id)}>
                  Open
                </Button>
                {active ? (
                  <span className="w-24 text-center text-xs font-medium text-indigo-400">Active</span>
                ) : (
                  <Button
                    variant="ghost"
                    className="w-24"
                    onClick={() => run(async () => onChanged(await Library.SetActiveProfile(game.id, p.id)))}
                  >
                    Set active
                  </Button>
                )}
                <Button variant="danger" disabled={profiles.length <= 1} onClick={() => removeProfile(p)}>
                  Remove
                </Button>
              </li>
            );
          })}
        </ul>
        <form onSubmit={createProfile} className="flex gap-2">
          <input
            className={inputClass}
            placeholder="New profile name"
            value={newProfile}
            onChange={(e) => setNewProfile(e.target.value)}
          />
          <Button type="submit" disabled={!newProfile.trim()}>
            Create profile
          </Button>
          <Button type="button" variant="ghost" onClick={() => setImporting(true)}>
            Import…
          </Button>
        </form>
      </section>

      {importing && (
        <ImportModal
          game={game}
          onClose={() => {
            setImporting(false);
            loadProfiles();
          }}
          onImported={(profileId) => {
            setImporting(false);
            onOpenProfile(profileId);
          }}
        />
      )}
    </div>
  );
}
