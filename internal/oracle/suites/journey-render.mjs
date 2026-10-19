// User journey diagrams drawn end to end: what Pele's render draws, read back out of its SVG as
// one line per thing drawn, in the order drawn. limu's test writes the same lines from its
// scene.
//
// Pele draws a face about its middle, in a group moved there; the scene holds where every
// part of it is on the page. Pele's SVG is drawn here with its numbers written out in full, as
// the oracle's hooks have num() write them while limuExactNumbers is set, and a number in a
// face is moved as limu moves it, the face's middle added to it, before it is written to two
// decimals as Pele writes it.
import { lone, mutator } from './parse.mjs';
import { FRAGMENTS, corpus, randomDiagrams } from './journey.mjs';

// Pele's escapes in attribute values and text, undone.
const unescape = (s) => s.replace(/&(amp|lt|gt|quot);/g, (_, e) => ({ amp: '&', lt: '<', gt: '>', quot: '"' })[e]);
const attr = (s, name) => {
  const m = new RegExp(`\\s${name}="([^"]*)"`).exec(s);
  return m ? unescape(m[1]) : undefined;
};
// A series' color as Pele writes it, by the name of limu's token.
const paintName = (p) => (p === undefined ? '-' : p.replace(/^var\(--pele-(series-\d),[^)]*\)$/, '$1'));
const TRANSLATE = /translate\(([^,]*),([^)]*)\)/;

// The lines of a journey diagram's SVG, drawn with its numbers in full; num writes a number as
// Pele does.
export function readSvg(svg, num) {
  const lines = [];
  const head = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  lines.push(`size ${num(Number(attr(head, 'width')))},${num(Number(attr(head, 'height')))}`);
  // The groups open about the element read: each one's move, fill and text-anchor.
  const stack = [];
  const inherited = (name) => {
    for (let k = stack.length - 1; k >= 0; k--) if (stack[k][name] !== undefined) return stack[k][name];
    return undefined;
  };
  const move = (v, axis) => {
    const moves = stack.filter((g) => g.move).map((g) => g.move);
    if (moves.length === 0) return v;
    let by = moves[0][axis];
    for (let k = 1; k < moves.length; k++) by = by + moves[k][axis];
    return by + v;
  };
  const pt = (x, y) => `${num(move(Number(x), 0))},${num(move(Number(y), 1))}`;
  const paint = (e) => `${paintName(attr(e, 'fill') ?? inherited('fill'))} ${paintName(attr(e, 'stroke'))}`;
  // How a path's stroke ends and turns its corners: its own, or what it inherits, or "-".
  const pen = (e) => `${attr(e, 'stroke-linecap') ?? inherited('cap') ?? '-'} ${attr(e, 'stroke-linejoin') ?? inherited('join') ?? '-'}`;
  const ellipse = (cx, cy, rx, ry) => {
    const x = move(Number(cx), 0);
    const y = move(Number(cy), 1);
    return `ellipse ${num(x)},${num(y)} ${num(x + Number(rx) - x)},${num(y + Number(ry) - y)}`;
  };
  // The subpaths of a path's d, each as one string of absolute commands, with H, V and v
  // written as L, and an arc as A with its radii and flags as written and its end moved.
  const subpaths = (d) => {
    const out = [];
    let cur = [];
    let x = 0;
    let y = 0;
    for (const m of d.matchAll(/([MLHVvAZ])([^MLHVvAZ]*)/g)) {
      const nums = m[2].trim() === '' ? [] : m[2].trim().split(/[ ,]+/);
      switch (m[1]) {
        case 'M':
          if (cur.length) out.push(cur.join(' '));
          [x, y] = nums;
          cur = [`M ${pt(x, y)}`];
          break;
        case 'L':
          [x, y] = nums;
          cur.push(`L ${pt(x, y)}`);
          break;
        case 'H':
          x = nums[0];
          cur.push(`L ${pt(x, y)}`);
          break;
        case 'V':
          y = nums[0];
          cur.push(`L ${pt(x, y)}`);
          break;
        case 'v':
          y = Number(y) + Number(nums[0]);
          cur.push(`L ${pt(x, y)}`);
          break;
        case 'A':
          [x, y] = nums.slice(5);
          cur.push(`A ${nums.slice(0, 5).join(',')} ${pt(x, y)}`);
          break;
        case 'Z':
          cur.push('Z');
          break;
      }
    }
    if (cur.length) out.push(cur.join(' '));
    return out;
  };
  // A text's first line, where it sits, its fill and anchor, and the text of each line.
  const text = (s, open) => {
    const m = / x="([^"]*)" y="([^"]*)"/.exec(s);
    const body = s.slice(open.length + 1);
    let content;
    if (body.includes('<tspan')) {
      const out = [];
      for (const t of body.matchAll(/<tspan([^>]*)>([^<]*)<\/tspan>/g)) {
        if (/ x="/.test(t[1]) || out.length === 0) out.push('');
        out[out.length - 1] += unescape(t[2]);
      }
      content = out;
    } else {
      content = [unescape(body)];
    }
    if (/class="pele-journey-score"/.test(open)) content = [num(Number(content[0]))];
    const fill = paintName(attr(open, 'fill') ?? inherited('fill'));
    const anchor = attr(open, 'text-anchor') ?? inherited('anchor') ?? '-';
    return `t ${pt(m[1], m[2])} ${fill} ${anchor} | ${content.join(' / ')}`;
  };

  const tags = /<(\/?)([a-z]+)\b([^>]*?)(\/?)>/g;
  // Everything after the root's opening tag and its accessible title and description.
  let body = svg.slice(svg.indexOf('>') + 1);
  body = body.replace(/^<title id="[^"]*">[^<]*<\/title>/, '').replace(/^<desc id="[^"]*">[^<]*<\/desc>/, '');
  for (const m of body.matchAll(tags)) {
    const [whole, close, name, , self] = m;
    if (close) {
      if (name === 'g') stack.pop();
      continue;
    }
    switch (name) {
      case 'g': {
        const t = TRANSLATE.exec(attr(whole, 'transform') ?? '');
        stack.push({
          move: t ? [Number(t[1]), Number(t[2])] : undefined,
          fill: attr(whole, 'fill'),
          anchor: attr(whole, 'text-anchor'),
          cap: attr(whole, 'stroke-linecap'),
          join: attr(whole, 'stroke-linejoin'),
        });
        const cls = attr(whole, 'class') ?? '';
        const id = attr(whole, 'data-id');
        if (cls.includes('pele-face')) lines.push(`face ${cls.split(' ')[1].slice('pele-face-'.length)}`);
        else if (cls.includes('pele-section')) lines.push(`section ${id}`);
        else if (cls.includes('pele-task')) lines.push(`task ${id}`);
        else if (cls.includes('pele-journey-step')) lines.push(`step ${id}`);
        else if (cls.includes('pele-legend-item')) lines.push(`legend ${id}`);
        break;
      }
      case 'path': {
        const cls = attr(whole, 'class');
        if (cls) lines.push(`${cls.slice('pele-'.length)} ${attr(whole, 'stroke-dasharray') ?? '-'}`);
        for (const p of subpaths(attr(whole, 'd'))) lines.push(`path ${p} ${paint(whole)} ${pen(whole)}`);
        break;
      }
      case 'circle':
        lines.push(`${ellipse(attr(whole, 'cx') ?? 0, attr(whole, 'cy') ?? 0, attr(whole, 'r'), attr(whole, 'r'))} ${paint(whole)}`);
        if (!self) {
          const tip = /^<title>([^<]*)<\/title>/.exec(body.slice(m.index + whole.length));
          if (tip) lines.push(`tip ${unescape(tip[1])}`);
        }
        break;
      case 'rect': {
        const rounded = attr(whole, 'rx') !== undefined || (attr(whole, 'style') ?? '').includes('rx:');
        const box = `${pt(attr(whole, 'x'), attr(whole, 'y'))},${num(Number(attr(whole, 'width')))},${num(Number(attr(whole, 'height')))}`;
        lines.push(`rect ${box}${rounded ? ' r' : ''} ${paint(whole)}`);
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
      if (!(e instanceof PeleError)) throw e;
      return false;
    } finally {
      globalThis.limuExactNumbers = false;
    }
    if (r.type !== 'journey') return false;
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
    // Every third one again in a narrow host, which turns what runs across and does not
    // fit, every eleventh in a wider one, and every seventh in another size of type.
    if (drawn % 3 === 0) draw(source, 360);
    if (drawn % 11 === 0) draw(source, 520, 18);
    if (drawn % 7 === 0) draw(source, undefined, 13);
    drawn++;
  };
  const base = corpus(pele);
  base.forEach(each);
  const diagrams = randomDiagrams(random(13));
  for (let i = 0; i < 900; i++) each(diagrams());
  const usable = [...seen].filter((s) => s.length <= 3000);
  const next = mutator(usable, FRAGMENTS, random(14));
  while (drawn < 2500) {
    const s = next();
    if (s.length <= 3000) each(s);
  }
  write('diagram/journey/testdata/pele/render.json', { cases: out });
}
