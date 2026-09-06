#!/bin/sh
set -eu

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT
npm pack --dry-run --json > "$tmp"
node - "$tmp" <<'NODE'
const fs = require("node:fs");
const actual = new Set(JSON.parse(fs.readFileSync(process.argv[2], "utf8"))[0].files.map(file => file.path));
const expected = new Set([
  "bin/ainovel-cli-vn.js",
  "LICENSE",
  "README.md",
  "package.json",
]);
const unexpected = [...actual].filter(file => !expected.has(file));
const missing = [...expected].filter(file => !actual.has(file));
if (unexpected.length || missing.length || actual.size !== expected.size) {
  console.error(JSON.stringify({ unexpected, missing }));
  process.exit(1);
}
NODE

node -e '
const fs = require("node:fs");
const packageJSON = JSON.parse(fs.readFileSync("package.json", "utf8"));
const lockJSON = JSON.parse(fs.readFileSync("package-lock.json", "utf8"));
if (packageJSON.name !== "ainovel-cli-vn" || packageJSON.license !== "Apache-2.0") process.exit(1);
if (lockJSON.name !== packageJSON.name || lockJSON.version !== packageJSON.version || lockJSON.packages[""].version !== packageJSON.version) process.exit(1);
'
exit 0
