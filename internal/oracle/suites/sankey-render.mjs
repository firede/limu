// Sankey diagrams drawn end to end: what Pele's render draws, read back out of its SVG as one
// line per thing drawn, in the order drawn, and the errors of the diagrams it refuses to draw.
// limu's test writes the same lines from its scene.
//
// Pele draws the chart in a group it moves by where the labels reach, and its title outside
// it. Its SVG is drawn here with its numbers written in full, as the oracle's hooks have num()
// write them while limuExactNumbers is set, and with each label's y the middle of its line in
// place of its baseline, as the scene holds a label. A number in the group is moved as limu's
// layout moves it, by the group's move added to it, and the lines hold every number in full,
// as String() writes it, so that a difference in the last bit shows. The style a diagram's
// config gives a bar or a band is written as Pele writes it, for the test to read as limu
// reads it.
import { lone } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { unescape, attr, n } from '../lib/svg.mjs';
import { FRAGMENTS, corpus, hostile, rowDiagrams } from './sankey.mjs';

const css = (e) => encodeURIComponent(attr(e, 'style') ?? '');
// A stroke's ends and corners, as the element's attributes set them, or SVG's initial values:
// no group of a sankey diagram sets them, nor does the style the config gives.
const capJoin = (e) => `${attr(e, 'stroke-linecap') ?? 'butt'} ${attr(e, 'stroke-linejoin') ?? 'miter'}`;

// The lines of a sankey diagram's SVG, drawn with its numbers in full.
export function readSvg(svg) {
  const lines = [];
  const head = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  lines.push(`size ${n(attr(head, 'width'))},${n(attr(head, 'height'))}`);
  if (/<title id=/.test(svg.slice(0, svg.indexOf('<g'))) || /<desc id=/.test(svg.slice(0, svg.indexOf('<g')))) lines.push('acc');
  // The title, as Pele's labelSvg writes it: where its first line is, and the text of each.
  const title = svg.indexOf('<text class="pele-title"');
  if (title >= 0) {
    const s = svg.slice(title, svg.indexOf('</text>', title));
    const open = s.slice(0, s.indexOf('>'));
    const body = s.slice(open.length + 1);
    let at = / x="([^"]*)" y="([^"]*)"/.exec(open);
    let content;
    if (body.includes('<tspan')) {
      content = [];
      for (const t of body.matchAll(/<tspan([^>]*)>([^<]*)<\/tspan>/g)) {
        if (/ x="/.test(t[1]) || content.length === 0) {
          if (content.length === 0) at = / x="([^"]*)" y="([^"]*)"/.exec(t[1]);
          content.push('');
        }
        content[content.length - 1] += unescape(t[2]);
      }
    } else {
      content = [unescape(body)];
    }
    lines.push(`title ${n(at[1])},${n(at[2])} ${attr(open, 'font-size') ?? '-'} | ${content.join(' | ')}`);
  }
  const g = /<g transform="translate\(([^,]*),([^)]*)\)">/.exec(svg);
  const ox = Number(g[1]);
  const oy = Number(g[2]);
  const pt = (x, y) => `${String(ox + Number(x))},${String(oy + Number(y))}`;
  const body = svg.slice(g.index + g[0].length);

  const links = /<g class="pele-sankey-links" fill-opacity="([^"]*)">/.exec(body);
  lines.push(`links ${links[1]}`);
  for (const m of body.matchAll(/<path class="pele-sankey-link"([^>]*)><title>([^<]*)<\/title><\/path>/g)) {
    const e = m[1];
    const d = [];
    for (const c of attr(e, 'd').matchAll(/([MCLZ])([^MCLZ]*)/g)) {
      const pairs = c[2].trim() === '' ? [] : c[2].trim().split(' ').map((p) => p.split(','));
      d.push([c[1], ...pairs.map(([x, y]) => pt(x, y))].join(' '));
    }
    lines.push(`link ${attr(e, 'data-id')} ${d.join(' ')} ${attr(e, 'fill')} css=${css(e)} ${capJoin(e)} | ${unescape(m[2])}`);
  }
  for (const m of body.matchAll(/<rect class="pele-sankey-node"[^>]*>/g)) {
    const e = m[0];
    lines.push(
      `node ${attr(e, 'data-id')} ${pt(attr(e, 'x'), attr(e, 'y'))},${n(attr(e, 'width'))},${n(attr(e, 'height'))} r${n(attr(e, 'rx'))}` +
        ` ${attr(e, 'fill')} ${attr(e, 'stroke') ?? '-'} css=${css(e)} ${capJoin(e)}`
    );
  }
  const labels = /<g class="pele-labels" font-size="([^"]*)">/.exec(body);
  lines.push(`labels ${labels[1]}`);
  for (const m of body.matchAll(/<g class="pele-label" data-id="([^"]*)">([\s\S]*?)<\/g>/g)) {
    const texts = [...m[2].matchAll(/<text([^>]*)>([\s\S]*?)<\/text>/g)];
    const main = texts[texts.length - 1];
    const y = attr(main[1], 'y');
    if (!y.startsWith('m')) throw new Error(`a label's y is ${y}`);
    const value = /<tspan class="pele-sankey-value" fill="([^"]*)">([^<]*)<\/tspan>/.exec(main[2]);
    // Pele writes a space between the id and the value.
    const id = unescape(value ? main[2].slice(0, value.index - 1) : main[2]);
    let line = `label ${unescape(m[1])} ${pt(attr(main[1], 'x'), y.slice(1))} ${attr(main[1], 'text-anchor')} ${attr(main[1], 'fill') ?? '-'} | ${id}`;
    if (value) line += ` | ${value[1]} ${unescape(value[2])}`;
    if (texts.length > 1) {
      // The outline under the text, which writes the same.
      const [, a, t] = texts[0];
      if (attr(a, 'x') !== attr(main[1], 'x') || attr(a, 'y') !== y || unescape(t) !== id + (value ? ' ' + unescape(value[2]) : '')) {
        throw new Error(`an outline unlike its label: ${m[0]}`);
      }
      line += ` | halo ${attr(a, 'fill')} ${attr(a, 'stroke')} ${attr(a, 'stroke-width')} ${attr(a, 'stroke-linejoin')}`;
    }
    lines.push(line);
  }
  return lines;
}

// Diagrams with every option of the config, in front matter or a directive.
export function configuredDiagrams(rnd) {
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  const chance = (p) => rnd() < p;
  const rows = rowDiagrams(rnd);
  const bodies = [
    'sankey\na,b,5\na,c,3\nb,d,4\nc,d,1.239\n',
    'sankey\na,b,4\nb,c,3\nc,d,3\nx,d,2\na,y,1\n',
    'sankey\na,b,1\na,c,1\na,d,1\n',
    'sankey\nA,Somewhere with a very long name indeed,1\n',
    'sankey\na,b,30\na,c,10\nb,d,30\n',
    'sankey\nsolo,sink,0.001\nbig,sink,1000\n',
  ];
  const options = [
    () => `width: ${pick(['300', '800', '1200', '160', '50', '0', '-5', '20000', '600.5', 'wide', '.inf'])}`,
    () => `height: ${pick(['150', '600', '50', '1', '0', '20000', '333.3', 'tall'])}`,
    () => `nodeWidth: ${pick(['24', '1', '0', '200', '500', '3.5', 'x'])}`,
    () => `nodePadding: ${pick(['0', '40', '5', '200', '300', '-1', '-0', '7.25'])}`,
    () => `showValues: ${pick(['false', 'true', '0', 'no', '"false"'])}`,
    () => `prefix: ${pick(['"$"', '"€ "', '3', '1.5', '""', '"&amp;"', '"#35;"', '"<b>"', 'true'])}`,
    () => `suffix: ${pick(['" M"', '"%"', '"TWh"', '2', '"&#35;"', '"; x"'])}`,
    () => `labelStyle: ${pick(['outlined', 'legacy', 'none', '1'])}`,
    () => `nodeAlignment: ${pick(['justify', 'left', 'right', 'center', 'nonsense', '3'])}`,
    () => `linkColor: ${pick(['gradient', 'source', 'target', '"#a1a1a1"', '"url(#x)"', 'red', '"none"', '"#12345680"', '"rgb(1, 2, 3)"', '""', '5', '"red;x"'])}`,
    () =>
      `nodeColors:\n${Array.from({ length: 1 + Math.floor(rnd() * 3) }, () => `      ${pick(['a', 'b', 'c', 'd', 'x', 'Coal', '__proto__', '"1"'])}: ${pick(['"#123456"', 'red', '"url(#x)"', '"none"', '3', '"#fff8"', '"transparent"', '"rgba(0,0,0,0.5)"'])}`).join('\n')}`,
  ];
  return () => {
    const body = chance(0.6) ? pick(bodies) : rows();
    const chosen = [];
    for (let k = 1 + Math.floor(rnd() * 4); k > 0; k--) chosen.push(pick(options)());
    if (chance(0.1)) {
      const json = {};
      for (const c of chosen) {
        const [key, ...rest] = c.split(':');
        const raw = rest.join(':').trim();
        if (raw.includes('\n')) continue;
        try {
          json[key] = JSON.parse(raw);
        } catch {
          json[key] = raw;
        }
      }
      return `%%{init: {"sankey": ${JSON.stringify(json)}}}%%\n${body}`;
    }
    const title = chance(0.25) ? `title: ${pick(['Energy', 'A title long enough to be wider than a narrow chart, and to wrap', '"<b>b</b> #35;"', '"中文"'])}\n` : '';
    return `---\n${title}config:\n  sankey:\n${chosen.map((c) => '    ' + c).join('\n')}\n---\n${body}`;
  };
}

// Larger diagrams, as Pele's stress tests draw them, made smaller.
function larger() {
  const repeat = (count, line) => Array.from({ length: count }, (_, i) => line(i)).join('');
  const H = 'sankey\n';
  let mesh = H;
  for (let i = 0; i < 12; i++) for (let j = 0; j < 12; j++) mesh += `a${i},b${j},${(i * 7 + j) % 13}\n`;
  let layered = H;
  for (let l = 0; l < 8; l++) for (let i = 0; i < 6; i++) for (let j = 0; j < 6; j += 3) layered += `${l}_${i},${l + 1}_${(i + j) % 6},${1 + ((i + j) % 5)}\n`;
  return [
    H + repeat(60, (i) => `${i},${i + 1},1\n`),
    H + repeat(40, (i) => `c,t${i},${i + 1}\n`),
    H + repeat(40, (i) => `s${i},c,${i + 1}\n`),
    H + 'a,b,1\n'.repeat(30),
    H + repeat(20, (i) => `n${i},n${i + 1},1\nhub,n${i + 1},1\n`),
    mesh,
    layered,
    H + repeat(30, (i) => `${i},${(i + 1) % 30},1\n`),
    H + 'a,b,1\nb,a,1\n' + repeat(30, (i) => `${i},${i + 1},1\n`) + '30,a,1\n',
    H + repeat(20, (i) => `${i},${i},1\n`),
    H + repeat(60, (i) => `a${i % 5},b${i % 6},${i % 2 ? '1e308' : '5e-324'}\n`),
    H + repeat(60, (i) => `a${i % 5},b${i % 6},x\n`),
    H + 'a,b,5e-324\nc,d,0\n',
    H + 'a,b,5e-324\n',
    H + 'a,b,1e-310\nb,c,1e-300\n',
    H + repeat(3, (i) => `${String(i).repeat(200)},${String(i + 1).repeat(200)},1\n`),
  ];
}

export default async function ({ pele, src, write, random }) {
  const { render, PeleError } = await src('index.ts');
  const { metricsMeasurer } = await src('text/measurer.ts');
  const { cleanupComments } = await src('preprocess.ts');

  const out = [];
  let refused = 0;
  const draw = (source, maxWidth, fontSize) => {
    let r;
    globalThis.limuExactNumbers = true;
    try {
      r = render(source, { measurer: metricsMeasurer, maxWidth, fontSize, outputLimit: Infinity });
    } catch (e) {
      if (!(e instanceof PeleError)) throw e;
      // A diagram that parses and is refused, as one with circular links is, is kept with
      // its error.
      if (e.code === 'semantic' && e.type === 'sankey') {
        out.push({ src: source, maxWidth, fontSize, lines: [`error ${e.code} ${e.message}`] });
        refused++;
        return true;
      }
      return false;
    } finally {
      globalThis.limuExactNumbers = false;
    }
    if (r.type !== 'sankey') return false;
    out.push({ src: source, maxWidth, fontSize, lines: readSvg(r.svg) });
    return true;
  };
  const NARROW = [360, 320, 280, 240, 200, 160, 120, 100, 600, 500, 680, 90];
  const SIZES = [13, 20, 11, 24, 15, 13.5];
  const seen = new Set();
  let drawn = 0;
  const each = (source) => {
    // A Go string cannot hold a surrogate with no partner.
    if (seen.has(source) || lone(source) || source.length > 4000) return;
    seen.add(source);
    if (!draw(source)) return;
    // Every third one again in a narrow host, which draws the chart narrower, and every fifth
    // in another size of type.
    if (drawn % 3 === 0) draw(source, NARROW[(drawn / 3) % NARROW.length]);
    if (drawn % 5 === 0) draw(source, drawn % 10 === 0 ? undefined : NARROW[(drawn / 5) % NARROW.length], SIZES[(drawn / 5) % SIZES.length]);
    drawn++;
  };
  corpus(pele, cleanupComments).forEach(each);
  hostile(pele).forEach(each);
  larger().forEach(each);
  const configured = configuredDiagrams(random(51));
  for (let i = 0; i < 1500; i++) each(configured());
  const rows = rowDiagrams(random(52));
  for (let i = 0; i < 800; i++) each(rows());
  const usable = [...seen].filter((s) => s.length <= 3000);
  const next = mutator(usable, FRAGMENTS, random(53));
  while (drawn < 4000) each(next());
  // Nodes thinner than a label, which claim room above and below them: the room is the claim
  // less the node's height, as it is kept, which a port that fused the product of the height
  // into the difference would change in the last bit.
  for (const [height, padding, rows] of [
    [65, 3, 'a,b,1.43\nc,e,6.71\n'],
    [67, 1, 'c,e,17.1\nd,e,2.18\na,c,8.55\nb,d,12.5\nd,e,21.8\n'],
    [75, 3, 'b,c,11.3\nd,e,26.5\nc,d,6.12\nb,c,11.3\nb,c,28\nc,e,17.8\na,b,7.58\n'],
    [106, 2, 'd,e,4.58\na,c,6.58\nb,c,24.9\nc,e,34.4\n'],
  ]) {
    each(`---\nconfig:\n  sankey:\n    height: ${height}\n    showValues: false\n    nodePadding: ${padding}\n---\nsankey\n${rows}`);
  }
  if (refused < 20) throw new Error(`only ${refused} diagrams refused`);
  // The test reads the style Pele writes, css=, as limu reads declarations, so the lines with
  // one are kept in full.
  write('diagram/sankey/testdata/pele/render.json', { cases: out }, { keep: (line) => / css=[^ ]/.test(line) });
}
