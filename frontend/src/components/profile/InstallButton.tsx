import { useState } from "react";
import { confirmDanger, errorMessage } from "../../api";
import { progressText } from "../../format";
import { Button, ErrorText } from "../ui";
import { dependantsOf, useProfile } from "./ProfileContext";

type Props = {
  namespace: string;
  name: string;
  latestVersion: string;
};

// InstallButton installs, updates or removes a regular mod in the open profile.
export default function InstallButton({ namespace, name, latestVersion }: Props) {
  const { installed, busy, progress, install, uninstall } = useProfile();
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

  const status = busy === id && progress ? progressText(progress) : "";

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-2">
        {mod && mod.version === latestVersion ? (
          <Button disabled>{mod.active ? "Installed" : "Installed (disabled)"}</Button>
        ) : (
          <Button
            variant="primary"
            disabled={busy !== null}
            onClick={() => act(() => install(namespace, name, latestVersion, false))}
          >
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
