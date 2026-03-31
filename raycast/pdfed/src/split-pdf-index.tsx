import { SplitPdfForm } from "./split-pdf";

export default function Command() {
  return (
    <SplitPdfForm
      initialMode="pdfindex"
      lockMode
      navigationTitle="Split (PDF Index)"
      submitTitle="Split (PDF Index)"
    />
  );
}
