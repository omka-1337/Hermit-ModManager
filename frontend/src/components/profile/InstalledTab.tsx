import { useState } from "react";
import { confirmDanger, errorMessage, Mod } from "../../api";
import { Button, ErrorText, Toggle } from "../ui";
import LaunchReport, { issueText } from "./LaunchReport";
import PackageIcon from "./PackageIcon";
import { dependantsOf, dependencyLabel, thunderstoreIconURL, useProfile } from "./ProfileContext";

export default function InstalledTab({ onBrowse }: { onBrowse: () => void }) {
  const { profile, installed, busy, report, uninstall, setEnabled, updates, checkingUpdates, checkUpdates, update, updateAll } =
    useProfile();
  const [error, setError] = useState("");
  const mods = profile.mods ?? [];

  const act = async (action: () => Promise<void>) => {
    setError("");
    try {
      await action();
    } catch (err) {
      setError(errorMessage(err));
    }
  };

  const remove = (mod: Mod) =>
    act(async () => {
      const dependants = dependantsOf(installed, mod.id).map((m) => m.name);
      const warning = dependants.length
        ? `\n\nThese mods depend on it and will be disabled: ${dependants.join(", ")}.`
        : "";
      if (await confirmDanger("Uninstall mod", `Uninstall ${mod.name}?${warning}`, "Uninstall")) {
        await uninstall(mod.id);
      }
    });

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
      <div className="mb-3 flex items-center justify-between gap-3 text-sm">
        <span className={updates.size ? "text-amber-400" : "text-zinc-500"}>
          {checkingUpdates
            ? "Checking for updates…"
            : updates.size
              ? `${updates.size} ${updates.size === 1 ? "update" : "updates"} available`
              : "All mods are up to date"}
        </span>
        <div className="flex gap-2">
          <Button variant="ghost" disabled={checkingUpdates || busy !== null} onClick={() => act(checkUpdates)}>
            Check again
          </Button>
          {updates.size > 0 && (
            <Button
              variant="primary"
              disabled={busy !== null}
              onClick={() =>
                act(async () => {
                  const result = await updateAll();
                  const failed = Object.entries(result?.failed ?? {});
                  if (failed.length) {
                    throw new Error(failed.map(([id, reason]) => `${id}: ${reason}`).join("\n"));
                  }
                })
              }
            >
              {busy === "update-all" ? "Updating…" : "Update all"}
            </Button>
          )}
        </div>
      </div>
      {report && <LaunchReport report={report} />}
      <ul className="divide-y divide-zinc-800">
        {mods.map((m) => {
          const unmet = (m.unmetDependencies ?? []).map((d) => dependencyLabel(installed, d));
          const dependants = dependantsOf(installed, m.id).filter((d) => d.enabled);
          const issue = report?.issues?.find((i) => i.modId === m.id);
          return (
            <li key={m.id} className="flex items-center gap-3 py-2.5">
              <Toggle
                checked={m.active}
                disabled={busy !== null || unmet.length > 0}
                title={unmet.length ? `Requires ${unmet.join(", ")}` : m.active ? "Disable" : "Enable"}
                onChange={(enabled) => act(() => setEnabled(m.id, enabled))}
              />
              <div className={`flex min-w-0 flex-1 items-center gap-3 ${m.active ? "" : "opacity-50"}`}>
                <PackageIcon url={m.source.type === "thunderstore" ? thunderstoreIconURL(m) : ""} size={40} />
                <div className="flex min-w-0 flex-col">
                  <span className="truncate text-sm font-medium">
                    {m.name} <span className="font-normal text-zinc-500">by {m.author}</span>
                  </span>
                  {issue && m.active && <span className="truncate text-xs text-red-400">{issueText(issue)}</span>}
                  {unmet.length > 0 ? (
                    <span className="truncate text-xs text-amber-400">
                      {m.enabled ? "Disabled: requires" : "Requires"} {unmet.join(", ")}
                    </span>
                  ) : (
                    dependants.length > 0 && (
                      <span className="truncate text-xs text-zinc-500">
                        Required by {dependants.map((d) => d.name).join(", ")}
                      </span>
                    )
                  )}
                </div>
              </div>
              <span className="text-xs text-zinc-500">
                {m.version}
                {updates.has(m.id) && <span className="text-amber-400"> → {updates.get(m.id)}</span>}
              </span>
              {updates.has(m.id) && (
                <Button disabled={busy !== null} onClick={() => act(() => update(m.id))}>
                  {busy === m.id ? "Updating…" : "Update"}
                </Button>
              )}
              <Button variant="danger" disabled={busy !== null} onClick={() => remove(m)}>
                {busy === m.id ? "Working…" : "Uninstall"}
              </Button>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
