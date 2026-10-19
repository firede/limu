// Use case diagrams drawn end to end: what Pele's render draws, read back out of its SVG as one
// line per thing drawn, in the order drawn, with numbers as the SVG writes them. limu's test
// writes the same lines from its scene.
import { lone, mutator } from './parse.mjs';
import { FRAGMENTS, corpus, drawableDiagrams } from './usecase.mjs';

// Pele's escapes in attribute values, undone.
const unescape = (s) => s.replace(/&(amp|lt|gt|quot);/g, (_, e) => ({ amp: '&', lt: '<', gt: '>', quot: '"' })[e]);
const attr = (s, name) => {
  const m = new RegExp(`\\s${name}="([^"]*)"`).exec(s);
  return m ? unescape(m[1]) : undefined;
};
const box = (s) => `${attr(s, 'x') ?? 0},${attr(s, 'y') ?? 0},${attr(s, 'width')},${attr(s, 'height')}`;
// How a shape is painted: its fill, with its opacity, and its stroke.
const paint = (s) => {
  const opacity = attr(s, 'fill-opacity');
  return `${attr(s, 'fill') ?? '-'}${opacity ? '/' + opacity : ''} ${attr(s, 'stroke') ?? '-'}`;
};
// Where a text's first line sits, the first x and y in it, and its own fill; s starts with
// the text.
const textAt = (s) => {
  const m = / x="([^"]*)" y="([^"]*)"/.exec(s);
  return `t ${m[1]},${m[2]} ${attr(s.slice(0, s.indexOf('>')), 'fill') ?? '-'}`;
};

// The subpaths of a path's d, each as one string of absolute commands, M, L, C and Z, with
// H and V written as L. A circle drawn as two arcs is written as an ellipse.
function subpaths(d) {
  const out = [];
  let cur = [];
  let x = '0';
  let y = '0';
  for (const m of d.matchAll(/([MLHVCAZ])([^MLHVCAZ]*)/g)) {
    const nums = m[2].trim() === '' ? [] : m[2].trim().split(/[ ,]+/);
    switch (m[1]) {
      case 'M':
        if (cur.length) out.push(cur.join(' '));
        cur = [`M ${nums[0]},${nums[1]}`];
        [x, y] = nums;
        break;
      case 'L':
        cur.push(`L ${nums[0]},${nums[1]}`);
        [x, y] = nums;
        break;
      case 'H':
        x = nums[0];
        cur.push(`L ${x},${y}`);
        break;
      case 'V':
        y = nums[0];
        cur.push(`L ${x},${y}`);
        break;
      case 'C':
        cur.push(`C ${nums[0]},${nums[1]} ${nums[2]},${nums[3]} ${nums[4]},${nums[5]}`);
        [x, y] = nums.slice(4);
        break;
      case 'A':
        cur.push(`A ${nums.join(',')}`);
        [x, y] = nums.slice(5);
        break;
      case 'Z':
        cur.push('Z');
        break;
    }
  }
  if (cur.length) out.push(cur.join(' '));
  return out.map((s) => {
    // M x,y1 A r,r,0,1,1,x,y2 A r,r,0,1,1,x,y1 Z: a circle on the line from y1 to y2.
    const m = /^M ([^,\s]+),([^,\s]+) A ([^,\s]+),\3,0,1,1,\1,([^,\s]+) A \3,\3,0,1,1,\1,\2 Z$/.exec(s);
    return m ? `ellipse ${m[1]},${(Number(m[2]) + Number(m[4])) / 2} ${m[3]},${m[3]}` : s;
  });
}

// Every element of a node or cluster, in order, as lines.
function shapes(s, lines) {
  for (const m of s.matchAll(/<(circle|ellipse|path|rect|svg|text|g) [^>]*>/g)) {
    const e = m[0];
    switch (m[1]) {
      case 'circle':
      case 'ellipse': {
        const rx = attr(e, 'r') ?? attr(e, 'rx');
        const ry = attr(e, 'r') ?? attr(e, 'ry');
        lines.push(`ellipse ${attr(e, 'cx') ?? 0},${attr(e, 'cy') ?? 0} ${rx},${ry} ${paint(e)}`);
        break;
      }
      case 'path':
        for (const p of subpaths(attr(e, 'd'))) lines.push(`${p.startsWith('ellipse') ? '' : 'path '}${p} ${paint(e)}`);
        break;
      case 'rect': {
        const rounded = attr(e, 'rx') !== undefined || (attr(e, 'style') ?? '').includes('rx:');
        lines.push(`rect ${box(e)}${rounded ? ' r' : ''} ${paint(e)}`);
        break;
      }
      case 'svg':
        lines.push(`icon ${attr(e, 'data-icon')} ${box(e)}`);
        break;
      case 'text': {
        const end = s.indexOf('</text>', m.index);
        lines.push(textAt(s.slice(m.index, end)));
        break;
      }
      case 'g':
        if (e.includes('pele-actor-figure')) lines.push(`figure ${/translate\(([^)]*)\)/.exec(e)[1]}`);
        break;
    }
  }
}

// A class attribute's classes after the first n, without the class of an edge's animation,
// which says how Mermaid would move its dashes.
const classesAfter = (cls, n) =>
  cls
    .split(' ')
    .slice(n)
    .filter((c) => !/^edge-animation-/.test(c))
    .join(' ');

const GROUP = /<g class="pele-(cluster|edge|edge-label|node)[ "]/g;

// The lines of a use case diagram's SVG.
export function readSvg(svg) {
  const lines = [];
  const head = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  lines.push(`size ${attr(head, 'width')},${attr(head, 'height')}`);
  const title = svg.indexOf('<text class="pele-title"');
  if (title >= 0) lines.push('title ' + textAt(svg.slice(title)));
  const starts = [...svg.matchAll(GROUP)];
  for (let k = 0; k < starts.length; k++) {
    const kind = starts[k][1];
    let s = svg.slice(starts[k].index, k + 1 < starts.length ? starts[k + 1].index : svg.length);
    const open = s.slice(0, s.indexOf('>'));
    s = s.slice(open.length);
    const id = attr(open, 'data-id');
    const cls = attr(open, 'class');
    switch (kind) {
      case 'cluster':
        lines.push(`cluster ${id} [${classesAfter(cls, 2)}]`);
        shapes(s, lines);
        break;
      case 'edge': {
        const type = cls.split(' ')[1].slice('pele-'.length);
        const path = /<path d="[^"]*"[^>]*>/.exec(s)[0];
        lines.push(`edge ${id} ${type} [${classesAfter(cls, 2)}] ${attr(path, 'stroke-dasharray') ?? '-'}`);
        for (const p of subpaths(attr(path, 'd'))) lines.push('path ' + p);
        for (const m of s.matchAll(/<(path|circle) class="pele-marker"[^>]*>/g)) {
          lines.push(m[1] === 'path' ? `m ${subpaths(attr(m[0], 'd')).join(' ')}` : `m o ${attr(m[0], 'cx')},${attr(m[0], 'cy')}`);
        }
        break;
      }
      case 'edge-label':
        lines.push(`label ${id} ${box(/<rect [^>]*>/.exec(s)[0])}`, textAt(s.slice(s.indexOf('<text'))));
        break;
      case 'node':
        lines.push(`node ${id} [${classesAfter(cls, 2)}] ${attr(open, 'transform')}`);
        shapes(s, lines);
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
      if (e instanceof RangeError) return false;
      if (!(e instanceof PeleError)) throw e;
      return false;
    }
    if (r.type !== 'usecase') return false;
    out.push({ src: source, maxWidth, fontSize, lines: readSvg(r.svg) });
    return true;
  };
  const seen = new Set();
  let drawn = 0;
  const each = (source) => {
    // A Go string cannot hold a surrogate with no partner.
    if (seen.has(source) || lone(source)) return;
    seen.add(source);
    if (!draw(source)) return;
    // Every third one again in a narrow host, which turns what runs across and does not fit,
    // and every seventh in another size of type.
    if (drawn % 3 === 0) draw(source, 360);
    if (drawn % 7 === 0) draw(source, undefined, 13);
    drawn++;
  };
  const base = corpus(pele);
  base.forEach(each);
  const drawable = drawableDiagrams(random(13));
  for (let i = 0; i < 1500; i++) each(drawable());
  const usable = [...seen].filter((s) => s.length <= 3000);
  const next = mutator(usable, FRAGMENTS, random(14));
  while (drawn < 3000) {
    const s = next();
    if (s.length <= 3000) each(s);
  }
  write('diagram/usecase/testdata/pele/render.json', { cases: out });
}
