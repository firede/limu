// Tree view diagrams parsed as Pele parses them: what preprocessBoxDrawing makes of the text,
// the tokens Pele's common/tokens.ts makes of it with the tree view's table, the tree
// parseTreeView makes of them, and the model buildTreeView makes of the tree, with the icon
// each entry shows and the layout settings under the configuration the text's front matter
// and directives give. The cases are Mermaid's documentation and specs, the inputs of Pele's
// own tests, cases written here, random diagrams, and all of those mutated at random.
import { readFileSync } from 'node:fs';
import { enc, peleError } from '../lib/value.mjs';
import { literals, lone, picker, testFiles, unquote, docs } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';

// The fragments of Pele's tests/treeview/oracle.test.ts, and more.
export const FRAGMENTS = [
  'treeView-beta', 'treeView-beta\n', 'title', 'title ', 'accTitle: ', 'accDescr: ', 'accDescr {', '}', ':::', ' :::', ' :::x',
  ' ::: y', ':::1', 'icon(', ' icon(', ' icon()', ' icon(a:b)', ' icon(a:)', ' icon(none)', ')', '##', ' ##', ' ## d', '#',
  '"', "'", '"a b"', "'c'", '""', '\n', '\r\n', '\r', ' ', '  ', '\t', '    ', '%%', '%% c', '%%{init: {}}%%', '---', '/',
  'src/', 'a.js', '├── ', '└── ', '│   ', 'x', 'é', ':', '::', '-', '_', ' ', '#35;', 'icon', 'a b', ' \n', '"a" b',
  '┣━━ ', '┗━ ', '┃   ', '├─', '─', '━', '│', '├', '└', '┃', '┣', '┗', '\n├── a', '\n│   └── b', '\n    └── c', '\n  d',
  ' :::highlight', ' icon(logos:react)', ' icon(file)', ' icon(folder)', ' icon(rust)', ' ## note', '😀', '中文', '\u3000', '\u00a0',
  '&amp;', '<b>b</b>', 'line 2', 'Line 9', '\t├── t', 'title T\n', 'accTitle: A\n', 'accDescr {\n d\n}\n', '\u2028',
];

// Seeds of Pele's tests/treeview/oracle.test.ts.
const SEEDS = [
  'treeView-beta',
  'treeView-beta\n',
  'treeView-beta\ntitle T\naccTitle: A\naccDescr: D\nroot/\n  a.js :::highlight icon(x:y) ## note\n  "b c" ## d\n',
  'treeView-beta\naccDescr {\n multi\n line\n}\n"a"\n    \'b\'\n',
  'treeView-beta\n%% comment\nsrc/\n    my file.ts ## some description\n    x :::a :::b icon() icon(c)\n',
  'treeView-beta\r\na\r\n  b\r\n',
  'treeView-beta\na:::b icon(c)d ##e\n',
  'treeView-beta\n"a" "b" c\n',
  'treeView-beta :::x\n',
  'treeView-beta title T\n',
];

const config = (lines, body) => `---\nconfig:\n  treeView:\n${lines.map((l) => '    ' + l).join('\n')}\n---\n${body}`;
const TREE = 'treeView-beta\n    my-project/\n        src/\n            index.js\n        package.json\n    README.md\n';
const BOXED = 'treeView-beta\n├── src/\n│   ├── index.ts\n│   └── utils.ts\n├── package.json\n└── README.md\n';

// Cases written here: what the preprocessor, the tokens and the model do at their edges.
export const WRITTEN = [
  TREE,
  BOXED,
  'treeView-beta\n│\n├── src/\n│\n│   └── "a\n',
  'treeView-beta\n├── \n',
  'treeView-beta\n├── a\n   b\n',
  'treeView-beta\n├── a\n├──\n',
  'treeView-beta\n├── a\n└──   \t\n',
  'treeView-beta\n├── a\nb\n├── c\n',
  'treeView-beta\nSection ─ A.txt\n├── x\n',
  'treeView-beta\n├── a\n  Section ─ A.txt\n',
  'treeView-beta\n├── a\n│ │ │\n──── ──\n└── b\n',
  'treeView-beta\n  ├── a\n  │   └── b\n  └── c\n',
  'treeView-beta\n ├── a\n │  └── b\n',
  'treeView-beta\n\t├── a\n\t│\t└── b\n',
  'treeView-beta\n├──a\n├───── b\n└─ c\n┣━━━ d\n┗━ e\n',
  'treeView-beta\n├── a\n%% comment\ntitle late\naccTitle: x\naccDescr: y\naccDescr {\n├── b\n',
  'treeView-beta\n├── a :::x\n│   └── b :::highlight icon(logos:react) ## main\n',
  'treeView-beta\n├── "a\n├── b"\n',
  'treeView-beta\n├── a\n:::x\n',
  'treeView-beta\n├── a\n├── "b\n',
  'treeView-beta\n├── a\n├── b !\n├── c "\n',
  'treeView-beta\n├── a\n├── b\n├── c\n├── d\n├── e\n├── f\n├── g\n├── h\n├── i\n├── j\n├── k "\n',
  'treeView-beta\n├── a\r\n│   └── b\r\n',
  'treeView-beta\r\n├── a\r\n',
  ' treeView-beta \n├── a\n',
  'treeView-beta\n├── 😀 a\n😀├── b\n   😀 ├── c\n',
  'treeView-beta\n├── a\n\u3000├── b\n\u00a0\u00a0\u00a0\u00a0└── c\n',
  'treeView-beta\n├── a\n\u3000b\n',
  'treeView-beta\n├── a\n ── b\n',
  'treeView-beta\n├── a\n│ b\n',
  'treeView-beta\n├── a\n    │   └── deep\n',
  'treeView-beta\n        ├── far\n├── near\n',
  'treeView-beta\n├── a\n      └── b\n',
  'treeView-beta\n├── a\n  └── b\n',
  'treeView-beta\n├── a\n      ├── b\n',
  'not-a-keyword\n├── a\n',
  'treeView-beta x\n├── a\n',
  'treeView-beta\na\n├── a\n',
  'treeView-beta\ntitle T\n├── a\n├── b ## d\n',
  'treeView-beta\n',
  'treeView-beta\n/\n',
  'treeView-beta\n//\n',
  'treeView-beta\n"/"\n',
  'treeView-beta\n""\n',
  "treeView-beta\n''\n",
  'treeView-beta\n"a\nb"\n',
  'treeView-beta\n"a\'b"\n',
  'treeView-beta\na\n b\n  c\n b2\nd\n',
  'treeView-beta\n    a\n  b\n      c\n        d\n    e\n',
  'treeView-beta\n\ta\n\t\tb\n  \tc\n',
  'treeView-beta\na  b   c\n"a  b   c"\n',
  'treeView-beta\na   \n b \t\n',
  'treeView-beta\na :::highlight\nb ::: highlight\nc :::high-light_2\nd :::1\ne :::\n',
  'treeView-beta\na icon(x) icon(y)\nb icon()\nc icon(a:b)\nd icon(a:)\ne icon(:b)\nf icon(a-b:c-d)\n',
  'treeView-beta\na ## d ## e\nb ##\nc ##   spaced   \nd ##\u3000x\u3000\n',
  'treeView-beta\na :::x icon(y) ## z\nb ## z :::x\nc icon(y) :::x\n',
  'treeView-beta\na:::x\nb icon(y)\nc##z\n',
  'treeView-beta\na :::x b\n',
  'treeView-beta\na icon(y) b\n',
  'treeView-beta\na\tb\t:::c\n',
  'treeView-beta\n:::x\n',
  'treeView-beta\nicon(x)\n',
  'treeView-beta\n##x\n',
  'treeView-beta\n  \n  a',
  'treeView-beta\n a\ntitle late',
  'treeView-beta title x\n a',
  'treeView-beta\n :::x',
  'treeView-beta\ntitle\na\n',
  'treeView-beta\ntitle   spaced   out  \na\n',
  'treeView-beta\naccTitle:   A  \naccDescr:   D  \na\n',
  'treeView-beta\naccDescr {\n  line one\n    line two\n}\na\n',
  'treeView-beta\naccDescr {\na\n',
  'treeView-beta\naccTitle: A\naccTitle: B\ntitle C\ntitle D\na\n',
  'treeView-beta\na %% c\nb\n',
  'treeView-beta\na%%c\n',
  'treeView-beta\n%%{init: {}}%%\na\n',
  'treeView-beta%%c\na\n',
  'treeView-beta %% c\na\n',
  'treeView-beta\n\n\na\n\n\nb\n',
  'treeView-beta\na\rb\n',
  '---\ntitle: Front\n---\ntreeView-beta\naccTitle: Acc\naccDescr: Described\n  a\n',
  '---\ntitle: Front\n---\ntreeView-beta\ntitle Own\n  a\n',
  '---\ntitle: Front\n---\ntreeView-beta\ntitle\n  a\n',
  'treeView-beta\n  "a  b   c"\n  <T>.ts ## x <br> y\n  #35;hash\n',
  'treeView-beta\n  App.tsx icon(logos:react)\n  index.js\n  x.rs icon(rust)\n  y icon()\n',
  config(['showIcons: true'], TREE),
  config(['showIcons: true', 'defaultIconPack: material', 'filenameIcons:', '  Dockerfile: docker', 'extensionIcons:', '  .ts: typescript', '  txt: none'],
    'treeView-beta\n  a.ts\n  Dockerfile\n  notes.txt\n  constructor\n  dir.ts/\n'),
  config(['showIcons: true', 'extensionIcons:', '  .TS: upper', '  ts: lower', '  .js: ""', '  json: 5', '  md: none'],
    'treeView-beta\n  a.ts\n  b.TS\n  c.js\n  d.json\n  e.md\n  .ts\n  f.\n  g.tar.gz\n  h\n'),
  config(['showIcons: true', 'filenameIcons:', '  a.ts: ""', '  b.ts: none', '  c: [1]', '  "x y": spaced', 'extensionIcons:', '  .ts: ts'],
    'treeView-beta\n  a.ts\n  b.ts\n  c\n  x y\n  dir/\n'),
  config(['showIcons: "false"'], TREE),
  config(['showIcons: 0'], TREE),
  config(['showIcons: 1', 'defaultIconPack: 5'], 'treeView-beta\n  a icon(x)\n  b icon(file)\n  c icon(p:x)\n'),
  config(['defaultIconPack: mdi'], 'treeView-beta\n  a icon(x)\n  b icon(folder)\n  c icon(p:x)\n  d icon(none)\n  e icon()\n'),
  config(['showIcons: true', 'filenameIcons: [1, 2]', 'extensionIcons: x'], 'treeView-beta\n  a.ts\n'),
  config(['rowIndent: 80', 'paddingY: 10', 'lineThickness: 3'], TREE),
  config(['rowIndent: 0', 'paddingX: 0', 'paddingY: 0', 'lineThickness: 0.25'], TREE),
  config(['rowIndent: 401', 'paddingX: 101', 'paddingY: -1', 'lineThickness: 13'], TREE),
  config(['rowIndent: 400', 'paddingX: 100', 'paddingY: 100', 'lineThickness: 12'], TREE),
  config(['rowIndent: "20"', 'paddingX: .nan', 'paddingY: .inf', 'lineThickness: 0.1'], TREE),
  config(['rowIndent: 7.5', 'paddingX: 2.25', 'paddingY: 0.5', 'lineThickness: 1.5'], TREE),
  '---\nconfig:\n  treeView: 5\n---\n' + TREE,
  '---\nconfig:\n  treeView: [1]\n---\n' + TREE,
  '%%{init: {"treeView": {"showIcons": true, "extensionIcons": {".js": "javascript"}}}}%%\n' + TREE,
  '%%{init: {"treeView": {"rowIndent": 30}}}%%\n' + TREE,
  'treeView-beta\n' + 'a/\n'.repeat(3) + '  b\n',
  '',
  'not a tree',
  'treeview-beta\na\n',
  'treeView\na\n',
];

// Strings in the sources of a folder of Pele's tests that hold the keyword, as Pele's
// loadCorpus collects them, from files whose names match; and the arrays of strings the box
// drawing specs join with line feeds.
function treeStrings(pele, dir, files) {
  const out = new Set();
  for (const file of testFiles(pele, dir, files)) {
    const src = readFileSync(file, 'utf8');
    for (const s of literals(src, /treeView/)) out.add(s);
    for (const m of src.matchAll(/\[((?:\s*'(?:[^'\\\n]|\\.)*',?)+)\s*\]\.join\(\s*'\\n'\s*\)/g)) {
      const lines = [...m[1].matchAll(/'((?:[^'\\\n]|\\.)*)'/g)].map((s) => unquote(s[1]));
      if (!lines.includes(undefined)) out.add(lines.join('\n'));
    }
  }
  return [...out];
}

// The corpus the suites start from: documentation, specs, the strings of Pele's tests, its
// seeds and the cases above.
export function corpus(pele) {
  return [
    ...new Set([
      ...docs(pele, 'treeview'),
      ...treeStrings(pele, 'tests/compat/treeview', /\.(spec|test)\.[jt]s$/),
      ...treeStrings(pele, 'tests/treeview', /\.test\.ts$/),
      ...treeStrings(pele, 'tests/render', /^treeview.*\.test\.ts$/i),
      ...SEEDS,
      ...WRITTEN,
    ]),
  ].filter((s) => !lone(s) && s.length <= 50000);
}

const FILES = [
  'index.js', 'package.json', 'README.md', 'App.tsx', 'utils.ts', 'Dockerfile', '.env', '.gitignore', 'notes.txt', 'main.go', 'go.mod',
  'a', 'x.rs', 'my file.ts', 'tsconfig.json', 'LICENSE', 'Makefile', 'image.PNG', 'archive.tar.gz', 'a.b.c.d', '中文.md', '😀.txt',
  'very-long-file-name-that-keeps-on-going-and-going.config.ts', '#35;hash', '&amp;amp', '<T>.ts', 'iiiiiiii', 'WWWWWWWW', 'a b c',
  'trailing.', '.', 'constructor', 'toString', '__proto__', 'é.js', 'tab\there',
];
const DIRS = ['src', 'lib', 'packages', 'test', 'docs', 'node_modules', 'components', 'a', 'my folder', '中文', '🚀 rocket', '.github', 'x'.repeat(30)];
const DESCRIPTIONS = ['entry point', 'main component', 'environment variables', 'x', 'a much longer description of what this file does in the project', '中文说明', '😀', '#35; &amp;', '<b>b</b>', 'spaced   out', 'word '.repeat(12)];
const CLASSES = ['highlight', 'highlight', 'mine', 'a-b', '_c', 'x1', 'Highlight'];
const ICONS = ['logos:react', 'rust', 'file', 'folder', 'none', '', 'mdi:home', 'material:typescript', 'p:x', 'a-b', 'fa:fa-car', 'mermaid-treeview:folder'];
const TITLES = ['T', 'Project', 'A title that is much wider than the tree it names, by a long way', '中文', '#35; &amp;', 'fa:fa-car icons', '<b>bold</b> title'];

// Random tree views, most of them well formed, with random configuration.
export function randomDiagrams(rnd) {
  const pick = picker(rnd);
  const chance = (p) => rnd() < p;
  return () => {
    const settings = [];
    if (chance(0.5)) {
      if (chance(0.5)) settings.push(`showIcons: ${pick(['true', 'true', 'false', '1', '0', '"no"'])}`);
      if (chance(0.3)) settings.push(`defaultIconPack: ${pick(['material', 'mdi', 'logos', '""', '5'])}`);
      if (chance(0.3)) settings.push(`filenameIcons: {${pick(['Dockerfile: docker', 'README.md: readme, LICENSE: none', 'package.json: npm', 'a: x'])}}`);
      if (chance(0.3)) settings.push(`extensionIcons: {${pick(['.ts: typescript', 'ts: ts, .js: js', '.md: none, .txt: text', '.json: "", json: j', '.TSX: react'])}}`);
      if (chance(0.3)) settings.push(`rowIndent: ${pick([0, 5, 10, 15, 20, 40, 80, 400, 401, -1, 7.5])}`);
      if (chance(0.3)) settings.push(`paddingX: ${pick([0, 1, 5, 10, 20, 100, 2.5, 101])}`);
      if (chance(0.3)) settings.push(`paddingY: ${pick([0, 1, 2, 5, 10, 30, 0.5, 100])}`);
      if (chance(0.2)) settings.push(`lineThickness: ${pick([0.25, 0.5, 1, 1.5, 2, 3, 12, 0.1, 13])}`);
    }
    let s = '';
    if (chance(0.1)) s += `---\ntitle: ${pick(TITLES)}\n---\n`;
    if (settings.length) {
      if (chance(0.75)) s += `---\nconfig:\n  treeView:\n${settings.map((l) => '    ' + l).join('\n')}\n---\n`;
      else {
        const json = settings
          .map((l) => l.replace(/^(\w+): (.*)$/, (_, k, v) => {
            if (v.startsWith('{')) v = '{' + v.slice(1, -1).split(', ').map((e) => e.replace(/^([^:]+): (.*)$/, (_, a, b) => `"${a}": ${b.startsWith('"') ? b : `"${b}"`}`)).join(', ') + '}';
            else if (!/^[-\d.]+$|^true$|^false$|^"/.test(v)) v = `"${v}"`;
            return `"${k}": ${v}`;
          }))
          .join(', ');
        s += `%%{init: {"treeView": {${json}}}}%%\n`;
      }
    }
    s += 'treeView-beta\n';
    if (chance(0.3)) s += `title ${pick(TITLES)}\n`;
    if (chance(0.1)) s += 'accTitle: Accessible\n';
    if (chance(0.1)) s += chance(0.5) ? 'accDescr: Described\n' : 'accDescr {\n  one\n   two\n}\n';
    const boxed = chance(0.3);
    const unit = boxed ? 4 : pick([2, 4, 4, 1, 3, 8]);
    const tabs = !boxed && chance(0.1);
    const count = 1 + Math.floor(rnd() * (chance(0.2) ? 40 : 12));
    // Each entry's depth, at most one deeper than the one before it.
    const depths = [];
    let depth = 0;
    for (let k = 0; k < count; k++) {
      depth = k === 0 ? 0 : Math.max(0, Math.min(depth + 1, depth + 1 - Math.floor(rnd() * 3)));
      if (chance(0.05)) depth = Math.floor(rnd() * 4);
      depths.push(depth);
    }
    for (let k = 0; k < count; k++) {
      const d = depths[k];
      // A directory is what the next entry goes into, and now and then an empty one.
      const dir = (k + 1 < count && depths[k + 1] > d) || chance(0.1);
      let name = dir ? pick(DIRS) + '/' : pick(FILES);
      if (chance(0.1)) name = chance(0.5) ? `"${name}"` : `'${name}'`;
      let line;
      if (boxed) {
        let last = true;
        for (let j = k + 1; j < count && depths[j] >= d; j++) if (depths[j] === d) { last = false; break; }
        let prefix = '';
        for (let level = 0; level < d; level++) {
          let open = false;
          for (let j = k + 1; j < count && depths[j] >= level; j++) if (depths[j] === level) { open = true; break; }
          prefix += open ? '│   ' : '    ';
        }
        const heavy = chance(0.1);
        line = d === 0 && chance(0.3) ? name : prefix + (last ? (heavy ? '┗━━ ' : '└── ') : heavy ? '┣━━ ' : '├── ') + name;
        if (d === 0 && line === name) line = name;
      } else {
        const indent = tabs ? '\t'.repeat(d + 1) : ' '.repeat(unit * (d + 1));
        line = (chance(0.1) && d === 0 ? '' : indent) + name;
      }
      if (chance(0.15)) line += ` :::${pick(CLASSES)}`;
      if (chance(0.15)) line += ` icon(${pick(ICONS)})`;
      if (chance(0.25)) line += ` ## ${pick(DESCRIPTIONS)}`;
      if (chance(0.05)) line += ' %% comment';
      s += line + (chance(0.05) ? '\r\n' : '\n');
      if (chance(0.03)) s += boxed && chance(0.5) ? '│\n' : '\n';
      if (chance(0.02)) s += '%% a comment\n';
    }
    return s;
  };
}

export default async function ({ src, pele, write, random }) {
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { treeView } = await src('diagrams/treeview/index.ts');
  const { parseTreeView, TREEVIEW_TOKENS } = await src('diagrams/treeview/parser.ts');
  const { preprocessBoxDrawing } = await src('diagrams/treeview/boxDrawing.ts');
  const { getNodeIcon, iconConfig } = await src('diagrams/treeview/icons.ts');
  const { treeSettings } = await src('diagrams/treeview/render.ts');
  const { tokenize } = await src('diagrams/common/tokens.ts');

  const caught = (e) => {
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  const opt = (x) => (x === undefined || x === '' ? undefined : x);
  const tokens = (source) => {
    try {
      const t = tokenize(source, TREEVIEW_TOKENS, 'treeView');
      return { tokens: t.kinds.slice(0, -1).map((k, i) => [TREEVIEW_TOKENS[k].name, source.slice(t.starts[i], t.ends[i])]) };
    } catch (e) {
      return { tokens: null };
    }
  };
  const direct = (source) => {
    try {
      const ast = parseTreeView(source);
      return {
        ...tokens(source),
        ast: {
          title: enc(ast.title),
          accTitle: enc(ast.accTitle),
          accDescr: enc(ast.accDescr),
          nodes: ast.nodes.map((n) => [enc(n.indent), n.name, enc(n.classAnnotation), enc(n.iconAnnotation), enc(n.descAnnotation)]),
        },
      };
    } catch (e) {
      return { ...tokens(source), ...caught(e) };
    }
  };
  const box = (source) => {
    try {
      const { text, lineMap } = preprocessBoxDrawing(source);
      return { text, lineMap: [...lineMap] };
    } catch (e) {
      return caught(e);
    }
  };
  const node = (n, icons) => [n.id, n.level, n.name, n.nodeType, enc(n.icon), enc(n.cssClass), enc(n.description), enc(getNodeIcon(n, icons)), n.children.map((c) => node(c, icons))];
  // Pele's parseSource for a tree view, and the raw text given to the parser alone.
  const run = (source) => {
    const out = { src: source, direct: direct(source), box: box(source) };
    let pre;
    try {
      pre = preprocess(source);
      out.type = detect(pre.text);
    } catch (e) {
      return { ...out, ...caught(e) };
    }
    if (out.type !== 'treeView') return out;
    try {
      const m = treeView.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title);
      const icons = iconConfig(pre.config);
      out.model = {
        title: opt(m.title),
        accTitle: opt(m.accTitle),
        accDescr: opt(m.accDescr),
        count: m.count,
        root: node(m.root, icons),
        icons: [icons.showIcons, icons.defaultIconPack],
        settings: Object.values(treeSettings(pre.config)).map((v) => enc(v)),
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
  const diagrams = randomDiagrams(random(71));
  for (let i = 0; i < 1500; i++) add(diagrams());
  // Mutated text; one in six is cut short, as a statement cut off by the end of the text ends
  // on a different error than one cut off by a line break.
  const rnd = random(72);
  const next = mutator([...seen].filter((s) => s.length <= 4000), FRAGMENTS, rnd);
  for (let i = 0; i < 4500; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  // Random strings of tokens, where the order of the token types and their ends matter.
  const trnd = random(73);
  const pieces = [...new Set([...FRAGMENTS, ...FRAGMENTS.map((f) => f.trim()).filter((f) => f !== '')])];
  const openers = ['treeView-beta\n', 'treeView-beta ', 'treeView-beta\n  a', 'treeView-beta\n├── ', 'treeView-beta\na ', 'treeView-beta\ntitle', 'treeView-beta\naccDescr', ''];
  for (let i = 0; i < 1500; i++) {
    let s = openers[Math.floor(trnd() * openers.length)];
    for (let k = 1 + Math.floor(trnd() * 8); k > 0; k--) s += pieces[Math.floor(trnd() * pieces.length)] + (trnd() < 0.3 ? ' ' : '');
    add(s);
  }
  // Pele's stress cases, cut to a size the fixture holds.
  const HEAD = 'treeView-beta\n';
  for (const s of [
    HEAD + 'a\n'.repeat(1000),
    HEAD + Array.from({ length: 60 }, (_, i) => ' '.repeat(i) + 'd/\n').join(''),
    HEAD + 'a' + ' '.repeat(2000) + 'b',
    HEAD + ('a :::' + ' '.repeat(95)).repeat(20) + '1',
    HEAD + 'a' + ' ::: 1 icon[ #'.repeat(150),
    HEAD + 'accDescr {\n'.repeat(200),
    HEAD + 'a\n' + ' accDescr {\n'.repeat(200),
    HEAD + '"a\n'.repeat(500),
    HEAD + 'a' + ' :::highlight'.repeat(150),
    HEAD + '├── a\n'.repeat(500),
    HEAD + Array.from({ length: 60 }, (_, i) => '│'.repeat(i) + '├─ x\n').join(''),
    HEAD + '├ a\n' + (' '.repeat(490) + '└ b\n').repeat(4),
    HEAD + '│   │   │\n'.repeat(300),
    HEAD + ('\t'.repeat(40) + '├── a\n').repeat(40),
    HEAD + 'a ─ b\n'.repeat(500),
  ]) {
    seen.delete(s);
    add(s);
  }
  write('diagram/treeview/testdata/pele/treeview.json', { cases: all.map(run) });
}
