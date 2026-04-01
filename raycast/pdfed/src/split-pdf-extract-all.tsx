import { SplitPdfForm } from "./split-pdf";

export default function Command() {
  return (
    <SplitPdfForm
      initialMode="extractAll"
      lockMode
      navigationTitle="Split (Extract All)"
      submitTitle="Extract All Pages"
    />
  );
}
