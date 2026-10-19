// Git graphs drawn end to end: what Pele's render draws, read back out of its SVG as one line
// per thing drawn, in the order drawn. limu's test writes the same lines from its scene.
//
// Pele draws the graph inside a group it moves into place, where the scene holds where
// everything is on the page. Pele's SVG is drawn here with its numbers written out in full, as
// the oracle's hooks have num() write them while limuExactNumbers is set; each number inside
// the group is moved as limu moves it, and only then written to two decimals as Pele writes it.
//
// A commit's label that Pele turns about a point with rotate() is held in the scene by its
// middle, turned, and the box behind it by its outline, a path turned point by point. Both are
// worked out here from Pele's numbers as limu works them out.
import { lone, mutator } from './parse.mjs';
import { FRAGMENTS, corpus, histories } from './gitgraph.mjs';

// Pele's escapes in attribute values and text, undone.
const unescape = (s) => s.replace(/&(amp|lt|gt|quot);/g, (_, e) => ({ amp: '&', lt: '<', gt: '>', quot: '"' })[e]);
const attr = (s, name) => {
  const m = new RegExp(`\\s${name}="([^"]*)"`).exec(s);
  return m ? unescape(m[1]) : undefined;
};
const TOKENS = { 'var(--_bg)': 'bg', 'var(--_fg)': 'fg', 'var(--_m)': 'muted', 'var(--_s)': 'surface', 'var(--_b)': 'border', none: 'none' };
// A paint as the test names a token of the theme.
const paintName = (v) => {
  if (v === undefined) return '-';
  const series = /^var\(--pele-series-(\d),[^)]*\)$/.exec(v);
  if (series) return `series-${series[1]}`;
  return TOKENS[v] ?? `?${v}`;
};

// Math.SQRT1_2, and 1 - k for the curves of a rounded corner, as limu's scene.Path.RoundRect
// has it.
const HALF = Math.SQRT1_2;
const KAPPA = 0.5522847498307936;
const ONE_MINUS_KAPPA = 0.4477152501692064;

// A box with corners rounded by radius, as limu's scene.Path.RoundRect draws it: a list of
// [op, ...numbers].
function roundRect(x, y, w, h, radius) {
  radius = Math.max(0, Math.min(radius, w / 2, h / 2));
  const x0 = x;
  const y0 = y;
  const x1 = x + w;
  const y1 = y + h;
  if (radius === 0) return [['M', x0, y0], ['L', x1, y0], ['L', x1, y1], ['L', x0, y1], ['Z']];
  const c = radius * ONE_MINUS_KAPPA;
  return [
    ['M', x0 + radius, y0],
    ['L', x1 - radius, y0],
    ['C', x1 - c, y0, x1, y0 + c, x1, y0 + radius],
    ['L', x1, y1 - radius],
    ['C', x1, y1 - c, x1 - c, y1, x1 - radius, y1],
    ['L', x0 + radius, y1],
    ['C', x0 + c, y1, x0, y1 - c, x0, y1 - radius],
    ['L', x0, y0 + radius],
    ['C', x0, y0 + c, x0 + c, y0, x0 + radius, y0],
    ['Z'],
  ];
}
void KAPPA;

// The lines of a git graph's SVG, drawn with its numbers in full; num writes a number as Pele
// does.
export function readSvg(svg, num) {
  const lines = [];
  const head = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  lines.push(`size ${num(Number(attr(head, 'width')))},${num(Number(attr(head, 'height')))}`);
  let dx = 0;
  let dy = 0;
  const mx = (v) => dx + Number(v);
  const my = (v) => dy + Number(v);
  const pt = (x, y) => `${num(mx(x))},${num(my(y))}`;
  // The groups open, each with the paint, stroke width, cap and join it gives what is in it.
  const stack = [{ fill: 'var(--_fg)', width: '1', cls: 'svg' }];
  const inherited = (key) => {
    for (let k = stack.length - 1; k >= 0; k--) if (stack[k][key] !== undefined) return stack[k][key];
    return undefined;
  };
  const paint = (e) => {
    const fill = attr(e, 'fill') ?? inherited('fill');
    const stroke = attr(e, 'stroke') ?? 'none';
    let s = `f=${paintName(fill)} s=${paintName(stroke)}`;
    if (stroke !== 'none') {
      s += ` w=${num(Number(attr(e, 'stroke-width') ?? inherited('width')))}`;
      // How the stroke ends and turns its corners, "-" for SVG's initial ones.
      s += ` c=${attr(e, 'stroke-linecap') ?? inherited('cap') ?? '-'} j=${attr(e, 'stroke-linejoin') ?? inherited('join') ?? '-'}`;
    }
    const dash = attr(e, 'stroke-dasharray');
    s += ` d=${dash === undefined ? '-' : dash.split(/[ ,]+/).map((v) => num(Number(v))).join(' ')}`;
    return s;
  };
  // A path's d as absolute commands, M, L, C and Z, its numbers moved; H, V and l are
  // written as L. A relative step is added to where it starts before it is moved.
  const path = (d) => {
    const out = [];
    let x = 0;
    let y = 0;
    for (const m of d.matchAll(/([MLHVCZl])([^MLHVCZl]*)/g)) {
      const n = m[2].trim() === '' ? [] : m[2].trim().split(/[ ,]+/).map(Number);
      switch (m[1]) {
        case 'M':
          [x, y] = n;
          out.push(`M ${pt(x, y)}`);
          break;
        case 'L':
          [x, y] = n;
          out.push(`L ${pt(x, y)}`);
          break;
        case 'l':
          x += n[0];
          y += n[1];
          out.push(`L ${pt(x, y)}`);
          break;
        case 'H':
          x = n[0];
          out.push(`L ${pt(x, y)}`);
          break;
        case 'V':
          y = n[0];
          out.push(`L ${pt(x, y)}`);
          break;
        case 'C':
          out.push(`C ${pt(n[0], n[1])} ${pt(n[2], n[3])} ${pt(n[4], n[5])}`);
          [x, y] = n.slice(4);
          break;
        case 'Z':
          out.push('Z');
          break;
      }
    }
    return out.join(' ');
  };
  // Where a text's first line sits, the first x and y in it; s starts with the text.
  const textAt = (s) => / x="([^"]*)" y="([^"]*)"/.exec(s);

  let turn = null;
  for (const m of svg.matchAll(/<(\/?)(svg|g|path|circle|rect|text|title|desc)\b([^>]*)>/g)) {
    const [whole, close, tag] = m;
    if (tag === 'svg' || tag === 'desc') continue;
    if (close) {
      if (tag === 'g') {
        stack.pop();
        turn = null;
      }
      continue;
    }
    switch (tag) {
      case 'title': {
        // The accessible title before the drawing has an id; a commit's message has none.
        if (attr(whole, 'id') !== undefined) break;
        const end = svg.indexOf('</title>', m.index);
        lines.push(`tip ${unescape(svg.slice(m.index + whole.length, end))}`);
        break;
      }
      case 'g': {
        const cls = attr(whole, 'class') ?? '';
        const t = /translate\(([^,]*),([^)]*)\)/.exec(attr(whole, 'transform') ?? '');
        if (t) {
          dx = Number(t[1]);
          dy = Number(t[2]);
        }
        const r = /rotate\(([^ ]*) ([^ ]*) ([^)]*)\)/.exec(attr(whole, 'transform') ?? '');
        stack.push({
          fill: attr(whole, 'fill'),
          width: attr(whole, 'stroke-width'),
          cap: attr(whole, 'stroke-linecap'),
          join: attr(whole, 'stroke-linejoin'),
          cls,
        });
        if (cls === 'pele-branch') lines.push(`branch ${attr(whole, 'data-id')}`);
        else if (cls.startsWith('pele-commit ')) lines.push(`commit ${attr(whole, 'data-id')} ${cls}`);
        else if (cls === 'pele-tag') lines.push('tag');
        else if (cls === 'pele-commit-label') {
          lines.push('label');
          if (r) turn = { angle: Number(r[1]), ax: mx(r[2]), ay: my(r[3]) };
        }
        break;
      }
      case 'path':
        if (/pele-edge/.test(attr(whole, 'class') ?? '')) lines.push(`edge ${attr(whole, 'class')}`);
        lines.push(`path ${path(attr(whole, 'd'))} ${paint(whole)}`);
        break;
      case 'circle': {
        const x = mx(attr(whole, 'cx'));
        const y = my(attr(whole, 'cy'));
        const r = Number(attr(whole, 'r'));
        lines.push(`circle ${num(x)},${num(y)} ${num(x + r - x)} ${paint(whole)}`);
        break;
      }
      case 'rect': {
        const radius = attr(whole, 'rx') ?? ((attr(whole, 'style') ?? '').includes('rx:var(--_r)') ? 'theme' : '-');
        if (turn) {
          // The box turned about the label's point, as a path.
          const { angle, ax, ay } = turn;
          const S = angle < 0 ? -HALF : HALF;
          const turned = (x, y) => {
            const vx = x - ax;
            const vy = y - ay;
            return [ax + vx * HALF - vy * S, ay + vx * S + vy * HALF];
          };
          const steps = roundRect(mx(attr(whole, 'x')), my(attr(whole, 'y')), Number(attr(whole, 'width')), Number(attr(whole, 'height')), Number(radius));
          const d = steps.map(([op, ...n]) => {
            const pts = [];
            for (let i = 0; i < n.length; i += 2) {
              const [x, y] = turned(n[i], n[i + 1]);
              pts.push(`${num(x)},${num(y)}`);
            }
            return [op, ...pts].join(' ');
          });
          lines.push(`path ${d.join(' ')} ${paint(whole)}`);
          break;
        }
        const box = `${pt(attr(whole, 'x'), attr(whole, 'y'))},${num(Number(attr(whole, 'width')))},${num(Number(attr(whole, 'height')))}`;
        lines.push(`rect ${box} r=${radius} ${paint(whole)}`);
        break;
      }
      case 'text': {
        const end = svg.indexOf('</text>', m.index);
        const s = svg.slice(m.index, end);
        const at = textAt(s);
        const fill = paintName(attr(whole, 'fill') ?? inherited('fill'));
        if (attr(whole, 'class') === 'pele-title') lines.push('title');
        if (turn) {
          // The label's middle, turned about its point: its middle across is where its
          // first line is centred, and along, the point's.
          const { angle, ax, ay } = turn;
          const S = angle < 0 ? -HALF : HALF;
          const vx = mx(at[1]) - ax;
          const vy = ay - ay;
          const x = ax + vx * HALF - vy * S;
          const y = ay + vx * S + vy * HALF;
          lines.push(`turned ${num(angle)} ${num(x)},${num(y)} f=${fill}`);
          break;
        }
        // The title is drawn outside the moved group.
        const outside = attr(whole, 'class') === 'pele-title';
        lines.push(`t ${outside ? `${num(Number(at[1]))},${num(Number(at[2]))}` : pt(at[1], at[2])} f=${fill}`);
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
    if (r.type !== 'gitGraph') return false;
    out.push({ src: source, maxWidth, fontSize, lines: readSvg(r.svg, num) });
    return true;
  };
  const seen = new Set();
  let drawn = 0;
  const each = (source) => {
    if (seen.has(source) || lone(source)) return;
    seen.add(source);
    if (!draw(source)) return;
    // Every third one again in a narrow host, which turns what runs across and does not fit,
    // and every seventh in another size of type.
    if (drawn % 3 === 0) draw(source, 360);
    if (drawn % 7 === 0) draw(source, undefined, 13);
    drawn++;
  };
  corpus(pele).forEach(each);
  const history = histories(random(31), { config: true });
  for (let i = 0; i < 1600; i++) each(history());
  const usable = [...seen].filter((s) => s.length <= 2000);
  const next = mutator(usable, FRAGMENTS, random(32));
  while (drawn < 2900) {
    const s = next();
    if (s.length <= 2000) each(s);
  }
  write('diagram/gitgraph/testdata/pele/render.json', { cases: out });
}
