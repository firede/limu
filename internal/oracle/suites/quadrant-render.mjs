// Quadrant charts drawn end to end: what Pele's render draws, read back out of its SVG as one
// line per thing drawn, in the order drawn. limu's test writes the same lines from its scene.
//
// Pele draws a quadrant chart with no groups that move what is in them but the turn of each
// label of the y axis, which the scene holds as the label's turn about the same point, so its
// numbers are where the scene has them. They are drawn here written out in full, as the
// oracle's hooks have num() write them while limuExactNumbers is set, and compared whole.
import { lone, mutator } from './parse.mjs';
import { capJoin } from './flowchart-render.mjs';
import { FRAGMENTS, corpus, randomDiagrams } from './quadrant.mjs';

// Pele's escapes in attribute values and text, undone.
const unescape = (s) => s.replace(/&(amp|lt|gt|quot);/g, (_, e) => ({ amp: '&', lt: '<', gt: '>', quot: '"' })[e]);
const attr = (s, name) => {
  const m = new RegExp(`\\s${name}="([^"]*)"`).exec(s);
  return m ? unescape(m[1]) : undefined;
};

// A color of a point's style, as limu's test writes one: a hex code of six digits, in lower
// case.
const color = (c) => {
  if (!/^#[\da-f]+$/i.test(c)) return c;
  const d = c.slice(1).toLowerCase();
  return '#' + (d.length === 3 ? [...d].map((x) => x + x).join('') : d);
};
// The value of a property in a style attribute, the last one given.
const styleOf = (e, prop) => {
  let out;
  for (const decl of (attr(e, 'style') ?? '').split(';')) {
    const at = decl.indexOf(':');
    if (at >= 0 && decl.slice(0, at).trim() === prop) out = decl.slice(at + 1).trim();
  }
  return out;
};

// The lines of the SVG of a quadrant chart. Each <text> or <tspan> that says where it is
// starts a line of text, with the size and paint of its type and its emphasis, b for bold and
// i for italic, as the groups it is in and its own attributes give them; a <tspan> that goes
// on the line before it is written with a + and its emphasis.
export function readSvg(svg) {
  const lines = [];
  const root = /<svg\b[^>]*>/.exec(svg)[0];
  lines.push(`size ${attr(root, 'width')},${attr(root, 'height')}`);
  const rootSize = /font:([\d.]+)px/.exec(attr(root, 'style'))[1];
  // The opening tags of the groups the walk is in, the root first.
  const stack = [root];
  const inherited = (name, fallback) => {
    for (let k = stack.length - 1; k >= 0; k--) {
      const v = attr(stack[k], name);
      if (v !== undefined) return v;
    }
    return fallback;
  };
  let face = '';
  let textFill = '';
  let size = '';
  for (const m of svg.slice(root.length).matchAll(/<(\/?)(g|text|tspan|rect|circle)\b([^>]*)>([^<]*)/g)) {
    const [, close, tag, rest, content] = m;
    const open = `<${tag}${rest}>`;
    if (close) {
      if (tag === 'g') stack.pop();
      continue;
    }
    switch (tag) {
      case 'g': {
        stack.push(open);
        const cls = attr(open, 'class');
        if (cls === 'pele-quadrant') lines.push(`quadrant ${attr(open, 'data-id')}`);
        const turn = /^translate\(([^,]*),([^)]*)\) rotate\(([^)]*)\)$/.exec(attr(open, 'transform') ?? '');
        if (turn) lines.push(`turn ${turn[1]},${turn[2]} ${turn[3]}`);
        break;
      }
      case 'rect': {
        const rounded = attr(open, 'rx') !== undefined || (attr(open, 'style') ?? '').includes('rx:') ? 'r' : '-';
        lines.push(
          `rect ${attr(open, 'x')},${attr(open, 'y')},${attr(open, 'width')},${attr(open, 'height')} ${rounded}` +
            ` ${attr(open, 'fill') ?? inherited('fill')} ${attr(open, 'stroke') ?? inherited('stroke', 'none')} ${capJoin([...stack, open])}`
        );
        break;
      }
      case 'circle': {
        const classes = (attr(open, 'class') ?? '').split(' ').filter((c) => c !== 'pele-point').join(' ');
        const fill = styleOf(open, 'fill') ?? attr(open, 'fill') ?? inherited('fill');
        const stroke = styleOf(open, 'stroke') ?? attr(open, 'stroke') ?? inherited('stroke', 'none');
        const width = styleOf(open, 'stroke-width');
        lines.push(
          `point ${attr(open, 'data-id')} [${classes}] ${attr(open, 'cx')},${attr(open, 'cy')} ${attr(open, 'r')}` +
            ` ${color(fill)} ${color(stroke)} ${width === undefined ? '-' : String(parseFloat(width))} ${capJoin([...stack, open])}`
        );
        break;
      }
      case 'text': {
        const cls = attr(open, 'class');
        face = (attr(open, 'font-weight') !== undefined ? 'b' : '') + (attr(open, 'font-style') === 'italic' ? 'i' : '');
        textFill = attr(open, 'fill') ?? inherited('fill');
        size = attr(open, 'font-size') ?? inherited('font-size', rootSize);
        const prefix = cls === 'pele-title' ? 'title ' : '';
        if (attr(open, 'x') !== undefined) {
          lines.push(`${prefix}t ${attr(open, 'x')},${attr(open, 'y')} ${size} ${textFill} ${face || '-'} ${unescape(content)}`);
        } else if (prefix) {
          lines.push('title');
        }
        break;
      }
      case 'tspan': {
        const bold = face.includes('b') || attr(open, 'font-weight') !== undefined;
        const italic = face.includes('i') || attr(open, 'font-style') === 'italic';
        const own = (bold ? 'b' : '') + (italic ? 'i' : '');
        if (attr(open, 'x') !== undefined) {
          lines.push(`t ${attr(open, 'x')},${attr(open, 'y')} ${size} ${textFill} ${own || '-'} ${unescape(content)}`);
        } else {
          lines.push(`+ ${own || '-'} ${unescape(content)}`);
        }
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
      return false;
    } finally {
      globalThis.limuExactNumbers = false;
    }
    if (r.type !== 'quadrantChart') return false;
    out.push({ src: source, maxWidth, fontSize, lines: readSvg(r.svg) });
    return true;
  };
  const NARROW = [400, 320, 280, 240, 200, 120, 600, 450];
  const SIZES = [13, 20, 11, 24];
  const seen = new Set();
  let drawn = 0;
  const each = (source) => {
    // A Go string cannot hold a surrogate with no partner.
    if (seen.has(source) || lone(source)) return;
    seen.add(source);
    if (!draw(source)) return;
    // Every third one again in a narrow host, which draws the chart narrower, and every
    // seventh in another size of type.
    if (drawn % 3 === 0) draw(source, NARROW[(drawn / 3) % NARROW.length]);
    if (drawn % 7 === 0) draw(source, drawn % 14 === 0 ? undefined : NARROW[(drawn / 7) % NARROW.length], SIZES[(drawn / 7) % SIZES.length]);
    drawn++;
  };
  corpus(pele).forEach(each);
  // Crowds of points: at one place, where no label finds a free one, and more than the
  // labels of which look for a free place at all.
  const crowd = (count, at) => 'quadrantChart\n' + Array.from({ length: count }, (_, i) => `P${i}: [${at(i)}]\n`).join('');
  each(crowd(60, () => '0.5, 0.5'));
  each(crowd(300, (i) => `0.${(i * 37) % 100}, 0.${(i * 61) % 100}`));
  each(crowd(301, (i) => `0.${(i * 37) % 100}, 0.${(i * 61) % 100}`));
  each(crowd(320, (i) => `0.${(i * 13) % 100}, 0.${(i * 7) % 100}`));
  const diagrams = randomDiagrams(random(31));
  for (let i = 0; i < 2000; i++) each(diagrams());
  const usable = [...seen].filter((s) => s.length <= 3000);
  const next = mutator(usable, FRAGMENTS, random(32));
  while (drawn < 3200) {
    const s = next();
    if (s.length <= 3000) each(s);
  }
  write('diagram/quadrant/testdata/pele/render.json', { cases: out });
}
