// Mermaid's palette theme variables, which Pele 1.13 reads as a diagram's own series colors:
// the cases that set them, and the lines that say what Pele made of them.
import { attr } from './svg.mjs';

// The series colors Pele sets on the svg, as --pele-series-N declarations in its style, a line
// for each, `series N value`, in the order they are declared. A port reads the value as a
// color of its own, so the suites keep these lines in full.
export function readSeries(open) {
  const lines = [];
  for (const decl of (attr(open, 'style') ?? '').split(';')) {
    const m = /^--pele-series-(\d):(.*)$/.exec(decl);
    if (m) lines.push(`series ${m[1]} ${m[2]}`);
  }
  return lines;
}

// Keeps the series lines in full, and what else keep keeps.
export const keepSeries =
  (keep = () => false) =>
  (line) =>
    line.startsWith('series ') || keep(line);

// Colors of every sort Pele's palette takes or refuses: named colors dark and light, hex codes
// of each length, the functions ink reads, with commas and without, at opacities above and
// below a half, those it passes on but ink can't read, and values it refuses, as a page's own
// variables, a declaration's end, an unbalanced parenthesis and a typo.
const COLORS = [
  'red', 'navy', 'gold', 'black', 'white', 'teal', 'Grey', 'darkgrey', ' MidnightBlue ', 'lightgreen',
  '#000', '#fff', '#1a2b3c', '#FFCC00', '#f0f8', '#11223380', '#abcdefAA', '#5b7bd5',
  'rgb(0, 0, 0)', 'rgb(255 255 255 / 0.3)', 'rgba(10,200,30,0.6)', 'rgb(10%, 50%, 90%)', 'rgb(250 250 250)',
  'hsl(0 50% 50% / 0.5)', 'hsl(200, 80%, 30%)', 'hsla(60, 100%, 50%, 0.9)', 'hsl(120deg 40% 80%)', 'hsl(0.5turn 60% 20%)',
  'transparent', 'currentColor',
  'color-mix(in srgb, red, blue)', 'var(--pele-accent)', 'var(--pele-series-2)', 'light-dark(red, blue)', 'oklch(0.5 0.1 200)',
  'var(--x)', 'red;fill:blue', 'rgb(0,0,0', 'url(x)', 'notacolor', '#12', 'expression(1)', '',
];

// Values of other types, which a palette passes over.
const OTHER = [5, null, true, ['red']];

// What each type's palette is named, and how many of its variables the cases set: Mindmap's
// starts at cScale0, which is the root's, and venn's skips some, as a palette that repeats
// what it sets does. A name of another palette, which the type passes over, is set too.
const PALETTES = {
  pie: { names: (i) => `pie${i}`, from: 0, to: 10, other: ['cScale0', 'git0', 'primaryColor'] },
  gitGraph: { names: (i) => `git${i}`, from: 0, to: 9, other: ['pie1', 'cScale0'] },
  timeline: { names: (i) => `cScale${i}`, from: 0, to: 9, other: ['pie1', 'fillType0'] },
  radar: { names: (i) => `cScale${i}`, from: 0, to: 9, other: ['pie1', 'git0'] },
  treemap: { names: (i) => `cScale${i}`, from: 0, to: 9, other: ['pie1', 'venn1'] },
  mindmap: { names: (i) => `cScale${i}`, from: 0, to: 10, other: ['pie1', 'git1'] },
  venn: { names: (i) => `venn${i}`, from: 0, to: 10, other: ['cScale0', 'pie1'] },
};

// A palette of themeVariables for type, chosen with rnd.
function themeVariables(type, rnd) {
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  const color = () => (rnd() < 0.04 ? pick(OTHER) : pick(COLORS));
  const vars = {};
  if (type === 'xychart') {
    const count = 1 + Math.floor(rnd() * 10);
    const list = [];
    for (let k = 0; k < count; k++) list.push(rnd() < 0.08 ? '' : String(pick(COLORS)));
    vars.xyChart = { plotColorPalette: rnd() < 0.05 ? pick(OTHER) : list.join(pick([',', ', ', ' , '])) };
    return vars;
  }
  const palette = PALETTES[type];
  // Some palettes set every series, and most a few.
  const chance = rnd() < 0.3 ? 1 : 0.2 + rnd() * 0.5;
  for (let i = palette.from; i <= palette.to; i++) if (rnd() < chance) vars[palette.names(i)] = color();
  if (rnd() < 0.2) vars[pick(palette.other)] = color();
  if (Object.keys(vars).length === 0) vars[palette.names(palette.from + 1)] = color();
  return vars;
}

// A source with a palette set for type, as a directive or in front matter, chosen with rnd.
function withPalette(source, type, rnd) {
  const vars = themeVariables(type, rnd);
  if (rnd() < 0.6) return `%%{init: ${JSON.stringify({ themeVariables: vars })}}%%\n${source}`;
  return `---\nconfig:\n  themeVariables: ${JSON.stringify(vars)}\n---\n${source}`;
}

// count sources of a diagram of type, each one of sources, which a suite has drawn, with a
// palette set, chosen with rnd. A source that already has front matter or a directive is
// passed over, so that the palette is the only configuration added.
export function paletted(sources, type, rnd, count) {
  const usable = sources.filter((s) => s.length <= 2000 && !s.startsWith('---') && !s.includes('%%{'));
  const out = [];
  for (let k = 0; k < count && usable.length > 0; k++) out.push(withPalette(usable[Math.floor(rnd() * usable.length)], type, rnd));
  return out;
}
