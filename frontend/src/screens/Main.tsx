import { useState } from "react";
import { Game } from "../api";
import GamePicker from "../components/GamePicker";
import GameView from "../components/GameView";
import ProfileView from "../components/profile/ProfileView";
import SettingsView from "./SettingsView";
import ConsoleShell from "./ConsoleShell";
import DesktopShell from "./DesktopShell";
import { Page } from "./shell";
import { useLayout } from "../uimode";
import { Button, Modal } from "../components/ui";

type Props = {
  games: Game[];
  selectedId: string | null;
  onSelect: (id: string | null) => void;
  onGamesChanged: (selectId?: string | null) => Promise<void>;
};

// Main owns the navigation state and builds the page content; the chrome around
// it comes from the shell the current layout asks for.
export default function Main({ games, selectedId, onSelect, onGamesChanged }: Props) {
  const deck = useLayout() === "deck";
  const [page, setPage] = useState<Page>("game");
  const [openProfile, setOpenProfile] = useState<{ gameId: string; profileId: string } | null>(null);
  const selected = games.find((g) => g.id === selectedId) ?? null;

  const openGame = (id: string) => {
    setOpenProfile(null);
    setPage("game");
    onSelect(id);
  };

  // A game added from either picker becomes the selected one.
  const addedGame = async (id: string) => {
    openGame(id);
    await onGamesChanged(id);
  };

  const shellProps = {
    games,
    selectedId,
    page,
    canGoBack: openProfile !== null || (page !== "game" && selected !== null),
    onOpenGame: openGame,
    onAdd: () => {
      setOpenProfile(null);
      setPage("add");
    },
    onSettings: () => {
      setOpenProfile(null);
      setPage("settings");
    },
    onBack: () => {
      if (openProfile) setOpenProfile(null);
      else if (page !== "game" && selected) openGame(selected.id);
    },
  };

  // The desktop adds games in a dialog over the current page, the console
  // layout on a page of its own.
  const addGamePage = (
    <div className="mx-auto flex max-w-3xl flex-col gap-4 p-6">
      <h2 className="text-xl font-semibold">Add game</h2>
      <GamePicker onAdded={(g) => addedGame(g.id)} />
    </div>
  );

  const content =
    page === "settings" ? (
      <SettingsView />
    ) : deck && (page === "add" || !selected) ? (
      // With no games at all the bar highlights Add game, so show its page.
      addGamePage
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
          <Button variant="primary" onClick={() => setPage("add")}>
            Add game
          </Button>
        )}
      </div>
    );

  const Shell = deck ? ConsoleShell : DesktopShell;

  return (
    <>
      <Shell {...shellProps}>{content}</Shell>
      {!deck && page === "add" && (
        <Modal title="Add game" size="lg" onClose={() => setPage("game")}>
          <GamePicker
            onAdded={(g) => addedGame(g.id)}
            actions={
              <Button variant="ghost" onClick={() => setPage("game")}>
                Close
              </Button>
            }
          />
        </Modal>
      )}
    </>
  );
}
