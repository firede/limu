// Quadrant charts parsed as Pele parses them, from text to tokens and the model: the sources
// of Mermaid's specs and documentation as Pele's tests collect them, the seeds of Pele's
// tests/quadrant/oracle.test.ts, cases written here, those sources mutated at random, charts
// built at random, and random strings of tokens.
import { enc, peleError } from '../lib/value.mjs';
import { lone, picker, testStrings, docs } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';

// The fragments of Pele's tests/quadrant/oracle.test.ts, and more.
export const FRAGMENTS = [
  ' ', '  ', '\n', '\r\n', '\t', ';', ':', '::', ':::', ': [', ':[', ' : [ ', '[', ']', ' ] ', ',', ', ', '0', '1',
  '0.5', '0.25', '1.5', '0,5', '0x5', '05', '055', '10', '.', '-', '--', '-->', '--->', ' --> ', '->', '>', '"', '"`',
  '`"', '`', '"a"', '"`md`"', '%%', '%% c', 'a%%b', '}%%', '%%{init: {}}%%', 'quadrantChart', 'QuadrantChart ',
  'quadrant-1 ', 'quadrant-2 ', 'quadrant-3 ', 'quadrant-4 ', 'quadrant-5', 'quadrant-', 'quadrant', 'x-axis ',
  'y-axis ', ' X-Axis ', 'x-axi', 'title', 'title ', 'Title x', 'titles', 'accTitle: ', 'accTitle :', 'accTitle',
  'accDescr: ', 'accDescr { ', 'accDescr{', '}', '{', 'classDef ', 'classDef a ', 'classDefs', 'class1', 'radius: 10',
  'color: #ff0000', 'stroke-color: #f0f', 'stroke-width: 5px', ' ,', '#', '_', '&', '+', '=', '*', '!', '$', '%', "'",
  '?', '\\', '/', '(', ')', '<', '>', '@', '|', '~', '^', 'A', 'b', 'Point', 'é', '需', '\u00a0', '\u3000', '\v', '\f',
  '\r', '\u0001', 'x', 'y', 'q', 't', 'c', '#35;',
  // More, for the edges of the lexer and the styles.
  'x\raxis', 'x-\raxis', '0\u00e95', '0😀5', '😀', '😀%%', '\u2028', '\u2029', '\ufeff', 'radius: 0', 'radius: 3000',
  'radius: x', 'radius:', 'color: abc', 'color: #abcd', 'stroke-width: 2', 'stroke-width: 0px', 'size: 4', ':::c',
  '"`**b** _i_`"', '"`a\nb`"', '<br>', '<b>x</b>', 'fa:fa-car', '&amp;', 'x-axis a --> b', 'y-axis a -->',
  '---\ntitle: T\n---\n', '0.', '0.0', '0.99', '1.0', '0e5', '0b1', '0o7', '0-5', '0 5', '0\t5',
];

// The seeds of Pele's tests/quadrant/oracle.test.ts.
export const SEEDS = [
  'quadrantChart\n  title T\n  accTitle: a title\n  accDescr: a description\n  A: [0.1, 0.2]',
  'quadrantChart\n  accDescr {\n    many\n    lines\n  }\n  x-axis "`left`" --> "`right`"\n  "`**P**`":::c: [1, 0] radius: 3',
  'quadrantChart;quadrant-1 a;quadrant-2 b;A: [0.5, 0.5];',
  'quadrantChart\n%% comment\n  A: [0.3, 0.6] %% trailing\n  B:::k: [ 0 , 1 ] color: #abc , radius : 4\n  classDef k stroke-width: 2px',
  '  \n ;quadrantChart x-axis a-b --> c d\n y-axis "only"\n',
];

// Cases written here: the edges of the lexer and the parser, the styles and their errors, and
// what the layout makes of labels, points and settings of every kind.
export const WRITTEN = [
  '',
  'quadrantChart',
  'quadrantChart\n',
  'quadrantcharts',
  'QUADRANTCHART\nA: [0.5, 0.5]',
  'quadrantChart x',
  'quadrantChart\ntitle',
  'quadrantChart\ntitle ',
  'quadrantChart\ntitle\nA: [0.5, 0.5]',
  'quadrantChart\ntitleX',
  'quadrantChart\ntitle_x',
  'quadrantChart\ntitle  spaced  title  ',
  'quadrantChart\ntitle a;b',
  'quadrantChart\naccTitle',
  'quadrantChart\naccTitle:',
  'quadrantChart\naccTitle  :  spaced  ',
  'quadrantChart\naccDescr:',
  'quadrantChart\naccDescr {',
  'quadrantChart\naccDescr {}',
  'quadrantChart\naccDescr {a}',
  'quadrantChart\naccDescr {\n  one\n\n   two\n}\nA: [0.5, 0.5]',
  'quadrantChart\naccDescr\n{\nx\n}',
  'quadrantChart\naccTitle {x}',
  'quadrantChart\nx-axis',
  'quadrantChart\nx-axis ',
  'quadrantChart\nx-axis Low',
  'quadrantChart\nx-axis Low -->',
  'quadrantChart\nx-axis Low --> ',
  'quadrantChart\nx-axis Low --> High',
  'quadrantChart\nx-axis Low ---> High',
  'quadrantChart\nx-axis Low -> High',
  'quadrantChart\nx-axis Low-->High',
  'quadrantChart\nx-axis "Low" --> "High"',
  'quadrantChart\nx-axis "`**Low**`" --> "`_High_`"',
  'quadrantChart\nx-axis --> High',
  'quadrantChart\nx-axis a - b --> c - d',
  'quadrantChart\nX-AXIS a --> b\nY-AXIS c --> d',
  'quadrantChart\ny-axis Bottom --> Top',
  'quadrantChart\ny-axis Bottom',
  'quadrantChart\ny-axis Bottom -->',
  'quadrantChart\ny-axis "" --> ""',
  'quadrantChart\nx-axis ""\ny-axis ""',
  'quadrantChart\nx-axis\ry',
  'quadrantChart\nx\raxis a',
  'quadrantChart\nquadrant-1',
  'quadrantChart\nquadrant-1 ',
  'quadrantChart\nquadrant-1 One\nquadrant-2 Two\nquadrant-3 Three\nquadrant-4 Four',
  'quadrantChart\nquadrant-5 Five',
  'quadrantChart\nquadrant-0 Zero',
  'quadrantChart\nquadrant-1 "`**bold** and _italic_`"',
  'quadrantChart\nquadrant-1 "a" b',
  'quadrantChart\nquadrant-1 a "b"',
  'quadrantChart\nquadrant-1 a\nquadrant-1 b',
  'quadrantChart\nquadrant-1 a-b c_d e.f g#h i&j k*l m+n o=p q,r s!t u$v w\'x y?z',
  'quadrantChart\nquadrant-1 é 中文 😀',
  'quadrantChart\nquadrant-1 "unclosed',
  'quadrantChart\nquadrant-1 "`unclosed',
  'quadrantChart\nquadrant-1 "`a"',
  'quadrantChart\nquadrant-1 "`a`',
  'quadrantChart\nquadrant-1 "`a``"',
  'quadrantChart\nquadrant-1 ""',
  'quadrantChart\nquadrant-1 "`"',
  'quadrantChart\nA',
  'quadrantChart\nA:',
  'quadrantChart\nA: [',
  'quadrantChart\nA: [0.5',
  'quadrantChart\nA: [0.5,',
  'quadrantChart\nA: [0.5, 0.5',
  'quadrantChart\nA: [0.5, 0.5]',
  'quadrantChart\nA: [0.5, 0.5] ',
  'quadrantChart\nA: [0.5,0.5]',
  'quadrantChart\nA:[0.5,0.5]',
  'quadrantChart\nA : [ 0.5 , 0.5 ]',
  'quadrantChart\nA\n:\n[\n0.5\n,\n0.5\n]',
  'quadrantChart\nA: [1, 1]\nB: [0, 0]\nC: [1, 0]\nD: [0, 1]',
  'quadrantChart\nA: [1.5, 0.5]',
  'quadrantChart\nA: [0.5, 2]',
  'quadrantChart\nA: [10, 0.5]',
  'quadrantChart\nA: [0x5, 0.5]',
  'quadrantChart\nA: [0b1, 0o1]',
  'quadrantChart\nA: [0e5, 0.5]',
  'quadrantChart\nA: [0,5, 0.5]',
  'quadrantChart\nA: [0é5, 0.5]',
  'quadrantChart\nA: [0😀5, 0.5]',
  'quadrantChart\nA: [0.' + '5'.repeat(40) + ', 0.5]',
  'quadrantChart\nA: [-0.5, 0.5]',
  'quadrantChart\nA: [.5, .5]',
  'quadrantChart\nA: [0.5, 0.5] radius: 10',
  'quadrantChart\nA: [0.5, 0.5] radius: 0',
  'quadrantChart\nA: [0.5, 0.5] radius: 2500',
  'quadrantChart\nA: [0.5, 0.5] radius: ' + '9'.repeat(400),
  'quadrantChart\nA: [0.5, 0.5] radius: 1.5',
  'quadrantChart\nA: [0.5, 0.5] radius: -1',
  'quadrantChart\nA: [0.5, 0.5] radius:',
  'quadrantChart\nA: [0.5, 0.5] radius',
  'quadrantChart\nA: [0.5, 0.5] radius: 5: 6',
  'quadrantChart\nA: [0.5, 0.5] radius : 5',
  'quadrantChart\nA: [0.5, 0.5] color: #ff0000',
  'quadrantChart\nA: [0.5, 0.5] color: ff0000',
  'quadrantChart\nA: [0.5, 0.5] color: #f00',
  'quadrantChart\nA: [0.5, 0.5] color: #ABCDEF',
  'quadrantChart\nA: [0.5, 0.5] color: #ff00',
  'quadrantChart\nA: [0.5, 0.5] color: red',
  'quadrantChart\nA: [0.5, 0.5] stroke-color: #00f',
  'quadrantChart\nA: [0.5, 0.5] stroke-width: 3px',
  'quadrantChart\nA: [0.5, 0.5] stroke-width: 3',
  'quadrantChart\nA: [0.5, 0.5] stroke-width: 900px',
  'quadrantChart\nA: [0.5, 0.5] stroke-color: #00f, stroke-width: 3px',
  'quadrantChart\nA: [0.5, 0.5] stroke-width: 3px, stroke-color: #00f, color: #0f0, radius: 8',
  'quadrantChart\nA: [0.5, 0.5] size: 4',
  'quadrantChart\nA: [0.5, 0.5] radius: 4,',
  'quadrantChart\nA: [0.5, 0.5] radius: 4, radius: 9',
  'quadrantChart\nA: [0.5, 0.5] radius: 4 color: #fff',
  'quadrantChart\nA:::c: [0.5, 0.5]',
  'quadrantChart\nA:::c: [0.5, 0.5]\nclassDef c radius: 12, color: #f0f',
  'quadrantChart\nclassDef c radius: 12\nclassDef d color: #0ff\nclassDef c stroke-width: 4px\nA:::c: [0.4, 0.4]\nB:::d: [0.6, 0.6] radius: 3',
  'quadrantChart\nclassDef c radius: x',
  'quadrantChart\nclassDef c',
  'quadrantChart\nclassDef c ',
  'quadrantChart\nclassDef  c radius: 1',
  'quadrantChart\nclassDef c-d,e:f radius: 1',
  'quadrantChart\nclassDef é radius: 1',
  'quadrantChart\nclassDefs c radius: 1',
  'quadrantChart\nA:::: [0.5, 0.5]',
  'quadrantChart\nA:::é: [0.5, 0.5]',
  'quadrantChart\nA ::: c: [0.5, 0.5]',
  'quadrantChart\nA:::c : [0.5, 0.5]',
  'quadrantChart\n"Quoted point": [0.5, 0.5]',
  'quadrantChart\n"`**Bold** point`": [0.5, 0.5]',
  'quadrantChart\nA b c: [0.5, 0.5]',
  'quadrantChart\n1: [0.5, 0.5]',
  'quadrantChart\n-: [0.5, 0.5]',
  'quadrantChart\n!: [0.5, 0.5]',
  'quadrantChart\nA: [0.5, 0.5]; B: [0.6, 0.6]',
  'quadrantChart\nA: [0.5, 0.5] %% c\nB: [0.2, 0.2]',
  'quadrantChart\nA%%: [0.5, 0.5]',
  'quadrantChart\n%%{init: {}}%%\nA: [0.5, 0.5]',
  'quadrantChart\n}%%x\nA: [0.5, 0.5]',
  'quadrantChart\n😀%%x\nA: [0.5, 0.5]',
  'quadrantChart\nA: [0.5, 0.5]\r\nB: [0.2, 0.3]\r\n',
  'quadrantChart\n\u00a0A: [0.5, 0.5]',
  'quadrantChart\nA\u00a0: [0.5, 0.5]',
  'quadrantChart\nA:\u3000[0.5, 0.5]',
  'quadrantChart\n\u2028A: [0.5, 0.5]',
  'quadrantChart\nA: [0.5,\u2028 0.5]',
  'quadrantChart\nA: [0.5, 0.5] \u2028',
  'quadrantChart\n@',
  'quadrantChart\n<b>x</b>: [0.5, 0.5]',
  'quadrantChart\n\u0001',
  '---\ntitle: Front matter\n---\nquadrantChart\nA: [0.5, 0.5]',
  '---\ntitle: Front matter\n---\nquadrantChart\ntitle Own\nA: [0.5, 0.5]',
  '---\nconfig:\n  quadrantChart:\n    chartWidth: 300\n    chartHeight: 200\n---\nquadrantChart\nquadrant-1 One\nA: [0.5, 0.5]',
  '%%{init: {"quadrantChart": {"pointRadius": 9, "pointTextPadding": 1}}}%%\nquadrantChart\nA: [0.5, 0.5]',
  '%%{init: {"quadrantChart": {"xAxisPosition": "bottom", "yAxisPosition": "right"}}}%%\nquadrantChart\nx-axis L --> R\ny-axis B --> T\nquadrant-1 Q',
  '%%{init: {"quadrantChart": {"xAxisPosition": "top"}}}%%\nquadrantChart\nx-axis L --> R\nA: [0.5, 0.5]',
  '%%{init: {"quadrantChart": {"chartWidth": 50, "chartHeight": 9000, "quadrantPadding": 300}}}%%\nquadrantChart\nquadrant-1 Q',
  '%%{init: {"quadrantChart": {"chartWidth": -5, "chartHeight": "300", "titlePadding": 40}}}%%\nquadrantChart\ntitle T\nquadrant-1 Q',
  '%%{init: {"quadrantChart": {"quadrantTextTopPadding": 30, "xAxisLabelPadding": 20, "yAxisLabelPadding": 0}}}%%\nquadrantChart\nx-axis L\ny-axis B\nquadrant-1 Q\nA: [0.5, 0.5]',
  'quadrantChart\ntitle A title that is much wider than the chart under it, so that it wraps onto more than a single line of text\nA: [0.5, 0.5]',
  'quadrantChart\nquadrant-1 A quadrant label that is far too long to fit within the width of its own quadrant\nquadrant-2 Supercalifragilisticexpialidociousandmoreandmore',
  'quadrantChart\nx-axis A very long label for the left of the x axis --> And a very long one for its right as well\ny-axis A very long label for the bottom of the y axis --> And a long one for its top',
  'quadrantChart\nA point with a very long label that wraps under it: [0.5, 0.5]',
  'quadrantChart\nA: [0.5, 0.5]\nB: [0.5, 0.5]\nC: [0.5, 0.5]\nD: [0.5, 0.5]\nE: [0.5, 0.5]\nF: [0.5, 0.5]',
  'quadrantChart\nA: [0, 0]\nB: [1, 1]\nC: [0, 1]\nD: [1, 0]\nE: [0.5, 0]\nF: [0, 0.5]',
  'quadrantChart\nA: [0.49, 0.51]\nB: [0.51, 0.49]\nC: [0.5, 0.5] radius: 30',
  'quadrantChart\nquadrant-1 "`a\n  b\n    c`"',
  'quadrantChart\nquadrant-1 one<br>two<br/>three',
  'quadrantChart\nquadrant-1 a &amp; b #35; c',
  'quadrantChart\nquadrant-1 fa:fa-car car',
  'quadrantChart\nquadrant-1 日本語の長いラベルは四分円の中で折り返されるはずですね本当に長い',
];

// The sources in Mermaid's specs and documentation as Pele's tests collect them:
// loadCorpus('quadrant', /quadrant/i).
function specSources(pele) {
  return [...new Set([...testStrings(pele, 'tests/compat/quadrant', /\.spec\.[jt]s$/, /quadrant/i), ...docs(pele, 'quadrant')])];
}

// The corpus the suites start from.
export function corpus(pele) {
  return [...new Set([...specSources(pele), ...SEEDS, ...WRITTEN])].filter((s) => !lone(s));
}

// Quadrant charts built at random, which mostly parse, with labels, points, styles and
// settings of every kind the layout handles differently.
export function randomDiagrams(rnd) {
  const pick = picker(rnd);
  // Words that may stand outside quotes, and those that may only stand inside them, or in a
  // title.
  const words = ['Low', 'High', 'Reach', 'Engagement', 'Campaign', 'a', 'very', 'long', 'label', 'that', 'wraps',
    'Supercalifragilistic', 'Averyveryverylongwordwithoutanyspaceinit', '日本語', '中文字', '😀', 'é', 'x', 'W', 'iiii',
    'MMMM', '1.5', 'a-b', 'c_d', 'e.f', 'A&B', '#1', 'why?', 'yes!', '50%', 'x*y', 'a+b=c', "it's", 'a/b', 'café'];
  const quoted = [...words, '<br>', '<b>x</b>', '#35;', '&amp;', 'fa:fa-car', '(a)', '[b]', '<', 'a:b', 'a;b', '_u_', '**s**'];
  const md = ['**bold**', '_it_', '*em*', '__u__', 'plain', 'two\nlines', '**a** b _c_', '<br>', '&lt;'];
  const phrase = (max, from = words) => {
    const out = [];
    for (let k = 1 + Math.floor(rnd() * max); k > 0; k--) out.push(pick(from));
    return out.join(' ');
  };
  const label = (max) => {
    const r = rnd();
    if (r < 0.12) return `"${phrase(max, quoted)}"`;
    if (r < 0.25) {
      const out = [];
      for (let k = 1 + Math.floor(rnd() * max); k > 0; k--) out.push(rnd() < 0.5 ? pick(md) : pick(quoted));
      return `"\`${out.join(' ')}\`"`;
    }
    return phrase(max);
  };
  const coord = () => pick(['0', '1', '0.5', '0.1', '0.25', '0.75', '0.9', '0.33', '0.66', '0.05', '0.95', '0.123', '0.5', '0.0', '0.999', '0x1', '0b1', '0x2']);
  const hex = () => pick(['#ff0000', '#0f0', '00f', '#ABCDEF', '#123456', '#abc', 'fff']);
  const style = () =>
    pick([
      () => `radius: ${pick(['0', '1', '3', '5', '8', '12', '20', '40', '100'])}`,
      () => `color: ${hex()}`,
      () => `stroke-color: ${hex()}`,
      () => `stroke-width: ${pick(['0', '1', '2', '5', '10', '30'])}px`,
    ])();
  const styles = () => {
    const out = [];
    for (let k = 1 + Math.floor(rnd() * 3); k > 0; k--) out.push(style());
    return out.join(pick([', ', ',', ' , ']));
  };
  const config = () => {
    const entries = [];
    const settings = [
      ['chartWidth', ['500', '300', '800', '150', '1200', '99']],
      ['chartHeight', ['500', '300', '700', '120', '1000']],
      ['titlePadding', ['10', '0', '30']],
      ['quadrantPadding', ['8', '0', '20', '50']],
      ['xAxisLabelPadding', ['5', '0', '15']],
      ['yAxisLabelPadding', ['5', '0', '15']],
      ['quadrantTextTopPadding', ['5', '0', '25']],
      ['pointRadius', ['5', '2', '10', '0']],
      ['pointTextPadding', ['5', '0', '12']],
      ['xAxisPosition', ['"top"', '"bottom"']],
      ['yAxisPosition', ['"left"', '"right"']],
    ];
    for (const [key, values] of settings) if (rnd() < 0.25) entries.push([key, pick(values)]);
    if (entries.length === 0) {
      const [key, values] = pick(settings);
      entries.push([key, pick(values)]);
    }
    if (rnd() < 0.5) {
      return `%%{init: {"quadrantChart": {${entries.map(([k, v]) => `"${k}": ${v}`).join(', ')}}}}%%\n`;
    }
    return `---\nconfig:\n  quadrantChart:\n${entries.map(([k, v]) => `    ${k}: ${v}`).join('\n')}\n---\n`;
  };
  return () => {
    let s = rnd() < 0.3 ? config() : '';
    s += 'quadrantChart\n';
    const indent = () => pick(['  ', '    ', '']);
    const lines = [];
    if (rnd() < 0.6) lines.push(`title ${phrase(rnd() < 0.2 ? 20 : 5, quoted)}`);
    if (rnd() < 0.1) lines.push(`accTitle: ${phrase(3)}`);
    if (rnd() < 0.1) lines.push(`accDescr: ${phrase(3)}`);
    if (rnd() < 0.65) lines.push(`x-axis ${label(3)}${rnd() < 0.75 ? ' --> ' + (rnd() < 0.9 ? label(rnd() < 0.15 ? 12 : 3) : '') : ''}`);
    if (rnd() < 0.65) lines.push(`y-axis ${label(3)}${rnd() < 0.75 ? ' --> ' + (rnd() < 0.9 ? label(rnd() < 0.15 ? 12 : 3) : '') : ''}`);
    for (let k = 1; k <= 4; k++) if (rnd() < 0.6) lines.push(`quadrant-${k} ${label(rnd() < 0.15 ? 14 : 3)}`);
    const classes = ['c1', 'c2', 'big', 'k'];
    const n = Math.floor(rnd() * (rnd() < 0.05 ? 30 : 8));
    for (let i = 0; i < n; i++) {
      let line = `${label(rnd() < 0.1 ? 10 : 2)}${rnd() < 0.2 ? ':::' + pick(classes) : ''}: [${coord()}, ${coord()}]`;
      if (rnd() < 0.3) line += ' ' + styles();
      lines.push(line);
    }
    for (const c of classes) if (rnd() < 0.2) lines.push(`classDef ${c} ${styles()}`);
    for (let i = lines.length - 1; i > 0; i--) {
      if (rnd() < 0.2) {
        const j = Math.floor(rnd() * (i + 1));
        [lines[i], lines[j]] = [lines[j], lines[i]];
      }
    }
    return s + lines.map((l) => indent() + l).join('\n') + '\n';
  };
}

const text = (t) => (t === undefined ? null : { text: t.text, type: t.type });
const pointStyle = (p) => {
  const out = {};
  if (p.radius !== undefined) out.radius = enc(p.radius);
  if (p.color !== undefined) out.color = p.color;
  if (p.strokeColor !== undefined) out.strokeColor = p.strokeColor;
  if (p.strokeWidth !== undefined) out.strokeWidth = p.strokeWidth;
  return out;
};

function dumpModel(m) {
  const opt = (x) => (x === undefined || x === '' ? undefined : x);
  return {
    title: opt(m.title),
    accTitle: opt(m.accTitle),
    accDescr: opt(m.accDescr),
    quadrants: m.quadrants.map(text),
    xAxisLeft: text(m.xAxisLeft),
    xAxisRight: text(m.xAxisRight),
    yAxisBottom: text(m.yAxisBottom),
    yAxisTop: text(m.yAxisTop),
    points: m.points.map((p) => ({ x: enc(p.x), y: enc(p.y), text: text(p.text), className: p.className, style: pointStyle(p) })),
    classes: [...m.classes].map(([name, style]) => [name, pointStyle(style)]),
  };
}

export default async function ({ src, pele, write, random }) {
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { quadrantChart } = await src('diagrams/quadrant/index.ts');
  const { QuadrantDb } = await src('diagrams/quadrant/db.ts');
  const { parseQuadrant } = await src('diagrams/quadrant/parser.ts');
  const { tokenize } = await src('diagrams/quadrant/lexer.ts');
  const { TOKEN_NAMES } = await src('diagrams/quadrant/tokens.ts');

  const caught = (e) => {
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  const tokens = (source) => {
    const t = tokenize(source);
    return t.types.map((type, i) => [TOKEN_NAMES[type], t.texts[i], t.starts[i]]);
  };
  const parsed = (source, parseIt) => {
    try {
      return { tokens: tokens(source), model: dumpModel(parseIt()) };
    } catch (e) {
      return { tokens: tokens(source), ...caught(e) };
    }
  };
  // Pele's parseSource for a quadrant chart, and the raw text given to the parser alone.
  const run = (source) => {
    const out = {
      src: source,
      direct: parsed(source, () => {
        const db = new QuadrantDb();
        parseQuadrant(source, db);
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
    if (out.type !== 'quadrantChart') return out;
    const text = encodeEntities(pre.text) + '\n';
    const piped = parsed(text, () => quadrantChart.parse(text, pre.config, pre.title));
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
  // Mutated text; one in six is cut short, as a statement cut off by the end of the text ends
  // on a different token than one cut off by a line break.
  const rnd = random(7);
  const next = mutator(base, FRAGMENTS, rnd);
  for (let i = 0; i < 4200; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  const diagrams = randomDiagrams(random(10));
  for (let i = 0; i < 1500; i++) add(diagrams());
  // Random strings of tokens, where keyword boundaries and the lexer's lookahead matter.
  const trnd = random(8);
  const pieces = [...new Set([...FRAGMENTS, ...FRAGMENTS.map((f) => f.trim()).filter((f) => f !== '')])];
  const openers = ['quadrantChart\n', 'quadrantChart\nA', 'quadrantChart\nA: [', 'quadrantChart\nx-axis ', 'quadrantChart\nclassDef c ',
    'quadrantChart\nA: [0.5, 0.5] ', 'quadrantChart\n"`', ''];
  for (let i = 0; i < 2000; i++) {
    let s = openers[Math.floor(trnd() * openers.length)];
    for (let k = 1 + Math.floor(trnd() * 8); k > 0; k--) s += pieces[Math.floor(trnd() * pieces.length)] + (trnd() < 0.3 ? ' ' : '');
    add(s);
  }

  write('diagram/quadrant/testdata/pele/quadrant.json', { cases: all.map(run) });
}
