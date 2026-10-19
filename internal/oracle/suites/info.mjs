// Info diagrams parsed as Pele parses them: the tokens Pele's common/tokens.ts makes of the
// text with the info diagram's table, the tree parseInfo makes of them, and the model
// buildInfo makes of the tree with the title the text's front matter gives. The cases are
// Mermaid's documentation and specs, the inputs of Pele's own tests, its hostile templates
// filled with its payloads, cases written here, random diagrams, and all of those mutated at
// random.
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { enc, lone, mutator, peleError } from './parse.mjs';

// The fragments of Pele's tests/info/oracle.test.ts, and more.
export const FRAGMENTS = [
  'info', 'showInfo', 'title', 'title ', 'accTitle: ', 'accDescr: ', 'accDescr {', 'accDescr {}', '}', ':', '\n', '\r\n', ' ', '\t',
  '%%', '%% c', '%%{init: {}}%%', '---', '---\n', 'x', 'é', ';', '#35;', 'info showInfo', '\ntitle t', 'Info',
  '\r', ' ', '　', '😀', '中文', '&amp;', '<b>x</b>', '**b**', '\naccTitle: a', '\naccDescr {\n  one\n   two\n}',
  '\ntitle A very long title of the diagram that goes on', 'showinfo', 'INFO', ' %% c\n', '---\ntitle: T\n---\n',
];

// The inputs of Pele's tests/info/oracle.test.ts beyond its corpus, and cases written here.
export const WRITTEN = [
  'info',
  'info\naccDescr {}\ntitle t',
  'info showInfo',
  'info\nshowInfo\n',
  '\n info \n showInfo \n title x',
  'info title x',
  'info\ntitle x\naccTitle: a\naccDescr: d',
  'info showInfo title x\n\n',
  'info title x\nshowInfo',
  'info\ntitle x\n\n\ntitle y',
  'info\ntitle x\ninfo',
  'infox',
  'info showInfox',
  'showInfo',
  'info info',
  'info showInfo showInfo',
  'info%% c\nshowInfo%% d',
  'info\naccDescr { a\n b }\n',
  'info\naccDescr { a\n b } title x',
  'info unsupported',
  '---\ntitle: x\n---\ninfo',
  '%%{init: {"theme":"dark"}}%%\ninfo showInfo %% c\n',
  '---\ntitle: From front matter\n---\ninfo',
  '---\ntitle: From front matter\n---\ninfo\ntitle Own',
  'info showInfo\naccTitle: Name\naccDescr: Text',
  'info\ntitle a\nshowInfo',
  'info\ntitle A title much wider than the version under it, by far, so very much wider',
  'info\ntitle **bold** and _it_ <b>b</b> #35; &amp;',
  'info\ntitle a<br>b',
  'info\ntitle fa:fa-car car',
  'info\ntitle 中文的标题',
  'info\naccTitle:    spaced\naccDescr {\n   one\n     two\n}',
  'info\naccTitle: \naccDescr: ',
  'info\ntitle',
  'info\ntitle  ',
  '---\ntitle: ""\n---\ninfo',
  '---\ntitle: 0\n---\ninfo',
  '---\ntitle: "  "\n---\ninfo',
  'info\r\ntitle x\r\n',
  'info\n\n\n',
  '',
  ' ',
  'info\n  title indented',
  'info\ttitle tabbed',
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
      if (!/info/.test(raw)) continue;
      try {
        out.add(new Function('return `' + raw.replace(/`/g, '\\`').replace(/\$\{/g, '\\${') + '`')());
      } catch {
        continue;
      }
    }
  }
  return [...out];
}

// An array literal of a test's source, evaluated.
function arrayIn(file, start) {
  const src = readFileSync(file, 'utf8');
  const from = src.indexOf(start);
  if (from < 0) throw new Error(`no ${start} in ${file}`);
  const open = from + start.length;
  const end = src.indexOf('\n];', open);
  return new Function('return [' + src.slice(open, end) + '\n];')();
}

// Pele's hostile templates, tests/render/info-security.test.ts, filled with its payloads.
export function hostileCorpus(pele) {
  const payloads = arrayIn(join(pele, 'tests/support/inert.ts'), 'export const PAYLOADS = [');
  const templates = arrayIn(join(pele, 'tests/render/info-security.test.ts'), 'const TEMPLATES: ((p: string) => string)[] = [');
  const out = [];
  for (const t of templates) for (const p of payloads) out.push(t(p));
  return out.filter((s) => !lone(s));
}

// The corpus the suites start from: documentation, specs, the strings of Pele's tests and the
// cases above.
export function corpus(pele) {
  const docs = JSON.parse(readFileSync(join(pele, 'tests/corpus/info-docs.json'), 'utf8'));
  return [
    ...new Set([
      ...docs,
      ...testStrings(pele, 'tests/compat/info', /\.(spec|test)\.[jt]s$/),
      ...testStrings(pele, 'tests/info', /oracle\.test\.ts$/),
      ...testStrings(pele, 'tests/render', /^info.*\.test\.ts$/),
      ...WRITTEN,
    ]),
  ].filter((s) => !lone(s) && s.length <= 4000);
}

const TITLES = ['T', 'Pele', 'A title', 'A title much wider than the version under it, by far', '**b** _i_', '中文', '#35; &amp;', 'a<br>b', '😀'];

// Random info diagrams, most of them well formed.
export function randomDiagrams(rnd) {
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  const chance = (p) => rnd() < p;
  return () => {
    let s = '';
    if (chance(0.2)) s += `---\ntitle: ${pick(TITLES)}\n---\n`;
    if (chance(0.1)) s += '%%{init: {"theme": "dark"}}%%\n';
    s += pick(['', '\n', '  ']) + 'info';
    if (chance(0.4)) s += pick([' showInfo', '\nshowInfo', '\n  showInfo']);
    const lines = [];
    if (chance(0.5)) lines.push(`title ${pick(TITLES)}`);
    if (chance(0.3)) lines.push(`accTitle: ${pick(TITLES)}`);
    if (chance(0.3)) lines.push(chance(0.5) ? `accDescr: ${pick(TITLES)}` : `accDescr {\n  ${pick(TITLES)}\n   ${pick(TITLES)}\n}`);
    if (chance(0.1)) lines.push('%% a comment');
    for (let k = lines.length - 1; k > 0; k--) {
      const j = Math.floor(rnd() * (k + 1));
      [lines[k], lines[j]] = [lines[j], lines[k]];
    }
    if (lines.length) s += (chance(0.2) ? ' ' : '\n') + lines.join(pick(['\n', '\n\n', '\r\n']));
    return s + (chance(0.5) ? '\n' : '');
  };
}

export default async function ({ src, pele, write, random }) {
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { info } = await src('diagrams/info/index.ts');
  const { parseInfo, INFO_TOKENS } = await src('diagrams/info/parser.ts');
  const { tokenize } = await src('diagrams/common/tokens.ts');

  const caught = (e) => {
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  const opt = (x) => (x === undefined || x === '' ? undefined : x);
  const tokens = (source) => {
    try {
      const t = tokenize(source, INFO_TOKENS, 'info');
      return { tokens: t.kinds.slice(0, -1).map((k, i) => [INFO_TOKENS[k].name, source.slice(t.starts[i], t.ends[i])]) };
    } catch (e) {
      return { tokens: null };
    }
  };
  const direct = (source) => {
    try {
      const ast = parseInfo(source);
      return { ...tokens(source), ast: { title: enc(ast.title), accTitle: enc(ast.accTitle), accDescr: enc(ast.accDescr) } };
    } catch (e) {
      return { ...tokens(source), ...caught(e) };
    }
  };
  // Pele's parseSource for an info diagram, and the raw text given to the parser alone.
  const run = (source) => {
    const out = { src: source, direct: direct(source) };
    let pre;
    try {
      pre = preprocess(source);
      out.type = detect(pre.text);
    } catch (e) {
      return { ...out, ...caught(e) };
    }
    if (out.type !== 'info') return out;
    try {
      const m = info.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title);
      out.model = { title: opt(m.title), accTitle: opt(m.accTitle), accDescr: opt(m.accDescr) };
    } catch (e) {
      Object.assign(out, caught(e));
    }
    return out;
  };

  const all = [];
  const seen = new Set();
  const add = (s) => {
    if (s.length <= 4000 && !lone(s) && !seen.has(s)) {
      seen.add(s);
      all.push(s);
    }
  };
  corpus(pele).forEach(add);
  hostileCorpus(pele).forEach(add);
  const diagrams = randomDiagrams(random(71));
  for (let i = 0; i < 400; i++) add(diagrams());
  // Mutated text; one in six is cut short.
  const rnd = random(72);
  const next = mutator([...seen], FRAGMENTS, rnd);
  for (let i = 0; i < 1500; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  // Random strings of tokens, where the order of the token types and their ends matter.
  const trnd = random(73);
  const pieces = [...new Set([...FRAGMENTS, ...FRAGMENTS.map((f) => f.trim()).filter((f) => f !== '')])];
  const openers = ['info\n', 'info ', 'info showInfo\n', 'info\ntitle', 'info\naccDescr', ''];
  for (let i = 0; i < 500; i++) {
    let s = openers[Math.floor(trnd() * openers.length)];
    for (let k = 1 + Math.floor(trnd() * 8); k > 0; k--) s += pieces[Math.floor(trnd() * pieces.length)] + (trnd() < 0.3 ? ' ' : '');
    add(s);
  }
  write('diagram/info/testdata/pele/info.json', { cases: all.map(run) });
}
