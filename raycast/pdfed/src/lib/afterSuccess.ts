import { Clipboard, getPreferenceValues, open } from "@raycast/api";

export type AfterSuccessMode = "none" | "copy" | "open" | "copy_and_open";

type Prefs = {
  afterSuccess?: string;
};

/**
 * Runs the extension preference action after a successful write (copy path, open in default app, or both).
 * Pass `null` when the primary output path is unknown (e.g. split with default cwd).
 */
export async function applyAfterSuccess(targetPath: string | null | undefined): Promise<void> {
  const raw = getPreferenceValues<Prefs>().afterSuccess ?? "none";
  const allowed: AfterSuccessMode[] = ["none", "copy", "open", "copy_and_open"];
  const mode = (allowed.includes(raw as AfterSuccessMode) ? raw : "none") as AfterSuccessMode;
  if (!targetPath?.trim() || mode === "none") return;

  const path = targetPath.trim();

  if (mode === "copy" || mode === "copy_and_open") {
    await Clipboard.copy(path);
  }
  if (mode === "open" || mode === "copy_and_open") {
    await open(path);
  }
}
