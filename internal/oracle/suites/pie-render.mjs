// Pie charts drawn end to end, some with a padding other than Pele's 8: what Pele's render draws, read back out of its SVG as one line
// per thing drawn, in the order drawn, and the errors of the charts it refuses to draw. limu's
// test writes the same lines from its scene.
//
// Pele draws the slices and their percentages in groups moved to the pie's center, where the
// scene holds every place on the page. Pele's SVG is drawn here with its numbers written out
// in full, as the oracle's hooks have num() write them while limuExactNumbers is set, and each
// number in a moved group is moved here as the scene's is, the center added to it, so that the
// lines hold every number in full, as String() writes it, and a difference in the last bit
// shows. A percentage's baseline is where Pele puts it before it adds the shift, which the
// hooks write as data-y, moved, with the shift added after, as limu's test adds it to the
// middle of the label.
import { lone, mutator } from './parse.mjs';
import { FRAGMENTS, corpus, randomDiagrams } from './pie.mjs';

// Pele's escapes in attribute values and text, undone.
const unescape = (s) => s.replace(/&(amp|lt|gt|quot);/g, (_, e) => ({ amp: '&', lt: '<', gt: '>', quot: '"' })[e]);
const attr = (s, name) => {
  const m = new RegExp(`\\s${name}="([^"]*)"`).exec(s);
  return m ? unescape(m[1]) : undefined;
};
const n = (x) => String(Number(x));
const pt = (x, y) => `${n(x)},${n(y)}`;
const TRANSLATE = /^translate\(([^,]*),([^)]*)\)$/;

// What a group or text of Pele's chart is called in the lines, by its class.
const CONTEXT = {
  'pele-slice-labels': 'slice-labels',
  'pele-legend-item': 'legend',
  'pele-title': 'title',
};

// A path's d moved by dx, dy: each subpath as one string of absolute commands, an arc with
// its radii, turn and flags.
function moved(d, dx, dy) {
  const out = [];
  for (const m of d.matchAll(/([MLAZ])([^MLAZ]*)/g)) {
    const nums = m[2].trim() === '' ? [] : m[2].trim().split(/[\s,]+/).map(Number);
    switch (m[1]) {
      case 'M':
        out.push(`M ${pt(dx + nums[0], dy + nums[1])}`);
        break;
      case 'L':
        out.push(`L ${pt(dx + nums[0], dy + nums[1])}`);
        break;
      case 'A':
        out.push(`A ${nums.slice(0, 5).map(n).join(',')} ${pt(dx + nums[5], dy + nums[6])}`);
        break;
      case 'Z':
        out.push('Z');
        break;
    }
  }
  return out.join(' ');
}

// The lines of a pie chart's SVG, drawn with its numbers in full.
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
  // The groups open about the element read: each one's context, move, fill, stroke, anchor,
  // size of type and the rest of its pen.
  const stack = [];
  const inherited = (name) => {
    for (let k = stack.length - 1; k >= 0; k--) if (stack[k][name] !== undefined) return stack[k][name];
    return undefined;
  };
  const context = (e) => CONTEXT[(attr(e, 'class') ?? '').split(' ')[0]] ?? inherited('context') ?? '?';
  const own = (e, name, inheritAs) => attr(e, name) ?? inherited(inheritAs ?? name);
  const pen = (e) =>
    `${own(e, 'stroke') ?? '-'} ${own(e, 'stroke-width', 'width') ?? '-'} ${own(e, 'stroke-linecap', 'cap') ?? 'butt'} ${own(e, 'stroke-linejoin', 'join') ?? 'miter'}`;
  // A text's first line, where it sits, its fill, anchor and size, and the text of each line.
  // A percentage is moved with its group, from where Pele puts it before the shift.
  const text = (s, open) => {
    const inner = s.slice(open.length + 1);
    let at = / x="([^"]*)" y="([^"]*)"/.exec(open);
    let content;
    if (inner.includes('<tspan')) {
      const out = [];
      for (const t of inner.matchAll(/<tspan([^>]*)>([^<]*)<\/tspan>/g)) {
        if (/ x="/.test(t[1]) || out.length === 0) {
          if (out.length === 0) at = / x="([^"]*)" y="([^"]*)"/.exec(t[1]);
          out.push('');
        }
        out[out.length - 1] += unescape(t[2]);
      }
      content = out;
    } else {
      content = [unescape(inner)];
    }
    const size = own(open, 'font-size', 'size') ?? String(fontSize ?? 16);
    let x = Number(at[1]);
    let y = Number(at[2]);
    const dy = attr(open, 'data-y');
    const move = inherited('move');
    if (move) {
      x = move[0] + x;
      y = dy === undefined ? move[1] + y : move[1] + Number(dy) + Number(size) * 0.35;
    }
    const fill = own(open, 'fill') ?? '-';
    const anchor = own(open, 'text-anchor', 'anchor') ?? '-';
    return `${context(open)} t ${pt(x, y)} ${fill} ${anchor} ${n(size)} | ${content.join(' | ')}`;
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
        const t = TRANSLATE.exec(attr(whole, 'transform') ?? '');
        stack.push({
          context: CONTEXT[cls],
          move: t ? [Number(t[1]), Number(t[2])] : undefined,
          fill: attr(whole, 'fill'),
          stroke: attr(whole, 'stroke'),
          width: attr(whole, 'stroke-width'),
          cap: attr(whole, 'stroke-linecap'),
          join: attr(whole, 'stroke-linejoin'),
          anchor: attr(whole, 'text-anchor'),
          size: attr(whole, 'font-size'),
        });
        if (cls === 'pele-legend-item') lines.push(`legend ${attr(whole, 'data-id')}`);
        break;
      }
      case 'path': {
        const [dx, dy] = inherited('move') ?? [0, 0];
        lines.push(`slice ${attr(whole, 'data-id')} ${own(whole, 'fill') ?? '-'} ${pen(whole)}`);
        lines.push(`path ${moved(attr(whole, 'd'), dx, dy)}`);
        break;
      }
      case 'circle': {
        const [dx, dy] = inherited('move') ?? [0, 0];
        const cx = dx + Number(attr(whole, 'cx') ?? 0);
        const cy = dy + Number(attr(whole, 'cy') ?? 0);
        lines.push(`circle ${pt(cx, cy)} ${n(attr(whole, 'r'))} ${own(whole, 'fill') ?? '-'} ${pen(whole)}`);
        break;
      }
      case 'rect': {
        const r = attr(whole, 'rx');
        const box = `${pt(attr(whole, 'x'), attr(whole, 'y'))},${pt(attr(whole, 'width'), attr(whole, 'height'))}`;
        lines.push(`rect ${box} ${r === undefined ? '-' : 'r' + n(r)} ${own(whole, 'fill') ?? '-'}`);
        break;
      }
      case 'text': {
        const end = body.indexOf('</text>', m.index);
        lines.push(text(body.slice(m.index, end), whole.slice(0, -1)));
        break;
      }
      case 'svg':
        lines.push('icon');
        break;
    }
  }
  return lines;
}

// Pies with many thin slices, whose percentages go outside the rim and are moved apart.
function thinDiagrams(rnd) {
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  return () => {
    let s = '';
    const settings = [];
    if (rnd() < 0.5) settings.push(`textPosition: ${pick([0.3, 0.5, 0.75, 0.9, 1])}`);
    if (rnd() < 0.4) settings.push(`donutHole: ${pick([0.2, 0.5, 0.9])}`);
    if (rnd() < 0.5) settings.push(`legendPosition: ${pick(['left', 'right', 'top', 'bottom', 'center'])}`);
    if (settings.length) s += `---\nconfig:\n  pie:\n${settings.map((l) => '    ' + l).join('\n')}\n---\n`;
    s += 'pie\n';
    const count = 2 + Math.floor(rnd() * 24);
    for (let k = 0; k < count; k++) {
      const value = rnd() < 0.7 ? 1 + Math.floor(rnd() * 5) : rnd() < 0.5 ? 50 + Math.floor(rnd() * 400) : rnd() * 3;
      s += ` "s${k}": ${value}\n`;
    }
    return s;
  };
}

export default async function ({ pele, src, write, random }) {
  const { render, PeleError } = await src('index.ts');
  const { metricsMeasurer } = await src('text/measurer.ts');

  const out = [];
  const draw = (source, maxWidth, fontSize, padding) => {
    let r;
    globalThis.limuExactNumbers = true;
    try {
      r = render(source, { measurer: metricsMeasurer, maxWidth, fontSize, padding, outputLimit: Infinity });
    } catch (e) {
      if (!(e instanceof PeleError)) throw e;
      return false;
    } finally {
      globalThis.limuExactNumbers = false;
    }
    if (r.type !== 'pie') return false;
    out.push({ src: source, maxWidth, fontSize, padding, lines: readSvg(r.svg, fontSize) });
    return true;
  };
  const NARROW = [360, 320, 280, 240, 200, 160, 120, 100, 600, 500, 400, 300];
  const SIZES = [13, 20, 11, 24, 15, 8];
  const PADS = [0, 20, 2.5];
  const seen = new Set();
  let drawn = 0;
  const each = (source) => {
    // A Go string cannot hold a surrogate with no partner.
    if (seen.has(source) || lone(source)) return;
    seen.add(source);
    if (!draw(source)) return;
    // Every third one again in a narrow host, which draws the pie smaller and its legend
    // under it, and every fifth in another size of type.
    if (drawn % 3 === 0) draw(source, NARROW[(drawn / 3) % NARROW.length]);
    if (drawn % 5 === 0) draw(source, drawn % 10 === 0 ? undefined : NARROW[(drawn / 5) % NARROW.length], SIZES[(drawn / 5) % SIZES.length]);
    // Every twentieth with a padding of its own, some in a narrow host, which takes the
    // padding from the room for the title.
    if (drawn % 20 === 0) draw(source, drawn % 40 === 0 ? undefined : NARROW[(drawn / 20) % NARROW.length], undefined, PADS[(drawn / 20) % PADS.length]);
    drawn++;
  };
  corpus(pele).forEach(each);
  const diagrams = randomDiagrams(random(41));
  for (let i = 0; i < 1500; i++) each(diagrams());
  const thin = thinDiagrams(random(43));
  for (let i = 0; i < 400; i++) each(thin());
  const usable = [...seen].filter((s) => s.length <= 3000);
  const next = mutator(usable, FRAGMENTS, random(42));
  while (drawn < 3000) {
    const s = next();
    if (s.length <= 3000) each(s);
  }
  write('diagram/pie/testdata/pele/render.json', { cases: out });
}
