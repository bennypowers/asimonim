import { platform, arch } from "node:process";
import { execSync } from "node:child_process";
import { readFileSync } from "node:fs";

const pkg = {
  "darwin-x64": "@pwrs/asimonim-darwin-x64",
  "darwin-arm64": "@pwrs/asimonim-darwin-arm64",
  "linux-x64": "@pwrs/asimonim-linux-x64",
  "linux-arm64": "@pwrs/asimonim-linux-arm64",
  "win32-x64": "@pwrs/asimonim-win32-x64",
  "win32-arm64": "@pwrs/asimonim-win32-arm64",
}[`${platform}-${arch}`];

if (!pkg) {
  console.error(
    `Unsupported platform: ${platform}-${arch}. Please check https://github.com/bennypowers/asimonim for supported platforms.`
  );
  process.exit(1);
}

const { optionalDependencies = {} } = JSON.parse(
  readFileSync(new URL("./package.json", import.meta.url), "utf8"),
);
const pkgVersion = optionalDependencies[pkg];
if (!pkgVersion) {
  console.error(`No version found for ${pkg} in optionalDependencies`);
  process.exit(1);
}

try {
  execSync(`npm install --no-save ${pkg}@${pkgVersion}`, { stdio: "inherit" });
} catch (err) {
  console.error(`Failed to install platform binary package: ${pkg}`);
  process.exit(1);
}
