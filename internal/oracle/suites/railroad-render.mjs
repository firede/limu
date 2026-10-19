// Railroad diagrams drawn end to end: what Pele's render draws, read back out of its SVG as one
// line per thing drawn, in the order drawn. limu's test writes the same lines from its scene.
//
// Pele draws the rules in a group it moves by half a unit each way, where the scene holds
// every place on the page. Pele's SVG is drawn here with its numbers written out in full, as
// the oracle's hooks have num() write them while limuExactNumbers is set, and with the names
// of rules and the texts of boxes written as the middle of their line, before Pele adds the
// shift to their baseline. A number inside the group is moved as limu's layout moves it,
// half a unit added to it; a number Pele writes relative to the one before it, as the arcs
// and arrows of its tracks, is added to that one first, as the scene holds where it ends;
// and the lines hold every number in full, as String() writes it, so that a difference in
// the last bit shows.
import { lone } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { unescape, attr, n, pt, readHead } from '../lib/svg.mjs';
import { FRAGMENTS, NOTATIONS, corpus, grammar } from './railroad.mjs';

// A number inside the group of rules, where the scene has it.
const off = (v) => 0.5 + Number(v);

// A path Pele writes in the group, with M, H, V, a, l, v and z, as absolute steps from
// where each starts: M x y, L x y, A rx ry rot large sweep x y and Z, every number in full.
function absolute(d) {
  const out = [];
  let x = 0;
  let y = 0;
  let sx = 0;
  let sy = 0;
  for (const m of d.matchAll(/([MHValvz])([^MHValvz]*)/g)) {
    const v = m[2].split(/[ ,]+/).filter((s) => s !== '').map(Number);
    switch (m[1]) {
      case 'M':
        x = off(v[0]);
        y = off(v[1]);
        sx = x;
        sy = y;
        out.push(`M ${n(x)} ${n(y)}`);
        break;
      case 'H':
        x = off(v[0]);
        out.push(`L ${n(x)} ${n(y)}`);
        break;
      case 'V':
        y = off(v[0]);
        out.push(`L ${n(x)} ${n(y)}`);
        break;
      case 'a':
        x = x + v[5];
        y = y + v[6];
        out.push(`A ${n(v[0])} ${n(v[1])} ${n(v[2])} ${v[3]} ${v[4]} ${n(x)} ${n(y)}`);
        break;
      case 'l':
        x = x + v[0];
        y = y + v[1];
        out.push(`L ${n(x)} ${n(y)}`);
        break;
      case 'v':
        y = y + v[0];
        out.push(`L ${n(x)} ${n(y)}`);
        break;
      case 'z':
        x = sx;
        y = sy;
        out.push('Z');
        break;
    }
  }
  return out.join(' ');
}

// What a stroke is drawn with: its paint, width, dashes, and cap and join, which no group
// about it sets.
const pen = (e) =>
  `${attr(e, 'stroke') ?? '-'} ${attr(e, 'stroke-width') ?? '-'} ${attr(e, 'stroke-dasharray') ?? '-'} ` +
  `${attr(e, 'stroke-linecap') ?? 'butt'} ${attr(e, 'stroke-linejoin') ?? 'miter'}`;

// The lines of a railroad diagram's SVG, drawn with its numbers in full.
export function readSvg(svg) {
  const lines = [];
  const { open: head, body, acc } = readHead(svg);
  lines.push(`size ${pt(attr(head, 'width'), attr(head, 'height'))}`, ...acc);
  // The title: where its first line sits and the text of each line.
  const t = /^<text class="pele-title"([^>]*)>([\s\S]*?)<\/text>/.exec(body);
  if (t) {
    let at = / x="([^"]*)" y="([^"]*)"/.exec(t[1]);
    let content = [unescape(t[2])];
    if (t[2].includes('<tspan')) {
      content = [];
      for (const s of t[2].matchAll(/<tspan([^>]*)>([^<]*)<\/tspan>/g)) {
        if (/ x="/.test(s[1]) || content.length === 0) {
          if (content.length === 0) at = / x="([^"]*)" y="([^"]*)"/.exec(s[1]);
          content.push('');
        }
        content[content.length - 1] += unescape(s[2]);
      }
    }
    lines.push(`title t ${pt(at[1], at[2])} | ${content.join(' | ')}`);
  }
  const from = body.indexOf('<g class="pele-rules"');
  if (from < 0) return lines;
  const rules = body.slice(from);
  const size = attr(rules.slice(0, rules.indexOf('>')), 'font-size');
  // The text of a box: where the middle of its line is, and, when it writes something, the
  // size and face of its type and what it writes. Special text writes the question marks
  // about it apart.
  const leafText = (g) => {
    const m = /<text([^>]*)>([\s\S]*?)<\/text>/.exec(g);
    const face = attr(m[1], 'font-family') !== undefined ? 'mono' : attr(m[1], 'font-style') === 'italic' ? 'italic' : '-';
    const at = `t ${pt(off(attr(m[1], 'x')), off(attr(m[1], 'y').slice(1)))}`;
    if (m[2].includes('<tspan')) {
      const parts = [...m[2].matchAll(/<tspan[^>]*>([^<]*)<\/tspan>/g)].map((s) => unescape(s[1]));
      const mid = unescape(m[2].replace(/<tspan[^>]*>[^<]*<\/tspan>/g, ''));
      return `${at} ${n(size)} ${face} | ${parts[0]}|${mid}|${parts[1]}`;
    }
    return m[2] === '' ? at : `${at} ${n(size)} ${face} | ${unescape(m[2])}`;
  };
  const box = (r) =>
    `${pt(off(attr(r, 'x')), off(attr(r, 'y')))},${pt(attr(r, 'width'), attr(r, 'height'))} ` +
    `${attr(r, 'rx') !== undefined ? n(attr(r, 'rx')) : (attr(r, 'style') ?? '').includes('rx:') ? 'theme' : '-'} ` +
    `${attr(r, 'fill')} ${pen(r)}`;
  // Each rule's group, and in it, in the order drawn, the name, the track, the arrows, the
  // dots, and the boxes and counts; the boxes hold no groups.
  for (const piece of rules.split('<g class="pele-rule" data-id="').slice(1)) {
    lines.push(`rule ${unescape(piece.slice(0, piece.indexOf('"')))}`);
    const g = piece.slice(piece.indexOf('>') + 1);
    for (const e of g.matchAll(/<(text|path|circle|g)\b([^>]*)>/g)) {
      const [, tag, attrs] = e;
      const cls = attr(attrs, 'class') ?? '';
      const content = () => unescape(g.slice(e.index + e[0].length, g.indexOf('</text>', e.index)));
      if (tag === 'text' && cls === 'pele-rule-name') {
        lines.push(`name t ${pt(off(attr(attrs, 'x')), off(attr(attrs, 'y').slice(1)))} ${n(attr(attrs, 'font-size'))} ${attr(attrs, 'font-weight')} ${attr(attrs, 'text-anchor')} | ${content()}`);
      } else if (tag === 'path' && cls === 'pele-edge pele-track') {
        lines.push(`track ${attr(attrs, 'fill')} ${pen(attrs)} | ${absolute(attr(attrs, 'd'))}`);
      } else if (tag === 'path' && cls === 'pele-marker') {
        lines.push(`arrows ${attr(attrs, 'fill')} ${attr(attrs, 'stroke') ?? '-'} | ${absolute(attr(attrs, 'd'))}`);
      } else if (tag === 'circle') {
        lines.push(`${cls.split(' ')[1].slice(5)} ${pt(off(attr(attrs, 'cx')), off(attr(attrs, 'cy')))} ${n(attr(attrs, 'r'))} ${attr(attrs, 'fill')}`);
      } else if (tag === 'text' && cls === 'pele-edge-label pele-count') {
        // Where the middle of the count's line is, as limu works it out from its baseline.
        const y = 0.5 + (Number(attr(attrs, 'y')) - Number(attr(attrs, 'font-size')) * 0.35);
        lines.push(`count t ${pt(off(attr(attrs, 'x')), y)} ${n(attr(attrs, 'font-size'))} ${attr(attrs, 'fill')} | ${content()}`);
      } else if (tag === 'g' && cls.startsWith('pele-node')) {
        const end = g.indexOf('</g>', e.index);
        const node = g.slice(e.index, end);
        const r = /<rect\b[^>]*>/.exec(node)[0];
        const kind = cls.split(' ')[1].slice(5);
        const id = kind === 'nonterminal' ? ` ${attr(attrs, 'data-id')}` : '';
        lines.push(`${kind}${id} ${box(r)} | ${leafText(node)}`);
      }
    }
  }
  return lines;
}

const ARCS = [0, 2.25, 4, 7.5, 10, 13, 20, 40, 1e9];
const PADDINGS = [0, 3.3, 5, 12, 20, -4, 250];
const HSEPS = [0, 1.5, 8, 16, 40, 1000];
const VSEPS = [0, 2.5, 10, 30, 600];
const FONTS = [9.5, 14, 20, 0, -1, 300, 13.37];
const MARKERS = [0, 2, 8, 100, 4.5];
const STROKES = [0, 1, 2, 1.5, 50];

// A railroad config block of a few random settings, as front matter or a directive.
function config(rnd) {
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  const settings = [];
  if (rnd() < 0.4) settings.push(['arcRadius', pick(ARCS)]);
  if (rnd() < 0.3) settings.push(['padding', pick(PADDINGS)]);
  if (rnd() < 0.3) settings.push(['horizontalSeparation', pick(HSEPS)]);
  if (rnd() < 0.3) settings.push(['verticalSeparation', pick(VSEPS)]);
  if (rnd() < 0.3) settings.push(['fontSize', pick(FONTS)]);
  if (rnd() < 0.2) settings.push(['markerRadius', pick(MARKERS)]);
  if (rnd() < 0.2) settings.push(['strokeWidth', pick(STROKES)]);
  if (rnd() < 0.15) settings.push(['showMarkers', pick([false, true, '"no"'])]);
  if (rnd() < 0.05) settings.push(['arcRadius', pick(['wide', '"5"', '.inf', '.nan', 'null'])]);
  if (settings.length === 0) settings.push(['arcRadius', pick(ARCS)]);
  if (rnd() < 0.7) return `---\nconfig:\n  railroad:\n${settings.map(([k, v]) => `    ${k}: ${v}`).join('\n')}\n---\n`;
  const json = settings.filter(([, v]) => typeof v !== 'string' || v.startsWith('"')).map(([k, v]) => `"${k}": ${v}`);
  return `%%{init: {"railroad": {${json.join(', ')}}}}%%\n`;
}

// Grammars written here for what they draw: long sequences that fold, choices of many
// widths, loops with counts, and text of every kind.
const DRAWN = [
  'railroad-ebnf-beta\nr = a b c d e f g h i j k l m n o p q r s t u v w x y z ;',
  'railroad-ebnf-beta\nr = "a long terminal" "another long terminal" "and a third" "and a fourth one" name ;',
  'railroad-ebnf-beta\nr = ( a b c d e f g h ) | ( i j k l m n o p ) | q ;',
  'railroad-ebnf-beta\nr = { a b c d e f g h i j } [ k l m n o p q r s t ] ;',
  'railroad-ebnf-beta\nr = a ( b | c )* [ d ] ;\ns = e ;',
  'railroad-ebnf-beta\ntitle T\nr = a ( b | [ c d ] | { e | f } )+ g? ;\ns = ( ( a+ )+ | b* )* - "x" ;',
  'railroad-ebnf-beta\nr = a | bb | "c" ;',
  'railroad-ebnf-beta\nr = "x" | "a much wider alternative" | y ;',
  'railroad-ebnf-beta\nr = ? any character ? - "x" ;',
  'railroad-ebnf-beta\nr = ?  ? ;\ns = ? a  b ? ;',
  'railroad-abnf-beta\nr = 3a 2*4b 2*c *5d ' + '9'.repeat(400) + 'e *f 1*g 1h 0*1i 0*0j 5*2k ;',
  'railroad-abnf-beta\nr = 2*( a / b / c ) 3[ "x" ] ;',
  'railroad-abnf-beta\nr = %x41 %d65.66 %b1000001 %x30-39 ;',
  'railroad-peg-beta\nR <- !Keyword &"k" . !( a b ) ;',
  'railroad-peg-beta\nR <- ( a / b / c )* d+ e? ;',
  'railroad-beta\nr = sequence(terminal("<br>"), terminal("*a*"), terminal("#35;"), terminal("&lt;"), terminal(" "), terminal("a  b"), terminal("\\n\\t"), nonterminal("fa:fa-user")) ;',
  'railroad-beta\nr = sequence(terminal(""), nonterminal(""), special("")) ;',
  'railroad-beta\nr = choice(sequence(terminal("a"), terminal("b"), terminal("c"), terminal("d"), terminal("e")), zeroOrMore(choice(terminal("f"), terminal("g")))) ;',
  'railroad-beta\nr = optional(oneOrMore(zeroOrMore(optional(terminal("deep"))))) ;',
  'railroad-beta\nr = nonterminal("#35;&amp;<br> x") ;\ns = special("  spaced  ") ;',
  'railroad-beta\nr = terminal("中文中文") nonterminal("😀") ;',
  'railroad-ebnf-beta\ntitle A title much wider than the rules it names, by a long way, it goes on\nr = a ;',
  'railroad-ebnf-beta\ntitle <b>Bold</b> and <i>italic</i> #35;\nr = a ;',
  'railroad-ebnf-beta\ntitle first<br>second\nr = a ;',
  'railroad-ebnf-beta\naccTitle: Grammar\naccDescr {\n  Two\n  lines\n}\nr = a ;',
  'railroad-ebnf-beta\nr = a ;\nr = b ;',
  'railroad-ebnf-beta\nrule_with_a_very_long_name_that_is_wider_than_its_track = a ;',
  'railroad-beta',
  'railroad-ebnf-beta\ntitle Only a title',
  'railroad-ebnf-beta\nr = ' + Array.from({ length: 30 }, (_, i) => `a${i}`).join(' ') + ' ;',
  'railroad-ebnf-beta\nr = ' + Array.from({ length: 12 }, (_, i) => `"terminal${i}"`).join(' | ') + ' ;',
  'railroad-ebnf-beta\nr = ' + Array.from({ length: 8 }, (_, i) => `( x${i} y${i} z${i} )`).join(' ') + ' ;',
  'railroad-ebnf-beta\nr = { ' + Array.from({ length: 16 }, (_, i) => `n${i}`).join(' ') + ' } ;',
  'railroad-ebnf-beta\nr = [ ' + Array.from({ length: 16 }, (_, i) => `n${i}`).join(' ') + ' ] ;',
  'railroad-ebnf-beta\nr = ( ' + Array.from({ length: 16 }, (_, i) => `n${i}`).join(' ') + ' )+ ;',
  'railroad-ebnf-beta\nr = a | ( ' + Array.from({ length: 16 }, (_, i) => `n${i}`).join(' ') + ' ) ;',
];

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
    if (r.type !== 'railroad') return false;
    out.push({ src: source, maxWidth, fontSize, lines: readSvg(r.svg) });
    return true;
  };
  const NARROW = [360, 320, 280, 240, 200, 160, 120, 100, 80, 600, 700, 1];
  const SIZES = [13, 20, 11, 24, 14.5];
  const seen = new Set();
  let drawn = 0;
  const each = (source) => {
    if (seen.has(source) || lone(source) || source.length > 1500) return;
    seen.add(source);
    if (!draw(source)) return;
    // Every other one again in a narrow host, which draws a railroad tightly, with its long
    // sequences folded, and every fifth in another size of type.
    if (drawn % 2 === 0) draw(source, NARROW[(drawn / 2) % NARROW.length]);
    if (drawn % 5 === 0) draw(source, drawn % 10 === 0 ? undefined : NARROW[(drawn / 5) % NARROW.length], SIZES[(drawn / 5) % SIZES.length]);
    drawn++;
  };
  const rnd = random(41);
  DRAWN.forEach(each);
  corpus(pele).forEach(each);
  // Random grammars in each notation, a third of them with a config block.
  for (let i = 0; i < 900; i++) {
    const s = grammar(NOTATIONS[i % 4], rnd);
    each(i % 3 === 0 ? config(rnd) + s : s);
  }
  // The texts written here and the corpus again with config blocks.
  for (const s of [...DRAWN, ...corpus(pele).filter((x) => x.length < 600)].slice(0, 400)) each(config(rnd) + s);
  const usable = [...seen].filter((s) => s.length <= 2000);
  const next = mutator(usable, FRAGMENTS, random(42));
  while (drawn < 2800) each(next());
  write('diagram/railroad/testdata/pele/render.json', { cases: out });
}
