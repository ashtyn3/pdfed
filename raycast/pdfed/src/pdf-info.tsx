import { Action, ActionPanel, Form, showToast, Toast, useNavigation } from "@raycast/api";
import { useForm } from "@raycast/utils";
import { dirname } from "path";
import { useEffect, useRef } from "react";
import PdfedResultDetail from "./components/PdfedResultDetail";
import { ensurePdfFiles } from "./lib/pdfed";
import { runPdfedJSON } from "./lib/pdfedJson";
import { formatPdfedErrorMessage } from "./lib/spawnEnv";

type Values = { input: string[] };

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
      const cwd = dirname(input);
      await showToast({ style: Toast.Style.Animated, title: "Reading PDF…" });
      try {
        const result = await runPdfedJSON(["info", input], { cwd });
        await showToast({ style: Toast.Style.Success, title: "PDF info loaded" });
        push(<PdfedResultDetail result={result} navigationTitle="PDF Info" />);
      } catch (e) {
        const msg = formatPdfedErrorMessage(e instanceof Error ? e.message : String(e));
        await showToast({ style: Toast.Style.Failure, title: "Info failed", message: msg });
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
          <Action.SubmitForm title="Show Info" onSubmit={handleSubmit} />
        </ActionPanel>
      }
    >
      <Form.FilePicker title="PDF file" allowMultipleSelection={false} {...itemProps.input} />
    </Form>
  );
}
