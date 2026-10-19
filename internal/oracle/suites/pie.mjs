// Pie charts parsed as Pele parses them: the tokens Pele's common/tokens.ts makes of the text
// with the pie's table, the tree parsePie makes of them, and the model buildPie makes of the
// tree with the title the text's front matter gives. The cases are Mermaid's documentation and
// specs, the inputs of Pele's own tests, cases written here, random charts, and all of those
// mutated at random.
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { enc, lone, mutator, peleError } from './parse.mjs';

// The fragments of Pele's tests/pie/oracle.test.ts, and more.
export const FRAGMENTS = [
  'pie', 'showData', 'title', 'title ', 'accTitle: ', 'accDescr: ', 'accDescr {', '}', ':', ' : ', '"', "'", '"a"', "'b'",
  '1', '0', '-1', '1.5', '01', '1.', '.5', '1e3', '\n', '\r\n', ' ', '\t', '%%', '%% c', '%%{init: {}}%%', '---', '---\n',
  '\\', '\\"', 'x', 'é', ';', ',', '#35;', '"a" : 1', '\n"b":2',
  '-0', '0.0', '00', '1..2', '1.2.3', '-.5', '--1', '99999999999999999999', '1' + '0'.repeat(400), '0.000001', '\r', '\u2028',
  '\u3000', '😀', '中文', '%%{', '}%%', '"a\\nb"', "'it\\'s'", '&amp;', '#quot;', '<br>', '<b>x</b>', 'accDescr {\n a\n}\n',
  '\n"tiny": 0.1', '\n"big": 1000', '\n"zero": 0', 'showData ', 'pie showData', '\ntitle T\n', '"a":', ': 5',
];

// The inputs of Pele's tests/pie/oracle.test.ts beyond its corpus.
const SEEDS = [
  'pie',
  'pie showData title T\n"a":1\n"b":2.5',
  'pie\n accTitle: at\n accDescr { multi\n line }\n title t\n "a" : 1',
  '---\ntitle: x\n---\npie\n"a":1',
  '%%{init: {"theme":"dark"}}%%\npie title A %% c\n"a":1 %% d\n',
];

const config = (lines, body) => `---\nconfig:\n  pie:\n${lines.map((l) => '    ' + l).join('\n')}\n---\n${body}`;
const PETS = 'pie title Pets adopted by volunteers\n    "Dogs" : 386\n    "Cats" : 85\n    "Rats" : 15\n';
const THIN = 'pie\n "Calcium" : 2\n "Potassium" : 2.4\n "Magnesium" : 46\n "Test" : 50';

// Cases written here: what the tokens and the model do at their edges, and Pele's render tests.
export const WRITTEN = [
  PETS,
  THIN,
  'pie title Pets\n "Dogs" : 386\n "Cats" : 85\n "Rats" : 15',
  'pie\n "a": 1',
  'pie showData\n "A" : 42.96\n "B" : 50',
  'pie\n "Big" : 1000\n "Tiny" : 1',
  'pie\n "Only" : 5',
  'pie\n "Zero" : 0',
  'pie\n "A" : 1\n "B" : 1',
  'pie showData\n "A" : 1\n "A" : 2\n "B" : 3',
  '---\ntitle: From front matter\n---\npie\n "A" : 1',
  '---\ntitle: From front matter\n---\npie title Own\n "A" : 1',
  'pie\n "dogs" : -60.67',
  'pie\n "dogs" : -0',
  'pie\n "a" : 1\n "dogs" : -1\n "dogs" : -2',
  'pie\n "a" : -' + '9'.repeat(400),
  'pie\n "a" : ' + '9'.repeat(400),
  'pie\n "a" : ' + '9'.repeat(400) + '\n "b" : 1',
  'pie\n "a" : 1' + '0'.repeat(30) + '\n "b" : 1',
  'pie\n "a" : 0.' + '0'.repeat(30) + '1\n "b" : 1',
  config(['donutHole: 0.5'], 'pie\n "A" : 1\n "B" : 1'),
  config(['legendPosition: bottom'], 'pie\n "A" : 1\n "B" : 1'),
  config(['textPosition: 0.5', 'donutHole: 0.2', 'highlightSlice: Potassium'], 'pie showData\n    title Key elements in Product X\n    "Calcium" : 42.96\n    "Potassium" : 50.05\n    "Magnesium" : 10.01\n    "Iron" :  5\n'),
  config(['legendPosition: left'], THIN),
  config(['legendPosition: top'], THIN),
  config(['legendPosition: center', 'donutHole: 0.9'], THIN),
  config(['legendPosition: nowhere'], THIN),
  config(['legendPosition: [left]'], THIN),
  config(['legendPosition: 5'], THIN),
  config(['legendPosition: null'], THIN),
  config(['textPosition: 0', 'donutHole: 0'], THIN),
  config(['textPosition: 1', 'donutHole: 0.91'], THIN),
  config(['textPosition: -0.1', 'donutHole: -0.5'], THIN),
  config(['textPosition: 1.5', 'donutHole: 2'], THIN),
  config(['textPosition: "0.5"', 'donutHole: "0.5"'], THIN),
  config(['textPosition: .nan', 'donutHole: .nan'], THIN),
  config(['textPosition: .inf', 'donutHole: .inf'], THIN),
  config(['donutHole: 0.5'], 'pie\n "Only" : 5'),
  config(['donutHole: 0.5'], 'pie\n "Zero" : 0'),
  '---\nconfig:\n  pie: 5\n---\npie\n "a": 1',
  '---\nconfig:\n  pie: [1, 2]\n---\npie\n "a": 1',
  '%%{init: {"pie": {"donutHole": 0.4, "legendPosition": "bottom"}}}%%\npie\n "a": 1\n "b": 2',
  '%%{init: {"pie": {"textPosition": 0.3}}}%%\npie\n "a": 1\n "b": 2',
  'pie\n "a": 0\n "b": 0',
  'pie\n "a": 0.004\n "b": 0.996',
  'pie\n "a": 0.005\n "b": 0.995',
  'pie\n "a": 0.0049\n "b": 0.9951',
  'pie\n "a": 1\n "b": 199',
  'pie\n "a": 1\n "b": 99',
  'pie\n "a": 1\n "b": 1\n "c": 1',
  'pie\n "a": 1\n "b": 1\n "c": 1\n "d": 1\n "e": 1\n "f": 1\n "g": 1\n "h": 1\n "i": 1\n "j": 1',
  'pie\n "a": 1\n "b": 2\n "c": 3\n "d": 4\n "e": 5\n "f": 6\n "g": 7\n "h": 8\n "i": 9\n "j": 10\n "k": 11',
  'pie\n "a": 3\n "b": 3\n "c": 3\n "d": 3\n "e": 3\n "f": 3\n "g": 3\n "h": 3\n "i": 3\n "j": 3\n "k": 3\n "l": 3\n "m": 3\n "n": 3\n "big": 300',
  'pie\n "a": 50\n "b": 50',
  'pie\n "a": 50.5\n "b": 49.5',
  'pie\n "a": 2\n "b": 1',
  'pie\n "a": 1\n "b": 2',
  'pie\n"a":1',
  'pie\n"a" :1\n"b": 2\n"c" : 3',
  "pie\n'a': 1\n'b': 2",
  'pie\n"a\\"b": 1\n\'c\\\'d\': 2',
  'pie\n"a": 1 "b": 2',
  'pie\n"a": 1\n"b"',
  'pie\n"a" 1',
  'pie\n"a":',
  'pie\n"a": x',
  'pie\n"a": 1.',
  'pie\n"a": .5',
  'pie\n"a": 1e3',
  'pie\n"a": 01',
  'pie\n"a": 00.5',
  'pie\n"a": 0.5.5',
  'pie\n"a": 1..5',
  'pie\n"a": 1.5.',
  'pie\n"a": 10.',
  'pie\n"a": 0x1',
  'pie\n"a": +1',
  'pie\n"a": --1',
  'pie\n"a": - 1',
  'pie\n"a": -.5',
  'pie\n"a": 1 %% c\n"b": 2',
  'pie\n"a": 1%%c\n"b": 2',
  'pie\n"a": 1\r\n"b": 2\r\n',
  'pie\r\n"a": 1\r"b": 2',
  'pie\n\u00a0"a": 1',
  'pie\n"a": 1\u3000',
  'pie\n\t"a"\t:\t1\t',
  'pie\n: 1',
  'pie\n1',
  'pie\na: 1',
  'pie\n"": 1',
  'pie\n"a\nb": 1',
  'pie\n"unclosed: 1',
  "pie\n'unclosed: 1",
  'pie\n"a\\": 1',
  'pie\n"😀": 1\n"中文": 2',
  'pie\n"<b>bold</b>": 1\n"a<br>b": 2\n"#35; &amp;": 3',
  'pie\n"&lt;b&gt;": 1',
  'pie\n"<script>x</script>": 1',
  'pie\n"' + 'word '.repeat(40) + '": 1',
  'pie showData\n"a": 1.5\n"b": 0.1\n"c": 100000000000000000000000',
  'pie showData\n"a": 0.000001\n"b": 1',
  'pie showData',
  'pie showData title T',
  'pie title T showData',
  'pie title',
  'pie title\n"a": 1',
  'pie\ntitle',
  'pie\ntitle   spaced    out   \n"a": 1',
  'pie\ntitle x %% c\n"a": 1',
  'pie\ntitle %%c\n"a": 1',
  'pie\ntitlex\n"a": 1',
  'pie\ntitle\tT\n"a": 1',
  'pie\n  title T\n"a": 1',
  'pie\ntitle A\ntitle B\n"a": 1',
  'pie\ntitle #35; &amp; <b>b</b>\n"a": 1',
  'pie\ntitle A very long title of a pie chart that is much wider than the pie and its legend are, by far\n"a": 1',
  'pie\ntitle T "a": 1',
  'pie\naccTitle: x\naccTitle: y',
  'pie\naccTitle:    spaced    x   ',
  'pie\naccTitle : x',
  'pie\naccTitle:x %% c',
  'pie\naccTitle',
  'pie\naccDescr: one %% c\n"a": 1',
  'pie\naccDescr :one',
  'pie\naccDescr{x}',
  'pie\naccDescr\n{\n  x\n}',
  'pie\naccDescr {\n  line one\n    line two\n\n\n  line three   \n}\n"a": 1',
  'pie\naccDescr {\n\t\n  a  b \r\n\r\n c\n}',
  'pie\naccDescr {',
  'pie\naccDescr {\n a\n}\n}',
  'pie\naccDescr { a } "a": 1',
  'pie\naccDescr: d\naccDescr {\n e\n}',
  'pie\n%%{init: {}}%%\n"a": 1',
  'pie\n"a": 1 %%{x}%%\n"b": 2',
  'pie\n"a": 1\n%%{\n"b": 2',
  'pie\n%%{a}%%b}%%\n"a": 1',
  'pie\n  %% indented comment\n"a": 1',
  'pie\n"a": 1 %%',
  'pie  %%  c\n"a": 1',
  'pie\n---\n"a": 1',
  'pie\n---\n---\n"a": 1',
  'pie\n"a": 1\n---\nx\n---\n"b": 2',
  'pie\n"a": 1\npie',
  'pie\nshowData',
  'pie showData showData',
  'pie\n"a": 1\ntitle T\n"b": 2\naccTitle: A\n"c": 3',
  'Pie\n"a": 1',
  'PIE\n"a": 1',
  'pie:\n"a": 1',
  'pies\n"a": 1',
  'piex',
  'pie%% c\n"a": 1',
  'pie ',
  'pie\n',
  '\n\n pie \n\n "a": 1',
  '',
  'not a pie',
];

// Strings in the sources of a folder of Pele's tests that hold the keyword, as Pele's
// loadCorpus and loadTestStrings collect them, from files whose names match.
function testStrings(pele, dir, files) {
  const out = new Set();
  const path = join(pele, dir);
  for (const file of readdirSync(path).filter((f) => files.test(f))) {
    const src = readFileSync(join(path, file), 'utf8');
    for (const m of src.matchAll(/`((?:[^`\\]|\\.)*)`|'((?:[^'\\\n]|\\.)*)'|"((?:[^"\\\n]|\\.)*)"/g)) {
      const raw = m[1] ?? m[2] ?? m[3];
      if (!/pie/.test(raw)) continue;
      try {
        out.add(new Function('return `' + raw.replace(/`/g, '\\`').replace(/\$\{/g, '\\${') + '`')());
      } catch {
        continue;
      }
    }
  }
  return [...out];
}

// The corpus the suites start from: documentation, specs, the strings of Pele's tests, its
// seeds and the cases above.
export function corpus(pele) {
  const docs = JSON.parse(readFileSync(join(pele, 'tests/corpus/pie-docs.json'), 'utf8'));
  return [
    ...new Set([
      ...docs,
      ...testStrings(pele, 'tests/compat/pie', /\.spec\.[jt]s$/),
      ...testStrings(pele, 'tests/compat/pie', /\.test\.[jt]s$/),
      ...testStrings(pele, 'tests/render', /^pie.*\.test\.ts$/),
      ...SEEDS,
      ...WRITTEN,
    ]),
  ].filter((s) => !lone(s) && s.length <= 50000);
}

const LABELS = [
  'a', 'x', 'Dogs', 'Cats', 'Rats', 'Calcium', 'Potassium', 'Magnesium', 'Iron', 'FRIENDLY', 'Other', '', ' spaced ',
  'Extraordinarily long label for a slice of the pie', 'two words', '中文标签', '😀 emoji', '#35;', '&amp; more', '#quot;q#quot;',
  'a\\"b', "it\\'s", '<b>bold</b>', '<i>it</i> and <b>b</b>', 'line<br>break', 'line\\nbreak', 'word '.repeat(12), 'iiiiiiiiiiii',
  'WWWWWW', 'é', 'a-b', '0', 'mixed 中文 text', 'x\u00a0y', 'trailing   ', 'Ünïcödé', 'fa:fa-car car', '**strong**',
];
const TITLES = [
  'T', 'Pets adopted by volunteers', 'Key elements in Product X', 'A title that is much wider than the pie it names, by a long way and more',
  '中文', '#35; &amp;', '<b>Bold</b> title', 'Two<br>lines',
];
const VALUES = [
  1, 2, 3, 5, 10, 15, 42.96, 50.05, 10.01, 85, 386, 100, 1000, 0, 0.1, 0.5, 0.004, 0.005, 2.4, 46, 50, 33.3, 66.7, 99, 199, 0.0001,
  12345.678, 7, 1e6, 3.14159, 999999,
];

// Random pie charts, most of them well formed, with random configuration.
export function randomDiagrams(rnd) {
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  const chance = (p) => rnd() < p;
  const quote = (label) => (label.includes("'") || chance(0.8) ? `"${label}"` : `'${label}'`);
  return () => {
    const settings = [];
    if (chance(0.6)) {
      if (chance(0.5)) settings.push(`textPosition: ${pick([0, 0.1, 0.25, 0.5, 0.6, 0.75, 0.8, 0.9, 1, 1.2, -1, '"x"'])}`);
      if (chance(0.5)) settings.push(`donutHole: ${pick([0, 0.1, 0.2, 0.25, 0.3, 0.5, 0.6, 0.75, 0.9, 0.95, 1, -0.5])}`);
      if (chance(0.6)) settings.push(`legendPosition: ${pick(['left', 'right', 'top', 'bottom', 'center', 'none', 'Left', '""'])}`);
    }
    let s = '';
    if (chance(0.1)) s += `---\ntitle: ${pick(TITLES)}\n---\n`;
    if (settings.length) {
      if (chance(0.7)) s += `---\nconfig:\n  pie:\n${settings.map((l) => '    ' + l).join('\n')}\n---\n`;
      else s += `%%{init: {"pie": {${settings.map((l) => l.replace(/^(\w+): (.*)$/, (_, k, v) => `"${k}": ${/^[-\d.]+$|^"/.test(v) ? v : `"${v}"`}`)).join(', ')}}}}%%\n`;
    }
    s += 'pie';
    if (chance(0.3)) s += ' showData';
    if (chance(0.15)) s += ` title ${pick(TITLES)}`;
    s += '\n';
    if (chance(0.3)) s += `title ${pick(TITLES)}\n`;
    if (chance(0.1)) s += 'accTitle: Accessible\n';
    if (chance(0.1)) s += chance(0.5) ? 'accDescr: Described\n' : 'accDescr {\n  one\n   two\n}\n';
    const count = chance(0.05) ? 0 : 1 + Math.floor(rnd() * (chance(0.2) ? 16 : 6));
    for (let k = 0; k < count; k++) {
      let value = chance(0.7) ? pick(VALUES) : Math.round(rnd() * 10 ** (1 + Math.floor(rnd() * 4))) / pick([1, 10, 100]);
      if (chance(0.01)) value = -value;
      s += `${chance(0.5) ? '    ' : ''}${quote(pick(LABELS))}${pick([' : ', ': ', ':', ' :  '])}${value}`;
      if (chance(0.05)) s += ' %% comment';
      s += chance(0.05) ? '\r\n' : '\n';
      if (chance(0.05)) s += '\n';
    }
    return s;
  };
}

export default async function ({ src, pele, write, random }) {
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { pie } = await src('diagrams/pie/index.ts');
  const { parsePie, PIE_TOKENS } = await src('diagrams/pie/parser.ts');
  const { tokenize } = await src('diagrams/common/tokens.ts');

  const caught = (e) => {
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  const opt = (x) => (x === undefined || x === '' ? undefined : x);
  const tokens = (source) => {
    try {
      const t = tokenize(source, PIE_TOKENS, 'pie');
      return { tokens: t.kinds.slice(0, -1).map((k, i) => [PIE_TOKENS[k].name, source.slice(t.starts[i], t.ends[i])]) };
    } catch (e) {
      return { tokens: null };
    }
  };
  const direct = (source) => {
    try {
      const ast = parsePie(source);
      return {
        ...tokens(source),
        ast: {
          showData: ast.showData,
          title: enc(ast.title),
          accTitle: enc(ast.accTitle),
          accDescr: enc(ast.accDescr),
          sections: ast.sections.map((s) => [s.label, enc(s.value)]),
        },
      };
    } catch (e) {
      return { ...tokens(source), ...caught(e) };
    }
  };
  // Pele's parseSource for a pie chart, and the raw text given to the parser alone.
  const run = (source) => {
    const out = { src: source, direct: direct(source) };
    let pre;
    try {
      pre = preprocess(source);
      out.type = detect(pre.text);
    } catch (e) {
      return { ...out, ...caught(e) };
    }
    if (out.type !== 'pie') return out;
    try {
      const m = pie.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title);
      out.model = {
        title: opt(m.title),
        accTitle: opt(m.accTitle),
        accDescr: opt(m.accDescr),
        showData: m.showData,
        sections: [...m.sections].map(([label, value]) => [label, enc(value)]),
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
  for (let i = 0; i < 3500; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  // Random strings of tokens, where the order of the token types and their ends matter.
  const trnd = random(8);
  const pieces = [...new Set([...FRAGMENTS, ...FRAGMENTS.map((f) => f.trim()).filter((f) => f !== '')])];
  const openers = ['pie\n', 'pie ', 'pie showData\n', 'pie\n"a"', 'pie\n"a":', 'pie\n"a": ', 'pie\ntitle', 'pie\naccDescr', ''];
  for (let i = 0; i < 1000; i++) {
    let s = openers[Math.floor(trnd() * openers.length)];
    for (let k = 1 + Math.floor(trnd() * 8); k > 0; k--) s += pieces[Math.floor(trnd() * pieces.length)] + (trnd() < 0.3 ? ' ' : '');
    add(s);
  }
  // Stress cases, which stop at limits or would rescan the text.
  for (const s of [
    'pie\n"a": ' + '0'.repeat(3000),
    'pie\n"a": ' + '1'.repeat(3000) + '.',
    'pie\n"a": 1.' + '0'.repeat(3000) + '.',
    'pie\n"a": -' + '0'.repeat(3000),
    'pie\n' + '0'.repeat(3000),
    'pie\n' + '---\n'.repeat(800),
    'pie\n"a": "' + 'a'.repeat(3000),
    'pie\n"' + '\\'.repeat(3000),
    'pie\n' + '%%{\n'.repeat(800),
    'pie\naccDescr {' + ' a\n'.repeat(1000),
    'pie\n' + ' '.repeat(3000) + '"a": 1',
    'pie\n' + '\n \t'.repeat(1000) + '"a": 1',
    'pie\n' + '%% c\n'.repeat(700) + '"a": 1',
    'pie\n' + 'title t\n'.repeat(400),
    'pie\n' + '"a": 1\n'.repeat(500),
  ]) {
    seen.delete(s);
    add(s);
  }
  write('diagram/pie/testdata/pele/pie.json', { cases: all.map(run) });
}
