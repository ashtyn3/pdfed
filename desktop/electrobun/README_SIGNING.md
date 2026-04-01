# macOS Codesign + Notarization

This app is configured to enable signing/notarization with env flags:

- `ELECTROBUN_CODESIGN=1` turns on `build.mac.codesign`
- `ELECTROBUN_NOTARIZE=1` turns on `build.mac.notarize`

## 1) Required signing env var

Set your Developer ID certificate name:

`ELECTROBUN_DEVELOPER_ID="Developer ID Application: Your Name (TEAMID)"`

## 2) Notarization credentials (choose one method)

### App Store Connect API key (recommended CI path)

- `ELECTROBUN_APPLEAPIISSUER`
- `ELECTROBUN_APPLEAPIKEY`
- `ELECTROBUN_APPLEAPIKEYPATH` (absolute path to `.p8`)

### Apple ID + app-specific password

- `ELECTROBUN_APPLEID`
- `ELECTROBUN_APPLEIDPASS`
- `ELECTROBUN_TEAMID`

## 3) Build commands

- Signed app only:
  - `bun run build:mac:signed`
- Signed + notarized app:
  - `bun run build:mac:notarized`

## 4) Optional local verification

- Verify signature:
  - `codesign --verify --deep --strict --verbose=2 "build/release-macos-arm64/pdfed.app"`
- Gatekeeper check:
  - `spctl --assess --type execute --verbose "build/release-macos-arm64/pdfed.app"`

