#!/usr/bin/env node
/**
 * Compatibility gate for event JSON Schemas.
 * A newer schema must keep every previously required property required
 * (additive optional fields are allowed; dropping required fields is not).
 */
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const eventsDir = path.join(root, "contracts/events");
const baselineDir = path.join(eventsDir, "baseline");

function loadSchema(filePath) {
  return JSON.parse(fs.readFileSync(filePath, "utf8"));
}

function requiredOf(schema) {
  return new Set(Array.isArray(schema.required) ? schema.required : []);
}

const schemas = fs
  .readdirSync(eventsDir)
  .filter((name) => name.endsWith(".schema.json"))
  .sort();

if (schemas.length === 0) {
  console.error("no event schemas found");
  process.exit(1);
}

let failed = false;
for (const name of schemas) {
  const current = loadSchema(path.join(eventsDir, name));
  const baselinePath = path.join(baselineDir, name);
  if (!fs.existsSync(baselinePath)) {
    console.log(`no baseline for ${name}; treating current as baseline candidate`);
    continue;
  }
  const baseline = loadSchema(baselinePath);
  const baselineRequired = requiredOf(baseline);
  const currentRequired = requiredOf(current);
  for (const field of baselineRequired) {
    if (!currentRequired.has(field)) {
      console.error(`${name}: required field "${field}" was removed`);
      failed = true;
    }
  }
  const baselineProps = baseline.properties ?? {};
  const currentProps = current.properties ?? {};
  for (const field of Object.keys(baselineProps)) {
    if (!(field in currentProps)) {
      console.error(`${name}: property "${field}" was removed`);
      failed = true;
    }
  }
}

if (failed) {
  process.exit(1);
}
console.log("event schema compatibility checks passed");
