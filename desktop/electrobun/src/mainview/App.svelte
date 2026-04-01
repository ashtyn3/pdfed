<script lang="ts">
  import { onMount } from "svelte";
  import Electrobun, { Electroview } from "electrobun/view";
  import type { PdfedDesktopRPC } from "../bun/types";

  const rpc = Electroview.defineRPC<PdfedDesktopRPC>({
    maxRequestTime: 10 * 60_000,
    handlers: {
      requests: {},
      messages: {},
    },
  });
  const eb = new Electrobun.Electroview({ rpc });

  type Operation =
    | "info"
    | "optimize"
    | "rotate"
    | "split"
    | "merge"
    | "search"
    | "encrypt"
    | "decrypt"
    | "add-images";
  type ThemeMode = "dark" | "light";
  type HistoryItem = {
    id: string;
    title: string;
    lines: string[];
    at: string;
    pending: boolean;
    operation?: Operation;
    inputPaths?: string[];
    outputPath?: string;
    formState?: FormState;
  };
  type FormState = {
    rotateDegrees: string;
    rotatePages: string;
    rotateDryRun: boolean;
    splitMode: SplitMode;
    splitRange: string;
    splitDryRun: boolean;
    mergeForce: boolean;
    mergeDryRun: boolean;
    optimizeDryRun: boolean;
    encryptUserPw: string;
    encryptOwnerPw: string;
    decryptPw: string;
    searchQuery: string;
    searchMax: string;
    searchThreshold: string;
    addImport: string;
    addPaper: string;
  };
  type SplitMode = "printed" | "pdfindex" | "extractAll";
  type HistoryGroup = {
    key: string;
    label: string;
    items: HistoryItem[];
  };
  type SuccessAction = "none" | "copy" | "reveal" | "open";
  type RecentTarget = "input" | "output";
  const operationOrder: Operation[] = ["info", "optimize", "rotate", "split", "merge", "encrypt", "decrypt", "add-images"];
  let operation = $state<Operation>("info");
  let inputPaths = $state<string[]>([]);
  let outputPath = $state<string>("");
  let status = $state<string>("Ready.");
  let optionsOpen = $state<boolean>(false);
  let rotateDegrees = $state<string>("90");
  let rotatePages = $state<string>("");
  let rotateDryRun = $state<boolean>(false);
  let splitMode = $state<SplitMode>("printed");
  let splitRange = $state<string>("");
  let splitDryRun = $state<boolean>(false);
  let mergeForce = $state<boolean>(false);
  let mergeDryRun = $state<boolean>(false);
  let optimizeDryRun = $state<boolean>(false);
  let encryptUserPw = $state<string>("");
  let encryptOwnerPw = $state<string>("");
  let decryptPw = $state<string>("");
  let searchQuery = $state<string>("");
  let searchMax = $state<string>("20");
  let searchThreshold = $state<string>("30");
  let addImport = $state<string>("");
  let addPaper = $state<string>("");

  let inputPath = $derived(inputPaths[0] || "");
  let resultData = $state<Record<string, unknown> | null>(null);
  let hasContext = $derived(Boolean(inputPath) || Boolean(outputPath) || status !== "Ready.");
  let step = $derived(getStep(operation, inputPath, outputPath, status));
  let hasResult = $derived(status !== "Ready." || resultData !== null);
  let isPendingResult = $derived(status.startsWith("Pending:"));
  let showResultPanel = $derived(hasResult && !(optionsOpen && isPendingResult));
  let sidebarOpen = $state<boolean>(false);
  let historyItems = $state<HistoryItem[]>([]);
  let historyGroups = $derived(groupHistoryByFile(historyItems));
  let historyReady = $state<boolean>(false);
  let recentInputs = $state<string[]>([]);
  let recentOutputs = $state<string[]>([]);
  let recentReady = $state<boolean>(false);
  let recentPickerOpen = $state<boolean>(false);
  let recentTarget = $state<RecentTarget>("input");
  let recentQuery = $state<string>("");
  let afterSuccessAction = $state<SuccessAction>("none");
  let runButtonLabel = $derived(requiresOptionsStep(operation) ? (optionsOpen ? "run now" : "configure") : "run");
  let prevStep = $state<number>(0);
  let stepPulse = $state<boolean>(false);
  let isRunning = $state<boolean>(false);
  let theme = $state<ThemeMode>("dark");
  let validationMessage = $derived(validateForm(operation));
  let recentCandidates = $derived(filteredRecentCandidates(recentTarget, recentQuery));

  onMount(() => {
    const stored = localStorage.getItem("pdfed-theme");
    if (stored === "dark" || stored === "light") {
      theme = stored;
    } else {
      theme = window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
    }
    applyTheme();

    try {
      const raw = localStorage.getItem("pdfed-history");
      if (raw) {
        const parsed = JSON.parse(raw) as unknown;
        if (Array.isArray(parsed)) {
          historyItems = parsed
            .filter((item): item is HistoryItem => {
              if (!item || typeof item !== "object") return false;
              const rec = item as Record<string, unknown>;
              return (
                typeof rec.id === "string" &&
                typeof rec.title === "string" &&
                Array.isArray(rec.lines) &&
                rec.lines.every((line) => typeof line === "string") &&
                typeof rec.at === "string" &&
                typeof rec.pending === "boolean" &&
                (rec.operation === undefined ||
                  rec.operation === "info" ||
                  rec.operation === "optimize" ||
                  rec.operation === "rotate" ||
                  rec.operation === "split" ||
                  rec.operation === "merge" ||
                  rec.operation === "search" ||
                  rec.operation === "encrypt" ||
                  rec.operation === "decrypt" ||
                  rec.operation === "add-images") &&
                (rec.inputPaths === undefined || (Array.isArray(rec.inputPaths) && rec.inputPaths.every((p) => typeof p === "string"))) &&
                (rec.outputPath === undefined || typeof rec.outputPath === "string") &&
                (rec.formState === undefined || typeof rec.formState === "object")
              );
            })
            .slice(0, 40);
        }
      }
    } catch {
      // Ignore malformed local storage and continue with empty history.
    } finally {
      historyReady = true;
    }

    try {
      const inRaw = localStorage.getItem("pdfed-recent-inputs");
      const outRaw = localStorage.getItem("pdfed-recent-outputs");
      const prefRaw = localStorage.getItem("pdfed-after-success");
      if (inRaw) {
        const parsed = JSON.parse(inRaw) as unknown;
        if (Array.isArray(parsed)) recentInputs = parsed.filter((p): p is string => typeof p === "string").slice(0, 30);
      }
      if (outRaw) {
        const parsed = JSON.parse(outRaw) as unknown;
        if (Array.isArray(parsed)) recentOutputs = parsed.filter((p): p is string => typeof p === "string").slice(0, 30);
      }
      if (prefRaw === "none" || prefRaw === "copy" || prefRaw === "reveal" || prefRaw === "open") {
        afterSuccessAction = prefRaw;
      }
    } catch {
      // ignore malformed recent/preference storage
    } finally {
      recentReady = true;
    }

    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        clearForm();
        return;
      }

      const key = e.key.toLowerCase();
      const mod = e.metaKey || e.ctrlKey;

      if (mod && key === "k") {
        e.preventDefault();
        recentPickerOpen = !recentPickerOpen;
        recentTarget = supportsOutputPicker(operation) ? "output" : "input";
        recentQuery = "";
        return;
      }

      if (isTypingTarget(e.target)) return;

      if (key === "enter") {
        e.preventDefault();
        void runOperation();
        return;
      }

      if (key === "i") {
        e.preventDefault();
        void pickPrimaryInput();
        return;
      }

      if (key === "o" && supportsOutputPicker(operation)) {
        e.preventDefault();
        void pickOutput();
        return;
      }

      if (key === "1") {
        e.preventDefault();
        operation = "info";
        optionsOpen = false;
        return;
      }

      if (key === "2") {
        e.preventDefault();
        operation = "optimize";
        optionsOpen = false;
        return;
      }

      if (key === "3") {
        e.preventDefault();
        operation = "rotate";
        optionsOpen = false;
        return;
      }

      if (key === "4") {
        e.preventDefault();
        operation = "split";
        optionsOpen = false;
        return;
      }

      if (key === "5") {
        e.preventDefault();
        operation = "merge";
        optionsOpen = false;
        return;
      }

      if (key === "6") {
        e.preventDefault();
        operation = "encrypt";
        optionsOpen = false;
        return;
      }

      if (key === "7") {
        e.preventDefault();
        operation = "decrypt";
        optionsOpen = false;
        return;
      }

      if (key === "8") {
        e.preventDefault();
        operation = "add-images";
        optionsOpen = false;
        return;
      }

      if (key === "[" || key === "]") {
        e.preventDefault();
        const idx = operationOrder.indexOf(operation);
        const next =
          key === "]"
            ? operationOrder[(idx + 1) % operationOrder.length]
            : operationOrder[(idx - 1 + operationOrder.length) % operationOrder.length];
        operation = next;
        optionsOpen = false;
        return;
      }

      if (mod && key === "d") {
        e.preventDefault();
        setTheme(theme === "dark" ? "light" : "dark");
        return;
      }

    };

    window.addEventListener("keydown", onKeyDown);
    return () => {
      window.removeEventListener("keydown", onKeyDown);
    };
  });

  $effect(() => {
    if (step > prevStep) {
      prevStep = step;
      stepPulse = true;
      const t = setTimeout(() => {
        stepPulse = false;
      }, 420);
      return () => clearTimeout(t);
    }
    prevStep = step;
  });

  $effect(() => {
    if (!historyReady) return;
    try {
      localStorage.setItem("pdfed-history", JSON.stringify(historyItems.slice(0, 40)));
    } catch {
      // Ignore storage quota/private mode failures.
    }
  });

  $effect(() => {
    if (!recentReady) return;
    try {
      localStorage.setItem("pdfed-recent-inputs", JSON.stringify(recentInputs.slice(0, 30)));
      localStorage.setItem("pdfed-recent-outputs", JSON.stringify(recentOutputs.slice(0, 30)));
      localStorage.setItem("pdfed-after-success", afterSuccessAction);
    } catch {
      // ignore storage failures
    }
  });

  async function pickPrimaryInput() {
    await runWithStatus(async () => {
      if (operation === "merge") {
        const res = await eb.rpc.request.pickFiles({ kind: "pdf", multiple: true });
        if (res.paths.length > 0) {
          inputPaths = res.paths;
          recentInputs = addRecentPaths(recentInputs, res.paths);
          if (requiresOptionsStep(operation)) {
            optionsOpen = true;
            status = "Pending: configure options, then run.";
          } else {
            status = `Pending: ${res.paths.length} PDFs selected.`;
          }
        } else {
          status = "Pending: no PDFs selected yet.";
        }
        return;
      }
      if (operation === "add-images") {
        const res = await eb.rpc.request.pickFiles({ kind: "image", multiple: true });
        if (res.paths.length > 0) {
          inputPaths = res.paths;
          recentInputs = addRecentPaths(recentInputs, res.paths);
          if (requiresOptionsStep(operation)) {
            optionsOpen = true;
            status = "Pending: configure options, then run.";
          } else {
            status = `Pending: ${res.paths.length} images selected.`;
          }
        } else {
          status = "Pending: no images selected yet.";
        }
        return;
      }

      const res = await eb.rpc.request.pickFiles({ kind: "pdf", multiple: false });
      if (res.paths.length > 0) {
        inputPaths = [res.paths[0]];
        recentInputs = addRecentPaths(recentInputs, [res.paths[0]]);
        if (requiresOptionsStep(operation)) {
          optionsOpen = true;
          status = "Pending: configure options, then run.";
        } else {
          status = "Pending: input selected. Run to execute.";
        }
        resultData = null;
      } else {
        status = "Pending: no input selected yet.";
        resultData = null;
      }
    });
  }

  async function pickOutput() {
    await runWithStatus(async () => {
      if (operation === "split") {
        const res = await eb.rpc.request.pickFiles({
          kind: "pdf",
          multiple: false,
          canChooseDirectory: true,
          canChooseFiles: false,
        });
        const picked = res.paths[0];
        if (picked) {
          outputPath = picked;
          recentOutputs = addRecentPaths(recentOutputs, [picked]);
          status = "Pending: output selected. Run to execute.";
        } else {
          status = "Pending: output not selected (optional).";
        }
        return;
      }

      const suggestedName = operation === "merge" ? "merged.pdf" : operation === "add-images" ? "images.pdf" : "output.pdf";
      const res = await eb.rpc.request.pickOutputPdf({ suggestedName });
      if (res.path) {
        outputPath = res.path;
        recentOutputs = addRecentPaths(recentOutputs, [res.path]);
        status = "Pending: output selected. Run to execute.";
        resultData = null;
      } else {
        status = "Pending: output not selected (optional).";
        resultData = null;
      }
    });
  }

  async function runOperation() {
    if (requiresOptionsStep(operation) && !optionsOpen) {
      optionsOpen = true;
      status = "Pending: configure options, then run.";
      return;
    }

    const missing = missingPrimaryInput(operation, inputPaths);
    if (missing) {
      status = missing;
      return;
    }

    await executeCurrent();
  }

  async function executeCurrent() {
    await runWithStatus(async () => {
      if (validationMessage) {
        status = `Error: ${validationMessage}`;
        return;
      }
      const built = buildArgs(operation);
      if (!built.ok) {
        status = `Error: ${built.error}`;
        return;
      }

      status = `Running ${operation}...`;
      isRunning = true;
      const res = await eb.rpc.request.runPdfedArgs({
        args: built.args,
        cwd: built.cwd,
      });
      if (res.ok) {
        const resultJson = (res.json ?? null) as Record<string, unknown> | null;
        resultData = resultJson;
        status = "Done.";
        const resolvedOutput = outputPathFromResult(resultJson) ?? outputPath;
        const actionTarget = resultActionPath(resultJson, inputPaths, outputPath);
        pushHistory({
          title: resultTitle(resultJson),
          lines: resultLines(resultJson),
          pending: false,
          operation,
          inputPaths: [...inputPaths],
          outputPath: resolvedOutput,
          formState: snapshotFormState(),
        });
        recentInputs = addRecentPaths(recentInputs, inputPaths);
        if (resolvedOutput) recentOutputs = addRecentPaths(recentOutputs, [resolvedOutput]);
        await applyAfterSuccess(actionTarget);
      } else {
        resultData = null;
        status = `Error: ${res.error}`;
      }
      optionsOpen = false;
      isRunning = false;
    });
  }

  async function runWithStatus(fn: () => Promise<void>) {
    try {
      await fn();
    } catch (err) {
      status = `Error: ${err instanceof Error ? err.message : String(err)}`;
    } finally {
      isRunning = false;
    }
  }

  async function stopRun() {
    const res = await eb.rpc.request.cancelRun({});
    if (res.ok && res.cancelled) status = "Cancelled.";
    else if (res.ok) status = "Nothing running.";
    else status = `Error: ${res.error}`;
    isRunning = false;
  }

  function setTheme(next: ThemeMode) {
    theme = next;
    localStorage.setItem("pdfed-theme", next);
    applyTheme();
  }

  function applyTheme() {
    document.body.setAttribute("data-theme", theme);
  }

  function getStep(op: Operation, input: string, output: string, result: string): number {
    if (!input) return 0;
    if (supportsOutputPicker(op) && !output && (op === "merge" || op === "add-images")) return 1;
    if (result !== "Ready." && !result.startsWith("Selected")) return 3;
    return 2;
  }

  function outputPathFromResult(json: Record<string, unknown> | null): string | null {
    if (!json) return null;
    const out = json.output;
    return typeof out === "string" && out.trim().length > 0 ? out.trim() : null;
  }

  function inputPathFromResult(json: Record<string, unknown> | null): string | null {
    if (!json) return null;
    const input = json.input;
    if (typeof input === "string" && input.trim().length > 0) return input.trim();
    const doc = json.document;
    if (doc && typeof doc === "object") {
      const file = (doc as Record<string, unknown>).file;
      if (typeof file === "string" && file.trim().length > 0) return file.trim();
    }
    return null;
  }

  function resultTitle(json: Record<string, unknown> | null): string {
    if (!json) return "Result";
    const cmd = json.command;
    return typeof cmd === "string" ? cmd : "Result";
  }

  function resultLines(json: Record<string, unknown> | null): string[] {
    if (!json) return [];
    const cmd = typeof json.command === "string" ? json.command : "";
    const lines: string[] = [];
    if (cmd === "info") {
      const doc = (json.document ?? {}) as Record<string, unknown>;
      lines.push(`Pages: ${String(doc.page_count ?? "?")}`);
      lines.push(`Version: ${String(doc.pdf_version ?? "?")}`);
      lines.push(`Size: ${String(doc.size_human ?? "?")}`);
      return lines;
    }
    if (cmd === "optimize") {
      lines.push(`Output: ${String(json.output ?? "?")}`);
      lines.push(`Before: ${String(json.input_size_human ?? "?")}  After: ${String(json.output_size_human ?? "?")}`);
      lines.push(`Saved: ${String(json.saved_human ?? "?")} (${String(json.saved_percent ?? "?")}%)`);
      return lines;
    }
    if (cmd === "rotate") {
      lines.push(`Output: ${String(json.output ?? "?")}`);
      lines.push(`Degrees: ${String(json.degrees ?? "?")}  Pages: ${String(json.pages ?? "all")}`);
      return lines;
    }
    if (cmd === "split") {
      lines.push(`Mode: ${String(json.mode ?? "?")}`);
      lines.push(`Output: ${String(json.output ?? "?")}`);
      if (json.page_count != null) lines.push(`Page count: ${String(json.page_count)}`);
      return lines;
    }
    if (cmd === "merge") {
      lines.push(`Output: ${String(json.output ?? "?")}`);
      lines.push(`Inputs: ${String(json.input_count ?? "?")}  Pages: ${String(json.page_count ?? "?")}`);
      return lines;
    }
    if (cmd === "add-images") {
      lines.push(`Output: ${String(json.output ?? "?")}`);
      lines.push(`Images: ${String(json.image_count ?? "?")}  Size: ${String(json.size_human ?? "?")}`);
      return lines;
    }
    return Object.entries(json)
      .filter(([k]) => !["ok", "command"].includes(k))
      .slice(0, 6)
      .map(([k, v]) => `${k}: ${typeof v === "object" ? "[object]" : String(v)}`);
  }

  function resultActionPath(
    json: Record<string, unknown> | null,
    inputs: string[],
    selectedOutput: string,
  ): string | null {
    return outputPathFromResult(json) ?? inputPathFromResult(json) ?? (selectedOutput || inputs[0] || null);
  }

  function isSplitExtractAllResult(json: Record<string, unknown> | null): boolean {
    if (!json) return false;
    return json.command === "split" && json.mode === "extract_all";
  }

  async function openResultPath() {
    const path = resultActionPath(resultData, inputPaths, outputPath);
    if (!path) return;
    const res = await eb.rpc.request.openPath({ path });
    if (!res.ok) status = `Error: ${res.error}`;
  }

  async function revealResultPath() {
    const path = resultActionPath(resultData, inputPaths, outputPath);
    if (!path) return;
    const res = await eb.rpc.request.revealPath({ path });
    if (!res.ok) status = `Error: ${res.error}`;
  }

  async function copyResultPath() {
    const path = resultActionPath(resultData, inputPaths, outputPath);
    if (!path) return;
    const res = await eb.rpc.request.copyText({ text: path });
    status = res.ok ? "Copied path." : `Error: ${res.error}`;
  }

  function supportsOutputPicker(op: Operation): boolean {
    return ["optimize", "rotate", "split", "merge", "encrypt", "decrypt", "add-images"].includes(op);
  }

  function requiresOptionsStep(op: Operation): boolean {
    return op !== "info";
  }

  function missingPrimaryInput(op: Operation, inputs: string[]): string | null {
    if (op === "merge") return inputs.length < 2 ? "Select at least 2 input PDFs first." : null;
    if (op === "add-images") return inputs.length < 1 ? "Select at least 1 image first." : null;
    return inputs.length < 1 ? "Select an input PDF first." : null;
  }

  function buildArgs(op: Operation): { ok: true; args: string[]; cwd: string } | { ok: false; error: string } {
    const first = inputPaths[0] || "";
    const cwd = inferCwd(first, outputPath);

    switch (op) {
      case "info":
        return { ok: true, args: ["info", first], cwd };
      case "optimize": {
        const args = ["optimize", first];
        if (outputPath) args.push("-o", outputPath);
        if (optimizeDryRun) args.push("--dry-run");
        return { ok: true, args, cwd };
      }
      case "rotate": {
        const args = ["rotate", first, rotateDegrees || "90"];
        if (rotatePages.trim()) args.push("-p", rotatePages.trim());
        if (outputPath) args.push("-o", outputPath);
        if (rotateDryRun) args.push("--dry-run");
        return { ok: true, args, cwd };
      }
      case "split": {
        const args = ["split", first];
        if (splitMode === "extractAll") args.push("-e");
        if (splitMode === "printed" && splitRange.trim()) args.push("-p", splitRange.trim());
        if (splitMode === "pdfindex" && splitRange.trim()) args.push("-P", splitRange.trim());
        if (splitMode !== "extractAll" && !splitRange.trim()) {
          return { ok: false, error: "Split needs a page range for the selected numbering mode." };
        }
        if (outputPath) args.push("-o", outputPath);
        if (splitDryRun) args.push("--dry-run");
        return { ok: true, args, cwd };
      }
      case "merge": {
        if (!outputPath) return { ok: false, error: "Merge requires an output PDF." };
        const args = ["merge", outputPath, ...inputPaths];
        if (mergeForce) args.push("--force");
        if (mergeDryRun) args.push("--dry-run");
        return { ok: true, args, cwd: inferCwd(outputPath, inputPaths[0] || "") };
      }
      case "search": {
        if (!searchQuery.trim()) return { ok: false, error: "Search query is required." };
        const args = ["search", first, searchQuery.trim(), "--no-interactive", "-n", searchMax || "20", "-t", searchThreshold || "30"];
        return { ok: true, args, cwd };
      }
      case "encrypt": {
        if (!encryptUserPw.trim()) return { ok: false, error: "Encrypt requires --user-pw." };
        const args = ["encrypt", first, "--user-pw", encryptUserPw];
        if (encryptOwnerPw.trim()) args.push("--owner-pw", encryptOwnerPw.trim());
        if (outputPath) args.push("-o", outputPath);
        return { ok: true, args, cwd };
      }
      case "decrypt": {
        if (!decryptPw.trim()) return { ok: false, error: "Decrypt requires --password." };
        const args = ["decrypt", first, "--password", decryptPw.trim()];
        if (outputPath) args.push("-o", outputPath);
        return { ok: true, args, cwd };
      }
      case "add-images": {
        if (!outputPath) return { ok: false, error: "add-images requires output PDF path." };
        const args = ["add-images", outputPath, ...inputPaths];
        if (addImport.trim()) args.push("--import", addImport.trim());
        if (addPaper.trim()) args.push("--paper", addPaper.trim());
        return { ok: true, args, cwd: inferCwd(outputPath, inputPaths[0] || "") };
      }
      default:
        return { ok: false, error: "Unsupported operation." };
    }
  }

  function inferCwd(primary: string, fallback: string): string {
    const p = primary || fallback;
    if (!p) return ".";
    const normalized = p.replaceAll("\\", "/");
    const idx = normalized.lastIndexOf("/");
    return idx > 0 ? normalized.slice(0, idx) : ".";
  }

  function filenameOnly(path: string): string {
    if (!path) return "";
    const normalized = path.replaceAll("\\", "/");
    const parts = normalized.split("/");
    return parts[parts.length - 1] || path;
  }

  function compactFilename(path: string, maxLen = 26): string {
    const name = filenameOnly(path);
    if (!name || name.length <= maxLen) return name;

    const dot = name.lastIndexOf(".");
    const ext = dot > 0 ? name.slice(dot) : "";
    const base = dot > 0 ? name.slice(0, dot) : name;

    const endLen = ext ? Math.min(8, Math.max(4, ext.length + 2)) : 6;
    const startLen = Math.max(5, maxLen - endLen - 2);
    const start = base.slice(0, startLen);
    const endSource = ext ? `${base.slice(-Math.max(2, endLen - ext.length))}${ext}` : name.slice(-endLen);
    return `${start}..${endSource}`;
  }

  function isTypingTarget(target: EventTarget | null): boolean {
    if (!(target instanceof HTMLElement)) return false;
    const tag = target.tagName.toLowerCase();
    return tag === "input" || tag === "textarea" || tag === "select" || target.isContentEditable;
  }

  function addRecentPaths(current: string[], paths: string[]): string[] {
    const next = [...current];
    for (const p of paths) {
      const path = p.trim();
      if (!path) continue;
      const existingIdx = next.findIndex((x) => x === path);
      if (existingIdx >= 0) next.splice(existingIdx, 1);
      next.unshift(path);
    }
    return next.slice(0, 30);
  }

  function filteredRecentCandidates(target: RecentTarget, query: string): string[] {
    const source = target === "input" ? recentInputs : recentOutputs;
    const q = query.trim().toLowerCase();
    if (!q) return source.slice(0, 20);
    return source
      .filter((p) => p.toLowerCase().includes(q) || filenameOnly(p).toLowerCase().includes(q))
      .slice(0, 20);
  }

  function applyRecentCandidate(path: string) {
    if (recentTarget === "input") {
      inputPaths = [path];
      recentInputs = addRecentPaths(recentInputs, [path]);
      status = "Pending: input selected from recent.";
    } else {
      outputPath = path;
      recentOutputs = addRecentPaths(recentOutputs, [path]);
      status = "Pending: output selected from recent.";
    }
    recentPickerOpen = false;
  }

  function snapshotFormState(): FormState {
    return {
      rotateDegrees,
      rotatePages,
      rotateDryRun,
      splitMode,
      splitRange,
      splitDryRun,
      mergeForce,
      mergeDryRun,
      optimizeDryRun,
      encryptUserPw,
      encryptOwnerPw,
      decryptPw,
      searchQuery,
      searchMax,
      searchThreshold,
      addImport,
      addPaper,
    };
  }

  function applyFormState(formState: FormState | undefined) {
    if (!formState) return;
    rotateDegrees = formState.rotateDegrees;
    rotatePages = formState.rotatePages;
    rotateDryRun = formState.rotateDryRun;
    splitMode = formState.splitMode;
    splitRange = formState.splitRange;
    splitDryRun = formState.splitDryRun;
    mergeForce = formState.mergeForce;
    mergeDryRun = formState.mergeDryRun;
    optimizeDryRun = formState.optimizeDryRun;
    encryptUserPw = formState.encryptUserPw;
    encryptOwnerPw = formState.encryptOwnerPw;
    decryptPw = formState.decryptPw;
    searchQuery = formState.searchQuery;
    searchMax = formState.searchMax;
    searchThreshold = formState.searchThreshold;
    addImport = formState.addImport;
    addPaper = formState.addPaper;
  }

  async function applyAfterSuccess(path: string | null) {
    if (afterSuccessAction === "none") return;
    if (!path) {
      status = "Done. No target path for after action.";
      return;
    }
    if (afterSuccessAction === "copy") {
      const ok = await copyText(path);
      if (ok) status = "Done. Copied path.";
      return;
    }
    if (afterSuccessAction === "reveal") {
      const res = await eb.rpc.request.revealPath({ path });
      if (!res.ok) status = `Error: ${res.error}`;
      else status = "Done. Revealed path.";
      return;
    }
    if (afterSuccessAction === "open") {
      status = `After action: opening ${compactFilename(path, 40)}...`;
      const res = await eb.rpc.request.openPath({ path });
      if (!res.ok) {
        status = `Error: ${res.error}`;
      } else {
        status = `Done. Opened: ${path}`;
      }
    }
  }

  async function copyText(text: string): Promise<boolean> {
    const res = await eb.rpc.request.copyText({ text });
    if (!res.ok) {
      status = `Error: ${res.error}`;
      return false;
    }
    return true;
  }

  function validateForm(op: Operation): string | null {
    if (op === "search") {
      const max = Number(searchMax);
      const threshold = Number(searchThreshold);
      if (!Number.isFinite(max) || max < 1) return "Max results must be a positive number.";
      if (!Number.isFinite(threshold) || threshold < 0 || threshold > 100) return "Threshold must be between 0 and 100.";
    }
    if (op === "rotate") {
      const deg = Number(rotateDegrees || "90");
      if (![90, 180, 270].includes(deg)) return "Rotate degrees must be 90, 180, or 270.";
    }
    if (op === "split" && splitMode !== "extractAll" && splitRange.trim().length === 0) {
      return "Split range is required.";
    }
    if (op === "encrypt" && !encryptUserPw.trim()) return "User password is required.";
    if (op === "decrypt" && !decryptPw.trim()) return "Password is required.";
    return null;
  }

  function pushHistory(item: {
    title: string;
    lines: string[];
    pending: boolean;
    operation?: Operation;
    inputPaths?: string[];
    outputPath?: string;
    formState?: FormState;
  }) {
    const at = new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
    historyItems = [
      { id: crypto.randomUUID(), at, ...item },
      ...historyItems,
    ].slice(0, 40);
  }

  function applyHistoryItem(item: HistoryItem) {
    if (item.operation) {
      operation = item.operation === "search" ? "info" : item.operation;
    }
    applyFormState(item.formState);
    if (typeof item.outputPath === "string" && item.outputPath.length > 0) {
      inputPaths = [item.outputPath];
      recentInputs = addRecentPaths(recentInputs, [item.outputPath]);
    } else if (item.inputPaths && item.inputPaths.length > 0) {
      inputPaths = [...item.inputPaths];
      recentInputs = addRecentPaths(recentInputs, item.inputPaths);
    }
    outputPath = "";
    resultData = null;
    optionsOpen = false;
    status = "Pending: restored from history.";
  }

  function clearHistory() {
    historyItems = [];
    status = "History cleared.";
  }

  function historySourcePath(item: HistoryItem): string {
    if (item.outputPath) return item.outputPath;
    if (item.inputPaths && item.inputPaths.length > 0) return item.inputPaths[0];
    return "";
  }

  function historyItemSignature(item: HistoryItem): string {
    return JSON.stringify({
      title: item.title,
      lines: item.lines,
      operation: item.operation ?? "",
      inputPaths: item.inputPaths ?? [],
      outputPath: item.outputPath ?? "",
      formState: item.formState ?? null,
      pending: item.pending,
    });
  }

  function groupHistoryByFile(items: HistoryItem[]): HistoryGroup[] {
    const map = new Map<string, HistoryGroup>();
    for (const item of items) {
      const sourcePath = historySourcePath(item);
      const key = sourcePath || "__unknown__";

      let group = map.get(key);
      if (!group) {
        group = {
          key,
          label: sourcePath ? filenameOnly(sourcePath) : "Unknown file",
          items: [],
        };
        map.set(key, group);
      }

      const prev = group.items[group.items.length - 1];
      if (prev && historyItemSignature(prev) === historyItemSignature(item)) {
        continue;
      }
      group.items.push(item);
    }
    return Array.from(map.values());
  }

  function clearForm() {
    inputPaths = [];
    outputPath = "";
    optionsOpen = false;
    resultData = null;
    status = "Ready.";
    isRunning = false;
    stepPulse = false;
    prevStep = 0;
    recentPickerOpen = false;
    recentTarget = "input";
    recentQuery = "";

    rotateDegrees = "90";
    rotatePages = "";
    rotateDryRun = false;

    splitMode = "printed";
    splitRange = "";
    splitDryRun = false;

    mergeForce = false;
    mergeDryRun = false;
    optimizeDryRun = false;

    encryptUserPw = "";
    encryptOwnerPw = "";
    decryptPw = "";

    searchQuery = "";
    searchMax = "20";
    searchThreshold = "30";

    addImport = "";
    addPaper = "";
  }
</script>

<main class="app" class:pre-run={step < 3 && !optionsOpen && !hasResult}>
  <div class="topbar">
    <div class="left-top">
      <button class="mini history-toggle" onclick={() => (sidebarOpen = !sidebarOpen)}>
        {sidebarOpen ? "Hide" : "History"}
      </button>
      <div class="brand">pdfed</div>
    </div>
    <div class="theme-toggle" role="group" aria-label="Theme">
      <select class="mini mini-select" bind:value={afterSuccessAction} aria-label="After success action">
        <option value="none">after: none</option>
        <option value="copy">after: copy path</option>
        <option value="reveal">after: reveal</option>
        <option value="open">after: open</option>
      </select>
      <button class="mini" class:active={theme === "light"} onclick={() => setTheme("light")}>Light</button>
      <button class="mini" class:active={theme === "dark"} onclick={() => setTheme("dark")}>Dark</button>
    </div>
  </div>

  <div class="workspace" class:with-sidebar={sidebarOpen}>
    {#if sidebarOpen}
      <aside class="sidebar">
        <div class="sidebar-header">
          <h3>Past Results</h3>
          <button class="mini clear-history" onclick={clearHistory}>Clear</button>
        </div>
        {#if historyGroups.length === 0}
          <p class="empty">No runs yet.</p>
        {:else}
          {#each historyGroups as group}
            <section class="history-group">
              <h4 title={group.key !== "__unknown__" ? group.key : ""}>{group.label}</h4>
              {#each group.items as item}
                <button class="history-card history-button" onclick={() => applyHistoryItem(item)}>
                  <header>
                    <strong>{item.title}</strong>
                    <span>{item.at}</span>
                  </header>
                  {#each item.lines.slice(0, 3) as line}
                    <p>{line}</p>
                  {/each}
                </button>
              {/each}
            </section>
          {/each}
        {/if}
      </aside>
    {/if}

    <div class="main-pane">
      <section class="composer" style={`--step:${step};`}>
        <div class="command-line">
      <select bind:value={operation} class="seg select op" aria-label="Operation">
        <option value="info">info</option>
        <option value="optimize">optimize</option>
        <option value="rotate">rotate</option>
        <option value="split">split</option>
        <option value="merge">merge</option>
        <option value="encrypt">encrypt</option>
        <option value="decrypt">decrypt</option>
        <option value="add-images">add-images</option>
      </select>

      <button class="seg picker input-picker" onclick={pickPrimaryInput}>
        {#if operation === "merge"}
          {inputPaths.length > 0 ? `${inputPaths.length} pdf(s)` : "select input pdfs"}
        {:else if operation === "add-images"}
          {inputPaths.length > 0 ? `${inputPaths.length} image(s)` : "select images"}
        {:else}
          {inputPath ? compactFilename(inputPath) : "select input.pdf"}
        {/if}
      </button>

      {#if supportsOutputPicker(operation)}
        <button class="seg picker output-picker" onclick={pickOutput}>
          {outputPath ? compactFilename(outputPath) : "select output.pdf (optional)"}
        </button>
      {/if}

      <button class="seg run run-btn" onclick={runOperation}>{runButtonLabel}</button>
      {#if isRunning}
        <button class="seg picker run-btn" onclick={stopRun}>stop</button>
      {/if}
        </div>
      </section>

      {#if recentPickerOpen}
        <section class="panel pending recent-panel">
          <h2>Recent files</h2>
          <div class="recent-controls">
            <button class="mini" class:active={recentTarget === "input"} onclick={() => (recentTarget = "input")}>Input</button>
            <button class="mini" class:active={recentTarget === "output"} onclick={() => (recentTarget = "output")}>Output</button>
            <input class="opt-input recent-search" bind:value={recentQuery} placeholder="filter recent files..." />
          </div>
          <div class="recent-list">
            {#if recentCandidates.length === 0}
              <p class="empty">No recent matches.</p>
            {:else}
              {#each recentCandidates as candidate}
                <button class="history-card history-button" onclick={() => applyRecentCandidate(candidate)}>
                  <header>
                    <strong>{compactFilename(candidate, 34)}</strong>
                  </header>
                  <p>{candidate}</p>
                </button>
              {/each}
            {/if}
          </div>
        </section>
      {/if}

      {#if optionsOpen}
        <section class="panel pending options-sheet">
      <h2>Options</h2>
      <div class="options-grid">
        {#if operation === "optimize"}
          <label><input type="checkbox" bind:checked={optimizeDryRun} /> dry run</label>
        {/if}
        {#if operation === "rotate"}
          <label>degrees <input class="opt-input" bind:value={rotateDegrees} placeholder="90 | 180 | 270" /></label>
          <label>pages <input class="opt-input" bind:value={rotatePages} placeholder="1-3,5" /></label>
          <label><input type="checkbox" bind:checked={rotateDryRun} /> dry run</label>
        {/if}
        {#if operation === "split"}
          <label>
            page mode
            <select class="opt-input" bind:value={splitMode}>
              <option value="printed">Viewer pages</option>
              <option value="pdfindex">PDF index pages</option>
              <option value="extractAll">All pages</option>
            </select>
          </label>
          {#if splitMode !== "extractAll"}
            <label>
              {splitMode === "printed" ? "viewer page range" : "index page range"}
              <input
                class="opt-input"
                bind:value={splitRange}
                placeholder="1-5,7"
              />
            </label>
          {/if}
          <label><input type="checkbox" bind:checked={splitDryRun} /> dry run</label>
        {/if}
        {#if operation === "merge"}
          <label><input type="checkbox" bind:checked={mergeForce} /> force overwrite</label>
          <label><input type="checkbox" bind:checked={mergeDryRun} /> dry run</label>
        {/if}
        {#if operation === "search"}
          <label>query <input class="opt-input" bind:value={searchQuery} placeholder="search text" /></label>
          <label>max results <input class="opt-input" bind:value={searchMax} placeholder="20" /></label>
          <label>threshold <input class="opt-input" bind:value={searchThreshold} placeholder="30" /></label>
        {/if}
        {#if operation === "encrypt"}
          <label>user password <input class="opt-input" bind:value={encryptUserPw} placeholder="required" /></label>
          <label>owner password <input class="opt-input" bind:value={encryptOwnerPw} placeholder="optional" /></label>
        {/if}
        {#if operation === "decrypt"}
          <label>password <input class="opt-input" bind:value={decryptPw} placeholder="required" /></label>
        {/if}
        {#if operation === "add-images"}
          <label>import opts <input class="opt-input" bind:value={addImport} placeholder="dpi:300,scalefactor:1" /></label>
          <label>paper <input class="opt-input" bind:value={addPaper} placeholder="A4, Letter" /></label>
        {/if}
      </div>
      <div class="options-actions">
        <button class="seg run" onclick={executeCurrent}>apply & run</button>
        <button class="seg picker" onclick={() => (optionsOpen = false)}>cancel</button>
      </div>
        </section>
      {/if}

      {#if showResultPanel}
        <section class="panel" class:elevated={step >= 3} class:pulse={stepPulse && step >= 3} class:pending={isPendingResult}>
      <h2>{isPendingResult ? "Pending" : resultTitle(resultData)}</h2>
      {#if isPendingResult || !resultData}
        <pre>{status}</pre>
      {:else}
        <div class="result-lines">
          {#each resultLines(resultData) as line}
            <p>{line}</p>
          {/each}
        </div>
        <div class="options-actions">
          <button class="seg picker" onclick={copyResultPath}>copy path</button>
          <button class="seg picker" onclick={revealResultPath}>{isSplitExtractAllResult(resultData) ? "reveal folder" : "reveal"}</button>
          <button class="seg run" onclick={openResultPath}>{isSplitExtractAllResult(resultData) ? "open folder" : "open"}</button>
        </div>
      {/if}
        </section>
      {/if}
    </div>
  </div>
</main>

<style>
  :global(body) {
    margin: 0;
    font-family: ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI", Inter, sans-serif;
    background: var(--bg);
    color: var(--text);
  }

  :global(*) {
    box-sizing: border-box;
  }

  :global(body[data-theme="light"]) {
    --bg: #f7f7f6;
    --text: #2f2f2f;
    --muted: #7d7d7a;
    --panel: #ffffff;
    --panel-border: #e8e8e6;
    --subtle: #f5f5f3;
    --subtle-border: #e9e9e7;
    --btn: #ffffff;
    --btn-border: #e4e4e2;
    --btn-hover: #f3f3f1;
    --primary: #2f2f2f;
    --path: #fafaf9;
    --path-text: #686763;
  }

  :global(body[data-theme="dark"]) {
    --bg: #0f1115;
    --text: #e8e8e7;
    --muted: #9a9a96;
    --panel: #17191f;
    --panel-border: #2b2e36;
    --subtle: #1d2027;
    --subtle-border: #2c3039;
    --btn: #20232b;
    --btn-border: #343845;
    --btn-hover: #2a2f39;
    --primary: #f2f2f1;
    --path: #14171d;
    --path-text: #b5b5b2;
  }

  .app {
    max-width: 920px;
    margin: 0 auto;
    padding: 20px 18px 22px;
    min-height: 100vh;
    display: flex;
    flex-direction: column;
  }

  .topbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }

  .left-top {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }

  .history-toggle {
    border-color: var(--subtle-border);
  }

  .workspace {
    display: grid;
    grid-template-columns: 1fr;
    gap: 12px;
    flex: 1;
  }

  .workspace.with-sidebar {
    grid-template-columns: 280px 1fr;
    align-items: start;
  }

  .main-pane {
    min-width: 0;
    display: flex;
    flex-direction: column;
  }

  .sidebar {
    background: var(--panel);
    border: 1px solid var(--panel-border);
    border-radius: 10px;
    padding: 8px;
    max-height: calc(100vh - 70px);
    overflow: auto;
    position: sticky;
    top: 12px;
  }

  .sidebar h3 {
    margin: 2px 2px 8px;
    font-size: 12px;
    color: var(--muted);
  }

  .sidebar-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin: 2px 2px 8px;
  }

  .sidebar-header h3 {
    margin: 0;
  }

  .clear-history {
    padding: 2px 8px;
    border-color: var(--subtle-border);
  }

  .empty {
    margin: 8px;
    font-size: 12px;
    color: var(--muted);
  }

  .history-group {
    margin-bottom: 10px;
  }

  .history-group h4 {
    margin: 2px 2px 6px;
    font-size: 11px;
    font-weight: 600;
    color: var(--muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .history-card {
    border: 1px solid var(--subtle-border);
    border-radius: 8px;
    background: var(--subtle);
    padding: 8px;
    margin-bottom: 8px;
  }

  .history-card header {
    display: flex;
    justify-content: space-between;
    gap: 8px;
    margin-bottom: 4px;
    font-size: 11px;
    color: var(--muted);
  }

  .history-card p {
    margin: 0 0 4px;
    font-size: 11px;
    color: var(--path-text);
  }

  .history-button {
    width: 100%;
    text-align: left;
    cursor: pointer;
  }

  .brand {
    font-size: 15px;
    font-weight: 600;
    letter-spacing: 0.2px;
    color: var(--text);
  }

  .theme-toggle {
    display: inline-flex;
    gap: 4px;
    background: var(--subtle);
    border: 1px solid var(--subtle-border);
    border-radius: 999px;
    padding: 3px;
  }

  .mini {
    border: 1px solid transparent;
    background: transparent;
    color: var(--muted);
    border-radius: 999px;
    padding: 4px 10px;
    font-size: 12px;
    cursor: pointer;
  }

  .mini.active {
    color: var(--text);
    background: var(--panel);
    border-color: var(--panel-border);
  }

  .mini-select {
    border-color: var(--subtle-border);
    color: var(--text);
    background: var(--panel);
    border-radius: 999px;
    height: 28px;
  }

  .composer {
    background: var(--panel);
    border: 1px solid var(--panel-border);
    border-radius: 14px;
    padding: 8px;
    margin-bottom: 8px;
    width: clamp(520px, 50vw, 920px);
    max-width: 100%;
    margin-left: auto;
    margin-right: auto;
    align-self: start;
    transition:
      transform 220ms ease,
      border-color 220ms ease;
  }

  .app.pre-run .composer {
    align-self: center;
    margin-top: auto;
    margin-bottom: auto;
  }

  .command-line {
    display: flex;
    gap: 6px;
    align-items: center;
    flex-wrap: wrap;
    width: 100%;
    min-height: 36px;
    background: var(--subtle);
    border: 1px solid var(--subtle-border);
    border-radius: 10px;
    overflow: visible;
  }

  .seg {
    height: 36px;
    background: transparent;
    color: var(--text);
    border: 0;
    border-right: 0;
    border-radius: 0;
    padding: 0 10px;
    font-size: 12px;
    line-height: 36px;
    margin: 0;
    min-width: 0;
  }

  .select {
    min-width: 92px;
    flex: 0 0 auto;
    border-radius: 8px;
  }

  .picker {
    flex: 1 1 220px;
    min-width: 160px;
    max-width: 100%;
    cursor: pointer;
    text-align: left;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--path-text);
    border-radius: 8px;
  }

  .run {
    white-space: nowrap;
    cursor: pointer;
    background: var(--primary);
    color: var(--bg);
    border-right: 0;
    padding: 0 14px;
    font-weight: 600;
    flex: 0 0 auto;
    border-radius: 8px;
  }

  @media (min-width: 820px) {
    .command-line {
      flex-wrap: nowrap;
    }

    .picker {
      flex: 1 1 0;
      min-width: 0;
    }

    .run-btn {
      flex: 0 0 auto;
    }
  }

  @media (max-width: 760px) {
    .workspace.with-sidebar {
      grid-template-columns: 1fr;
    }

    .composer {
      width: 100%;
    }

    .command-line {
      display: grid;
      grid-template-columns: 1fr auto;
      grid-template-areas:
        "op run"
        "input input"
        "output output";
      gap: 6px;
      border: 0;
      background: transparent;
      padding: 0;
      min-height: auto;
    }

    .op {
      grid-area: op;
    }

    .run-btn {
      grid-area: run;
      width: auto;
    }

    .input-picker {
      grid-area: input;
    }

    .output-picker {
      grid-area: output;
    }

  }

  .options-grid {
    display: grid;
    gap: 8px;
    margin-bottom: 8px;
  }

  .recent-panel {
    margin-top: 0;
  }

  .recent-controls {
    display: flex;
    gap: 8px;
    align-items: center;
    margin-bottom: 8px;
  }

  .recent-search {
    min-width: 220px;
  }

  .recent-list {
    max-height: 220px;
    overflow: auto;
  }

  .options-grid label {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 12px;
    color: var(--text);
    flex-wrap: wrap;
  }

  .opt-input {
    height: 28px;
    min-width: 160px;
    background: var(--subtle);
    color: var(--text);
    border: 1px solid var(--subtle-border);
    border-radius: 6px;
    padding: 0 8px;
    font-size: 12px;
  }

  .options-actions {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
  }

  .panel {
    background: var(--panel);
    border: 1px solid var(--panel-border);
    border-radius: 8px;
    padding: 8px;
    margin-bottom: 8px;
    width: clamp(520px, 50vw, 920px);
    max-width: 100%;
    margin-left: auto;
    margin-right: auto;
    transition:
      transform 220ms ease,
      box-shadow 220ms ease;
  }

  .options-sheet {
    margin-top: 2px;
    margin-bottom: 14px;
    animation: optionsDropIn 220ms ease both;
  }

  .panel.pending {
    border-style: dashed;
    border-color: var(--subtle-border);
    box-shadow: 0 6px 14px rgba(0, 0, 0, 0.08);
  }

  .panel.elevated {
    transform: translateY(-6px);
    box-shadow: 0 10px 22px rgba(0, 0, 0, 0.18);
  }

  .panel.pulse {
    animation: resultShadowPulse 420ms ease;
  }

  h2 {
    margin: 0 0 4px;
    font-size: 12px;
    font-weight: 600;
    color: var(--muted);
  }

  .picker:hover,
  .run:hover,
  .select:hover {
    filter: brightness(0.95);
  }

  pre {
    margin: 0;
    background: var(--path);
    border: 1px solid var(--subtle-border);
    border-radius: 6px;
    padding: 7px 8px;
    white-space: pre-wrap;
    color: var(--path-text);
    font-size: 11px;
    line-height: 1.35;
  }

  .result-lines p {
    margin: 0 0 6px;
    font-size: 12px;
    color: var(--path-text);
  }

  .panel.pending pre {
    font-style: italic;
    opacity: 0.9;
  }

  @keyframes resultShadowPulse {
    0% {
      box-shadow: 0 0 0 rgba(0, 0, 0, 0);
      transform: translateY(-4px);
    }
    45% {
      box-shadow: 0 14px 30px rgba(0, 0, 0, 0.22);
      transform: translateY(-8px);
    }
    100% {
      box-shadow: 0 10px 22px rgba(0, 0, 0, 0.18);
      transform: translateY(-6px);
    }
  }

  @keyframes optionsDropIn {
    from {
      opacity: 0;
      transform: translateY(-8px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

</style>
