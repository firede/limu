// Block diagrams parsed as Pele parses them, from text to tokens and to the model BlockDb
// fills: Mermaid's documentation and specs, the inputs of Pele's own tests, cases written here,
// random diagrams, those mutated at random, and random strings of tokens. Each case is lexed
// and parsed as the raw text, and parsed as Pele's parse does it, after the preprocessing every
// diagram goes through.
import { enc, peleError } from '../lib/value.mjs';
import { lone, picker, testStrings, docs } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { locator } from '../lib/located.mjs';

// The fragments of Pele's tests/block/oracle.test.ts, and more.
export const FRAGMENTS = [
  ' ', '\n', '\t', '\r\n', ' ', 'block', 'block-beta', 'block:', 'block:g', 'block\n', 'end', 'end\n', ' end ',
  'columns 3', 'columns auto', 'columns ', 'columns 0', 'columns\n2', 'space', 'space:2', 'space:', ':2', ':', ':03',
  '-->', '---', '--', '--x', '--o', 'x--', 'o--', '<--', '<-->', '==>', '===', '==', '<==>', '-.->', '-.-', '-.',
  '.->', '<-.->', '-..-', '~~~', '->', '-', '=', '.', ' -- "x" --> ', ' == "x" ==> ', ' -. "x" .-> ', '"', '`',
  '"`', '`"', '"x"', '""', '["x"]', '("x")', '(("x"))', '((("x")))', '{"x"}', '{{"x"}}', '(["x"])', '[["x"]]',
  '[("x")]', '>"x"]', '[/"x"/]', '[\\"x"\\]', '[/"x"\\]', '[\\"x"/]', '(-"x"-)', '[|"x"]', '<["x"]>(right)',
  '<["x"]>(up, down)', '<["x"]>( x )', '<["x"]>(x,y)', '<[', ']>', ']>(', '(', ')', '((', '))', '(((', ')))', '[',
  ']', '[[', ']]', '{', '}', '{{', '}}', '([', '])', '[(', ')]', '[/', '/]', '[\\', '\\]', '(-', '-)', '>', '<',
  '[|', ',', ', ', 'right', 'left', 'up', 'down', 'x', 'y', 'o', 'A', 'B', 'id1', 'a b', '1', '23', 'é', ';',
  'classDef ', 'classDef a fill:#f96', 'classDef DEFAULT x', 'classDef a\n', 'class ', 'class A a', 'class A,B a',
  'class A, B a', 'style ', 'style A fill:#f96,stroke:#333', 'style A,B x', 'default', 'linkStyle', 'interpolate',
  'accTitle: t', 'accDescr: d', 'accDescr { d }', 'accDescr {}', 'accTitle', '%%', '#', '&nbsp;', '__proto__',
  'constructor', ' A --> A ', ' A --> B ', '\nA --> B\n', 'space:3', 'block:B:2\n', 'columns 2\n', '\u3000', '😀', '中文',
  '#35;', '<br>', 'classDef c stroke-linejoin:round\n', '\nstyle A font-weight:bold;font-size:20px\n', 'columns 99999999999999999999',
];

// The inputs of Pele's tests/block/oracle.test.ts beyond its corpus.
const SEEDS = [
  'block\n A --> B --> C',
  'block\n A --> B --> C --> D\n E',
  'block\n A["a"] -- "x" --> B(("b")) ==> C\n D --> E\n',
  'block\n block:A --> B --> C\n D\n end\n E --> F --> G',
  'block\n block\n A --> B --> C\n end',
];

const configured = (lines, body) => `---\nconfig:\n  block:\n${lines.map((l) => '    ' + l).join('\n')}\n---\n${body}`;

// Cases written here: what the tokens and the model do at their edges, and what Pele draws.
export const WRITTEN = [
  '',
  'block',
  'block\n',
  'block-beta\na',
  'block-betax\na',
  'blocks\na',
  'block a',
  'block\na b c',
  'block\ncolumns 3\na b c d e f g',
  'block\ncolumns auto\na b c',
  'block\ncolumns autox\na',
  'block\ncolumns 0\na b',
  'block\ncolumns 0',
  'block\ncolumns 1\na b c',
  'block\ncolumns 2\na:2 b c:3 d',
  'block\ncolumns 3\na:5 b',
  'block\na:0 b:00 c',
  'block\na:' + '9'.repeat(400) + ' b c',
  'block\ncolumns 3\na:' + '9'.repeat(400) + ' b c',
  'block\ncolumns ' + '9'.repeat(400) + '\na b',
  'block\ncolumns 99999999999999999999\na b',
  'block\nblock:g:99999999\na\nend\nb',
  'block\ncolumns\n2\na',
  'block\ncolumns 2 a b c',
  'block\ncolumns 2\ncolumns 3\na b c d',
  'block\na\ncolumns 2\nb c',
  'block\nspace\na',
  'block\nspace:3\na',
  'block\na space:2 b',
  'block\ncolumns 3\nspace a space:2 b space',
  'block\nspace:0 a',
  'block\nspace:',
  'block\nspacex',
  'block\nspace:100001',
  'block\nspace:60000 space:60000',
  'block\na space:5 b',
  'block\na["A label"]',
  'block\na("round")\nb(("circle"))\nc((("double")))\nd{"diamond"}\ne{{"hex"}}\nf(["stadium"])\ng[["sub"]]\nh[("cyl")]\ni>"odd"]\nj[/"lean r"/]\nk[\\"lean l"\\]\nl[/"trap"\\]\nm[\\"inv"/]',
  'block\na(-"x"-)\nb[|"y"]\nc)"z"(\nd{"w"}}',
  'block\na[x]',
  'block\na["x"',
  'block\na["x"]]',
  'block\na[""]',
  'block\na["`**md** text`"]',
  'block\na["`unclosed"]',
  'block\na["`x"`"]',
  'block\na<["arrow"]>(right)',
  'block\na<["arrow"]>(left)\nb<["arrow"]>(up)\nc<["arrow"]>(down)\nd<["arrow"]>(x)\ne<["arrow"]>(y)',
  'block\na<["both"]>(right, left)\nb<["ways"]>(up,down)\nc<["corner"]>(up, right)\nd<["three"]>(up, right, down)\ne<["all"]>(up, down, left, right)',
  'block\na<["&nbsp;&nbsp;"]>(down)',
  'block\na<[" "]>(right)',
  'block\na<["x"]>( up )',
  'block\na<["x"]>()',
  'block\na<["x"]>(sideways)',
  'block\na<["x"]>(right',
  'block\na<["x"]>  (right)',
  'block\na<["x"]',
  'block\na<["x"]>(right):3 b',
  'block\ncolumns 3\na<["wide arrow"]>(right):3\nb c d',
  'block\ncolumns 3\na<["tall"]>(down):2\nb c d',
  'block\nA --> B',
  'block\nA --- B',
  'block\nA -.-> B',
  'block\nA ==> B',
  'block\nA <--> B',
  'block\nA x--x B',
  'block\nA o--o B',
  'block\nA --x B\nA --o B',
  'block\nA ~~~ B',
  'block\nA -> B',
  'block\nA -- "label" --> B',
  'block\nA == "thick label" ==> B',
  'block\nA -. "dotted" .-> B',
  'block\nA -- "unclosed --> B',
  'block\nA -- "x" B',
  'block\nA --> A',
  'block\nA --> A\nA --> A\nA -- "loop" --> A',
  'block\na(("c")) --> a\nb{"d"} --> b',
  'block\nA space B\nA --> B\nA --> B\nB --> A',
  'block\nA space B\nA -- "one" --> B\nA -- "two" --> B',
  'block\ncolumns 3\nA space B\nspace C space\nA --> C\nC --> B\nA --> B',
  'block\ncolumns 3\nA B C\nD E F\nA --> F\nD --> C\nA --> E',
  'block\ncolumns 2\nA B\nC D\nA --> D\nB --> C',
  'block\nA["a"] --> B["b"] --> C["c"]',
  'block\nA:2 --> B',
  'block\nA --> B:2',
  'block\nA --> Ghost',
  'block\nblock:G\nA B\nend\nG --> A',
  'block\nblock:G\nA\nend\nA --> G',
  'block\nblock\na\nend',
  'block\nblock\nend',
  'block\nend',
  'block\na\nend',
  'block\nblock:\na\nend',
  'block\nblock:g\na\nend',
  'block\nblock:g["Title"]\na\nend',
  'block\nblock:g:2\na\nend\nb',
  'block\ncolumns 3\nblock:g:2\ncolumns 2\na b c\nend\nd',
  'block\nblock:g\nblock:h\nblock:i\na\nend\nend\nend',
  'block\nblock:g\na\nblock\nb\nend\nc\nend',
  'block\nblock:A --> B\nC\nend',
  'block\nblock:A --> B --> C\nD --> E\nend\nF',
  'block\nblock:a\nb\nend\nblock:a\nc\nend',
  'block\na\na["again"]\na("round")',
  'block\na["one"]\na',
  'block\nroot\nroot --> a',
  'block\nid-1 space id-1-0',
  'block\nclassDef hot fill:#f96,stroke:#333\nclass a hot\na',
  'block\na\nclass a hot\nclassDef hot fill:#f96',
  'block\na b\nclass a,b hot\nclassDef hot fill:#f96,color:#fff',
  'block\na b\nclass a, b hot\nclassDef hot fill:#f96',
  'block\na\nclassDef DEFAULT fill:#eee',
  'block\na\nclassDef default fill:#eee,stroke:#aaa',
  'block\na\nclassDef c fill:red;stroke:blue\nclass a c',
  'block\na\nclassDef c color:red,fill:blue,background-color:green\nclass a c',
  'block\na\nclassDef c\nclass a c',
  'block\na\nclassDef',
  'block\na\nclass a',
  'block\na\nclass',
  'block\na\nstyle a fill:#f96,stroke:#333,stroke-width:4px',
  'block\na\nstyle a',
  'block\na\nstyle a,b fill:red',
  'block\nstyle a fill:red\na',
  'block\nstyle ghost fill:red\nclass ghost hot\na',
  'block\na\nstyle a font-weight:bold;font-size:24px',
  'block\na\nstyle a font-style:italic',
  'block\na\nstyle a stroke-linejoin:round,stroke-linecap:square',
  'block\na<["x"]>(right)\nstyle a stroke-linejoin:bevel',
  'block\nblock:g\na\nend\nstyle g fill:red,fill-opacity:0.2',
  'block\nblock:g\na\nend\nstyle g fill:red',
  'block\nblock:g\na\nend\nclass g hot\nclassDef hot stroke:#f00,stroke-linecap:round',
  'block\na\nlinkStyle 0 stroke:red',
  'block\na\ndefault',
  'block\na\ninterpolate',
  'block\naccTitle: t\na',
  'block\naccDescr: d\na',
  'block\naccDescr {\nd\n}\na',
  'block\na %% comment\nb',
  'block\n%% comment\na',
  'block\na;b',
  'block\na-b',
  'block\na:b',
  'block\na=b',
  'block\n中文 😀 é',
  'block\na["中文标签"]\nb["😀 emoji"]',
  'block\na["#35; &amp; &lt;b&gt;"]',
  'block\na["<b>bold</b> <i>it</i>"]\nb["one<br>two<br/>three"]',
  'block\na["a long label that wraps past the width a block has, which is two hundred units, more than once"]',
  'block\nblock:g["a title wider than any of the blocks the composite holds, by some way"]\na\nend',
  'block\na["fa:fa-car car"]',
  'block\n  a\n  b\n',
  '\n\nblock\n\n a \n\n',
  'block\r\na\r\nb',
  '---\ntitle: Block diagram\n---\nblock\na b',
  '---\ntitle: A title much wider than the diagram under it, which is only one block\n---\nblock\na',
  configured(['padding: 4'], 'block\ncolumns 2\na b c\na --> c'),
  configured(['padding: 0'], 'block\ncolumns 2\na b c'),
  configured(['padding: 30'], 'block\nblock:g\na b\nend'),
  configured(['padding: 500'], 'block\na b'),
  configured(['padding: -5'], 'block\na b'),
  configured(['padding: "8"'], 'block\na b'),
  configured(['padding: .inf'], 'block\na b'),
  configured(['padding: .nan'], 'block\na b'),
  configured(['padding: 2.5'], 'block\nblock:g\na\nend\nb'),
  '---\nconfig:\n  block: 5\n---\nblock\na b',
  '%%{init: {"block": {"padding": 20}}}%%\nblock\na b',
  '%%{init: {"block": {"padding": 1}}}%%\nblock\ncolumns 3\na b c\nd e f',
];

// The corpus the suites start from: documentation, specs, the strings of Pele's tests, its
// seeds and the cases above.
export function corpus(pele) {
  return [
    ...new Set([
      ...docs(pele, 'block'),
      ...testStrings(pele, 'tests/compat/block', /\.spec\.[jt]s$/, /block/),
      ...testStrings(pele, 'tests/block', /\.test\.ts$/, /block/),
      ...testStrings(pele, 'tests/render', /^block.*\.test\.ts$/, /block/),
      ...SEEDS,
      ...WRITTEN,
    ]),
  ].filter((s) => !lone(s) && s.length <= 4000 && /block/.test(s));
}

const LABELS = [
  'a', 'x', 'Database', 'A label', 'A wide one in the middle', 'two words', '中文标签', '😀', '#35;', '&amp;', ' ',
  'a long label that wraps past the width a block has, which is two hundred units', 'iiii', 'WWWWWW', 'é', '<b>b</b>',
  'one<br>two', '&nbsp;&nbsp;&nbsp;', 'Q?',
];
const SHAPES = ['', '', '', '[]', '()', '(())', '((()))', '{}', '{{}}', '([])', '[[]]', '[()]', '>]', '[//]', '[\\\\]', '[/\\]', '[\\/]', '<[]>'];
const DIRS = ['right', 'left', 'up', 'down', 'x', 'y'];
const LINKS = ['-->', '---', '-.->', '==>', '<-->', 'x--x', 'o--o', '--x', '--o', '===', '-.-', '<==>'];
const STYLES = [
  'fill:#f96', 'stroke:#333', 'stroke-width:4px', 'color:#fff', 'font-weight:bold', 'font-size:20px', 'font-style:italic',
  'stroke-dasharray:5 5', 'fill-opacity:0.3', 'stroke-linejoin:round', 'stroke-linecap:square', 'stroke-linejoin:bevel',
];

// Random block diagrams, most of them well formed: blocks of every shape, arrows, composites
// nested in each other, spaces, column settings, edges with and without labels, classes and
// styles, a title and the diagram's padding.
export function randomDiagrams(rnd) {
  const pick = picker(rnd);
  const chance = (p) => rnd() < p;
  return () => {
    let s = '';
    const settings = [];
    if (chance(0.15)) settings.push(`padding: ${pick([0, 2, 4, 6, 8, 12, 20, 40, 2.5, -1, 300])}`);
    if (chance(0.1)) s += `---\ntitle: ${pick(['Blocks', 'A wide title of a block diagram, wider than most of them', '中文'])}\n`;
    if (settings.length) s += (s ? '' : '---\n') + `config:\n  block:\n${settings.map((l) => '    ' + l).join('\n')}\n`;
    if (s) s += '---\n';
    s += chance(0.8) ? 'block\n' : 'block-beta\n';
    const ids = [];
    let count = 0;
    const leaf = (depth) => {
      const id = chance(0.1) && ids.length ? pick(ids) : 'n' + count++;
      ids.push(id);
      const shape = pick(SHAPES);
      let text = id;
      if (shape === '<[]>') {
        const dirs = [];
        for (let k = 1 + Math.floor(rnd() * (chance(0.7) ? 1 : 3)); k > 0; k--) dirs.push(pick(DIRS));
        text += `<["${pick(LABELS)}"]>(${dirs.join(pick([', ', ',', ' , ']))})`;
      } else if (shape) {
        const h = shape.length / 2;
        text += `${shape.slice(0, h)}"${pick(LABELS)}"${shape.slice(h)}`;
      }
      if (chance(0.2)) text += ':' + pick([1, 2, 2, 3, 4, 0, 9]);
      return '  '.repeat(depth) + text;
    };
    const body = (depth, budget) => {
      const lines = [];
      if (chance(0.5)) lines.push('  '.repeat(depth) + 'columns ' + pick([1, 2, 2, 3, 3, 4, 5, 'auto']));
      const n = 1 + Math.floor(rnd() * budget);
      let row = [];
      for (let k = 0; k < n; k++) {
        const r = rnd();
        if (r < 0.12 && depth < 3) {
          const head = chance(0.5) ? `block:g${count++}${chance(0.3) ? `["${pick(LABELS)}"]` : ''}${chance(0.3) ? ':' + pick([1, 2, 3]) : ''}` : 'block';
          if (head !== 'block') ids.push(head.slice(6).replace(/[[:].*$/, ''));
          lines.push(...row.splice(0), '  '.repeat(depth) + head, ...body(depth + 1, Math.max(2, budget - 3)), '  '.repeat(depth) + 'end');
        } else if (r < 0.2) {
          row.push('  '.repeat(depth) + (chance(0.5) ? 'space' : 'space:' + pick([1, 2, 3])));
        } else {
          row.push(leaf(depth));
        }
        if (chance(0.4)) lines.push(row.splice(0).map((l, i) => (i ? l.trim() : l)).join(' '));
      }
      if (row.length) lines.push(row.map((l, i) => (i ? l.trim() : l)).join(' '));
      return lines;
    };
    s += body(1, chance(0.3) ? 14 : 7).join('\n') + '\n';
    const edges = Math.floor(rnd() * (ids.length + 2));
    for (let k = 0; k < edges && ids.length; k++) {
      const a = pick(ids);
      const b = chance(0.08) ? a : pick(ids);
      const link = pick(LINKS);
      if (chance(0.25)) {
        const line = link.includes('=') ? '==' : link.includes('.') ? '-.' : '--';
        s += `  ${a} ${line} "${pick(['label', 'a longer label', '中文', 'x'])}" ${link.startsWith('<') || link.length < 3 ? '-->' : link.replace(/^[<xo]/, '').replace(/^-\./, '.-')} ${b}\n`;
      } else {
        s += `  ${a} ${link} ${b}\n`;
      }
    }
    if (chance(0.3) && ids.length) {
      s += `  classDef hot ${pick(STYLES)}${chance(0.5) ? ',' + pick(STYLES) : ''}\n  class ${pick(ids)}${chance(0.3) ? ',' + pick(ids) : ''} hot\n`;
    }
    if (chance(0.25) && ids.length) s += `  style ${pick(ids)} ${pick(STYLES)}${chance(0.5) ? ',' + pick(STYLES) : ''}\n`;
    return s;
  };
}

// The model as the oracle writes it: every field of each block, in a fixed order.
function flat(b) {
  return [
    enc(b.id), enc(b.type), enc(b.label), enc(b.columns), enc(b.widthInColumns), enc(b.width), enc(b.directions),
    enc(b.classes), enc(b.styles), enc(b.colorIndex),
  ];
}
function tree(b) {
  return [...flat(b), b.children === undefined ? enc(undefined) : b.children.map(tree)];
}
export function dumpModel(m) {
  return {
    title: m.title === '' ? undefined : m.title,
    root: tree(m.root),
    blocks: m.blocks.map((b) => b.id),
    edges: m.edges.map((e) => [
      e.id, e.start, e.end, enc(e.label), e.type, e.thickness, e.pattern, enc(e.directions), e.arrowTypeEnd, e.arrowTypeStart,
    ]),
    classes: [...m.classes].map(([k, c]) => [k, c.id, c.styles, c.textStyles]),
    byId: [...m.byId].map(([k, b]) => [k, ...flat(b), b.children === undefined ? enc(undefined) : b.children.map((c) => c.id)]),
  };
}

export default async function ({ src, pele, write, random }) {
  const located = await locator(src);
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { block } = await src('diagrams/block/index.ts');
  const { tokenize } = await src('diagrams/block/lexer.ts');
  const { parseBlock } = await src('diagrams/block/parser.ts');
  const { BlockDb } = await src('diagrams/block/db.ts');

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
        const db = new BlockDb();
        parseBlock(source, db);
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
    if (out.type !== 'block') return out;
    const piped = parsed(() => located(source, false, () => block.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title)));
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
  for (const s of base) add(s + '\n');
  const diagrams = randomDiagrams(random(31));
  for (let i = 0; i < 1500; i++) add(diagrams());
  // Mutated text; one in six is cut short, as a statement cut off by the end of the text ends
  // on a different error than one cut off by a line break.
  const rnd = random(32);
  const next = mutator([...seen], FRAGMENTS, rnd);
  for (let i = 0; i < 4000; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  // Random strings of tokens, where the order of the token types and their ends matter.
  const trnd = random(33);
  const pieces = [...new Set([...FRAGMENTS, ...FRAGMENTS.map((f) => f.trim()).filter((f) => f !== '')])];
  const openers = ['block\n', 'block ', 'block\nA', 'block\nA ', 'block\nA["x"]', 'block\nA<["x"]>(', 'block\nA --', 'block\nclassDef ', 'block\nstyle ', 'block\nblock:', ''];
  for (let i = 0; i < 1500; i++) {
    let s = openers[Math.floor(trnd() * openers.length)];
    for (let k = 1 + Math.floor(trnd() * 8); k > 0; k--) s += pieces[Math.floor(trnd() * pieces.length)] + (trnd() < 0.3 ? ' ' : '');
    add(s);
  }
  // Pele's stress cases, made smaller.
  for (const s of [
    'block\n' + 'block\n'.repeat(300) + 'a\n' + 'end\n'.repeat(300),
    'block\n' + 'block\n'.repeat(600) + 'a',
    'block\na\n' + 'end\n'.repeat(600),
    'block\na0' + Array.from({ length: 300 }, (_, i) => `-->a${i + 1}`).join(''),
    'block\nA<["x"]>(' + 'up, down , x,y '.repeat(200) + ')',
    'block\nA<["x"]>(' + ' '.repeat(3000) + 'sideways)',
    'block\nA\nclass ' + 'A,'.repeat(1500) + 'A hot\nclassDef hot fill:#f96',
    'block\nA\nstyle A ' + ','.repeat(3000),
    'block\nA\nclassDef c ' + 'fill:red,'.repeat(300) + 'x\nclass A c',
    'block\n__proto__ constructor toString\nclassDef __proto__ fill:red\nclass constructor,toString __proto__\nstyle __proto__ fill:blue\n__proto__ --> constructor',
    'block ' + 'a;'.repeat(1500),
    'block\n' + '(([[{{'.repeat(500),
    'block\nA' + '-'.repeat(1000) + '.'.repeat(1000) + '='.repeat(1000),
    'block\nA -' + '.'.repeat(3000) + 'x',
    'block\n' + '--> '.repeat(900),
  ]) {
    add(s);
  }
  // A case whose blocks run to thousands, as space:99999 makes them, is left out: the fixture
  // would hold every one of them.
  const cases = all.map(run).filter((c) => JSON.stringify(c).length <= 60000);
  write('diagram/block/testdata/pele/block.json', { cases });
}
