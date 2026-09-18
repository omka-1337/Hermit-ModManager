import { createContext, ReactNode, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { BrowseView, Settings, SettingsStore, UIMode } from "./api";

// Layout is the interface the app is currently drawn for: a desktop window or
// the Steam Deck's small touch screen.
export type Layout = "desktop" | "deck";

type UIModeState = {
  layout: Layout;
  // preference is what the user chose: auto follows the hardware.
  preference: UIMode;
  steamDeck: boolean;
  setPreference: (mode: UIMode) => Promise<void>;
  // cards is the mod browser layout in use; browseView is what was chosen.
  cards: boolean;
  browseView: BrowseView;
  setBrowseView: (view: BrowseView) => Promise<void>;
};

const UIModeContext = createContext<UIModeState | null>(null);

export function useUIMode(): UIModeState {
  const ctx = useContext(UIModeContext);
  if (!ctx) throw new Error("useUIMode must be used inside UIModeProvider");
  return ctx;
}

// useLayout is for components that only switch on the layout. It also works
// outside the provider, e.g. in the window frame shown while loading.
export function useLayout(): Layout {
  return useContext(UIModeContext)?.layout ?? "desktop";
}

type Props = {
  settings: Settings;
  steamDeck: boolean;
  children: ReactNode;
};

export function UIModeProvider({ settings, steamDeck, children }: Props) {
  const [preference, setStoredPreference] = useState<UIMode>(settings.uiMode);
  const [browseView, setStoredBrowseView] = useState<BrowseView>(settings.browseView);
  const layout: Layout =
    preference === UIMode.UIModeDeck || (preference === UIMode.UIModeAuto && steamDeck) ? "deck" : "desktop";
  // Cards suit a screen driven by fingers and a controller; the desktop lists
  // more packages at once, so each layout has its own default.
  const cards =
    browseView === BrowseView.BrowseViewCards || (browseView === BrowseView.BrowseViewAuto && layout === "deck");

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

  const setBrowseView = useCallback(async (view: BrowseView) => {
    const current = await SettingsStore.Get();
    const saved = await SettingsStore.Update({ ...current, browseView: view });
    setStoredBrowseView(saved.browseView);
  }, []);

  const value = useMemo<UIModeState>(
    () => ({ layout, preference, steamDeck, setPreference, cards, browseView, setBrowseView }),
    [layout, preference, steamDeck, setPreference, cards, browseView, setBrowseView],
  );
  return <UIModeContext.Provider value={value}>{children}</UIModeContext.Provider>;
}
