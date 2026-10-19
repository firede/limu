// Tree views drawn end to end: what Pele's render draws, read back out of its SVG as one line
// per thing drawn, in the order drawn. limu's test writes the same lines from its scene.
//
// Pele draws a tree view with no groups that move what is in them, so its numbers are where
// the scene has them. Pele's SVG is drawn here with its numbers written out in full, as the
// oracle's hooks have num() write them while limuExactNumbers is set; the hooks also have each
// label write, in place of its text, the point the scene holds it at and all that is in it,
// and each name and description written in one line the middle of its row. The lines hold
// every number in full, as String() writes it, so that a difference in the last bit shows.
//
// A label of one line of plain text is written as its text and its style; any other with the
// size, the widths and the spans of its lines. An icon Pele draws itself is written by the
// name of its glyph, one the host draws by the name the host was given.
import { lone, mutator } from './parse.mjs';
import { FRAGMENTS, corpus, randomDiagrams } from './treeview.mjs';

// The icons the host has, in the cases drawn with icons: one of Pele's own among them, which
// the host's then takes the place of.
export const ICONS = ['logos:react', 'mdi:home', 'material:typescript', 'mermaid-treeview:folder', 'fa:fa-car', 'p:x', 'material:docker'];

// Pele's escapes in attribute values and text, undone.
const unescape = (s) => s.replace(/&(amp|lt|gt|quot);/g, (_, e) => ({ amp: '&', lt: '<', gt: '>', quot: '"' })[e]);
const attr = (s, name) => {
  const m = new RegExp(`\\s${name}="([^"]*)"`).exec(s);
  return m ? unescape(m[1]) : undefined;
};
const n = (v) => String(Number(v));
const pt = (x, y) => `${n(x)},${n(y)}`;
// The control characters Pele's esc drops.
const clean = (s) => s.replace(/[\x00-\x08\x0b\x0c\x0e-\x1f\x7f]+/g, '');

// The subpaths of a path's d, as one string of absolute commands, with H and V written as L.
function steps(d) {
  const out = [];
  let x = 0;
  let y = 0;
  for (const m of d.matchAll(/([MLHVQ])([^MLHVQ]*)/g)) {
    const nums = m[2].trim() === '' ? [] : m[2].trim().split(/[ ,]/).map(Number);
    switch (m[1]) {
      case 'M':
      case 'L':
        [x, y] = nums;
        out.push(`${m[1]} ${pt(x, y)}`);
        break;
      case 'H':
        x = nums[0];
        out.push(`L ${pt(x, y)}`);
        break;
      case 'V':
        y = nums[0];
        out.push(`L ${pt(x, y)}`);
        break;
      case 'Q':
        out.push(`Q ${pt(nums[0], nums[1])} ${pt(nums[2], nums[3])}`);
        [x, y] = [nums[2], nums[3]];
        break;
    }
  }
  return out.join(' ');
}

// The lines of a tree view's SVG, drawn with its numbers in full.
export function readSvg(svg, glyphs) {
  const lines = [];
  const head = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  lines.push(`size ${pt(attr(head, 'width'), attr(head, 'height'))}`);
  let body = svg.slice(svg.indexOf('>') + 1, svg.lastIndexOf('</svg>'));
  const title = /^<title id="[^"]*">([^<]*)<\/title>/.exec(body);
  if (title) {
    lines.push(`acc-title ${unescape(title[1])}`);
    body = body.slice(title[0].length);
  }
  const desc = /^<desc id="[^"]*">([^<]*)<\/desc>/.exec(body);
  if (desc) {
    lines.push(`acc-desc ${unescape(desc[1])}`);
    body = body.slice(desc[0].length);
  }
  // What a label is written with before its place: its anchor, class, fill, weight and style.
  const look = (anchor, attrs) =>
    [anchor, attr(attrs, 'class') ?? '-', attr(attrs, 'fill') ?? '-', attr(attrs, 'font-weight') ? 'b' : '-', attr(attrs, 'font-style') === 'italic' ? 'i' : '-'].join(' ');
  let guides = null;
  let icon = null;
  for (const m of body.matchAll(/<(\/?)([a-z-]+)\b([^>]*?)(\/?)>/g)) {
    const [whole, close, name] = m;
    if (close) {
      if (name === 'g') guides = null;
      if (name === 'svg' && icon) {
        lines.push(icon.line(icon.glyph), ...icon.content);
        icon = null;
      }
      continue;
    }
    if (icon) {
      if (name === 'path' && attr(whole, 'stroke-width') !== undefined) {
        const d = attr(whole, 'd');
        icon.glyph = [...glyphs].find(([, g]) => g === d)?.[0] ?? '?';
        icon.content.push(`glyph ${icon.glyph} ${attr(whole, 'stroke')} ${attr(whole, 'stroke-width')} ${icon.pen}`);
      } else if (icon.content.length === 0) {
        icon.content.push('host');
      }
      continue;
    }
    switch (name) {
      case 'g': {
        const cls = attr(whole, 'class') ?? '';
        if (cls === 'pele-tree-guides') {
          guides = `${attr(whole, 'stroke')} ${n(attr(whole, 'stroke-width'))} ${attr(whole, 'fill')}`;
          break;
        }
        const parts = cls.split(' ');
        lines.push(`node ${parts[1].slice('pele-tree-'.length)} ${attr(whole, 'data-id')} [${parts.slice(2).join(' ')}]`);
        break;
      }
      case 'rect':
        lines.push(
          `rect ${pt(attr(whole, 'x'), attr(whole, 'y'))},${pt(attr(whole, 'width'), attr(whole, 'height'))} ` +
            `${attr(whole, 'rx') !== undefined || (attr(whole, 'style') ?? '').includes('rx:') ? 'r' : '-'} ${attr(whole, 'fill')}/${attr(whole, 'fill-opacity')}`
        );
        break;
      case 'path':
        if (guides) lines.push(`guide ${steps(attr(whole, 'd'))} ${guides} butt miter`);
        else lines.push(`path ${attr(whole, 'd')}`);
        break;
      case 'svg': {
        const box = `${pt(attr(whole, 'x'), attr(whole, 'y'))},${pt(attr(whole, 'width'), attr(whole, 'height'))}`;
        const named = attr(whole, 'data-icon');
        icon = {
          line: (glyph) => `icon ${glyph ?? named} ${box}`,
          pen: `${attr(whole, 'stroke-linecap')} ${attr(whole, 'stroke-linejoin')} ${attr(whole, 'fill')}`,
          content: [],
        };
        if (whole.endsWith('/>')) icon = null;
        break;
      }
      case 'text': {
        const end = body.indexOf('</text>', m.index);
        const y = attr(whole, 'y');
        if (!y.startsWith('m')) throw new Error(`a text not moved: ${whole}`);
        const style = (attr(whole, 'font-weight') ? 1 : 0) | (attr(whole, 'font-style') === 'italic' ? 2 : 0);
        lines.push(`label ${look('start', whole)} ${pt(attr(whole, 'x'), y.slice(1))} | ${style}:${unescape(body.slice(m.index + whole.length, end))}`);
        break;
      }
      case 'limu-label': {
        const attrs = decodeURIComponent(attr(whole, 'attrs'));
        const d = JSON.parse(decodeURIComponent(attr(whole, 'data')));
        const at = `${look(attr(whole, 'anchor'), attrs)} ${pt(attr(whole, 'x'), attr(whole, 'y'))}`;
        if (d.lines.length === 1 && d.lines[0].length === 1 && d.lines[0][0][0] === null) {
          const [, text, style] = d.lines[0][0];
          lines.push(`label ${at} | ${style}:${clean(text)}`);
          break;
        }
        const spans = d.lines.map((line) => line.map(([icon, text, style, width]) => (icon !== null ? `i:${icon}:${n(width)}` : `t:${style}:${text}:${n(width)}`)).join('|'));
        lines.push(`label ${at} ${n(d.size)},${n(d.lineHeight)},${n(d.width)},${n(d.height)} | ${d.widths.map(n).join(',')} | ${spans.join(' / ')}`);
        break;
      }
      case 'tspan':
        break;
      default:
        lines.push(`element ${name}`);
    }
  }
  return lines;
}

export default async function ({ pele, src, write, random }) {
  const { render, PeleError } = await src('index.ts');
  const { metricsMeasurer } = await src('text/measurer.ts');
  const { GLYPHS } = await src('diagrams/treeview/icons.ts');

  const out = [];
  const host = (name) => (ICONS.includes(name) ? '<path d="M0,0"/>' : undefined);
  let tight = 0;
  const draw = (source, maxWidth, fontSize, icons) => {
    let r;
    globalThis.limuExactNumbers = true;
    try {
      r = render(source, { measurer: metricsMeasurer, maxWidth, fontSize, icons: icons ? host : undefined, outputLimit: Infinity });
    } catch (e) {
      if (!(e instanceof PeleError)) throw e;
      return false;
    } finally {
      globalThis.limuExactNumbers = false;
    }
    if (r.type !== 'treeView') return false;
    const lines = readSvg(r.svg, GLYPHS);
    // A name laid out tightly is written by leftLabelSvg, a loose one in a text of its own.
    if (r.svg.includes('anchor="start"')) tight++;
    out.push({ src: source, maxWidth, fontSize, icons: icons || undefined, lines });
    return true;
  };
  const NARROW = [320, 240, 200, 160, 120, 100, 80, 60, 40, 400, 280, 500];
  const SIZES = [13, 20, 11, 24, 15, 13.5];
  const seen = new Set();
  let drawn = 0;
  const each = (source) => {
    // A Go string cannot hold a surrogate with no partner.
    if (seen.has(source) || lone(source) || source.length > 3000) return;
    seen.add(source);
    if (!draw(source, undefined, undefined, drawn % 3 === 1)) return;
    // Every other one again in a narrow host, which lays the tree out tightly when it does
    // not fit, and every fifth in another size of type.
    if (drawn % 2 === 0) draw(source, NARROW[(drawn / 2) % NARROW.length], undefined, drawn % 4 === 0);
    if (drawn % 5 === 0) draw(source, drawn % 10 === 0 ? undefined : NARROW[(drawn / 5) % NARROW.length], SIZES[(drawn / 5) % SIZES.length], true);
    drawn++;
  };
  corpus(pele).forEach(each);
  const diagrams = randomDiagrams(random(81));
  for (let i = 0; i < 1500; i++) each(diagrams());
  const usable = [...seen].filter((s) => s.length <= 3000);
  const next = mutator(usable, FRAGMENTS, random(82));
  while (drawn < 2500) each(next());
  console.log(`treeview-render: ${out.length} drawings, ${tight} laid out tightly`);
  write('diagram/treeview/testdata/pele/render.json', { cases: out });
}
