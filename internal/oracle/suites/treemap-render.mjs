// Treemaps drawn end to end: what Pele's render draws, read back out of its SVG as one line
// per thing drawn, in the order drawn. limu's test writes the same lines from its scene.
//
// Pele draws a treemap with no groups that move what is in them, so its numbers are where the
// scene has them. They are drawn here written out in full, as the oracle's hooks have num()
// write them while limuExactNumbers is set, and compared whole: a difference in the last bit
// is a difference in the port. The styles an author's classes give are written as Pele writes
// them on each shape and text, for the test to read as limu reads them.
import { lone } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { unescape, attr, stroke } from '../lib/svg.mjs';
import { FRAGMENTS, corpus, randomDiagrams } from './treemap.mjs';

// A stroke's ends or corners as a browser takes them on an element, as lib/svg.mjs's stroke
// settles them; no group of a treemap sets either.
const linecap = (e) => stroke('stroke-linecap', [e]);
const linejoin = (e) => stroke('stroke-linejoin', [e]);
// The style Pele writes on an element, without the corner radius every rounded box takes.
const css = (e) => encodeURIComponent((attr(e, 'style') ?? '').replace(/^rx:var\(--_r\);?/, ''));

// The lines of text in s: each <text> or <tspan> that says where it is, with the size of its
// type, its anchor, its fill and weight, what of the author's style it has, and what it writes;
// a <tspan> that goes on the line before it is written with a + and what it writes. What a text
// does not say it takes from up, the group it is in.
function texts(s, up, prefix, lines) {
  let text = up;
  for (const m of s.matchAll(/<(text|tspan)\b([^>]*)>([^<]*)/g)) {
    const [, tag, attrs, content] = m;
    if (tag === 'text') {
      text = {
        size: attr(attrs, 'font-size') ?? up.size,
        anchor: attr(attrs, 'text-anchor') ?? up.anchor,
        fill: attr(attrs, 'fill') ?? up.fill,
        weight: attr(attrs, 'font-weight') ?? '-',
        css: css(attrs),
      };
    }
    const x = attr(attrs, 'x');
    const y = attr(attrs, 'y');
    if (x === undefined) {
      if (tag === 'tspan') lines.push(`${prefix}+ ${unescape(content)}`);
      continue;
    }
    lines.push(`${prefix}t ${x},${y} ${text.size} ${text.anchor} ${text.fill} ${text.weight} css=${text.css} ${unescape(content)}`);
  }
}

export function readSvg(svg) {
  const lines = [];
  const head = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  lines.push(`size ${attr(head, 'width')},${attr(head, 'height')}`);
  const root = { size: '-', anchor: 'start', fill: 'var(--_fg)' };
  const title = svg.indexOf('<text class="pele-title"');
  if (title >= 0) texts(svg.slice(title, svg.indexOf('</text>', title)), root, 'title ', lines);
  const groups = [
    [/<g class="pele-sections" font-size="([^"]*)">/, (m) => ({ ...root, size: m[1] })],
    [/<g class="pele-leaves" text-anchor="middle" fill="var\(--_bg\)">/, () => ({ ...root, anchor: 'middle', fill: 'var(--_bg)' })],
  ];
  for (const [re, inside] of groups) {
    const g = re.exec(svg);
    if (!g) continue;
    const up = inside(g);
    const from = g.index + g[0].length;
    for (const m of svg.slice(from).matchAll(/<g class="pele-node pele-(section|leaf)([^"]*)" data-id="([^"]*)">([\s\S]*?)<\/g>|<\/g>/g)) {
      // The group of sections or leaves ends at the first </g> that ends no node.
      if (m[1] === undefined) break;
      const rect = /<rect [^>]*>/.exec(m[4])[0];
      const box = ['x', 'y', 'width', 'height'].map((k) => attr(rect, k)).join(',');
      const rounded = (attr(rect, 'style') ?? '').startsWith('rx:var(--_r)') ? 'r' : '-';
      lines.push(
        `${m[1]} ${unescape(m[3])} [${m[2].trim()}] ${box} ${rounded} ${attr(rect, 'fill') ?? '-'} ${attr(rect, 'fill-opacity') ?? '-'}` +
          ` ${attr(rect, 'stroke') ?? '-'} ${linecap(rect)} ${linejoin(rect)} css=${css(rect)}`
      );
      texts(m[4].slice(m[4].indexOf(rect) + rect.length), up, '', lines);
    }
  }
  return lines;
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
    if (r.type !== 'treemap') return false;
    out.push({ src: source, maxWidth, fontSize, padding, lines: readSvg(r.svg) });
    return true;
  };
  const NARROW = [360, 320, 280, 240, 200, 160, 120, 600, 500, 90];
  const SIZES = [13, 20, 11, 24, 13.5, 9];
  const PADS = [0, 20, 2.5];
  const seen = new Set();
  let drawn = 0;
  const each = (source) => {
    // A Go string cannot hold a surrogate with no partner.
    if (seen.has(source) || lone(source) || source.length > 4000) return;
    seen.add(source);
    if (!draw(source)) return;
    // Every third one again in a narrow host, which caps the treemap's width and wraps its
    // title, and every fifth in another size of type.
    if (drawn % 3 === 0) draw(source, NARROW[(drawn / 3) % NARROW.length]);
    if (drawn % 5 === 0) draw(source, drawn % 10 === 0 ? undefined : NARROW[(drawn / 5) % NARROW.length], SIZES[(drawn / 5) % SIZES.length]);
    // Every twentieth with the host's padding, which takes the place of diagramPadding, some
    // in a narrow host, where it is taken from the treemap's width.
    if (drawn % 20 === 0) draw(source, drawn % 40 === 0 ? undefined : NARROW[(drawn / 20) % NARROW.length], undefined, PADS[(drawn / 20) % PADS.length]);
    drawn++;
  };
  corpus(pele).forEach(each);
  const diagrams = randomDiagrams(random(31));
  for (let i = 0; i < 3500; i++) each(diagrams());
  // Larger ones, as Pele's stress tests draw them, made smaller: many leaves of equal and of
  // very unequal value, many sections, and deep nesting.
  const repeat = (count, line) => Array.from({ length: count }, (_, i) => line(i)).join('');
  for (const s of [
    'treemap\n' + repeat(150, (i) => `"l${i}": ${i + 1}\n`),
    'treemap\n"s"\n' + repeat(150, () => ' "l": 1\n'),
    'treemap\n' + repeat(150, (i) => `"l": ${i % 2 ? '0.0001' : '99999999'}\n`),
    'treemap\n' + repeat(80, (i) => `"s${i}"\n  "l": ${i + 1}\n`),
    'treemap\n' + repeat(30, (i) => ' '.repeat(i) + `"s${i}"\n`) + ' '.repeat(30) + '"leaf": 1\n',
    'treemap\n' + repeat(30, (i) => ' '.repeat(i) + `"s${i}"\n` + ' '.repeat(i + 1) + `"l${i}": ${30 - i}\n`),
  ]) {
    each(s);
  }
  const usable = [...seen].filter((s) => s.length <= 3000);
  const next = mutator(usable, FRAGMENTS, random(32));
  while (drawn < 5000) each(next());
  write('diagram/treemap/testdata/pele/render.json', { cases: out });
}
