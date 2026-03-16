import { readFile, writeFile } from "node:fs/promises";

const targets = [
  "linux-x64",
  "linux-arm64",
  "darwin-x64",
  "darwin-arm64",
  "win32-x64",
  "win32-arm64",
];

const out = new URL('../npm/package.json', import.meta.url);
const entryPointPkgJson = JSON.parse(await readFile(out, 'utf8'));

const version = process.env.RELEASE_TAG?.replace(/^v/, '') ?? entryPointPkgJson.version;

await writeFile(out, JSON.stringify({
  ...entryPointPkgJson,
  optionalDependencies: Object.fromEntries(targets.map(t => [
    `@pwrs/asimonim-${t}`,
    version,
  ])),
}, null, 2) + '\n', 'utf8');
