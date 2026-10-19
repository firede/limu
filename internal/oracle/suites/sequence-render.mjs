// Sequence diagrams drawn end to end: what Pele's render draws, read back out of its SVG as
// one line per thing drawn, in the order drawn. limu's test writes the same lines from its
// scene.
//
// Pele draws a sequence diagram in one group, moved by the padding past the diagram's
// extents; the scene holds where everything is on the page. Pele's SVG is drawn here with
// its numbers written out in full, as the oracle's hooks have num() write them while
// limuExactNumbers is set; each number of a place is moved by the group's move, as limu
// moves it, and only then written to two decimals as Pele writes it.
import { join } from 'node:path';
import { lone } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { unescape, attr, stroke } from '../lib/svg.mjs';
import { FRAGMENTS, corpus, drawableDiagrams } from './sequence.mjs';

// A stroke's ends or corners as a browser takes them on an element, as lib/svg.mjs's stroke
// settles them, from what it inherits.
const linecap = (e, inherited) => stroke('stroke-linecap', [e], inherited);
const linejoin = (e, inherited) => stroke('stroke-linejoin', [e], inherited);

// The groups that hold one thing drawn, by the first of their classes; the others are the
// layers things are drawn in, which give what is in them its paint.
const OBJECTS = new Set(['pele-cluster', 'pele-frame', 'pele-edge', 'pele-sequence-number', 'pele-edge-label', 'pele-note', 'pele-node']);

// The lines of a sequence diagram's SVG, drawn with its numbers in full; num writes a number
// as Pele does.
export function readSvg(svg, num) {
  const lines = [];
  const head = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  lines.push(`size ${num(Number(attr(head, 'width')))},${num(Number(attr(head, 'height')))}`);
  const t = /<g transform="translate\(([^,]*),([^)]*)\)">/.exec(svg);
  const mx = Number(t[1]);
  const my = Number(t[2]);
  const pt = (x, y) => `${num(mx + Number(x))},${num(my + Number(y))}`;
  const size = (v) => num(Number(v));

  // The subpaths of a path's d, each as one string of absolute commands, M, L, A and Z, with
  // H and V written as L, and an arc, or arcs one after another, by where they end.
  const subpaths = (d) => {
    const out = [];
    let cur = [];
    let x = '0';
    let y = '0';
    for (const m of d.matchAll(/([MLHVAZ])([^MLHVAZ]*)/g)) {
      const nums = m[2].trim() === '' ? [] : m[2].trim().split(/[ ,]+/);
      switch (m[1]) {
        case 'M':
          if (cur.length) out.push(cur);
          [x, y] = nums;
          cur = [['M', x, y]];
          break;
        case 'L':
          [x, y] = nums;
          cur.push(['L', x, y]);
          break;
        case 'H':
          x = nums[0];
          cur.push(['L', x, y]);
          break;
        case 'V':
          y = nums[0];
          cur.push(['L', x, y]);
          break;
        case 'A':
          [x, y] = nums.slice(5);
          if (cur[cur.length - 1][0] === 'A') cur.pop();
          cur.push(['A', x, y]);
          break;
        case 'Z':
          cur.push(['Z']);
          break;
      }
    }
    if (cur.length) out.push(cur);
    return out.map((s) => s.map(([op, ...p]) => (p.length ? `${op} ${pt(p[0], p[1])}` : op)).join(' '));
  };

  const stack = [{ fill: 'var(--_fg)', stroke: 'none', width: '-', cap: undefined, join: undefined }];
  // How a stroked thing ends and turns its corners, after what it is in: " cap=" and " join="
  // with the value a browser draws, left out where it is SVG's initial butt or miter.
  const ends = (e, stroke) => {
    if (stroke === 'none') return '';
    const up = stack[stack.length - 1];
    const cap = linecap(e, up.cap);
    const join = linejoin(e, up.join);
    return `${cap && cap !== 'butt' ? ' cap=' + cap : ''}${join && join !== 'miter' ? ' join=' + join : ''}`;
  };
  const paint = (e, straight = false) => {
    const up = stack[stack.length - 1];
    const opacity = attr(e, 'fill-opacity');
    const fill = straight ? 'none' : `${attr(e, 'fill') ?? up.fill}${opacity ? '/' + opacity : ''}`;
    const width = attr(e, 'stroke-width') ?? up.width;
    const style = attr(e, 'style') ?? '';
    // A fill the author gave, as written in the style.
    const author = /(?:^|;)fill:([^;]*)/.exec(style);
    const stroke = attr(e, 'stroke') ?? up.stroke;
    return `${fill} ${stroke}${width === '-' ? '' : ' w=' + width}${ends(e, stroke)}${author ? ' author=' + JSON.stringify(author[1]) : ''}`;
  };
  const id = (e) => {
    const v = attr(e, 'data-id');
    return v === undefined ? '' : ' #' + v;
  };

  const body = svg.slice(t.index + t[0].length);
  for (const m of body.matchAll(/<(\/?)(g|rect|path|circle|text)\b([^>]*)>/g)) {
    const e = m[0];
    if (m[1]) {
      if (m[2] === 'g') stack.pop();
      continue;
    }
    switch (m[2]) {
      case 'g': {
        const up = stack[stack.length - 1];
        stack.push({
          fill: attr(e, 'fill') ?? up.fill,
          stroke: attr(e, 'stroke') ?? up.stroke,
          width: attr(e, 'stroke-width') ?? up.width,
          cap: linecap(e, up.cap),
          join: linejoin(e, up.join),
        });
        const cls = attr(e, 'class') ?? '';
        if (OBJECTS.has(cls.split(' ')[0])) lines.push(`g ${cls}${id(e)}`);
        break;
      }
      case 'rect': {
        const style = attr(e, 'style') ?? '';
        const radius = style.includes('rx:var(--_r)') ? 'r' : attr(e, 'rx') !== undefined ? 'rx=' + attr(e, 'rx') : '-';
        lines.push(`rect ${pt(attr(e, 'x'), attr(e, 'y'))},${size(attr(e, 'width'))},${size(attr(e, 'height'))} ${radius} ${paint(e)}${id(e)}`);
        break;
      }
      case 'path': {
        const subs = subpaths(attr(e, 'd'));
        // A straight line has nothing inside to fill.
        const straight = subs.every((s) => /^M \S+ L \S+$/.test(s));
        const dash = attr(e, 'stroke-dasharray');
        for (const s of subs) lines.push(`path ${s} ${paint(e, straight)}${dash ? ' dash=' + dash : ''}${id(e)}`);
        break;
      }
      case 'circle': {
        const x = mx + Number(attr(e, 'cx'));
        const y = my + Number(attr(e, 'cy'));
        lines.push(`ellipse ${num(x)},${num(y)} ${num(x + Number(attr(e, 'r')) - x)} ${paint(e)}`);
        break;
      }
      case 'text': {
        const end = body.indexOf('</text>', m.index);
        const s = body.slice(m.index, end);
        const at = / x="([^"]*)" y="([^"]*)"/.exec(s);
        const fill = attr(e, 'fill') ?? stack[stack.length - 1].fill;
        const content = unescape(
          s
            .slice(e.length)
            .replace(/<tspan[^>]* x="[^"]*"[^>]*>/g, '\u0001')
            .replace(/<[^>]*>/g, '')
        )
          .split('\u0001')
          .filter((l, i) => i > 0 || l !== '')
          .join('|');
        lines.push(`text ${pt(at[1], at[2])} ${fill} ${JSON.stringify(content)}`);
        break;
      }
    }
  }
  return lines;
}

export default async function ({ src, pele, write, random }) {
  const { render, PeleError } = await src('index.ts');
  const { metricsMeasurer } = await src('text/measurer.ts');
  const { num } = await src('svg/builder.ts');

  const out = [];
  const draw = (source, maxWidth, fontSize) => {
    let r;
    globalThis.limuExactNumbers = true;
    try {
      r = render(source, { measurer: metricsMeasurer, maxWidth, fontSize, outputLimit: Infinity });
    } catch (e) {
      if (e instanceof RangeError) return false;
      if (!(e instanceof PeleError)) throw e;
      return false;
    } finally {
      globalThis.limuExactNumbers = false;
    }
    if (r.type !== 'sequence') return false;
    out.push({ src: source, maxWidth, fontSize, lines: readSvg(r.svg, num) });
    return true;
  };
  const seen = new Set();
  let drawn = 0;
  const each = (source) => {
    // A Go string cannot hold a surrogate with no partner.
    if (seen.has(source) || lone(source)) return;
    seen.add(source);
    if (!draw(source)) return;
    // Every third one again in a narrow host, which draws it tightly where it does not fit,
    // and every seventh in another size of type.
    if (drawn % 3 === 0) draw(source, 360);
    if (drawn % 7 === 0) draw(source, undefined, 13);
    drawn++;
  };
  const base = corpus(pele);
  base.forEach(each);
  const drawable = drawableDiagrams(random(13));
  for (let i = 0; i < 1200; i++) each(drawable());
  const usable = [...seen].filter((s) => s.length <= 3000);
  const next = mutator(usable, FRAGMENTS, random(14));
  while (out.length < 3200) {
    const s = next();
    if (s.length <= 3000) each(s);
  }
  write('diagram/sequence/testdata/pele/render.json', { cases: out });
}
