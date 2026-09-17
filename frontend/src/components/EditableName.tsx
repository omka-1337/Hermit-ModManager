import { useState } from "react";
import { inputClass } from "./ui";

type Props = {
  value: string;
  onSave: (name: string) => Promise<void>;
  className?: string;
};

// EditableName shows a name that turns into an input on double click.
export default function EditableName({ value, onSave, className = "" }: Props) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(value);

  const commit = async () => {
    setEditing(false);
    const name = draft.trim();
    if (name && name !== value) {
      await onSave(name);
    }
  };

  if (!editing) {
    return (
      <span
        className={`cursor-text ${className}`}
        title="Double-click to rename"
        onDoubleClick={() => {
          setDraft(value);
          setEditing(true);
        }}
      >
        {value}
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
