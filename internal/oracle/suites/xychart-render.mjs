// XY charts drawn end to end: what Pele's render draws, read back out of its SVG as one line
// per thing drawn, in the order drawn, and the errors of the charts it refuses to draw. limu's
// test writes the same lines from its scene.
//
// Pele draws a chart with no groups that move what is in them but for the title of its left
// axis, which it turns about a point, so its numbers are where the scene has them. Pele's SVG
// is drawn here with its numbers written out in full, as the oracle's hooks have num() write
// them while limuExactNumbers is set, and the lines hold every number in full, as String()
// writes it, so that a difference in the last bit shows. A number Pele writes relative to the
// one before it is added to that one first, as the scene holds where it ends.
import { lone, mutator } from './parse.mjs';
import { FRAGMENTS, corpus, drawableDiagrams, hostile } from './xychart.mjs';

// Pele's escapes in attribute values and text, undone.
const unescape = (s) => s.replace(/&(amp|lt|gt|quot);/g, (_, e) => ({ amp: '&', lt: '<', gt: '>', quot: '"' })[e]);
const attr = (s, name) => {
  const m = new RegExp(`\\s${name}="([^"]*)"`).exec(s);
  return m ? unescape(m[1]) : undefined;
};
const n = (s) => String(Number(s));
const pt = (x, y) => `${n(x)},${n(y)}`;
const TRANSLATE = /translate\(([^,]*),([^)]*)\) rotate\(([^)]*)\)/;

// What a group of Pele's chart is called in the lines, by its class.
const CONTEXT = {
  'pele-grid': 'grid',
  'pele-axis': 'axis',
  'pele-axis-labels': 'axis-labels',
  'pele-axis-titles': 'axis-titles',
  'pele-data-labels': 'data-labels',
  'pele-point-labels': 'point-labels',
  'pele-legend-item': 'legend',
  'pele-title': 'title',
};

// The lines of an XY chart's SVG, drawn with its numbers in full.
export function readSvg(svg, fontSize) {
  const lines = [];
  const head = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  lines.push(`size ${pt(attr(head, 'width'), attr(head, 'height'))}`);
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
  // The groups open about the element read: each one's class, fill and size of type.
  const stack = [];
  const inherited = (name) => {
    for (let k = stack.length - 1; k >= 0; k--) if (stack[k][name] !== undefined) return stack[k][name];
    return undefined;
  };
  const context = (e) => {
    const own = CONTEXT[(attr(e, 'class') ?? '').split(' ')[0]];
    if (own) return own;
    for (let k = stack.length - 1; k >= 0; k--) if (stack[k].context) return stack[k].context;
    return '?';
  };
  // The subpaths of a path's d, each as one string of absolute commands, with H, V, h and v
  // written as L.
  const subpaths = (d) => {
    const out = [];
    let cur = [];
    let x = 0;
    let y = 0;
    for (const m of d.matchAll(/([MLHVhvZ])([^MLHVhvZ]*)/g)) {
      const nums = m[2].trim() === '' ? [] : m[2].trim().split(',');
      switch (m[1]) {
        case 'M':
          if (cur.length) out.push(cur.join(' '));
          [x, y] = nums.map(Number);
          cur = [`M ${pt(x, y)}`];
          break;
        case 'L':
          [x, y] = nums.map(Number);
          cur.push(`L ${pt(x, y)}`);
          break;
        case 'H':
          x = Number(nums[0]);
          cur.push(`L ${pt(x, y)}`);
          break;
        case 'V':
          y = Number(nums[0]);
          cur.push(`L ${pt(x, y)}`);
          break;
        case 'h':
          x = x + Number(nums[0]);
          cur.push(`L ${pt(x, y)}`);
          break;
        case 'v':
          y = y + Number(nums[0]);
          cur.push(`L ${pt(x, y)}`);
          break;
        case 'Z':
          cur.push('Z');
          break;
      }
    }
    if (cur.length) out.push(cur.join(' '));
    return out;
  };
  // A text's first line, where it sits, its fill, anchor and size, and the text of each line.
  const text = (s, open) => {
    const body = s.slice(open.length + 1);
    let at = / x="([^"]*)" y="([^"]*)"/.exec(open);
    let content;
    if (body.includes('<tspan')) {
      const out = [];
      for (const t of body.matchAll(/<tspan([^>]*)>([^<]*)<\/tspan>/g)) {
        if (/ x="/.test(t[1]) || out.length === 0) {
          if (out.length === 0) at = / x="([^"]*)" y="([^"]*)"/.exec(t[1]);
          out.push('');
        }
        out[out.length - 1] += unescape(t[2]);
      }
      content = out;
    } else {
      content = [unescape(body)];
    }
    const fill = attr(open, 'fill') ?? inherited('fill') ?? '-';
    const anchor = attr(open, 'text-anchor') ?? '-';
    const size = attr(open, 'font-size') ?? inherited('size') ?? String(fontSize ?? 16);
    return `${context(open)} t ${pt(at[1], at[2])} ${fill} ${anchor} ${n(size)} | ${content.join(' | ')}`;
  };

  const tags = /<(\/?)([a-z]+)\b([^>]*?)(\/?)>/g;
  for (const m of body.matchAll(tags)) {
    const [whole, close, name] = m;
    if (close) {
      if (name === 'g') stack.pop();
      continue;
    }
    switch (name) {
      case 'g': {
        const cls = attr(whole, 'class') ?? '';
        const id = attr(whole, 'data-id');
        const t = TRANSLATE.exec(attr(whole, 'transform') ?? '');
        const series = { 'pele-series pele-bars': 'bars', 'pele-series pele-line': 'line' }[cls];
        stack.push({ context: CONTEXT[cls] ?? series, fill: attr(whole, 'fill'), size: attr(whole, 'font-size') });
        if (cls === 'pele-series pele-bars') lines.push(`bars ${id ?? '-'} ${attr(whole, 'fill')}`);
        else if (cls === 'pele-series pele-line') lines.push(`line ${id ?? '-'} ${attr(whole, 'fill')}`);
        else if (cls === 'pele-legend-item') lines.push(`legend ${id}`);
        if (t) lines.push(`rotate ${pt(t[1], t[2])} ${n(t[3])}`);
        break;
      }
      case 'path': {
        // The fill of a path is left out: Pele's paths are drawn unfilled, or are straight
        // lines, which a fill leaves as they are.
        const pen = `${attr(whole, 'stroke') ?? '-'} ${attr(whole, 'stroke-width') ?? '-'} ${attr(whole, 'stroke-linecap') ?? '-'} ${attr(whole, 'stroke-linejoin') ?? '-'}`;
        for (const p of subpaths(attr(whole, 'd'))) lines.push(`${context(whole)} path ${p} ${pen}`);
        break;
      }
      case 'circle':
        lines.push(`circle ${pt(attr(whole, 'cx'), attr(whole, 'cy'))} ${n(attr(whole, 'r'))} ${attr(whole, 'fill') ?? inherited('fill') ?? '-'}`);
        break;
      case 'rect': {
        const r = attr(whole, 'rx');
        const box = `${pt(attr(whole, 'x'), attr(whole, 'y'))},${pt(attr(whole, 'width'), attr(whole, 'height'))}`;
        lines.push(`rect ${box} ${r === undefined ? '-' : 'r' + n(r)} ${attr(whole, 'fill') ?? inherited('fill') ?? '-'}`);
        break;
      }
      case 'text': {
        const end = body.indexOf('</text>', m.index);
        lines.push(text(body.slice(m.index, end), whole.slice(0, -1)));
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
      // A chart that parses and is refused, as one with no series is, is kept with its error.
      if (e.code === 'semantic' && e.type === 'xychart') {
        out.push({ src: source, maxWidth, fontSize, lines: [`error ${e.code} ${e.message}`] });
        return true;
      }
      return false;
    } finally {
      globalThis.limuExactNumbers = false;
    }
    if (r.type !== 'xychart') return false;
    out.push({ src: source, maxWidth, fontSize, lines: readSvg(r.svg, fontSize) });
    return true;
  };
  const NARROW = [360, 320, 280, 240, 200, 160, 120, 100, 600, 500, 680];
  const SIZES = [13, 20, 11, 24, 15];
  const seen = new Set();
  let drawn = 0;
  const each = (source) => {
    // A Go string cannot hold a surrogate with no partner.
    if (seen.has(source) || lone(source)) return;
    seen.add(source);
    if (!draw(source)) return;
    // Every third one again in a narrow host, which draws the chart narrower, and every fifth
    // in another size of type.
    if (drawn % 3 === 0) draw(source, NARROW[(drawn / 3) % NARROW.length]);
    if (drawn % 5 === 0) draw(source, drawn % 10 === 0 ? undefined : NARROW[(drawn / 5) % NARROW.length], SIZES[(drawn / 5) % SIZES.length]);
    drawn++;
  };
  corpus(pele).forEach(each);
  hostile(pele).forEach(each);
  const diagrams = drawableDiagrams(random(41));
  for (let i = 0; i < 1500; i++) each(diagrams());
  const usable = [...seen].filter((s) => s.length <= 3000);
  const next = mutator(usable, FRAGMENTS, random(42));
  while (drawn < 2600) {
    const s = next();
    if (s.length <= 3000) each(s);
  }
  write('diagram/xychart/testdata/pele/render.json', { cases: out });
}
