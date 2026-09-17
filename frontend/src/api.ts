import { CancelError, Dialogs } from "@wailsio/runtime";

export { Backend, Library, Runtime } from "../bindings/bepinexmodmanager/internal/library";
export type { Game, GameCandidate, Profile } from "../bindings/bepinexmodmanager/internal/library";
export { Store as SettingsStore } from "../bindings/bepinexmodmanager/internal/settings";
export type { Settings } from "../bindings/bepinexmodmanager/internal/settings";
export { BrowseService, InfoService } from "../bindings/bepinexmodmanager/internal/app";
export { Ordering } from "../bindings/bepinexmodmanager/internal/thunderstore";
export type {
  Community,
  Filters,
  PackageDetail,
  PackageList,
  PackageSummary,
} from "../bindings/bepinexmodmanager/internal/thunderstore";
export type { AppInfo } from "../bindings/bepinexmodmanager/internal/app";

// isCancelled reports whether err comes from cancelling a CancellablePromise.
export function isCancelled(err: unknown): boolean {
  return err instanceof CancelError;
}

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
