import { readdir, readFile } from "node:fs/promises";
import { extname, join, relative } from "node:path";

const roots = ["src", "e2e"];
const sourceExtensions = new Set([".ts", ".tsx"]);
const explicitAny = /\bany\b/g;
const failures = [];

async function visit(directory) {
  for (const entry of await readdir(directory, { withFileTypes: true })) {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) {
      await visit(path);
      continue;
    }
    if (!sourceExtensions.has(extname(entry.name))) continue;
    const lines = (await readFile(path, "utf8")).split(/\r?\n/);
    for (let index = 0; index < lines.length; index += 1) {
      if (explicitAny.test(lines[index])) {
        failures.push(`${relative(process.cwd(), path)}:${index + 1}: ${lines[index].trim()}`);
      }
      explicitAny.lastIndex = 0;
    }
  }
}

for (const root of roots) await visit(root);
if (failures.length > 0) {
  console.error("Explicit any is not allowed:\n" + failures.join("\n"));
  process.exit(1);
}
console.log("No explicit any found in frontend TypeScript.");
