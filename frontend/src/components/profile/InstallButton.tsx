import { useState } from "react";
import { confirmDanger, errorMessage, Stage } from "../../api";
import { formatBytes } from "../../format";
import { Button, ErrorText } from "../ui";
import { dependantsOf, useProfile } from "./ProfileContext";

type Props = {
  namespace: string;
  name: string;
  latestVersion: string;
  modpack: boolean;
};

export default function InstallButton({ namespace, name, latestVersion, modpack }: Props) {
  const { profile, installed, busy, progress, install, installModpack, uninstall } = useProfile();
  const [error, setError] = useState("");
  const id = `${namespace}-${name}`;
  const mod = installed.get(id);

  const act = async (action: () => Promise<void>) => {
    setError("");
    try {
      await action();
    } catch (err) {
      setError(errorMessage(err));
    }
  };

  const remove = () =>
    act(async () => {
      const dependants = dependantsOf(installed, id).map((m) => m.name);
      const warning = dependants.length
        ? `\n\nThese mods depend on it and will be disabled: ${dependants.join(", ")}.`
        : "";
      if (await confirmDanger("Uninstall mod", `Uninstall ${name}?${warning}`, "Uninstall")) {
        await uninstall(id);
      }
    });

  let status = "";
  if (busy === id && progress) {
    const what = progress.package.replace(/-\d+\.\d+\.\d+$/, "");
    const count = progress.steps > 1 ? ` ${progress.step}/${progress.steps}` : "";
    if (progress.stage === Stage.StageDownload) {
      const size =
        progress.total > 0
          ? `${Math.floor((progress.done / progress.total) * 100)}%`
          : formatBytes(progress.done);
      status = `Downloading${count} · ${what} ${size}`;
    } else {
      status = `Installing${count} · ${what}`;
    }
  }

  if (modpack) {
    const current = profile.modpack?.replace(/-\d+\.\d+\.\d+$/, "") === id;
    return (
      <div className="flex flex-col gap-2">
        <div className="flex items-center gap-2">
          <Button
            variant="primary"
            disabled={busy !== null}
            onClick={() => act(() => installModpack(namespace, name, latestVersion))}
          >
            {busy === id ? "Installing…" : "Install as new profile"}
          </Button>
          {current && <span className="text-xs text-zinc-500">This profile was created from it</span>}
        </div>
        <p className="text-xs text-zinc-500">A modpack becomes its own profile with the exact mod versions it lists.</p>
        {status && <p className="text-xs text-zinc-400">{status}</p>}
        <ErrorText>{error}</ErrorText>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-2">
        {mod && mod.version === latestVersion ? (
          <Button disabled>{mod.active ? "Installed" : "Installed (disabled)"}</Button>
        ) : (
          <Button variant="primary" disabled={busy !== null} onClick={() => act(() => install(namespace, name, latestVersion, modpack))}>
            {busy === id ? "Installing…" : mod ? `Update to ${latestVersion}` : "Install"}
          </Button>
        )}
        {mod && (
          <Button variant="danger" disabled={busy !== null} onClick={remove}>
            Uninstall
          </Button>
        )}
      </div>
      {status && <p className="text-xs text-zinc-400">{status}</p>}
      <ErrorText>{error}</ErrorText>
    </div>
  );
}
