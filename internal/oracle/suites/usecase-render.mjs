// Use case diagrams drawn end to end: what Pele's render draws, read back out of its SVG as one
// line per thing drawn, in the order drawn. limu's test writes the same lines from its scene.
//
// A scene has no groups to move what is in them, so it holds where everything is on the page,
// where Pele writes the inside of a node about the node's middle, and an actor's figure about a
// point of its own. Pele's SVG is drawn here with its numbers written out in full, as the
// oracle's hooks have num() write them while limuExactNumbers is set. A number inside groups is
// moved as limu's render moves it: the groups' moves are added up from the outermost in, the
// number is added to their sum, and only then is it written to two decimals as Pele writes it.
import { join } from 'node:path';
import { lone } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { attr, TRANSLATE, stroke } from '../lib/svg.mjs';
import { FRAGMENTS, corpus, drawableDiagrams } from './usecase.mjs';

// A stroke's ends or corners as a browser takes them on an element, as lib/svg.mjs's stroke
// settles them, from what it inherits.
const linecap = (e, inherited) => stroke('stroke-linecap', [e], inherited);
const linejoin = (e, inherited) => stroke('stroke-linejoin', [e], inherited);
// How an element inside groups whose caps and joins are up ends and turns its corners, after
// its own: " cap=" and " join=" with what a browser draws, left out where it is SVG's initial
// butt or miter, and the join left out too where the element has no corners.
const ends = (e, up, corners = true) => {
  const cap = linecap(e, up.cap);
  const join = corners ? linejoin(e, up.join) : undefined;
  return `${cap && cap !== 'butt' ? ' cap=' + cap : ''}${join && join !== 'miter' ? ' join=' + join : ''}`;
};
// The caps and joins inside each group that opens in svg, by where it opens.
function groupEnds(svg) {
  const at = new Map();
  const stack = [{}];
  for (const m of svg.matchAll(/<(\/?)g\b[^>]*>/g)) {
    if (m[1]) {
      stack.pop();
      continue;
    }
    const up = stack[stack.length - 1];
    const inside = { cap: linecap(m[0], up.cap), join: linejoin(m[0], up.join) };
    stack.push(inside);
    at.set(m.index, inside);
  }
  return at;
}

// How a shape is painted: its fill, with its opacity, and its stroke, and when it has a
// stroke, its caps and joins after those of up, the group it is in; its joins only where it
// has corners, which a circle and a single straight line do not.
const paint = (s, up, corners = true) => {
  const opacity = attr(s, 'fill-opacity');
  const stroke = attr(s, 'stroke');
  return `${attr(s, 'fill') ?? '-'}${opacity ? '/' + opacity : ''} ${stroke ?? '-'}${stroke && stroke !== 'none' ? ends(s, up, corners) : ''}`;
};

// How the numbers of what is drawn are read: moves holds the moves of the groups they are in,
// the outermost first, and num writes a number as Pele does.
function frame(num, moves) {
  const move = (v, axis) => {
    if (moves.length === 0) return v;
    let by = moves[0][axis];
    for (let k = 1; k < moves.length; k++) by = by + moves[k][axis];
    return by + v;
  };
  // A point, moved; a size, which no group moves.
  const pt = (x, y) => `${num(move(Number(x), 0))},${num(move(Number(y), 1))}`;
  const size = (v) => num(Number(v));
  // An ellipse as the scene holds it, by its middle and its right end and bottom: the middle,
  // moved, and the radii as far as those ends are from it.
  const ellipse = (cx, cy, rx, ry) => {
    const x = move(Number(cx), 0);
    const y = move(Number(cy), 1);
    return `ellipse ${num(x)},${num(y)} ${num(x + Number(rx) - x)},${num(y + Number(ry) - y)}`;
  };
  const box = (s) => `${pt(attr(s, 'x') ?? 0, attr(s, 'y') ?? 0)},${size(attr(s, 'width'))},${size(attr(s, 'height'))}`;
  // Where a text's first line sits, the first x and y in it, and its own fill; s starts with
  // the text.
  const textAt = (s) => {
    const m = / x="([^"]*)" y="([^"]*)"/.exec(s);
    return `t ${pt(m[1], m[2])} ${attr(s.slice(0, s.indexOf('>')), 'fill') ?? '-'}`;
  };
  // The subpaths of a path's d, each as one string of absolute commands, M, L, C and Z, with
  // H and V written as L. A circle drawn as two arcs is written as an ellipse, its middle
  // found before it is moved.
  const subpaths = (d) => {
    const out = [];
    let cur = [];
    let x = '0';
    let y = '0';
    for (const m of d.matchAll(/([MLHVCAZ])([^MLHVCAZ]*)/g)) {
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
        case 'C':
          cur.push(['C', ...nums]);
          [x, y] = nums.slice(4);
          break;
        case 'A':
          cur.push(['A', ...nums]);
          [x, y] = nums.slice(5);
          break;
        case 'Z':
          cur.push(['Z']);
          break;
      }
    }
    if (cur.length) out.push(cur);
    return out.map((s) => {
      // M x,y1 A r,r,0,1,1,x,y2 A r,r,0,1,1,x,y1 Z: a circle on the line from y1 to y2.
      const [m, a, b, z] = s;
      if (
        s.length === 4 &&
        a[0] === 'A' &&
        b[0] === 'A' &&
        z[0] === 'Z' &&
        a.slice(1).join() === [a[1], a[1], 0, 1, 1, m[1], a[7]].join() &&
        b.slice(1).join() === [a[1], a[1], 0, 1, 1, m[1], m[2]].join()
      ) {
        return ellipse(m[1], (Number(m[2]) + Number(a[7])) / 2, a[1], a[1]);
      }
      return s
        .map(([op, ...nums]) => {
          if (op === 'A') return `A ${nums.join(',')}`;
          const pts = [];
          for (let i = 0; i < nums.length; i += 2) pts.push(pt(nums[i], nums[i + 1]));
          return [op, ...pts].join(' ');
        })
        .join(' ');
    });
  };
  return { pt, size, box, ellipse, textAt, subpaths, moves };
}

// Every element of a node or cluster, in order, as lines; up is the caps and joins inside its
// group. A group inside, such as an actor's figure, moves what is in it.
function shapes(s, lines, f, up) {
  const open = [];
  const within = [up];
  for (const m of s.matchAll(/<(\/?)(circle|ellipse|path|rect|svg|text|g)\b[^>]*>/g)) {
    const e = m[0];
    if (m[1]) {
      if (m[2] === 'g') {
        // What is read runs on past the end of the group it starts in.
        if (within.length > 1) within.pop();
        if (open.pop()) f.moves.pop();
      }
      continue;
    }
    switch (m[2]) {
      case 'circle':
      case 'ellipse': {
        const rx = attr(e, 'r') ?? attr(e, 'rx');
        const ry = attr(e, 'r') ?? attr(e, 'ry');
        lines.push(`${f.ellipse(attr(e, 'cx') ?? 0, attr(e, 'cy') ?? 0, rx, ry)} ${paint(e, within[within.length - 1], false)}`);
        break;
      }
      case 'path':
        for (const p of f.subpaths(attr(e, 'd'))) {
          const corners = !p.startsWith('ellipse') && !/^M \S+ L \S+$/.test(p);
          lines.push(`${p.startsWith('ellipse') ? '' : 'path '}${p} ${paint(e, within[within.length - 1], corners)}`);
        }
        break;
      case 'rect': {
        const rounded = attr(e, 'rx') !== undefined || (attr(e, 'style') ?? '').includes('rx:');
        lines.push(`rect ${f.box(e)}${rounded ? ' r' : ''} ${paint(e, within[within.length - 1])}`);
        break;
      }
      case 'svg':
        lines.push(`icon ${attr(e, 'data-icon')} ${f.box(e)}`);
        break;
      case 'text': {
        const end = s.indexOf('</text>', m.index);
        lines.push(f.textAt(s.slice(m.index, end)));
        break;
      }
      case 'g': {
        const t = TRANSLATE.exec(attr(e, 'transform') ?? '');
        if (t) f.moves.push([Number(t[1]), Number(t[2])]);
        open.push(t !== null);
        const top = within[within.length - 1];
        within.push({ cap: linecap(e, top.cap), join: linejoin(e, top.join) });
        break;
      }
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

// Fills an author sets, and what lies on them: the texts of a boundary and of nodes on fills
// light and dark, relations inside a boundary's fill, which take the color of its labels, and
// relations that cross its border, or lie in a fill no color is picked on, which are laid on
// casings. Pele's casings follow its boundaries, and are read with the last of them.
const FILLS = [
  'usecase-beta\nactor A\nsystemBoundary S["Shop"]\n  U(Buy) <<UC>>\n  V(Pay)\n  W[Refund]\nend\nA --> U\nU --> V\nV ..> : extend W\nU ..> : include W\nU --o V\nU --x W\nU --> U\nstyle S fill:#123',
  'usecase-beta\ndirection LR\nsystemBoundary P["Pkg"]@{ type: package }\n  U\n  V\n  W\nend\nX --o U\nW x-- X\nU e1@--> V\nV e2@--> W\nstyle P fill:lightyellow\nstyle e1 stroke:#f00,stroke-width:3px,opacity:0.5\nstyle e2 stroke-width:2px,stroke-opacity:0.4',
  'usecase-beta\nsystemBoundary S\n  U\n  V\nend\nX --> U\nU --> V\nU e1@--> X\nstyle S fill:#12345678\nstyle e1 stroke:none',
  'usecase-beta\nsystemBoundary S\n  U\n  V\nend\nU --> V\nX --> V\nstyle S fill:#123,color:#ff0',
  'usecase-beta\nsystemBoundary S\n  U\n  V\nend\nU --> V\nX --> V\nstyle S fill:#123,color:notacolor',
  'usecase-beta\nsystemBoundary S\n  U\nend\nsystemBoundary T\n  V\nend\nU --> V\nU ..> : extend V\nclassDef dark fill:navy\nclass S,T dark',
  'usecase-beta\nsystemBoundary S\n  U\n  V\nend\nU --> V\nstyle S fill:transparent',
  'usecase-beta\nsystemBoundary S\n  U\n  V\nend\nU --> V\nstyle S fill:rgb(0 0 0 / 40%)',
  'usecase-beta\nsystemBoundary S\n  U\n  V\nend\nU --> V\nstyle S fill:#000,fill-opacity:0.3',
  'usecase-beta\nsystemBoundary S\n  U\n  V\nend\nU --> V\nstyle S fill:#000,opacity:calc(1)',
  'usecase-beta\nactor A <<Person>>\nU(Use) <<UC>>\nV[Box] <<Core>>\nnote for U "A note"\njson t@{"id":1,"items":[{"sku":"A"}]}\njson q@{"a":"b"}\nA --> U\nU --> t\nstyle A fill:#000\nstyle U fill:#000\nstyle V fill:#ffd\nstyle t fill:#202020\nstyle q fill:#202020,color:#0f0',
  'usecase-beta\nU@{ business: true }\nV\nU --> V\nclassDef c fill:hsl(240, 100%, 25%)\nclass U,V c',
];

// The lines of a use case diagram's SVG, drawn with its numbers in full; num writes a number as
// Pele does.
export function readSvg(svg, num) {
  const lines = [];
  const page = frame(num, []);
  const head = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  lines.push(`size ${page.size(attr(head, 'width'))},${page.size(attr(head, 'height'))}`);
  const title = svg.indexOf('<text class="pele-title"');
  if (title >= 0) lines.push('title ' + page.textAt(svg.slice(title)));
  const inGroup = groupEnds(svg);
  const starts = [...svg.matchAll(GROUP)];
  for (let k = 0; k < starts.length; k++) {
    const kind = starts[k][1];
    let s = svg.slice(starts[k].index, k + 1 < starts.length ? starts[k + 1].index : svg.length);
    const open = s.slice(0, s.indexOf('>'));
    s = s.slice(open.length);
    const id = attr(open, 'data-id');
    const cls = attr(open, 'class');
    // The caps and joins inside the group.
    const up = inGroup.get(starts[k].index);
    switch (kind) {
      case 'cluster':
        lines.push(`cluster ${id} [${classesAfter(cls, 2)}]`);
        shapes(s, lines, page, up);
        break;
      case 'edge': {
        const type = cls.split(' ')[1].slice('pele-'.length);
        const path = /<path d="[^"]*"[^>]*>/.exec(s)[0];
        // An edge inside a fill is stroked in the color ink picks on it.
        const ink = attr(open, 'stroke');
        lines.push(`edge ${id} ${type} [${classesAfter(cls, 2)}] ${attr(path, 'stroke-dasharray') ?? '-'}${ink ? ' ink=' + ink : ''}`);
        for (const p of page.subpaths(attr(path, 'd'))) lines.push('path ' + p + ends(path, up));
        for (const m of s.matchAll(/<(path|circle) class="pele-marker"[^>]*>/g)) {
          const stroked = attr(m[0], 'stroke') !== 'none';
          lines.push((m[1] === 'path' ? `m ${page.subpaths(attr(m[0], 'd')).join(' ')}` : `m o ${page.pt(attr(m[0], 'cx'), attr(m[0], 'cy'))}`) + (stroked ? ends(m[0], up) : ''));
        }
        break;
      }
      case 'edge-label':
        lines.push(`label ${id} ${page.box(/<rect [^>]*>/.exec(s)[0])}`, page.textAt(s.slice(s.indexOf('<text'))));
        break;
      case 'node': {
        // The node's middle, which moves everything in it, and is checked through where that
        // is drawn: a scene holds no middle of its own for a node.
        const t = TRANSLATE.exec(attr(open, 'transform'));
        lines.push(`node ${id} [${classesAfter(cls, 2)}]`);
        shapes(s, lines, frame(num, [[Number(t[1]), Number(t[2])]]), up);
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
    if (r.type !== 'usecase') return false;
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
  FILLS.forEach(each);
  write('diagram/usecase/testdata/pele/render.json', { cases: out });
}
