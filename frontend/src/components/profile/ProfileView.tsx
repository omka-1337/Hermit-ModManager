import { useEffect, useState } from "react";
import { errorMessage, Game, Library, Profile } from "../../api";
import { Button, ErrorText } from "../ui";
import BrowseTab from "./BrowseTab";
import InstalledTab from "./InstalledTab";

type Tab = "installed" | "browse";

type Props = {
  game: Game;
  profileId: string;
  onBack: () => void;
};

export default function ProfileView({ game, profileId, onBack }: Props) {
  const [profile, setProfile] = useState<Profile | null>(null);
  const [tab, setTab] = useState<Tab>("installed");
  const [error, setError] = useState("");

  useEffect(() => {
    Library.GetProfile(game.id, profileId)
      .then(setProfile)
      .catch((err) => setError(errorMessage(err)));
  }, [game.id, profileId]);

  const tabs: { id: Tab; label: string }[] = [
    { id: "installed", label: `Installed${profile?.mods?.length ? ` (${profile.mods.length})` : ""}` },
    { id: "browse", label: "Browse" },
  ];

  return (
    <div className="flex h-full flex-col">
      <header className="flex items-center gap-3 border-b border-zinc-800 px-6 pt-4">
        <Button variant="ghost" onClick={onBack} aria-label="Back to game">
          ←
        </Button>
        <div className="flex min-w-0 flex-col pb-3">
          <span className="text-xs text-zinc-500">{game.name}</span>
          <span className="truncate text-lg font-semibold">{profile?.name ?? "…"}</span>
        </div>
        <nav className="ml-6 flex gap-1 self-end">
          {tabs.map((t) => (
            <button
              key={t.id}
              onClick={() => setTab(t.id)}
              className={`border-b-2 px-3 pb-3 text-sm ${
                tab === t.id
                  ? "border-indigo-500 text-zinc-100"
                  : "border-transparent text-zinc-400 hover:text-zinc-200"
              }`}
            >
              {t.label}
            </button>
          ))}
        </nav>
        {game.activeProfile === profileId && (
          <span className="ml-auto self-center pb-3 text-xs font-medium text-indigo-400">Active profile</span>
        )}
      </header>

      <div className="min-h-0 flex-1">
        {error && (
          <div className="p-6">
            <ErrorText>{error}</ErrorText>
          </div>
        )}
        {profile && tab === "installed" && <InstalledTab profile={profile} onBrowse={() => setTab("browse")} />}
        {/* Browse stays mounted so search and scroll survive tab switches. */}
        <div className={tab === "browse" ? "h-full" : "hidden"}>
          <BrowseTab game={game} />
        </div>
      </div>
    </div>
  );
}
