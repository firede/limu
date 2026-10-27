// Radar charts parsed as Pele parses them: the tokens Pele's common/tokens.ts makes of the
// text with the radar's table, the tree parseRadar makes of them, and the model buildRadar
// makes of the tree. The cases are Mermaid's documentation and specs, the inputs of Pele's own
// tests, cases written here, random charts, and all of those mutated at random.
import { enc, peleError } from '../lib/value.mjs';
import { lone, picker, testStrings, docs } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { locator } from '../lib/located.mjs';

// The fragments of Pele's tests/radar/oracle.test.ts, and more.
export const FRAGMENTS = [
  'radar-beta', 'radar-beta:', 'radar-beta :', 'title', 'title ', 'accTitle: ', 'accDescr: ', 'accDescr {', 'accDescr {}', '}', '{', '[', ']',
  ':', ' : ', ',', ', ', '"', "'", '"a"', "'b'", '["L"]', '1', '0', '00', '1.5', '0.5', '01', '1.', '.5', '1.2.3', '1e3', '-1',
  '\n', '\r\n', ' ', '\t', '%%', '%% c', '%%{init: {}}%%', '---', '---\n', '\\', 'x', 'A', 'B', 'a-b', 'a_1', '-', '_', 'é', ';',
  '#35;', 'axis', 'axis ', 'curve', 'curve ', 'ticks', 'ticks 5', 'max', 'max 10', 'min', 'min 0', 'graticule', 'graticule polygon',
  'circle', 'polygon', 'showLegend', 'showLegend false', 'true', 'false', 'axis A, B', '\ncurve c{1, 2}', 'curve d{ A: 1, B 2 }',
  'axisx', 'maxi', 'trues',
  '\r', ' ', '　', '😀', '中文', '<br>', '&amp;', '"a\\nb"', '99999999999999999999', '0.000001', '1000000', '7.25',
  '\n axis Z["Zed"]', '\n curve q["Q"]{3, 1, 4, 1, 5}', '\n ticks 0', '\n ticks 3', '\n min 2', '\n max 3', '\n showLegend true',
  '%%{init: {"radar": {"width": 200}}}%%\n', 'a very long label of many words', '<b>b</b>', 'fa:fa-car',
];

const HEADLESS = /^\s*(?:axis|curve|ticks|min|max|graticule|showLegend|title|accTitle|accDescr)\b/;

// The inputs of Pele's tests/radar/oracle.test.ts beyond its corpus.
const SEEDS = [
  'radar-beta',
  'radar-beta\naccDescr {}\naxis A',
  'radar-beta\naccDescr {\naxis A\ncurve c{1}',
  'radar-beta\naccDescr {\n}\naccDescr {\naxis A',
  'radar-beta:',
  'radar-beta :',
  'radar-beta:\naxis A, B["Bee"]\ncurve c{1,2}',
  'radar-beta axis A, B curve c{1, 2} ticks 5 max 10',
  'radar-beta\naxis A,B\ncurve c { B: 2, A 1 }\n',
  'radar-beta\naxis A,B\ncurve c { X: 2, A 1 }\n',
  'radar-beta\ncurve c { X: 2 }\n',
  'radar-beta\naxis A\ncurve c{\n\n1,\n\n2\n\n}, d["D"]{ A:3 }\n',
  'radar-beta\naxis A\ncurve c{1, A 2}',
  'radar-beta\naxis A\ncurve c{}',
  'radar-beta\naxis A\ncurve c{1,}',
  'radar-beta\naxis A\ncurve c{1\n,2}',
  'radar-beta\ntitle T\naxis A',
  'radar-beta title T axis A',
  'radar-beta\n title T %% c\n accTitle: at\n accDescr: ad\naxis A',
  'radar-beta\naxis axis',
  'radar-beta\naxis maxi, min-1, _x, 1a',
  'radar-beta\naxis a-, b',
  'radar-beta\naxis true',
  'radar-beta\naxis circles',
  'radar-beta\nshowLegend true, ticks 3, max 1.5, min 0, graticule polygon',
  'radar-beta\nshowLegend TRUE',
  'radar-beta\nticks 5 ticks 6\nticks 7',
  'radar-beta\nticks 05',
  'radar-beta\nmax 1.2.3',
  'radar-beta\nmax 5.',
  'radar-beta\nmax -5',
  'radar-beta\ngraticule circle polygon',
  'radar-beta\naxis A,\nB',
  'radar-beta\naxis A\n,B',
  'radar-beta\ncurve c\n{1}',
  'radar-beta\naxis A [ "x" ]',
  "radar-beta\naxis A['x\\ny']",
  'radar-betax',
  'radar-beta-x',
  'radar-beta:x',
  'radar-beta: axis A',
  'radar-beta%% c\naxis A',
  'radar-beta\naxis A\ncurve A{A 1}, A{A: 2}',
  'radar-beta\naxis A, A\ncurve c{A 1, A 2}',
  'radar-beta\ncurve c{1} axis A',
  '---\ntitle: x\n---\nradar-beta\naxis A',
  '\n\n  radar-beta  \n\n',
  'radar-beta\naccDescr {\n a\n b\n}\naxis A',
  'radar-beta\ntitle T\ntitle U\n\ntitle',
  'radar-beta\naxis A\ncurve c{ A:: 1 }',
  'radar-beta\naxis A\ncurve c{ 1 2 }',
  'radar-beta\nmax 99999999999999999999999999999999999999',
  'radar-beta\nmax 0.10',
  'radar-beta\ncurve c{00}',
  'radar-beta\ncurve c{0.5.}',
  'radar-beta\ncurve c{0000.5, 12.34.5, 100.25}',
  'radar-beta\ntitle x axis A',
  '%%{init: {"theme":"dark"}}%%\nradar-beta\naxis A %% c\ncurve c{1} %% d\n',
];

const config = (lines, body) => `---\nconfig:\n  radar:\n${lines.map((l) => '    ' + l).join('\n')}\n---\n${body}`;
const BASIC = 'radar-beta\n  axis A, B, C, D\n  curve one["First"]{1,2,3,4}\n  curve two{4,3,2,1}';

// Cases written here: what the tokens, the model and the drawing do at their edges.
export const WRITTEN = [
  BASIC,
  BASIC + '\n  graticule polygon\n  ticks 3',
  BASIC + '\n  showLegend false',
  'radar-beta\n axis A, B, C\n curve c{50, 0, 0}\n graticule polygon',
  'radar-beta\n axis A, B, C\n curve c{50, 0, 0}\n max 100\n graticule polygon',
  'radar-beta\n axis A, B, C\n curve c{50, 0, 0}\n max 100\n min 20\n graticule polygon',
  'radar-beta\n axis A, B, C\n curve c{50, 0, 0}\n max 25',
  'radar-beta\n axis A, B, C\n curve c{50, 0, 0}\n min 60\n max 100',
  config(['width: 600', 'height: 600'], BASIC + '\n showLegend false'),
  config(['marginTop: 100', 'marginLeft: 120'], BASIC + '\n showLegend false'),
  config(['marginBottom: 50', 'marginRight: 3000'], BASIC),
  config(['axisScaleFactor: 0.5'], BASIC),
  config(['axisLabelFactor: 1.5'], BASIC),
  config(['curveTension: 0'], BASIC),
  config(['curveTension: 0.5'], BASIC),
  config(['width: -5', 'height: "tall"', 'curveTension: 99', 'marginTop: -1'], BASIC),
  config(['width: 5000', 'height: 0.5', 'axisScaleFactor: 11', 'axisLabelFactor: -1'], BASIC),
  config(['width: .inf', 'height: .nan'], BASIC),
  config(['width: 1', 'height: 1'], BASIC),
  config(['width: 333.3', 'height: 177.7', 'curveTension: 0.33'], BASIC),
  '---\nconfig:\n  radar: 5\n---\n' + BASIC,
  '---\nconfig:\n  radar: [1, 2]\n---\n' + BASIC,
  '%%{init: {"radar": {"width": 200, "height": 400}}}%%\n' + BASIC,
  '%%{init: {"radar": {"width": "200"}}}%%\n' + BASIC,
  'radar-beta\n axis A, B, C\n curve full{1,2,3}\n curve short{1,2}',
  'radar-beta\n axis A, B, C\n curve c{ C: 3, A: 1, B: 2 }',
  'radar-beta\n axis A["Alpha"], B\n curve c{ B: 2 }',
  'radar-beta\n curve c{ B: 2 }',
  'radar-beta\n ticks 3\n ticks 100\n max 5\n max 7\n graticule polygon\n showLegend false',
  '---\ntitle: From front matter\n---\n' + BASIC,
  '---\ntitle: From front matter\n---\n' + BASIC + '\n title Own',
  'radar-beta\n axis a["Top"], b["A very long label on the right side"], c["Bottom"], d["A very long label on the left side"]\n curve c{1,2,3,4}',
  'radar-beta\n axis A, B, C',
  'radar-beta\n axis A\n curve c{5}',
  'radar-beta\n curve c{1,2,3}',
  'radar-beta\n axis A, B\n curve c{1,1}\n min 1\n max 1',
  'radar-beta\n axis A, B\n curve c{1,2}\n min 9\n max 2',
  'radar-beta\n axis A, B\n curve c{0,0}',
  'radar-beta\n axis A, B\n curve c{1,2}\n ticks 0',
  'radar-beta\n axis A, B\n curve c{1,2}\n ticks 0.3\n graticule polygon',
  'radar-beta\n axis A, B\n curve c{1,2}\n ticks 2.5',
  'radar-beta\n axis A, B\n curve c{1,2}\n max ' + '9'.repeat(400),
  'radar-beta\n axis A, B\n curve c{' + '9'.repeat(400) + ',2}',
  'radar-beta\n axis A, B, C\n curve c{' + '9'.repeat(400) + ',2,3}\n max ' + '9'.repeat(400),
  BASIC + '\n accTitle: Name\n accDescr: Text',
  BASIC + '\n accDescr {\n  one\n    two\n }',
  'radar-beta\n axis A, B, C, D, E, F, G, H, I, J, K, L\n curve c{1,2,3,4,5,6,7,8,9,10,11,12}\n curve d{12,11,10,9,8,7,6,5,4,3,2,1}',
  'radar-beta\n axis A, B, C, D, E, F, G\n curve c{1,2,3,4,5,6,7}\n graticule polygon\n ticks 7',
  'radar-beta\n axis A, B\n ' + Array.from({ length: 12 }, (_, i) => `curve c${i}["Curve ${i}"]{${i}, ${12 - i}}`).join('\n '),
  'radar-beta\n axis A["<b>Bold</b> axis"], B["two<br>lines"], C["&amp; #35;"], D["中文标签"], E["😀"]\n curve c["<i>it</i>"]{1,2,3,4,5}',
  'radar-beta\n axis A["fa:fa-car Car"], B\n curve c{1,2}',
  'radar-beta\n axis A[""], B[" "]\n curve c[""]{1,2}, d[" "]{2, 1}',
  'radar-beta\n title A very long title of a radar chart that is far wider than the chart it names, by a long way\n axis A, B, C\n curve c{1,2,3}',
  'radar-beta\n title Wide\n axis A, B, C\n curve c{1,2,3}\n showLegend false',
  'radar-beta\n axis A, B, C\n curve c{-1, 0.5, 2}\n min -2\n max 2',
  'radar-beta\n axis A, B, C\n curve c{0.1, 0.2, 0.3}',
  'radar-beta\n axis A, B, C\n curve c{1e3, 2, 3}',
  'radar-beta\n axis A, B, C, D, E\n curve c{3, 0, 3, 0, 3}\n min 0\n max 3\n ticks 32',
  'radar-beta\n axis ' + Array.from({ length: 40 }, (_, i) => `a${i}`).join(', ') + '\n curve c{' + Array.from({ length: 40 }, (_, i) => (i * 7) % 11).join(', ') + '}',
  'radar-beta\n axis A\n curve c{1}, d{2}',
  'radar-beta\n axis A, B\n curve c{1, 2}\n graticule polygon\n ticks 1',
  'radar-beta\naxis A, B, C\ncurve c["Line one<br>line two"]{1, 2, 3}',
  'Radar-beta\naxis A',
  'radar\naxis A',
  'radar-beta-x\naxis A',
  '',
];

// The corpus the suites start from: documentation, specs, the strings of Pele's tests, its
// seeds and the cases above.
export function corpus(pele) {
  const headless = testStrings(pele, 'tests/compat/radar', /\.test\.[jt]s$/, HEADLESS);
  return [
    ...new Set([
      ...docs(pele, 'radar'),
      ...testStrings(pele, 'tests/compat/radar', /\.spec\.[jt]s$/, /radar/),
      ...testStrings(pele, 'tests/compat/radar', /\.test\.[jt]s$/, /radar/),
      ...headless.map((s) => (s.includes('radar') ? s : `radar-beta\naxis my-axis, ax1, ax2\n${s}`)),
      ...testStrings(pele, 'tests/render', /^radar.*\.test\.ts$/, /radar/),
      ...SEEDS,
      ...WRITTEN,
    ]),
  ].filter((s) => !lone(s) && s.length <= 50000);
}

const NAMES = ['A', 'B', 'C', 'math', 'science', 'english', 'speed', 'power', 'x1', 'my-axis', 'a_b', 'Q', 'r', 'z9'];
const LABELS = [
  'Math', 'Science', 'English', 'History', 'Food Quality', 'Service', 'Price', 'Ambiance', 'Alice', 'Bob', 'A very long label on the side',
  '中文', '😀 emoji', '#35;', '&amp; more', 'two<br>lines', '<b>bold</b>', 'x y', ' spaced ', '', 'WWWWWWWW', 'iiii', 'Line one two three',
];
const TITLES = ['T', 'Grades', 'Restaurant Comparison', 'A title much wider than the chart it names, by a long way indeed', '中文'];

// Random radar charts, most of them well formed, with random configuration. Small ones have
// fewer axes and curves, for a fixture of what is drawn to stay small.
export function randomDiagrams(rnd, small = false) {
  const pick = picker(rnd);
  const chance = (p) => rnd() < p;
  const value = () =>
    chance(0.7)
      ? String(Math.floor(rnd() * 100))
      : pick(['0', '1', '0.5', '2.25', '100', '1000', '3.14159', '0.001', '7.5', '99999', String(Math.floor(rnd() * 1000) / 10)]);
  const label = () => {
    const l = pick(LABELS);
    return l.includes('"') || chance(0.85) ? `["${l}"]` : `['${l}']`;
  };
  return () => {
    const settings = [];
    if (chance(0.5)) {
      if (chance(0.3)) settings.push(`width: ${pick([100, 200, 250, 320, 400, 500, 640, 1, 0, -5, 333.3, 4000, 5000])}`);
      if (chance(0.3)) settings.push(`height: ${pick([100, 200, 250, 320, 400, 600, 1, 0, 217.5])}`);
      for (const side of ['marginTop', 'marginBottom', 'marginLeft', 'marginRight']) {
        if (chance(0.1)) settings.push(`${side}: ${pick([0, 5, 16, 30, 100, 2.5, -1])}`);
      }
      if (chance(0.2)) settings.push(`axisScaleFactor: ${pick([0, 0.25, 0.5, 1, 1.2, 2, 10, 11])}`);
      if (chance(0.2)) settings.push(`axisLabelFactor: ${pick([0, 0.5, 1, 1.05, 1.2, 1.5, 3])}`);
      if (chance(0.2)) settings.push(`curveTension: ${pick([0, 0.1, 0.17, 0.25, 0.5, 1, 2, 0.333])}`);
    }
    let s = '';
    if (chance(0.1)) s += `---\ntitle: ${pick(TITLES)}\n---\n`;
    if (settings.length) {
      if (chance(0.7)) s += `---\nconfig:\n  radar:\n${settings.map((l) => '    ' + l).join('\n')}\n---\n`;
      else s += `%%{init: {"radar": {${settings.map((l) => l.replace(/^(\w+): (.*)$/, '"$1": $2')).join(', ')}}}}%%\n`;
    }
    s += chance(0.9) ? 'radar-beta' : 'radar-beta:';
    s += '\n';
    if (chance(0.3)) s += `  title ${pick(TITLES)}\n`;
    if (chance(0.1)) s += '  accTitle: Accessible\n';
    if (chance(0.1)) s += chance(0.5) ? '  accDescr: Described\n' : '  accDescr {\n  one\n   two\n}\n';
    const count = chance(0.1) ? pick(small ? [1, 2, 7] : [1, 2, 16, 25]) : 3 + Math.floor(rnd() * (small ? 3 : 6));
    const axes = [];
    for (let i = 0; i < count; i++) axes.push(chance(0.05) ? pick(NAMES) : `${pick(NAMES)}${i}`);
    let line = '  axis ';
    axes.forEach((a, i) => {
      if (i > 0) {
        if (chance(0.3)) {
          s += line + '\n';
          line = '  axis ';
        } else line += chance(0.5) ? ', ' : ',';
      }
      line += a + (chance(0.5) ? label() : '');
    });
    s += line + '\n';
    const curves = chance(0.1) ? Math.floor(rnd() * (small ? 10 : 12)) : 1 + Math.floor(rnd() * (small ? 2 : 4));
    for (let k = 0; k < curves; k++) {
      s += `  curve c${k}${chance(0.6) ? label() : ''}{`;
      const n = chance(0.05) ? count - 1 : count;
      if (chance(0.25)) {
        const order = axes.map((a) => a).sort(() => rnd() - 0.5);
        s += order.slice(0, chance(0.05) ? n - 1 : n).map((a) => `${a}${chance(0.5) ? ':' : ''} ${value()}`).join(', ');
      } else {
        s += Array.from({ length: n }, value).join(chance(0.5) ? ', ' : ',');
      }
      s += '}\n';
    }
    if (chance(0.3)) s += `  graticule ${pick(['circle', 'polygon'])}\n`;
    if (chance(0.3)) s += `  ticks ${pick([0, 1, 2, 3, 5, 8, 10, 32, 50, 2.5])}\n`;
    if (chance(0.3)) s += `  max ${pick([1, 5, 10, 50, 100, 150, 0.5, 1000])}\n`;
    if (chance(0.2)) s += `  min ${pick([0, 1, 5, 10, 20, 0.25])}\n`;
    if (chance(0.2)) s += `  showLegend ${pick(['true', 'false'])}\n`;
    return s;
  };
}

export default async function ({ src, pele, write, random }) {
  const located = await locator(src);
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { radar } = await src('diagrams/radar/index.ts');
  const { parseRadar, RADAR_TOKENS, lexRadar } = await src('diagrams/radar/parser.ts');

  const caught = (e) => {
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  const opt = (x) => (x === undefined || x === '' ? undefined : x);
  const tokens = (source) => {
    try {
      const t = lexRadar(source);
      return { tokens: t.kinds.slice(0, -1).map((k, i) => [RADAR_TOKENS[k].name, source.slice(t.starts[i], t.ends[i])]) };
    } catch (e) {
      return { tokens: null };
    }
  };
  const direct = (source) => {
    try {
      const ast = parseRadar(source);
      return {
        ...tokens(source),
        ast: {
          title: enc(ast.title),
          accTitle: enc(ast.accTitle),
          accDescr: enc(ast.accDescr),
          axes: ast.axes.map((a) => [a.name, enc(a.label)]),
          curves: ast.curves.map((c) => [c.name, enc(c.label), c.entries.map((e) => [enc(e.axis?.$refText), enc(e.value)])]),
          options: ast.options.map((o) => [o.name, enc(o.value)]),
        },
      };
    } catch (e) {
      return { ...tokens(source), ...caught(e) };
    }
  };
  // Pele's parseSource for a radar chart, and the raw text given to the parser alone.
  const run = (source) => {
    const out = { src: source, direct: direct(source) };
    let pre;
    try {
      pre = preprocess(source);
      out.type = detect(pre.text);
    } catch (e) {
      return { ...out, ...caught(e) };
    }
    if (out.type !== 'radar') return out;
    try {
      const m = located(source, false, () => radar.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title));
      out.model = {
        title: opt(m.title),
        accTitle: opt(m.accTitle),
        accDescr: opt(m.accDescr),
        axes: m.axes.map((a) => [a.name, a.label]),
        curves: m.curves.map((c) => [c.name, c.label, c.entries.map((v) => enc(v))]),
        options: [m.options.showLegend, enc(m.options.ticks), enc(m.options.max), enc(m.options.min), m.options.graticule],
      };
    } catch (e) {
      Object.assign(out, caught(e));
    }
    return out;
  };

  const base = corpus(pele);
  const all = [...base];
  const seen = new Set(all);
  const add = (s) => {
    if (s.length <= 4000 && !lone(s) && !seen.has(s)) {
      seen.add(s);
      all.push(s);
    }
  };
  const diagrams = randomDiagrams(random(31));
  for (let i = 0; i < 1500; i++) add(diagrams());
  // Mutated text; one in six is cut short, as a statement cut off by the end of the text ends
  // on a different error than one cut off by a line break.
  const rnd = random(7);
  const next = mutator([...seen].filter((s) => s.length <= 4000), FRAGMENTS, rnd);
  for (let i = 0; i < 4000; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  // Random strings of tokens, where the order of the token types and their ends matter.
  const trnd = random(8);
  const pieces = [...new Set([...FRAGMENTS, ...FRAGMENTS.map((f) => f.trim()).filter((f) => f !== '')])];
  const openers = ['radar-beta\n', 'radar-beta ', 'radar-beta:', 'radar-beta\naxis ', 'radar-beta\ncurve c{', 'radar-beta\naxis A\ncurve c{A', 'radar-beta\nmax ', 'radar-beta\naccDescr', ''];
  for (let i = 0; i < 1200; i++) {
    let s = openers[Math.floor(trnd() * openers.length)];
    for (let k = 1 + Math.floor(trnd() * 8); k > 0; k--) s += pieces[Math.floor(trnd() * pieces.length)] + (trnd() < 0.3 ? ' ' : '');
    add(s);
  }
  // Pele's stress cases, made smaller, which the matchers without backtracking are for.
  const N = 3000;
  for (const s of [
    'radar-beta\ncurve c{' + '0'.repeat(N) + '}',
    'radar-beta\nmax ' + '0'.repeat(N) + '.5',
    'radar-beta\naxis a\ncurve c{1}\nmax ' + '9'.repeat(N),
    'radar-beta\ncurve c{' + '1.2'.repeat(N / 3) + '}',
    'radar-beta\naxis a' + '-'.repeat(N),
    'radar-beta\n' + 'axis'.repeat(N / 4),
    'radar-beta\n' + 'accDescr {'.repeat(N / 10),
    'radar-beta\naccDescr {' + ' a\n'.repeat(N / 3),
    'radar-beta\naxis a["' + 'x'.repeat(N),
    'radar-beta\naxis a' + '['.repeat(N / 2),
    'radar-beta\n' + '%%{\n'.repeat(N / 4),
    'radar-beta\n' + '---\n'.repeat(N / 4),
    'radar-beta\n' + ' '.repeat(N) + 'axis a',
    'radar-beta\n' + '0.'.repeat(N / 2),
    'radar-beta\nmax ' + '1'.repeat(N) + '.' + '2'.repeat(N) + '.',
  ]) {
    seen.delete(s);
    add(s);
  }
  write('diagram/radar/testdata/pele/radar.json', { cases: all.map(run) });
}
