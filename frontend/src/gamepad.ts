import { useEffect, useRef } from "react";

// Standard gamepad mapping (https://w3c.github.io/gamepad/#remapping), which
// is what Steam Input reports for the Steam Deck's own controls.
const buttons = {
  a: 0,
  b: 1,
  lb: 4,
  rb: 5,
  lt: 6,
  rt: 7,
  up: 12,
  down: 13,
  left: 14,
  right: 15,
} as const;

// Keys that stand in for the buttons: the arrows are the d-pad, so the same
// navigation works from a keyboard (and can be tested without a controller).
const keys: Record<string, keyof Handlers> = {
  ArrowUp: "onUp",
  ArrowDown: "onDown",
  ArrowLeft: "onLeft",
  ArrowRight: "onRight",
  Enter: "onAccept",
  Escape: "onBack",
  Backspace: "onBack",
  KeyQ: "onTabPrev",
  KeyE: "onTabNext",
  BracketLeft: "onPrev",
  BracketRight: "onNext",
  PageUp: "onPrev",
  PageDown: "onNext",
};

export type Handlers = {
  // onPrev and onNext are the shoulder buttons: they switch pages wherever the
  // user is.
  onPrev?: () => void;
  onNext?: () => void;
  // The rest is the d-pad, which moves within the current page.
  onUp?: () => void;
  onDown?: () => void;
  onLeft?: () => void;
  onRight?: () => void;
  onAccept?: () => void;
  onBack?: () => void;
  // The triggers move between the tabs of the current page.
  onTabPrev?: () => void;
  onTabNext?: () => void;
};

// useGamepad calls the handlers when a controller button is pressed: LB/RB
// switch, the d-pad moves, A accepts and B goes back.
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
      if (edge(buttons.lb)) latest.current.onPrev?.();
      if (edge(buttons.rb)) latest.current.onNext?.();
      if (edge(buttons.up)) latest.current.onUp?.();
      if (edge(buttons.down)) latest.current.onDown?.();
      if (edge(buttons.left)) latest.current.onLeft?.();
      if (edge(buttons.right)) latest.current.onRight?.();
      if (edge(buttons.lt)) latest.current.onTabPrev?.();
      if (edge(buttons.rt)) latest.current.onTabNext?.();
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
