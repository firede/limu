// Packet diagrams drawn end to end: what Pele's render draws, read back out of its SVG as one
// line per thing drawn, in the order drawn. limu's test writes the same lines from its scene.
//
// Pele draws a packet with no groups that move what is in them, so its numbers are where the
// scene has them, and are compared as Pele writes them, to two decimals.
import { lone, mutator } from './parse.mjs';
import { FRAGMENTS, corpus, randomDiagrams } from './packet.mjs';

// Pele's escapes in attribute values and text, undone.
const unescape = (s) => s.replace(/&(amp|lt|gt|quot);/g, (_, e) => ({ amp: '&', lt: '<', gt: '>', quot: '"' })[e]);
const attr = (s, name) => {
  const m = new RegExp(`\\s${name}="([^"]*)"`).exec(s);
  return m ? unescape(m[1]) : undefined;
};

// The lines of text in s: each <text> or <tspan> that says where it is, with the size of its
// type (that of the <text> it is in, or size), its anchor, and what it writes; a <tspan> that goes on
// the line before it is written with a + and what it writes.
function texts(s, size, anchor, prefix, lines) {
  let current = size;
  for (const m of s.matchAll(/<(text|tspan)\b([^>]*)>([^<]*)/g)) {
    const [, tag, attrs, content] = m;
    if (tag === 'text') current = attr(attrs, 'font-size') ?? size;
    const x = attr(attrs, 'x');
    const y = attr(attrs, 'y');
    if (x === undefined) {
      if (tag === 'tspan') lines.push(`${prefix}+ ${unescape(content)}`);
      continue;
    }
    lines.push(`${prefix}t ${x},${y} ${current} ${attr(attrs, 'text-anchor') ?? anchor} ${unescape(content)}`);
  }
}

export function readSvg(svg) {
  const lines = [];
  const head = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  lines.push(`size ${attr(head, 'width')},${attr(head, 'height')}`);
  const fields = /<g class="pele-packet-fields" font-size="([^"]*)"[^>]*>/.exec(svg);
  if (fields) {
    for (const m of svg.matchAll(/<g class="pele-node" data-id="([^"]*)">([\s\S]*?)<\/g>/g)) {
      const rect = /<rect [^>]*>/.exec(m[2])[0];
      lines.push(
        `field ${unescape(m[1])} ${attr(rect, 'x')},${attr(rect, 'y')},${attr(rect, 'width')},${attr(rect, 'height')}` +
          ` ${attr(rect, 'rx') || (attr(rect, 'style') ?? '').includes('rx:') ? 'r' : '-'} ${attr(rect, 'fill')} ${attr(rect, 'stroke')}`
      );
      texts(m[2], fields[1], 'middle', '', lines);
    }
  }
  const bits = /<g class="pele-packet-bits" font-size="([^"]*)" fill="([^"]*)">([\s\S]*?)<\/g>/.exec(svg);
  if (bits) texts(bits[3], `${bits[1]} ${bits[2]}`, 'start', 'bit ', lines);
  const title = svg.indexOf('<text class="pele-title"');
  if (title >= 0) texts(svg.slice(title, svg.indexOf('</text>', title)), '-', 'middle', 'title ', lines);
  return lines;
}

export default async function ({ pele, src, write, random }) {
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
    if (r.type !== 'packet') return false;
    out.push({ src: source, maxWidth, fontSize, lines: readSvg(r.svg) });
    return true;
  };
  const NARROW = [360, 320, 280, 240, 200, 160, 120, 100, 80, 600, 700];
  const SIZES = [13, 20, 11, 24];
  const seen = new Set();
  let drawn = 0;
  const each = (source) => {
    // A Go string cannot hold a surrogate with no partner.
    if (seen.has(source) || lone(source)) return;
    seen.add(source);
    if (!draw(source)) return;
    // Every other one again in a narrow host, which lays a packet out down when it does not fit, and
    // every fifth in another size of type.
    if (drawn % 2 === 0) draw(source, NARROW[drawn % NARROW.length]);
    if (drawn % 5 === 0) draw(source, drawn % 10 === 0 ? undefined : NARROW[(drawn / 5) % NARROW.length], SIZES[(drawn / 5) % SIZES.length]);
    drawn++;
  };
  // The largest of Pele's cases make lines by the ten thousand; they are drawn once, across.
  const base = corpus(pele).filter((s) => s.length <= 3000 && !/9{9}|\+20000|\+1000000000|bitsPerRow: (1|\.inf|5000)\n/.test(s));
  base.forEach(each);
  const diagrams = randomDiagrams(random(31));
  for (let i = 0; i < 1500; i++) each(diagrams());
  const usable = [...seen].filter((s) => s.length <= 3000);
  const next = mutator(usable, FRAGMENTS.filter((f) => !/9{9}/.test(f)), random(32));
  while (drawn < 2500) {
    const s = next();
    if (s.length <= 3000 && !/9{6}|\+[1-9]\d{4}/.test(s)) each(s);
  }
  write('diagram/packet/testdata/pele/render.json', { cases: out });
}
