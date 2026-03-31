import { SplitPdfForm } from "./split-pdf";

export default function Command() {
  return (
    <SplitPdfForm
      initialMode="printed"
      lockMode
      navigationTitle="Split (Printed Pages)"
      submitTitle="Split (Printed)"
    />
  );
}
