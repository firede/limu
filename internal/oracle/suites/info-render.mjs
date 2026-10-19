// Info diagrams drawn end to end: what Pele's render draws, read back out of its SVG as one
// line per thing drawn, in the order drawn. limu's test writes the same lines from its scene.
//
// Pele draws its own name and version, where limu draws its own. The oracle's hooks have Pele
// draw the text limuInfoText gives in their place, each case with one of the texts limu
// draws, so that the two draw the same. Pele's SVG is drawn with its numbers written out in
// full, and each label as the point it is centered on and all that is in it, in place of
// Pele's drawing of it.
import { lone } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { attr, n, readHead } from '../lib/svg.mjs';
import { readLabel, withoutPele } from '../lib/limu.mjs';
import { FRAGMENTS, corpus, hostileCorpus, randomDiagrams } from './info.mjs';

// What limu draws for its name and version: built from its own sources, and built as a
// module of a program at a version, a prerelease and a pseudo-version.
export const TEXTS = ['limu (devel)', 'limu v0.1.0', 'limu v1.12.0-rc.1+meta', 'limu v0.0.0-20261019120000-abcdef123456'];

// The lines of an info diagram's SVG, drawn with its numbers in full.
export function readSvg(svg) {
  svg = withoutPele(svg);
  const lines = [];
  const { open: head, body, acc } = readHead(svg);
  lines.push(`size ${n(Number(attr(head, 'width')))},${n(Number(attr(head, 'height')))}`, ...acc);
  for (const m of body.matchAll(/<(limu-label|[a-z]+)\b[^>]*>/g)) {
    const e = m[0];
    if (m[1] !== 'limu-label') {
      lines.push(`element ${m[1]}`);
      continue;
    }
    const l = readLabel(e);
    const cls = /class="([^"]*)"/.exec(l.attrs)?.[1];
    const fill = /fill="([^"]*)"/.exec(l.attrs)?.[1] ?? '-';
    const spans = l.lines.map((line) => line.map((s) => (s.icon !== undefined ? `i:${s.icon}:${n(s.width)}` : `t:${s.style}:${s.text}:${n(s.width)}`)).join('|'));
    lines.push(
      `label ${cls} ${fill} ${n(l.x)},${n(l.y)} ${n(l.size)},${n(l.lineHeight)},${n(l.width)},${n(l.height)} ` +
        `| ${l.widths.map(n).join(',')} | ${spans.join(' / ')}`
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
