// Ishikawa diagrams parsed as Pele parses them, from text to tokens and to IshikawaModel:
// Mermaid's documentation and specs, the sources of Pele's tests, its hostile templates
// filled with its payloads, cases written here, those mutated at random, outlines built at
// random, and random strings of tokens. Each case is tokenized and parsed as the raw text,
// and parsed as Pele's parse does it, after the preprocessing every diagram goes through.
import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import { lone, mutator, peleError } from './parse.mjs';

// The fragments of Pele's tests/ishikawa/oracle.test.ts, and more.
export const FRAGMENTS = [
  ' ', '  ', '    ', '\t', '\n', '\n\n', '\r', '\r\n', ' \n', '\n ', '\n  \n', '\n\t\n  ', ' ', '　', ' ', '﻿',
  '\v', '\f', '%', '%%', '%% c', ' %% c', '\n%% c', '\n  %% c\n', 'x %% y', '%%\r', '%% x', 'a', 'Cause', 'Sub cause',
  'A b c', '  indented', '\n    deeper', 'ishikawa', 'ishikawa-beta', 'ISHIKAWA', 'Ishikawa-Beta', 'ishikawax', 'ishikawa-betax',
  'ishikawa-', 'ishikawa_', 'ishikawa beta', 'ishikawa-beta\n', '\nishikawa', ' ishikawa', 'é', '日本', '<b>x</b>', '"q"', '#35;',
  '-', ':', '0', '😀', '&amp;', '<br>', '<br/>', 'fa:fa-car ', '**b**', '\n      ', '\n        x', '---\n', '%%{init: {}}%%\n',
  'a much longer cause than the others are', 'word '.repeat(12), '中文中文中文中文中文中文中文中文中文中文中文中文',
];

// The inputs of Pele's tests/ishikawa/oracle.test.ts beyond its corpus.
const SEEDS = [
  'ishikawa-beta\n    Blurry Photo\n        Process\n            Out of focus\n        User\n            Shaky hands\n',
  'ishikawa-beta\nProblem\nCause A\n  Subcause A1\nCause B\n',
  'ishikawa-beta\n    Problem\nCause A\n  Subcause A1\nCause B\n',
  'ishikawa\nEffect\n\tOne\n\t\tTwo\n\t\t\tThree\n\t\t\t\tFour\n\tFive\n',
  '%% first\n\n%% second\nishikawa-beta\n%% inside\n  Effect\n\n\n  %% between\n  Cause\n    Sub %% not a comment\n\n',
  'ishikawa-beta Effect on one line',
  'ISHIKAWA-BETA\nEffect\r\n  Cause\r\n    Sub\r\n',
  'ishikawa-beta\nEffect\n  A\n      B\n    C\n  D\n E\nF\n',
  'ishikawa-beta%% nothing else',
  'ishikawa-beta\n  Effect\n   \n\n  Cause\n   ',
];

const I = 'ishikawa-beta\n';

// Cases written here: the edges of the lexer and of the model, and what the render draws.
export const WRITTEN = [
  `${I}  Late delivery\n    People\n    Process\n`,
  `${I}  Just an effect\n`,
  `${I}  E\n    One\n    Two\n    Three\n    Four\n    Five\n`,
  `${I}  Effect\n    Category\n      Cause\n        Sub\n          Deeper\n            Deepest\n`,
  `${I}  Effect\n    A\n      a1\n      a2\n      a3\n        a3i\n        a3ii\n      a4\n    B\n      b1\n        b1i\n          b1i1\n    C\n    D\n      d1\n`,
  `${I}  A rather long effect that wraps onto more lines than one, as it is wider than the head allows\n    Category\n      A cause that is long enough to wrap, being wider than two hundred\n`,
  `${I}  Effect\n    A category name that is wider than one hundred and eighty units\n`,
  `${I}  <b>Bold</b> effect\n    Category <i>it</i>\n      Cause &amp; more #35;\n        fa:fa-car Drive\n`,
  `${I}  Effect\n    中文的类别\n      中文的原因很长很长很长很长很长很长很长很长很长很长很长很长\n`,
  `${I}  Effect\n    😀 emoji\n      a<br>b\n`,
  '---\ntitle: Why photos blur\n---\n' + I + '  Blurry photo\n    Process\n      Out of focus\n',
  '---\ntitle: A title much wider than the diagram it names, by quite a long way indeed, so it is\n---\n' + I + '  E\n    C\n',
  '---\ntitle: ""\n---\n' + I + '  E\n',
  '---\ntitle:\n---\n' + I + '  E\n',
  '---\nconfig:\n  ishikawa:\n    diagramPadding: 40\n  fontSize: 20\n---\n' + I + '  E\n    C\n      c\n',
  '%%{init: {"ishikawa": {"useMaxWidth": false}}}%%\n' + I + '  E\n    C\n',
  `${I}`,
  'ishikawa',
  'ishikawa-beta',
  'ishikawa-beta\n',
  'ishikawa-beta\n\n',
  'ishikawa-beta\n  \n',
  'ishikawa\n  E',
  'ishikawa E',
  'ishikawa-beta E\n  C',
  'ishikawa-beta  E',
  'ishikawa-beta\tE',
  'ishikawa-betaE',
  'ishikawa-bet\n  E',
  'ishikawa-\n  E',
  'Ishikawa-Beta\n  E',
  'ISHIKAWA\n  E',
  'ishikawa\n\n\n  E\n\n\n    C',
  'ishikawa\n  E\n    C\n  ishikawa\n',
  'ishikawa\n  E\nishikawa',
  'ishikawa\n  E\n    ishikawa-beta x',
  'ishikawa\n  E\n    ishikawax',
  'ishikawa\n  E\n    ishikawa_x',
  'ishikawa\n  E\n    ishikawa-x',
  'ishikawa\n  E %% c\n    C',
  'ishikawa\n  E\n  %% c\n    C',
  'ishikawa\n  E\n%%c\n    C',
  'ishikawa\n  E\n    %',
  'ishikawa\n  E\n    % x',
  'ishikawa\n  E\n    50% done',
  'ishikawa\n  E\r\n    C\r\n',
  'ishikawa\r\n  E\r\n    C',
  'ishikawa\r  E\r    C',
  'ishikawa\n  E     C',
  'ishikawa\n  E\n    C',
  'ishikawa\n  E\n　　C\n　　　　c',
  'ishikawa\n  E\n﻿C',
  'ishikawa\n  E\n    C   \n      c\t\t\n',
  'ishikawa\n  E\n    C\n  D\n',
  'ishikawa\n      E\n    C\n  D\n',
  'ishikawa\n  E\n      C\n    D\n  F\n',
  'ishikawa\n  E\n    C\n        c\n      d\n    e\n',
  'ishikawa\nE\nC\nD\n',
  'ishikawa\n  E\n\tC\n\t\tc\n  D\n',
  'ishikawa\n  E\n    __proto__\n      constructor\n        toString\n          hasOwnProperty\n    __proto__\n',
  'ishikawa\n  E\n    C\n      ' + 'word '.repeat(40),
  'ishikawa\n  E\n    ' + 'x'.repeat(300),
  'ishikawa\n  ' + 'y'.repeat(300) + '\n    C',
  'ishikawa\n  E\n' + '    c\n'.repeat(30),
  'ishikawa\n  E\n    C\n' + '      c\n'.repeat(30),
  'ishikawa\n  E\n    C\n      c\n' + '        s\n'.repeat(30),
  'ishikawa\n' + Array.from({ length: 40 }, (_, i) => ' '.repeat(i + 1) + `n${i}\n`).join(''),
  'ishikawa\n E\n' + Array.from({ length: 20 }, (_, i) => ' '.repeat(1 + (i % 2) * 200) + 'n\n').join(''),
  'ishikawa\n' + Array.from({ length: 60 }, (_, i) => ' '.repeat(1 + ((i * 7) % 13)) + 'n\n').join(''),
  'ishikawa\n  E\n' + '    c\n      d\n        e\n'.repeat(12),
  'not ishikawa',
  '',
];

// The literals of a test file that hold the keyword, as Pele's loadCorpus finds them.
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

// An array literal of a test's source, evaluated with the names given.
function arrayIn(file, start, names = {}) {
  const src = readFileSync(file, 'utf8');
  const from = src.indexOf(start);
  if (from < 0) throw new Error(`no ${start} in ${file}`);
  const open = from + start.length;
  const end = src.indexOf('\n];', open);
  return new Function(...Object.keys(names), 'return [' + src.slice(open, end) + '\n];')(...Object.values(names));
}

// Pele's hostile templates, tests/render/ishikawa-security.test.ts, filled with its payloads.
export function hostileCorpus(pele) {
  const payloads = arrayIn(join(pele, 'tests/support/inert.ts'), 'export const PAYLOADS = [');
  const templates = arrayIn(join(pele, 'tests/render/ishikawa-security.test.ts'), 'const TEMPLATES: ((p: string) => string)[] = [', { I });
  const out = [];
  for (const t of templates) for (const p of payloads) out.push(t(p));
  return out.filter((s) => !lone(s) && s.length <= 4000);
}

// The corpus the suites start from: documentation, specs, Pele's tests and the cases above.
export function corpus(pele) {
  const out = new Set();
  const dir = join(pele, 'tests/compat/ishikawa');
  for (const file of readdirSync(dir).filter((f) => /\.spec\.[jt]s$/.test(f))) for (const s of literals(join(dir, file), /ishikawa/i)) out.add(s);
  for (const s of JSON.parse(readFileSync(join(pele, 'tests/corpus/ishikawa-docs.json'), 'utf8'))) out.add(s);
  for (const file of ['tests/render/ishikawa.test.ts', 'tests/render/ishikawa-security.test.ts']) {
    for (const s of literals(join(pele, file), /ishikawa/i)) out.add(s);
  }
  for (const s of SEEDS) out.add(s);
  for (const s of WRITTEN) out.add(s);
  return [...out].filter((s) => !lone(s) && s.length <= 4000);
}

const WORDS = [
  'Effect', 'Blurry photo', 'Late delivery', 'Process', 'People', 'Equipment', 'Environment', 'Materials', 'Methods',
  'Measurement', 'Out of focus', 'Shaky hands', 'Too dark', 'Dirty lens', 'Subject moved too quickly', 'LENS', 'x', 'A',
  'Shutter speed too slow', 'Protective film not removed', 'a cause with rather a lot of words in it, so that it wraps',
  '中文原因', '😀 emoji', '<b>bold</b> cause', 'a<br>b', '#35; &amp; more', 'fa:fa-car Drive', 'iiiiiiii', 'WWWWWWWW',
  'supercalifragilisticexpialidocious', 'one two', 'Ünïcödé',
];
const TITLES = ['Why photos blur', 'T', 'A title wider than the diagram it names, by quite a long way, really', '中文'];

// Outlines built at random: an effect, categories and causes to a random depth, with blank
// lines, comments and the indentation of each level chosen at random.
export function outlines(rnd) {
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  const chance = (p) => rnd() < p;
  return () => {
    let s = '';
    if (chance(0.15)) s += `---\ntitle: ${pick(TITLES)}\n---\n`;
    s += pick(['ishikawa-beta', 'ishikawa-beta', 'ishikawa', 'Ishikawa-BETA']) + '\n';
    const unit = pick(['  ', '    ', '\t', ' ', '   ']);
    const base = pick(['', unit, unit + unit]);
    s += base + pick(WORDS) + '\n';
    const cats = Math.floor(rnd() * (chance(0.15) ? 12 : 6));
    const deep = chance(0.2) ? 6 : 3;
    const line = (depth) => {
      let out = base + unit.repeat(depth) + pick(WORDS);
      if (chance(0.03)) out += ' %% not a comment';
      out += chance(0.04) ? '\r\n' : '\n';
      if (chance(0.04)) out += pick(['\n', '%% c\n', '  %% c\n', '   \n']);
      return out;
    };
    const causes = (depth, room) => {
      const n = Math.floor(rnd() * (depth === 2 ? 5 : 3));
      for (let k = 0; k < n && room > 0; k++) {
        s += line(depth);
        if (depth < deep && chance(0.5)) causes(depth + 1, room - 1);
      }
    };
    for (let c = 0; c < cats; c++) {
      s += line(1);
      causes(2, deep);
    }
    if (chance(0.03)) s += pick(WORDS) + '\n';
    return s;
  };
}

function dumpNode(node) {
  return [node.text, node.children.map(dumpNode)];
}

function dumpModel(m) {
  return { title: m.title === undefined ? { $: 'undefined' } : m.title, root: m.root ? dumpNode(m.root) : null };
}

export default async function ({ src, pele, write, random }) {
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { ishikawa } = await src('diagrams/ishikawa/index.ts');
  const { IshikawaDb } = await src('diagrams/ishikawa/db.ts');
  const { parseIshikawa } = await src('diagrams/ishikawa/parser.ts');
  const { tokenize, TOKEN_NAMES } = await src('diagrams/ishikawa/lexer.ts');

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
      tokens: t.types.slice(0, -2).map((k, i) => [TOKEN_NAMES[k], source.slice(t.starts[i], t.ends[i])]),
      direct: parsed(() => {
        const db = new IshikawaDb();
        parseIshikawa(source, db);
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
    if (out.type !== 'ishikawa') return out;
    Object.assign(out, parsed(() => ishikawa.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title)));
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
  for (const s of hostileCorpus(pele)) add(s);
  const outline = outlines(random(31));
  for (let i = 0; i < 1500; i++) add(outline());
  // Mutated text; one in six is cut short.
  const rnd = random(32);
  const next = mutator(base, FRAGMENTS, rnd);
  for (let i = 0; i < 3000; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  // Random strings of tokens, as Pele's test strings them, where every kind of white space
  // meets every other and the keyword turns up where it should not.
  const trnd = random(33);
  const openers = ['ishikawa-beta\n', 'ishikawa-beta\nEffect\n', 'ishikawa\n  E\n', 'ishikawa-beta', '%% c\nishikawa-beta\n', '\n', ''];
  for (let i = 0; i < 1500; i++) {
    let s = openers[Math.floor(trnd() * openers.length)];
    for (let k = 1 + Math.floor(trnd() * 8); k > 0; k--) s += FRAGMENTS[Math.floor(trnd() * FRAGMENTS.length)];
    add(s);
  }
  // Whole lines at random depths, with blank lines and comments between them.
  const lrnd = random(34);
  const lpick = (list) => list[Math.floor(lrnd() * list.length)];
  for (let i = 0; i < 1000; i++) {
    let s = lpick(['ishikawa-beta', 'ishikawa', 'Ishikawa-BETA']) + lpick(['\n', '\n', '\r\n', ' ', '\n\n']);
    for (let k = Math.floor(lrnd() * 12); k > 0; k--) {
      s += lpick(['', ' ', '  ', '    ', '      ', '\t', '\t\t', ' \t ']) + lpick(['Cause', 'A b', 'x', '%% note', '', 'ishikawa']) + lpick(['\n', '\n', '\n', '\r\n', '\n\n', ' \n']);
    }
    add(s);
  }

  write('diagram/ishikawa/testdata/pele/ishikawa.json', { cases: all.map(run) });
}
