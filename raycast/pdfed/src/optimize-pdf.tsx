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
  output: string;
};

export default function Command() {
  const { push } = useNavigation();
  const { handleSubmit, itemProps, values } = useForm<Values>({
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
      const out = values.output.trim();
      const outAbs = out ? resolveUserPath(out, inputDir) : "";
      const cwd = dirname(outAbs || input);

      await showToast({ style: Toast.Style.Animated, title: "Optimizing…" });
      const args = ["optimize", input];
      if (outAbs) args.push("-o", outAbs);
      try {
        const result = await runPdfedJSON(args, { cwd });
        await showToast({
          style: Toast.Style.Success,
          title: "Optimize complete",
          message: summarizePdfedResult(result),
        });
        push(<PdfedResultDetail result={result} navigationTitle="Optimize" />);
        await applyAfterSuccess(outputPathFromResult(result));
      } catch (e) {
        const msg = formatPdfedErrorMessage(e instanceof Error ? e.message : String(e));
        await showToast({ style: Toast.Style.Failure, title: "Optimize failed", message: msg });
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
          <Action.SubmitForm title="Optimize" onSubmit={handleSubmit} />
        </ActionPanel>
      }
    >
      <Form.FilePicker title="PDF file" allowMultipleSelection={false} {...itemProps.input} />
      <Form.TextField
        title="Save as (optional)"
        placeholder="Leave empty to replace file in place"
        {...itemProps.output}
      />
    </Form>
  );
}
