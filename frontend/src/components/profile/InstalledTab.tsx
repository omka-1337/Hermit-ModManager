import { useState } from "react";
import { confirmDanger, errorMessage, Mod } from "../../api";
import { Button, ErrorText } from "../ui";
import PackageIcon from "./PackageIcon";
import { dependantsOf, thunderstoreIconURL, useProfile } from "./ProfileContext";

export default function InstalledTab({ onBrowse }: { onBrowse: () => void }) {
  const { profile, installed, busy, uninstall } = useProfile();
  const [error, setError] = useState("");
  const mods = profile.mods ?? [];

  const remove = async (mod: Mod) => {
    setError("");
    try {
      const dependants = dependantsOf(installed, mod.id).map((m) => m.name);
      const warning = dependants.length ? `\n\nThese installed mods depend on it: ${dependants.join(", ")}.` : "";
      if (await confirmDanger("Uninstall mod", `Uninstall ${mod.name}?${warning}`, "Uninstall")) {
        await uninstall(mod.id);
      }
    } catch (err) {
      setError(errorMessage(err));
    }
  };

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
    <div className="h-full overflow-y-auto px-6 py-3">
      <ErrorText>{error}</ErrorText>
      <ul className="divide-y divide-zinc-800">
        {mods.map((m) => {
          const dependants = dependantsOf(installed, m.id);
          return (
            <li key={m.id} className="flex items-center gap-3 py-2.5">
              <PackageIcon url={m.source.type === "thunderstore" ? thunderstoreIconURL(m) : ""} size={40} />
              <div className="flex min-w-0 flex-1 flex-col">
                <span className="truncate text-sm font-medium">
                  {m.name} <span className="font-normal text-zinc-500">by {m.author}</span>
                </span>
                {dependants.length > 0 && (
                  <span className="truncate text-xs text-zinc-500">
                    Required by {dependants.map((d) => d.name).join(", ")}
                  </span>
                )}
              </div>
              <span className="text-xs text-zinc-500">{m.version}</span>
              <Button variant="danger" disabled={busy !== null} onClick={() => remove(m)}>
                {busy === m.id ? "Removing…" : "Uninstall"}
              </Button>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
