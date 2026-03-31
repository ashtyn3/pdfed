import { Action, ActionPanel, Detail, Icon, open, showInFinder } from "@raycast/api";
import { existsSync, statSync } from "fs";
import { markdownFromPdfedResult, outputPathFromResult, type PdfedJSONSuccess } from "../lib/pdfedJson";

type Props = {
  result: PdfedJSONSuccess;
  navigationTitle: string;
};

export default function PdfedResultDetail({ result, navigationTitle }: Props) {
  const p = outputPathFromResult(result);
  const canReveal = p && existsSync(p);
  const canOpenFile = Boolean(p && existsSync(p) && statSync(p).isFile());

  return (
    <Detail
      navigationTitle={navigationTitle}
      markdown={markdownFromPdfedResult(result)}
      actions={
        <ActionPanel>
          {canReveal ? (
            <>
              {canOpenFile ? <Action title="Open in Default App" icon={Icon.Document} onAction={() => open(p!)} /> : null}
              <Action
                title="Show in Finder"
                icon={Icon.Finder}
                shortcut={{ modifiers: ["cmd", "shift"], key: "f" }}
                onAction={() => showInFinder(p!)}
              />
            </>
          ) : null}
          {typeof p === "string" && p ? (
            <Action.CopyToClipboard
              title="Copy Path to Clipboard"
              content={p}
              shortcut={{ modifiers: ["cmd"], key: "c" }}
            />
          ) : null}
        </ActionPanel>
      }
    />
  );
}
