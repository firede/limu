// Venn diagrams drawn end to end: what Pele's render draws, read back out of its SVG as one line
// per thing drawn, in the order drawn. limu's test writes the same lines from its scene.
//
// Pele draws a venn diagram with no groups that move what is in them, so its numbers are where
// the scene has them. They are drawn here written out in full, as the oracle's hooks have num()
// write them while limuExactNumbers is set, and compared whole: a difference in the last bit is
// a difference in the port. The outline of an overlap is written as Pele writes its path, arcs
// with their radii and flags, for the test to draw again as limu draws an arc. The styles an
// author gives are written as Pele writes them on each circle, path and text, for the test to
// read as limu reads them.
import { lone } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { unescape, attr, n, pt, stroke } from '../lib/svg.mjs';
import { corpus, drawableDiagrams, fragments, hostile } from './venn.mjs';

// A stroke's ends or corners as a browser takes them on an element, as lib/svg.mjs's stroke
// settles them; no group of a venn diagram sets either.
const linecap = (e) => stroke('stroke-linecap', [e]);
const linejoin = (e) => stroke('stroke-linejoin', [e]);
const css = (e) => encodeURIComponent(attr(e, 'style') ?? '');

// A path's d, its commands apart and its numbers written out.
function path(d) {
  const out = [];
  for (const m of d.matchAll(/([MAL])([^MALZ]*)|Z/g)) {
    if (m[0] === 'Z') {
      out.push('Z');
      continue;
    }
    const nums = m[2].trim().split(/[ ,]+/).map(n);
    out.push(`${m[1]} ${nums.join(' ')}`);
  }
  return out.join(' ');
}

// A text: what it is, its id, where its first line is, its fill, size, weight and style, what
// of the author's style it has, and the text of each line, a span with a weight or style of
// its own marked [b] or [i].
function text(kind, id, s, fontSize) {
  const open = /^<text\b([^>]*)>/.exec(s);
  const attrs = open[1];
  const body = s.slice(open[0].length);
  let at = / x="([^"]*)" y="([^"]*)"/.exec(attrs);
  const content = [];
  if (body.includes('<tspan')) {
    at = undefined;
    for (const t of body.matchAll(/<tspan([^>]*)>([^<]*)<\/tspan>/g)) {
      const x = attr(t[1], 'x');
      if (x !== undefined) {
        if (!at) at = [, x, attr(t[1], 'y')];
        content.push('');
      }
      const marks = (attr(t[1], 'font-weight') ? 'b' : '') + (attr(t[1], 'font-style') ? 'i' : '');
      content[content.length - 1] += (marks ? `[${marks}]` : '') + unescape(t[2]);
    }
  } else {
    content.push(unescape(body));
  }
  const fill = attr(attrs, 'fill') ?? 'var(--_fg)';
  const size = attr(attrs, 'font-size') ?? String(fontSize ?? 16);
  return `${kind} ${id} t ${pt(at[1], at[2])} ${fill} ${n(size)} ${attr(attrs, 'font-weight') ?? '-'} ${attr(attrs, 'font-style') ?? '-'} css=${css(attrs)} | ${content.join(' | ')}`;
}

export function readSvg(svg, fontSize) {
  const lines = [];
  const head = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  lines.push(`size ${pt(attr(head, 'width'), attr(head, 'height'))}`);
  const title = svg.indexOf('<text class="pele-title"');
  if (title >= 0) lines.push(text('title', '-', svg.slice(title, svg.indexOf('</text>', title)), fontSize));
  for (const m of svg.matchAll(/<circle class="pele-venn-set"[^>]*>/g)) {
    const e = m[0];
    lines.push(
      `set ${attr(e, 'data-id')} ${pt(attr(e, 'cx'), attr(e, 'cy'))} ${n(attr(e, 'r'))} ${attr(e, 'fill')} ${attr(e, 'fill-opacity')} ` +
        `${attr(e, 'stroke')} ${linecap(e)} ${linejoin(e)} css=${css(e)}`
    );
  }
  for (const m of svg.matchAll(/<path class="pele-venn-overlap"[^>]*>/g)) {
    const e = m[0];
    lines.push(`overlap ${attr(e, 'data-id')} ${attr(e, 'fill')} ${linecap(e)} ${linejoin(e)} css=${css(e)} | ${path(attr(e, 'd'))}`);
  }
  for (const m of svg.matchAll(/<g class="pele-venn-region" data-id="([^"]*)">([\s\S]*?)<\/g>/g)) {
    const region = unescape(m[1]);
    for (const t of m[2].matchAll(/<text\b[\s\S]*?<\/text>/g)) {
      const open = /^<text\b[^>]*>/.exec(t[0])[0];
      const kind = attr(open, 'class') === 'pele-venn-text' ? 'text' : 'label';
      lines.push(text(kind, kind === 'text' ? attr(open, 'data-id') : region, t[0].slice(0, -'</text>'.length), fontSize));
    }
  }
  return lines;
}

export default async function ({ pele, src, write, random }) {
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
    if (r.type !== 'venn') return false;
    out.push({ src: source, maxWidth, fontSize, lines: readSvg(r.svg, fontSize) });
    return true;
  };
  // Pele draws a venn diagram at the size it is, whatever the host's width; a few are drawn in
  // a narrow host to show it.
  const NARROW = [320, 200, 600];
  const SIZES = [13, 20, 11, 24, 13.5, 9];
  const seen = new Set();
  let drawn = 0;
  const each = (source) => {
    // A Go string cannot hold a surrogate with no partner.
    if (seen.has(source) || lone(source) || source.length > 4000) return;
    seen.add(source);
    if (!draw(source)) return;
    // Every fifth one again in another size of type, and every fiftieth in a narrow host.
    if (drawn % 5 === 0) draw(source, undefined, SIZES[(drawn / 5) % SIZES.length]);
    if (drawn % 50 === 0) draw(source, NARROW[(drawn / 50) % NARROW.length]);
    drawn++;
  };
  corpus(pele).forEach(each);
  hostile(pele).forEach(each);
  const diagrams = drawableDiagrams(random(31));
  for (let i = 0; i < 3000; i++) each(diagrams());
  // Larger ones, as Pele's stress tests draw them, made smaller.
  const repeat = (count, line) => Array.from({ length: count }, (_, i) => line(i)).join('');
  const V = 'venn-beta\n';
  for (const s of [
    V + repeat(100, (i) => `set s${i}\n`),
    V + repeat(80, (i) => `set s${i}:${(i % 7) + 1}\n`) + repeat(79, (i) => `union s${i},s${i + 1}\n`),
    V + repeat(20, (i) => `set s${i}\n`) + 'union ' + repeat(19, (i) => `s${i},`) + 's19["all"]\n',
    V + repeat(12, (i) => `set s${i}\n`) + repeat(12, (i) => `union s${i},s${(i + 5) % 12}\n`),
    V + 'set A\nset B\nset C\nunion A,B,C\n' + '  text "t t"["l"]\n'.repeat(30),
    V + 'set A\n' + '  text t\n'.repeat(60),
    V + `set A:0\nset B:-1000000000\nset C:${'9'.repeat(400)}\nset D:.${'0'.repeat(300)}1\nunion A,B:${'9'.repeat(400)}\nunion C,D:-5\nunion A,B,C,D["x"]:0\n`,
    V + 'set A:1e300\nset B:1e300\nunion A,B:1e299\n',
    V + 'set A:1e-300\nset B\nunion A,B\n',
    V + 'set A\n' + 'union A,A,A,A["x"]\n'.repeat(30),
  ]) {
    each(s);
  }
  const usable = [...seen].filter((s) => s.length <= 3000);
  const next = mutator(usable, fragments(pele), random(32));
  while (drawn < 5000) each(next());
  write('diagram/venn/testdata/pele/render.json', { cases: out });
}
