import { Action, ActionPanel, Form, showToast, Toast, useNavigation } from "@raycast/api";
import { useForm, FormValidation } from "@raycast/utils";
import { dirname } from "path";
import PdfedResultDetail from "./components/PdfedResultDetail";
import { applyAfterSuccess } from "./lib/afterSuccess";
import { ensureImageFiles } from "./lib/pdfed";
import { resolveUserPath } from "./lib/paths";
import { outputPathFromResult, runPdfedJSON, summarizePdfedResult } from "./lib/pdfedJson";
import { formatPdfedErrorMessage } from "./lib/spawnEnv";

type Values = {
  outputPdf: string;
  images: string[];
  paper: string;
};

export default function Command() {
  const { push } = useNavigation();
  const { handleSubmit, itemProps } = useForm<Values>({
    validation: {
      outputPdf: FormValidation.Required,
      images: (v) => {
        if (!v?.length) return "Pick at least one image";
      },
    },
    async onSubmit(values) {
      ensureImageFiles(values.images);
      const first = values.images[0];
      const inputDir = dirname(first);
      let rawOut = values.outputPdf.trim();
      if (!rawOut.toLowerCase().endsWith(".pdf")) rawOut += ".pdf";
      const outAbs = resolveUserPath(rawOut, inputDir);
      const cwd = dirname(outAbs);
      const paper = values.paper.trim();

      await showToast({ style: Toast.Style.Animated, title: "Adding images…" });
      const args = ["add-images", outAbs, ...values.images];
      if (paper) args.push("--paper", paper);
      try {
        const result = await runPdfedJSON(args, { cwd });
        await showToast({
          style: Toast.Style.Success,
          title: "Images added",
          message: summarizePdfedResult(result),
        });
        push(<PdfedResultDetail result={result} navigationTitle="Add images" />);
        await applyAfterSuccess(outputPathFromResult(result));
      } catch (e) {
        const msg = formatPdfedErrorMessage(e instanceof Error ? e.message : String(e));
        await showToast({ style: Toast.Style.Failure, title: "add-images failed", message: msg });
      }
    },
  });

  return (
    <Form
      actions={
        <ActionPanel>
          <Action.SubmitForm title="Add Images" onSubmit={handleSubmit} />
        </ActionPanel>
      }
    >
      <Form.TextField
        title="Output PDF path"
        placeholder="/path/to/doc.pdf or out.pdf (next to first image)"
        {...itemProps.outputPdf}
      />
      <Form.FilePicker title="Images" allowMultipleSelection {...itemProps.images} />
      <Form.TextField title="Paper size (optional)" placeholder="e.g. A4, Letter, A4L" {...itemProps.paper} />
    </Form>
  );
}
