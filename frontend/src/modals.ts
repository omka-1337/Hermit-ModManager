import { useEffect, useState } from "react";

// Dialogs stack: only the topmost one takes the controller, and the console
// shell stops navigating while any of them is open.
let stack: number[] = [];
let next = 1;
const listeners = new Set<() => void>();

function notify() {
  for (const listener of listeners) listener();
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  return () => void listeners.delete(listener);
}

// useModalLayer registers a dialog while active and reports whether it is the
// topmost one, so a confirmation opened over a dialog takes over the buttons.
export function useModalLayer(active: boolean): boolean {
  const [top, setTop] = useState(false);

  useEffect(() => {
    if (!active) {
      setTop(false);
      return;
    }
    const id = next++;
    stack.push(id);
    const update = () => setTop(stack[stack.length - 1] === id);
    const unsubscribe = subscribe(update);
    update();
    notify();
    return () => {
      stack = stack.filter((other) => other !== id);
      unsubscribe();
      notify();
    };
  }, [active]);

  return top;
}

// useModalOpen reports whether any dialog is currently on screen.
export function useModalOpen(): boolean {
  const [open, setOpen] = useState(stack.length > 0);
  useEffect(() => {
    const update = () => setOpen(stack.length > 0);
    const unsubscribe = subscribe(update);
    update();
    return unsubscribe;
  }, []);
  return open;
}
