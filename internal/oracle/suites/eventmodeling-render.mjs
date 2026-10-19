// Event modeling diagrams drawn end to end: what Pele's render draws, read back out of its SVG
// as one line per thing drawn, in the order drawn. limu's test writes the same lines from its
// scene.
//
// Pele draws an event model with no groups that move what is in them, so its numbers are
// where the scene has them. Pele's SVG is drawn here with its numbers written out in full, as
// the oracle's hooks have num() write them while limuExactNumbers is set, and the lines hold
// every number in full, as String() writes it, so that a difference in the last bit shows.
//
// A text is written as where its first line sits, its class, anchor, size, fill, weight and
// family, and the text of each of its lines. Under a frame tinted in its series color Pele
// draws the box in the background first; where that box has a stroke, down, the frame's own
// stroke covers it, and the line leaves it out.
//
// Cases whose example data or notes Pele's hardWrap would cut between the two halves of a
// character past the Basic Multilingual Plane are left out: a Go string holds no half of one.
import { lone } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { unescape, attr, n, pt, readHead } from '../lib/svg.mjs';
import { FRAGMENTS, corpus, randomDiagrams } from './eventmodeling.mjs';

// The subpaths of a path's d, each as one string of absolute commands, with H and V written
// as L.
function subpaths(d) {
  const out = [];
  let cur = [];
  let x = 0;
  let y = 0;
  for (const m of d.matchAll(/([MLHVCZ])([^MLHVCZ]*)/g)) {
    const nums = m[2].trim() === '' ? [] : m[2].trim().split(/[ ,]/).map(Number);
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
      case 'C':
        cur.push(`C ${pt(nums[0], nums[1])} ${pt(nums[2], nums[3])} ${pt(nums[4], nums[5])}`);
        [x, y] = [nums[4], nums[5]];
        break;
      case 'Z':
        cur.push('Z');
        break;
    }
  }
  if (cur.length) out.push(cur.join(' '));
  return out;
}

// The lines of an event model's SVG, drawn with its numbers in full.
export function readSvg(svg, fontSize) {
  const lines = [];
  const { open: head, body: all, acc } = readHead(svg);
  lines.push(`size ${pt(attr(head, 'width'), attr(head, 'height'))}`, ...acc);
  const body = all.slice(0, all.lastIndexOf('</svg>'));
  // Whether the group being read is a frame's.
  const groups = [];
  const text = (s, open) => {
    let at = / x="([^"]*)" y="([^"]*)"/.exec(open);
    const content = [];
    const inner = s.slice(open.length + 1);
    if (inner.includes('<tspan')) {
      for (const t of inner.matchAll(/<tspan([^>]*)>([^<]*)<\/tspan>/g)) {
        if (/ x="/.test(t[1]) || content.length === 0) {
          if (content.length === 0) at = / x="([^"]*)" y="([^"]*)"/.exec(t[1]);
          content.push('');
        }
        content[content.length - 1] += unescape(t[2]);
      }
    } else {
      content.push(unescape(inner));
    }
    const look = [
      attr(open, 'class') ?? '-',
      pt(at[1], at[2]),
      attr(open, 'text-anchor') ?? 'start',
      n(attr(open, 'font-size') ?? fontSize ?? 16),
      attr(open, 'fill') ?? '-',
      attr(open, 'font-weight') === undefined ? '-' : 'b',
      attr(open, 'font-family') === undefined ? '-' : 'mono',
    ];
    return `t ${look.join(' ')} | ${content.join(' | ')}`;
  };
  for (const m of body.matchAll(/<(\/?)([a-z]+)\b([^>]*?)(\/?)>/g)) {
    const [whole, close, name] = m;
    if (close) {
      if (name === 'g') groups.pop();
      continue;
    }
    switch (name) {
      case 'g': {
        const cls = attr(whole, 'class') ?? '';
        const id = attr(whole, 'data-id');
        const node = cls.startsWith('pele-node pele-em-');
        groups.push(node);
        if (cls === 'pele-cluster pele-em-lane') lines.push(`lane ${id}`);
        else if (node) lines.push(`node ${id} ${cls.split(' ')[1].slice(8)}${cls.includes('pele-em-reset') ? ' reset' : ''}`);
        else if (cls === 'pele-edge') lines.push(`edge ${id}`);
        else if (cls === 'pele-em-note' || cls === 'pele-em-specification') lines.push(`${cls.slice(8)} ${id}`);
        else lines.push(`group ${cls}`);
        break;
      }
      case 'rect': {
        const box = `${pt(attr(whole, 'x'), attr(whole, 'y'))},${pt(attr(whole, 'width'), attr(whole, 'height'))}`;
        const r = attr(whole, 'rx') !== undefined || (attr(whole, 'style') ?? '').includes('rx:') ? 'r' : '-';
        // The background under a frame's tint, whose stroke the tint's covers.
        const under = groups[groups.length - 1] && body.startsWith('<rect', m.index + whole.length);
        const stroke = under ? '-' : attr(whole, 'stroke') ?? '-';
        lines.push(`rect ${box} ${r} ${attr(whole, 'fill') ?? '-'} ${attr(whole, 'fill-opacity') ?? '-'} ${stroke}`);
        break;
      }
      case 'path': {
        const d = attr(whole, 'd');
        if (attr(whole, 'class') === 'pele-marker') {
          lines.push(`marker ${subpaths(d).join(' ')} ${attr(whole, 'fill') ?? '-'}`);
          break;
        }
        const pen = `${attr(whole, 'stroke') ?? '-'} ${attr(whole, 'fill') ?? '-'} ${attr(whole, 'stroke-linecap') ?? 'butt'} ${attr(whole, 'stroke-linejoin') ?? 'miter'}`;
        for (const p of subpaths(d)) lines.push(`path ${p} ${pen}`);
        break;
      }
      case 'text': {
        const end = body.indexOf('</text>', m.index);
        lines.push(text(body.slice(m.index, end), whole.slice(0, -1)));
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
  const { render, parse, PeleError } = await src('index.ts');
  const { metricsMeasurer } = await src('text/measurer.ts');
  const { decodeEntities } = await src('text/entities.ts');

  // Whether Pele's hardWrap cuts a line of the text between the halves of a character.
  const splits = (s, columns) =>
    decodeEntities(s)
      .split('\n')
      .some((line) => {
        for (let i = columns; i < line.length; i += columns) {
          const c = line.charCodeAt(i - 1);
          if (c >= 0xd800 && c <= 0xdbff) return true;
        }
        return false;
      });
  const out = [];
  const draw = (source, maxWidth, fontSize, autoDirection, directionBreakpoint) => {
    let r;
    globalThis.limuExactNumbers = true;
    try {
      const options = { measurer: metricsMeasurer, maxWidth, fontSize, autoDirection, directionBreakpoint, outputLimit: Infinity };
      const model = parse(source, options);
      if (model.type !== 'eventmodeling') return false;
      if (model.boxes.some((b) => b.data && splits(b.data, 48)) || model.notes.some((note) => splits(note.text, 80))) return false;
      r = render(source, options);
    } catch (e) {
      if (!(e instanceof PeleError)) throw e;
      return false;
    } finally {
      globalThis.limuExactNumbers = false;
    }
    out.push({ src: source, maxWidth, fontSize, autoDirection, directionBreakpoint, lines: readSvg(r.svg, fontSize) });
    return true;
  };
  const NARROW = [360, 320, 280, 240, 200, 160, 120, 100, 600, 500, 400, 700];
  const SIZES = [13, 20, 11, 24, 15, 13.5];
  const seen = new Set();
  let drawn = 0;
  const each = (source) => {
    // A Go string cannot hold a surrogate with no partner.
    if (seen.has(source) || lone(source) || source.length > 3000) return;
    seen.add(source);
    if (!draw(source)) return;
    // Every other one again in a narrow host, which lays the frames out down when they do not
    // fit, now and then with the direction kept or another breakpoint, and every fifth in
    // another size of type.
    if (drawn % 2 === 0) {
      const narrow = NARROW[(drawn / 2) % NARROW.length];
      const keep = drawn % 22 === 0 ? false : undefined;
      const breakpoint = drawn % 26 === 2 ? 300 : drawn % 26 === 14 ? 900 : undefined;
      draw(source, narrow, undefined, keep, breakpoint);
    }
    if (drawn % 5 === 0) draw(source, drawn % 10 === 0 ? undefined : NARROW[(drawn / 5) % NARROW.length], SIZES[(drawn / 5) % SIZES.length]);
    drawn++;
  };
  corpus(pele).forEach(each);
  const diagrams = randomDiagrams(random(61));
  for (let i = 0; i < 1500; i++) each(diagrams());
  const usable = [...seen].filter((s) => s.length <= 3000);
  const next = mutator(usable, FRAGMENTS, random(62));
  while (drawn < 2500) each(next());
  write('diagram/eventmodeling/testdata/pele/render.json', { cases: out });
}
