import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { Events } from "@wailsio/runtime";
import {
  confirmDanger,
  Conflict,
  ConflictReason,
  Game,
  InstallService,
  LaunchService,
  Mod,
  Profile,
  Progress,
  Report,
} from "../../api";

type ProfileState = {
  game: Game;
  profile: Profile;
  installed: Map<string, Mod>;
  // busy is the mod id currently being installed or removed; one at a time per profile.
  busy: string | null;
  progress: Progress | null;
  // report describes the last game session of this profile, if any.
  report: Report | null;
  install: (namespace: string, name: string, version: string) => Promise<void>;
  uninstall: (modId: string) => Promise<void>;
  setEnabled: (modId: string, enabled: boolean) => Promise<void>;
};

const ProfileContext = createContext<ProfileState | null>(null);

export function useProfile(): ProfileState {
  const ctx = useContext(ProfileContext);
  if (!ctx) throw new Error("useProfile must be used inside ProfileProvider");
  return ctx;
}

type Props = {
  game: Game;
  profile: Profile;
  onProfileChange: (profile: Profile) => void;
  children: React.ReactNode;
};

export function ProfileProvider({ game, profile, onProfileChange, children }: Props) {
  const [busy, setBusy] = useState<string | null>(null);
  const [progress, setProgress] = useState<Progress | null>(null);
  const [report, setReport] = useState<Report | null>(null);

  // The game runs in a separate wrapper process; poll to notice when a session
  // ends, then pick up its report and any state changes.
  useEffect(() => {
    let wasRunning = false;
    const check = async () => {
      try {
        const info = await LaunchService.GetLaunchInfo(game.id);
        if (wasRunning && !info.running) {
          onProfileChange(await InstallService.OpenProfile(game.id, profile.id));
        }
        wasRunning = info.running;
        setReport(await LaunchService.GetLaunchReport(game.id, profile.id));
      } catch (err) {
        console.error(err);
      }
    };
    check();
    const timer = setInterval(check, 5000);
    return () => clearInterval(timer);
  }, [game.id, profile.id, onProfileChange]);

  useEffect(
    () =>
      Events.On("install:progress", (ev) => {
        const p = ev.data;
        if (p.gameId === game.id && p.profileId === profile.id) setProgress(p);
      }),
    [game.id, profile.id],
  );

  const run = useCallback(
    async (id: string, action: () => Promise<Profile>) => {
      setBusy(id);
      setProgress(null);
      try {
        onProfileChange(await action());
      } finally {
        setBusy(null);
        setProgress(null);
      }
    },
    [onProfileChange],
  );

  const value = useMemo<ProfileState>(
    () => ({
      game,
      profile,
      installed: new Map((profile.mods ?? []).map((m) => [m.id, m])),
      busy,
      progress,
      report,
      install: (namespace, name, version) =>
        run(`${namespace}-${name}`, async () => {
          const plan = await InstallService.PlanInstall(game.id, profile.id, namespace, name, version);
          const conflicts = plan.conflicts ?? [];
          const blocking = conflicts.find((c) => c.blocking);
          if (blocking) {
            throw new Error(`${packageName(blocking.package)} cannot be installed together with ${blocking.modName}.`);
          }
          if (conflicts.length > 0) {
            const ok = await confirmDanger("Conflicting mods", conflictMessage(conflicts), "Replace");
            if (!ok) return profile;
          }
          return InstallService.InstallPackage(game.id, profile.id, namespace, name, version, conflicts.length > 0);
        }),
      uninstall: (modId) => run(modId, () => InstallService.UninstallMod(game.id, profile.id, modId)),
      setEnabled: (modId, enabled) =>
        run(modId, () => InstallService.SetModEnabled(game.id, profile.id, modId, enabled)),
    }),
    [game, profile, busy, progress, report, run],
  );

  return <ProfileContext.Provider value={value}>{children}</ProfileContext.Provider>;
}

// dependantsOf returns installed mods that declare a dependency on modId.
export function dependantsOf(installed: Map<string, Mod>, modId: string): Mod[] {
  return [...installed.values()].filter((m) =>
    (m.dependencies ?? []).some((d) => d.startsWith(`${modId}-`) && !d.slice(modId.length + 1).includes("-")),
  );
}

// dependencyLabel describes an unmet dependency string "<author>-<name>-<version>".
export function dependencyLabel(installed: Map<string, Mod>, dep: string): string {
  const id = dep.replace(/-\d+\.\d+\.\d+$/, "");
  const name = id.slice(id.lastIndexOf("-") + 1);
  return installed.has(id) ? `${name} (disabled)` : `${name} (not installed)`;
}

export function thunderstoreIconURL(mod: Mod): string {
  return `https://gcdn.thunderstore.io/live/repository/icons/${mod.id}-${mod.version}.png`;
}

function packageName(pkg: string): string {
  const id = pkg.replace(/-\d+\.\d+\.\d+$/, "");
  return id.slice(id.lastIndexOf("-") + 1);
}

function conflictMessage(conflicts: Conflict[]): string {
  const lines = conflicts.map((c) =>
    c.reason === ConflictReason.ConflictLoader
      ? `• ${packageName(c.package)} is a BepInEx loader, only one can be installed: ${c.modName} will be uninstalled.`
      : `• ${packageName(c.package)} overwrites files of ${c.modName} (${(c.files ?? []).join(", ")}): ${c.modName} will be uninstalled.`,
  );
  return `${lines.join("\n")}\n\nMods that depend on the uninstalled ones will be disabled.`;
}
