import { Action, ActionPanel, Form, showToast, Toast, useNavigation } from "@raycast/api";
import { useForm } from "@raycast/utils";
import { dirname } from "path";
import { useEffect, useRef } from "react";
import PdfedResultDetail from "./components/PdfedResultDetail";
import { applyAfterSuccess } from "./lib/afterSuccess";
import { ensurePdfFiles } from "./lib/pdfed";
import { resolveUserPath } from "./lib/paths";
import { outputPathFromResult, runPdfedJSON, summarizePdfedResult } from "./lib/pdfedJson";
import { formatPdfedErrorMessage } from "./lib/spawnEnv";

type Values = {
  input: string[];
  degrees: string;
  pages: string;
  output: string;
};

export default function Command() {
  const { push } = useNavigation();
  const { handleSubmit, itemProps, values } = useForm<Values>({
    initialValues: { degrees: "90" },
    validation: {
      input: (v) => {
        if (!v?.length) return "Pick a PDF file";
        if (!v[0].toLowerCase().endsWith(".pdf")) return "Must be a PDF";
      },
    },
    async onSubmit(values) {
      const input = values.input[0];
      ensurePdfFiles([input]);
      const inputDir = dirname(input);
      const pages = values.pages.trim();
      const out = values.output.trim();
      const outAbs = out ? resolveUserPath(out, inputDir) : "";
      const cwd = dirname(outAbs || input);

      await showToast({ style: Toast.Style.Animated, title: "Rotating…" });
      const args = ["rotate", input, values.degrees];
      if (pages) args.push("-p", pages);
      if (outAbs) args.push("-o", outAbs);
      try {
        const result = await runPdfedJSON(args, { cwd });
        await showToast({
          style: Toast.Style.Success,
          title: "Rotate complete",
          message: summarizePdfedResult(result),
        });
        push(<PdfedResultDetail result={result} navigationTitle="Rotate" />);
        await applyAfterSuccess(outputPathFromResult(result));
      } catch (e) {
        const msg = formatPdfedErrorMessage(e instanceof Error ? e.message : String(e));
        await showToast({ style: Toast.Style.Failure, title: "Rotate failed", message: msg });
      }
    },
  });
  const lastAutoInputRef = useRef<string>("");

  useEffect(() => {
    const input = values.input?.[0] ?? "";
    if (!input) return;
    if (lastAutoInputRef.current === input) return;
    lastAutoInputRef.current = input;
    void handleSubmit(values);
  }, [values, handleSubmit]);

  return (
    <Form
      actions={
        <ActionPanel>
          <Action.SubmitForm title="Rotate" onSubmit={handleSubmit} />
        </ActionPanel>
      }
    >
      <Form.FilePicker title="PDF file" allowMultipleSelection={false} {...itemProps.input} />
      <Form.Dropdown title="Degrees (clockwise)" {...itemProps.degrees}>
        <Form.Dropdown.Item value="90" title="90°" />
        <Form.Dropdown.Item value="180" title="180°" />
        <Form.Dropdown.Item value="270" title="270°" />
      </Form.Dropdown>
      <Form.TextField title="Pages (optional)" placeholder="e.g. 1-3,5 — leave empty for all pages" {...itemProps.pages} />
      <Form.TextField title="Output path (optional)" placeholder="Leave empty to modify file in place" {...itemProps.output} />
    </Form>
  );
}
