import * as path from "node:path";
import * as os from "node:os";
import { ExtensionContext, workspace, window } from "vscode";

import {
  LanguageClient,
  LanguageClientOptions,
  TransportKind,
} from "vscode-languageclient/node";

let client: LanguageClient | undefined;

const SETTINGS_MAP = [
  "tokensFiles",
  "prefix",
  "groupMarkers",
  "networkFallback",
  "networkTimeout",
  "cdn",
] as const;

async function migrateSettings() {
  const asimonimConfig = workspace.getConfiguration("asimonim");
  if (!asimonimConfig.get<boolean>("migrateFromDTLS", true)) {
    return;
  }

  const oldConfig = workspace.getConfiguration("designTokensLanguageServer");
  let migrated = false;

  for (const key of SETTINGS_MAP) {
    const oldValue = oldConfig.inspect(key);
    if (!oldValue) continue;

    try {
      // Migrate workspace settings
      if (oldValue.workspaceValue !== undefined) {
        const newValue = asimonimConfig.inspect(key);
        if (newValue?.workspaceValue === undefined) {
          await asimonimConfig.update(key, oldValue.workspaceValue, false);
          migrated = true;
        }
      }

      // Migrate global settings
      if (oldValue.globalValue !== undefined) {
        const newValue = asimonimConfig.inspect(key);
        if (newValue?.globalValue === undefined) {
          await asimonimConfig.update(key, oldValue.globalValue, true);
          migrated = true;
        }
      }
    } catch (err) {
      console.warn(`Failed to migrate setting "${key}": ${err}`);
    }
  }

  if (migrated) {
    window.showInformationMessage(
      "Migrated settings from designTokensLanguageServer to asimonim. " +
        "You can remove the old designTokensLanguageServer settings.",
    );
  }
}

export async function activate(context: ExtensionContext) {
  await migrateSettings();

  const platform = os.platform();
  const arch = os.arch();

  // Determine the OS-specific binary name
  // Uses standard platform naming: linux-x64, darwin-arm64, win32-x64, etc.
  const binaryName = (() => {
    const archMapping: Record<string, string> = {
      arm64: "arm64",
      x64: "x64",
    };

    const osMapping: Record<string, string> = {
      darwin: "darwin",
      linux: "linux",
      win32: "win32",
    };

    const archSuffix = archMapping[arch];
    const osSuffix = osMapping[platform];

    if (!archSuffix || !osSuffix) {
      throw new Error(
        `Unsupported platform or architecture: ${platform}-${arch}`,
      );
    }

    const ext = platform === "win32" ? ".exe" : "";
    return `asimonim-${osSuffix}-${archSuffix}${ext}`;
  })();

  const command = context.asAbsolutePath(
    path.join("dist", "bin", binaryName),
  );

  const args = ["lsp"];

  const clientOptions: LanguageClientOptions = {
    documentSelector: [
      { scheme: "file", language: "css" },
      { scheme: "file", language: "html" },
      { scheme: "file", language: "javascript" },
      { scheme: "file", language: "javascriptreact" },
      { scheme: "file", language: "typescript" },
      { scheme: "file", language: "typescriptreact" },
      { scheme: "file", language: "json" },
      { scheme: "file", language: "yaml" },
    ],
  };

  client = new LanguageClient(
    "asimonim",
    "Design Tokens Language Server",
    {
      run: { command, args, transport: TransportKind.stdio },
      debug: { command, args, transport: TransportKind.stdio },
    },
    clientOptions,
  );

  try {
    await client.start();
  } catch (error) {
    const message = error instanceof Error ? error.message : String(error);
    window.showErrorMessage(`Design Tokens Language Server failed to start: ${message}`);
    try { await client.stop(); } catch { /* ignore stop errors */ }
    client = undefined;
  }
}

export function deactivate(): Thenable<void> | undefined {
  if (!client) {
    return undefined;
  }
  return client.stop();
}
