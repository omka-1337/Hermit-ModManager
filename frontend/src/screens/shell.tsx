import { createContext, ReactNode, useContext, useEffect } from "react";
import { Game } from "../api";

// Page is what the shell's chrome is pointing at. The content itself is built
// by Main and passed in as children.
export type Page = "game" | "add" | "settings";

// ShellProps is the contract between Main, which owns the state, and a shell,
// which owns the chrome: a sidebar on the desktop, a game bar on the deck.
export type ShellProps = {
  games: Game[];
  selectedId: string | null;
  page: Page;
  // canGoBack tells the console shell whether B has somewhere to return to.
  canGoBack: boolean;
  onOpenGame: (id: string) => void;
  onAdd: () => void;
  onSettings: () => void;
  onBack: () => void;
  children: ReactNode;
};

// Pages can add their own button hints to the console layout's hint bar; on
// the desktop there is no provider and the hook does nothing.
export const HintsContext = createContext<((hints: string[]) => void) | null>(null);

export function usePageHints(hints: string[]) {
  const publish = useContext(HintsContext);
  const key = hints.join("|");
  useEffect(() => {
    publish?.(key ? key.split("|") : []);
    return () => publish?.([]);
  }, [publish, key]);
}

export function PlusIcon() {
  return (
    <svg viewBox="0 0 24 24" className="h-5 w-5 shrink-0" fill="none" stroke="currentColor" strokeWidth="2">
      <path d="M12 5v14M5 12h14" strokeLinecap="round" />
    </svg>
  );
}

export function GearIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      className="h-5 w-5 shrink-0"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
    >
      <path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
}

export function PowerIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      className="h-5 w-5 shrink-0"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
    >
      <path d="M12 4v8" />
      <path d="M7.5 7a7 7 0 1 0 9 0" />
    </svg>
  );
}
