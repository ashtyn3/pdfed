import { Action, ActionPanel, Form, showToast, Toast, useNavigation } from "@raycast/api";
import { useForm, FormValidation } from "@raycast/utils";
import { dirname, join } from "path";
import PdfedResultDetail from "./components/PdfedResultDetail";
import { applyAfterSuccess } from "./lib/afterSuccess";
import { ensurePdfFiles } from "./lib/pdfed";
import { outputPathFromResult, runPdfedJSON, summarizePdfedResult } from "./lib/pdfedJson";
import { formatPdfedErrorMessage } from "./lib/spawnEnv";

type Values = {
  inputs: string[];
  outputName: string;
  overwrite: boolean;
};

export default function Command() {
  const { push } = useNavigation();
  const { handleSubmit, itemProps } = useForm<Values>({
    initialValues: {
      outputName: "merged.pdf",
      overwrite: false,
    },
    validation: {
      inputs: (v) => {
        const pdfs = (v ?? []).filter((p) => p.toLowerCase().endsWith(".pdf"));
        if (pdfs.length < 2) return "Pick at least two PDF files";
      },
      outputName: FormValidation.Required,
    },
    async onSubmit(values) {
      const inputs = values.inputs.filter((p) => p.toLowerCase().endsWith(".pdf"));
      ensurePdfFiles(inputs, "input PDF");
      let name = values.outputName.trim();
      if (!name.toLowerCase().endsWith(".pdf")) name += ".pdf";
      const outPath = join(dirname(inputs[0]), name);
      const cwd = dirname(inputs[0]);

      await showToast({ style: Toast.Style.Animated, title: "Merging…" });
      const args = ["merge", outPath, ...inputs];
      if (values.overwrite) args.push("-f");
      try {
        const result = await runPdfedJSON(args, { cwd });
        await showToast({
          style: Toast.Style.Success,
          title: "Merge complete",
          message: summarizePdfedResult(result),
        });
        push(<PdfedResultDetail result={result} navigationTitle="Merge" />);
        await applyAfterSuccess(outputPathFromResult(result));
      } catch (e) {
        const msg = formatPdfedErrorMessage(e instanceof Error ? e.message : String(e));
        await showToast({ style: Toast.Style.Failure, title: "Merge failed", message: msg });
      }
    },
  });

  return (
    <Form
      actions={
        <ActionPanel>
          <Action.SubmitForm title="Merge" onSubmit={handleSubmit} />
        </ActionPanel>
      }
    >
      <Form.FilePicker title="PDF files" allowMultipleSelection {...itemProps.inputs} />
      <Form.TextField title="Output filename" placeholder="merged.pdf" {...itemProps.outputName} />
      <Form.Checkbox title="Overwrite if output exists" label="Use -f" {...itemProps.overwrite} />
    </Form>
  );
}
