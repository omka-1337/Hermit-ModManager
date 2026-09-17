import { CancelError, Dialogs } from "@wailsio/runtime";

export { Backend, Library, Runtime } from "../bindings/bepinexmodmanager/internal/library";
export type { Game, GameCandidate, Mod, Profile } from "../bindings/bepinexmodmanager/internal/library";
export { Store as SettingsStore } from "../bindings/bepinexmodmanager/internal/settings";
export type { Settings } from "../bindings/bepinexmodmanager/internal/settings";
export {
  BrowseService,
  IconService,
  InfoService,
  InstallService,
  LaunchService,
  ShareService,
} from "../bindings/bepinexmodmanager/internal/app";
export type { ImportProgress, ImportResult, Preview } from "../bindings/bepinexmodmanager/internal/profileshare";
export type { LaunchInfo } from "../bindings/bepinexmodmanager/internal/app";
export { ConflictReason, Stage } from "../bindings/bepinexmodmanager/internal/modinstall";
export type { Conflict, InstallPlan, Progress } from "../bindings/bepinexmodmanager/internal/modinstall";
export { IssueKind } from "../bindings/bepinexmodmanager/internal/launch";
export type { Issue, Report } from "../bindings/bepinexmodmanager/internal/launch";
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

// confirm asks a yes/no question; resolves true if the action button was chosen.
export async function confirm(title: string, message: string, action: string): Promise<boolean> {
  const answer = await Dialogs.Question({
    Title: title,
    Message: message,
    Buttons: [{ Label: action, IsDefault: true }, { Label: "Cancel", IsCancel: true }],
  });
  return answer === action;
}
