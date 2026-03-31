# pdfed (Raycast)

Run the **pdfed** CLI from Raycast: merge, optimize, rotate, split, inspect metadata, and append images as PDF pages.

## Requirements

- macOS with [Raycast](https://www.raycast.com/)
- [Node.js](https://nodejs.org/) **22.14+** (matches `@raycast/api`; needed to compile the extension)
- `pdfed` installed and discoverable, or an explicit path set in extension preferences

Install the CLI from this repo (for example):

```bash
go install ./...
```

Homebrew or other installs that place `pdfed` in `/opt/homebrew/bin` or `/usr/local/bin` work without configuration.

## Install the extension

Raycast runs **compiled JavaScript** next to `package.json`, not the `.tsx` sources. Pick **one** workflow:

### Option A — Development (live reload)

1. In a terminal:
   ```bash
   cd raycast/pdfed
   npm install
   npm run dev
   ```
2. Leave **`npm run dev` running** (it compiles and wires the extension into Raycast).
3. Raycast → **Import Extension** → select the `raycast/pdfed` folder (must contain `package.json`).

### Option B — One-shot build (no dev server)

1. In a terminal:
   ```bash
   cd raycast/pdfed
   npm install
   npm run build
   ```
2. That writes `merge-pdfs.js`, `optimize-pdf.js`, etc. next to `package.json`.
3. Raycast → **Import Extension** → select `raycast/pdfed` again (or reload the extension).

`npm run build` and `npm run dev` use the `ray` CLI from your local `node_modules` (`npx`), so you do **not** need a global install.

### Preferences (optional)

In **Raycast → Extensions → pdfed → Extension Preferences**:

- **pdfed binary path** if `pdfed` is not on the PATH Raycast sees
- **After a successful write**: do nothing, copy the output path, open in the default app (e.g. Preview), or both

### Troubleshooting

**“Could not find command's executable JS file”** — You opened the extension before it was compiled. Run **`npm install`** then either **`npm run dev`** (keep it running) or **`npm run build`**, then try the command again.

## Development

```bash
cd raycast/pdfed
npm install
npm run dev    # live reload while developing
npm run build  # emit *.js next to package.json (same as Option B)
npm run lint
```

## Commands

| Command | CLI |
|--------|-----|
| Merge PDFs | `pdfed --json merge …` |
| Optimize PDF | `pdfed --json optimize …` |
| Rotate PDF | `pdfed --json rotate …` |
| Split PDF | `pdfed --json split …` (always passes page flags so the terminal TUI never opens) |
| PDF Info | `pdfed --json info …` |
| Add Images to PDF | `pdfed --json add-images …` |

The extension parses JSON results, shows a **result** screen (tables + raw JSON), and uses **merge / split / …** with `--json`. Use a **pdfed** binary built from this repo so `--json` exists.

### Permissions (macOS)

Raycast runs the `pdfed` process with a **working directory next to your file** so default outputs are not written to an unwritable location.

If you still see **operation not permitted** or **permission denied** when reading or writing PDFs (common for **Desktop**, **Documents**, **Downloads**, or external volumes):

1. Open **System Settings → Privacy & Security → Full Disk Access**
2. Enable **Raycast** (and restart Raycast if needed)

Ensure the **pdfed** binary is executable (`chmod +x`) if you copied it by hand.

### Path environment

The extension runs `pdfed` with `PATH` extended to include `/opt/homebrew/bin` and `/usr/local/bin` so a Homebrew-installed `pdfed` is found even when Raycast’s environment is minimal.

Interactive-only flows (`search`, split with no arguments) are not exposed here by design.
