import { cp, mkdir, rm } from "node:fs/promises";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const webRoot = join(dirname(fileURLToPath(import.meta.url)), "..");
const sourceRoot = join(webRoot, "node_modules", "vditor", "dist");
const targetRoot = join(webRoot, "public", "vendor", "vditor", "dist");

await rm(targetRoot, { recursive: true, force: true });
await mkdir(targetRoot, { recursive: true });

for (const relativePath of [
  "js/lute",
  "js/icons",
  "js/i18n",
  "css/content-theme",
  "images/emoji",
]) {
  await cp(join(sourceRoot, relativePath), join(targetRoot, relativePath), {
    recursive: true,
  });
}
