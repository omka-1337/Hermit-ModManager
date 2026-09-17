import { useEffect, useState } from "react";
import { errorMessage, Game, Library, SettingsStore } from "./api";
import WindowFrame from "./components/WindowFrame";
import Main from "./screens/Main";
import Setup from "./screens/Setup";

type State = { status: "loading" } | { status: "error"; message: string } | { status: "ready"; setup: boolean };

function App() {
  const [state, setState] = useState<State>({ status: "loading" });
  const [games, setGames] = useState<Game[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);

  // reloadGames refreshes the list; selectId overrides the selection (null clears it).
  const reloadGames = async (selectId?: string | null) => {
    const list = (await Library.ListGames()) ?? [];
    setGames(list);
    setSelectedId((current) => {
      const want = selectId === undefined ? current : selectId;
      return list.some((g) => g.id === want) ? want : (list[0]?.id ?? null);
    });
  };

  useEffect(() => {
    Promise.all([SettingsStore.Get(), reloadGames()])
      .then(([settings]) => setState({ status: "ready", setup: !settings.setupCompleted }))
      .catch((err) => setState({ status: "error", message: errorMessage(err) }));
  }, []);

  const finishSetup = async (game?: Game) => {
    try {
      await SettingsStore.CompleteSetup();
      await reloadGames(game?.id);
      setState({ status: "ready", setup: false });
    } catch (err) {
      setState({ status: "error", message: errorMessage(err) });
    }
  };

  let screen: React.ReactNode = null;
  switch (state.status) {
    case "error":
      screen = <div className="p-8 text-red-400">Failed to load: {state.message}</div>;
      break;
    case "ready":
      screen = state.setup ? (
        <Setup onFinish={finishSetup} />
      ) : (
        <Main games={games} selectedId={selectedId} onSelect={setSelectedId} onGamesChanged={reloadGames} />
      );
  }
  return <WindowFrame>{screen}</WindowFrame>;
}

export default App;
