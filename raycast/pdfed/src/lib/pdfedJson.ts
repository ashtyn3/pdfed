import { homedir } from "os";
import { spawn } from "child_process";
import { isAbsolute, resolve as resolvePath } from "path";
import { assertPdfedSupportsJsonFlag, getPdfedPath } from "./pdfed";
import { buildSpawnEnv } from "./spawnEnv";

export type PdfedJSONSuccess = { ok: true; command: string } & Record<string, unknown>;
export type PdfedJSONError = { ok: false; error: string };

export type RunPdfedJSONOptions = {
  /**
   * Working directory for the pdfed process. Use the directory of the main input/output file
   * so default relative paths from pdfed land next to the document (Raycast’s cwd is often not writable).
   */
  cwd: string;
};

function parseStdout(stdout: string): unknown {
  const t = stdout.trim();
  if (!t) return null;
  return JSON.parse(t) as unknown;
}

/**
 * Runs pdfed with `--json` prepended. Resolves with the parsed success object, or rejects with a clear error.
 */
export function runPdfedJSON(args: string[], options: RunPdfedJSONOptions): Promise<PdfedJSONSuccess> {
  assertPdfedSupportsJsonFlag();
  const bin = getPdfedPath();
  const fullArgs = ["--json", ...args];
  const cwd = options.cwd && options.cwd.length > 0 ? options.cwd : homedir();

  return new Promise((resolve, reject) => {
    const child = spawn(bin, fullArgs, {
      cwd,
      stdio: ["ignore", "pipe", "pipe"],
      env: buildSpawnEnv(),
    });
    const outChunks: Buffer[] = [];
    const errChunks: Buffer[] = [];
    child.stdout?.on("data", (c: Buffer) => outChunks.push(c));
    child.stderr?.on("data", (c: Buffer) => errChunks.push(c));
    child.on("error", (err) => reject(err));
    child.on("close", (code) => {
      const stdout = Buffer.concat(outChunks).toString("utf8");
      const stderr = Buffer.concat(errChunks).toString("utf8");
      const tryRejectFromStdout = (): boolean => {
        try {
          const obj = parseStdout(stdout) as PdfedJSONError | null;
          if (obj && typeof obj === "object" && obj.ok === false && typeof obj.error === "string") {
            reject(new Error(obj.error));
            return true;
          }
        } catch {
          /* ignore */
        }
        return false;
      };

      if (code === 0) {
        try {
          const obj = parseStdout(stdout) as PdfedJSONSuccess | PdfedJSONError | null;
          if (!obj || typeof obj !== "object") {
            reject(new Error("pdfed returned empty or invalid JSON"));
            return;
          }
          if ("ok" in obj && obj.ok === false) {
            reject(new Error(typeof obj.error === "string" ? obj.error : "pdfed error"));
            return;
          }
          if (obj.ok !== true) {
            reject(new Error("pdfed JSON missing ok: true"));
            return;
          }
          const normalized = { ...obj } as PdfedJSONSuccess;
          if (typeof normalized.output === "string" && normalized.output.trim()) {
            const rawOut = normalized.output.trim();
            normalized.output = isAbsolute(rawOut) ? rawOut : resolvePath(cwd, rawOut);
          }
          resolve(normalized);
        } catch {
          reject(new Error(`Could not parse pdfed JSON: ${stdout.slice(0, 300)}`));
        }
      } else {
        if (!tryRejectFromStdout()) {
          reject(new Error((stderr || stdout || `pdfed exited with code ${code}`).trim()));
        }
      }
    });
  });
}

/** Primary path for copy / open-in-app preferences. */
export function outputPathFromResult(r: PdfedJSONSuccess): string | null {
  const o = r.output;
  if (typeof o === "string" && o.trim()) return o.trim();
  return null;
}

/** Human-oriented summary for Toast (plain text). */
export function summarizePdfedResult(r: PdfedJSONSuccess): string {
  const lines: string[] = [];
  switch (r.command) {
    case "merge": {
      if (r.dry_run) {
        lines.push(`Dry run — would write: ${String(r.output)}`);
        lines.push(`${String(r.input_count)} inputs, ${String(r.page_count)} pages`);
      } else {
        lines.push(`Output: ${String(r.output)}`);
        lines.push(`Size: ${String(r.size_human)} · ${String(r.page_count)} pages · ${String(r.input_count)} files`);
      }
      break;
    }
    case "optimize": {
      if (r.dry_run) {
        lines.push(`Dry run — would optimize → ${String(r.output)}`);
      } else {
        lines.push(`Output: ${String(r.output)}`);
        lines.push(`${String(r.input_size_human)} → ${String(r.output_size_human)} (saved ${String(r.saved_human)}, ${Number(r.saved_percent).toFixed(1)}%)`);
      }
      break;
    }
    case "rotate": {
      if (r.dry_run) {
        lines.push(`Dry run — would rotate ${String(r.degrees)}° → ${String(r.output)}`);
      } else {
        lines.push(`Output: ${String(r.output)} (${String(r.size_human)})`);
        lines.push(`${String(r.degrees)}°${r.pages ? ` · pages ${String(r.pages)}` : ""}`);
      }
      break;
    }
    case "split": {
      if (r.dry_run) {
        lines.push(`Dry run — ${String(r.mode)}`);
        lines.push(`Would write: ${String(r.output)}`);
      } else if (r.mode === "extract_all") {
        lines.push(`Extracted ${String(r.page_count)} pages → ${String(r.output)}`);
      } else {
        lines.push(`Output: ${String(r.output)}`);
        lines.push(`Size: ${String(r.size_human)} · ${String(r.pdf_page_count)} PDF pages`);
      }
      break;
    }
    case "add-images": {
      lines.push(`Output: ${String(r.output)}`);
      lines.push(`Size: ${String(r.size_human)} · ${String(r.image_count)} image(s)`);
      break;
    }
    case "info": {
      const doc = r.document as Record<string, unknown> | undefined;
      lines.push(`Pages: ${doc ? String(doc.page_count) : "?"}`);
      lines.push(`Version: ${doc ? String(doc.pdf_version) : "?"}`);
      lines.push(`Size: ${doc ? String(doc.size_human) : "?"}`);
      break;
    }
    default:
      lines.push(JSON.stringify(r, null, 2));
  }
  return lines.join("\n");
}

function mdRow(label: string, value: string): string {
  return `| ${label} | ${value.replace(/\|/g, "\\|")} |\n`;
}

/** Markdown for Detail view (tables only). */
export function markdownFromPdfedResult(r: PdfedJSONSuccess): string {
  switch (r.command) {
    case "info":
      return markdownFromInfoResult(r);
    case "merge":
      return markdownFromMergeResult(r);
    case "optimize":
      return markdownFromOptimizeResult(r);
    case "rotate":
      return markdownFromRotateResult(r);
    case "split":
      return markdownFromSplitResult(r);
    case "add-images":
      return markdownFromAddImagesResult(r);
    default:
      return "## Result\n\nCompleted successfully.";
  }
}

function markdownFromMergeResult(r: PdfedJSONSuccess): string {
  let md = "## Merge\n\n| Field | Value |\n| --- | --- |\n";
  md += mdRow("Dry run", r.dry_run ? "Yes" : "No");
  if (r.output != null) md += mdRow("Output", String(r.output));
  if (r.size_human != null) md += mdRow("Size", String(r.size_human));
  if (r.size_bytes != null) md += mdRow("Size (bytes)", String(r.size_bytes));
  if (r.input_count != null) md += mdRow("Input PDFs", String(r.input_count));
  if (r.page_count != null) md += mdRow("Total pages", String(r.page_count));
  if (Array.isArray(r.inputs)) {
    md += "\n### Input files\n\n";
    for (const p of r.inputs as string[]) {
      md += `- \`${p}\`\n`;
    }
  }
  return md;
}

function markdownFromOptimizeResult(r: PdfedJSONSuccess): string {
  let md = "## Optimize\n\n| Field | Value |\n| --- | --- |\n";
  md += mdRow("Dry run", r.dry_run ? "Yes" : "No");
  if (r.input != null) md += mdRow("Input", String(r.input));
  if (r.output != null) md += mdRow("Output", String(r.output));
  if (r.input_size_human != null) md += mdRow("Before", String(r.input_size_human));
  if (r.output_size_human != null) md += mdRow("After", String(r.output_size_human));
  if (r.saved_human != null) md += mdRow("Saved", String(r.saved_human));
  if (r.saved_percent != null) md += mdRow("Saved %", `${Number(r.saved_percent).toFixed(1)}%`);
  return md;
}

function markdownFromRotateResult(r: PdfedJSONSuccess): string {
  let md = "## Rotate\n\n| Field | Value |\n| --- | --- |\n";
  md += mdRow("Dry run", r.dry_run ? "Yes" : "No");
  if (r.input != null) md += mdRow("Input", String(r.input));
  if (r.output != null) md += mdRow("Output", String(r.output));
  if (r.degrees != null) md += mdRow("Degrees", String(r.degrees));
  if (r.pages != null) md += mdRow("Pages", String(r.pages));
  md += mdRow("In place", r.in_place === true ? "Yes" : "No");
  if (r.size_human != null) md += mdRow("Size", String(r.size_human));
  return md;
}

function markdownFromSplitResult(r: PdfedJSONSuccess): string {
  let md = "## Split\n\n| Field | Value |\n| --- | --- |\n";
  md += mdRow("Dry run", r.dry_run ? "Yes" : "No");
  if (r.mode != null) md += mdRow("Mode", String(r.mode));
  if (r.input != null) md += mdRow("Input", String(r.input));
  if (r.output != null) md += mdRow("Output", String(r.output));
  if (r.range != null) md += mdRow("Range", String(r.range));
  if (r.page_count != null) md += mdRow("Page count", String(r.page_count));
  if (r.pdf_page_count != null) md += mdRow("PDF pages extracted", String(r.pdf_page_count));
  if (r.size_human != null) md += mdRow("Output size", String(r.size_human));
  if (Array.isArray(r.pdf_pages)) {
    md += "\n### PDF page indices\n\n`" + (r.pdf_pages as number[]).join(", ") + "`\n";
  }
  if (Array.isArray(r.would_create)) {
    md += "\n### Would create\n\n";
    for (const p of r.would_create as string[]) {
      md += `- \`${p}\`\n`;
    }
  }
  return md;
}

function markdownFromAddImagesResult(r: PdfedJSONSuccess): string {
  let md = "## Add images\n\n| Field | Value |\n| --- | --- |\n";
  if (r.output != null) md += mdRow("Output", String(r.output));
  if (r.image_count != null) md += mdRow("Images", String(r.image_count));
  if (r.appended != null) md += mdRow("Appended to existing", r.appended === true ? "Yes" : "No");
  if (r.size_human != null) md += mdRow("Size", String(r.size_human));
  return md;
}

function markdownFromInfoResult(r: PdfedJSONSuccess): string {
  const doc = (r.document ?? {}) as Record<string, unknown>;
  const meta = (r.metadata ?? {}) as Record<string, unknown>;
  const feat = (r.features ?? {}) as Record<string, unknown>;
  const str = (k: string, o: Record<string, unknown>) => (o[k] != null && o[k] !== "" ? String(o[k]) : "—");
  const bool = (k: string, o: Record<string, unknown>) => (o[k] === true ? "Yes" : o[k] === false ? "No" : "—");

  let md = "## Document\n\n";
  md += "| Field | Value |\n| --- | --- |\n";
  md += mdRow("File", str("file", doc));
  md += mdRow("Size", str("size_human", doc));
  md += mdRow("PDF version", str("pdf_version", doc));
  md += mdRow("Pages", str("page_count", doc));
  if (doc.page_width_pt != null) {
    md += mdRow(
      "Page size",
      `${str("page_width_pt", doc)} × ${str("page_height_pt", doc)} pt (${Number(doc.page_width_mm).toFixed(1)} × ${Number(doc.page_height_mm).toFixed(1)} mm)`,
    );
  }
  md += "\n## Metadata\n\n| Field | Value |\n| --- | --- |\n";
  md += mdRow("Title", str("title", meta));
  md += mdRow("Author", str("author", meta));
  md += mdRow("Subject", str("subject", meta));
  md += mdRow("Creator", str("creator", meta));
  md += mdRow("Producer", str("producer", meta));
  md += mdRow("Created", str("created", meta));
  md += mdRow("Modified", str("modified", meta));
  const kw = meta.keywords;
  md += mdRow("Keywords", Array.isArray(kw) ? (kw as string[]).join(", ") : "—");
  md += "\n## Features\n\n| Feature | |\n| --- | --- |\n";
  md += mdRow("Encrypted", bool("encrypted", feat));
  md += mdRow("Linearized", bool("linearized", feat));
  md += mdRow("Tagged", bool("tagged", feat));
  md += mdRow("Watermarked", bool("watermarked", feat));
  md += mdRow("Outlines", bool("outlines", feat));
  md += mdRow("Form fields", bool("form_fields", feat));
  md += mdRow("Signatures", bool("signatures", feat));
  md += mdRow("Attachments", bool("has_attachments", feat));
  return md;
}
