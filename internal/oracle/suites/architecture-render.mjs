// Architecture diagrams drawn end to end: what Pele's render draws, read back out of its SVG as
// one line per thing drawn, in the order drawn. limu's test writes the same lines from its
// scene.
//
// Pele writes the body of a diagram about a point of its own, and the inside of a node about
// the node, where a scene holds where everything is on the page. Pele's SVG is drawn here with
// its numbers written out in full, as the oracle's hooks have num() write them while
// limuExactNumbers is set; the hooks also have each label write, in place of its text, the
// point it is centered on and all that is in it, and each marker its tip and the way it
// points. A number inside groups is moved as limu's layout moves it: by the innermost group's
// move first, then by each one out from it. The lines hold every number in full, as String()
// writes it, so that a difference in the last bit shows.
import { lone, mutator } from './parse.mjs';
import { FRAGMENTS, corpus, randomDiagrams, statements } from './architecture.mjs';

// The icons the host has, in the cases drawn with icons: one of Pele's own among them, which
// the host's then takes the place of.
export const ICONS = ['logos:aws-lambda', 'mdi:home', 'star', 'database', 'fa:fa-car'];

// Pele's escapes in attribute values and text, undone.
const unescape = (s) => s.replace(/&(amp|lt|gt|quot);/g, (_, e) => ({ amp: '&', lt: '<', gt: '>', quot: '"' })[e]);
const attr = (s, name) => {
  const m = new RegExp(`\\s${name}="([^"]*)"`).exec(s);
  return m ? unescape(m[1]) : undefined;
};
const n = (v) => String(v);
const TRANSLATE = /translate\(([^,]*),([^)]*)\)/;

// What a label is called in the lines, by the class Pele gives its text.
const CONTEXT = {
  'pele-title': 'title',
  'pele-cluster-label': 'group',
  'pele-label': 'service',
  'pele-icon-text': 'icon-text',
};

// The lines of an architecture diagram's SVG, drawn with its numbers in full.
export function readSvg(svg) {
  const lines = [];
  const head = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  lines.push(`size ${n(Number(attr(head, 'width')))},${n(Number(attr(head, 'height')))}`);
  let body = svg.slice(svg.indexOf('>') + 1);
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
  const moves = [];
  const open = [];
  // The caps and joins the groups that are open set.
  const pens = [];
  const move = (v, axis) => {
    let r = Number(v);
    for (let k = moves.length - 1; k >= 0; k--) r = r + moves[k][axis];
    return r;
  };
  const pt = (x, y) => `${n(move(x, 0))},${n(move(y, 1))}`;
  const pen = () => {
    let cap = 'butt';
    let join = 'miter';
    for (const p of pens) {
      cap = p.cap ?? cap;
      join = p.join ?? join;
    }
    return `${cap} ${join}`;
  };
  // Inside an icon, whose drawing is its own.
  let icon = null;
  for (const m of body.matchAll(/<(\/?)(g|rect|path|circle|svg|limu-label|limu-marker)\b[^>]*>/g)) {
    const e = m[0];
    if (m[1]) {
      if (m[2] === 'g') {
        if (open.pop()) moves.pop();
        pens.pop();
      }
      if (m[2] === 'svg' && icon) {
        // An icon with nothing in it, one neither the host nor Pele knows, draws nothing.
        if (icon.content.length) lines.push(icon.line, ...icon.content);
        icon = null;
      }
      continue;
    }
    if (icon) {
      if (m[2] === 'path' && attr(e, 'stroke-width') !== undefined) {
        icon.content.push(`glyph ${attr(e, 'd')} ${attr(e, 'stroke')} ${attr(e, 'stroke-width')} ${icon.pen}`);
      } else if (icon.content.length === 0) {
        icon.content.push('host');
      }
      continue;
    }
    switch (m[2]) {
      case 'g': {
        const cls = attr(e, 'class') ?? '';
        const id = attr(e, 'data-id');
        const kind = { 'pele-cluster': 'group', 'pele-edge': 'edge', 'pele-edge-label': 'edge-label', 'pele-node pele-service': 'service', 'pele-node pele-junction': 'junction' }[cls];
        if (kind) lines.push(`${kind} ${id}`);
        const t = TRANSLATE.exec(attr(e, 'transform') ?? '');
        if (t) moves.push([Number(t[1]), Number(t[2])]);
        open.push(t !== null);
        pens.push({ cap: attr(e, 'stroke-linecap'), join: attr(e, 'stroke-linejoin') });
        break;
      }
      case 'rect': {
        const style = attr(e, 'style') ?? '';
        const r = style.includes('rx:var(--_r)') ? 'r' : attr(e, 'rx') !== undefined ? 'r' + attr(e, 'rx') : '-';
        const opacity = attr(e, 'fill-opacity');
        lines.push(
          `rect ${pt(attr(e, 'x') ?? 0, attr(e, 'y') ?? 0)},${n(Number(attr(e, 'width')))},${n(Number(attr(e, 'height')))} ${r} ` +
            `${attr(e, 'fill') ?? '-'}${opacity ? '/' + opacity : ''} ${attr(e, 'stroke') ?? '-'} ${pen()}`
        );
        break;
      }
      case 'path': {
        // An edge's path: M, L and Q, each point moved.
        const out = [];
        for (const c of attr(e, 'd').matchAll(/([MLQ])([^MLQ]*)/g)) {
          const nums = c[2].trim().split(/[ ,]+/);
          let s = c[1];
          for (let i = 0; i < nums.length; i += 2) s += ` ${pt(nums[i], nums[i + 1])}`;
          out.push(s);
        }
        lines.push(`path ${out.join(' ')} ${pen()}`);
        break;
      }
      case 'circle':
        lines.push(`circle ${pt(attr(e, 'cx') ?? 0, attr(e, 'cy') ?? 0)} ${n(Number(attr(e, 'r')))} ${attr(e, 'fill')}`);
        break;
      case 'svg':
        icon = {
          line: `icon ${attr(e, 'data-icon')} ${pt(attr(e, 'x'), attr(e, 'y'))},${n(Number(attr(e, 'width')))},${n(Number(attr(e, 'height')))}`,
          pen: `${attr(e, 'stroke-linecap')} ${attr(e, 'stroke-linejoin')} ${attr(e, 'fill')}`,
          content: [],
        };
        if (e.endsWith('/>')) icon = null;
        break;
      case 'limu-marker':
        lines.push(`marker ${attr(e, 'type')} ${pt(attr(e, 'x'), attr(e, 'y'))} ${n(Number(attr(e, 'dx')))},${n(Number(attr(e, 'dy')))} ${attr(e, 'fill')}`);
        break;
      case 'limu-label': {
        const attrs = decodeURIComponent(attr(e, 'attrs'));
        const cls = /class="([^"]*)"/.exec(attrs)?.[1];
        const fill = /fill="([^"]*)"/.exec(attrs)?.[1] ?? '-';
        const d = JSON.parse(decodeURIComponent(attr(e, 'data')));
        const spans = d.lines.map((line) => line.map(([icon, text, style, width]) => (icon !== null ? `i:${icon}:${n(width)}` : `t:${style}:${text}:${n(width)}`)).join('|'));
        lines.push(
          `label ${CONTEXT[cls] ?? 'edge'} ${fill} ${pt(attr(e, 'x'), attr(e, 'y'))} ${n(d.size)},${n(d.lineHeight)},${n(d.width)},${n(d.height)} ` +
            `| ${d.widths.map(n).join(',')} | ${spans.join(' / ')}`
        );
        break;
      }
    }
  }
  return lines;
}

export default async function ({ src, pele, write, random }) {
  const { render, PeleError } = await src('index.ts');
  const { metricsMeasurer } = await src('text/measurer.ts');

  const out = [];
  const host = (name) => (ICONS.includes(name) ? '<path d="M0,0"/>' : undefined);
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
    if (r.type !== 'architecture') return false;
    out.push({ src: source, maxWidth, fontSize, icons: icons || undefined, lines: readSvg(r.svg) });
    return true;
  };
  const seen = new Set();
  let drawn = 0;
  const narrow = [360, 480, 200, 100, 640, 40];
  const each = (source) => {
    if (seen.has(source) || lone(source) || source.length > 4000) return;
    seen.add(source);
    if (!draw(source, undefined, undefined, drawn % 4 === 1)) return;
    // Some again in a narrow host, which lays the diagram out tightly, and some in another
    // size of type.
    if (drawn % 3 === 0) draw(source, narrow[(drawn / 3) % narrow.length], undefined, drawn % 2 === 0);
    if (drawn % 7 === 0) draw(source, undefined, drawn % 14 === 0 ? 13 : 13.5);
    drawn++;
  };
  corpus(pele).forEach(each);
  const diagrams = randomDiagrams(random(41));
  for (let i = 0; i < 1500; i++) each(diagrams());
  const built = statements(random(42));
  for (let i = 0; i < 1200; i++) each(built());
  const usable = [...seen].filter((s) => s.length <= 3000);
  const next = mutator(usable, FRAGMENTS, random(43));
  for (let i = 0; drawn < 3000 && i < 40000; i++) each(next());
  // Larger diagrams, where the grid has to move blocks aside.
  const large = randomDiagrams(random(44));
  let big = 0;
  for (let i = 0; big < 100 && i < 5000; i++) {
    const s = large();
    if (s.split('\n').length > 25 && !seen.has(s)) {
      each(s);
      big++;
    }
  }
  write('diagram/architecture/testdata/pele/render.json', { cases: out });
}
