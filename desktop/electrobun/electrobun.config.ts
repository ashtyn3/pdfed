import type { ElectrobunConfig } from "electrobun";

const codesignEnabled = process.env.ELECTROBUN_CODESIGN === "1";
const notarizeEnabled = process.env.ELECTROBUN_NOTARIZE === "1";

export default {
  app: {
    name: "pdfed",
    identifier: "dev.pdfed.desktop",
    version: "0.1.0",
  },
  build: {
    views: {
      mainview: {
        entrypoint: "src/mainview/noop.ts",
      },
    },
    copy: {
      "dist/mainview/src/mainview/index.html": "views/mainview/index.html",
      "dist/mainview/app.js": "views/mainview/app.js",
      "dist/mainview/app.css": "views/mainview/app.css",
    },
    mac: {
      codesign: codesignEnabled,
      notarize: notarizeEnabled,
      // Electrobun/Bun runtime needs JIT-related entitlements for signed builds.
      entitlements: {
        "com.apple.security.cs.allow-jit": true,
        "com.apple.security.cs.allow-unsigned-executable-memory": true,
        "com.apple.security.cs.disable-library-validation": true,
      },
      bundleCEF: false,
    },
    linux: {
      bundleCEF: false,
    },
    win: {
      bundleCEF: false,
    },
  },
} satisfies ElectrobunConfig;
