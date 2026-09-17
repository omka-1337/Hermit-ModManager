import { useEffect, useRef } from "react";

// Standard gamepad mapping (https://w3c.github.io/gamepad/#remapping), which
// is what Steam Input reports for the Steam Deck's own controls.
const buttons = {
  a: 0,
  b: 1,
  lb: 4,
  rb: 5,
  left: 14,
  right: 15,
} as const;

// Keys that stand in for the buttons while testing on a desktop.
const keys: Record<string, keyof Handlers> = {
  BracketLeft: "onPrev",
  BracketRight: "onNext",
  PageUp: "onPrev",
  PageDown: "onNext",
  Escape: "onBack",
};

export type Handlers = {
  onPrev?: () => void;
  onNext?: () => void;
  onAccept?: () => void;
  onBack?: () => void;
};

// useGamepad calls the handlers when a controller button is pressed: LB/RB and
// the D-pad left/right switch, A accepts, B goes back. Keyboard equivalents
// exist so the same navigation can be used (and tested) without a controller.
export function useGamepad(handlers: Handlers, enabled = true) {
  const latest = useRef(handlers);
  latest.current = handlers;

  useEffect(() => {
    if (!enabled) return;

    const onKeyDown = (e: KeyboardEvent) => {
      const target = e.target as HTMLElement | null;
      // Let text fields keep their keys.
      if (target && ["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName)) return;
      const handler = keys[e.code];
      if (handler && latest.current[handler]) {
        e.preventDefault();
        latest.current[handler]?.();
      }
    };
    window.addEventListener("keydown", onKeyDown);

    // Gamepads are polled; the API has no press events.
    let frame = 0;
    const pressed = new Map<number, boolean>();
    const poll = () => {
      frame = requestAnimationFrame(poll);
      const pad = navigator.getGamepads?.().find((p) => p?.connected);
      if (!pad) return;
      const edge = (index: number) => {
        const down = (pad.buttons[index]?.value ?? 0) > 0.5;
        const was = pressed.get(index) ?? false;
        pressed.set(index, down);
        return down && !was;
      };
      if (edge(buttons.lb) || edge(buttons.left)) latest.current.onPrev?.();
      if (edge(buttons.rb) || edge(buttons.right)) latest.current.onNext?.();
      if (edge(buttons.a)) latest.current.onAccept?.();
      if (edge(buttons.b)) latest.current.onBack?.();
    };
    frame = requestAnimationFrame(poll);

    return () => {
      window.removeEventListener("keydown", onKeyDown);
      cancelAnimationFrame(frame);
    };
  }, [enabled]);
}
