import { CancelError, Dialogs } from "@wailsio/runtime";

export { Backend, Library, Runtime } from "../bindings/hermit/internal/library";
export type { Game, GameCandidate, Mod, Profile } from "../bindings/hermit/internal/library";
export { Store as SettingsStore } from "../bindings/hermit/internal/settings";
export { UIMode } from "../bindings/hermit/internal/settings";
export type { Settings } from "../bindings/hermit/internal/settings";
export {
  BrowseService,
  ConfigService,
  IconService,
  InfoService,
  InstallService,
  LaunchService,
  ShareService,
} from "../bindings/hermit/internal/app";
export type { ImportProgress, ImportResult, Preview } from "../bindings/hermit/internal/profileshare";
export type { ConfigContent } from "../bindings/hermit/internal/app";
export type { Change, Entry, FileInfo as ConfigFileInfo } from "../bindings/hermit/internal/configs";
export type { LaunchInfo } from "../bindings/hermit/internal/app";
export { ConflictReason, Stage } from "../bindings/hermit/internal/modinstall";
export type {
  Conflict,
  InstallPlan,
  LocalPackage,
  Progress,
  UpdateResult,
} from "../bindings/hermit/internal/modinstall";
export type { GitHubRepo } from "../bindings/hermit/internal/app";
export type { Asset as GitHubAsset, Release as GitHubRelease } from "../bindings/hermit/internal/github";
export { IssueKind } from "../bindings/hermit/internal/launch";
export type { Issue, Report } from "../bindings/hermit/internal/launch";
export { Ordering } from "../bindings/hermit/internal/thunderstore";
export type {
  Community,
  Filters,
  PackageDetail,
  PackageList,
  PackageSummary,
} from "../bindings/hermit/internal/thunderstore";
export type { AppInfo } from "../bindings/hermit/internal/app";

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
