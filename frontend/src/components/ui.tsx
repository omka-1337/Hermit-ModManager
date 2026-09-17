import type { ButtonHTMLAttributes, ReactNode } from "react";
import { Runtime } from "../api";

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
  return <span className="rounded bg-zinc-800 px-2 py-0.5 text-xs text-zinc-300">{runtimeLabels[runtime] ?? runtimeLabels[Runtime.RuntimeUnknown]}</span>;
}

export function ErrorText({ children }: { children: ReactNode }) {
  return children ? <p className="text-sm text-red-400">{children}</p> : null;
}

export function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: ReactNode }) {
  return (
    <div className="fixed inset-0 z-10 flex items-center justify-center bg-black/60 p-6" onMouseDown={onClose}>
      <div
        className="w-full max-w-lg rounded-lg border border-zinc-800 bg-zinc-900 p-6 shadow-xl"
        onMouseDown={(e) => e.stopPropagation()}
      >
        <h2 className="mb-4 text-lg font-semibold">{title}</h2>
        {children}
      </div>
    </div>
  );
}
