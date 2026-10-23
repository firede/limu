// Labels as Pele writes them into its SVG: each laid out by layoutLabel, then written centered
// on a point by labelSvg and starting at one by leftLabelSvg, and read back as what a browser
// is given to place. limu's svg package writes the same labels about its items' points, a
// Middle and a Start label, and reads its own SVG back the same way.
//
// A label is read as a line for each run of text placed on its own, and then a line for each
// icon:
//
//   t <anchor> <x>,<y> <spans>
//   icon <name> <x>,<y>,<width>,<height>
//
// A run starts where a <text> or <tspan> has an x, and the text and spans after it follow it
// until the next. Its anchor is the text-anchor of the element the run starts at, or of the
// <text> it is in, start where neither has one. Its spans are its text, each piece in a face
// of its own written with the face's letters before it, as [bi:…]: b bold, i italic, m
// monospaced (any font-family), u underlined, and _ lowered or ^ raised (baseline-shift).
// Adjacent pieces in one face are joined. A run of nothing but spaces draws nothing and is
// not read, as Pele writes one for an empty line in labelSvg and none in leftLabelSvg.
import { attr, unescape } from '../lib/svg.mjs';

// Texts written here, for what places a line: one span or many, faces, breaks and empty
// lines, wrapping, spaces at either end, icons at the start, the middle and the end of a line,
// and characters measured otherwise than Latin letters.
const WRITTEN = [
  'Start',
  'Is it <b>ready</b>?',
  'one<br>two words<br>three',
  'a<br><br>b',
  '<b>bold</b> and <i>italic</i> and <b><i>both</i></b>',
  '**bold** and _italic_ in markdown',
  'a line long enough that it wraps at the width it is given, more than once',
  '  spaced  ',
  'fa:fa-car Car',
  'go fa:fa-car now',
  'Car fa:fa-car',
  'fa:fa-car<br>under an icon<br>fab:fa-github',
  'fa:fa-missing is no icon',
  '中文的标签和 Latin',
  '😀 emoji',
  '&amp; &lt;b&gt; entities',
  'rules: String?',
  'fragments: [FragmentID]',
  'Call <code>render()</code> once',
  '<code>all of it code</code>',
  'H<sub>2</sub>O and x<sup>2</sup>',
  'a <sub>b <sup>c</sup> d</sub> e',
  '<u>under</u> and <ins>inserted</ins>',
  '<b><code>bold code</code></b> <kbd>Ctrl</kbd>',
  'soft\u00adly hy\u00adphen\u00adated words that wrap at the width they are given',
];

const PIECES = [
  'word', 'Word', 'longerword', 'a', ' ', ' ', '  ', '<b>', '</b>', '<i>', '</i>', '<br>', '**', '_', 'fa:fa-car',
  'fab:fa-github', '中文', '😀', 'é', '&amp;', ':', '(', ')', '<sub>', '</sub>', '<sup>', '</sup>', '<code>',
  '</code>', '<u>', '</u>',
];

const ICONS = ['fa:fa-car', 'fab:fa-github'];

// The lines of a label's SVG, as the head of this file says.
export function readLabel(svg) {
  const runs = [];
  const icons = [];
  // The faces and anchors of the elements open, innermost last.
  const open = [];
  let run;
  const TOKEN = /<(\/?)([A-Za-z][\w-]*)([^>]*?)(\/?)>|([^<]+)/g;
  let skip = 0;
  for (const m of svg.matchAll(TOKEN)) {
    const [, close, name, attrs, empty, data] = m;
    if (skip > 0) {
      if (name === 'svg') skip += close ? -1 : empty ? 0 : 1;
      continue;
    }
    if (data !== undefined) {
      if (!run || open.length === 0) continue;
      const top = open[open.length - 1];
      const face = (top.bold ? 'b' : '') + (top.italic ? 'i' : '') + (top.mono ? 'm' : '') + (top.under ? 'u' : '') + top.shift;
      const last = run.spans[run.spans.length - 1];
      if (last && last.face === face) last.text += unescape(data);
      else run.spans.push({ face, text: unescape(data) });
      continue;
    }
    if (name === 'svg' && !close) {
      const icon = attr(attrs, 'data-icon');
      if (icon !== undefined) {
        icons.push(`icon ${icon} ${attr(attrs, 'x')},${attr(attrs, 'y')},${attr(attrs, 'width')},${attr(attrs, 'height')}`);
        if (!empty) skip = 1;
      }
      continue;
    }
    if (name !== 'text' && name !== 'tspan') continue;
    if (close) {
      open.pop();
      continue;
    }
    const up = open[open.length - 1] ?? { bold: false, italic: false, mono: false, under: false, shift: '', anchor: 'start' };
    const shift = attr(attrs, 'baseline-shift');
    const el = {
      bold: up.bold || attr(attrs, 'font-weight') !== undefined,
      italic: up.italic || attr(attrs, 'font-style') === 'italic',
      mono: up.mono || attr(attrs, 'font-family') !== undefined,
      under: up.under || attr(attrs, 'text-decoration') === 'underline',
      shift: shift === 'sub' ? '_' : shift === 'super' ? '^' : up.shift,
      anchor: name === 'text' ? attr(attrs, 'text-anchor') ?? 'start' : up.anchor,
    };
    if (!empty) open.push(el);
    if (attr(attrs, 'x') !== undefined) {
      run = { anchor: attr(attrs, 'text-anchor') ?? el.anchor, x: attr(attrs, 'x'), y: attr(attrs, 'y'), spans: [] };
      runs.push(run);
    }
  }
  const lines = [];
  for (const r of runs) {
    const text = r.spans.map((s) => s.text).join('');
    if (/^ *$/.test(text)) continue;
    const spans = r.spans.map((s) => (s.face ? `[${s.face}:${s.text}]` : s.text)).join('');
    lines.push(`t ${r.anchor} ${r.x},${r.y} ${spans}`);
  }
  return lines.concat(icons);
}

export default async function ({ src, write, random }) {
  const { metricsMeasurer } = await src('text/measurer.ts');
  const { layoutLabel } = await src('text/label.ts');
  const { withIcons } = await src('text/icons.ts');
  const { labelSvg } = await src('svg/builder.ts');
  const { leftLabelSvg } = await src('diagrams/common/literal-label.ts');

  const pick = random(20261023);
  const texts = [...WRITTEN];
  while (texts.length < 400) {
    let s = '';
    for (let n = 1 + Math.floor(pick() * 12); n > 0; n--) s += PIECES[Math.floor(pick() * PIECES.length)];
    texts.push(s);
  }
  const resolve = (name) => (ICONS.includes(name) ? '<path/>' : undefined);
  const cases = texts.map((text, i) => {
    const markdown = i < WRITTEN.length ? text.includes('**') : pick() < 0.4;
    const size = [12, 14, 16, 13.5][i < WRITTEN.length ? 2 : Math.floor(pick() * 4)];
    const maxWidth = i < WRITTEN.length ? 200 : [30, 60, 100, 200, 4000][Math.floor(pick() * 5)];
    const face = i < WRITTEN.length ? 0 : Math.floor(pick() * 4);
    const label = withIcons(resolve, () => layoutLabel(text, markdown, metricsMeasurer, size, maxWidth, face));
    // A label at (100, 50): centered on it, and starting at it with its middle there.
    const middle = readLabel(labelSvg(label, 100, 50, '', resolve));
    const start = readLabel(leftLabelSvg(label, 100, 50 - label.height / 2, ''));
    return { text, markdown, size, maxWidth, face, label, middle, start };
  });
  write('svg/testdata/pele/svg.json', { cases });
}
