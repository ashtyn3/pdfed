import { getPreferenceValues } from "@raycast/api";
import { accessSync, constants, existsSync, statSync } from "fs";
import { spawn, spawnSync } from "child_process";
import { join } from "path";

let cachedPath: string | undefined;
const jsonSupportCache = new Map<string, boolean>();

function isExecutableFile(filePath: string): boolean {
  try {
    if (!statSync(filePath).isFile()) return false;
    accessSync(filePath, constants.X_OK);
    return true;
  } catch {
    return false;
  }
}

/**
 * Resolves the pdfed binary: preference override, then common install locations, then PATH.
 */
export function getPdfedPath(): string {
  if (cachedPath) return cachedPath;

  const { pdfedPath: pref } = getPreferenceValues<{ pdfedPath?: string }>();
  const trimmed = pref?.trim();
  if (trimmed) {
    if (!isExecutableFile(trimmed)) {
      throw new Error(`pdfed not found or not executable at configured path: ${trimmed}`);
    }
    cachedPath = trimmed;
    return trimmed;
  }

  const candidates = ["/opt/homebrew/bin/pdfed", "/usr/local/bin/pdfed"];
  for (const c of candidates) {
    if (isExecutableFile(c)) {
      cachedPath = c;
      return c;
    }
  }

  const pathEnv = process.env.PATH ?? "";
  for (const dir of pathEnv.split(":")) {
    if (!dir) continue;
    const p = join(dir, "pdfed");
    if (isExecutableFile(p)) {
      cachedPath = p;
      return p;
    }
  }

  throw new Error(
    "Could not find pdfed. Install it (e.g. go install / Homebrew) or set the pdfed binary path in Raycast extension preferences.",
  );
}

export function resetPdfedPathCache(): void {
  cachedPath = undefined;
}

/**
 * Ensures the resolved pdfed binary supports the `--json` global flag.
 * Raycast commands depend on machine-readable output; older binaries will misparse arguments.
 */
export function assertPdfedSupportsJsonFlag(): void {
  const bin = getPdfedPath();
  const cached = jsonSupportCache.get(bin);
  if (cached === true) return;
  if (cached === false) {
    throw new Error(
      `The configured pdfed binary does not support --json: ${bin}\n` +
        "Install/update pdfed from this repo and set that binary path in Raycast extension preferences.",
    );
  }

  const probe = spawnSync(bin, ["--json", "info", "/__pdfed_json_probe_missing__.pdf"], {
    encoding: "utf8",
    env: { ...process.env, NO_COLOR: "1" },
  });

  const stdout = (probe.stdout ?? "").trim();
  let supportsJson = false;
  if (stdout.startsWith("{")) {
    try {
      const parsed = JSON.parse(stdout) as { ok?: unknown; error?: unknown };
      supportsJson = typeof parsed === "object" && parsed !== null && "ok" in parsed;
    } catch {
      supportsJson = false;
    }
  }

  // Fallback for binaries that may print JSON to stderr.
  if (!supportsJson) {
    const stderr = (probe.stderr ?? "").trim();
    if (stderr.startsWith("{")) {
      try {
        const parsed = JSON.parse(stderr) as { ok?: unknown; error?: unknown };
        supportsJson = typeof parsed === "object" && parsed !== null && "ok" in parsed;
      } catch {
        supportsJson = false;
      }
    }
  }

  jsonSupportCache.set(bin, supportsJson);
  if (!supportsJson) {
    throw new Error(
      `The configured pdfed binary does not support --json: ${bin}\n` +
        "Set Raycast's pdfed binary path to the binary you verified manually (or update/reinstall it).",
    );
  }
}

export type RunPdfedOptions = {
  cwd?: string;
};

/**
 * Runs pdfed with the given arguments (do not include `pdfed`; -q is not added automatically).
 * Captures stdout/stderr UTF-8. Rejects on non-zero exit with stderr (or stdout) message.
 */
export function runPdfed(args: string[], options: RunPdfedOptions = {}): Promise<{ stdout: string; stderr: string }> {
  const bin = getPdfedPath();
  return new Promise((resolve, reject) => {
    const child = spawn(bin, args, {
      cwd: options.cwd,
      stdio: ["ignore", "pipe", "pipe"],
      env: { ...process.env, NO_COLOR: "1" },
    });
    const outChunks: Buffer[] = [];
    const errChunks: Buffer[] = [];
    child.stdout?.on("data", (c: Buffer) => outChunks.push(c));
    child.stderr?.on("data", (c: Buffer) => errChunks.push(c));
    child.on("error", (err) => reject(err));
    child.on("close", (code) => {
      const stdout = Buffer.concat(outChunks).toString("utf8");
      const stderr = Buffer.concat(errChunks).toString("utf8");
      if (code === 0) {
        resolve({ stdout, stderr });
      } else {
        const msg = (stderr || stdout || `pdfed exited with code ${code}`).trim();
        reject(new Error(msg));
      }
    });
  });
}

export function ensurePdfFiles(paths: string[], label = "PDF"): void {
  const ok = paths.filter((p) => existsSync(p) && statSync(p).isFile() && p.toLowerCase().endsWith(".pdf"));
  if (ok.length !== paths.length) {
    throw new Error(`Select valid ${label} files that still exist on disk.`);
  }
}

export function ensureImageFiles(paths: string[]): void {
  const exts = [".jpg", ".jpeg", ".png", ".webp", ".tif", ".tiff"];
  const ok = paths.filter((p) => {
    if (!existsSync(p) || !statSync(p).isFile()) return false;
    const low = p.toLowerCase();
    return exts.some((e) => low.endsWith(e));
  });
  if (ok.length !== paths.length) {
    throw new Error("Select valid image files (JPEG, PNG, WebP, TIFF) that still exist on disk.");
  }
}
