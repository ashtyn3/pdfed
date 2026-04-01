import type { RPCSchema } from "electrobun/bun";

export type PdfedDesktopRPC = {
  bun: RPCSchema<{
    requests: {
      pickPdf: { params: {}; response: { path: string | null } };
      pickFiles: {
        params: { kind: "pdf" | "image"; multiple?: boolean; canChooseDirectory?: boolean; canChooseFiles?: boolean };
        response: { paths: string[] };
      };
      pickOutputPdf: { params: { suggestedName: string }; response: { path: string | null } };
      runPdfedArgs: {
        params: { args: string[]; cwd?: string };
        response: { ok: boolean; json?: unknown; error?: string };
      };
      cancelRun: { params: {}; response: { ok: boolean; cancelled: boolean; error?: string } };
      openPath: { params: { path: string }; response: { ok: boolean; error?: string } };
      revealPath: { params: { path: string }; response: { ok: boolean; error?: string } };
      copyText: { params: { text: string }; response: { ok: boolean; error?: string } };
      runInfo: { params: { input: string }; response: { ok: boolean; json?: unknown; error?: string } };
      runOptimize: {
        params: { input: string; output?: string };
        response: { ok: boolean; json?: unknown; error?: string };
      };
    };
    messages: {};
  }>;
  webview: RPCSchema<{
    requests: {};
    messages: {};
  }>;
};
