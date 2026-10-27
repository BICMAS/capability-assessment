import { dirname } from "node:path";
import { fileURLToPath } from "node:url";

import { FlatCompat } from "@eslint/eslintrc";

const __filename = fileURLToPath(import.meta.url);
const __dirname = dirname(__filename);
const compat = new FlatCompat({ baseDirectory: __dirname });

/** @type {import("eslint").Linter.Config[]} */
const config = [
  { ignores: [".next/**", "node_modules/**", "lib/generated/**", "next-env.d.ts", "e2e/**"] },
  ...compat.extends("next/core-web-vitals", "next/typescript"),
];

export default config;
