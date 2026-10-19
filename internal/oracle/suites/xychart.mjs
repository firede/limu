// XY charts parsed as Pele parses them, from text to tokens and to XyChartModel: Mermaid's
// documentation and specs, the inputs of Pele's own tests, cases written here, those mutated at
// random, charts built from random statements, and random strings of tokens. Each case is lexed
// and parsed as the raw text, and parsed as Pele's parse does it, after the preprocessing every
// diagram goes through.
import { join } from 'node:path';
import { enc, peleError } from '../lib/value.mjs';
import { fileLiterals, testFiles, docs, arrayIn, lone, picker } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';

// The text Mermaid's broken rule for markdown strings matches, as Pele's lexer has it.
export const BROKEN =
  '`)' + ' '.repeat(36) + '{ this.pushstate(md_string); }\n<md_string>(?:(?!`").)+' + ' '.repeat(18) + '{ return md_str; }\n<md_string>(?:`';

// Pele's fragments, tests/xychart/oracle.test.ts, and some for what Pele draws.
export const FRAGMENTS = [
  ' ', '  ', '\n', '\r\n', '\r', ' \n', '\n ', '\t', ';', ':', '[', ']', '[ ', ' ]', ',', ', ', '1', '23', '4.5', '.5',
  '5.', '-1', '+2', '-.3', '1.2.3', '012', '-', '--', '-->', ' --> ', '->', '>', '"', '""', '"a b"', '"`', '`"', '`',
  '`)', '%%', '%% c', 'a%%b', '}%%', '%%{init: {}}%%', 'xychart', 'xychart-beta', 'XYChart ', 'xychart-', 'xycharts',
  'vertical', 'horizontal', ' Horizontal ', 'verticals', 'x-axis ', 'y-axis ', 'X-Axis', 'x-axis_', 'x-axi', 'line ',
  'bar ', 'Line', 'BAR', 'lines', 'bars', 'bar1', 'line [1]', 'bar [2, 3]', 'title', 'title ', 'Title x', 'titles',
  'accTitle: ', 'accTitle :', 'accTitle', 'accDescr: ', 'accDescr { ', 'accDescr{', '}', '{', '#', '_', '&', '+', '=',
  '*', '.', '!', '$', '%', "'", '?', '\\', '/', '(', ')', '<', '@', '|', '~', '^', 'A', 'b', 'cat', 'é', ' ',
  '　', '\v', '\f', '\u0001', 'x', 'y', 'l', 'v', 'h', '#35;', BROKEN, BROKEN.toUpperCase(), BROKEN.slice(0, 40),
  '😀', '中文', '<br>', '<b>b</b>', '&amp;', '0.1', '1e3', '99999999999999999999', '-0', '0.000001', '1000000', '7.25',
  'x\raxis', 'x-axis\r', 'y\r-axis', '\r\n\r\n',
];

const BAND = 'x-axis [jan, feb, mar, apr, may, jun, jul, aug, sep, oct, nov, dec]';
const SALES = '5000, 6000, 7500, 8200, 9500, 10500, 11000, 10200, 9200, 8500, 7000, 6000';
const configured = (body, chart) => `---\nconfig:\n  xyChart:\n${body}\n---\n${chart}`;

// The inputs of Pele's tests/xychart/oracle.test.ts beyond its corpus.
export const SEEDS = [
  'xychart\n  title "T"\n  accTitle: a title\n  accDescr: a description\n  line [1, 2, 3]',
  'xychart horizontal\n  accDescr {\n    many\n    lines\n  }\n  x-axis months [jan, "feb", mar]\n  bar "a" [1, 2, 3]\n  line b [3 "x", 2, 1]',
  'xychart;title a-b_c;x-axis 1 --> 10;y-axis "y" -5 --> +5.5;line [1,2];bar [.5]',
  'xychart-beta\n%% comment\n  x-axis x 0 --> 1 %% trailing\n  y-axis 0 --> 1   \n  line [0.1, 0.9]   \n  bar [0.5, 0.2]',
  'xychart\n  x-axis "title only"\n  y-axis title\n  line "s" [1]\n',
  `xychart\nx-axis a ${BROKEN} [b]\nline [1]`,
  `xychart\nline [1 ${BROKEN} 2]\nbar [3]`,
];

// Cases written here, for the corners of the grammar and of the drawing.
export const WRITTEN = [
  '',
  'xychart',
  'xychart\n',
  'xychart vertical',
  'xychart horizontal\nline [1]',
  'xychart verticalx\nline [1]',
  'xychart\nxychart\nline [1]',
  'xychart\n  title\n  line [1]',
  'xychart\n  title "a" b\n  line [1]',
  'xychart\n  title a b c\n  line [1, 2]',
  'xychart\n  title "  spaced  "\n  line [1, 2]',
  'xychart\n  title ""\n  line [1, 2]',
  'xychart\n  title "unclosed\n  line [1, 2]',
  'xychart\n  x-axis\n  line [1, 2]',
  'xychart\n  x-axis []\n  line [1, 2]',
  'xychart\n  x-axis [a,]\n  line [1, 2]',
  'xychart\n  x-axis [a b, c d]\n  line [1, 2]',
  'xychart\n  x-axis ["a b", "c d"]\n  line [1, 2]',
  'xychart\n  x-axis 5 --> 5\n  line [1, 2, 3]',
  'xychart\n  x-axis 10 --> 0\n  line [1, 2, 3]',
  'xychart\n  x-axis -1.5 --> +2.5\n  line [1, 2, 3, 4]',
  'xychart\n  x-axis 0 --> 1\n  line [1, 2, 3]',
  'xychart\n  x-axis 0 --> 0.3\n  line [1, 2, 3, 4]',
  'xychart\n  x-axis "t" 1 -->\n  line [1]',
  'xychart\n  x-axis 1 --> 2 --> 3\n  line [1]',
  'xychart\n  y-axis\n  line [1, 2]',
  'xychart\n  y-axis "t"\n  y-axis 0 --> 10\n  line [1, 2]',
  'xychart\n  y-axis 0 --> 10\n  y-axis "t"\n  line [1, 2]',
  'xychart\n  y-axis 10 --> 0\n  bar [1, 2]',
  'xychart\n  y-axis 5 --> 5\n  bar [1, 2]',
  'xychart\n  y-axis [a, b]\n  bar [1, 2]',
  'xychart\n  line\n  [1, 2]',
  'xychart\n  line [1,\n 2]',
  'xychart\n  line [1 \n, 2]',
  'xychart\n  line []',
  'xychart\n  line [a]',
  'xychart\n  line [1 "a" "b"]',
  'xychart\n  line [1 "", 2 " "]',
  'xychart\n  line [1 " p ", 2]',
  'xychart\n  bar [1 "p", 2 "q"]',
  'xychart\n  line "s" [1]\n  line "s" [2]',
  'xychart\n  line [1, 2, 3, 4, 5, 6]\n  x-axis [a, b]',
  'xychart\n  x-axis [a, b, c, d]\n  bar [1, 2]\n  line [1, 2]',
  'xychart\n  x-axis [a, b]\n  bar [1, 2, 3, 4]',
  'xychart\n  bar [1, 2]\n  bar [3, 4, 5]\n  line [6]',
  'xychart\n  line [7]',
  'xychart\n  line [-0, 0]',
  'xychart\n  bar [0, 0, 0]',
  'xychart\n  line [0.0000001, 0.0000002]',
  'xychart\n  bar [99999999999999999999, 1]\n  line [-99999999999999999999]',
  `xychart\n  line [${'9'.repeat(400)}, 1]`,
  `xychart\n  y-axis -${'9'.repeat(400)} --> ${'9'.repeat(400)}\n  bar [1, 2]`,
  'xychart\n  accTitle:\n  accDescr:\n  line [1]',
  'xychart\n  accTitle: a\n  accDescr {\n    b\n      c\n  }\n  line [1]',
  'xychart\n  accDescr {',
  'xychart\n  accDescr {}line [1]',
  'xychart\n  accTitle',
  'xychart\n  title T\n',
  'xychart\n  line [1]\n  title Late',
  '---\ntitle: Front\n---\nxychart\n  line [1, 2]',
  '---\ntitle: Front\n---\nxychart\n  title Own\n  line [1, 2]',
  'xychart\n  title a #35; b &amp; c\n  line [1]',
  'xychart\n  title "a #35; b"\n  line [1]',
  'xychart\n  x-axis ["#quot;q#quot;", "<b>b</b>", "fa:fa-car x"]\n  bar [1, 2, 3]',
  'xychart\r\n  title Windows\r\n  line [1, 2]\r\n',
  'xychart\r  line [1, 2]\r',
  'xychart\n  line [1, 2]\r\n  bar [3, 4]',
  'xychart\n  line [1, 2]  \n  bar [3, 4]',
  'xychart\n  x-axis [a, b]  \n  bar [3, 4]',
  'xychart\n  x-axis [a, b]\r  bar [3, 4]',
  'XYCHART\n  TITLE T\n  X-AXIS [A, B]\n  Y-AXIS 0 --> 9\n  BAR [1, 2]\n  LINE [2, 1]',
  'xychart\n  line [1]%%x\n  bar [2]',
  'xychart\n  line [1] %% c\n  bar [2]',
  'xychart\n  x-axis x [a]\n  bar [1]',
  'xychart\n  x-axis [é]\n  line [1]',
  'xychart\n  title 😀\n  line [1]',
  'xychart\n  line "😀%%" [1]',
  '\n\n  xychart\n\n  line [1]',
  configured('    width: 900\n    height: 400', `xychart\n  title "Sales Revenue"\n  ${BAND}\n  bar [${SALES}]`),
];

// Pele's hostile templates and extreme numbers, tests/render/xychart-security.test.ts.
export function hostile(pele) {
  const file = join(pele, 'tests/render/xychart-security.test.ts');
  const payloads = arrayIn(join(pele, 'tests/support/inert.ts'), 'export const PAYLOADS = [');
  const templates = arrayIn(file, 'const TEMPLATES: ((p: string) => string)[] = [');
  const out = [];
  for (const t of templates) for (const p of payloads) out.push(t(p));
  out.push(...arrayIn(file, 'const charts = [', { huge: '9'.repeat(400) }, '\n    ];'));
  for (const value of ['1e999', '-1', '0', '0.0001', '99999999', 'NaN', '.inf', '-.inf', '[1]', '{a: 1}', 'null']) {
    out.push(`---\nconfig:\n  xyChart:\n    width: ${value}\n    height: ${value}\n    showDataLabel: true\n---\nxychart\n  x-axis x [a, b]\n  bar "s" [1, 2]`);
  }
  return out.filter((s) => !lone(s));
}

// The corpus the suites start from: documentation, specs, Pele's tests and the cases above.
export function corpus(pele) {
  const out = new Set();
  for (const file of testFiles(pele, 'tests/compat/xychart', /\.spec\.[jt]s$/)) for (const s of fileLiterals(file, /xychart/i)) out.add(s);
  for (const s of docs(pele, 'xychart')) out.add(s);
  for (const s of fileLiterals(join(pele, 'tests/render/xychart.test.ts'), /xychart/i)) out.add(s);
  for (const s of SEEDS) out.add(s);
  for (const s of WRITTEN) out.add(s);
  return [...out].filter((s) => !lone(s) && s.length <= 4000);
}

// Charts of random statements, most of which get through the grammar.
export function statementDiagrams(rnd) {
  const pick = picker(rnd);
  const word = () => pick(['a', 'Sales', '"Revenue (in $)"', '"x y"', 'b_c', '"é"', '"😀"', 'a-b', '12', '"<br>"', 'x+y', '"#35;"']);
  const num = () => pick(['0', '1', '-1', '2.5', '.5', '+3', '100', '-0.25', '1000', '7']);
  const nums = () => Array.from({ length: Math.floor(rnd() * 6) }, () => num() + (rnd() < 0.2 ? ` ${pick(['"p"', '"a b"', '""'])}` : ''));
  const statements = [
    () => `title ${word()}`,
    () => `x-axis ${rnd() < 0.5 ? word() + ' ' : ''}[${Array.from({ length: 1 + Math.floor(rnd() * 5) }, word).join(', ')}]`,
    () => `x-axis ${rnd() < 0.5 ? word() + ' ' : ''}${num()} --> ${num()}`,
    () => `y-axis ${rnd() < 0.5 ? word() + ' ' : ''}${num()} --> ${num()}`,
    () => `y-axis ${word()}`,
    () => `${pick(['line', 'bar'])} ${rnd() < 0.5 ? word() + ' ' : ''}[${nums().join(', ')}]`,
    () => `${pick(['line', 'bar'])} [${nums().join(', ')}]`,
    () => `accTitle: ${word()}`,
    () => `accDescr: ${word()}`,
    () => `accDescr {\n  ${word()}\n   ${word()}\n}`,
    () => pick(['%% c', '', '  ', 'xychart', ';', 'horizontal', 'title']),
  ];
  return () => {
    let src = pick(['xychart\n', 'xychart-beta\n', 'xychart horizontal\n', 'xychart vertical\n', '  xychart\n', 'XYCHART\n']);
    for (let k = 1 + Math.floor(rnd() * 7); k > 0; k--) src += pick(['', '  ', '\t']) + pick(statements)() + pick(['\n', '\n', '\r\n', '\n\n', ';']);
    return src;
  };
}

// Whole charts that Pele draws: categories or numbers on the x axis, ranges set or taken from
// the data, bars and lines of every sign and size, labelled points, titles and labels of every
// kind of text, and the settings of front matter and directives.
export function drawableDiagrams(rnd) {
  const pick = picker(rnd);
  const chance = (p) => rnd() < p;
  const titles = ['Sales Revenue', 'T', '"A title long enough to be wider than the chart it names, and to wrap over lines"', '"#35; &amp; <b>bold</b>"',
    '"中文的标题"', '"a<br>b"', 'Q1 results', '"😀 emoji"'];
  const axisTitles = ['Month', '"Revenue (in $)"', 'x', '"A rather long axis title that may not fit"', '"Temperature °C"', '"a<br>b"', '"<i>it</i>"'];
  const categoryNames = ['jan', 'feb', 'mar', 'apr', 'may', 'jun', 'jul', 'aug', 'sep', 'oct', 'nov', 'dec', '"Q1 2024"',
    '"A long category name"', '"Supercalifragilisticexpialidocious"', '"中文"', '"a<br>b"', 'x', '"#quot;q#quot;"', '"fa:fa-car x"', '"a b c d e f g"'];
  const seriesNames = ['', '"2024"', '"Revenue"', 'Sales', '"A long series name that wraps in the legend"', '"中文"', '"<b>b</b>"', 'a'];
  const value = (scale) => {
    const r = rnd();
    if (r < 0.1) return pick(['0', '-0', '1', '-1']);
    if (r < 0.15) return pick(['99999999999999999999', '0.0000001', '1e3', '123456789', '-123456789', '0.30000000000000004']);
    const v = (rnd() * 2 - (chance(0.3) ? 1 : 0)) * scale;
    return chance(0.5) ? String(Math.round(v)) : v.toFixed(Math.floor(rnd() * 4));
  };
  const config = () => {
    const lines = [];
    if (chance(0.3)) lines.push(`    width: ${pick(['900', '400', '200', '50', '1200.5', '5000', '0', '-5', 'wide', '320'])}`);
    if (chance(0.3)) lines.push(`    height: ${pick(['600', '300', '150', '99', '700.7', '9000', '0', 'tall'])}`);
    if (chance(0.25)) lines.push(`    chartOrientation: ${pick(['horizontal', 'vertical', 'other', '3'])}`);
    if (chance(0.35)) lines.push(`    showDataLabel: ${pick(['true', 'false', '1', 'yes'])}`);
    if (chance(0.15)) lines.push(`    showDataLabelOutsideBar: ${pick(['true', 'false'])}`);
    if (chance(0.15)) lines.push(`    showTitle: ${pick(['true', 'false', '0'])}`);
    if (chance(0.15)) lines.push(`    showLegend: ${pick(['true', 'false'])}`);
    for (const axis of ['xAxis', 'yAxis']) {
      if (!chance(0.25)) continue;
      if (chance(0.1)) {
        lines.push(`    ${axis}: ${pick(['3', '[1]', 'x'])}`);
        continue;
      }
      lines.push(`    ${axis}:`);
      for (const key of ['showLabel', 'showTitle', 'showTick', 'showAxisLine']) {
        if (chance(0.4)) lines.push(`      ${key}: ${pick(['true', 'false', 'false', 'no'])}`);
      }
      if (lines[lines.length - 1].endsWith(':')) lines.push('      showLabel: true');
    }
    return lines;
  };
  return () => {
    let head = '';
    const lines = config();
    if (lines.length > 0 && chance(0.85)) {
      head = '---\n';
      if (chance(0.2)) head += `title: ${pick(['Front', 'A title from the front matter'])}\n`;
      head += `config:\n  xyChart:\n${lines.join('\n')}\n---\n`;
    } else if (chance(0.1)) {
      head = `%%{init: {"xyChart": {"width": ${pick(['500', '800'])}, "showDataLabel": ${pick(['true', 'false'])}, "chartOrientation": "${pick(['horizontal', 'vertical'])}"}}}%%\n`;
    }
    const out = [pick(['xychart', 'xychart', 'xychart-beta', 'xychart horizontal', 'xychart vertical'])];
    if (chance(0.6)) out.push(`  title ${pick(titles)}`);
    if (chance(0.1)) out.push('  accTitle: Acc', '  accDescr: Description');
    const scale = pick([1, 10, 100, 1000, 10000, 0.01, 1e6]);
    let count = 1 + Math.floor(rnd() * (chance(0.15) ? 30 : 10));
    const axis = rnd();
    if (axis < 0.6) {
      const names = Array.from({ length: count }, (_, i) => (chance(0.7) ? pick(categoryNames) : `c${i}`));
      out.push(`  x-axis ${chance(0.6) ? pick(axisTitles) + ' ' : ''}[${names.join(', ')}]`);
      if (chance(0.15)) count += Math.floor(rnd() * 3) - 1;
    } else if (axis < 0.8) {
      out.push(`  x-axis ${chance(0.5) ? pick(axisTitles) + ' ' : ''}${value(100)} --> ${value(100)}`);
    } else if (axis < 0.85) {
      out.push(`  x-axis ${pick(axisTitles)}`);
    }
    if (chance(0.4)) {
      const lo = value(scale);
      const hi = chance(0.8) ? value(scale) : lo;
      out.push(`  y-axis ${chance(0.6) ? pick(axisTitles) + ' ' : ''}${lo} --> ${hi}`);
    } else if (chance(0.2)) {
      out.push(`  y-axis ${pick(axisTitles)}`);
    }
    const series = 1 + Math.floor(rnd() * 4);
    for (let s = 0; s < series; s++) {
      const kind = pick(['bar', 'line', 'line']);
      const n = Math.max(1, count + (chance(0.15) ? Math.floor(rnd() * 5) - 2 : 0));
      const values = Array.from({ length: n }, () => {
        const v = value(scale);
        return kind === 'line' && chance(0.15) ? `${v} ${pick(['"Launch"', '"p"', '"A long point label"', '""', '"中"', '"<b>x</b>"'])}` : v;
      });
      out.push(`  ${kind} ${pick(seriesNames)} [${values.join(', ')}]`.replace(/ {2,}\[/, ' ['));
    }
    return head + out.join('\n') + pick(['', '\n']);
  };
}

const num = (v) => enc(v);

// A model as the oracle writes it, its fields in order, a string left undefined left out.
function dumpModel(m) {
  const axis = (a) =>
    a.type === 'band' ? { type: a.type, title: a.title, categories: a.categories } : { type: a.type, title: a.title, min: num(a.min), max: num(a.max) };
  return {
    title: m.title || undefined,
    accTitle: m.accTitle || undefined,
    accDescr: m.accDescr || undefined,
    orientation: m.orientation ?? null,
    xAxis: axis(m.xAxis),
    yAxis: axis(m.yAxis),
    hasSetXAxis: m.hasSetXAxis,
    hasSetYAxis: m.hasSetYAxis,
    plots: m.plots.map((p) => ({
      type: p.type,
      title: p.title,
      data: p.data.map(([x, v]) => [x, v === undefined ? { $: 'undefined' } : num(v)]),
      pointLabels: p.pointLabels ?? null,
    })),
  };
}

// Each case's tokens, with where each starts in UTF-16 code units, and its model or error.
export default async function ({ src, pele, write, random }) {
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { xychart } = await src('diagrams/xychart/index.ts');
  const { tokenize } = await src('diagrams/xychart/lexer.ts');
  const { parseXyChart } = await src('diagrams/xychart/parser.ts');
  const { XyChartDb } = await src('diagrams/xychart/db.ts');

  const caught = (e) => {
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  const parsed = (parseIt) => {
    try {
      return { model: dumpModel(parseIt()) };
    } catch (e) {
      return caught(e);
    }
  };
  const run = (source) => {
    const t = tokenize(source);
    const out = {
      src: source,
      tokens: t.types.map((type, i) => [type, t.texts[i], t.starts[i]]),
      direct: parsed(() => {
        const db = new XyChartDb();
        parseXyChart(source, db);
        return db;
      }),
    };
    let pre;
    try {
      pre = preprocess(source);
      out.type = detect(pre.text);
    } catch (e) {
      return { ...out, ...caught(e) };
    }
    if (out.type !== 'xychart') return out;
    const piped = parsed(() => xychart.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title));
    // Most text comes out of the pipeline as it went in, and parses the same.
    if (JSON.stringify(piped) === JSON.stringify(out.direct)) return { ...out, same: true };
    return { ...out, ...piped };
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
  for (const s of hostile(pele)) add(s);
  // Mutated text; one in six is cut short.
  const rnd = random(7);
  const next = mutator(base, FRAGMENTS, rnd);
  for (let i = 0; i < 3500; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  const diagrams = statementDiagrams(random(10));
  for (let i = 0; i < 1500; i++) add(diagrams());
  const drawable = drawableDiagrams(random(12));
  for (let i = 0; i < 500; i++) add(drawable());
  // Random strings of tokens, where keyword boundaries and the lookahead matter.
  const trnd = random(8);
  const pieces = [...new Set([...FRAGMENTS, ...FRAGMENTS.map((f) => f.trim()).filter((f) => f !== '')])];
  const openers = ['xychart\n', 'xychart', 'xychart\n  line [', 'xychart\n  x-axis ', 'xychart\n  x-axis [', 'xychart\n  y-axis ', 'xychart\n  bar ', 'xychart\n  accDescr', ''];
  for (let i = 0; i < 1500; i++) {
    let s = openers[Math.floor(trnd() * openers.length)];
    for (let k = 1 + Math.floor(trnd() * 10); k > 0; k--) s += pieces[Math.floor(trnd() * pieces.length)] + (trnd() < 0.3 ? ' ' : '');
    add(s);
  }

  write('diagram/xychart/testdata/pele/xychart.json', { cases: all.map(run) });
}
