import { useEffect, useState } from "react";
import { AppInfo, errorMessage, InfoService, Settings, SettingsStore, UIMode } from "../api";
import { Browser } from "@wailsio/runtime";
import { Button, ErrorText } from "../components/ui";
import { useUIMode } from "../uimode";

const repository = "https://github.com/omka-1337/Hermit-ModManager";

export default function SettingsView() {
  const [settings, setSettings] = useState<Settings | null>(null);
  const [error, setError] = useState("");
  const [info, setInfo] = useState<AppInfo | null>(null);
  const { preference, layout, steamDeck, setPreference } = useUIMode();

  useEffect(() => {
    InfoService.GetInfo().then(setInfo).catch(console.error);
    SettingsStore.Get()
      .then(setSettings)
      .catch((err) => setError(errorMessage(err)));
  }, []);

  const act = async (action: () => Promise<void>) => {
    setError("");
    try {
      await action();
    } catch (err) {
      setError(errorMessage(err));
    }
  };

  const update = (patch: Partial<Settings>) =>
    act(async () => {
      if (settings) setSettings(await SettingsStore.Update({ ...settings, ...patch }));
    });

  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-6 p-8">
      <h1 className="text-2xl font-semibold">Settings</h1>
      <ErrorText>{error}</ErrorText>

      <section className="flex flex-col gap-3">
        <h2 className="text-sm font-semibold tracking-wide text-zinc-400 uppercase">Interface</h2>
        <div className="flex flex-col gap-2 rounded-lg border border-zinc-800 px-4 py-3">
          <span className="text-sm font-medium">Layout</span>
          <span className="text-xs text-zinc-500">
            The Steam Deck layout scales everything up for the handheld screen and shows package details over the
            list instead of beside it. Some changes need a restart to resize the window.
          </span>
          <div className="mt-1 flex gap-2">
            {[
              { mode: UIMode.UIModeAuto, label: `Automatic (${steamDeck ? "Steam Deck" : "desktop"})` },
              { mode: UIMode.UIModeDesktop, label: "Desktop" },
              { mode: UIMode.UIModeDeck, label: "Steam Deck" },
            ].map((option) => (
              <Button
                key={option.mode}
                variant={preference === option.mode ? "primary" : "secondary"}
                onClick={() => act(() => setPreference(option.mode))}
              >
                {option.label}
              </Button>
            ))}
          </div>
          <span className="text-xs text-zinc-500">Now using the {layout === "deck" ? "Steam Deck" : "desktop"} layout.</span>
        </div>
      </section>
      {settings && (
        <section className="flex flex-col gap-3">
          <h2 className="text-sm font-semibold tracking-wide text-zinc-400 uppercase">Mod browser</h2>
          <label className="flex cursor-pointer items-start gap-3 rounded-lg border border-zinc-800 px-4 py-3">
            <input
              type="checkbox"
              className="mt-1 accent-indigo-500"
              checked={settings.allowNsfw}
              onChange={(e) => update({ allowNsfw: e.target.checked })}
            />
            <span className="flex flex-col gap-0.5">
              <span className="text-sm font-medium">Allow NSFW content</span>
              <span className="text-xs text-zinc-500">Show packages marked as NSFW when browsing Thunderstore.</span>
            </span>
          </label>
        </section>
      )}
      {info && (
        <section className="flex flex-col gap-1 text-sm">
          <h2 className="text-sm font-semibold tracking-wide text-zinc-400 uppercase">About</h2>
          <p>{info.name}</p>
          <p className="text-zinc-500">
            v{info.version} · {info.os}/{info.arch}
          </p>
          <p className="text-xs text-zinc-500">
            Copyright (C) 2026 Omka. Free software under the{" "}
            <button
              className="text-indigo-400 hover:underline"
              onClick={() => Browser.OpenURL("https://www.gnu.org/licenses/gpl-3.0.html")}
            >
              GNU GPL v3
            </button>{" "}
            or later, with no warranty. Source:{" "}
            <button
              className="text-indigo-400 hover:underline"
              onClick={() => Browser.OpenURL(repository)}
            >
              {repository.replace("https://", "")}
            </button>
          </p>
        </section>
      )}
    </div>
  );
}
