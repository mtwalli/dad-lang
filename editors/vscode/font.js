// Installs Ḍād Mono, the coding font bundled with the extension, for the current user.
// VS Code can only use fonts installed on the computer, so the extension offers to copy
// the font files into the user's own font folder. No administrator rights are needed.
"use strict";

const vscode = require("vscode");
const cp = require("child_process");
const fs = require("fs");
const os = require("os");
const path = require("path");

const FILES = ["DadMono-Regular.ttf", "DadMono-Bold.ttf"];
const PROMPT_KEY = "dad.fontPrompt"; // "never" once the user says not to ask again
let asked = false; // ask at most once per session

/** The folder fonts are installed into for this user. */
function userFontDir() {
  switch (process.platform) {
    case "darwin":
      return path.join(os.homedir(), "Library", "Fonts");
    case "win32":
      return path.join(process.env.LOCALAPPDATA || path.join(os.homedir(), "AppData", "Local"), "Microsoft", "Windows", "Fonts");
    default:
      return path.join(process.env.XDG_DATA_HOME || path.join(os.homedir(), ".local", "share"), "fonts");
  }
}

/** Folders where an installed copy may already be, user's own first. */
function fontDirs() {
  const dirs = [userFontDir()];
  if (process.platform === "darwin") dirs.push("/Library/Fonts");
  else if (process.platform === "win32") dirs.push(path.join(process.env.WINDIR || "C:\\Windows", "Fonts"));
  else dirs.push(path.join(os.homedir(), ".fonts"), "/usr/local/share/fonts", "/usr/share/fonts");
  return dirs;
}

function isInstalled() {
  return fontDirs().some((dir) => fs.existsSync(path.join(dir, FILES[0])));
}

/** Copies the font files and registers them where the system needs it. Returns the folder. */
function install(extensionPath) {
  const dest = userFontDir();
  fs.mkdirSync(dest, { recursive: true });
  for (const file of FILES) {
    fs.copyFileSync(path.join(extensionPath, "fonts", file), path.join(dest, file));
  }
  if (process.platform === "win32") {
    // Per-user fonts on Windows must also be listed in the registry.
    const key = "HKCU\\Software\\Microsoft\\Windows NT\\CurrentVersion\\Fonts";
    for (const [file, name] of [[FILES[0], "Dad Mono (TrueType)"], [FILES[1], "Dad Mono Bold (TrueType)"]]) {
      cp.execFileSync("reg", ["add", key, "/v", name, "/t", "REG_SZ", "/d", path.join(dest, file), "/f"], { windowsHide: true });
    }
  } else if (process.platform !== "darwin") {
    try {
      cp.execFileSync("fc-cache", ["-f", dest], { stdio: "ignore" });
    } catch {
      // fontconfig picks the files up on its next scan anyway.
    }
  }
  return dest;
}

async function installCommand(context) {
  try {
    const dest = install(context.extensionPath);
    const choice = await vscode.window.showInformationMessage(
      `Ḍād Mono is installed in ${dest}. Reload VS Code to use it for .dad files.`,
      "Reload now"
    );
    if (choice === "Reload now") vscode.commands.executeCommand("workbench.action.reloadWindow");
  } catch (err) {
    vscode.window.showErrorMessage(
      `Ḍād: couldn't install the font (${err.message}). You can install it by hand: see the editor guide.`,
      "Open guide"
    ).then((c) => {
      if (c) vscode.env.openExternal(vscode.Uri.parse("https://mtwalli.github.io/dad-lang/editor/"));
    });
  }
}

/** Offers the font once per session, when a .dad file is open and the font is missing. */
async function maybeOffer(context, doc) {
  if (asked || !doc || doc.languageId !== "dad") return;
  if (context.globalState.get(PROMPT_KEY) === "never" || isInstalled()) return;
  asked = true;
  const choice = await vscode.window.showInformationMessage(
    "Install Ḍād Mono? It's the coding font made for Ḍād: Kufi Arabic that lines up with Latin letters and digits.",
    "Install font",
    "Not now",
    "Don't ask again"
  );
  if (choice === "Install font") await installCommand(context);
  else if (choice === "Don't ask again") await context.globalState.update(PROMPT_KEY, "never");
}

module.exports = { installCommand, maybeOffer, isInstalled, userFontDir, FILES, _reset: () => (asked = false) };
