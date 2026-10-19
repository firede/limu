// Treemaps parsed as Pele parses them: the tokens Pele's common/tokens.ts makes of the text
// with the treemap's table, the tree parseTreemap makes of them, and the model buildTreemap
// makes of the tree with the title the text's front matter gives. The cases are Mermaid's
// documentation and specs, the inputs of Pele's own tests, cases written here, random
// treemaps, and all of those mutated at random.
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { enc, lone, mutator, peleError } from './parse.mjs';

// The fragments of Pele's tests/treemap/oracle.test.ts, and more.
export const FRAGMENTS = [
  'treemap', 'treemap-beta', 'title', 'title ', 'accTitle: ', 'accDescr: ', 'accDescr {', 'accDescr {}', '}', ':', ' : ', '::', ':::', ':::c1',
  ',', ' , ', '"', "'", '"a"', "'b'", '"a": 1', '"s"\n', '1', '0', '1.5', '1,000', '_', '.', '1e3', '-1', '\n', '\r\n', '\r',
  ' ', '  ', '    ', '\t', '%%', '%% c', '%%{init: {}}%%', '---', 'x', 'ab', 'é', ';', '#35;', 'classDef', 'classDef ',
  'classDef c1 fill:red', 'classDef c1 fill:red,stroke:blue;', 'class', '\n  "b": 2', '\n    "c":3:::c1',
  '\u2028', '\u3000', '\u00a0', '😀', '中文', '&amp;', '#quot;', '<br>', '\\,', 'rgb(1\\,2\\,3)', '!important', 'url(x)',
  '\n"s"\n  "t"\n    "u": 4', '99999999999999999999', '0.0001', '1,234,567.89', '___', '..', ',,', ':::c2', 'classDef c2 color:#fff',
];

// The inputs of Pele's tests/treemap/oracle.test.ts beyond its corpus.
const SEEDS = [
  'treemap',
  'treemap\naccDescr {}\n"A"',
  'treemap\nclassDef ab ;\n"A"',
  'treemap\nclassDef ab \n\n',
  'treemap-beta\n"A"\n  "B": 10\n',
  'treemap-betax',
  '  treemap',
  '\ntreemap\n',
  'treemap\n"A"   \n"B"',
  'treemap "A" "B"',
  'treemap\n"A" : 1,000.5:::cls\n',
  'treemap\n"A":1\n"B",2\n',
  'treemap\n"A" :::x',
  'treemap\n"A":::x1 "B"',
  'treemap\n  classDef foo fill:red,stroke:blue;\n"A"',
  'treemap\nclassDef foo\n',
  'treemap\nclassDef foo fill:red; "A"',
  'treemap\nclassDef\n\nfoo   a b c  \n"A"',
  'treemap\nclassDef a fill:red',
  'treemap title Hello\n"A"',
  'treemap\n  title   Hello  world %% c\n accTitle: x\n accDescr { a\n b }\n"A"',
  'treemap\n"A": _',
  'treemap\n"A": .',
  'treemap\n"A": 1.2.3',
  'treemap\n"A": ,5',
  'treemap\n"A":\n 5',
  'treemap\n"A"\n: 5',
  "treemap\n'A': 5 %% comment\n'B'",
  'treemap\n%% c\n"A"',
  'treemap\n"A\n multi": 5',
  'treemap\n"A" ',
  'treemap ',
  'treemap\n\t"A"\n\t\t"B":1',
  'treemap\n"A": 5 :::c',
  'treemap\n"A": 5::: c',
  'treemap\n"A"\n\n\n   \n"B"',
  '---\ntitle: x\n---\ntreemap\n"A"',
  '%%{init: {}}%%\ntreemap\n"A"',
  'treemap\n"A": 5 6',
  'treemap\n"A": 5 "B": 6',
  'treemap\nclass foo bar',
  'treemap\ntitle\n"A"',
  'treemap\ntitle a\ntitle b\n"A"',
  'treemap\r\n"A"\r\n  "B": 3\r\n',
  'treemap\n"A"  ,  7',
  'treemap\n"A"  ,',
  'treemap\n"A":::',
  'treemap\n "A"\n"B"',
  'treemap treemap',
  'treemap\n"A": 5:::c d',
  'treemap\n"A"\n classDef ab x\n classDef cd',
  'treemap\nclassDef ab x;;',
];

const config = (lines, body) => `---\nconfig:\n  treemap:\n${lines.map((l) => '    ' + l).join('\n')}\n---\n${body}`;
export const BASIC = 'treemap-beta\n"Category A"\n    "Item A1": 10\n    "Item A2": 20\n"Category B"\n    "Item B1": 15\n    "Item B2": 25\n';
const BUDGET = 'treemap\n"Budget"\n  "Operations"\n    "Salaries": 700000\n    "Equipment": 200000\n    "Supplies": 100000\n  "Marketing"\n    "Advertising": 400000\n    "Events": 100000';

// Cases written here: what the tokens, the tree and the model do at their edges, and Pele's
// render tests.
export const WRITTEN = [
  BASIC,
  BUDGET,
  config(['showValues: false'], BASIC),
  config(['nodeWidth: 30', 'nodeHeight: 20'], BASIC),
  config(['diagramPadding: 50'], BASIC),
  config(['padding: 20'], BASIC),
  config(['padding: 0'], BASIC),
  config(['padding: 300'], BASIC),
  config(["valueFormat: '$0,0'"], 'treemap\n"Budget"\n  "Salaries": 700000\n  "Equipment": 200000'),
  config(['nodeWidth: -3', 'nodeHeight: "x"', 'padding: 100000', 'valueFormat: 7'], BASIC),
  config(['nodeWidth: .inf', 'nodeHeight: .nan'], BASIC),
  config(['nodeWidth: 0.5', 'nodeHeight: 1'], BASIC),
  config(['nodeWidth: 400', 'nodeHeight: 400', 'diagramPadding: 2000'], BASIC),
  config(["valueFormat: '.2f'"], BASIC),
  config(["valueFormat: ',.1%'"], 'treemap\n"a": 0.25\n"b": 0.125\n"c": 0.0001'),
  config(["valueFormat: '.3e'"], 'treemap\n"a": 123456789\n"b": 0.000012345'),
  config(["valueFormat: 'd'"], 'treemap\n"a": 2.5\n"b": 3.5\n"c": 1234567.5'),
  config(["valueFormat: '$,.2f'"], 'treemap\n"s"\n  "a": 1234.5\n  "b": 0.005'),
  config(["valueFormat: '~'"], 'treemap\n"a": 1.5\n"b": 100'),
  config(["valueFormat: '.2~f'"], 'treemap\n"a": 1.5\n"b": 100\n"c": 1.25'),
  config(["valueFormat: '+,'"], 'treemap\n"a": 1234567\n"b": 0.1'),
  config(["valueFormat: '>10,.2f'"], 'treemap\n"a": 1234567'),
  config(["valueFormat: 'x'"], 'treemap\n"a": 255'),
  config(["valueFormat: ''"], 'treemap\n"a": 255'),
  config(["valueFormat: '.0'"], 'treemap\n"a": 255'),
  config(["valueFormat: '.25f'"], 'treemap\n"a": 0.1'),
  config(['valueFormat: ",.2f"', 'showValues: true'], BUDGET),
  config(['showValues: "no"'], BASIC),
  '---\nconfig:\n  treemap: 5\n---\n' + BASIC,
  '---\nconfig:\n  treemap: [1, 2]\n---\n' + BASIC,
  '%%{init: {"treemap": {"padding": 10, "valueFormat": "$,"}}}%%\n' + BASIC,
  '---\ntitle: From front matter\n---\n' + BASIC,
  '---\ntitle: From front matter\n---\ntreemap\ntitle Own\n"a": 1',
  '---\ntitle: From front matter\n---\ntreemap\ntitle\n"a": 1',
  'treemap\n"Only a section"',
  'treemap\n"a": 0\n"b": .',
  'treemap\n"a": ' + '9'.repeat(400),
  'treemap\n"a": ' + '9'.repeat(400) + '\n"b": 5',
  'treemap\n"s"\n  "a": 1e308\n  "b": 1.7e308\n"c": 4',
  'treemap\n"s"\n  "a": 179769313486231570000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000\n  "b": 179769313486231570000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000\n"c": 4',
  'treemap\naccTitle: Name\naccDescr: Text\n"a": 1',
  'treemap\n"A section with a very long name indeed"\n  "big": 1000\n  "A leaf with quite a long name": 60\n  "tiny": 12\n"other": 3000',
  'treemap\n"frame":::frame\n  "a": 1\nclassDef frame stroke:red',
  'treemap\n"s":::c\n  "a": 1:::c\nclassDef c fill:red,color:blue,stroke:#FFD600;',
  'treemap\n"s":::c\n  "a": 1:::c\nclassDef c fill-opacity:0.5,stroke-width:3px',
  'treemap\n"s":::c\n  "a": 1:::c\nclassDef c stroke:rgb(1\\,2\\,3),fill:hsl(10\\,20%\\,30%)',
  'treemap\n"s":::c\n  "a": 1:::c\nclassDef c stroke:x"fill:y',
  'treemap\n"s":::c\n  "a": 1:::c\nclassDef c stroke:x&fill:y',
  'treemap\n"s":::c\n  "a": 1:::c\nclassDef c FILL:red',
  'treemap\n"s":::c\n  "a": 1:::c\nclassDef c fill :red',
  'treemap\n"s":::c\n  "a": 1:::c\nclassDef c fill:url(x)',
  'treemap\n"s":::c\n  "a": 1:::c\nclassDef c fill:red !important,stroke-linecap:round,stroke-linejoin:round',
  'treemap\n"s":::c\n  "a": 1:::c\nclassDef c stroke-linejoin:bevel;stroke-linecap:square',
  'treemap\n"s":::c\n  "a": 1:::c\nclassDef c opacity:0.5,font-weight:bold,font-size:20px',
  'treemap\n"s":::c\n  "a": 1:::c\nclassDef c §§§fill:red',
  'treemap\n"s":::undefined\n  "a": 1:::nope\nclassDef c fill:red',
  'treemap\n"s":::c\nclassDef c fill:red\nclassDef c stroke:blue\nclassDef c fill:green',
  'treemap\n"a":::c\nclassDef c\nclassDef c fill:red',
  'treemap\n"a": 1\n"b": 1\n"c": 1\n"d": 1\n"e": 1\n"f": 1\n"g": 1\n"h": 1\n"i": 1\n"j": 1',
  'treemap\n"a"\n  "b"\n    "c"\n      "d"\n        "e": 1',
  'treemap\n"a"\n    "b": 1\n  "c": 2\n "d": 3\n"e": 4',
  'treemap\n  "a"\n "b": 1',
  'treemap\n"a": 1,2,3',
  'treemap\n"a": 1__2',
  'treemap\n"a": 0.000001',
  'treemap\n"a": 00012',
  'treemap\n"a": 5.\n"b": .5',
  'treemap\n"a": -5',
  'treemap\n"": 5',
  'treemap\n"  ": 5',
  'treemap\n"#35; &amp; <b>b</b>": 5',
  'treemap\n"a&lt;b": 5',
  "treemap\n'single \"quoted\"': 5",
  'treemap\n"😀😀😀": 5\n"中文": 3',
  'treemap\n"a\\"b": 5',
  'treemap\n"a" 5',
  'treemap\n"a":5:::c:::d',
  'treemap\n"a"\t:\t5',
  'treemap\n"a"\u3000: 5',
  'treemap\n\u00a0"a": 5',
  'treemap\n"a": 5\u2028"b": 6',
  'treemap\n"a"\r"b"',
  'treemap\n"a": 5 %% c\n %% d\n"b": 6',
  'treemap\n%%{init: {}}%%\n"a": 1',
  'treemap\naccDescr {\n  line one\n    line two\n}\n"a": 1',
  'treemap\naccDescr\n{\n x\n}\n"a": 1',
  'treemap\naccDescr: one %% c\n"a": 1',
  'treemap\naccTitle:    spaced    x   \n"a": 1',
  'treemap\ntitle   spaced    out   \n"a": 1',
  'treemap\ntitle A very long title of a treemap that is much wider than the treemap is, by quite a long way indeed, and wraps\n"a": 1',
  'treemap\ntitle #35; &amp; <b>b</b>\n"a": 1',
  'treemap\ntitlex\n"a": 1',
  'treemap-beta\ntitle T\n"a": 1',
  'TREEMAP\n"a": 1',
  'treemaps\n"a": 1',
  'treemap:\n"a": 1',
  'treemap-beta-beta',
  '',
  'not a treemap',
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
      if (!/treemap/.test(raw)) continue;
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
  const docs = JSON.parse(readFileSync(join(pele, 'tests/corpus/treemap-docs.json'), 'utf8'));
  return [
    ...new Set([
      ...docs,
      ...testStrings(pele, 'tests/compat/treemap', /\.test\.[jt]s$/),
      ...testStrings(pele, 'tests/render', /^treemap.*\.test\.ts$/),
      ...SEEDS,
      ...WRITTEN,
    ]),
  ].filter((s) => !lone(s) && s.length <= 50000);
}

const NAMES = [
  'a', 'Root', 'Category A', 'Item A1', 'Phones', 'Computers', 'Accessories', "Men's", 'Salaries', 'Equipment', 'x', '',
  ' spaced ', 'A name long enough that it never fits in a box of any size', 'two words', '中文标签', '😀 emoji', '#35;', '&amp; more',
  '#quot;q#quot;', '<b>bold</b>', 'iiiiiiiiiiii', 'WWWWWWWW', 'Ünïcödé', 'é', 'mixed 中文 text', 'x\u00a0y', 'trailing   ', 'a,b',
];
const VALUES = [
  '1', '2', '3', '5', '10', '12', '15', '20', '25', '30', '50', '100', '250', '1000', '1,000', '1,234,567', '1_000', '0', '0.5', '.5',
  '5.', '0.001', '1.5', '99.99', '700000', '123456789', '1e3', '_', '.', '9'.repeat(30), '0.0000001', '3.14159', '12,5', '1.2.3',
];
const CLASSES = ['c1', 'c2', 'frame', 'hot', 'cold'];
const STYLES = [
  'fill:red', 'fill:#f9f', 'stroke:blue', 'stroke:#333', 'color:white', 'color:#fff', 'stroke-width:3px', 'stroke-width:0',
  'fill-opacity:0.5', 'stroke-dasharray:4 2', 'stroke-linecap:round', 'stroke-linejoin:round', 'stroke-linejoin:bevel', 'opacity:0.7',
  'font-weight:bold', 'font-size:20px', 'fill:rgb(10\\,20\\,30)', 'fill:red !important', 'fill:url(x)', 'rx:10', 'FILL:green', 'stroke:none',
];
const TITLES = ['T', 'Budget', 'Product categories', 'A title that is much wider than the treemap it names, by a long way at that', '中文', '#35; &amp;'];
const FORMATS = [',', '$0,0', '$,', ',.2f', '.1f', '.0%', ',.1%', '.3e', 'e', 'd', ',d', '~', '.2~f', '+,', '$,.2f', '.3', 'x', '', '>10,.2f', ' ', '(,.2f', '-$,'];

// Random treemaps, most of them well formed, with random configuration.
export function randomDiagrams(rnd) {
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  const chance = (p) => rnd() < p;
  const quote = (name) => (name.includes("'") || chance(0.85) ? `"${name}"` : `'${name}'`);
  return () => {
    const settings = [];
    if (chance(0.5)) {
      if (chance(0.3)) settings.push(`showValues: ${pick(['true', 'false', '"no"', '0'])}`);
      if (chance(0.4)) settings.push(`valueFormat: '${pick(FORMATS)}'`);
      if (chance(0.3)) settings.push(`padding: ${pick([0, 1, 2, 4, 8, 10, 20, 50, 2.5, -1, 300])}`);
      if (chance(0.3)) settings.push(`nodeWidth: ${pick([1, 10, 20, 30, 40, 64, 80, 100, 12.5, 400, 500, 0])}`);
      if (chance(0.3)) settings.push(`nodeHeight: ${pick([1, 5, 10, 20, 30, 40, 60, 100, 7.5, 400, 0])}`);
      if (chance(0.2)) settings.push(`diagramPadding: ${pick([0, 1, 8, 20, 50, 7.5, -3])}`);
    }
    let s = '';
    if (chance(0.1)) s += `---\ntitle: ${pick(TITLES)}\n---\n`;
    if (settings.length) {
      if (chance(0.7)) s += `---\nconfig:\n  treemap:\n${settings.map((l) => '    ' + l).join('\n')}\n---\n`;
      else {
        const json = settings.map((l) => {
          const [, k, v] = /^(\w+): (.*)$/.exec(l);
          return `"${k}": ${v.startsWith("'") ? JSON.stringify(v.slice(1, -1)) : v}`;
        });
        s += `%%{init: {"treemap": {${json.join(', ')}}}}%%\n`;
      }
    }
    s += chance(0.6) ? 'treemap' : 'treemap-beta';
    s += '\n';
    if (chance(0.3)) s += `title ${pick(TITLES)}\n`;
    if (chance(0.1)) s += 'accTitle: Accessible\n';
    if (chance(0.1)) s += chance(0.5) ? 'accDescr: Described\n' : 'accDescr {\n  one\n   two\n}\n';
    const unit = chance(0.2) ? '\t' : ' '.repeat(pick([1, 2, 2, 4]));
    const rows = 1 + Math.floor(rnd() * (chance(0.2) ? 40 : 12));
    let depth = 0;
    const used = new Set();
    for (let k = 0; k < rows; k++) {
      depth = Math.max(0, Math.min(depth + pick([-2, -1, 0, 0, 1, 1]), 5));
      const section = chance(0.35);
      let row = unit.repeat(depth) + quote(pick(NAMES));
      if (!section) row += `${pick([':', ': ', ' : ', ',', ' , ', ':'])}${pick(VALUES)}`;
      if (chance(0.15)) {
        const c = pick(CLASSES);
        used.add(c);
        row += `:::${c}`;
      }
      if (chance(0.05)) row += ' %% comment';
      s += row + (chance(0.05) ? '\r\n' : '\n');
      if (chance(0.05)) s += '\n';
      if (section) depth++;
    }
    for (const c of used) {
      if (chance(0.8)) {
        const n = 1 + Math.floor(rnd() * 3);
        const styles = Array.from({ length: n }, () => pick(STYLES)).join(chance(0.8) ? ',' : ';');
        s += `classDef ${c} ${styles}${chance(0.3) ? ';' : ''}\n`;
      }
    }
    return s;
  };
}

const dumpNode = (n) => ({
  name: n.name,
  children: n.children === undefined ? enc(undefined) : n.children.map(dumpNode),
  value: enc(n.value),
  classSelector: enc(n.classSelector),
  css: enc(n.cssCompiledStyles),
});

export default async function ({ src, pele, write, random }) {
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { treemap } = await src('diagrams/treemap/index.ts');
  const { parseTreemap, TREEMAP_TOKENS } = await src('diagrams/treemap/parser.ts');
  const { tokenize } = await src('diagrams/common/tokens.ts');

  const caught = (e) => {
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  const opt = (x) => (x === undefined || x === '' ? undefined : x);
  const tokens = (source) => {
    try {
      const t = tokenize(source, TREEMAP_TOKENS, 'treemap');
      return { tokens: t.kinds.slice(0, -1).map((k, i) => [TREEMAP_TOKENS[k].name, source.slice(t.starts[i], t.ends[i])]) };
    } catch (e) {
      return { tokens: null };
    }
  };
  const direct = (source) => {
    try {
      const ast = parseTreemap(source);
      return {
        ...tokens(source),
        ast: {
          title: enc(ast.title),
          accTitle: enc(ast.accTitle),
          accDescr: enc(ast.accDescr),
          rows: ast.TreemapRows.map((r) =>
            r.$type === 'ClassDefStatement'
              ? ['classDef', enc(r.indent), r.className, enc(r.styleText)]
              : ['row', enc(r.indent), r.item.$type, r.item.name, enc(r.item.value), enc(r.item.classSelector)]
          ),
        },
      };
    } catch (e) {
      return { ...tokens(source), ...caught(e) };
    }
  };
  // Pele's parseSource for a treemap, and the raw text given to the parser alone.
  const run = (source) => {
    const out = { src: source, direct: direct(source) };
    let pre;
    try {
      pre = preprocess(source);
      out.type = detect(pre.text);
    } catch (e) {
      return { ...out, ...caught(e) };
    }
    if (out.type !== 'treemap') return out;
    try {
      const m = treemap.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title);
      out.model = {
        title: opt(m.title),
        accTitle: opt(m.accTitle),
        accDescr: opt(m.accDescr),
        roots: m.roots.map(dumpNode),
        classes: [...m.classes].map(([k, v]) => [k, v]),
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
  const diagrams = randomDiagrams(random(21));
  for (let i = 0; i < 1500; i++) add(diagrams());
  // Mutated text; one in six is cut short, as a row cut off by the end of the text ends on a
  // different error than one cut off by a line break.
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
  const openers = ['treemap\n', 'treemap ', 'treemap\n"a"', 'treemap\n"a":', 'treemap\n  ', 'treemap\nclassDef', 'treemap\ntitle', 'treemap\naccDescr', ''];
  for (let i = 0; i < 1000; i++) {
    let s = openers[Math.floor(trnd() * openers.length)];
    for (let k = 1 + Math.floor(trnd() * 8); k > 0; k--) s += pieces[Math.floor(trnd() * pieces.length)] + (trnd() < 0.3 ? ' ' : '');
    add(s);
  }
  // Pele's stress cases, made smaller, which stop at limits.
  const N = 3000;
  for (const s of [
    'treemap\n"a": ' + '9'.repeat(N),
    'treemap\n"a": ' + '.,_'.repeat(N / 3),
    'treemap\nclassDef' + ' '.repeat(N) + '1',
    'treemap\n' + 'classDef a\n'.repeat(N / 11),
    'treemap\nclassDef' + '\n '.repeat(N / 2) + 'ab',
    'treemap\n' + 'accDescr\n'.repeat(N / 10) + '{',
    'treemap\naccDescr {' + ' a\n'.repeat(N / 3),
    'treemap\n"' + 'a'.repeat(N),
    'treemap\n' + '"'.repeat(N / 2),
    'treemap\n"a"' + ':'.repeat(N),
    'treemap\n' + ' '.repeat(N) + '"a": 1',
    'treemap\n' + '%% c\n'.repeat(N / 5) + '"a": 1',
    'treemap\n' + Array.from({ length: 60 }, (_, i) => ' '.repeat(i) + `"s${i}"\n`).join('') + ' '.repeat(60) + '"leaf": 1\n',
  ]) {
    seen.delete(s);
    add(s);
  }
  write('diagram/treemap/testdata/pele/treemap.json', { cases: all.map(run) });
}
