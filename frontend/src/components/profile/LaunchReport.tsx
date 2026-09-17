import { useState } from "react";
import { Issue, IssueKind, Report } from "../../api";
import { formatAgo } from "../../format";

export function issueText(issue: Issue): string {
  switch (issue.kind) {
    case IssueKind.IssueIncompatible:
      return `Not loaded: incompatible with ${issue.detail}`;
    case IssueKind.IssueMissingDependencies:
      return `Not loaded: missing ${issue.detail}`;
    case IssueKind.IssueDependencyNotLoaded:
      return "Not loaded: a dependency failed to load";
    case IssueKind.IssueNewerVersionExists:
      return `Skipped: a newer copy is loaded (${issue.detail})`;
    case IssueKind.IssueProcessFilter:
      return "Skipped: not meant for this game process";
    default:
      return `Failed to load: ${issue.detail}`;
  }
}

// LaunchReport summarises the last game session of the profile.
export default function LaunchReport({ report }: { report: Report }) {
  const [expanded, setExpanded] = useState(false);
  const issues = report.issues ?? [];
  const when = formatAgo(report.finishedAt);

  if (!report.bepinexStarted) {
    return (
      <div className="mb-3 rounded-lg border border-amber-500/30 bg-amber-500/5 px-4 py-3 text-sm">
        <p className="text-amber-400">BepInEx did not start during the last launch ({when}).</p>
        <p className="mt-1 text-xs text-zinc-400">
          Check that the Steam launch options are set and that the game was started after they were saved.
        </p>
      </div>
    );
  }

  const tone = issues.length ? "border-amber-500/30 bg-amber-500/5" : "border-zinc-800";
  return (
    <div className={`mb-3 rounded-lg border px-4 py-3 text-sm ${tone}`}>
      <div className="flex items-center justify-between gap-3">
        <span>
          Last launch {when}: {report.loaded?.length ?? 0} plugins loaded
          {issues.length > 0 && <span className="text-amber-400">, {issues.length} not loaded</span>}
          <span className="text-zinc-500"> · BepInEx {report.bepinexVersion}</span>
        </span>
        {issues.length > 0 && (
          <button className="text-xs text-zinc-400 hover:text-zinc-200" onClick={() => setExpanded(!expanded)}>
            {expanded ? "Hide" : "Details"}
          </button>
        )}
      </div>
      {expanded && (
        <ul className="mt-2 flex flex-col gap-1 text-xs select-text">
          {issues.map((issue, i) => (
            <li key={i}>
              <span className="text-zinc-200">{issue.plugin}</span>{" "}
              <span className="text-zinc-400">{issueText(issue)}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
