// Ishikawa diagrams drawn end to end: what Pele's render draws, read back out of its SVG as one
// line per thing drawn, in the order drawn. limu's test writes the same lines from its scene.
//
// Pele writes the body of a diagram drawn across about a point of its own, where a scene holds
// where everything is on the page. Pele's SVG is drawn here with its numbers written out in
// full, as the oracle's hooks have num() write them while limuExactNumbers is set; the hooks
// also have each label write, in place of its text, the point it is centered on and all that
// is in it, and each marker its tip and the way it points. A number in the body is moved as
// limu's layout moves it: the number plus the body's move, in that order. The lines hold every
// number in full, as String() writes it, so that a difference in the last bit shows.
import { lone, mutator } from './parse.mjs';
import { capJoin } from './render.mjs';
import { FRAGMENTS, corpus, hostileCorpus, outlines } from './ishikawa.mjs';

// The icons the host has, in the cases drawn with icons.
export const ICONS = ['fa:fa-car', 'fa:fa-user', 'fab:fa-github'];

// Pele's escapes in attribute values, undone.
const unescape = (s) => s.replace(/&(amp|lt|gt|quot);/g, (_, e) => ({ amp: '&', lt: '<', gt: '>', quot: '"' })[e]);
const attr = (s, name) => {
  const m = new RegExp(`\\s${name}="([^"]*)"`).exec(s);
  return m ? unescape(m[1]) : undefined;
};
const n = (v) => String(v);
const TRANSLATE = /translate\(([^,]*),([^)]*)\)/;

// The lines of an Ishikawa diagram's SVG, drawn with its numbers in full, in type of size
// fontSize where nothing sets another.
export function readSvg(svg, fontSize) {
  const lines = [];
  const head = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  lines.push(`size ${n(Number(attr(head, 'width')))},${n(Number(attr(head, 'height')))}`);
  const body = svg.slice(svg.indexOf('>') + 1);
  // The groups that are open: their tags, and how far each moves what is in it.
  const open = [];
  const move = (v, axis) => {
    let r = Number(v);
    for (let k = open.length - 1; k >= 0; k--) if (open[k].move) r = r + open[k].move[axis];
    return r;
  };
  const pt = (x, y) => `${n(move(x, 0))},${n(move(y, 1))}`;
  const pen = (e) => capJoin([...open.map((g) => g.tag), e]);
  const paint = (e) => `${attr(e, 'fill') ?? '-'} ${attr(e, 'stroke') ?? '-'}`;
  // A path's d, moved: M and L as they are, H and V as lines from where the path is, A with
  // its radii, turn and flags before its end, and Z.
  const path = (d) => {
    const out = [];
    let x = 0;
    let y = 0;
    for (const m of d.matchAll(/([MLHVAZ])([^MLHVAZ]*)/g)) {
      const a = m[2].trim() === '' ? [] : m[2].trim().split(/[ ,]+/);
      switch (m[1]) {
        case 'M':
        case 'L':
          x = move(a[0], 0);
          y = move(a[1], 1);
          out.push(`${m[1]} ${n(x)},${n(y)}`);
          break;
        case 'H':
          x = move(a[0], 0);
          out.push(`L ${n(x)},${n(y)}`);
          break;
        case 'V':
          y = move(a[0], 1);
          out.push(`L ${n(x)},${n(y)}`);
          break;
        case 'A':
          x = move(a[5], 0);
          y = move(a[6], 1);
          out.push(`A ${a.slice(0, 5).map((v) => n(Number(v))).join(',')} ${n(x)},${n(y)}`);
          break;
        case 'Z':
          out.push('Z');
          break;
      }
    }
    return out.join(' ');
  };
  for (const m of body.matchAll(/<(\/?)(g|rect|path|limu-label|limu-marker)\b[^>]*>/g)) {
    const e = m[0];
    if (m[1]) {
      if (m[2] === 'g') open.pop();
      continue;
    }
    switch (m[2]) {
      case 'g': {
        const cls = attr(e, 'class') ?? '';
        const id = attr(e, 'data-id');
        if (cls === 'pele-ishikawa-category') lines.push(`category ${id}`);
        else if (cls === 'pele-ishikawa-head') lines.push(`head ${id}`);
        else if (cls.startsWith('pele-node ')) lines.push(`node ${cls.slice('pele-node pele-ishikawa-'.length)} ${id}`);
        const t = TRANSLATE.exec(attr(e, 'transform') ?? '');
        open.push({ tag: e, move: t ? [Number(t[1]), Number(t[2])] : null, size: attr(e, 'font-size') });
        if (e.endsWith('/>')) open.pop();
        break;
      }
      case 'rect': {
        const r = (attr(e, 'style') ?? '').includes('rx:var(--_r)') ? 'r' : attr(e, 'rx') !== undefined ? 'r' + attr(e, 'rx') : '-';
        lines.push(`rect ${pt(attr(e, 'x'), attr(e, 'y'))},${n(Number(attr(e, 'width')))},${n(Number(attr(e, 'height')))} ${r} ${paint(e)} ${pen(e)}`);
        break;
      }
      case 'path': {
        const cls = attr(e, 'class');
        const kind = { 'pele-ishikawa-spine': 'spine', 'pele-ishikawa-bones': 'bones', 'pele-edge': 'link' }[cls] ?? (cls === undefined ? 'outline' : cls);
        lines.push(`${kind} ${path(attr(e, 'd'))} ${paint(e)} ${pen(e)}`);
        break;
      }
      case 'limu-marker':
        lines.push(`marker ${attr(e, 'type')} ${pt(attr(e, 'x'), attr(e, 'y'))} ${n(Number(attr(e, 'dx')))},${n(Number(attr(e, 'dy')))} ${attr(e, 'fill')}`);
        break;
      case 'limu-label': {
        const attrs = decodeURIComponent(attr(e, 'attrs'));
        const fill = /fill="([^"]*)"/.exec(attrs)?.[1] ?? '-';
        const weight = /font-weight="([^"]*)"/.exec(attrs)?.[1] ?? '-';
        let size = /font-size="([^"]*)"/.exec(attrs)?.[1];
        for (let k = open.length - 1; k >= 0 && size === undefined; k--) size = open[k].size;
        const d = JSON.parse(decodeURIComponent(attr(e, 'data')));
        const spans = d.lines.map((line) => line.map(([icon, text, style, width]) => (icon !== null ? `i:${icon}:${n(width)}` : `${style}:${text}:${n(width)}`)).join(' '));
        lines.push(
          `label ${fill} ${weight} ${n(Number(size ?? fontSize))} ${pt(attr(e, 'x'), attr(e, 'y'))} ${n(d.size)},${n(d.lineHeight)},${n(d.width)},${n(d.height)} ` +
            `| ${d.widths.map(n).join(',')} | ${spans.join(' | ')}`
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
    if (r.type !== 'ishikawa') return false;
    out.push({ src: source, maxWidth, fontSize, icons: icons || undefined, lines: readSvg(r.svg, fontSize ?? 16) });
    return true;
  };
  const seen = new Set();
  let drawn = 0;
  const narrow = [360, 480, 200, 100, 600, 300, 140, 40];
  const each = (source) => {
    // A Go string cannot hold a surrogate with no partner.
    if (seen.has(source) || lone(source) || source.length > 4000) return;
    seen.add(source);
    if (!draw(source, undefined, undefined, drawn % 5 === 1)) return;
    // Every other one again in a narrow host, which draws it down where that is narrower, and
    // every seventh in another size of type.
    if (drawn % 2 === 0) draw(source, narrow[(drawn / 2) % narrow.length], drawn % 6 === 0 ? 13 : undefined, drawn % 5 === 2);
    if (drawn % 7 === 0) draw(source, undefined, drawn % 14 === 0 ? 13 : 20);
    drawn++;
  };
  corpus(pele).forEach(each);
  hostileCorpus(pele).forEach(each);
  const built = outlines(random(51));
  for (let i = 0; i < 1500; i++) each(built());
  const usable = [...seen].filter((s) => s.length <= 3000);
  const next = mutator(usable, FRAGMENTS, random(52));
  for (let i = 0; drawn < 3000 && i < 40000; i++) each(next());
  write('diagram/ishikawa/testdata/pele/render.json', { cases: out });
}
