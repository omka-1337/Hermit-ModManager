// Directional focus for controller and arrow-key navigation: the page is not a
// single list, so the next element is chosen by where it sits on screen.

export type Direction = "up" | "down" | "left" | "right";

const focusable =
  'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])';

function candidates(root: HTMLElement): HTMLElement[] {
  return Array.from(root.querySelectorAll<HTMLElement>(focusable)).filter((el) => {
    if (el.hidden) return false;
    const rect = el.getBoundingClientRect();
    return rect.width > 0 && rect.height > 0;
  });
}

function put(el: HTMLElement): void {
  el.focus({ preventScroll: true });
  el.scrollIntoView({ block: "nearest", inline: "nearest" });
}

// focusFirst moves focus to where a page wants it: the control marked with
// data-focus-first, or else the first one. Returns false when the page has no
// controls, so the caller can stay where it is.
export function focusFirst(root: HTMLElement | null): boolean {
  if (!root) return false;
  const list = candidates(root);
  const target = list.find((el) => el.hasAttribute("data-focus-first")) ?? list[0];
  if (!target) return false;
  put(target);
  return true;
}

// moveFocus focuses the nearest control in the given direction, preferring the
// ones straight ahead over those off to the side. Returns false at the edge.
export function moveFocus(dir: Direction, root: HTMLElement | null): boolean {
  if (!root) return false;
  const list = candidates(root);
  if (list.length === 0) return false;

  const active = document.activeElement as HTMLElement | null;
  if (!active || !root.contains(active)) {
    put(list[0]);
    return true;
  }

  const from = active.getBoundingClientRect();
  let best: { el: HTMLElement; cost: number } | null = null;
  for (const el of list) {
    if (el === active) continue;
    const rect = el.getBoundingClientRect();
    const dx = rect.left + rect.width / 2 - (from.left + from.width / 2);
    const dy = rect.top + rect.height / 2 - (from.top + from.height / 2);
    const ahead = dir === "left" ? -dx : dir === "right" ? dx : dir === "up" ? -dy : dy;
    const aside = dir === "up" || dir === "down" ? Math.abs(dx) : Math.abs(dy);
    // Anything level with the current element, or mostly sideways, is not in
    // this direction at all.
    if (ahead <= 1 || aside > ahead * 2 + 48) continue;
    const cost = ahead + aside * 2;
    if (!best || cost < best.cost) best = { el, cost };
  }
  if (!best) return false;
  put(best.el);
  return true;
}

// clickFocused activates whatever the page has focused, the way A does.
export function clickFocused(root: HTMLElement | null): void {
  const active = document.activeElement as HTMLElement | null;
  if (active && root?.contains(active)) active.click();
}
