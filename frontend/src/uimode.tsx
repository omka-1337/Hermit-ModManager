import { createContext, ReactNode, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { Settings, SettingsStore, UIMode } from "./api";

// Layout is the interface the app is currently drawn for: a desktop window or
// the Steam Deck's small touch screen.
export type Layout = "desktop" | "deck";

type UIModeState = {
  layout: Layout;
  // preference is what the user chose: auto follows the hardware.
  preference: UIMode;
  steamDeck: boolean;
  setPreference: (mode: UIMode) => Promise<void>;
};

const UIModeContext = createContext<UIModeState | null>(null);

export function useUIMode(): UIModeState {
  const ctx = useContext(UIModeContext);
  if (!ctx) throw new Error("useUIMode must be used inside UIModeProvider");
  return ctx;
}

// useLayout is a shortcut for components that only switch on the layout.
export function useLayout(): Layout {
  return useUIMode().layout;
}

type Props = {
  settings: Settings;
  steamDeck: boolean;
  children: ReactNode;
};

export function UIModeProvider({ settings, steamDeck, children }: Props) {
  const [preference, setStoredPreference] = useState<UIMode>(settings.uiMode);
  const layout: Layout =
    preference === UIMode.UIModeDeck || (preference === UIMode.UIModeAuto && steamDeck) ? "deck" : "desktop";

  // The deck layout is scaled up through the root font size, so CSS needs to
  // know the layout too.
  useEffect(() => {
    document.documentElement.dataset.layout = layout;
  }, [layout]);

  const setPreference = useCallback(async (mode: UIMode) => {
    const current = await SettingsStore.Get();
    const saved = await SettingsStore.Update({ ...current, uiMode: mode });
    setStoredPreference(saved.uiMode);
  }, []);

  const value = useMemo<UIModeState>(
    () => ({ layout, preference, steamDeck, setPreference }),
    [layout, preference, steamDeck, setPreference],
  );
  return <UIModeContext.Provider value={value}>{children}</UIModeContext.Provider>;
}
