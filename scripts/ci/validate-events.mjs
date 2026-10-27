#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
import { createRequire } from "node:module";
import { fileURLToPath } from "node:url";
import { spawnSync } from "node:child_process";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const toolDir = path.dirname(fileURLToPath(import.meta.url));
const eventsDir = path.join(root, "contracts/events");
const examplesDir = path.join(eventsDir, "examples");

function ensureAjv() {
  const marker = path.join(toolDir, "package.json");
  if (!fs.existsSync(marker)) {
    fs.writeFileSync(marker, JSON.stringify({ name: "ci-event-tools", private: true }, null, 2));
  }
  try {
    createRequire(marker).resolve("ajv/dist/2020.js");
    createRequire(marker).resolve("ajv-formats");
  } catch {
    const install = spawnSync(
      "npm",
      ["install", "--no-save", "--no-package-lock", "ajv@8.17.1", "ajv-formats@3.0.1"],
      { cwd: toolDir, stdio: "inherit" },
    );
    if (install.status !== 0) {
      process.exit(install.status ?? 1);
    }
  }
}

ensureAjv();

const require = createRequire(path.join(toolDir, "package.json"));
const Ajv2020 = require("ajv/dist/2020.js");
const addFormats = require("ajv-formats");

function compiler() {
  const ajv = new Ajv2020({ allErrors: true, strict: false });
  addFormats(ajv);
  return ajv;
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
  const schemaPath = path.join(eventsDir, name);
  const schema = JSON.parse(fs.readFileSync(schemaPath, "utf8"));
  try {
    compiler().compile(schema);
    console.log(`compiled ${name}`);
  } catch (error) {
    console.error(`schema compile failed for ${name}:`, error.message);
    failed = true;
  }
}

const examples = fs.existsSync(examplesDir)
  ? fs.readdirSync(examplesDir).filter((name) => name.endsWith(".json")).sort()
  : [];
if (examples.length === 0) {
  console.error("no event examples found");
  process.exit(1);
}

for (const name of examples) {
  const base = name.replace(/\.json$/, "");
  const schemaPath = path.join(eventsDir, `${base}.schema.json`);
  if (!fs.existsSync(schemaPath)) {
    console.error(`missing schema for example ${name}`);
    failed = true;
    continue;
  }
  const schema = JSON.parse(fs.readFileSync(schemaPath, "utf8"));
  const validate = compiler().compile(schema);
  const data = JSON.parse(fs.readFileSync(path.join(examplesDir, name), "utf8"));
  if (!validate(data)) {
    console.error(`example ${name} failed:`, validate.errors);
    failed = true;
  } else {
    console.log(`validated ${name}`);
  }
}

const compatibility = spawnSync(process.execPath, [path.join(root, "scripts/ci/check-event-compatibility.mjs")], {
  cwd: root,
  stdio: "inherit",
});
if (compatibility.status !== 0) {
  failed = true;
}

if (failed) {
  process.exit(1);
}
console.log("event schema checks passed");
