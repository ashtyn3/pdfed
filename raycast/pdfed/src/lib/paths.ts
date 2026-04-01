import { homedir } from "os";
import { isAbsolute, resolve } from "path";

/** Expands leading `~` to the user home directory. */
export function expandUserPath(p: string): string {
  const t = p.trim();
  if (t === "" || t === "~") return homedir();
  if (t.startsWith("~/")) return resolve(homedir(), t.slice(2));
  return t;
}

/** Absolute path suitable for pdfed and spawn cwd. */
export function resolveUserPath(p: string, baseDir?: string): string {
  const e = expandUserPath(p);
  if (isAbsolute(e)) return e;
  return resolve(baseDir ?? homedir(), e);
}
