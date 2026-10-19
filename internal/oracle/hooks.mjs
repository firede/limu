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
// where a group moves it. Where the scene holds a label otherwise than Pele writes it, a
// patch writes what the suite needs to compare it: a kanban board's textBlock writes, in place of each line of text, where the
// line starts or ends and what is in it; a pie's percentages write, as data-y, where they are
// before Pele adds the shift to their baseline; a radar chart's labelSvg writes, before each
// label, the point it is centered on, its size and the text of its lines; and a sankey
// diagram's labels write the middle of their line as y, with an m before it; and an
// architecture diagram's labels write, in place of their text, the point they are centered on
// and all that is in them, and its markers their tips and the way they point; and a tree
// view's labels write, in place of their text, the point the scene holds them at and all that
// is in them, and its names and descriptions in one line the middle of their row as y, with
// an m before it.
const patch = (url, source, find, replace) => {
  if (!source.includes(find)) throw new Error(`no ${find} in ${url}`);
  return source.replace(find, replace);
};
const NUM = 'export function num(value: number): string {';
const EXACT = 'if ((globalThis as any).limuExactNumbers)';
const CLASS_MARKER = 'export function classMarker(';
const TEXT_BLOCK = 'function textBlock(';
const PIE_OUTSIDE = 'outside += `<text x="${num(x)}"';
const PIE_INSIDE = 'labels += `<text x="${num(Math.sin(mid) * r)}"';
const dataY = (y) => `<text\${(globalThis as any).limuExactNumbers ? \` data-y="\${${y}}"\` : ''}`;
const RADAR_IMPORT = "import { escText, labelSvg, num } from '../../svg/builder.js';";
const SANKEY_Y = 'y="${num(y + small * 0.35)}"';
const ARCH_EDGES = "import { edgeLabelSvg, marker, markerTrim } from '../../svg/edges.js';";
const TREE_BUILDER = "import { esc, escText, labelSvg, num } from '../../svg/builder.js';";
const TREE_LITERAL = "import { literalLabel, leftLabelSvg } from '../common/literal-label.js';";
const TREE_BASELINE = /num\(\s*cy \+ baseline\s*\)/g;

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
  if (url.endsWith('/src/diagrams/pie/render.ts')) {
    const loaded = await next(url, ctx);
    let source = String(loaded.source);
    source = patch(url, source, PIE_OUTSIDE, PIE_OUTSIDE.replace('<text', dataY('y')));
    source = patch(url, source, PIE_INSIDE, PIE_INSIDE.replace('<text', dataY('-Math.cos(mid) * r')));
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
  if (url.endsWith('/src/diagrams/architecture/render.ts')) {
    const loaded = await next(url, ctx);
    let source = patch(url, String(loaded.source), RADAR_IMPORT, RADAR_IMPORT.replace('labelSvg', 'labelSvg as labelSvgPele'));
    source = patch(url, source, ARCH_EDGES, ARCH_EDGES.replace('edgeLabelSvg, marker', 'edgeLabelSvg as edgeLabelSvgPele, marker as markerPele'));
    source += `
function limuLabel(label: Label, cx: number, cy: number, attrs: string): string {
  const lines = label.lines.map((line) => line.map((span) => [span.icon ?? null, span.text, span.style, span.width]));
  const data = { size: label.size, lineHeight: label.lineHeight, width: label.width, height: label.height, widths: label.widths, lines };
  return \`<limu-label x="\${cx}" y="\${cy}" attrs="\${encodeURIComponent(attrs)}" data="\${encodeURIComponent(JSON.stringify(data))}"/>\`;
}
function labelSvg(label: Label, cx: number, cy: number, attrs: string, icons?: IconResolver): string {
  if (!(globalThis as any).limuExactNumbers) return labelSvgPele(label, cx, cy, attrs, icons);
  return label.lines.length === 0 ? '' : limuLabel(label, cx, cy, attrs);
}
function edgeLabelSvg(id: string, label: Label, x: number, y: number, attrs: string, icons?: IconResolver): string {
  if (!(globalThis as any).limuExactNumbers) return edgeLabelSvgPele(id, label, x, y, attrs, icons);
  const s = edgeLabelSvgPele(id, { ...label, lines: [] }, x, y, attrs, icons);
  return s.slice(0, -4) + limuLabel(label, x, y, attrs) + '</g>';
}
function marker(type: string, x: number, y: number, dx: number, dy: number, color: string): string {
  if (!(globalThis as any).limuExactNumbers) return markerPele(type, x, y, dx, dy, color);
  return \`<limu-marker type="\${type}" x="\${x}" y="\${y}" dx="\${dx}" dy="\${dy}" fill="\${color}"/>\`;
}
`;
    return { ...loaded, source, shortCircuit: true };
  }
  if (url.endsWith('/src/diagrams/treeview/render.ts')) {
    const loaded = await next(url, ctx);
    let source = patch(url, String(loaded.source), TREE_BUILDER, TREE_BUILDER.replace('labelSvg', 'labelSvg as labelSvgPele'));
    source = patch(url, source, TREE_LITERAL, TREE_LITERAL.replace('leftLabelSvg', 'leftLabelSvg as leftLabelSvgPele'));
    if (source.match(TREE_BASELINE)?.length !== 2) throw new Error(`no cy + baseline in ${url}`);
    source = source.replace(TREE_BASELINE, `(${EXACT.slice(4, -1)} ? 'm' + cy : num(cy + baseline))`);
    source += `
function limuLabel(label: Label, x: number, y: number, attrs: string, anchor: string): string {
  const lines = label.lines.map((line) => line.map((span) => [span.icon ?? null, span.text, span.style, span.width]));
  const data = { size: label.size, lineHeight: label.lineHeight, width: label.width, height: label.height, widths: label.widths, lines };
  return \`<limu-label x="\${x}" y="\${y}" anchor="\${anchor}" attrs="\${encodeURIComponent(attrs)}" data="\${encodeURIComponent(JSON.stringify(data))}"/>\`;
}
function labelSvg(label: Label, cx: number, cy: number, attrs: string): string {
  const s = labelSvgPele(label, cx, cy, attrs);
  if (!${EXACT.slice(4, -1)} || s === '') return s;
  return limuLabel(label, cx, cy, attrs, 'middle');
}
function leftLabelSvg(label: Label, x: number, top: number, attrs: string): string {
  const s = leftLabelSvgPele(label, x, top, attrs);
  if (!${EXACT.slice(4, -1)} || s === '') return s;
  return limuLabel(label, x, top + label.height / 2, attrs, 'start');
}
`;
    return { ...loaded, source, shortCircuit: true };
  }
  if (url.endsWith('/src/diagrams/sankey/render.ts')) {
    const loaded = await next(url, ctx);
    const source = patch(url, String(loaded.source), SANKEY_Y, `y="\${${EXACT.slice(4, -1)} ? 'm' + y : num(y + small * 0.35)}"`);
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
