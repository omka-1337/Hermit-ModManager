import { useEffect, useState, type ButtonHTMLAttributes, type ReactNode } from "react";
import { Backend, Runtime } from "../api";
import { useLayout } from "../uimode";

type Variant = "primary" | "secondary" | "ghost" | "danger";

const variants: Record<Variant, string> = {
  primary: "bg-indigo-600 text-white hover:bg-indigo-500",
  secondary: "bg-zinc-800 text-zinc-100 hover:bg-zinc-700",
  ghost: "text-zinc-400 hover:bg-zinc-800 hover:text-zinc-100",
  danger: "text-red-400 hover:bg-red-500/10",
};

export function Button({
  variant = "secondary",
  className = "",
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: Variant }) {
  return (
    <button
      {...props}
      className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors disabled:pointer-events-none disabled:opacity-50 ${variants[variant]} ${className}`}
    />
  );
}

// Spinner marks work that takes a while: checking for updates, updating a mod.
export function Spinner({ className = "" }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={`h-4 w-4 shrink-0 animate-spin ${className}`} fill="none" aria-hidden>
      <circle cx="12" cy="12" r="9" stroke="currentColor" strokeOpacity="0.25" strokeWidth="3" />
      <path d="M21 12a9 9 0 0 0-9-9" stroke="currentColor" strokeWidth="3" strokeLinecap="round" />
    </svg>
  );
}

export const inputClass =
  "w-full rounded-md border border-zinc-700 bg-zinc-900 px-3 py-1.5 text-sm outline-none focus:border-indigo-500 select-text";

const runtimeLabels: Partial<Record<Runtime, string>> = {
  [Runtime.RuntimeNative]: "Linux native",
  [Runtime.RuntimeProton]: "Proton",
  [Runtime.RuntimeUnknown]: "Unknown runtime",
};

export function RuntimeBadge({ runtime }: { runtime: Runtime }) {
  return <span className="shrink-0 rounded bg-zinc-800 px-2 py-0.5 text-xs whitespace-nowrap text-zinc-300">{runtimeLabels[runtime] ?? runtimeLabels[Runtime.RuntimeUnknown]}</span>;
}

const backendLabels: Partial<Record<Backend, string>> = {
  [Backend.BackendMono]: "Mono",
  [Backend.BackendIL2CPP]: "IL2CPP",
};

export function BackendBadge({ backend }: { backend: Backend }) {
  const label = backendLabels[backend];
  return label ? <span className="shrink-0 rounded bg-zinc-800 px-2 py-0.5 text-xs whitespace-nowrap text-zinc-300">{label}</span> : null;
}

type ToggleProps = {
  checked: boolean;
  disabled?: boolean;
  title?: string;
  onChange: (checked: boolean) => void;
};

export function Toggle({ checked, disabled, title, onChange }: ToggleProps) {
  // A finger needs a bigger switch than a mouse pointer.
  const deck = useLayout() === "deck";
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      title={title}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={`relative shrink-0 rounded-full transition-colors disabled:cursor-not-allowed disabled:opacity-40 ${
        deck ? "h-7 w-12" : "h-5 w-9"
      } ${checked ? "bg-indigo-600" : "bg-zinc-700"}`}
    >
      <span
        className={`absolute rounded-full bg-white transition-transform ${
          deck ? "top-1 left-1 h-5 w-5" : "top-0.5 left-0.5 h-4 w-4"
        } ${checked ? (deck ? "translate-x-5" : "translate-x-4") : ""}`}
      />
    </button>
  );
}

export function NsfwBadge() {
  return <span className="shrink-0 rounded bg-red-500/15 px-1.5 py-0.5 text-[10px] font-semibold whitespace-nowrap text-red-400">NSFW</span>;
}

export function ErrorText({ children }: { children: ReactNode }) {
  return children ? <p className="text-sm whitespace-pre-line text-red-400">{children}</p> : null;
}

type ModalProps = {
  title: string;
  onClose: () => void;
  children: ReactNode;
  // size: md for forms, lg for lists, xl for full browsers filling most of the window.
  size?: "md" | "lg" | "xl";
};

const modalSizes = {
  md: "max-w-lg",
  lg: "max-w-2xl",
  xl: "flex h-[85vh] max-w-6xl flex-col",
};

// On the Deck's small screen dialogs use the whole window.
const deckModalSizes = {
  md: "max-w-xl",
  lg: "max-w-3xl",
  xl: "flex h-full max-w-none flex-1 flex-col",
};

// Open dialogs are counted so controller navigation behind them can pause.
let openModals = 0;
const modalListeners = new Set<(open: boolean) => void>();

// useModalOpen reports whether any dialog is currently on screen.
export function useModalOpen(): boolean {
  const [open, setOpen] = useState(openModals > 0);
  useEffect(() => {
    modalListeners.add(setOpen);
    setOpen(openModals > 0);
    return () => void modalListeners.delete(setOpen);
  }, []);
  return open;
}

export function Modal({ title, onClose, children, size = "md" }: ModalProps) {
  const sizes = useLayout() === "deck" ? deckModalSizes : modalSizes;

  useEffect(() => {
    openModals++;
    modalListeners.forEach((notify) => notify(true));
    return () => {
      openModals--;
      modalListeners.forEach((notify) => notify(openModals > 0));
    };
  }, []);

  return (
    <div
      className={`fixed inset-0 z-10 flex items-center justify-center bg-black/60 ${sizes === deckModalSizes ? "p-3" : "p-6"}`}
      onMouseDown={onClose}
    >
      <div
        className={`w-full ${sizes[size]} rounded-lg border border-zinc-800 bg-zinc-900 p-6 shadow-xl`}
        onMouseDown={(e) => e.stopPropagation()}
      >
        <h2 className="mb-4 text-lg font-semibold">{title}</h2>
        {size === "xl" ? <div className="flex min-h-0 flex-1 flex-col">{children}</div> : children}
      </div>
    </div>
  );
}
