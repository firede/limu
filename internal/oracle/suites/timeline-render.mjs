// Timeline diagrams drawn end to end: what Pele's render draws, read back out of its SVG as one
// line per thing drawn, in the order drawn. limu's test writes the same lines from its scene.
//
// Pele draws a timeline with no group that moves what is in it, so the numbers of its SVG are
// where things are on the page, as the scene holds them, and are compared as Pele writes them.
import { lone } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { unescape, attr } from '../lib/svg.mjs';
import { FRAGMENTS, corpus, drawableDiagrams } from './timeline.mjs';

// How a shape is painted: its fill, with its opacity, its stroke, and the stroke's width.
const paint = (s) => {
  const opacity = attr(s, 'fill-opacity');
  const width = attr(s, 'stroke-width');
  return `${attr(s, 'fill') ?? '-'}${opacity ? '/' + opacity : ''} ${attr(s, 'stroke') ?? '-'}${width ? ' w' + width : ''}`;
};

// The subpaths of a path's d, each as one string of absolute commands, M, L and Z, with H and V
// written as L.
function subpaths(d) {
  const out = [];
  let cur = [];
  let x = '0';
  let y = '0';
  for (const m of d.matchAll(/([MLHVZ])([^MLHVZ]*)/g)) {
    const nums = m[2].trim() === '' ? [] : m[2].trim().split(/[ ,]+/);
    switch (m[1]) {
      case 'M':
        if (cur.length) out.push(cur.join(' '));
        [x, y] = nums;
        cur = [`M ${x},${y}`];
        break;
      case 'L':
        [x, y] = nums;
        cur.push(`L ${x},${y}`);
        break;
      case 'H':
        x = nums[0];
        cur.push(`L ${x},${y}`);
        break;
      case 'V':
        y = nums[0];
        cur.push(`L ${x},${y}`);
        break;
      case 'Z':
        cur.push('Z');
        break;
    }
  }
  if (cur.length) out.push(cur.join(' '));
  return out;
}

// A text: where its first line sits, its fill and size, and the text of each line, its spans
// joined.
function textLine(s, size) {
  const open = s.slice(0, s.indexOf('>'));
  const m = / x="([^"]*)" y="([^"]*)"/.exec(s);
  const body = s.slice(open.length + 1);
  let lines;
  if (!body.startsWith('<tspan')) {
    lines = [unescape(body)];
  } else {
    lines = [];
    for (const t of body.matchAll(/<tspan([^>]*)>([^<]*)<\/tspan>/g)) {
      if (/ x="/.test(t[1])) lines.push('');
      lines[lines.length - 1] += unescape(t[2]);
    }
  }
  return `t ${m[1]},${m[2]} ${attr(open, 'fill') ?? '-'} ${attr(open, 'font-size') ?? size} | ${lines.join(' | ')}`;
}

// The lines of a timeline's SVG.
export function readSvg(svg, size) {
  const lines = [];
  const head = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  lines.push(`size ${attr(head, 'width')},${attr(head, 'height')}`);
  for (const m of svg.matchAll(/<(rect|text|path|circle|g)\b[^>]*>/g)) {
    const e = m[0];
    switch (m[1]) {
      case 'g': {
        const cls = attr(e, 'class') ?? '';
        const id = attr(e, 'data-id');
        lines.push(`${cls.split(' ')[1].slice('pele-'.length)}${id === undefined ? '' : ' ' + id}`);
        break;
      }
      case 'rect':
        lines.push(`rect ${attr(e, 'x')},${attr(e, 'y')},${attr(e, 'width')},${attr(e, 'height')}${attr(e, 'rx') || /rx:/.test(attr(e, 'style') ?? '') ? ' r' : ''} ${paint(e)}`);
        break;
      case 'text': {
        const end = svg.indexOf('</text>', m.index);
        lines.push((attr(e, 'class') === 'pele-title' ? 'title ' : '') + textLine(svg.slice(m.index, end), size));
        break;
      }
      case 'path':
        lines.push(`${attr(e, 'class')} ${subpaths(attr(e, 'd')).join(' ')} ${paint(e)}`);
        break;
      case 'circle':
        lines.push(`circle ${attr(e, 'cx')},${attr(e, 'cy')} ${attr(e, 'r')} ${paint(e)}`);
        break;
    }
  }
  return lines;
}

export default async function ({ src, pele, write, random }) {
  const { render, PeleError } = await src('index.ts');
  const { metricsMeasurer } = await src('text/measurer.ts');

  const out = [];
  const draw = (source, maxWidth, fontSize) => {
    let r;
    try {
      r = render(source, { measurer: metricsMeasurer, maxWidth, fontSize, outputLimit: Infinity });
    } catch (e) {
      if (!(e instanceof PeleError)) throw e;
      return false;
    }
    if (r.type !== 'timeline') return false;
    out.push({ src: source, maxWidth, fontSize, lines: readSvg(r.svg, fontSize ?? 16) });
    return true;
  };
  const seen = new Set();
  let drawn = 0;
  const each = (source) => {
    // A Go string cannot hold a surrogate with no partner.
    if (seen.has(source) || lone(source)) return;
    seen.add(source);
    if (!draw(source)) return;
    // Every third one again in hosts of a few widths, which turn what runs across and does not
    // fit and share the width between the columns of what runs down, and every seventh in
    // another size of type.
    if (drawn % 3 === 0) draw(source, [360, 240, 500, 700][(drawn / 3) % 4]);
    if (drawn % 7 === 0) draw(source, undefined, [13, 20][(drawn / 7) % 2]);
    drawn++;
  };
  corpus(pele).forEach(each);
  const drawable = drawableDiagrams(random(13));
  for (let i = 0; i < 1500; i++) each(drawable());
  const usable = [...seen].filter((s) => s.length <= 3000);
  const next = mutator(usable, FRAGMENTS, random(14));
  while (drawn < 3000) {
    const s = next();
    if (s.length <= 3000) each(s);
  }
  write('diagram/timeline/testdata/pele/render.json', { cases: out });
}
