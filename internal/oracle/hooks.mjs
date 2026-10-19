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
// them whole; the suite rounds them as num would once it is done with them. While it is set,
// classMarker also writes what it draws a marker from, so that the marker can be drawn again
// where a group moves it.
const patch = (url, source, find, replace) => {
  if (!source.includes(find)) throw new Error(`no ${find} in ${url}`);
  return source.replace(find, replace);
};
const NUM = 'export function num(value: number): string {';
const EXACT = 'if ((globalThis as any).limuExactNumbers)';
const CLASS_MARKER = 'export function classMarker(';

export async function load(url, ctx, next) {
  if (url.endsWith('/src/svg/builder.ts')) {
    const loaded = await next(url, ctx);
    const source = patch(url, String(loaded.source), NUM, `${NUM}\n  ${EXACT} return String(value);`);
    return { ...loaded, source, shortCircuit: true };
  }
  if (url.endsWith('/src/diagrams/class/markers.ts')) {
    const loaded = await next(url, ctx);
    const source =
      patch(url, String(loaded.source), CLASS_MARKER, 'function drawMarker(') +
      `
export function classMarker(type: EndMarker, x: number, y: number, dx: number, dy: number, color: string): string {
  const s = drawMarker(type, x, y, dx, dy, color);
  ${EXACT} return s.replace('class="pele-marker"', \`class="pele-marker" data-m="\${type} \${x} \${y} \${dx} \${dy}"\`);
  return s;
}
`;
    return { ...loaded, source, shortCircuit: true };
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
