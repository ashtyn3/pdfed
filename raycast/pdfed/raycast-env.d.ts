/// <reference types="@raycast/api">

/* 🚧 🚧 🚧
 * This file is auto-generated from the extension's manifest.
 * Do not modify manually. Instead, update the `package.json` file.
 * 🚧 🚧 🚧 */

/* eslint-disable @typescript-eslint/ban-types */

type ExtensionPreferences = {
  /** pdfed binary path - Optional absolute path to the pdfed executable. Leave empty to auto-detect (Homebrew, /usr/local, PATH). */
  "pdfedPath"?: string,
  /** After a successful write - Copy the output path, open it in the default app (e.g. Preview), both, or neither. */
  "afterSuccess": "none" | "copy" | "open" | "copy_and_open"
}

/** Preferences accessible in all the extension's commands */
declare type Preferences = ExtensionPreferences

declare namespace Preferences {
  /** Preferences accessible in the `merge-pdfs` command */
  export type MergePdfs = ExtensionPreferences & {}
  /** Preferences accessible in the `optimize-pdf` command */
  export type OptimizePdf = ExtensionPreferences & {}
  /** Preferences accessible in the `rotate-pdf` command */
  export type RotatePdf = ExtensionPreferences & {}
  /** Preferences accessible in the `split-pdf` command */
  export type SplitPdf = ExtensionPreferences & {}
  /** Preferences accessible in the `split-pdf-printed` command */
  export type SplitPdfPrinted = ExtensionPreferences & {}
  /** Preferences accessible in the `split-pdf-index` command */
  export type SplitPdfIndex = ExtensionPreferences & {}
  /** Preferences accessible in the `split-pdf-extract-all` command */
  export type SplitPdfExtractAll = ExtensionPreferences & {}
  /** Preferences accessible in the `pdf-info` command */
  export type PdfInfo = ExtensionPreferences & {}
  /** Preferences accessible in the `add-images` command */
  export type AddImages = ExtensionPreferences & {}
}

declare namespace Arguments {
  /** Arguments passed to the `merge-pdfs` command */
  export type MergePdfs = {}
  /** Arguments passed to the `optimize-pdf` command */
  export type OptimizePdf = {}
  /** Arguments passed to the `rotate-pdf` command */
  export type RotatePdf = {}
  /** Arguments passed to the `split-pdf` command */
  export type SplitPdf = {}
  /** Arguments passed to the `split-pdf-printed` command */
  export type SplitPdfPrinted = {}
  /** Arguments passed to the `split-pdf-index` command */
  export type SplitPdfIndex = {}
  /** Arguments passed to the `split-pdf-extract-all` command */
  export type SplitPdfExtractAll = {}
  /** Arguments passed to the `pdf-info` command */
  export type PdfInfo = {}
  /** Arguments passed to the `add-images` command */
  export type AddImages = {}
}

