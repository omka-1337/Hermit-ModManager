import { ReactNode, useState } from "react";

type Props = {
  label: string;
  children: ReactNode;
};

// Tooltip shows a label to the right of its child while hovered. It is
// positioned fixed so scrolling containers do not clip it.
export default function Tooltip({ label, children }: Props) {
  const [pos, setPos] = useState<{ x: number; y: number } | null>(null);

  return (
    <div
      onMouseEnter={(e) => {
        const r = e.currentTarget.getBoundingClientRect();
        setPos({ x: r.right + 10, y: r.top + r.height / 2 });
      }}
      onMouseLeave={() => setPos(null)}
    >
      {children}
      {pos && (
        <div
          role="tooltip"
          style={{ left: pos.x, top: pos.y }}
          className="pointer-events-none fixed z-50 -translate-y-1/2 rounded-md border border-zinc-700 bg-zinc-800 px-2.5 py-1 text-sm whitespace-nowrap text-zinc-100 shadow-lg"
        >
          {label}
        </div>
      )}
    </div>
  );
}
