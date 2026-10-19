// Lets Node run Pele's TypeScript sources as they are: an import of `x.js` that is not there
// finds `x.ts`, and a JSON file is a module with its fields as named exports.
import { existsSync, readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';

export async function resolve(spec, ctx, next) {
  if (spec.endsWith('.js') && ctx.parentURL?.endsWith('.ts')) {
    const url = new URL(spec, ctx.parentURL);
    const ts = url.href.slice(0, -3) + '.ts';
    if (!existsSync(fileURLToPath(url)) && existsSync(fileURLToPath(ts))) return next(ts, ctx);
  }
  return next(spec, ctx);
}

// Pele's num writes a number to two decimals. A suite that reads numbers back out of an SVG
// to work with them sets globalThis.limuExactNumbers while Pele draws, and num then writes
// them whole; the suite rounds them as num would once it is done with them.
const NUM = 'export function num(value: number): string {';

export async function load(url, ctx, next) {
  if (url.endsWith('/src/svg/builder.ts')) {
    const loaded = await next(url, ctx);
    const source = String(loaded.source);
    if (!source.includes(NUM)) throw new Error(`no num in ${url}`);
    const exact = `${NUM}\n  if ((globalThis as any).limuExactNumbers) return String(value);`;
    return { ...loaded, source: source.replace(NUM, exact), shortCircuit: true };
  }
  if (url.endsWith('.json')) {
    const data = JSON.parse(readFileSync(fileURLToPath(url), 'utf8'));
    let source = `export default ${JSON.stringify(data)};`;
    for (const [k, v] of Object.entries(data)) {
      if (/^[A-Za-z_$][\w$]*$/.test(k)) source += `export const ${k} = ${JSON.stringify(v)};`;
    }
    return { format: 'module', source, shortCircuit: true };
  }
  return next(url, ctx);
}
