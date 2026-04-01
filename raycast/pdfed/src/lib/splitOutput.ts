import { basename, join } from "path";

/**
 * Best-effort prediction of pdfed split output path (matches cmd/split.go).
 * Returns null when the file path depends on the process working directory.
 */
export function predictSplitOutputPath(
  inputPath: string,
  splitMode: string,
  ranges: string,
  outputField: string,
): string | null {
  const base = basename(inputPath);
  const baseName = base.replace(/\.pdf$/i, "");
  const rangeStr = ranges.trim();
  const sanitized = rangeStr.replace(/,/g, "_");

  if (splitMode === "extractAll") {
    const outDir = outputField.trim() || ".";
    if (outDir === ".") return null;
    return outDir;
  }

  const out = outputField.trim();
  if (!out) return null;

  if (out.toLowerCase().endsWith(".pdf")) {
    return out;
  }
  return join(out, `${baseName}_pages_${sanitized}.pdf`);
}
