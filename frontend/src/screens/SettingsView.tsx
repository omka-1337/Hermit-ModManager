import { useEffect, useState } from "react";
import { errorMessage, Settings, SettingsStore } from "../api";
import { ErrorText } from "../components/ui";

export default function SettingsView() {
  const [settings, setSettings] = useState<Settings | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    SettingsStore.Get()
      .then(setSettings)
      .catch((err) => setError(errorMessage(err)));
  }, []);

  const update = async (patch: Partial<Settings>) => {
    if (!settings) return;
    setError("");
    try {
      setSettings(await SettingsStore.Update({ ...settings, ...patch }));
    } catch (err) {
      setError(errorMessage(err));
    }
  };

  return (
    <div className="mx-auto flex max-w-3xl flex-col gap-6 p-8">
      <h1 className="text-2xl font-semibold">Settings</h1>
      <ErrorText>{error}</ErrorText>
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
    </div>
  );
}
