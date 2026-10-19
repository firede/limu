// Info diagrams drawn end to end: what Pele's render draws, read back out of its SVG as one
// line per thing drawn, in the order drawn. limu's test writes the same lines from its scene.
//
// Pele draws its own name and version, where limu draws its own. The oracle's hooks have Pele
// draw the text limuInfoText gives in their place, each case with one of the texts limu
// draws, so that the two draw the same. Pele's SVG is drawn with its numbers written out in
// full, and each label as the point it is centered on and all that is in it.
import { lone, mutator } from './parse.mjs';
import { FRAGMENTS, corpus, hostileCorpus, randomDiagrams } from './info.mjs';

// What limu draws for its name and version: built from its own sources, and built as a
// module of a program at a version, a prerelease and a pseudo-version.
export const TEXTS = ['limu (devel)', 'limu v0.1.0', 'limu v1.12.0-rc.1+meta', 'limu v0.0.0-20261019120000-abcdef123456'];

const unescape = (s) => s.replace(/&(amp|lt|gt|quot);/g, (_, e) => ({ amp: '&', lt: '<', gt: '>', quot: '"' })[e]);
const attr = (s, name) => {
  const m = new RegExp(`\\s${name}="([^"]*)"`).exec(s);
  return m ? unescape(m[1]) : undefined;
};
const n = (v) => String(v);

// The lines of an info diagram's SVG, drawn with its numbers in full.
export function readSvg(svg) {
  const lines = [];
  const head = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  lines.push(`size ${n(Number(attr(head, 'width')))},${n(Number(attr(head, 'height')))}`);
  let body = svg.slice(svg.indexOf('>') + 1);
  const title = /^<title id="[^"]*">([^<]*)<\/title>/.exec(body);
  if (title) {
    lines.push(`acc-title ${unescape(title[1])}`);
    body = body.slice(title[0].length);
  }
  const desc = /^<desc id="[^"]*">([^<]*)<\/desc>/.exec(body);
  if (desc) {
    lines.push(`acc-desc ${unescape(desc[1])}`);
    body = body.slice(desc[0].length);
  }
  for (const m of body.matchAll(/<(limu-label|[a-z]+)\b[^>]*>/g)) {
    const e = m[0];
    if (m[1] !== 'limu-label') {
      lines.push(`element ${m[1]}`);
      continue;
    }
    const attrs = decodeURIComponent(attr(e, 'attrs'));
    const cls = /class="([^"]*)"/.exec(attrs)?.[1];
    const fill = /fill="([^"]*)"/.exec(attrs)?.[1] ?? '-';
    const d = JSON.parse(decodeURIComponent(attr(e, 'data')));
    const spans = d.lines.map((line) => line.map(([icon, text, style, width]) => (icon !== null ? `i:${icon}:${n(width)}` : `t:${style}:${text}:${n(width)}`)).join('|'));
    lines.push(
      `label ${cls} ${fill} ${n(Number(attr(e, 'x')))},${n(Number(attr(e, 'y')))} ${n(d.size)},${n(d.lineHeight)},${n(d.width)},${n(d.height)} ` +
        `| ${d.widths.map(n).join(',')} | ${spans.join(' / ')}`
    );
  }
  return lines;
}

export default async function ({ src, pele, write, random }) {
  const { render, PeleError } = await src('index.ts');
  const { metricsMeasurer } = await src('text/measurer.ts');

  const out = [];
  const draw = (source, text, fontSize, icons) => {
    let r;
    globalThis.limuExactNumbers = true;
    globalThis.limuInfoText = text;
    try {
      r = render(source, { measurer: metricsMeasurer, fontSize, icons: icons ? () => '<path d="M0,0"/>' : undefined });
    } catch (e) {
      if (!(e instanceof PeleError)) throw e;
      return false;
    } finally {
      globalThis.limuExactNumbers = false;
      globalThis.limuInfoText = undefined;
    }
    if (r.type !== 'info') return false;
    out.push({ src: source, text, fontSize, icons: icons || undefined, lines: readSvg(r.svg) });
    return true;
  };
  const seen = new Set();
  let drawn = 0;
  const each = (source) => {
    if (seen.has(source) || lone(source) || source.length > 4000) return;
    seen.add(source);
    if (!draw(source, TEXTS[drawn % TEXTS.length], undefined, drawn % 5 === 2)) return;
    if (drawn % 3 === 0) draw(source, TEXTS[(drawn + 1) % TEXTS.length], drawn % 2 === 0 ? 13 : 21.5);
    drawn++;
  };
  corpus(pele).forEach(each);
  hostileCorpus(pele).forEach(each);
  const diagrams = randomDiagrams(random(81));
  for (let i = 0; i < 300; i++) each(diagrams());
  const next = mutator([...seen], FRAGMENTS, random(82));
  for (let i = 0; drawn < 600 && i < 20000; i++) each(next());
  write('diagram/info/testdata/pele/render.json', { cases: out });
}
