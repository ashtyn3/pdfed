import { homedir } from "os";

const PATH_EXTRA = ["/opt/homebrew/bin", "/usr/local/bin", "/usr/bin", "/bin"];

/** PATH + common binary locations so `pdfed` and Homebrew deps resolve under Raycast. */
export function buildSpawnEnv(): NodeJS.ProcessEnv {
  const parts = (process.env.PATH ?? "").split(":").filter(Boolean);
  const seen = new Set(parts);
  for (const p of PATH_EXTRA) {
    if (!seen.has(p)) {
      parts.push(p);
      seen.add(p);
    }
  }
  return {
    ...process.env,
    PATH: parts.join(":"),
    HOME: process.env.HOME ?? homedir(),
  };
}

/** If message looks like a macOS permission error, append a short hint. */
export function formatPdfedErrorMessage(raw: string): string {
  const lower = raw.toLowerCase();
  const perm =
    lower.includes("operation not permitted") ||
    lower.includes("permission denied") ||
    lower.includes("eacces") ||
    lower.includes("eperm");
  if (!perm) return raw;
  return (
    raw +
    "\n\nIf files are on Desktop/Documents or an external disk: enable Full Disk Access for Raycast in System Settings → Privacy & Security. " +
    "Also ensure the output folder is writable."
  );
}
