import { Profile } from "../../api";
import { Button } from "../ui";

type Props = {
  profile: Profile;
  onBrowse: () => void;
};

export default function InstalledTab({ profile, onBrowse }: Props) {
  const mods = profile.mods ?? [];
  if (mods.length === 0) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-3">
        <p className="text-zinc-400">No mods installed in this profile.</p>
        <Button variant="primary" onClick={onBrowse}>
          Browse mods
        </Button>
      </div>
    );
  }
  return (
    <ul className="divide-y divide-zinc-800 px-6 py-3">
      {mods.map((m) => (
        <li key={m.id} className="flex items-center gap-3 py-2.5">
          <span className="flex-1 text-sm">{m.name}</span>
          <span className="text-xs text-zinc-500">{m.version}</span>
        </li>
      ))}
    </ul>
  );
}
