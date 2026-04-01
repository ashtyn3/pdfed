import { ApplicationMenu, BrowserView, BrowserWindow, Utils } from "electrobun/bun";
import { spawn } from "node:child_process";
import { dirname, isAbsolute, resolve } from "node:path";
import type { PdfedDesktopRPC } from "./types";

type PdfedResponse = { ok: boolean; [k: string]: unknown };
let activeRun: ReturnType<typeof spawn> | null = null;

const rpc = BrowserView.defineRPC<PdfedDesktopRPC>({
  // File picker and long-running PDF operations can exceed short RPC windows.
  maxRequestTime: 10 * 60_000,
  handlers: {
    requests: {
      pickPdf: async () => {
        const paths = await Utils.openFileDialog({
          allowedFileTypes: "pdf",
          canChooseFiles: true,
          allowsMultipleSelection: false,
          canChooseDirectory: false,
        });
        return { path: firstNonEmptyPath(paths) };
      },
      pickFiles: async ({ kind, multiple, canChooseDirectory, canChooseFiles }) => {
        const paths = await Utils.openFileDialog({
          allowedFileTypes: kind === "image" ? "png,jpg,jpeg,webp,gif,tiff,bmp" : "pdf",
          canChooseFiles: canChooseFiles ?? true,
          allowsMultipleSelection: multiple ?? true,
          canChooseDirectory: canChooseDirectory ?? false,
        });
        const cleaned = (paths ?? []).map((p) => p.trim()).filter((p) => p.length > 0);
        return { paths: cleaned };
      },
      pickOutputPdf: async ({ suggestedName }) => {
        const paths = await Utils.openFileDialog({
          allowedFileTypes: "pdf",
          canChooseFiles: false,
          allowsMultipleSelection: false,
          canChooseDirectory: true,
        });
        const folder = firstNonEmptyPath(paths);
        if (!folder) return { path: null };
        return { path: resolve(folder, suggestedName || "optimized.pdf") };
      },
      runPdfedArgs: async ({ args, cwd }) => runPdfed(args, cwd || process.cwd()),
      cancelRun: async () => {
        if (!activeRun) return { ok: true, cancelled: false };
        try {
          activeRun.kill("SIGTERM");
          return { ok: true, cancelled: true };
        } catch (err) {
          return { ok: false, cancelled: false, error: err instanceof Error ? err.message : String(err) };
        }
      },
      openPath: async ({ path }) => {
        try {
          const opened = Utils.openPath(path);
          if (!opened) return { ok: false, error: "Failed to open path." };
          return { ok: true };
        } catch (err) {
          return { ok: false, error: err instanceof Error ? err.message : String(err) };
        }
      },
      revealPath: async ({ path }) => {
        try {
          await spawnPlatformReveal(path);
          return { ok: true };
        } catch (err) {
          return { ok: false, error: err instanceof Error ? err.message : String(err) };
        }
      },
      copyText: async ({ text }) => {
        try {
          Utils.clipboardWriteText(text);
          return { ok: true };
        } catch (err) {
          return { ok: false, error: err instanceof Error ? err.message : String(err) };
        }
      },
      runInfo: async ({ input }) => runPdfed(["info", input], dirname(input)),
      runOptimize: async ({ input, output }) => {
        const args = ["optimize", input];
        if (output && output.trim().length > 0) args.push("-o", output);
        return runPdfed(args, dirname(input));
      },
    },
    messages: {},
  },
});

const mainWindow = new BrowserWindow({
  title: "pdfed",
  url: "views://mainview/index.html",
  frame: {
    width: 980,
    height: 700,
    x: 120,
    y: 120,
  },
  rpc,
});

ApplicationMenu.setApplicationMenu([
  {
    label: "pdfed",
    submenu: [
      { role: "about" },
      { type: "divider" },
      { role: "hide" },
      { role: "hideOthers" },
      { role: "showAll" },
      { type: "divider" },
      { role: "quit" },
    ],
  },
  {
    label: "Edit",
    submenu: [
      { role: "undo" },
      { role: "redo" },
      { type: "divider" },
      { role: "cut" },
      { role: "copy" },
      { role: "paste" },
      { role: "selectAll" },
    ],
  },
  {
    label: "Window",
    submenu: [{ role: "minimize" }, { role: "zoom" }, { role: "close" }],
  },
  {
    label: "Help",
    submenu: [{ label: "Keyboard Shortcuts", action: "show-shortcuts", accelerator: "CmdOrCtrl+/" }],
  },
]);

ApplicationMenu.on("application-menu-clicked", (event: unknown) => {
  const action = readMenuAction(event);
  if (action === "show-shortcuts") {
    void showShortcutDialog();
  }
});

mainWindow.on("close", () => {
  Utils.quit();
});

async function showShortcutDialog() {
  const shortcutsText = [
    "Enter - Run command",
    "I - Pick input PDF",
    "O - Pick output PDF (optimize mode)",
    "Cmd/Ctrl + K - Recent file picker",
    "Esc - Clear form",
    "1..8 - Switch operation",
    "[ / ] - Previous/next operation",
    "1 info, 2 optimize, 3 rotate, 4 split, 5 merge",
    "6 encrypt, 7 decrypt, 8 add-images",
    "Cmd/Ctrl + D - Toggle light/dark theme",
    "Cmd/Ctrl + / - Open this shortcuts dialog",
  ].join("\n");

  await Utils.showMessageBox({
    type: "info",
    title: "Keyboard Shortcuts",
    message: shortcutsText,
    buttons: ["OK"],
  });
}

async function runPdfed(args: string[], cwd: string) {
  const bin = process.env.PDFED_BIN || "pdfed";
  const fullArgs = ["--json", ...args];
  const { stdout, stderr, code } = await spawnCollect(bin, fullArgs, cwd);

  const text = stdout.trim();
  if (!text) {
    return { ok: false, error: stderr.trim() || `pdfed exited with code ${code}` };
  }

  try {
    const json = JSON.parse(text) as PdfedResponse;
    if (!json.ok) {
      return { ok: false, error: String(json.error ?? "pdfed reported failure"), json };
    }
    normalizeResultPaths(json, cwd);
    return { ok: true, json };
  } catch {
    return { ok: false, error: `Could not parse JSON: ${text.slice(0, 300)}` };
  }
}

function normalizeResultPaths(json: PdfedResponse, cwd: string) {
  const normalizeOne = (v: unknown): string | null => {
    if (typeof v !== "string") return null;
    const trimmed = v.trim();
    if (!trimmed) return null;
    return isAbsolute(trimmed) ? trimmed : resolve(cwd, trimmed);
  };

  const output = normalizeOne(json.output);
  if (output) json.output = output;

  const input = normalizeOne(json.input);
  if (input) json.input = input;

  const doc = json.document;
  if (doc && typeof doc === "object") {
    const docRec = doc as Record<string, unknown>;
    const file = normalizeOne(docRec.file);
    if (file) docRec.file = file;
  }

  if (Array.isArray(json.would_create)) {
    json.would_create = json.would_create
      .map((item) => normalizeOne(item) ?? item)
      .filter((item) => typeof item === "string");
  }
}

function spawnCollect(bin: string, args: string[], cwd: string) {
  return new Promise<{ stdout: string; stderr: string; code: number | null }>((resolvePromise, reject) => {
    const child = spawn(bin, args, { cwd, stdio: ["ignore", "pipe", "pipe"] });
    activeRun = child;
    let stdout = "";
    let stderr = "";
    child.stdout.on("data", (chunk) => (stdout += String(chunk)));
    child.stderr.on("data", (chunk) => (stderr += String(chunk)));
    child.on("error", (err) => {
      if (activeRun === child) activeRun = null;
      reject(err);
    });
    child.on("close", (code) => {
      if (activeRun === child) activeRun = null;
      resolvePromise({ stdout, stderr, code });
    });
  });
}

function firstNonEmptyPath(paths: string[] | undefined): string | null {
  if (!paths || paths.length === 0) return null;
  const first = paths[0]?.trim();
  return first && first.length > 0 ? first : null;
}

function readMenuAction(event: unknown): string | null {
  if (!event || typeof event !== "object") return null;
  const e = event as { data?: { action?: string }; action?: string };
  if (typeof e.action === "string") return e.action;
  if (typeof e.data?.action === "string") return e.data.action;
  return null;
}

async function spawnPlatformOpen(target: string) {
  const platform = process.platform;
  if (platform === "darwin") return spawnDetached("open", [target]);
  if (platform === "win32") return spawnDetached("cmd", ["/c", "start", "", target]);
  return spawnDetached("xdg-open", [target]);
}

async function spawnPlatformReveal(target: string) {
  const platform = process.platform;
  if (platform === "darwin") return spawnDetached("open", ["-R", target]);
  if (platform === "win32") return spawnDetached("explorer", ["/select,", target]);
  return spawnDetached("xdg-open", [dirname(target)]);
}

function spawnDetached(command: string, args: string[]) {
  return new Promise<void>((resolvePromise, reject) => {
    const child = spawn(command, args, { stdio: "ignore", detached: false });
    child.on("error", reject);
    child.on("close", (code) => {
      if (code === 0) resolvePromise();
      else reject(new Error(`${command} exited with code ${code}`));
    });
  });
}
