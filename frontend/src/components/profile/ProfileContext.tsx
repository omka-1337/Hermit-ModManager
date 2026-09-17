import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { Events } from "@wailsio/runtime";
import { Game, InstallService, Mod, Profile, Progress } from "../../api";

type ProfileState = {
  game: Game;
  profile: Profile;
  installed: Map<string, Mod>;
  // busy is the mod id currently being installed or removed; one at a time per profile.
  busy: string | null;
  progress: Progress | null;
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
      install: (namespace, name, version) =>
        run(`${namespace}-${name}`, () => InstallService.InstallPackage(game.id, profile.id, namespace, name, version)),
      uninstall: (modId) => run(modId, () => InstallService.UninstallMod(game.id, profile.id, modId)),
      setEnabled: (modId, enabled) =>
        run(modId, () => InstallService.SetModEnabled(game.id, profile.id, modId, enabled)),
    }),
    [game, profile, busy, progress, run],
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
