import { Action, ActionPanel, Form, showToast, Toast, useNavigation } from "@raycast/api";
import { useForm } from "@raycast/utils";
import { dirname } from "path";
import PdfedResultDetail from "./components/PdfedResultDetail";
import { applyAfterSuccess } from "./lib/afterSuccess";
import { ensurePdfFiles } from "./lib/pdfed";
import { resolveUserPath } from "./lib/paths";
import { outputPathFromResult, runPdfedJSON, summarizePdfedResult } from "./lib/pdfedJson";
import { predictSplitOutputPath } from "./lib/splitOutput";
import { formatPdfedErrorMessage } from "./lib/spawnEnv";

type Values = {
  input: string[];
  /** printed | pdfindex | extractAll */
  splitMode: string;
  ranges: string;
  output: string;
};

type SplitMode = "printed" | "pdfindex" | "extractAll";

type SplitPdfFormProps = {
  initialMode?: SplitMode;
  lockMode?: boolean;
  navigationTitle?: string;
  submitTitle?: string;
};

export function SplitPdfForm({
  initialMode = "printed",
  lockMode = false,
  navigationTitle = "Split",
  submitTitle = "Split",
}: SplitPdfFormProps) {
  const { push } = useNavigation();
  const { handleSubmit, itemProps, values, setValidationError } = useForm<Values>({
    initialValues: { splitMode: initialMode },
    validation: {
      input: (v) => {
        if (!v?.length) return "Pick a PDF file";
        if (!v[0].toLowerCase().endsWith(".pdf")) return "Must be a PDF";
      },
    },
    async onSubmit(v) {
      if (v.splitMode !== "extractAll" && !v.ranges?.trim()) {
        setValidationError("ranges", "Enter page ranges (e.g. 1-5,7)");
        return false;
      }
      const input = v.input[0];
      ensurePdfFiles([input]);
      const inputDir = dirname(input);
      const out = v.output.trim();
      const outAbs = out ? resolveUserPath(out, inputDir) : "";
      const cwd = dirname(outAbs || input);

      await showToast({ style: Toast.Style.Animated, title: "Splitting…" });
      const args = ["split", input];
      if (v.splitMode === "printed") {
        args.push("-p", v.ranges.trim());
        if (outAbs) args.push("-o", outAbs);
      } else if (v.splitMode === "pdfindex") {
        args.push("-P", v.ranges.trim());
        if (outAbs) args.push("-o", outAbs);
      } else {
        args.push("-e");
        if (outAbs) args.push("-o", outAbs);
      }

      try {
        const result = await runPdfedJSON(args, { cwd });
        await showToast({
          style: Toast.Style.Success,
          title: `${navigationTitle} complete`,
          message: summarizePdfedResult(result),
        });
        push(<PdfedResultDetail result={result} navigationTitle={navigationTitle} />);
        await applyAfterSuccess(
          outputPathFromResult(result) ?? predictSplitOutputPath(input, v.splitMode, v.ranges, outAbs || out),
        );
      } catch (e) {
        const msg = formatPdfedErrorMessage(e instanceof Error ? e.message : String(e));
        await showToast({ style: Toast.Style.Failure, title: "Split failed", message: msg });
      }
    },
  });

  const extractAll = values.splitMode === "extractAll";

  return (
    <Form
      actions={
        <ActionPanel>
          <Action.SubmitForm title={submitTitle} onSubmit={handleSubmit} />
        </ActionPanel>
      }
    >
      <Form.FilePicker title="PDF file" allowMultipleSelection={false} {...itemProps.input} />
      {!lockMode ? (
        <Form.Dropdown title="Mode" {...itemProps.splitMode}>
          <Form.Dropdown.Item value="printed" title="Printed page numbers (-p)" />
          <Form.Dropdown.Item value="pdfindex" title="PDF page index (-P)" />
          <Form.Dropdown.Item value="extractAll" title="Each page to its own file (-e)" />
        </Form.Dropdown>
      ) : null}
      <Form.TextField
        title="Page ranges"
        placeholder={extractAll ? "Not used for extract-all" : "e.g. 1-5,7,10-12"}
        {...itemProps.ranges}
      />
      <Form.TextField
        title="Output file or directory (optional)"
        placeholder="Default name/path chosen by pdfed"
        {...itemProps.output}
      />
    </Form>
  );
}

export default function Command() {
  return <SplitPdfForm />;
}
