// C4 diagrams drawn end to end: what Pele's render draws, read back out of its SVG as one line
// per thing drawn, in the order drawn, and the errors of the diagrams it refuses to draw.
// limu's test writes the same lines from its scene.
//
// Pele draws all but the title in a group moved by the diagram's padding, and the inside of
// each shape about the shape's middle; a scene holds every place on the page. Pele's SVG is
// drawn here with its numbers written out in full, as the oracle's hooks have num() write them
// while limuExactNumbers is set. A number inside groups is moved as limu moves it: the groups'
// moves are added up from the outermost in, and the number is added to their sum. The lines
// hold every number in full, as String() writes it, so that a difference in the last bit
// shows. The hooks also have the C4 renderer write, before each label, the point it is drawn
// about, its size and its lines, before each marker its tip and its direction, and on each
// shape its width and height, which the SVG holds only worked into other numbers.
//
// The author's colors are written as the style attribute Pele gives an element, after css=;
// the test reads them with limu's reader of declarations before it compares.
import { join } from 'node:path';
import { peleError } from '../lib/value.mjs';
import { lone } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { unescape, attr, n, readHead } from '../lib/svg.mjs';
import { readLabel, readMarker } from '../lib/limu.mjs';
import { FRAGMENTS, corpus, drawableDiagrams, statementDiagrams } from './c4.mjs';

const TRANSLATE = /^translate\(([^,]*),([^)]*)\)$/;

// The lines of a C4 diagram's SVG, drawn with its numbers in full.
export function readSvg(svg) {
  const lines = [];
  const { open: head, body: all, acc } = readHead(svg);
  lines.push(`size ${n(attr(head, 'width'))},${n(attr(head, 'height'))}`, ...acc);
  const body = all.slice(0, all.lastIndexOf('</svg>'));

  // The groups open about the element read: each one's move, if it has one, its class, and
  // the cap and join of the lines in it.
  const stack = [{ cls: '', cap: 'butt', join: 'miter' }];
  const top = () => stack[stack.length - 1];
  const within = (cls) => stack.some((g) => g.cls.split(' ').includes(cls));
  const move = (v, axis) => {
    const moves = stack.filter((g) => g.move).map((g) => g.move);
    if (moves.length === 0) return Number(v);
    let by = moves[0][axis];
    for (let k = 1; k < moves.length; k++) by = by + moves[k][axis];
    return by + Number(v);
  };
  const pt = (x, y) => `${String(move(x, 0))},${String(move(y, 1))}`;
  const box = (e) => `${pt(attr(e, 'x') ?? 0, attr(e, 'y') ?? 0)},${n(attr(e, 'width'))},${n(attr(e, 'height'))}`;
  const css = (e) => ` css=${attr(e, 'style') ?? ''}`;
  const paint = (e) => `${attr(e, 'fill') ?? '-'} ${attr(e, 'stroke') ?? '-'} ${attr(e, 'stroke-dasharray') ?? '-'}`;
  // The subpaths of a line's d, as absolute commands M, L, Q, C and Z, with H and V written as
  // L from where the line was.
  const path = (d) => {
    const out = [];
    let x = 0;
    let y = 0;
    for (const m of d.matchAll(/([MLHVQCZ])([^MLHVQCZ]*)/g)) {
      const nums = m[2].trim() === '' ? [] : m[2].trim().split(/[ ,]+/).map(Number);
      switch (m[1]) {
        case 'H':
          x = nums[0];
          out.push(`L ${pt(x, y)}`);
          break;
        case 'V':
          y = nums[0];
          out.push(`L ${pt(x, y)}`);
          break;
        case 'Z':
          out.push('Z');
          break;
        default: {
          const pts = [];
          for (let i = 0; i < nums.length; i += 2) pts.push(pt(nums[i], nums[i + 1]));
          out.push(`${m[1]} ${pts.join(' ')}`);
          x = nums[nums.length - 2];
          y = nums[nums.length - 1];
        }
      }
    }
    return out.join(' ');
  };
  // A label written before its text: whether it is centered or starts where it is drawn
  // about, that point, its size, the size of its type and its lines, each as its spans' faces
  // and texts.
  const label = (e) => {
    const l = readLabel(e);
    const text = l.lines.map((line) => line.map((s) => s.style + ':' + s.text).join('+')).join(' | ');
    return `${l.align === 'start' ? 'l' : 't'} ${pt(l.x, l.y)} ${n(l.width)},${n(l.height)} ${n(l.size)} | ${text}`;
  };
  let pending;
  const links = [];

  const tags = /<(\/?)([a-zA-Z-]+)\b([^>]*?)(\/?)>/g;
  for (const m of body.matchAll(tags)) {
    const [whole, close, name, , selfClosing] = m;
    if (close) {
      if (name === 'g') stack.pop();
      if (name === 'a') {
        // A link around nothing shows nothing, and a scene leaves it out.
        const at = links.pop();
        if (at === lines.length - 1) lines.pop();
      }
      continue;
    }
    switch (name) {
      case 'g': {
        const cls = attr(whole, 'class') ?? '';
        const t = TRANSLATE.exec(attr(whole, 'transform') ?? '');
        const up = top();
        const cap = attr(whole, 'stroke-linecap') ?? up.cap;
        const join = attr(whole, 'stroke-linejoin') ?? up.join;
        stack.push({ cls, move: t ? [Number(t[1]), Number(t[2])] : undefined, cap, join, fill: attr(whole, 'fill') ?? up.fill, stroke: attr(whole, 'stroke') ?? up.stroke });
        const id = attr(whole, 'data-id');
        const classes = cls.split(' ');
        if (classes[0] === 'pele-cluster') lines.push(`cluster ${classes[1].slice('pele-c4-'.length)} ${id}`);
        else if (classes[0] === 'pele-edge') lines.push(`edge ${id} ${top().fill ?? '-'} ${top().stroke ?? '-'} ${cap} ${join}${css(whole)}`);
        else if (classes[0] === 'pele-edge-label') lines.push(`label ${id}`);
        else if (classes[0] === 'pele-node') {
          // The move of the node is its middle, and its box is about that, as the scene
          // holds it.
          const [w, h] = attr(whole, 'data-wh').split(' ').map(Number);
          lines.push(`node ${id} ${classes.slice(1).join(' ')} box ${String(move(0, 0) - w / 2)},${String(move(0, 1) - h / 2)},${w},${h}`);
        }
        break;
      }
      case 'a':
        lines.push(`link ${attr(whole, 'href')}`);
        links.push(lines.length - 1);
        break;
      case 'title': {
        const end = body.indexOf('</title>', m.index);
        lines.push(`tip ${unescape(body.slice(m.index + whole.length, end))}`);
        break;
      }
      case 'rect':
        if (within('pele-cluster')) {
          lines.push(`rect ${box(whole)}${(attr(whole, 'style') ?? '').includes('rx:') ? ' r' : ''} ${paint(whole)}${css(whole)}`);
        } else if (within('pele-edge-label')) {
          lines.push(`patch ${box(whole)} ${n(attr(whole, 'rx'))} ${paint(whole)}`);
        } else if (within('pele-node')) {
          // A box with the theme's corners, or one of the outline's pieces.
          if ((attr(whole, 'style') ?? '').includes('rx:')) lines.push(`part rect ${box(whole)} ${paint(whole)} ${top().cap} ${top().join}${css(whole)}`);
          else lines.push(`part shape ${paint(whole)} ${top().cap} ${top().join}${css(whole)}`);
        } else lines.push('rect outside');
        break;
      case 'circle':
      case 'path':
        if (attr(whole, 'class') === 'pele-marker') break;
        if (within('pele-node')) lines.push(`part shape ${paint(whole)} ${top().cap} ${top().join}${css(whole)}`);
        else if (within('pele-edge')) lines.push(`path ${path(attr(whole, 'd'))}`);
        else lines.push(`${name} outside`);
        break;
      case 'limu-marker': {
        const k = readMarker(whole);
        lines.push(`m ${k.type} ${pt(k.x, k.y)} ${n(k.dx)},${n(k.dy)}`);
        break;
      }
      case 'limu-label':
        pending = label(whole);
        break;
      case 'text':
        // Each label is read from what was written before it; its first text gives its paint.
        if (pending !== undefined) {
          const cls = attr(whole, 'class');
          lines.push(`${pending} ${cls === 'pele-title' ? 'title ' : ''}${attr(whole, 'fill') ?? '-'}${css(whole)}`);
          pending = undefined;
        }
        break;
      case 'tspan':
      case 'svg':
        break;
      default:
        if (!selfClosing) lines.push(`unknown ${name}`);
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
    globalThis.limuExactNumbers = true;
    try {
      r = render(source, { measurer: metricsMeasurer, maxWidth, fontSize, outputLimit: Infinity });
    } catch (e) {
      if (!(e instanceof PeleError)) throw e;
      // An error of the drawing, after the text was read.
      if (e.code !== 'semantic') return false;
      out.push({ src: source, maxWidth, fontSize, error: peleError(e) });
      return true;
    } finally {
      globalThis.limuExactNumbers = false;
    }
    if (r.type !== 'c4') return false;
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
    // Every third one again in a narrow host, which lays out tightly what does not fit, and
    // every seventh in another size of type.
    if (drawn % 3 === 0) draw(source, [200, 360, 480, 640][drawn % 4]);
    if (drawn % 7 === 0) draw(source, undefined, [13, 20][drawn % 2]);
    drawn++;
  };
  corpus(pele).forEach(each);
  const drawable = drawableDiagrams(random(41));
  for (let i = 0; i < 2200; i++) each(drawable());
  const statements = statementDiagrams(random(42));
  for (let i = 0; i < 600; i++) each(statements());
  const usable = [...seen].filter((s) => s.length <= 3000);
  const next = mutator(usable, FRAGMENTS, random(43));
  for (let tries = 0; drawn < 3500 && tries < 20000; tries++) {
    const s = next();
    if (s.length <= 3000) each(s);
  }
  // The test reads the style Pele writes, css= and all that follows it on the line, as limu
  // reads declarations, so the lines with one are kept in full.
  const keep = (line) => {
    const at = line.indexOf(' css=');
    return at >= 0 && at + ' css='.length < line.length;
  };
  write('diagram/c4/testdata/pele/render.json', { cases: out }, { keep });
}
