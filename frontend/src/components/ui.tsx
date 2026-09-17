import type { ButtonHTMLAttributes, ReactNode } from "react";
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
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      title={title}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={`relative h-5 w-9 shrink-0 rounded-full transition-colors disabled:cursor-not-allowed disabled:opacity-40 ${
        checked ? "bg-indigo-600" : "bg-zinc-700"
      }`}
    >
      <span
        className={`absolute top-0.5 left-0.5 h-4 w-4 rounded-full bg-white transition-transform ${
          checked ? "translate-x-4" : ""
        }`}
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

export function Modal({ title, onClose, children, size = "md" }: ModalProps) {
  const sizes = useLayout() === "deck" ? deckModalSizes : modalSizes;
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
