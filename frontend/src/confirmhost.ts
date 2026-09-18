import { Dialogs } from "@wailsio/runtime";

// Questions asked before an action. This module has no other imports on
// purpose: api.ts pulls it in, so anything heavier would import api back.

export type Question = {
  title: string;
  message: string;
  action: string;
  // danger marks an irreversible action: Cancel is then the safe default.
  danger?: boolean;
};

let host: ((q: Question) => Promise<boolean>) | null = null;

// setConfirmHost routes questions to a dialog drawn inside the window. Without
// one (and on the desktop) they go to the system dialog.
export function setConfirmHost(fn: ((q: Question) => Promise<boolean>) | null): void {
  host = fn;
}

export function ask(question: Question): Promise<boolean> {
  if (host) return host(question);
  return Dialogs.Question({
    Title: question.title,
    Message: question.message,
    Buttons: question.danger
      ? [{ Label: question.action }, { Label: "Cancel", IsCancel: true, IsDefault: true }]
      : [{ Label: question.action, IsDefault: true }, { Label: "Cancel", IsCancel: true }],
  }).then((answer) => answer === question.action);
}
