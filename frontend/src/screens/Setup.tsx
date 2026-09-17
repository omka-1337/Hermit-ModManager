import { Game } from "../api";
import AddGameForm from "../components/AddGameForm";
import { Button } from "../components/ui";

type Props = {
  onFinish: (game?: Game) => void;
};

export default function Setup({ onFinish }: Props) {
  return (
    <div className="flex h-full items-center justify-center p-6">
      <div className="w-full max-w-lg rounded-lg border border-zinc-800 bg-zinc-900 p-8">
        <h1 className="mb-2 text-2xl font-semibold">Welcome</h1>
        <p className="mb-6 text-sm text-zinc-400">
          Add your first game to get started. Mods are kept in profiles inside the manager, so the game folder stays
          untouched. You can also do this later.
        </p>
        <AddGameForm
          onAdded={onFinish}
          extraActions={
            <Button type="button" variant="ghost" onClick={() => onFinish()}>
              Skip
            </Button>
          }
        />
      </div>
    </div>
  );
}
