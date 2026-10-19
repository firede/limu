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
// where a group moves it, and a kanban board's textBlock writes, in place of each line of
// text, where the line starts or ends and what is in it, as the scene holds a label. A radar
// chart's labelSvg also writes, before each label, the point it is centered on, its size and
// the text of its lines, as the scene holds them.
const patch = (url, source, find, replace) => {
  if (!source.includes(find)) throw new Error(`no ${find} in ${url}`);
  return source.replace(find, replace);
};
const NUM = 'export function num(value: number): string {';
const EXACT = 'if ((globalThis as any).limuExactNumbers)';
const CLASS_MARKER = 'export function classMarker(';
const TEXT_BLOCK = 'function textBlock(';
const RADAR_IMPORT = "import { escText, labelSvg, num } from '../../svg/builder.js';";

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
  if (url.endsWith('/src/diagrams/kanban/render.ts')) {
    const loaded = await next(url, ctx);
    const source =
      patch(url, String(loaded.source), TEXT_BLOCK, 'function textBlockPele(') +
      `
function textBlock(label: Label, x: number, y: number, attrs: string, icons: IconResolver | undefined, end = false): string {
  if (!(globalThis as any).limuExactNumbers) return textBlockPele(label, x, y, attrs, icons, end);
  let out = '';
  for (let i = 0; i < label.lines.length; i++) {
    const spans = label.lines[i]
      .map((s) => (s.icon !== undefined ? 'i:' + s.icon + ':' + s.width : 't:' + s.style + ':' + s.text + ':' + s.width))
      .join('|');
    out += \`<limu-line x="\${x}" y="\${y + (i + 0.5) * label.lineHeight}" end="\${end ? 1 : 0}" w="\${label.widths[i]}" attrs="\${encodeURIComponent(attrs)}" spans="\${encodeURIComponent(spans)}"/>\`;
  }
  return out;
}
`;
    return { ...loaded, source, shortCircuit: true };
  }
  if (url.endsWith('/src/diagrams/radar/render.ts')) {
    const loaded = await next(url, ctx);
    const source =
      patch(url, String(loaded.source), RADAR_IMPORT, RADAR_IMPORT.replace('labelSvg', 'labelSvg as labelSvgPele')) +
      `
function labelSvg(label: Label, cx: number, cy: number, attrs: string): string {
  const s = labelSvgPele(label, cx, cy, attrs);
  if (!(globalThis as any).limuExactNumbers || s === '') return s;
  const lines = label.lines.map((line) => line.map((span) => span.text).join(''));
  return \`<limu-label x="\${cx}" y="\${cy}" w="\${label.width}" h="\${label.height}" lines="\${encodeURIComponent(JSON.stringify(lines))}"/>\` + s;
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
