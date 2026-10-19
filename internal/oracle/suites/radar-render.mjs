// Radar charts drawn end to end: what Pele's render draws, read back out of its SVG as one
// line per thing drawn, in the order drawn. limu's test writes the same lines from its scene.
//
// Pele draws a chart about its center, in a group it then moves to its place. Its SVG is drawn
// here with its numbers written out in full, as the oracle's hooks have num() write them while
// limuExactNumbers is set, and each number in the group is moved as the scene moves it, the
// group's offset added to it, so that the lines hold every number in full, as String() writes
// it, and a difference in the last bit shows. The hooks also have each label written with the
// point it is centered on, its size and the text of its lines, which is what the scene holds of
// a label.
import { readLabel } from '../lib/limu.mjs';
import { lone, mutator } from './parse.mjs';
import { FRAGMENTS, corpus, randomDiagrams } from './radar.mjs';

// Pele's escapes in attribute values and text, undone.
const unescape = (s) => s.replace(/&(amp|lt|gt|quot);/g, (_, e) => ({ amp: '&', lt: '<', gt: '>', quot: '"' })[e]);
const attr = (s, name) => {
  const m = new RegExp(`\\s${name}="([^"]*)"`).exec(s);
  return m ? unescape(m[1]) : undefined;
};
const n = (x) => String(x);
const pt = (x, y) => `${n(x)},${n(y)}`;
const TRANSLATE = /^translate\(([^,]*),([^)]*)\)$/;

// What a group of Pele's chart is called in the lines, by its class.
const CONTEXT = {
  'pele-graticule': 'graticule',
  'pele-axes': 'axes',
  'pele-curves': 'curve',
  'pele-axis-labels': 'axis-label',
  'pele-legend': 'legend',
};

// The lines of a radar chart's SVG, drawn with its numbers in full.
export function readSvg(svg, fontSize) {
  const lines = [];
  const head = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  lines.push(`size ${pt(attr(head, 'width'), attr(head, 'height'))}`);
  const rootFill = attr(head, 'fill');
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
  // The groups open about the element read, each with what it gives the elements in it.
  const stack = [{ fill: rootFill, dx: 0, dy: 0 }];
  const inherited = (e, name) => {
    const own = attr(e, name);
    if (own !== undefined) return own;
    for (let k = stack.length - 1; k >= 0; k--) if (stack[k][name] !== undefined) return stack[k][name];
    return undefined;
  };
  const context = (e) => {
    const own = CONTEXT[attr(e, 'class') ?? ''];
    if (own) return own;
    for (let k = stack.length - 1; k >= 0; k--) if (stack[k].context) return stack[k].context;
    return 'title';
  };
  const id = (e) => {
    const own = attr(e, 'data-id');
    if (own !== undefined) return own;
    for (let k = stack.length - 1; k >= 0; k--) if (stack[k].id !== undefined) return stack[k].id;
    return '-';
  };
  const offset = () => stack[stack.length - 1];
  // A point of the group's, moved to where it is on the page.
  const at = (x, y) => pt(Number(x) + offset().dx, Number(y) + offset().dy);
  const pen = (e) =>
    [
      inherited(e, 'fill') ?? '-',
      inherited(e, 'fill-opacity') ?? '-',
      inherited(e, 'stroke') ?? '-',
      inherited(e, 'stroke-width') ?? '-',
      inherited(e, 'stroke-linecap') ?? 'butt',
      inherited(e, 'stroke-linejoin') ?? 'miter',
    ].join(' ');
  // What an element is in the lines: its context, and the id of a curve, an axis or an entry
  // of the legend.
  const what = (e) => {
    const c = context(e);
    return c === 'graticule' || c === 'axes' ? c : `${c} ${id(e)}`;
  };

  for (const m of body.matchAll(/<(\/?)([a-z][a-z-]*)\b([^>]*?)(\/?)>/g)) {
    const [whole, close, name, , self] = m;
    if (close) {
      if (name === 'g') stack.pop();
      continue;
    }
    switch (name) {
      case 'g': {
        const t = TRANSLATE.exec(attr(whole, 'transform') ?? '');
        const top = offset();
        const group = {
          context: CONTEXT[attr(whole, 'class') ?? ''],
          id: attr(whole, 'data-id'),
          fill: attr(whole, 'fill'),
          'fill-opacity': attr(whole, 'fill-opacity'),
          stroke: attr(whole, 'stroke'),
          'stroke-width': attr(whole, 'stroke-width'),
          'stroke-linecap': attr(whole, 'stroke-linecap'),
          'stroke-linejoin': attr(whole, 'stroke-linejoin'),
          'font-size': attr(whole, 'font-size'),
          dx: t ? top.dx + Number(t[1]) : top.dx,
          dy: t ? top.dy + Number(t[2]) : top.dy,
        };
        if (self) break;
        stack.push(group);
        break;
      }
      case 'circle': {
        const cx = Number(attr(whole, 'cx') ?? 0);
        const cy = Number(attr(whole, 'cy') ?? 0);
        const r = Number(attr(whole, 'r'));
        lines.push(`${what(whole)} circle ${at(cx, cy)} ${at(cx + r, cy + r)} ${pen(whole)}`);
        break;
      }
      case 'polygon': {
        const points = attr(whole, 'points').trim();
        if (points === '') break;
        const steps = points.split(/\s+/).map((p, i) => `${i === 0 ? 'M' : 'L'} ${at(...p.split(','))}`);
        lines.push(`${what(whole)} path ${steps.join(' ')} Z ${pen(whole)}`);
        break;
      }
      case 'path': {
        let cur = [];
        const out = [];
        for (const c of attr(whole, 'd').matchAll(/([MLCZ])([^MLCZ]*)/g)) {
          const nums = c[2].trim() === '' ? [] : c[2].trim().split(/[\s,]+/);
          if (c[1] === 'M' && cur.length) {
            out.push(cur.join(' '));
            cur = [];
          }
          let step = c[1];
          for (let k = 0; k < nums.length; k += 2) step += ' ' + at(nums[k], nums[k + 1]);
          cur.push(step);
        }
        if (cur.length) out.push(cur.join(' '));
        for (const p of out) lines.push(`${what(whole)} path ${p} ${pen(whole)}`);
        break;
      }
      case 'rect': {
        const box = `${at(attr(whole, 'x'), attr(whole, 'y'))},${pt(attr(whole, 'width'), attr(whole, 'height'))}`;
        const rx = attr(whole, 'rx');
        lines.push(`${what(whole)} rect ${box} ${rx === undefined ? '-' : 'r' + rx} ${pen(whole)}`);
        break;
      }
      case 'limu-label': {
        const l = readLabel(whole);
        const text = l.lines.map((line) => line.map((s) => s.text).join(''));
        const size = inherited(whole, 'font-size') ?? String(fontSize ?? 16);
        lines.push(
          `${what(whole)} label ${at(l.x, l.y)} ${pt(l.width, l.height)} ` +
            `${inherited(whole, 'fill') ?? '-'} ${size} | ${text.join(' | ')}`
        );
        break;
      }
    }
  }
  return lines;
}

export default async function ({ pele, src, write, random }) {
  const { render, PeleError } = await src('index.ts');
  const { metricsMeasurer } = await src('text/measurer.ts');

  const out = [];
  const draw = (source, maxWidth, fontSize) => {
    let r;
    globalThis.limuExactNumbers = true;
    try {
      r = render(source, { measurer: metricsMeasurer, maxWidth, fontSize, outputLimit: Infinity });
    } catch (e) {
      if (!(e instanceof PeleError)) throw e;
      return false;
    } finally {
      globalThis.limuExactNumbers = false;
    }
    if (r.type !== 'radar') return false;
    out.push({ src: source, maxWidth, fontSize, lines: readSvg(r.svg, fontSize) });
    return true;
  };
  const NARROW = [360, 320, 280, 240, 200, 160, 120, 100, 600, 500, 680, 420];
  const SIZES = [13, 20, 11, 24, 15];
  const seen = new Set();
  let drawn = 0;
  const each = (source) => {
    // A Go string cannot hold a surrogate with no partner.
    if (seen.has(source) || lone(source) || source.length > 20000) return;
    seen.add(source);
    if (!draw(source)) return;
    // Every third one again in a narrow host, which draws the chart narrower and its legend
    // below it, and every fifth in another size of type.
    if (drawn % 3 === 0) draw(source, NARROW[(drawn / 3) % NARROW.length]);
    if (drawn % 5 === 0) draw(source, drawn % 10 === 0 ? undefined : NARROW[(drawn / 5) % NARROW.length], SIZES[(drawn / 5) % SIZES.length]);
    drawn++;
  };
  corpus(pele).forEach(each);
  const diagrams = randomDiagrams(random(51), true);
  for (let i = 0; i < 1800; i++) each(diagrams());
  const usable = [...seen].filter((s) => s.length <= 600);
  const next = mutator(usable, FRAGMENTS, random(52));
  while (drawn < 2800) {
    const s = next();
    if (s.length <= 600) each(s);
  }
  write('diagram/radar/testdata/pele/render.json', { cases: out });
}
