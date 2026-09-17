import { useState } from "react";
import { useLayout } from "../uimode";
import { inputClass } from "./ui";

type Props = {
  value: string;
  onSave: (name: string) => Promise<void>;
  className?: string;
};

// EditableName shows a name that turns into an input on double click, or on a
// tap of the pencil button where there is no double click (Steam Deck).
export default function EditableName({ value, onSave, className = "" }: Props) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(value);
  const deck = useLayout() === "deck";

  const commit = async () => {
    setEditing(false);
    const name = draft.trim();
    if (name && name !== value) {
      await onSave(name);
    }
  };

  const startEditing = () => {
    setDraft(value);
    setEditing(true);
  };

  if (!editing) {
    return (
      <span className="flex min-w-0 items-center gap-2">
        <span className={`cursor-text truncate ${className}`} title="Double-click to rename" onDoubleClick={startEditing}>
          {value}
        </span>
        {deck && (
          <button aria-label="Rename" onClick={startEditing} className="shrink-0 rounded p-1.5 text-zinc-500 active:bg-zinc-800">
            <svg viewBox="0 0 24 24" className="h-4 w-4" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
              <path d="M12 20h9M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4Z" />
            </svg>
          </button>
        )}
      </span>
    );
  }
  return (
    <input
      className={`${inputClass} ${className}`}
      value={draft}
      autoFocus
      onChange={(e) => setDraft(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => {
        if (e.key === "Enter") e.currentTarget.blur();
        if (e.key === "Escape") setEditing(false);
      }}
    />
  );
}
