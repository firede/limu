// Venn diagrams parsed as Pele parses them, from text to tokens and to VennModel: Mermaid's
// documentation and specs, the inputs of Pele's own tests, cases written here, those mutated at
// random, diagrams built from random statements, and random strings of tokens. Each case is
// lexed with the lexer's question about indentation answered no and answered yes, parsed as the
// raw text, and parsed as Pele's parse does it, after the preprocessing every diagram goes
// through.
import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import { enc, lone, mutator, peleError } from './parse.mjs';

// An array literal of a test's source, evaluated with the names given.
export function arrayIn(file, start, names = {}, close = '\n];') {
  const src = readFileSync(file, 'utf8');
  const from = src.indexOf(start);
  if (from < 0) throw new Error(`no ${start} in ${file}`);
  const open = from + start.length;
  const end = src.indexOf(close, open);
  return new Function(...Object.keys(names), 'return [' + src.slice(open, end) + '\n];')(...Object.values(names));
}

// Every string literal in a file of Pele's tests that keyword matches, as its loadCorpus finds
// them.
function literals(file, keyword) {
  const out = [];
  const src = readFileSync(file, 'utf8');
  for (const m of src.matchAll(/`((?:[^`\\]|\\.)*)`|'((?:[^'\\\n]|\\.)*)'|"((?:[^"\\\n]|\\.)*)"/g)) {
    const raw = m[1] ?? m[2] ?? m[3];
    if (!keyword.test(raw)) continue;
    try {
      out.push(new Function('return `' + raw.replace(/`/g, '\\`').replace(/\$\{/g, '\\${') + '`')());
    } catch {
      continue;
    }
  }
  return out;
}

// Pele's fragments and seeds, tests/venn/oracle.test.ts, and fragments for the pipeline and
// for what Pele draws.
export function fragments(pele) {
  const own = [
    '---\n', '---\ntitle: Front\n---\n', 'title: x\n', '%%{init: {"venn": {"padding": 3}}}%%\n', '#35;', '#quot;', '&amp;', '<br>',
    '<b>b</b>', '<i>i</i>', 'fa:fa-car', '😀', '中文', 'A very long label that wraps', ':0', ':1e3', ':999999999999', ':0.0001',
    'font-weight:bold', 'font-style:italic', 'fill-opacity:0.5', 'stroke-dasharray:4 2', 'stroke-linecap:round', 'opacity:.5',
  ];
  return [...arrayIn(join(pele, 'tests/venn/oracle.test.ts'), 'const FRAGMENTS = ['), ...own];
}

const configured = (body, diagram) => `---\nconfig:\n${body}\n---\n${diagram}`;

// Cases written here, for the corners of the grammar and of the drawing.
export const WRITTEN = [
  '',
  'venn-beta',
  'venn-beta\n',
  'venn-beta set A',
  'venn-beta\nvenn-beta\nset A',
  'venn-beta\nset',
  'venn-beta\nset A,B',
  'venn-beta\nset A $',
  'venn-beta\nunion A',
  'venn-beta\nset A\nunion A,B',
  'venn-beta\nset A\nunion A,B,C,D',
  'venn-beta\nset A\nset B\nunion A,B\nunion B,A["again"]:4',
  'venn-beta\nset A\nset B\nunion A, B',
  'venn-beta\nset A\nset B\nunion A ,B',
  'venn-beta\n  text x',
  'venn-beta\ntext A x',
  'venn-beta\nset A\n  text',
  'venn-beta\nset A\n  text 5["l"]',
  'venn-beta\nset A\n  text "q"[l]',
  'venn-beta\nset A\n  text x["  "]',
  'venn-beta\nset A[""]',
  'venn-beta\nset A["  "]',
  'venn-beta\nset A[ ]',
  'venn-beta\nset A[]',
  'venn-beta\nset A:',
  'venn-beta\nset A:-0',
  'venn-beta\nset A:+.5',
  'venn-beta\nset A:1.5.2',
  `venn-beta\nset A:${'9'.repeat(400)}\nset B`,
  `venn-beta\nset A:.${'0'.repeat(330)}1\nset B`,
  'venn-beta\nset A\nstyle A',
  'venn-beta\nset A\nstyle A fill',
  'venn-beta\nset A\nstyle A fill:',
  'venn-beta\nset A\nstyle A fill:red,',
  'venn-beta\nset A\nstyle A fill:red, fill:blue, FILL:green',
  'venn-beta\nset A\nstyle A fill:""',
  'venn-beta\nset A\nstyle A fill:"  red  "',
  'venn-beta\nset A\nstyle A fill:rgb(1, 2, 3) rgba(1,2,3,.5) #abc 5 x',
  'venn-beta\nset A\nstyle A fill:#ab',
  'venn-beta\ntitle',
  'venn-beta\ntitle ',
  'venn-beta\ntitle  two spaces',
  'venn-beta\ntitle\tA tab',
  'venn-beta\ntitle　Ideographic',
  'venn-beta\ntitle No-break',
  'venn-beta\ntitle # hash',
  'venn-beta\ntitle a#b\nset A',
  'venn-beta\ntitle a;b\nset A',
  'venn-beta\ntitle "  quoted  "\nset A',
  'venn-beta\nTitle Case\nset A',
  'venn-beta\nset A\n\ttext t\n \ttext u\n\n   text v\n%% c\n  text w',
  'venn-beta\nset A\n  set B\n  text b',
  'venn-beta\nset A\nunion A,A\n  text aa',
  'venn-beta\nset A\nset B\nunion A,B text t',
  'venn-beta\nset A text t',
  'venn-beta\nset A\n  text t["l"] text u',
  'venn-beta\nset é\nset 中文\nset "😀"\nunion é,中文,"😀"',
  'venn-beta\nset "￿"\nset "😀"\nunion "￿","😀"',
  'venn-beta\nset A\n😀',
  'venn-beta\nset A 😀',
  'venn-beta\nset A\né%%x\nset B',
  'venn-beta\nset A\n😀%%x\nset B',
  'venn-beta\nset A%%{x\nset B',
  'venn-beta\nset A}%%x',
  'venn-beta\r\nset A\r\n  text x\r\nset B\r\n',
  'venn-beta\rset A\r  text x',
  '\n\n  venn-beta\n\n  set A',
  'VENN-BETA\nSET A\nUNION A,A',
  'venn-beta\nset __proto__\nset constructor\nunion __proto__,constructor["toString"]\n  text hasOwnProperty["__proto__"]\nstyle __proto__ __proto__:red, constructor:blue\nstyle hasOwnProperty color:red',
  '---\ntitle: Front\n---\nvenn-beta\nset A',
  '---\ntitle: Front\n---\nvenn-beta\ntitle Inside\nset A',
  '---\ntitle: "  spaced  "\n---\nvenn-beta\nset A',
  configured('  venn:\n    padding: 40\n    width: 300', 'venn-beta\nset A\nset B\nunion A,B'),
  '%%{init: {"venn": {"useDebugLayout": true}}}%%\nvenn-beta\nset A\nset B\nunion A,B["x"]',
  'venn-beta\n  set A["Alpha"]:20\n    text A1["React"]\n  set B[Beta]:12\n  union A,B["AB"]:3\n  style A fill:#ff6b6b, stroke-width:3\n  style A,B color:#333\n  style A1 color:red\n  style B fill-opacity:0.5\n  style A stroke:black',
];

// Pele's hostile templates, tests/render/venn-security.test.ts.
export function hostile(pele) {
  const payloads = arrayIn(join(pele, 'tests/support/inert.ts'), 'export const PAYLOADS = [');
  const templates = arrayIn(join(pele, 'tests/render/venn-security.test.ts'), 'const TEMPLATES: ((p: string) => string)[] = [', { V: 'venn-beta\n' });
  const out = [];
  for (const t of templates) for (const p of payloads) out.push(t(p));
  return out.filter((s) => !lone(s));
}

// The corpus the suites start from: documentation, specs, Pele's tests and the cases above.
export function corpus(pele) {
  const out = new Set();
  const dir = join(pele, 'tests/compat/venn');
  for (const file of readdirSync(dir).filter((f) => /\.spec\.[jt]s$/.test(f))) for (const s of literals(join(dir, file), /venn-beta/)) out.add(s);
  for (const s of JSON.parse(readFileSync(join(pele, 'tests/corpus/venn-docs.json'), 'utf8'))) out.add(s);
  const render = join(pele, 'tests/render/venn.test.ts');
  for (const s of literals(render, /venn-beta/)) out.add(s);
  // What Pele's render tests draw with svgOf, which puts the header before it.
  for (const m of readFileSync(render, 'utf8').matchAll(/svgOf\(\s*(`(?:[^`\\]|\\.)*`|'(?:[^'\\\n]|\\.)*')/g)) {
    try {
      out.add('venn-beta\n' + new Function('return ' + m[1])());
    } catch {
      continue;
    }
  }
  for (const s of arrayIn(join(pele, 'tests/venn/oracle.test.ts'), 'const SEEDS = [')) out.add(s);
  for (const s of WRITTEN) out.add(s);
  return [...out].filter((s) => !lone(s) && s.length <= 4000);
}

// Diagrams of random statements over a few ids, most of which parse, as Pele's oracle test
// builds them, with titles and styles of more kinds.
export function statementDiagrams(rnd) {
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  const id = () => pick(['A', 'B', 'C', '"D e"', 'x-1', '_y']);
  const label = () => pick(['', '', '["L"]', '[L m]', '[""]', '["a<br>b"]', '["#35; x"]']);
  const size = () => pick(['', '', ':5', ': 2.5', ':-1', ':.5', ':0']);
  const indent = () => pick(['', '', '  ', '    ', '\t', ' ']);
  const tail = () => pick(['t1', '"q r"', '7', 't2["L"]', '"q"[L]', '7["L"]', 't3 %% c', 'text', 'set']);
  const statements = [
    () => `set ${id()}${label()}${size()}`,
    () => `union ${id()},${id()}${label()}${size()}`,
    () => `union ${id()},${id()},${id()}${label()}`,
    () => `text ${tail()}`,
    () => `text ${id()} ${tail()}`,
    () => `text ${id()},${id()} ${tail()}`,
    () => `Text ${tail()}`,
    () => `style ${id()} fill:${pick(['red', '#ff6b6b', 'rgb(1, 2, 3)', '"x y"', 'a b 5'])}`,
    () => `style ${id()},${id()} color:#333, stroke-width:${pick(['2', '2px', '.5'])}`,
    () => `title ${pick(['T', 'A b c', '"quoted"', 'x # y', 'a; b'])}`,
    () => pick(['', '   ', '%% comment', 'x%% eaten', '%%{ directive }%%']),
  ];
  return () => {
    let src = pick(['venn-beta', '  venn-beta', '\nvenn-beta']) + pick(['\n', '\n', ' ', '\r\n']);
    for (let k = 1 + Math.floor(rnd() * 8); k > 0; k--) src += indent() + pick(statements)() + pick(['\n', '\n', '\n', ' ', '\r\n', '\n\n']);
    return src;
  };
}

// Whole diagrams that Pele draws: one set to many, with and without sizes, the overlaps of
// pairs, of three and of more, labels and texts of every kind of text, styles on sets,
// overlaps and texts, titles, and front matter.
export function drawableDiagrams(rnd) {
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  const chance = (p) => rnd() < p;
  const names = ['A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'Frontend', 'Backend', '"Data Science"', 'x_1', '"é"', '"中文"', 'Ops'];
  const labels = ['Alpha', '"Beta"', '"A rather long label for a set that goes on and on"', '"a<br>b"', '"<b>bold</b> text"', '"#35; &amp; x"',
    '"中文的标签"', '"😀 emoji"', 'Shared', '"x"', '"Supercalifragilisticexpialidocious"', '"<i>it</i>"', '"fa:fa-car car"'];
  const texts = ['t1', '"a note"', '7', 'React["React"]', 'API["API layer"]', 'L["A long text in a region"]', 'n["#quot;q#quot;"]', 'k["中文"]'];
  const sizeOf = () => {
    const r = rnd();
    if (r < 0.4) return '';
    if (r < 0.5) return `:${pick(['0', '-5', '1e9', '0.001', '99999999999999999999', '.5'])}`;
    return `:${chance(0.5) ? Math.round(rnd() * 40) : (rnd() * 30).toFixed(Math.floor(rnd() * 3))}`;
  };
  const styleOf = () =>
    pick([
      'fill:#ff6b6b', 'color:#333', 'fill:gold', 'stroke:black', 'stroke-width:3', 'fill-opacity:0.5', 'fill-opacity:1', 'fill-opacity:0',
      'fill-opacity:abc', 'fill:red, fill-opacity:0.7', 'font-weight:bold', 'font-style:italic', 'font-weight:700, color:red',
      'stroke-dasharray:4 2', 'stroke-linecap:round', 'stroke-linejoin:bevel', 'opacity:0.5', 'fill:none', 'fill:"url(x)"',
      'color:rgb(1, 2, 3)', 'fill:rgba(10,20,30,0.5)', 'font-size:30px', 'stroke:#00f, stroke-width:2px', 'fill:"#abc"',
    ]);
  return () => {
    let head = '';
    if (chance(0.15)) {
      head = '---\n';
      if (chance(0.5)) head += `title: ${pick(['Front', 'A title from the front matter that is long'])}\n`;
      if (chance(0.5)) head += `config:\n  venn:\n    padding: ${pick(['3', '40'])}\n  fontSize: ${pick(['20', '10'])}\n`;
      head += '---\n';
    }
    const out = ['venn-beta'];
    if (chance(0.4)) out.push(`  title ${pick(['Team overlap', '"Quoted title"', 'What makes a good feature', 'T', 'A title much wider than the diagram it names, which is narrow'])}`);
    const r = rnd();
    const count = r < 0.1 ? 1 : r < 0.35 ? 2 : r < 0.7 ? 3 : r < 0.9 ? 4 + Math.floor(rnd() * 4) : 8 + Math.floor(rnd() * (chance(0.2) ? 70 : 10));
    const sets = [];
    for (let i = 0; i < count; i++) {
      const name = i < names.length && chance(0.8) ? names[i] : `s${i}`;
      if (sets.includes(name)) continue;
      sets.push(name);
      out.push(`  set ${name}${chance(0.3) ? `[${pick(labels)}]` : ''}${sizeOf()}`);
      if (chance(0.15)) for (let k = 1 + Math.floor(rnd() * 3); k > 0; k--) out.push(`    text ${pick(texts)}`);
    }
    const unions = [];
    if (sets.length >= 2) {
      const pairs = Math.floor(rnd() * Math.min(8, (sets.length * (sets.length - 1)) / 2 + 1));
      for (let k = 0; k < pairs; k++) {
        const a = pick(sets);
        let b = pick(sets);
        if (b === a && chance(0.9)) b = sets[(sets.indexOf(a) + 1) % sets.length];
        unions.push([a, b]);
      }
      if (sets.length >= 3 && chance(0.5)) {
        const shuffled = [...sets].sort(() => rnd() - 0.5);
        unions.push(shuffled.slice(0, chance(0.8) ? 3 : Math.min(sets.length, 3 + Math.floor(rnd() * 3))));
      }
    }
    for (const u of unions) {
      out.push(`  union ${u.join(',')}${chance(0.4) ? `[${pick(labels)}]` : ''}${sizeOf()}`);
      if (chance(0.2)) out.push(`    text ${pick(texts)}`);
    }
    if (chance(0.2) && sets.length > 0) out.push(`  text ${pick(sets)} ${pick(texts)}`);
    if (chance(0.1) && sets.length > 1) out.push(`  text ${sets[0]},${sets[1]} orphan`);
    if (chance(0.45)) {
      for (let k = 1 + Math.floor(rnd() * 4); k > 0; k--) {
        const r2 = rnd();
        const target = r2 < 0.55 || unions.length === 0 ? pick(sets) : r2 < 0.85 ? pick(unions).join(',') : pick(['t1', 'React', 'API', 'L', 'k']);
        out.push(`  style ${target} ${Array.from({ length: 1 + Math.floor(rnd() * 3) }, styleOf).join(', ')}`);
      }
    }
    return head + out.join('\n') + pick(['', '\n']);
  };
}

// A model as the oracle writes it, its fields in order, what is left undefined as null.
function dumpModel(m) {
  return {
    title: m.title ?? null,
    subsets: m.subsets.map((s) => ({ sets: s.sets, size: enc(s.size), label: s.label ?? null })),
    textNodes: m.textNodes.map((t) => ({ sets: t.sets, id: t.id, label: t.label ?? null })),
    styles: m.styles.map((s) => ({ targets: s.targets, styles: [...s.styles] })),
  };
}

// Each case's tokens, with where each starts in UTF-16 code units, and its model or error.
export default async function ({ src, pele, write, random }) {
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { venn } = await src('diagrams/venn/index.ts');
  const { VennLexer, T, TOKEN_NAMES } = await src('diagrams/venn/lexer.ts');
  const { parseVenn } = await src('diagrams/venn/parser.ts');
  const { VennDb } = await src('diagrams/venn/db.ts');

  // The tokens of the text, with the lexer's question about indentation answered as given.
  const tokens = (source, indent) => {
    const lexer = new VennLexer(source, { getIndentMode: () => indent, setIndentMode: () => undefined });
    const out = [];
    for (;;) {
      const type = lexer.next();
      out.push([TOKEN_NAMES[type], lexer.text, lexer.start]);
      if (type === T.END || type === T.ERROR) break;
    }
    return out;
  };
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
    const out = {
      src: source,
      tokens: tokens(source, false),
      indented: tokens(source, true),
      direct: parsed(() => {
        const db = new VennDb();
        parseVenn(source, db);
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
    if (out.type !== 'venn') return out;
    const piped = parsed(() => venn.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title));
    // Most text comes out of the pipeline as it went in, and parses the same.
    if (JSON.stringify(piped) === JSON.stringify(out.direct)) return { ...out, same: true };
    return { ...out, ...piped };
  };

  const frags = fragments(pele);
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
  const next = mutator(base, frags, rnd);
  for (let i = 0; i < 3500; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  const diagrams = statementDiagrams(random(10));
  for (let i = 0; i < 1500; i++) add(diagrams());
  const drawable = drawableDiagrams(random(12));
  for (let i = 0; i < 300; i++) add(drawable());
  // Random strings of tokens, after each kind of statement and in each state of the lexer, as
  // Pele's oracle test strings them.
  const trnd = random(8);
  const pieces = [...new Set([...frags, ...frags.map((f) => f.trim()).filter((f) => f !== '')])];
  const openers = [
    'venn-beta\n', 'venn-beta\nset A\n', 'venn-beta\nset A\nset B\nunion A,B\n  ', 'venn-beta\nset A\n  text ', 'venn-beta\nset A\nstyle A ',
    'venn-beta\nset A\nstyle A fill:', 'venn-beta\nset A\ntext A ', 'venn-beta ', '',
  ];
  for (let i = 0; i < 2000; i++) {
    let s = openers[Math.floor(trnd() * openers.length)];
    for (let k = 1 + Math.floor(trnd() * 8); k > 0; k--) s += pieces[Math.floor(trnd() * pieces.length)] + (trnd() < 0.4 ? ' ' : trnd() < 0.3 ? '\n' : '');
    add(s);
  }

  write('diagram/venn/testdata/pele/venn.json', { cases: all.map(run) });
}
