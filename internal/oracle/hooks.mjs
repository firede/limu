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
// label, the point it is centered on, its size and the text of its lines; a sankey
// diagram's labels write the middle of their line as y, with an m before it; an
// architecture diagram's labels write, in place of their text, the point they are centered on
// and all that is in them, and its markers their tips and the way they point; a block
// diagram's labelSvg writes, before each label, the point it is centered on, its size and its
// spans, and its blocks write the size they are drawn at as data-limu; a Wardley
// map's labels write, before each, the point it is drawn about, its measures, its
// attributes and its spans, and its markers where their tips are and the way they point; a
// railroad diagram's names of rules and texts of its boxes write as y, with an m before it,
// the middle of their line before Pele adds the shift to their baseline; a C4
// diagram's labels write, before each one, the point it is drawn about, its size, the size
// of its type and its lines, its markers write their tip and direction, and its shapes write
// their width and height as data-wh; and an Ishikawa
// diagram's labels write, in place of their text, the point they are centered on and all that
// is in them, and its markers their tips and the way they point. A mindmap's
// labels write the same, with how their lines are aligned, and its nodes their size and top;
// an info diagram's labels write the same, and it draws the text limuInfoText gives in place
// of Pele's name and version.
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
const BLOCK_NODE = 'transform="translate(${num(v.x)},${num(v.y)})">`';
const WARDLEY_BUILDER = "import { esc, escText, labelSvg, num, sharedStyle, spanStyle, textStyle } from '../../svg/builder.js';";
const WARDLEY_EDGES = "import { marker, markerTrim } from '../../svg/edges.js';";
const WARDLEY_START = 'function startLabel(';
const RAILROAD_NAME_Y = 'y="${num(top + headHeight / 2 + headSize * 0.35)}"';
const RAILROAD_TEXT_Y = 'y="${num(y + textShift)}"';
const C4_BUILDER = "import { esc, escText, labelSvg, num } from '../../svg/builder.js';";
const C4_EDGES = "import { ARROW, marker, markerTrim } from '../../svg/edges.js';";
const C4_LEFT = "import { leftLabelSvg } from '../common/literal-label.js';";
const C4_NODE = 'transform="translate(${num(node.x + w / 2)},${num(node.y + h / 2)})"';
const ISHIKAWA_EDGES = "import { ARROW, marker } from '../../svg/edges.js';";
const MINDMAP_LEFT = 'function leftLabel(';
const MINDMAP_NODE = 'transform="translate(${num(cx)},${num(top + h[i] / 2)})">';
const INFO_IMPORT = "import { labelSvg } from '../../svg/builder.js';";
const INFO_TEXT = 'layoutLabel(`Pele v${version}`';
// A label as a mindmap's and an info diagram's patches write it: the point it is drawn about,
// how its lines are aligned there, and all that is in it.
const LIMU_LABEL = `
function limuLabel(label: Label, x: number, y: number, attrs: string, align: string): string {
  const lines = label.lines.map((line) => line.map((span) => [span.icon ?? null, span.text, span.style, span.width]));
  const data = { size: label.size, lineHeight: label.lineHeight, width: label.width, height: label.height, widths: label.widths, lines };
  return \`<limu-label x="\${x}" y="\${y}" align="\${align}" attrs="\${encodeURIComponent(attrs)}" data="\${encodeURIComponent(JSON.stringify(data))}"/>\`;
}`;

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
  if (url.endsWith('/src/diagrams/c4/render.ts')) {
    const loaded = await next(url, ctx);
    let source = String(loaded.source);
    source = patch(url, source, C4_BUILDER, C4_BUILDER.replace('labelSvg', 'labelSvg as labelSvgPele'));
    source = patch(url, source, C4_EDGES, C4_EDGES.replace('marker,', 'marker as markerPele,'));
    source = patch(url, source, C4_LEFT, C4_LEFT.replace('leftLabelSvg', 'leftLabelSvg as leftLabelSvgPele'));
    source = patch(url, source, C4_NODE, `${C4_NODE}\${${EXACT.slice(4, -1)} ? \` data-wh="\${w} \${h}"\` : ''}`);
    source += `
const limuLines = (label: any): string =>
  encodeURIComponent(JSON.stringify(label.lines.map((line: any) => line.map((span: any) => span.style + ':' + span.text))));
function labelSvg(label: any, cx: number, cy: number, attrs: string): string {
  const s = labelSvgPele(label, cx, cy, attrs);
  if (!${EXACT.slice(4, -1)} || s === '') return s;
  return \`<limu-label x="\${cx}" y="\${cy}" w="\${label.width}" h="\${label.height}" s="\${label.size}" lines="\${limuLines(label)}"/>\` + s;
}
function leftLabelSvg(label: any, x: number, top: number, attrs: string): string {
  const s = leftLabelSvgPele(label, x, top, attrs);
  if (!${EXACT.slice(4, -1)} || s === '') return s;
  return \`<limu-left x="\${x}" y="\${top + label.height / 2}" w="\${label.width}" h="\${label.height}" s="\${label.size}" lines="\${limuLines(label)}"/>\` + s;
}
function marker(type: string, x: number, y: number, dx: number, dy: number, color: string): string {
  const s = markerPele(type, x, y, dx, dy, color);
  if (!${EXACT.slice(4, -1)}) return s;
  return \`<limu-marker type="\${type}" x="\${x}" y="\${y}" dx="\${dx}" dy="\${dy}"/>\` + s;
}
`;
    return { ...loaded, source, shortCircuit: true };
  }
  if (url.endsWith('/src/diagrams/mindmap/render.ts')) {
    const loaded = await next(url, ctx);
    let source = patch(url, String(loaded.source), RADAR_IMPORT, RADAR_IMPORT.replace('labelSvg', 'labelSvg as labelSvgPele'));
    source = patch(url, source, MINDMAP_LEFT, MINDMAP_LEFT.replace('leftLabel', 'leftLabelPele'));
    source = patch(url, source, MINDMAP_NODE, `${MINDMAP_NODE.slice(0, -1)}\${${EXACT.slice(4, -1)} ? \` data-w="\${w[i]}" data-h="\${h[i]}" data-top="\${top}"\` : ''}>`);
    source += `${LIMU_LABEL}
function labelSvg(label: Label, cx: number, cy: number, attrs: string, icons?: IconResolver): string {
  if (!(globalThis as any).limuExactNumbers) return labelSvgPele(label, cx, cy, attrs, icons);
  return label.lines.length === 0 ? '' : limuLabel(label, cx, cy, attrs, 'middle');
}
function leftLabel(label: Label, left: number, cy: number, icons: RenderOptions['icons']): string {
  if (!(globalThis as any).limuExactNumbers) return leftLabelPele(label, left, cy, icons);
  return label.lines.length === 0 ? '' : limuLabel(label, left, cy, ' class="pele-label"', 'start');
}
`;
    if (!source.includes('type IconResolver')) source = `import type { IconResolver } from '../../types.js';\n` + source;
    return { ...loaded, source, shortCircuit: true };
  }
  if (url.endsWith('/src/diagrams/info/render.ts')) {
    const loaded = await next(url, ctx);
    let source = patch(url, String(loaded.source), INFO_IMPORT, INFO_IMPORT.replace('labelSvg', 'labelSvg as labelSvgPele'));
    source = patch(url, source, INFO_TEXT, `layoutLabel((globalThis as any).limuInfoText ?? \`Pele v\${version}\``);
    source += `${LIMU_LABEL}
function labelSvg(label: Label, cx: number, cy: number, attrs: string): string {
  if (!(globalThis as any).limuExactNumbers) return labelSvgPele(label, cx, cy, attrs);
  return label.lines.length === 0 ? '' : limuLabel(label, cx, cy, attrs, 'middle');
}
`;
    source = `import type { Label } from '../../text/label.js';\n` + source;
    return { ...loaded, source, shortCircuit: true };
  }
  if (url.endsWith('/src/diagrams/sankey/render.ts')) {
    const loaded = await next(url, ctx);
    const source = patch(url, String(loaded.source), SANKEY_Y, `y="\${${EXACT.slice(4, -1)} ? 'm' + y : num(y + small * 0.35)}"`);
    return { ...loaded, source, shortCircuit: true };
  }
  if (url.endsWith('/src/diagrams/block/render.ts')) {
    const loaded = await next(url, ctx);
    let source = patch(url, String(loaded.source), RADAR_IMPORT, RADAR_IMPORT.replace('labelSvg', 'labelSvg as labelSvgPele'));
    source = patch(url, source, BLOCK_NODE, BLOCK_NODE.replace('>`', `\${${EXACT.slice(4, -1)} ? \` data-limu="\${v.bw} \${v.bh}"\` : ''}>\``));
    source += `
function labelSvg(label: Label, cx: number, cy: number, attrs: string, icons?: any): string {
  const s = labelSvgPele(label, cx, cy, attrs, icons);
  if (!(globalThis as any).limuExactNumbers || s === '') return s;
  const lines = label.lines.map((line) => line.map((span) => [span.text, span.style, span.icon ?? null, span.width]));
  return \`<limu-label x="\${cx}" y="\${cy}" w="\${label.width}" h="\${label.height}" lines="\${encodeURIComponent(JSON.stringify(lines))}"/>\` + s;
}
`;
    return { ...loaded, source, shortCircuit: true };
  }
  if (url.endsWith('/src/diagrams/wardley/render.ts')) {
    const loaded = await next(url, ctx);
    let source = String(loaded.source);
    source = patch(url, source, WARDLEY_BUILDER, WARDLEY_BUILDER.replace('labelSvg,', 'labelSvg as labelSvgPele,'));
    source = patch(url, source, WARDLEY_EDGES, WARDLEY_EDGES.replace('marker,', 'marker as markerPele,'));
    source = patch(url, source, WARDLEY_START, 'function startLabelPele(');
    source += `
function limuLabel(tag: string, label: Label, x: number, y: number, attrs: string): string {
  const lines = label.lines.map((line) => line.map((s) => (s.icon !== undefined ? ['i', s.icon] : ['t', s.style, s.text])));
  return \`<\${tag} x="\${x}" y="\${y}" w="\${label.width}" h="\${label.height}" size="\${label.size}" lh="\${label.lineHeight}" attrs="\${encodeURIComponent(attrs)}" lines="\${encodeURIComponent(JSON.stringify(lines))}"/>\`;
}
function labelSvg(label: Label, cx: number, cy: number, attrs: string, icons?: any): string {
  const s = labelSvgPele(label, cx, cy, attrs, icons);
  if (!(globalThis as any).limuExactNumbers || s === '') return s;
  return limuLabel('limu-label', label, cx, cy, attrs) + s;
}
function startLabel(label: Label, x: number, y: number, attrs: string): string {
  const s = startLabelPele(label, x, y, attrs);
  if (!(globalThis as any).limuExactNumbers || s === '') return s;
  return limuLabel('limu-start', label, x, y, attrs) + s;
}
function marker(type: string, x: number, y: number, dx: number, dy: number, color: string): string {
  const s = markerPele(type, x, y, dx, dy, color);
  if (!(globalThis as any).limuExactNumbers) return s;
  return \`<limu-marker type="\${type}" x="\${x}" y="\${y}" dx="\${dx}" dy="\${dy}" color="\${color}"/>\` + s;
}
`;
    return { ...loaded, source, shortCircuit: true };
  }
  if (url.endsWith('/src/diagrams/ishikawa/render.ts')) {
    const loaded = await next(url, ctx);
    let source = patch(url, String(loaded.source), RADAR_IMPORT, RADAR_IMPORT.replace('labelSvg', 'labelSvg as labelSvgPele'));
    source = patch(url, source, ISHIKAWA_EDGES, ISHIKAWA_EDGES.replace('marker', 'marker as markerPele'));
    source += `
function labelSvg(label: Label, cx: number, cy: number, attrs: string): string {
  if (!(globalThis as any).limuExactNumbers) return labelSvgPele(label, cx, cy, attrs);
  if (label.lines.length === 0) return '';
  const lines = label.lines.map((line) => line.map((span) => [span.icon ?? null, span.text, span.style, span.width]));
  const data = { size: label.size, lineHeight: label.lineHeight, width: label.width, height: label.height, widths: label.widths, lines };
  return \`<limu-label x="\${cx}" y="\${cy}" attrs="\${encodeURIComponent(attrs)}" data="\${encodeURIComponent(JSON.stringify(data))}"/>\`;
}
function marker(type: string, x: number, y: number, dx: number, dy: number, color: string): string {
  if (!(globalThis as any).limuExactNumbers) return markerPele(type, x, y, dx, dy, color);
  return \`<limu-marker type="\${type}" x="\${x}" y="\${y}" dx="\${dx}" dy="\${dy}" fill="\${color}"/>\`;
}
`;
    return { ...loaded, source, shortCircuit: true };
  }
  if (url.endsWith('/src/diagrams/railroad/render.ts')) {
    const loaded = await next(url, ctx);
    const exact = EXACT.slice(4, -1);
    let source = String(loaded.source);
    source = patch(url, source, RAILROAD_NAME_Y, `y="\${${exact} ? 'm' + (top + headHeight / 2) : num(top + headHeight / 2 + headSize * 0.35)}"`);
    source = patch(url, source, RAILROAD_TEXT_Y, `y="\${${exact} ? 'm' + y : num(y + textShift)}"`);
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
