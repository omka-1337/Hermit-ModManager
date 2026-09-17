import { Dialogs } from "@wailsio/runtime";

export { Library, Runtime } from "../bindings/bepinexmodmanager/internal/library";
export type { Game, GameCandidate, Profile } from "../bindings/bepinexmodmanager/internal/library";
export { Store as SettingsStore } from "../bindings/bepinexmodmanager/internal/settings";
export { InfoService } from "../bindings/bepinexmodmanager/internal/app";
export type { AppInfo } from "../bindings/bepinexmodmanager/internal/app";

export function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

// confirmDanger asks before an irreversible action; resolves true if confirmed.
export async function confirmDanger(title: string, message: string, action: string): Promise<boolean> {
  const answer = await Dialogs.Question({
    Title: title,
    Message: message,
    Buttons: [{ Label: action }, { Label: "Cancel", IsCancel: true, IsDefault: true }],
  });
  return answer === action;
}
