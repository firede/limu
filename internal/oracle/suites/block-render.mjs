// Block diagrams drawn end to end: what Pele's render draws, read back out of its SVG as one
// line per thing drawn, in the order drawn, and the errors of the diagrams it refuses to draw.
// limu's test writes the same lines from its scene.
//
// A scene has no groups to move what is in them, so it holds where everything is on the page,
// where Pele writes the inside of a block about the block's middle. Pele's SVG is drawn here
// with its numbers written out in full, as the oracle's hooks have num() write them while
// limuExactNumbers is set, and the lines hold every number in full, as String() writes it, so
// that a difference in the last bit shows. A number inside a block is moved as limu moves it:
// the block's middle plus the number, in that order. While the flag is set the hooks also
// have each label of a block diagram write the point it is centered on, its size and its
// spans, and each block the size it is drawn at, which its shape is drawn from: the shapes
// themselves are drawShape's, which the draw suite checks.
import { lone } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { attr, TRANSLATE, n, pt, capJoin } from '../lib/svg.mjs';
import { readLabel } from '../lib/limu.mjs';
import { FRAGMENTS, corpus, randomDiagrams } from './block.mjs';

// The last opacity an element's style attribute gives a property, as a browser takes it over
// the element's attribute: a number, or a percentage, clamped to [0, 1].
function styleOpacity(tag, prop) {
  let v;
  for (const decl of (attr(tag, 'style') ?? '').split(';')) {
    const at = decl.indexOf(':');
    if (at < 0 || decl.slice(0, at).trim().toLowerCase() !== prop) continue;
    const raw = decl.slice(at + 1).trim();
    const m = /^([+-]?(?:\d+\.?\d*|\.\d+)(?:e[+-]?\d+)?)(%?)$/i.exec(raw);
    if (m) v = Math.max(0, Math.min(1, Number(m[1]) / (m[2] ? 100 : 1)));
  }
  return v;
}

// How a shape is painted, from its attributes and the opacities its style gives them:
// "fill[/opacity] stroke[/opacity]", "-" for what it does not give. A paint at opacity 0 draws
// nothing and is written none, and one at opacity 1 is written without it, as a scene holds
// them.
function paint(tag) {
  const one = (p, o) => {
    if (o === undefined || p === 'none' || p === '-') return p;
    const v = Number(o);
    return v <= 0 ? 'none' : v >= 1 ? p : `${p}/${n(v)}`;
  };
  const fill = one(attr(tag, 'fill') ?? '-', styleOpacity(tag, 'fill-opacity') ?? attr(tag, 'fill-opacity'));
  const stroke = one(attr(tag, 'stroke') ?? '-', styleOpacity(tag, 'stroke-opacity'));
  return `${fill} ${stroke}`;
}

// A path's d with every number in full, moved by dx, dy: M, L, Q and Z as they are, and A
// with its radii, turn and flags before its end.
function path(d, dx = 0, dy = 0) {
  const out = [];
  for (const m of d.matchAll(/([MLQAZ])([^MLQAZ]*)/g)) {
    const a = m[2].trim() === '' ? [] : m[2].trim().split(/[ ,]+/).map(Number);
    if (m[1] === 'A') {
      out.push(`A ${a.slice(0, 5).map(n).join(',')} ${pt(a[5] + dx, a[6] + dy)}`);
      continue;
    }
    const pts = [];
    for (let i = 0; i < a.length; i += 2) pts.push(pt(a[i] + dx, a[i + 1] + dy));
    out.push([m[1], ...pts].join(' '));
  }
  return out.join(' ');
}

// The line of a label as the hooks write it before its text: the point it is centered on,
// moved by dx, dy as limu moves it, its size, the fill of its text, and its spans.
function label(s, dx, dy) {
  const at = s.indexOf('<limu-label ');
  if (at < 0) return undefined;
  const l = readLabel(s.slice(at, s.indexOf('/>', at) + 2));
  const text = s.slice(s.indexOf('<text', at), s.indexOf('>', s.indexOf('<text', at)));
  const spans = l.lines
    .map((line) => line.map((p) => (p.icon !== undefined ? `i:${p.icon}:${n(p.width)}` : `${p.style}:${p.text}:${n(p.width)}`)).join(' '))
    .join(' | ');
  return `t ${dx === undefined ? pt(l.x, l.y) : pt(dx + l.x, dy + l.y)} ${pt(l.width, l.height)} ${attr(text, 'fill') ?? '-'} | ${spans}`;
}

const GROUP = /<g class="pele-(cluster|edge|edge-label|node)[ "]/g;
const SHAPE = /<(rect|circle|ellipse|path|polygon)\b[^>]*>/g;

// The lines of a block diagram's SVG, drawn with its numbers in full.
export function readSvg(svg) {
  const lines = [];
  const head = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  lines.push(`size ${pt(attr(head, 'width'), attr(head, 'height'))}`);
  const title = svg.indexOf('<text class="pele-title"');
  if (title >= 0) lines.push('title ' + label(svg.slice(svg.lastIndexOf('<limu-label ', title))));
  const edgesOpen = /<g class="pele-edges"[^>]*>/.exec(svg)?.[0];
  const starts = [...svg.matchAll(GROUP)];
  for (let k = 0; k < starts.length; k++) {
    const kind = starts[k][1];
    const s = svg.slice(starts[k].index, k + 1 < starts.length ? starts[k + 1].index : svg.length);
    const open = s.slice(0, s.indexOf('>'));
    const id = attr(open, 'data-id');
    const cls = attr(open, 'class').split(' ');
    switch (kind) {
      case 'cluster': {
        const rect = /<rect [^>]*>/.exec(s)[0];
        const rounded = attr(rect, 'rx') !== undefined || (attr(rect, 'style') ?? '').includes('rx:');
        lines.push(`cluster ${id} [${cls.slice(1).join(' ')}] ${pt(attr(rect, 'x'), attr(rect, 'y'))},${pt(attr(rect, 'width'), attr(rect, 'height'))}${rounded ? ' r' : ''} ${paint(rect)}`);
        lines.push(`pen ${capJoin([open, rect])}`);
        const l = label(s);
        if (l) lines.push(l);
        break;
      }
      case 'node': {
        const t = TRANSLATE.exec(attr(open, 'transform'));
        const tx = Number(t[1]);
        const ty = Number(t[2]);
        const [bw, bh] = attr(open, 'data-limu').split(' ').map(Number);
        const shape = cls[1].slice('pele-shape-'.length);
        lines.push(`node ${id} ${shape} [${cls.slice(2).join(' ')}] ${pt(tx - bw / 2, ty - bh / 2)},${pt(bw, bh)}`);
        const end = s.indexOf('<limu-label ');
        const shapes = [...(end < 0 ? s : s.slice(0, end)).matchAll(SHAPE)].map((m) => m[0]);
        for (const e of shapes) {
          lines.push(`part ${paint(e)}`);
          // The outline of any other shape is drawShape's, which the draw suite checks.
          if (shape === 'block-arrow' && e.startsWith('<polygon')) {
            const p = attr(e, 'points').split(/[ ,]+/).map(Number);
            const moved = [];
            for (let i = 0; i < p.length; i += 2) moved.push(pt(tx + p[i], ty + p[i + 1]));
            lines.push(`polygon ${moved.join(' ')}`);
          }
        }
        if (shapes.length > 0) lines.push(`pen ${capJoin([open, shapes[0]])}`);
        const l = label(s, tx, ty);
        if (l) lines.push(l);
        break;
      }
      case 'edge': {
        const p = /<path d="[^"]*"[^>]*>/.exec(s)[0];
        lines.push(`edge ${id} ${attr(p, 'stroke-dasharray') ?? '-'} ${attr(p, 'stroke-width') ?? '-'}`);
        lines.push(`path ${path(attr(p, 'd'))}`);
        lines.push(`pen ${capJoin([edgesOpen, open, p])}`);
        for (const m of s.matchAll(/<(path|circle) class="pele-marker"[^>]*>/g)) {
          const e = m[0];
          const shape = m[1] === 'path' ? path(attr(e, 'd')) : `o ${pt(attr(e, 'cx'), attr(e, 'cy'))} ${n(attr(e, 'r'))}`;
          lines.push(`m ${shape} ${attr(e, 'fill') ?? '-'} ${attr(e, 'stroke') ?? '-'}`);
        }
        break;
      }
      case 'edge-label': {
        const rect = /<rect [^>]*>/.exec(s)[0];
        const text = s.slice(s.indexOf('<text'));
        const m = / x="([^"]*)" y="([^"]*)"/.exec(text);
        lines.push(`elabel ${id} ${pt(attr(rect, 'x'), attr(rect, 'y'))},${pt(attr(rect, 'width'), attr(rect, 'height'))} ${attr(rect, 'fill')}`);
        lines.push(`t ${pt(m[1], m[2])}`);
        break;
      }
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
      // Text that does not parse is the parse suite's; what parses and is not drawn is here.
      if (e.code === 'semantic') out.push({ src: source, maxWidth, fontSize, error: e.message });
      return false;
    } finally {
      globalThis.limuExactNumbers = false;
    }
    if (r.type !== 'block') return false;
    out.push({ src: source, maxWidth, fontSize, lines: readSvg(r.svg) });
    return true;
  };
  const seen = new Set();
  let drawn = 0;
  const each = (source) => {
    // A Go string cannot hold a surrogate with no partner.
    if (seen.has(source) || lone(source) || source.length > 4000) return;
    seen.add(source);
    if (!draw(source)) return;
    // Every third one again in a narrow host, which draws it tightly where that is narrower,
    // and every seventh in another size of type.
    if (drawn % 3 === 0) draw(source, pickWidth(drawn));
    if (drawn % 7 === 0) draw(source, undefined, drawn % 2 ? 13 : 20);
    drawn++;
  };
  const pickWidth = (i) => [120, 200, 360][i % 3];
  corpus(pele).forEach(each);
  const diagrams = randomDiagrams(random(41));
  for (let i = 0; i < 2000; i++) each(diagrams());
  const usable = [...seen].filter((s) => s.length <= 3000);
  const next = mutator(usable, FRAGMENTS, random(42));
  for (let tries = 0; drawn < 4000 && tries < 40000; tries++) each(next());
  // A case whose blocks run to thousands, as space:9999 makes them, is left out: the fixture
  // would hold every one of them.
  const cases = out.filter((c) => JSON.stringify(c).length <= 40000);
  write('diagram/block/testdata/pele/render.json', { cases });
}
