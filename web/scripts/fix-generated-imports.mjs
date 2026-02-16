/**
 * Post-process swagger-typescript-api generated files to use `import type`
 * where needed. Required for verbatimModuleSyntax compatibility.
 *
 * 1. data-contracts.ts only exports types/interfaces → all imports become `import type`.
 * 2. http-client.ts exports a mix of types (RequestParams, QueryParamsType, …)
 *    and values (HttpClient, ContentType). We split the import into a value import
 *    and a type-only import.
 */
import { readdirSync, readFileSync, writeFileSync } from "fs";
import { join } from "path";

const dir = "src/api/generated";
const SKIP = new Set(["data-contracts.ts", "http-client.ts"]);

// Values exported from http-client.ts (enum + class survive erasure)
const HTTP_CLIENT_VALUES = new Set(["HttpClient", "ContentType"]);

for (const file of readdirSync(dir)) {
  if (!file.endsWith(".ts") || SKIP.has(file)) continue;
  const filePath = join(dir, file);
  let content = readFileSync(filePath, "utf8");

  // 1. data-contracts: all type-only → import type { … }
  content = content.replace(
    /import \{([^}]+)\} from "\.\/data-contracts"/gs,
    'import type {$1} from "./data-contracts"',
  );

  // 2. http-client: split into value + type imports
  content = content.replace(
    /import \{([^}]+)\} from "\.\/http-client";/gs,
    (_match, names) => {
      const items = names
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean);
      const values = items.filter((n) => HTTP_CLIENT_VALUES.has(n));
      const types = items.filter((n) => !HTTP_CLIENT_VALUES.has(n));
      const parts = [];
      if (types.length) {
        parts.push(`import type { ${types.join(", ")} } from "./http-client";`);
      }
      if (values.length) {
        parts.push(`import { ${values.join(", ")} } from "./http-client";`);
      }
      return parts.join("\n");
    },
  );

  writeFileSync(filePath, content);
}
