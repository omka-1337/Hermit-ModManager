import { useEffect, useState } from "react";
import { Events } from "@wailsio/runtime";
import { errorMessage, Game, InstallService, PackageDetail, Progress } from "../../api";
import { progressText } from "../../format";
import { Button, ErrorText } from "../ui";

type Props = {
  game: Game;
  pkg: PackageDetail;
  onInstalled: (profileId: string) => void;
};

// ModpackInstallButton creates a new profile from a modpack.
export default function ModpackInstallButton({ game, pkg, onInstalled }: Props) {
  const id = `${pkg.namespace}-${pkg.name}`;
  const [busy, setBusy] = useState(false);
  const [progress, setProgress] = useState<Progress | null>(null);
  const [error, setError] = useState("");

  useEffect(
    () =>
      Events.On("install:progress", (ev) => {
        if (ev.data.gameId === game.id && ev.data.target === id) setProgress(ev.data);
      }),
    [game.id, id],
  );

  const install = async () => {
    setError("");
    setProgress(null);
    setBusy(true);
    try {
      const profile = await InstallService.InstallModpack(
        game.id,
        pkg.namespace,
        pkg.name,
        pkg.latest_version_number,
        pkg.name.replace(/_/g, " "),
      );
      onInstalled(profile.id);
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="flex flex-col gap-2">
      <Button variant="primary" className="self-start" disabled={busy} onClick={install}>
        {busy ? "Creating profile…" : "Create profile from modpack"}
      </Button>
      <p className="text-xs text-zinc-500">
        {pkg.dependencies?.length ?? 0} mods in their exact versions, with the modpack&apos;s configs.
      </p>
      {busy && progress && <p className="text-xs text-zinc-400">{progressText(progress)}</p>}
      <ErrorText>{error}</ErrorText>
    </div>
  );
}
