import { useEffect, useState } from "react";
import { IconService } from "../api";

// Icons are small data URLs read from Steam's cache; fetch each app only once.
// Resolved ones are kept separately so a remount can paint the icon on its
// first frame: going through the promise again would show the placeholder
// tile for a frame, which reads as a flicker when switching games.
const icons = new Map<string, Promise<string>>();
const loaded = new Map<string, string>();

function steamIcon(appId: string): Promise<string> {
  let icon = icons.get(appId);
  if (!icon) {
    icon = IconService.GetSteamIcon(appId)
      .catch(() => "")
      .then((src) => {
        loaded.set(appId, src);
        return src;
      });
    icons.set(appId, icon);
  }
  return icon;
}

const tileColors = ["bg-indigo-600", "bg-emerald-600", "bg-amber-600", "bg-rose-600", "bg-sky-600", "bg-violet-600"];

type Props = {
  name: string;
  steamAppId?: string;
  size: number;
  className?: string;
};

// GameIcon shows a game's Steam icon, or a colored tile with its initial.
export default function GameIcon({ name, steamAppId, size, className = "" }: Props) {
  const [src, setSrc] = useState(() => (steamAppId ? (loaded.get(steamAppId) ?? "") : ""));

  useEffect(() => {
    if (!steamAppId) {
      setSrc("");
      return;
    }
    const known = loaded.get(steamAppId);
    if (known !== undefined) {
      setSrc(known);
      return;
    }
    let active = true;
    setSrc("");
    steamIcon(steamAppId).then((s) => active && setSrc(s));
    return () => {
      active = false;
    };
  }, [steamAppId]);

  const style = { width: size, height: size };
  if (src) {
    return <img src={src} alt="" style={style} className={`shrink-0 rounded-lg object-cover ${className}`} />;
  }
  const color = tileColors[[...name].reduce((sum, c) => sum + c.charCodeAt(0), 0) % tileColors.length];
  return (
    <div
      style={{ ...style, fontSize: size * 0.45 }}
      className={`flex shrink-0 items-center justify-center rounded-lg font-semibold text-white ${color} ${className}`}
    >
      {[...name.trim()][0]?.toUpperCase() ?? "?"}
    </div>
  );
}
