// Mindmaps drawn end to end: what Pele's render draws, read back out of its SVG as one line per
// thing drawn, in the order drawn. limu's test writes the same lines from its scene.
//
// Pele writes the map about a point of its own, and the inside of a node about the node, where
// a scene holds where everything is on the page. Pele's SVG is drawn here with its numbers
// written out in full, as the oracle's hooks have num() write them while limuExactNumbers is
// set; the hooks also have each label write, in place of its text, the point it is drawn about
// and all that is in it, and each node its size and its top. A number inside groups is moved as
// limu's layout moves it: by the innermost group's move first, then by each one out from it.
// The lines hold every number in full, as String() writes it, so that a difference in the last
// bit shows.
import { lone, mutator } from './parse.mjs';
import { FRAGMENTS, corpus, hostileCorpus, maps } from './mindmap.mjs';

// The icons the host has, in the cases drawn with icons.
export const ICONS = ['fa fa-book', 'star', 'phone', 'fa:fa-car', 'mdi:home', 'fa fa-x'];

// Pele's escapes in attribute values and text, undone.
const unescape = (s) => s.replace(/&(amp|lt|gt|quot);/g, (_, e) => ({ amp: '&', lt: '<', gt: '>', quot: '"' })[e]);
const attr = (s, name) => {
  const m = new RegExp(`\\s${name}="([^"]*)"`).exec(s);
  return m ? unescape(m[1]) : undefined;
};
const n = (v) => String(v);
const TRANSLATE = /translate\(([^,]*),([^)]*)\)/;

// The lines of a mindmap's SVG, drawn with its numbers in full.
export function readSvg(svg) {
  const lines = [];
  const head = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  lines.push(`size ${n(Number(attr(head, 'width')))},${n(Number(attr(head, 'height')))}`);
  const body = svg.slice(svg.indexOf('>') + 1);
  const moves = [];
  const open = [];
  // What the groups that are open set: caps, joins, and an edge's stroke and its width.
  const pens = [];
  const move = (v, axis) => {
    let r = Number(v);
    for (let k = moves.length - 1; k >= 0; k--) r = r + moves[k][axis];
    return r;
  };
  const pt = (x, y) => `${n(move(x, 0))},${n(move(y, 1))}`;
  const pen = (name, fallback) => {
    let v = fallback;
    for (const p of pens) v = p[name] ?? v;
    return v;
  };
  const capJoin = () => `${pen('cap', 'butt')} ${pen('join', 'miter')}`;
  // Inside an icon, whose drawing is the host's.
  let icon = false;
  for (const m of body.matchAll(/<(\/?)(g|rect|path|polygon|circle|ellipse|svg|limu-label)\b[^>]*>/g)) {
    const e = m[0];
    if (m[1]) {
      if (m[2] === 'g') {
        if (open.pop()) moves.pop();
        pens.pop();
      }
      if (m[2] === 'svg') icon = false;
      continue;
    }
    if (icon) continue;
    switch (m[2]) {
      case 'g': {
        const cls = attr(e, 'class') ?? '';
        let b;
        if ((b = /^pele-branch pele-section-(\d+)$/.exec(cls))) lines.push(`branch ${b[1]}`);
        else if (cls.startsWith('pele-node ')) {
          const names = cls.split(' ').slice(1);
          const shape = names.shift().replace(/^pele-shape-/, '');
          const root = names[0] === 'pele-root';
          if (root) names.shift();
          const t = TRANSLATE.exec(attr(e, 'transform'));
          const w = Number(attr(e, 'data-w'));
          const h = Number(attr(e, 'data-h'));
          lines.push(
            `node ${attr(e, 'data-id')} ${shape}${root ? ' root' : ''} [${names.join(' ')}] ` +
              `${n(move(Number(t[1]) - w / 2, 0))},${n(move(attr(e, 'data-top'), 1))},${n(w)},${n(h)}`
          );
        } else if (cls !== 'pele-edges' && cls !== '') lines.push(`group ${cls}`);
        const t = TRANSLATE.exec(attr(e, 'transform') ?? '');
        if (t) moves.push([Number(t[1]), Number(t[2])]);
        open.push(t !== null);
        pens.push({
          cap: attr(e, 'stroke-linecap'),
          join: attr(e, 'stroke-linejoin'),
          stroke: attr(e, 'stroke'),
          width: attr(e, 'stroke-width'),
          fill: attr(e, 'fill'),
        });
        break;
      }
      case 'path':
        if (attr(e, 'class') === 'pele-edge') {
          // An edge's path: M, V, H, Q and C, the steps across and down written as lines to
          // the point they reach.
          const out = [];
          let x = 0;
          let y = 0;
          for (const c of attr(e, 'd').matchAll(/([MVHQC])([^MVHQC]*)/g)) {
            const nums = c[2].trim().split(/[ ,]+/).map(Number);
            if (c[1] === 'V') y = nums[0];
            else if (c[1] === 'H') x = nums[0];
            let s = c[1] === 'V' || c[1] === 'H' ? 'L' : c[1];
            if (c[1] === 'V' || c[1] === 'H') s += ` ${pt(x, y)}`;
            else {
              for (let i = 0; i < nums.length; i += 2) s += ` ${pt(nums[i], nums[i + 1])}`;
              x = nums[nums.length - 2];
              y = nums[nums.length - 1];
            }
            out.push(s);
          }
          lines.push(`edge ${out.join(' ')} ${pen('fill', '-')} ${pen('stroke', '-')} ${pen('width', '-')} ${capJoin()}`);
          break;
        }
      // falls through: a piece of a shape
      case 'rect':
      case 'polygon':
      case 'circle':
      case 'ellipse': {
        const style = attr(e, 'style') ?? '';
        const opacity = attr(e, 'fill-opacity');
        lines.push(
          `part ${style.includes('rx:var(--_r)') ? 'r' : '-'} ${attr(e, 'fill') ?? '-'}${opacity ? '/' + opacity : ''} ${attr(e, 'stroke') ?? '-'} ` +
            `${attr(e, 'stroke-width') ?? '-'} ${capJoin()}`
        );
        break;
      }
      case 'svg':
        icon = !e.endsWith('/>');
        lines.push(`icon ${attr(e, 'data-icon')} ${pt(attr(e, 'x'), attr(e, 'y'))},${n(Number(attr(e, 'width')))},${n(Number(attr(e, 'height')))}`);
        break;
      case 'limu-label': {
        const attrs = decodeURIComponent(attr(e, 'attrs'));
        const cls = /class="([^"]*)"/.exec(attrs)?.[1];
        const fill = /fill="([^"]*)"/.exec(attrs)?.[1] ?? '-';
        const d = JSON.parse(decodeURIComponent(attr(e, 'data')));
        const spans = d.lines.map((line) => line.map(([icon, text, style, width]) => (icon !== null ? `i:${icon}:${n(width)}` : `t:${style}:${text}:${n(width)}`)).join('|'));
        lines.push(
          `label ${cls} ${fill} ${attr(e, 'align')} ${pt(attr(e, 'x'), attr(e, 'y'))} ${n(d.size)},${n(d.lineHeight)},${n(d.width)},${n(d.height)} ` +
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
  const draw = (source, opt) => {
    let r;
    globalThis.limuExactNumbers = true;
    try {
      r = render(source, {
        measurer: metricsMeasurer,
        maxWidth: opt.maxWidth,
        fontSize: opt.fontSize,
        directionBreakpoint: opt.breakpoint,
        autoDirection: opt.keep ? false : undefined,
        icons: opt.icons ? host : undefined,
        outputLimit: Infinity,
      });
    } catch (e) {
      if (!(e instanceof PeleError)) throw e;
      return false;
    } finally {
      globalThis.limuExactNumbers = false;
    }
    if (r.type !== 'mindmap') return false;
    out.push({ src: source, ...opt, icons: opt.icons || undefined, lines: readSvg(r.svg) });
    return true;
  };
  const seen = new Set();
  let drawn = 0;
  const narrow = [360, 480, 200, 100, 600, 40, 320, 250];
  const each = (source) => {
    if (seen.has(source) || lone(source) || source.length > 4000) return;
    seen.add(source);
    if (!draw(source, { icons: drawn % 4 === 1 })) return;
    // The largest maps are left out, to keep the fixture small; the random maps have many.
    if (out[out.length - 1].lines.length > 100) {
      out.pop();
      return;
    }
    // Some again in a narrow host, which lays the map out running down where that is
    // narrower, and some in another size of type or with another breakpoint.
    if (drawn % 2 === 0) draw(source, { maxWidth: narrow[(drawn / 2) % narrow.length], icons: drawn % 3 === 0 });
    if (drawn % 7 === 0) draw(source, { fontSize: drawn % 14 === 0 ? 13 : 13.5 });
    if (drawn % 11 === 0) draw(source, { maxWidth: 700, breakpoint: 900 });
    if (drawn % 13 === 0) draw(source, { maxWidth: 200, keep: true });
    drawn++;
  };
  corpus(pele).forEach(each);
  hostileCorpus(pele).forEach(each);
  const map = maps(random(61));
  for (let i = 0; i < 1000; i++) each(map());
  const usable = [...seen].filter((s) => s.length <= 3000);
  const next = mutator(usable, FRAGMENTS, random(62));
  for (let i = 0; drawn < 2500 && i < 40000; i++) each(next());
  write('diagram/mindmap/testdata/pele/render.json', { cases: out });
}
