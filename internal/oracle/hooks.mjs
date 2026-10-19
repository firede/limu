// Lets Node run Pele's TypeScript sources as they are: an import of `x.js` that is not there
// finds `x.ts`, and a JSON file is a module with its fields as named exports.
//
// A suite that reads numbers back out of an SVG to work with them sets
// globalThis.limuExactNumbers while Pele draws, and the patches below then have Pele write what
// the suite needs to compare with the scene; while it is not set, Pele draws as it does. The
// labels and markers are written as lib/limu.mjs says, before what Pele writes of them.
//
//   diagram       patched                                  writes
//   all           num                                      numbers in full, as String writes them
//   class         classMarker                              its markers
//   kanban        textBlock                                its labels
//   pie           the percentages                          data-y, where they are before the shift to their baseline
//   radar         labelSvg                                 its labels
//   architecture  labelSvg, edgeLabelSvg, marker           its labels and markers
//   c4            labelSvg, leftLabelSvg, marker, nodes    its labels and markers; data-wh, the size of a shape
//   tree view     labelSvg, leftLabelSvg, texts            its labels; as y, m and the middle of a row
//   mindmap       labelSvg, leftLabel, nodes               its labels; data-w, data-h and data-top, a node's size and top
//   info          labelSvg, the name                       its labels; the text limuInfoText gives in place of Pele's name
//   sankey        the labels' baseline                     as y, m and the middle of the line
//   block         labelSvg, blocks                         its labels; data-limu, the size a block is drawn at
//   wardley       labelSvg, startLabel, marker             its labels and markers
//   ishikawa      labelSvg, marker                         its labels and markers
//   cynefin       labelSvg, marker                         its labels and markers
//   railroad      the rules' names, the boxes' texts       as y, m and the middle of the line
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

const EXACT = '(globalThis as any).limuExactNumbers';
const LIMU = new URL('./lib/limu.mjs', import.meta.url).href;

// Replaces find in source with what replace gives, where find occurs exactly count times: a
// string, which is put in as it is, or a function of what is found.
function patch(url, source, find, replace, count = 1) {
  const re =
    typeof find === 'string'
      ? new RegExp(find.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'g')
      : new RegExp(find.source, find.flags.includes('g') ? find.flags : find.flags + 'g');
  const found = source.match(re)?.length ?? 0;
  if (found !== count) throw new Error(`${found} of ${find} in ${url}, not ${count}`);
  return source.replace(re, typeof replace === 'string' ? () => replace : replace);
}

// The kinds of patch. Each is a function of the URL and the source of a file of Pele's, which
// gives the source patched.

// Renames each of the names the file imports from the module, name to namePele, and adds the
// wrapper of each, which takes the name and calls namePele.
const wrapImport = (from, wrappers) => (url, source) => {
  const line = new RegExp(`^import \\{([^}]*)\\} from '${from.replace(/\./g, '\\.')}';$`, 'm');
  source = patch(url, source, line, (whole) => {
    for (const name of Object.keys(wrappers)) whole = patch(url, whole, new RegExp(`\\b${name}\\b`, 'g'), `${name} as ${name}Pele`);
    return whole;
  });
  return source + Object.entries(wrappers).map(([name, wrap]) => wrap(name)).join('');
};

// Renames the file's own function name to namePele, and adds wrap's function of that name.
const wrapLocal = (name, wrap) => (url, source) => patch(url, source, `function ${name}(`, `function ${name}Pele(`) + wrap(name);

// Adds an attribute to a template, between before and after, which occur together once.
const attribute = (before, after, text) => (url, source) =>
  patch(url, source, before + after, `${before}\${${EXACT} ? \` ${text}\` : ''}${after}`);

// Writes, in place of a baseline that is num(of) and found count times, m and mid, the middle of
// the line, which the suite moves before it adds the shift to the baseline as Pele adds it.
const midline = (find, mid, count = 1) => (url, source) =>
  patch(url, source, find, (num) => `(${EXACT} ? 'm' + (${mid}) : ${num})`, count);

// A patch of its own: find replaced with replace.
const replace = (find, replace) => (url, source) => patch(url, source, find, replace);

// The wrappers. A label's takes the parameters given, the first of them the label, calls the
// wrapped function with them, and writes the label before what it gives: drawn about (x, y),
// aligned as align says, with the attributes attrs. A marker's takes a marker's parameters.
const label = (params, x, y, align, attrs = 'attrs') => (name) => `
function ${name}(${params}): string {
  const s = ${name}Pele(${params.split(',').map((p) => p.split(/[?:]/)[0].trim()).join(', ')});
  return ${EXACT} ? limuLabel(label, ${x}, ${y}, ${align}, ${attrs}, s) : s;
}
`;
const marker = (name) => `
function ${name}(type: string, x: number, y: number, dx: number, dy: number, color: string): string {
  const s = ${name}Pele(type, x, y, dx, dy, color);
  return ${EXACT} ? limuMarker(type, x, y, dx, dy, color, s) : s;
}
`;
// An edge's label, drawn on a patch of the background: the label is written in the group,
// before what Pele writes of its text.
const edgeLabel = (name) => `
function ${name}(id: string, label: any, x: number, y: number, attrs: string, icons?: any): string {
  const s = ${name}Pele(id, label, x, y, attrs, icons);
  if (!${EXACT}) return s;
  const bare = ${name}Pele(id, { ...label, lines: [] }, x, y, attrs, icons).slice(0, -'</g>'.length);
  return bare + limuLabel(label, x, y, 'middle', attrs, s.slice(bare.length, -'</g>'.length)) + '</g>';
}
`;
const centered = label('label: any, cx: number, cy: number, attrs: string, icons?: any', 'cx', 'cy', "'middle'");
// Pele's leftLabelSvg, whose lines start at x and whose top is at top.
const leftward = label('label: any, x: number, top: number, attrs: string', 'x', 'top + label.height / 2', "'start'");

const BUILDER = '../../svg/builder.js';
const EDGES = '../../svg/edges.js';
const LITERAL = '../common/literal-label.js';

// Pele's files, by their paths under src, and the patches each takes, in turn.
const PATCHES = {
  'svg/builder.ts': [replace('export function num(value: number): string {', `export function num(value: number): string {\n  if (${EXACT}) return String(value);`)],
  'diagrams/class/markers.ts': [wrapLocal('classMarker', (name) => `export ${marker(name).trimStart()}`)],
  'diagrams/kanban/render.ts': [
    wrapLocal('textBlock', label('label: any, x: number, y: number, attrs: string, icons: any, end?: boolean', 'x', 'y', "end ? 'end' : 'start'")),
  ],
  'diagrams/pie/render.ts': [
    attribute('outside += `<text', ' x="${num(x)}"', 'data-y="${y}"'),
    attribute('labels += `<text', ' x="${num(Math.sin(mid) * r)}"', 'data-y="${-Math.cos(mid) * r}"'),
  ],
  'diagrams/radar/render.ts': [wrapImport(BUILDER, { labelSvg: centered })],
  'diagrams/architecture/render.ts': [wrapImport(BUILDER, { labelSvg: centered }), wrapImport(EDGES, { edgeLabelSvg: edgeLabel, marker })],
  'diagrams/c4/render.ts': [
    wrapImport(BUILDER, { labelSvg: centered }),
    wrapImport(EDGES, { marker }),
    wrapImport(LITERAL, { leftLabelSvg: leftward }),
    attribute('transform="translate(${num(node.x + w / 2)},${num(node.y + h / 2)})"', '', 'data-wh="${w} ${h}"'),
  ],
  'diagrams/treeview/render.ts': [
    wrapImport(BUILDER, { labelSvg: centered }),
    wrapImport(LITERAL, { leftLabelSvg: leftward }),
    midline(/num\(\s*cy \+ baseline\s*\)/g, 'cy', 2),
  ],
  'diagrams/mindmap/render.ts': [
    wrapImport(BUILDER, { labelSvg: centered }),
    wrapLocal('leftLabel', label('label: any, left: number, cy: number, icons: any', 'left', 'cy', "'start'", `' class="pele-label"'`)),
    attribute('transform="translate(${num(cx)},${num(top + h[i] / 2)})"', '>', 'data-w="${w[i]}" data-h="${h[i]}" data-top="${top}"'),
  ],
  'diagrams/info/render.ts': [
    wrapImport(BUILDER, { labelSvg: centered }),
    replace('layoutLabel(`Pele v${version}`', `layoutLabel((globalThis as any).limuInfoText ?? \`Pele v\${version}\``),
  ],
  'diagrams/sankey/render.ts': [midline('num(y + small * 0.35)', 'y')],
  'diagrams/block/render.ts': [
    wrapImport(BUILDER, { labelSvg: centered }),
    attribute('transform="translate(${num(v.x)},${num(v.y)})"', '>`', 'data-limu="${v.bw} ${v.bh}"'),
  ],
  'diagrams/wardley/render.ts': [
    wrapImport(BUILDER, { labelSvg: centered }),
    wrapImport(EDGES, { marker }),
    wrapLocal('startLabel', label('label: any, x: number, y: number, attrs: string', 'x', 'y', "'start'")),
  ],
  'diagrams/ishikawa/render.ts': [wrapImport(BUILDER, { labelSvg: centered }), wrapImport(EDGES, { marker })],
  'diagrams/cynefin/render.ts': [wrapImport(BUILDER, { labelSvg: centered }), wrapImport(EDGES, { marker })],
  'diagrams/railroad/render.ts': [
    midline('num(top + headHeight / 2 + headSize * 0.35)', 'top + headHeight / 2'),
    midline('num(y + textShift)', 'y'),
  ],
};

export async function load(url, ctx, next) {
  const file = Object.keys(PATCHES).find((path) => url.endsWith('/src/' + path));
  if (file) {
    const loaded = await next(url, ctx);
    let source = String(loaded.source);
    for (const p of PATCHES[file]) source = p(url, source);
    if (source.includes('limuLabel(') || source.includes('limuMarker(')) {
      source = `import { writeLabel as limuLabel, writeMarker as limuMarker } from '${LIMU}';\n` + source;
    }
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
