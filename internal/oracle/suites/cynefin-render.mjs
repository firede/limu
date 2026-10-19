// Cynefin diagrams drawn end to end: what Pele's render draws, read back out of its SVG as one
// line per thing drawn, in the order drawn. limu's test writes the same lines from its scene.
//
// Pele's SVG is drawn here with its numbers written in full, as the oracle's hooks have num()
// write them while limuExactNumbers is set, and with each label and marker written as the
// point it is drawn about, as the hooks have Pele's cynefin render write them. Pele draws
// everything where it is on the page, so no number is moved. A path's data is written as
// Pele writes it, arcs and all; limu's test reads it with scene.ParsePath, which draws arcs
// with the curves limu's scene draws them with. The lines hold every number in full, as
// String() writes it, so that a difference in the last bit shows.
//
// The fixture also holds the seeded numbers of the boundaries, seededRandom and hashString
// of a range of seeds and strings, where JavaScript's conversions between doubles and 32-bit
// integers matter.
import { lone } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { unescape, attr, n } from '../lib/svg.mjs';
import { readLabel, readMarker } from '../lib/limu.mjs';
import { BASIC, FRAGMENTS, corpus, hostile, randomDiagrams } from './cynefin.mjs';

// The icons the host has, in the cases drawn with icons.
export const ICONS = ['star', 'fa:fa-car', 'fa:fa-coffee'];

const FACES = ['-', 'b', 'i', 'bi'];
// A color of the series as limu's test names it.
const paint = (v) => (v === undefined ? '-' : v.replace(/^var\(--pele-series-(\d),[^)]*\)$/, 'series-$1'));

// The lines of a diagram's SVG, drawn with its numbers in full.
export function readSvg(svg) {
  const lines = [];
  const head = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  lines.push(`size ${attr(head, 'width')},${attr(head, 'height')}`);
  const title = /<title id="[^"]*">([^<]*)<\/title>/.exec(svg);
  if (title) lines.push(`acc-title ${unescape(title[1])}`);
  const desc = /<desc id="[^"]*">([^<]*)<\/desc>/.exec(svg);
  if (desc) lines.push(`acc-desc ${unescape(desc[1])}`);
  // What each open group gives what is in it, the root's first.
  const PAINT = ['fill', 'fill-opacity', 'stroke', 'stroke-width', 'stroke-dasharray', 'stroke-linecap', 'stroke-linejoin'];
  const groups = [{ fill: 'var(--_fg)' }];
  const inherited = (e, name) => attr(e, name) ?? groups.findLast((g) => g[name] !== undefined)?.[name];
  const pen = (e) => {
    const v = Object.fromEntries(PAINT.map((name) => [name, inherited(e, name)]));
    const fill = paint(v.fill) + (v['fill-opacity'] !== undefined ? '@' + v['fill-opacity'] : '');
    return `${fill} ${paint(v.stroke)} ${v['stroke-width'] ?? '-'} ${(v['stroke-dasharray'] ?? '-').replace(/ /g, ',')} ${v['stroke-linecap'] ?? 'butt'} ${v['stroke-linejoin'] ?? 'miter'}`;
  };
  const label = (e) => {
    const l = readLabel(e);
    const own = (name) => new RegExp(`\\s${name}="([^"]*)"`).exec(l.attrs)?.[1] ?? '-';
    const spans = l.lines
      .map((line, i) => n(l.widths[i]) + ': ' + line.map((s) => (s.icon !== undefined ? `i:${s.icon}` : `t:${FACES[s.style & 3]}:${s.text}`) + ' ' + n(s.width)).join(' + '))
      .join(' | ');
    return `label ${own('class')} ${own('font-size')} ${paint(own('fill') === '-' ? undefined : own('fill'))} ${own('font-weight')} ${own('font-style')} ${n(l.x)},${n(l.y)} ${n(l.width)},${n(l.height)} ${n(l.size)} ${n(l.lineHeight)} | ${spans}`;
  };

  const tags = /<(\/?)(g|path|rect|svg|limu-label|limu-marker)\b[^>]*>/g;
  // Inside an icon, whose drawing is the host's.
  let icon = false;
  for (const m of svg.matchAll(tags)) {
    const e = m[0];
    if (m[1]) {
      if (m[2] === 'g' && !icon) groups.pop();
      if (m[2] === 'svg') icon = false;
      continue;
    }
    if (icon) continue;
    switch (m[2]) {
      case 'svg':
        icon = / class="pele-icon"/.test(e) && !e.endsWith('/>');
        break;
      case 'g': {
        groups.push(Object.fromEntries(PAINT.map((name) => [name, attr(e, name)])));
        const cls = attr(e, 'class') ?? '';
        if (/^pele-(cynefin-domain|node|edge)( |$)/.test(cls)) lines.push(`group ${cls} ${attr(e, 'data-id') ?? '-'}`);
        break;
      }
      case 'path':
        // A marker's outline is the scene's to draw from where its tip is.
        if (attr(e, 'class') !== 'pele-marker') lines.push(`path ${attr(e, 'class') ?? '-'} ${attr(e, 'data-id') ?? '-'} ${attr(e, 'd')} ${pen(e)}`);
        break;
      case 'rect': {
        const r = attr(e, 'rx') !== undefined || (attr(e, 'style') ?? '').includes('rx:') ? 'r' : '-';
        lines.push(`rect ${n(Number(attr(e, 'x')))},${n(Number(attr(e, 'y')))},${n(Number(attr(e, 'width')))},${n(Number(attr(e, 'height')))} ${r} ${pen(e)}`);
        break;
      }
      case 'limu-label':
        lines.push(label(e));
        break;
      case 'limu-marker': {
        const k = readMarker(e);
        lines.push(`marker ${k.type} ${n(k.x)},${n(k.y)} ${n(k.dx)},${n(k.dy)} ${k.color}`);
        break;
      }
    }
  }
  return lines;
}

export default async function ({ pele, src, write, random }) {
  const { render, PeleError } = await src('index.ts');
  const { metricsMeasurer } = await src('text/measurer.ts');
  const { seededRandom, hashString } = await src('diagrams/cynefin/boundaries.ts');

  const out = [];
  const host = (name) => (ICONS.includes(name) ? '<path d="M0,0"/>' : undefined);
  const draw = (source, fontSize, icons) => {
    let r;
    globalThis.limuExactNumbers = true;
    try {
      r = render(source, { measurer: metricsMeasurer, fontSize, icons: icons ? host : undefined, outputLimit: Infinity });
    } catch (e) {
      if (!(e instanceof PeleError)) throw e;
      return false;
    } finally {
      globalThis.limuExactNumbers = false;
    }
    if (r.type !== 'cynefin') return false;
    out.push({ src: source, fontSize, icons: icons || undefined, lines: readSvg(r.svg) });
    return true;
  };
  const SIZES = [13, 20, 11, 24, 15, 18];
  const seen = new Set();
  let drawn = 0;
  const each = (source) => {
    // A Go string cannot hold a surrogate with no partner.
    if (seen.has(source) || lone(source) || source.length > 4000) return;
    seen.add(source);
    if (!draw(source, undefined, drawn % 9 === 4)) return;
    // Every fifth one again in another size of type.
    if (drawn % 5 === 0) draw(source, SIZES[(drawn / 5) % SIZES.length]);
    drawn++;
  };
  corpus(pele).forEach(each);
  hostile().forEach(each);
  // Diagrams with icons in their labels, for the hosts that have them.
  for (const s of [
    'cynefin-beta\ntitle fa:fa-car Diagram\ncomplex\n"fa:fa-car Car"\n"fa:fa-coffee"\nconfusion\n"x fa:fa-nope y"\ncomplex --> clear : "fa:fa-star"\n',
    'cynefin-beta\nclear\n"fa:fa-star star <b>bold</b>"\nchaotic --> complex : "fa:fa-coffee Coffee break"\n',
  ]) {
    seen.delete(s);
    draw(s, undefined, true);
    draw(s, 20, true);
  }
  // Every seed of a range of them, as the boundaries are drawn from.
  const settings = (lines) => `---\nconfig:\n  cynefin:\n${lines.map((l) => '    ' + l).join('\n')}\n---\n${BASIC}`;
  for (const seed of [1, 2, 3, -1, 0.5, 1e-9, -0.5, 77.77, 2 ** 31 - 1, 2 ** 31, -(2 ** 31), 2 ** 32 - 1, 2 ** 32 + 5, 2 ** 53, 9.99e14, -9.99e14]) {
    each(settings([`seed: ${seed}`, 'boundaryAmplitude: 30']));
  }
  const diagrams = randomDiagrams(random(81));
  for (let i = 0; i < 900; i++) each(diagrams());
  const usable = [...seen].filter((s) => s.length <= 3000);
  const next = mutator(usable, FRAGMENTS, random(82));
  while (drawn < 1600) each(next());

  // The seeded numbers themselves.
  const rnd = random(83);
  const seeds = [0, 1, -1, 0.5, -0.5, 0x6d2b79f5, -0x6d2b79f5, 2 ** 31, 2 ** 32, 2 ** 52 + 0.5, 1e15, -1e15, 1e300, -1e300, NaN, Infinity, -Infinity];
  for (let i = 0; i < 2000; i++) {
    const r = rnd();
    seeds.push(i % 4 === 0 ? Math.floor(r * 2 ** 32) - 2 ** 31 : i % 4 === 1 ? (r - 0.5) * 2e15 : i % 4 === 2 ? Math.floor(r * 4000) - 2000 : (r - 0.5) * 100);
  }
  const randoms = seeds.filter((s) => Number.isFinite(s)).map((s) => [s, seededRandom(s)]);
  const strings = ['', 'pele', 'other', 'pele-0-1', 'pele-12-345', 'a', 'é', '😀', '中文', 'x'.repeat(100), 'the quick brown fox jumps over the lazy dog'];
  const hashes = strings.map((s) => [s, hashString(s)]);
  write('diagram/cynefin/testdata/pele/render.json', { cases: out, randoms, hashes });
}
