import { useCallback, useEffect, useState } from "react";
import { Clipboard } from "@wailsio/runtime";
import { confirm, errorMessage, Game, LaunchInfo, LaunchService } from "../api";
import { Button, ErrorText } from "./ui";

// Both the running game and the Steam launch options change behind our back:
// the game is started and closed outside the manager, and Steam writes its
// config with a delay. Polling keeps the page honest without a refresh.
const launchPollMs = 3000;

export function useLaunchInfo(gameId: string) {
  const [info, setInfo] = useState<LaunchInfo | null>(null);
  const reload = useCallback(() => {
    LaunchService.GetLaunchInfo(gameId).then(setInfo).catch(console.error);
  }, [gameId]);

  useEffect(() => {
    reload();
    const timer = setInterval(() => {
      // Nothing to show while the window is hidden.
      if (!document.hidden) reload();
    }, launchPollMs);
    const onVisible = () => {
      if (!document.hidden) reload();
    };
    document.addEventListener("visibilitychange", onVisible);
    return () => {
      clearInterval(timer);
      document.removeEventListener("visibilitychange", onVisible);
    };
  }, [reload]);

  return { info, reload };
}

type PlayProps = {
  game: Game;
  profileId: string;
  label?: string;
  // onPlayed runs after the game was started; the active profile may have changed.
  onPlayed?: () => void;
};

export function PlayButton({ game, profileId, label = "Play", onPlayed }: PlayProps) {
  const { info, reload } = useLaunchInfo(game.id);
  const [error, setError] = useState("");
  const [starting, setStarting] = useState(false);

  const play = async () => {
    setError("");
    try {
      const fresh = await LaunchService.GetLaunchInfo(game.id);
      if (!fresh.configured) {
        const ok = await confirm(
          "Launch options not found",
          "Mods load only when Steam runs the game through the mod manager. Set the launch options shown on the game page.\n\n" +
            "Steam may not have saved them to disk yet. Launch anyway?",
          "Launch",
        );
        if (!ok) return;
      }
      setStarting(true);
      await LaunchService.Play(game.id, profileId);
      onPlayed?.();
      // Steam takes a few seconds to start the game.
      setTimeout(() => {
        setStarting(false);
        reload();
      }, 8000);
    } catch (err) {
      setStarting(false);
      setError(errorMessage(err));
    }
  };

  const disabled = !info?.supported || info.running || starting;
  return (
    <div className="flex flex-col items-end gap-1">
      <Button variant="primary" disabled={disabled} onClick={play} title={info?.supported ? "" : info?.reason}>
        {info?.running ? "Running" : starting ? "Starting…" : label}
      </Button>
      <ErrorText>{error}</ErrorText>
    </div>
  );
}

export function LaunchSetup({ game }: { game: Game }) {
  const { info, reload } = useLaunchInfo(game.id);
  const [copied, setCopied] = useState(false);

  if (info && !info.supported) {
    return <p className="text-sm text-zinc-400">{info.reason}</p>;
  }

  // While the info is being read the card is drawn as it will look, so the
  // page keeps its shape instead of collapsing on every game switch.
  const shown: LaunchInfo = info ?? { supported: true, configured: false, running: false, runningProfile: "", reason: "", launchOptions: "" };

  const copy = async () => {
    await Clipboard.SetText(shown.launchOptions);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className={`flex flex-col gap-3 rounded-lg border border-zinc-800 p-4 ${info ? "" : "opacity-60"}`}>
      <div className="flex items-center justify-between gap-3">
        <span className="text-sm font-medium">Steam launch options</span>
        {!info ? (
          <span className="text-xs text-zinc-500">Checking…</span>
        ) : shown.configured ? (
          <span className="text-xs text-emerald-400">✓ Set in Steam</span>
        ) : (
          <span className="text-xs text-amber-400">Not detected</span>
        )}
      </div>
      <p className="text-xs text-zinc-400">
        In Steam open the game&apos;s Properties → General → Launch Options and paste this. The active profile is then
        loaded whenever the game starts, including from Steam directly. The game folder is left untouched: Proton
        games get links to the profile only while running, native Linux games load BepInEx straight from the
        profile.
      </p>
      <div className="flex gap-2">
        <code className="flex-1 truncate rounded-md bg-zinc-900 px-3 py-1.5 font-mono text-xs text-zinc-300 select-text">
          {shown.launchOptions}
        </code>
        <Button disabled={!info} onClick={copy}>
          {copied ? "Copied" : "Copy"}
        </Button>
        {!shown.configured && (
          <Button variant="ghost" disabled={!info} onClick={reload}>
            Check again
          </Button>
        )}
      </div>
      {!shown.configured && (
        <p className="text-xs text-zinc-500">Steam saves launch options with a delay, sometimes only when it exits.</p>
      )}
    </div>
  );
}
