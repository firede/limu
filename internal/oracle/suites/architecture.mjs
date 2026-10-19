// Architecture diagrams parsed as Pele parses them: the tokens Pele's common/tokens.ts makes of
// the text with the architecture's table, the tree parseArchitecture makes of them, and the
// model populate makes of the tree, after the preprocessing every diagram goes through. The
// cases are Mermaid's documentation and specs, the strings of Pele's own tests, cases written
// here, diagrams built from statements as Pele's test builds them, random diagrams, and all of
// those mutated at random.
//
// A string Pele leaves undefined and one it leaves empty are written alike, as null: nothing
// after the parser tells them apart.
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { lone, mutator, peleError } from './parse.mjs';

// The fragments of Pele's tests/architecture/oracle.test.ts, and more.
export const FRAGMENTS = [
  'architecture-beta', 'architecture', 'service ', 'group ', 'junction ', 'align ', 'row ', 'column ', ' in ', 'in', 'title', 'title ',
  'accTitle: ', 'accDescr: ', 'accDescr {', '}', ':', ' : ', '--', '-', '---', '---\n', '<', '>', '-->', '<--', '<-->', '{group}', '{', 'L', 'R',
  'T', 'B', ':L', 'R:', ':T -- B:', ':R --> L:', ' -[x]- ', '-[', ']-', '(', ')', '(cloud)', '(a:b-c)', '()', '[', ']', '[x]', '[]', '["', '"]',
  "['", "']", '["a b"]', "['a']", '[\\"]', '"', "'", '"t"', '\\', '\\"', "\\'", '\n', '\r\n', ' ', '\t', '%%', '%% c', '%%{init: {}}%%', 'a',
  'b', 'db', 'x-y', 'a-', '-a', '_', '1', 'é', '采', ';', ',', '#35;', 'service a', '\nservice b(server)[B] in g', '\ngroup g(cloud)[G]',
  '\na:L -- R:b', '\njunction j', '\nalign row a b',
  // And for what the db checks and Pele draws.
  '(database)', '(disk)', '(internet)', '(server)', '(logos:aws)', '[A long title for a service that wraps]', '[**b** _i_]',
  '[fa:fa-car x]', '[a<br>b]', '[&amp;]', ' ', '　', '😀', '中文', '\r', 'a{group}', '{group}:R', ' in a', ' in g',
  '\nservice c in a', '\ngroup g in g', '\nalign column a a', '\nalign row a', '\na:R -- L:g',
];

// The inputs of Pele's tests/architecture/oracle.test.ts beyond its corpus.
const SEEDS = [
  'architecture-beta',
  'architecture-beta title T\nservice a(server)[A]\nservice b "txt" [B]\na:R --> L:b',
  'architecture-beta\n accTitle: at\n accDescr { multi\n line }\n title t\n group g(cloud)[G]\n service a in g\n junction j in g\n a:B -[lbl]- T:j',
  'architecture-beta\ngroup one\ngroup two in one\nservice a(disk)["Q \\" q"] in two\nservice b(x:y-z)[\'s \\\' s\'] in one\na{group}:T <--> B:b{group}',
  '---\ntitle: x\n---\narchitecture-beta\nservice a\nservice b\nalign column a b',
  '%%{init: {"theme":"dark"}}%%\narchitecture-beta %% c\nservice a %% d\nservice b\na:L<-[ t ]->R:b %% e\n',
  'architecture-beta\nservice a\nservice b\nservice c\na:R -- L:b\nb:R -- T:c\nalign row a b c\n',
  'architecture-beta\nservice a\nservice b\na:R -- L:b\n---\nskipped: yes\n---\nservice c\nb:B -[x]- T:c\n  ---\n---\n',
  'architecture-beta\nservice a\n  %%{init: {}}%%\nservice b %%{ x }%%\n',
  'architecture-beta\nservice a\nservice b\na:R >--< L:b\na:T >-[x]-< B:b\na{group}:L <-- R:b\na:B --> T:b{group}\na : L - [ y ] - R : b\n',
  'architecture-beta\ngroup g\ngroup h(a-b:c)["t"] in g\nservice s \'i\'[t] in h\nservice in-x(_)[ ] in g\njunction j in g\nalign column s j in-x\n',
];

const head = 'architecture-beta\n';
const config = (lines, body) => `---\nconfig:\n  architecture:\n${lines.map((l) => '    ' + l).join('\n')}\n---\n${body}`;

// Cases written here: the db's errors, what the tokens do at their edges, and the cases of
// Pele's render tests.
export const WRITTEN = [
  head + 'service a\nservice a',
  head + 'group a\njunction a',
  head + 'group a\ngroup a',
  head + 'service a\ngroup a',
  head + 'junction a\nservice a',
  head + 'service a in g',
  head + 'junction j in g',
  head + 'group h in g',
  head + 'group g in g',
  head + 'group g\nservice g in g',
  head + 'service a\nservice b in a',
  head + 'junction a\ngroup g in a',
  head + 'service a\na:R -- L:b',
  head + 'service b\na:R -- L:b',
  head + 'group g\nservice a in g\nservice b in g\na{group}:R -- L:b',
  head + 'group g\nservice a in g\nservice b in g\na:R -- L:b{group}',
  head + 'group g\nservice a in g\nservice b in g\na{group}:R -- L:b{group}',
  head + 'group g\ngroup h\nservice a in g\nservice b in h\na{group}:R -- L:b{group}',
  head + 'group g\nservice a in g\nservice b\na{group}:R -- L:b',
  head + 'service a\nservice b\nalign row a b c',
  head + 'service a\nservice b\nalign column a a',
  head + 'service a\ngroup g\nalign row a g',
  head + 'service a\nalign row a',
  head + 'group g\nservice a\na:R -- L:g',
  head + 'group g\nservice a\ng:R -- L:a',
  head + 'group g\ngroup h\ng:R -- L:h',
  'architecture\nservice a',
  head + 'service Lambda',
  head + 'service lambda',
  head + 'service bravo',
  head + 'service Tango',
  head + 'service x\nservice R2\nx:R -- L:R2',
  head + 'title T\ngroup g(cloud)[G]\nservice a(server)[A] in g\njunction j\na:R -[x]-> L:j\nalign row a j',
  head + 'group g[Nothing here]\ngroup h in g',
  head + 'service a\nservice b\na:R -- L:b\na:R --> L:b\na:R <-- L:b\na:R <--> L:b',
  head + 'service a\na:R -- L:a',
  head + 'service a[]\nservice b[ ]\nservice c[""]\nservice d[\'\']\nservice e["]\nservice f[\'"]',
  head + 'service a["x"y"]\nservice b["x\\"]\nservice c["a\nb"]\nservice d["a\\\nb"]',
  head + 'service a[a[b]]\nservice b[a]b]\nservice c[a\rb]',
  head + 'service a(x)(y)\nservice b()[t]\nservice c(a b)\nservice d(é)',
  head + 'service a "t"(x)\nservice b(x) "t"\nservice c "t" [T] in g',
  head + 'service a "line\\nbreak"\nservice b \'q\\\'s\'\nservice c ""',
  head + 'service a\nservice b\na:L -- R:b\na:T -- B:b\na:B -- T:b\na:R -- R:b\na:L -- L:b\na:T -- T:b\na:B -- B:b',
  head + 'service a\nservice b\na:X -- L:b',
  head + 'service a\nservice b\na:R - L:b',
  head + 'service a\nservice b\na:R --- L:b',
  head + 'service a\nservice b\na:R -[t] L:b',
  head + 'service a\nservice b\na:R -[t]-> L:b{group}',
  head + 'service a\nservice b\na R -- L:b',
  head + 'service a\nservice b\na:R -- L b',
  head + 'service a\nservice b\na:R -- L:',
  head + 'service a\nservice b\na:R --',
  head + 'service a\nservice b\na:R',
  head + 'service a\nservice b\na:',
  head + 'service a\nservice b\na',
  head + 'service',
  head + 'group',
  head + 'junction',
  head + 'align',
  head + 'align row',
  head + 'align row a',
  head + 'align diagonal a b',
  head + 'service a in',
  head + 'service a service b',
  head + 'service a\n\n\n\nservice b\r\nservice c\r\n',
  head + 'service a %% c\n%% d\n  %% e\nservice b',
  head + 'title\nservice a',
  head + 'title   spaced    out   \nservice a',
  head + 'title A\ntitle B',
  head + 'titles\nservice a',
  head + 'accTitle: x\naccTitle: y',
  head + 'accTitle:   spaced  ',
  head + 'accDescr: one\naccDescr {\n  two\n   three\n}',
  head + 'accDescr {\n\t\n  a  b \r\n\r\n c\n}',
  head + 'accDescr {',
  'architecture-beta title Simple Architecture Diagram',
  'architecture-beta-x\nservice a',
  'architecture-betax',
  '  architecture-beta  \nservice a',
  '\n\n architecture-beta\nservice a',
  head + 'architecture-beta',
  '---\ntitle: From front matter\n---\narchitecture-beta\nservice a',
  '---\ntitle: From front matter\n---\narchitecture-beta\ntitle Own\nservice a',
  config(['iconSize: 96'], head + 'group g[G]\nservice a[A] in g\nservice b[B] in g\na:R -- L:b'),
  config(['padding: 60'], head + 'group g[G]\nservice a[A] in g\nservice b[B] in g\na:R -- L:b'),
  config(['fontSize: 24'], head + 'group g[G]\nservice a[A] in g\nservice b[B] in g\na:R -- L:b'),
  config(['iconSize: -5'], head + 'group g[G]\nservice a[A] in g\nservice b[B] in g\na:R -- L:b'),
  head + 'group g(cloud)[G]\nservice a(database)[A] in g\nservice b(logos:aws-lambda)[B] in g\nservice c "S3"[C]\nservice d',
  head + 'group one\ngroup two\nservice a in one\nservice b in two\na{group}:B --> T:b{group}',
  head + 'group one\ngroup two\njunction a in one\njunction b in two\na{group}:R --> L:b{group}',
  head + 'title Platform\nservice a[Alpha]\nservice b[Beta]\njunction j\na:R -[calls]- L:j\nj:R --> L:b',
  head +
    'group fe[Frontend]\ngroup be[Backend]\ngroup inner[Inner] in be\nservice web in fe\nservice gw in be\n' +
    'service auth in inner\nservice out\nservice mid\nweb:R -- L:mid\nmid:R -- L:gw\ngw:B -- T:auth\nauth:R -- L:out\nweb:B -- T:out',
];

// Strings in the sources of a folder of Pele's tests that hold the word architecture, as
// Pele's loadCorpus collects them, from files whose names match.
function testStrings(pele, dir, files) {
  const out = new Set();
  const path = join(pele, dir);
  for (const file of readdirSync(path).filter((f) => files.test(f))) {
    const src = readFileSync(join(path, file), 'utf8');
    for (const m of src.matchAll(/`((?:[^`\\]|\\.)*)`|'((?:[^'\\\n]|\\.)*)'|"((?:[^"\\\n]|\\.)*)"/g)) {
      const raw = m[1] ?? m[2] ?? m[3];
      if (!/architecture/.test(raw)) continue;
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
  const docs = JSON.parse(readFileSync(join(pele, 'tests/corpus/architecture-docs.json'), 'utf8'));
  return [
    ...new Set([
      ...docs,
      ...testStrings(pele, 'tests/compat/architecture', /\.(spec|test)\.[jt]s$/),
      ...testStrings(pele, 'tests/render', /^architecture.*\.test\.ts$/),
      ...SEEDS,
      ...WRITTEN,
    ]),
  ].filter((s) => !lone(s) && s.length <= 50000);
}

// Pele's generator in tests/architecture/oracle.test.ts: statements built from the grammar,
// most of them valid, then spoiled by a few token edits.
const IDS = ['a', 'b', 'db', 'x-y', 'n1', '_', '1', 'in1', 'rows', 'servicex', 'La', 'Tx', 'B', 'in', 'row', 'group', 'title', 'titles', 'accTitle', 'a-'];
const ICONS = ['(cloud)', '(a:b-c)', '(_)', '(server)', '(x-1)', '()', '(a b)'];
const TITLES = ['[T]', '["q"]', "['q']", '[ sp  aced ]', '[]', '["]', '[a]', '[\\"]', '["a\\"b"]', '[a"b]', '[é 采]', '["a\nb"]'];
const STRINGS = ['"s"', "'s'", '"a\\"b"', '""', '"two\nlines"'];
const SIDES = ['L', 'R', 'T', 'B'];
const LINES = ['title A title', 'title', 'accTitle: An acc title', 'accDescr: An acc descr', 'accDescr {\n many\n lines\n}', '%% comment', '%%{init: {}}%%'];
const WORDS = ['service', 'group', 'junction', 'align', 'row', 'column', 'in', ':', '--', '-', '---', '<', '>', '{group}', 'architecture-beta'];
const SEPARATORS = [' ', ' ', ' ', ' ', ' ', ' ', ' ', '', '  ', '\t'];
const BREAKS = ['\n', '\n', '\n', '\n', '\n', '\n', '\n', '\n', '\n', '\n', '\n', '\n\n', '\r\n', ' %% c\n', '\n  ', ' ', ''];

export function statements(rnd) {
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  const maybe = (p, ...tokens) => (rnd() < p ? tokens : []);
  const id = () => (rnd() < 0.93 ? pick(IDS.slice(0, 7)) : pick(IDS));
  const statement = () => {
    const r = rnd();
    if (r < 0.25) {
      const icon = rnd() < 0.5 ? maybe(0.7, pick(ICONS)) : maybe(0.5, pick(STRINGS));
      return ['service', id(), ...icon, ...maybe(0.6, pick(TITLES)), ...maybe(0.3, 'in', id())];
    }
    if (r < 0.4) return ['group', id(), ...maybe(0.5, pick(ICONS)), ...maybe(0.6, pick(TITLES)), ...maybe(0.3, 'in', id())];
    if (r < 0.5) return ['junction', id(), ...maybe(0.4, 'in', id())];
    if (r < 0.85) {
      const line = rnd() < 0.6 ? ['--'] : ['-', pick(TITLES), '-'];
      return [
        id(), ...maybe(0.2, '{group}'), ':', pick(SIDES), ...maybe(0.3, pick(['<', '>'])), ...line,
        ...maybe(0.4, pick(['<', '>'])), pick(SIDES), ':', id(), ...maybe(0.2, '{group}'),
      ];
    }
    if (r < 0.93) return ['align', pick(['row', 'column']), id(), id(), ...maybe(0.4, id()), ...maybe(0.2, id())];
    return [pick(LINES)];
  };
  const vocabulary = [...IDS, ...ICONS, ...TITLES, ...STRINGS, ...SIDES, ...LINES, ...WORDS];
  return () => {
    let src = rnd() < 0.92 ? 'architecture-beta' : pick(['\n architecture-beta ', 'architecture', '', 'architecture-beta-x']);
    src += pick(BREAKS);
    for (let count = Math.floor(rnd() * 4); count >= 0; count--) {
      const tokens = statement();
      if (rnd() < 0.3) {
        for (let edits = 1 + Math.floor(rnd() * 2); edits > 0; edits--) {
          const at = Math.floor(rnd() * (tokens.length + 1));
          const r = rnd();
          if (r < 0.3) tokens.splice(at, 1);
          else if (r < 0.55) tokens.splice(at, 0, pick(vocabulary));
          else if (r < 0.7) tokens.splice(at, 1, pick(vocabulary));
          else if (r < 0.85 && at < tokens.length) tokens.splice(at, 0, tokens[at]);
          else if (at + 1 < tokens.length) tokens.splice(at, 2, tokens[at + 1], tokens[at]);
        }
      }
      for (const token of tokens) src += token + pick(SEPARATORS);
      src += pick(BREAKS);
    }
    return src;
  };
}

const NAMES = ['database', 'server', 'disk', 'internet', 'cloud', 'logos:aws-lambda', 'mdi:home', 'unknown', 'star', 'fa:fa-car'];
const SERVICE_TITLES = [
  'API', 'Database', 'Storage', 'Server', 'Gateway', 'A', ' ', 'Load balancer', 'Authentication service', 'Message queue for events',
  'A very long title of a service that has to wrap over several lines', '中文服务', '😀 emoji', '#35; &amp;', '**bold** and _italic_',
  'fa:fa-car Car', 'a<br>b', 'Supercalifragilisticexpialidocious', 'x y z w v u t s r q p o n m l k j i h g f e d c b a',
];
const EDGE_TITLES = ['calls', 'reads', 'writes to', 'HTTP', 'a much longer label for an edge', 'x', '中文', '**b**', 'fa:fa-car go'];
const ICON_TEXTS = ['S3', 'DB', 'A much longer icon text that is cut short', 'x', '中文', 'two words', '😀😀😀😀😀😀😀😀', 'WWWWWWWWWW', 'a\\nb'];
const GROUP_TITLES = ['API', 'Backend', 'Frontend', 'Private network', 'A group with a long title that needs room', ' ', '中文', '**G**'];

// Random architecture diagrams, most of them well formed, over groups nested at random, and
// with random configuration.
export function randomDiagrams(rnd) {
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  const chance = (p) => rnd() < p;
  const n = (k) => Math.floor(rnd() * k);
  return () => {
    const settings = [];
    if (chance(0.3)) {
      if (chance(0.5)) settings.push(`iconSize: ${pick([8, 16, 24, 32, 40, 48, 64, 80, 96, 13.5, 7, 600, -5, '"x"'])}`);
      if (chance(0.4)) settings.push(`fontSize: ${pick([4, 10, 12, 13, 14, 15.5, 18, 20, 24, 2, 200, '.nan'])}`);
      if (chance(0.4)) settings.push(`padding: ${pick([0, 4, 8, 12, 20, 30, 40, 60, 2.5, 300, -1])}`);
    }
    let s = '';
    if (chance(0.1)) s += `---\ntitle: ${pick(['T', 'Front matter title', '中文'])}\n---\n`;
    if (settings.length) {
      if (chance(0.7)) s += `---\nconfig:\n  architecture:\n${settings.map((l) => '    ' + l).join('\n')}\n---\n`;
      else s += `%%{init: {"architecture": {${settings.map((l) => l.replace(/^(\w+): (.*)$/, (_, k, v) => `"${k}": ${/^[-\d.]+$|^"/.test(v) ? v : `"${v}"`}`)).join(', ')}}}}%%\n`;
    }
    s += 'architecture-beta\n';
    if (chance(0.2)) s += `title ${pick(['Platform', 'A title that is much wider than the diagram under it, by a long way', '中文'])}\n`;
    if (chance(0.05)) s += 'accTitle: Accessible\naccDescr: Described\n';
    const groups = chance(0.3) ? 0 : n(chance(0.2) ? 8 : 4);
    const parent = [];
    for (let g = 0; g < groups; g++) {
      parent.push(g > 0 && chance(0.4) ? n(g) : -1);
      const icon = chance(0.4) ? `(${pick(NAMES)})` : '';
      const title = chance(0.7) ? `[${pick(GROUP_TITLES)}]` : '';
      s += `group g${g}${icon}${title}${parent[g] >= 0 ? ` in g${parent[g]}` : ''}\n`;
    }
    const count = 1 + n(chance(0.2) ? 24 : 9);
    const where = [];
    for (let i = 0; i < count; i++) {
      where.push(groups > 0 && chance(0.6) ? n(groups) : -1);
      const inside = where[i] >= 0 ? ` in g${where[i]}` : '';
      if (chance(0.15)) {
        s += `junction n${i}${inside}\n`;
        continue;
      }
      const icon = chance(0.6) ? `(${pick(NAMES)})` : chance(0.2) ? ` "${pick(ICON_TEXTS)}"` : '';
      const title = chance(0.7) ? `[${pick(SERVICE_TITLES)}]` : '';
      s += `service n${i}${icon}${title}${inside}\n`;
    }
    for (let k = n(count * 2); k > 0; k--) {
      const a = n(count);
      const b = n(count);
      const crossing = where[a] !== where[b] || where[a] < 0;
      const ga = crossing && where[a] >= 0 && chance(0.3) ? '{group}' : '';
      const gb = crossing && where[b] >= 0 && chance(0.3) ? '{group}' : '';
      const line = chance(0.25) ? `-[${pick(EDGE_TITLES)}]-` : '--';
      s += `n${a}${ga}:${pick(SIDES)} ${chance(0.2) ? '<' : ''}${line}${chance(0.4) ? '>' : ''} ${pick(SIDES)}:n${b}${gb}\n`;
    }
    if (count > 2 && chance(0.25)) {
      const members = [...new Set([n(count), n(count), n(count), n(count)])];
      if (members.length >= 2) s += `align ${pick(['row', 'column'])} ${members.map((m) => 'n' + m).join(' ')}\n`;
    }
    return s;
  };
}

export default async function ({ src, pele, write, random }) {
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { architecture } = await src('diagrams/architecture/index.ts');
  const { parseArchitecture, ARCHITECTURE_TOKENS } = await src('diagrams/architecture/parser.ts');
  const { tokenize } = await src('diagrams/common/tokens.ts');

  const caught = (e) => {
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  const o = (x) => (x === undefined || x === '' ? null : x);
  const tokens = (source) => {
    try {
      const t = tokenize(source, ARCHITECTURE_TOKENS, 'architecture');
      return { tokens: t.kinds.slice(0, -1).map((k, i) => [ARCHITECTURE_TOKENS[k].name, source.slice(t.starts[i], t.ends[i])]) };
    } catch {
      return { tokens: null };
    }
  };
  const edge = (e) => [e.lhsId, !!e.lhsGroup, e.lhsDir, !!e.lhsInto, o(e.title), !!e.rhsInto, e.rhsDir, e.rhsId, !!e.rhsGroup];
  const direct = (source) => {
    try {
      const ast = parseArchitecture(source);
      return {
        ...tokens(source),
        ast: {
          title: o(ast.title),
          accTitle: o(ast.accTitle),
          accDescr: o(ast.accDescr),
          groups: ast.groups.map((g) => [g.id, o(g.icon), o(g.title), o(g.in)]),
          services: ast.services.map((s) => [s.id, o(s.icon), o(s.iconText), o(s.title), o(s.in)]),
          junctions: ast.junctions.map((j) => [j.id, o(j.in)]),
          edges: ast.edges.map(edge),
          alignments: ast.alignments.map((a) => [a.direction, a.members]),
        },
      };
    } catch (e) {
      return { ...tokens(source), ...caught(e) };
    }
  };
  // Pele's parseSource for an architecture diagram, and the raw text given to the parser alone.
  const run = (source) => {
    const out = { src: source, direct: direct(source) };
    let pre;
    try {
      pre = preprocess(source);
      out.type = detect(pre.text);
    } catch (e) {
      return { ...out, ...caught(e) };
    }
    if (out.type !== 'architecture') return out;
    try {
      const m = architecture.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title);
      out.model = {
        title: o(m.title),
        accTitle: o(m.accTitle),
        accDescr: o(m.accDescr),
        nodes: [...m.nodes.values()].map((n) =>
          n.type === 'service' ? [n.id, 'service', o(n.icon), o(n.iconText), o(n.title), o(n.in)] : [n.id, 'junction', o(n.in)]
        ),
        groups: [...m.groups.values()].map((g) => [g.id, o(g.icon), o(g.title), o(g.in)]),
        edges: m.edges.map(edge),
        hints: m.layoutHints.map((h) => [h.direction, h.members]),
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
  const built = statements(random(8));
  for (let i = 0; i < 2000; i++) add(built());
  const diagrams = randomDiagrams(random(21));
  for (let i = 0; i < 800; i++) add(diagrams());
  // Mutated text; one in six is cut short, as a statement cut off by the end of the text ends
  // on a different error than one cut off by a line break.
  const rnd = random(7);
  const next = mutator([...seen].filter((s) => s.length <= 4000), FRAGMENTS, rnd);
  for (let i = 0; i < 3000; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  // Random strings of tokens, where the order of the token types and their ends matter.
  const trnd = random(9);
  const pieces = [...new Set([...FRAGMENTS, ...FRAGMENTS.map((f) => f.trim()).filter((f) => f !== '')])];
  const openers = [head, 'architecture-beta ', head + 'service a', head + 'a:', head + 'a:R ', head + 'group g', head + 'align row ', ''];
  for (let i = 0; i < 1000; i++) {
    let s = openers[Math.floor(trnd() * openers.length)];
    for (let k = 1 + Math.floor(trnd() * 8); k > 0; k--) s += pieces[Math.floor(trnd() * pieces.length)] + (trnd() < 0.3 ? ' ' : '');
    add(s);
  }
  // Text that runs on, where a token's pattern could rescan it.
  for (const s of [
    head + 'service a[' + 'x'.repeat(3000),
    head + 'service a["' + 'x'.repeat(3000),
    head + 'service a["' + '\\'.repeat(3000) + ']',
    head + 'service a(' + 'x'.repeat(3000),
    head + 'a' + '-'.repeat(3000),
    head + '---\n'.repeat(800),
    head + '%%{\n'.repeat(800),
    head + 'accDescr {' + ' a\n'.repeat(1000),
    head + ' '.repeat(3000) + 'service a',
  ]) {
    seen.delete(s);
    add(s);
  }
  write('diagram/architecture/testdata/pele/architecture.json', { cases: all.map(run) });
}
