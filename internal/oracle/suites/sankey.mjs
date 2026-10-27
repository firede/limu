// Sankey diagrams parsed as Pele parses them, from text to tokens and to SankeyModel:
// Mermaid's documentation and specs, the inputs of Pele's own tests, cases written here, those
// mutated at random, rows of random fields, and random strings of fragments. Each case is
// lexed, prepared as Mermaid's prepareTextForParsing prepares it, and parsed as the raw text,
// and parsed as Pele's parse does it, after the preprocessing every diagram goes through.
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { enc, peleError } from '../lib/value.mjs';
import { arrayIn, docs, fileLiterals, lone, picker } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { locator } from '../lib/located.mjs';

// The fragments of Pele's tests/sankey/oracle.test.ts, and some for the preparation of the
// text and the reading of numbers.
export const FRAGMENTS = [
  ' ', '  ', '\n', '\n\n', '\r\n', '\r', '\t', ',', ',,', '"', '""', '"""', '","', '"a,b"', '"a ""b"" c"', '"\n"', 'a', 'b',
  'A b', '1', '0', '-1', '1.5', '.5', '1e3', '1e', 'NaN', 'Infinity', 'x1', '10%', 'sankey', 'sankey-beta', 'sankey\n',
  'SANKEY', 'sankey-', 'sankeyx', 'sankey-betax', "'", '&', '/', '-', ';', '#', '%', '%%', '~', '\x7f', '\x1f', 'é', 'ö',
  '__proto__', 'constructor', 'a,b,1', '\na,b,1', 'a,b,1\n', '"a","b","1"', ' a , b , 1 ',
  ' ', ' ', '　', '﻿', '\v', '\f', '\r\r\n', ' \n ', '-0', '+5', '1e-3', '1E+2', '0x10', '1_000',
  '-Infinity', '+Infinity', '5e-324', '1e308', '1e309', '.', '-.', '1.2.3', '9'.repeat(30), 'sanKey', 'K',
  '😀', '中文', '#35;', '&amp;', '<b>b</b>', '"x\r\ny"', 'Sankey-Beta', '"', '%% c\n',
];

const energy = (pele) => readFileSync(join(pele, 'tests/compat/sankey/upstream/energy.csv'), 'utf8');

// Mermaid's own preparation of the text, as Pele's oracle test has it.
const reference = (text) => text.replaceAll(/^[^\S\n\r]+|[^\S\n\r]+$/g, '').replaceAll(/([\n\r])+/g, '\n').trim();

// Cases written here, for the corners of the grammar and of the drawing.
export const WRITTEN = [
  '',
  // Names in Unicode, and entity codes for characters XML does not allow, which are errors.
  'sankey\nÉlectricité,水力,5\n😀,"b, 中",2',
  'sankey\n"Ünïcödé ""q""",ab c,1',
  'sankey\na&#1;b,c,1',
  'sankey\na,"b&#xFFFE;",1',
  'sankey\na&#9;b,c&#xFFFF;,1',
  'sankey\na&#x1F600;b,c&#65533;,1',
  'sankey\na\u0085b,c,1',
  'sankey\na￾b,c,1',
  'sankey',
  'sankey\n',
  'sankey-beta',
  'sankey\na,b,1',
  'sankey\na,b,1\n',
  'sankey\na,b',
  'sankey\na,b,',
  'sankey\na,,1',
  'sankey\n,b,1',
  'sankey\na,b,1,2',
  'sankey\na,b,1\n\n\nc,d,2',
  'sankey a,b,1',
  'sankey\r\na,b,1\r\nb,c,2\r\n',
  'sankey\ra,b,1\rb,c,2',
  'sankey\na,b,"1"',
  'sankey\n"a","b","1"',
  'sankey\n"a",b,1\n"a",c,2',
  'sankey\n"a ""quoted"" name",b,1',
  'sankey\n"a, with comma",b,1',
  'sankey\n"line\nbreak",b,1',
  'sankey\n"unclosed,b,1',
  'sankey\n"a"x,b,1',
  'sankey\na,b,1 2',
  'sankey\na,b, 7 ',
  'sankey\n  a  ,  b  ,  7  ',
  'sankey\na,b,x',
  'sankey\na,b,-5',
  'sankey\na,b,0\nb,c,0',
  'sankey\na,b,1e400',
  'sankey\na,b,5e-324',
  'sankey\na,b,5e-324\nc,d,0',
  'sankey\na,b,1e308\nb,c,1e308\nd,b,1e308',
  'sankey\na,a,1',
  'sankey\na,b,1\nb,a,1',
  'sankey\na,b,1\nb,c,1\nc,a,1',
  'sankey\nx,y,1\na,b,1\nb,c,1\nc,b,1',
  'sankey\na,b,1\nb,c,2\n"c ""d""","e,f",3.5\n',
  'sankey-beta\n\n  a , b , 10  \n\n"x",y,"7"\n',
  'SANKEY\na,b,abc\na,a,1',
  'sankey\na\tb,c,1',
  'sankey\né,b,1',
  'sankey\na,b,1\t',
  'sankey\n__proto__,constructor,1\nconstructor,toString,2',
  'sankey\na #35; b,c &amp; d,1',
  'sankey\n<b>bold</b>,b,1',
  '---\ntitle: Energy\n---\nsankey\na,b,1',
  '---\ntitle: Energy\n---\nsankey-beta\na,b,1\nb,c,0.5',
  'sankey\n%% comment\na,b,1\n%% another\nb,c,2',
  '%%{init: {"sankey": {"width": 300}}}%%\nsankey\na,b,1',
  '  sankey\n  a,b,1\n  b,c,2  ',
  'sankey a,b,1',
  'sankey\n a,b,1 ',
  'sankeyK\na,b,1',
  'sanKey\na,b,1',
  'sanKey-beta\na,b,1',
];

// Pele's hostile templates, tests/render/sankey-security.test.ts, with each of its payloads.
export function hostile(pele) {
  const payloads = arrayIn(join(pele, 'tests/support/inert.ts'), 'export const PAYLOADS = [');
  const templates = arrayIn(join(pele, 'tests/render/sankey-security.test.ts'), 'const TEMPLATES: ((p: string) => string)[] = [', {
    BODY: 'a,b,2\nb,c,1\n',
    quoted: (p) => `"${p.replaceAll('"', '""')}"`,
    json: (p) => JSON.stringify(p),
  });
  const out = [];
  for (const t of templates) for (const p of payloads) out.push(t(p));
  return out.filter((s) => !lone(s) && s.length <= 4000);
}

// The corpus the suites start from: documentation, specs, Pele's tests and the cases above.
export function corpus(pele, cleanupComments) {
  const out = new Set();
  for (const s of docs(pele, 'sankey')) out.add(s);
  for (const file of ['tests/compat/sankey/sankey.spec.ts', 'tests/render/sankey.test.ts', 'tests/sankey/layout.test.ts', 'tests/render/max-width.test.ts']) {
    for (const s of fileLiterals(join(pele, file), /sankey/i)) out.add(s);
  }
  const csv = energy(pele);
  for (const head of ['sankey-beta\n\n ', 'sankey\n\n ']) {
    out.add(head + csv);
    out.add(cleanupComments(head + csv));
  }
  for (const s of WRITTEN) out.add(s);
  for (const s of [...out]) out.add(reference(s));
  return [...out].filter((s) => !lone(s) && s.length <= 6000);
}

// Diagrams of random rows, most of which get through the grammar.
export function rowDiagrams(rnd) {
  const pick = picker(rnd);
  const names = ['a', 'b', 'c', 'Coal', 'Gas', '"x y"', '"a, b"', '"q ""q"""', ' pad ', 'Losses', '"é"', 'n1', 'n2', 'n3', '#35;', '&amp;'];
  const amounts = ['1', '2.5', '0', '-1', '.5', '100', '1e3', 'x', '"7"', ' 3 ', '0.001', '12345.678', 'NaN', '1e400', '-0'];
  return () => {
    let src = pick(['sankey\n', 'sankey-beta\n', 'sankey\n\n', 'SANKEY\n', '  sankey\n', 'sankey\r\n']);
    const rows = 1 + Math.floor(rnd() * 8);
    for (let k = 0; k < rows; k++) {
      src += `${pick(names)},${pick(names)},${pick(amounts)}`;
      if (rnd() < 0.05) src += pick([',', ',1', ' x']);
      src += pick(['\n', '\n', '\r\n', '\n\n', '\r', '']);
    }
    return src;
  };
}

// A model as the oracle writes it: its title where it has one, each node as its id and index,
// and each link as the indexes of its ends and its value.
function dumpModel(m) {
  return {
    title: m.title || undefined,
    accTitle: m.accTitle || undefined,
    accDescr: m.accDescr || undefined,
    nodes: m.nodes.map((n) => [n.id, n.index]),
    links: m.links.map((l) => [l.source.index, l.target.index, enc(l.value)]),
  };
}

// Each case's tokens, with where each starts in UTF-16 code units, its text as prepared, and
// its model or error.
export default async function ({ src, pele, write, random }) {
  const located = await locator(src);
  const { preprocess, encodeEntities, cleanupComments } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { sankey } = await src('diagrams/sankey/index.ts');
  const { tokenize } = await src('diagrams/sankey/lexer.ts');
  const { parseSankey, prepareTextForParsing } = await src('diagrams/sankey/parser.ts');
  const { SankeyDb } = await src('diagrams/sankey/db.ts');

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
      prepared: prepareTextForParsing(source),
      direct: parsed(() => {
        const db = new SankeyDb();
        parseSankey(source, db);
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
    if (out.type !== 'sankey') return out;
    const piped = parsed(() => located(source, false, () => sankey.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title)));
    if (JSON.stringify(piped) === JSON.stringify(out.direct)) return { ...out, same: true };
    return { ...out, ...piped };
  };

  const base = corpus(pele, cleanupComments);
  const all = [];
  const seen = new Set();
  const add = (s) => {
    if (s.length <= 6000 && !lone(s) && !seen.has(s)) {
      seen.add(s);
      all.push(s);
    }
  };
  base.forEach(add);
  hostile(pele).forEach(add);
  // Mutated text; one in six is cut short, and every other one is also prepared as Mermaid
  // hands it to its parser.
  const rnd = random(17);
  const next = mutator(base, FRAGMENTS, rnd);
  for (let i = 0; i < 3500; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
    if (i % 2 === 0) add(reference(s));
  }
  const rows = rowDiagrams(random(18));
  for (let i = 0; i < 1500; i++) add(rows());
  // Random strings of fragments, as Pele's oracle test assembles them.
  const frnd = random(19);
  for (let i = 0; i < 2000; i++) {
    let s = frnd() < 0.7 ? 'sankey\n' : '';
    for (let k = 1 + Math.floor(frnd() * 14); k > 0; k--) s += FRAGMENTS[Math.floor(frnd() * FRAGMENTS.length)];
    add(s);
  }

  write('diagram/sankey/testdata/pele/sankey.json', { cases: all.map(run) });
}
