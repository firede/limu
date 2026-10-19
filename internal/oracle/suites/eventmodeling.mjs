// Event modeling diagrams parsed as Pele parses them: the tokens Pele's common/tokens.ts makes
// of the text with the diagram's table, the tree parseEventModel makes of them, and the model
// buildEventModel makes of the tree. The cases are Mermaid's documentation and specs, the
// inputs of Pele's own tests, cases written here, random diagrams, and all of those mutated
// at random.
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { enc, lone, mutator, peleError } from './parse.mjs';

// The fragments of Pele's tests/eventmodeling/oracle.test.ts, and more.
export const FRAGMENTS = [
  'eventmodeling', 'eventmodeling\n', 'tf', 'timeframe', 'rf', 'resetframe', 'tf 01 evt A', '\ntf 09 cmd B.C', 'rf 7 ui U', 'entity', 'entity A',
  'data', 'data D {\n x\n}\n', 'note', 'note 01 {\n n\n}', 'gwt', 'gwt 01 given evt A then evt B', 'given', 'when', 'then', 'rmo', 'readmodel',
  'ui', 'cmd', 'command', 'evt', 'event', 'pcr', 'processor', 'json', 'jsobj', 'figma', 'salt', 'uri', 'md', 'html', 'text', '`json`', '`md`',
  '`', '.', '->>', ' ->> 01', '->', '>>', '[[', ']]', '[[D]]', '{', '}', '{ a: 1 }', '{}', '{\n', '\n}', '\n}\n', '"', "'", '"q"', "'s'", '0', '01', '123',
  '1234', ' title T', ' accTitle: T', ' accDescr: D', ' accDescr {', 'title', 'accTitle', '\n', '\r\n', '\r', ' ', '\t', ' ', '%%', '%% c',
  '%%{init: {}}%%', '---', '---\n', '/*', '*/', '/* c */', '//', '// c', 'x', 'A', '_a', 'é', ':', ';', ',', '#35;', 'tfx', 'eventx', 'evt2',
  ' ', '　', '😀', '中文', '&amp;', '<b>b</b>', ' ->> 02', ' ->> 1', '[[Payload]]', '`text`', 'Cart.', '.Item', 'a.b.c',
  '\ntf 02 rmo View', '\nrf 03 pcr Auto', '\ntf 04 ui Screen', '\ndata Payload {\n  a: 1\n}\n', '\nnote 02 `md` {\n  line\n}\n',
  '\ngwt 02\n given\n  evt A\n when\n  cmd B\n then\n  evt C\n', '{ "a": { "b": 1 } }', "'it is'", '{\t\n', '}x', '} ', '}\r\n',
];

const DATA = 'data Payload {\n  description: string\n    nested: true\n}\n';

// Cases written here: what the tokens and the model do at their edges.
export const WRITTEN = [
  'eventmodeling\ntf 01 ui CartUI\ntf 02 cmd AddItem\ntf 03 evt ItemAdded\n',
  'eventmodeling\ntf 01 ui A\ntf 02 cmd B\ntf 03 evt C\ntf 04 rmo D\ntf 05 pcr E\n',
  'eventmodeling\ntf 01 cmd A\ntf 02 evt B\ntf 03 cmd C\ntf 04 cmd D\ntf 05 evt E\n',
  'eventmodeling\ntf 01 evt A\ntf 02 evt B\ntf 03 rmo C\ntf 04 rmo D\n',
  'eventmodeling\ntf 01 ui A\ntf 02 cmd B\nrf 03 evt C\ntf 04 pcr D\n',
  'eventmodeling\ntf 01 evt A\ntf 02 ui B\nrf 03 rmo C ->> 01\n',
  'eventmodeling\nrf 02 evt A\nrf 03 evt B\ntf 01 rmo C ->> 03 ->> 02 ->> 09\n',
  'eventmodeling\ntf 01 evt A ->> 02\ntf 02 cmd B\n',
  'eventmodeling\ntf 01 evt A ->> 01\ntf 02 cmd B ->> 02 ->> 01\n',
  'eventmodeling\ntf 01 cmd Cart.Add\ntf 02 evt Cart.Added\ntf 03 cmd Cart.Remove\ntf 04 evt Other.Seen\ntf 05 evt Plain\ntf 06 evt a.b.c\n',
  'eventmodeling\ntf 01 ui N.A\ntf 02 pcr M.B\ntf 03 ui N.C\ntf 04 rmo N.D\ntf 05 evt N.E\ntf 06 ui A\n',
  'eventmodeling\ntf 01 cmd A { productId: 7 }\ntf 02 evt B [[Payload]] { ignored: true }\ntf 03 evt C "{ "a": 1 }"\ntf 04 evt D [[Missing]]\ntf 05 evt E {x}\n' + DATA,
  'eventmodeling\n%% a real comment\ntf 01 evt A [[D]]\ndata D {\n  %% kept\n  a: 1\n}\n',
  'eventmodeling\ntf 01 cmd Update\ntf 02 evt Updated\nentity Updated\nnote 02 `md` {\n  first line\n  second line\n}\n' +
    'gwt 01\n given\n  evt Updated\n  evt Other\n when\n  cmd Update\n then\n  evt Updated\ngwt 02 given evt A then evt B\n',
  'eventmodeling title Checkout\ntf 01 ui A accTitle: Acc\ntf 02 cmd B accDescr: Described\n',
  '---\ntitle: Front\n---\neventmodeling\ntf 01 ui A\n',
  '---\ntitle: Front\n---\neventmodeling title Own\ntf 01 ui A\n',
  '---\ntitle: Front\n---\neventmodeling title\ntf 01 ui A\n',
  '---\nconfig:\n  eventmodeling:\n    padding: 40\n  fontSize: 30\n---\neventmodeling\ntf 01 ui A\n',
  '%%{init: {"eventmodeling": {"x": 1}, "theme": "dark"}}%%\neventmodeling\ntf 01 ui A\n',
  'eventmodeling\n  tf 01 ui UI\n  %% a comment line\n  rf 01 evt Event\n',
  'eventmodeling\ntf 1 ui A\ntf 01 ui B\ntf 001 ui C\ntf 1 ui D\n',
  'eventmodeling\ntitle T\n',
  'eventmodeling\ntf 01 evt data\n',
  'eventmodeling\ntf 01 evt A {\n',
  'eventmodeling\ntf 1234 evt A\n',
  'eventmodeling\ntf 01 A\n',
  'eventmodeling\ntf 01 evt A {}\ntf 02 evt B { }\ntf 03 evt C ""\ntf 04 evt D \'\'\ntf 05 evt E "a{b"\ntf 06 evt F "}{"\ntf 07 evt G "{x}"\n',
  'eventmodeling\ntf 01 evt A { a } b }\ntf 02 evt B { a } [[x]] { b }\ntf 03 evt C {{{}}}\n',
  'eventmodeling\ntf 01 evt A [[D]]\ndata D {\n\n   a\n\t\tb\n    \n  c   \n\n}\n',
  'eventmodeling\ntf 01 evt A [[D]]\ndata D {\r\n  a\r\n   b\r\n}\r\n',
  'eventmodeling\ntf 01 evt A [[D]]\ndata D {\n　a\n　　b\n}\n',
  'eventmodeling\ntf 01 evt A [[D]]\ndata D {\n  a\n}\ndata D {\n  b\n}\n',
  'eventmodeling\ntf 01 evt A [[D]]\ndata D `json` {\n  { "a": 1 }\n}x\n}\n',
  'eventmodeling\ntf 01 evt A [[D]]\ndata D {\n}\n',
  'eventmodeling\ntf 01 evt A [[D]]\ndata D {  \n}',
  'eventmodeling\ntf 01 evt A [[D]]\ndata D {\n a\n}%% c\n',
  'eventmodeling\ndata D {\n a\n}tf 01 evt A\n',
  'eventmodeling\nnote 01 {\n}\nnote 02 {\n  \n}\n',
  'eventmodeling\ntf 01 evt A\nnote 01 `html` {\n <b>x</b> &amp; #35;\n}',
  'eventmodeling\ntf 01 evt A { ' + 'x'.repeat(60) + ' }\n',
  'eventmodeling\ntf 01 evt A { ' + '😀'.repeat(30) + ' }\n',
  'eventmodeling\ntf 01 evt A { 中文中文 }\ntf 02 cmd B "a b"\n',
  'eventmodeling\ntf 01 evt A\ntf 01 evt B\n',
  'eventmodeling\ntf 01 evt A\n/* tf 01 evt B */\ntf 02 evt C // tf 02 evt D\n',
  'eventmodeling\ntf 01 evt A /* unclosed\ntf 02 evt C\n',
  'eventmodeling accDescr {\n multi\n   line\n}\ntf 1 event E "inline"\ntf 2 command C \'single\'\ntf 3 readmodel R `text`"t"\n',
  'eventmodeling\ntf 01 ui A accDescr {\n unclosed\ntf 02 ui B accDescr: d\n',
  'eventmodeling\ntf 01 ui A accTitle: one\ntf 02 ui B accTitle: two\n',
  'eventmodeling\ntf 01 ui A accTitle:\n',
  'eventmodeling\ntf 01 ui A title x\n',
  'eventmodeling\n title x\n',
  'eventmodeling\ttitle x\n',
  'eventmodeling title   a   b  %% c\n',
  'eventmodeling\ngwt 01 given evt A when then evt B\n',
  'eventmodeling\ngwt 01 given then evt A\n',
  'eventmodeling\ngwt 01 given evt A.B then evt C\n',
  'eventmodeling\ngwt 01 given evt A cmd B ui C rmo D pcr E then evt F\n',
  'eventmodeling\ngwt 01 given evt A\n',
  'eventmodeling\nentity A.B.C\nentity D\nentity\n',
  'eventmodeling\ntf 01 evt A ->>\n',
  'eventmodeling\ntf 01 evt A ->> 02 [[D\n',
  'eventmodeling\ntf 01 evt A `json`\n',
  'eventmodeling\ntf 01 evt A `xml` {}\n',
  'eventmodeling\ntf 01 evt A `json {}\n',
  'eventmodeling\ntf 01 evt A.\n',
  'eventmodeling\ntf 01 evt .A\n',
  'eventmodeling\ntf 01 evt __proto__.constructor [[toString]]\ntf 02 evt hasOwnProperty\ndata toString {\n x\n}\n',
  'eventmodeling\ntf 999 evt A\ntf 000 evt B\ntf 12 evt C\n',
  'eventmodeling\ntf 0101 evt A\n',
  'eventmodeling\ntfevt\n',
  'eventmodeling\ntf01 evt A\n',
  'eventmodeling\ntf 01evt A\n',
  'eventmodeling\ntf 01 evtA\n',
  'eventmodeling\ntf 01 evt A{x}\n',
  'eventmodeling\r\ntimeframe 01 ui A\r\nresetframe 02 processor P ->> 01\r\ndata D {\r\n x\r\n}\r\n',
  'eventmodeling\rtf 01 ui A\rtf 02 cmd B',
  'eventmodeling tf 01 ui A tf 02 cmd B tf 03 evt C　',
  'eventmodeling\n tf 01 ui A\n',
  'eventmodeling%%c\ntf 01 ui tfx\ntf 02 ui eventx\n',
  'eventmodeling %%{init: {}}%%\ntf 01 ui A\n',
  'eventmodeling\n---\nx\n---\ntf 01 ui A\n',
  'eventmodelingx\n',
  'eventmodeling-beta\n',
  'eventmodeling:\n',
  'EventModeling\n',
  ' eventmodeling\n',
  'eventmodeling',
  'eventmodeling\n',
  'eventmodeling ',
  '',
  'not eventmodeling',
];

// Strings in the sources of a folder of Pele's tests that hold the keyword, as Pele's
// loadCorpus collects them, from files whose names match.
function testStrings(pele, dir, files) {
  const out = new Set();
  const path = join(pele, dir);
  for (const file of readdirSync(path).filter((f) => files.test(f))) {
    const src = readFileSync(join(path, file), 'utf8');
    for (const m of src.matchAll(/`((?:[^`\\]|\\.)*)`|'((?:[^'\\\n]|\\.)*)'|"((?:[^"\\\n]|\\.)*)"/g)) {
      const raw = m[1] ?? m[2] ?? m[3];
      if (!/eventmodeling/.test(raw)) continue;
      try {
        out.add(new Function('return `' + raw.replace(/`/g, '\\`').replace(/\$\{/g, '\\${') + '`')());
      } catch {
        continue;
      }
    }
  }
  return [...out];
}

// The corpus the suites start from: documentation, specs, the strings of Pele's tests and the
// cases above.
export function corpus(pele) {
  const docs = JSON.parse(readFileSync(join(pele, 'tests/corpus/eventmodeling-docs.json'), 'utf8'));
  return [
    ...new Set([
      ...docs,
      ...testStrings(pele, 'tests/compat/eventmodeling', /\.(spec|test)\.[jt]s$/),
      ...testStrings(pele, 'tests/eventmodeling', /\.test\.ts$/),
      ...testStrings(pele, 'tests/render', /^eventmodeling.*\.test\.ts$/),
      ...WRITTEN,
    ]),
  ].filter((s) => !lone(s) && s.length <= 50000);
}

const NAMES = ['CartUI', 'AddItem', 'ItemAdded', 'Cart', 'Inventory', 'A', 'B', 'x', '_a1', 'VeryLongEntityNameThatIsWiderThanTheBox', 'Ab_9'];
const NAMESPACES = ['Cart', 'Order', 'External', 'N', 'ns_1'];
const KINDS = ['ui', 'cmd', 'evt', 'rmo', 'pcr', 'command', 'event', 'readmodel', 'processor'];
const TYPES = ['json', 'jsobj', 'figma', 'salt', 'uri', 'md', 'html', 'text'];
const VALUES = [
  'description: string', 'productId: 7', 'a: 1, b: 2', '"a": { "b": [1, 2, 3] }', 'x', '', ' ', '中文 text', '😀 emoji',
  'a long line of example data that runs past the forty-eight columns a frame shows', '#35; &amp; <b>b</b>', 'tab\there',
];
const BLOCKS = [
  '  description: string\n  image: string\n  price: number',
  '  a: 1',
  '    nested\n  less\n      more',
  '  %% kept\n  a: 1',
  '',
  '  ' + 'y'.repeat(100),
  '  first\n\n  third',
  '  { "a": {\n      "b": true\n  }}',
  '\t<div>html</div>',
  '  中文中文中文中文中文中文中文中文中文中文中文中文中文中文中文中文中文中文中文中文中文中文中文中文中文中文',
  '  ' + 'ab😀'.repeat(30),
];
const TITLES = ['Checkout', 'A title of an event model that is a good deal wider than the lanes', '中文', '#35; &amp;', 'a <b>b</b>'];

// Random event modeling diagrams, most of them well formed.
export function randomDiagrams(rnd) {
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  const chance = (p) => rnd() < p;
  const id = (n) => (chance(0.8) ? String(n).padStart(2, '0') : String(n));
  return () => {
    let s = '';
    if (chance(0.1)) s += `---\ntitle: ${pick(TITLES)}\n---\n`;
    s += 'eventmodeling';
    if (chance(0.15)) s += ` title ${pick(TITLES)}`;
    s += '\n';
    if (chance(0.1)) s += chance(0.5) ? ' accTitle: Accessible\n' : ' accDescr {\n  one\n   two\n}\n';
    const frames = chance(0.1) ? 0 : 1 + Math.floor(rnd() * (chance(0.2) ? 18 : 7));
    const refs = [];
    const blocks = [];
    for (let k = 1; k <= frames; k++) {
      const reset = chance(0.12);
      let line = `${reset ? pick(['rf', 'resetframe']) : pick(['tf', 'tf', 'timeframe'])} ${id(chance(0.02) ? k - 1 : k)} ${pick(KINDS)} `;
      line += chance(0.2) ? `${pick(NAMESPACES)}.${pick(NAMES)}` : chance(0.03) ? `${pick(NAMES)}.${pick(NAMES)}.${pick(NAMES)}` : pick(NAMES);
      if (chance(0.15) && k > 1) for (let j = 1 + Math.floor(rnd() * 3); j > 0; j--) line += ` ->> ${id(1 + Math.floor(rnd() * (k + 1)))}`;
      if (chance(0.2)) {
        const ref = `D${Math.floor(rnd() * 4)}`;
        line += ` [[${ref}]]`;
        refs.push(ref);
      }
      if (chance(0.3)) {
        if (chance(0.3)) line += ` \`${pick(TYPES)}\``;
        const v = pick(VALUES);
        line += chance(0.7) ? ` { ${v} }` : chance(0.5) ? ` "${v}"` : ` '${v}'`;
      }
      s += (chance(0.2) ? '  ' : '') + line + (chance(0.05) ? '\r\n' : '\n');
    }
    for (const ref of refs) {
      if (chance(0.8) && !blocks.includes(ref)) {
        blocks.push(ref);
        s += `data ${ref}${chance(0.2) ? ` \`${pick(TYPES)}\`` : ''} {\n${pick(BLOCKS)}\n}\n`;
      }
    }
    if (chance(0.3)) s += `entity ${pick(NAMES)}\n`;
    for (let j = chance(0.3) ? 1 + Math.floor(rnd() * 3) : 0; j > 0; j--) {
      s += `note ${id(1 + Math.floor(rnd() * (frames + 1)))}${chance(0.2) ? ` \`${pick(TYPES)}\`` : ''} {\n${pick(BLOCKS)}\n}\n`;
    }
    for (let j = chance(0.3) ? 1 + Math.floor(rnd() * 2) : 0; j > 0; j--) {
      const statements = (n) => Array.from({ length: n }, () => `${pick(KINDS)} ${pick(NAMES)}`).join(chance(0.5) ? ' ' : '\n  ');
      s += `gwt ${id(1 + Math.floor(rnd() * (frames + 1)))} given ${statements(1 + Math.floor(rnd() * 3))}`;
      if (chance(0.5)) s += ` when ${statements(1 + Math.floor(rnd() * 2))}`;
      s += ` then ${statements(1 + Math.floor(rnd() * 3))}\n`;
    }
    if (chance(0.1)) s += '%% a comment\n';
    return s;
  };
}

// Pele's stress cases, cut down to what a fixture holds.
export const STRESS = [
  'eventmodeling\n' + 'tf 01 evt A {\n'.repeat(200),
  'eventmodeling\ndata D {\n' + '}x\n'.repeat(500),
  'eventmodeling\n' + ('{'.repeat(99) + '\n').repeat(10),
  'eventmodeling\ntf 01 evt A ' + '"'.repeat(1000),
  'eventmodeling\n' + "tf 01 evt A '\n".repeat(100),
  'eventmodeling\n/*' + ' x\n'.repeat(300),
  'eventmodeling\n' + '/* '.repeat(300),
  'eventmodeling\n' + 'tf 01 evt A accDescr {\n'.repeat(100),
  'eventmodeling\n' + ('x' + ' '.repeat(90) + 'accDescr ').repeat(10),
  'eventmodeling\n' + 'a title b\n'.repeat(100),
  'eventmodeling\n' + 'x %%{\n'.repeat(100),
  'eventmodeling\n' + 'x\n---\n'.repeat(100),
  'eventmodeling\n' + 'tf ' + '1'.repeat(1000),
  'eventmodeling\n' + 'tfevtuidatanote'.repeat(100),
  'eventmodeling\ntf 01 evt ' + 'a.'.repeat(500) + 'z\n',
  'eventmodeling\ntf 001 cmd A\ntf 002 evt B' + ' ->> 001'.repeat(300) + '\n',
];

export default async function ({ src, pele, write, random }) {
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { eventmodeling } = await src('diagrams/eventmodeling/index.ts');
  const { parseEventModel, EVENTMODELING_TOKENS } = await src('diagrams/eventmodeling/parser.ts');
  const { tokenize } = await src('diagrams/common/tokens.ts');

  const caught = (e) => {
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  const opt = (x) => (x === undefined || x === '' ? undefined : x);
  const tokens = (source) => {
    try {
      const t = tokenize(source, EVENTMODELING_TOKENS, 'eventmodeling');
      return { tokens: t.kinds.slice(0, -1).map((k, i) => [EVENTMODELING_TOKENS[k].name, source.slice(t.starts[i], t.ends[i])]) };
    } catch (e) {
      return { tokens: null };
    }
  };
  const refs = (list) => list.map((r) => r.$refText ?? r.entityIdentifier.$refText);
  const direct = (source) => {
    try {
      const ast = parseEventModel(source);
      return {
        ...tokens(source),
        ast: {
          title: enc(ast.title),
          accTitle: enc(ast.accTitle),
          accDescr: enc(ast.accDescr),
          entities: ast.modelEntities.map((e) => e.name),
          frames: ast.frames.map((f) => [
            f.$type,
            f.name,
            f.modelEntityType,
            f.entityIdentifier,
            refs(f.sourceFrames),
            enc(f.dataReference?.$refText),
            enc(f.dataType),
            enc(f.dataInlineValue),
          ]),
          data: ast.dataEntities.map((d) => [d.name, enc(d.dataType), d.dataBlockValue]),
          notes: ast.noteEntities.map((n) => [n.sourceFrame.$refText, enc(n.dataType), n.dataBlockValue]),
          gwts: ast.gwtEntities.map((g) => [g.sourceFrame.$refText, refs(g.givenStatements), refs(g.whenStatements), refs(g.thenStatements)]),
        },
      };
    } catch (e) {
      return { ...tokens(source), ...caught(e) };
    }
  };
  // Pele's parseSource for an event modeling diagram, and the raw text given to the parser
  // alone.
  const run = (source) => {
    const out = { src: source, direct: direct(source) };
    let pre;
    try {
      pre = preprocess(source);
      out.type = detect(pre.text);
    } catch (e) {
      return { ...out, ...caught(e) };
    }
    if (out.type !== 'eventmodeling') return out;
    try {
      const m = eventmodeling.parse(encodeEntities(pre.withComments) + '\n', pre.config, pre.title);
      out.model = {
        title: opt(m.title),
        accTitle: opt(m.accTitle),
        accDescr: opt(m.accDescr),
        lanes: m.lanes.map((l) => [l.index, l.label, l.group, enc(l.namespace)]),
        boxes: m.boxes.map((b) => [b.id, b.kind, b.entity, b.name, enc(b.namespace), b.reset, m.lanes.indexOf(b.lane), enc(b.data), enc(b.dataType)]),
        relations: m.relations.map((r) => [r.source.id, r.target.id]),
        notes: m.notes.map((n) => [n.frame, n.text, enc(n.dataType)]),
        specifications: m.specifications.map((s) => [s.frame, s.given, s.when, s.then]),
        entities: m.entities,
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
  const diagrams = randomDiagrams(random(51));
  for (let i = 0; i < 1500; i++) add(diagrams());
  // Mutated text; one in six is cut short, as a statement cut off by the end of the text ends
  // on a different error than one cut off by a line break.
  const rnd = random(52);
  const next = mutator([...seen].filter((s) => s.length <= 4000), FRAGMENTS, rnd);
  for (let i = 0; i < 5500; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  // Random strings of tokens, where the order of the token types and their ends matter.
  const trnd = random(53);
  const pieces = [...new Set([...FRAGMENTS, ...FRAGMENTS.map((f) => f.trim()).filter((f) => f !== '')])];
  const openers = ['eventmodeling\n', 'eventmodeling ', 'eventmodeling\ntf 01 ', 'eventmodeling\ntf 01 evt A', 'eventmodeling\ndata D', 'eventmodeling\ngwt 01 ', ''];
  for (let i = 0; i < 1000; i++) {
    let s = openers[Math.floor(trnd() * openers.length)];
    for (let k = 1 + Math.floor(trnd() * 8); k > 0; k--) s += pieces[Math.floor(trnd() * pieces.length)] + (trnd() < 0.3 ? ' ' : '');
    add(s);
  }
  for (const s of STRESS) {
    seen.delete(s);
    add(s);
  }
  write('diagram/eventmodeling/testdata/pele/eventmodeling.json', { cases: all.map(run) });
}
