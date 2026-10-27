// Timeline diagrams parsed as Pele parses them, from text to tokens and to TimelineModel:
// Mermaid's documentation and specs, the sources of Pele's tests, cases written here, those
// sources mutated at random, diagrams built from random statements, and random strings of
// tokens. Each case is lexed and parsed as the raw text, and parsed as Pele's parse does it,
// after the preprocessing every diagram goes through.
import { peleError } from '../lib/value.mjs';
import { lone, picker, testStrings, docs } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { locator } from '../lib/located.mjs';

// Pele's fragments, tests/timeline/oracle.test.ts, and some for what Pele draws.
export const FRAGMENTS = [
  'timeline', 'timeline ', 'timeline LR', 'timeline TD', 'timeline\tTD', 'Timeline lr', 'TIMELINE', ' LR', ' TD', 'LR', 'TD',
  'lr', 'td', 'LRx', 'TD_', 'timelines', 'title ', 'title', 'title\n', 'TITLE ', 'titles', 'section ', 'section', 'section\n',
  'Section ', 'sections', 'accTitle: ', 'accTitle', 'accTitle :', 'acctitle:', 'accDescr: ', 'accDescr', 'accDescr {',
  'ACCDESCR{', 'accDescr\n:', 'accTitle\n\n:\n', '{', '}', '}\n', ':', ' : ', ': ', ' :', ':\n', '::', ': :', ':x', ' ', '  ',
  '\n', '\n\n', '\t', '\r', '\r\n', '\u000b', '\u000c', '\u00a0', '\u2028', '\u2029', '\ufeff', '#', '# c', ';', '%', '%%',
  '%% c', '%{', '%%{', '%%{init: {}}%%', 'a%%b', '}%%', '\n%%', ' %%', '<br>', '<br/>', 'x', 'é', '日本', '1', '2002', '_',
  '-', '.', ',', '"', "'", '2002 : a', ' : b : c', '\n     : d',
  ':\u3000', 'section\u3000', 'title\u00a0', '😀', '😀%%', '#35;', '&amp;', '<b>', 'fa:fa-car', '**b**',
  'a long event text that has to wrap over several lines of its card', 'Supercalifragilisticexpialidocious',
];

// Pele's seeds, tests/timeline/oracle.test.ts.
export const SEEDS = [
  'timeline',
  'timeline LR',
  'timeline TD\n',
  'timeline\n    title History\n    2002 : LinkedIn\n    2004 : Facebook : Google\n         : Orkut\n    2005 : YouTube\n',
  'timeline TD\n  title T\n  section One\n    A : a1 : a2\n    B\n  section Two\n    C : c1\n',
  'timeline\n  accTitle: Acc title\n  accDescr: Acc description\n  2001 : a\n',
  'timeline\n  accDescr {\n    Several\n    lines\n  }\n  2001 : a\n',
  'timeline\n  accDescr {}title x\n  2001 : a',
  'timeline\n  %% comment\n  2001 : a %% not a comment\n  # comment\n  2002 # comment\n  x%%hidden\n',
  'timeline\n  title\nnext line\n  section\nnext line\n  a :\nb\n',
  'timeline\n  a : b:c : d::e : f:\n  a :b\n  a : \n',
  'timeline\n  a : 50% done\n  % comment\n  %{ not a comment\n',
  'TIMELINE lr\n  TITLE Caps\n  SECTION Caps\n  ACCTITLE: caps\n  ACCDESCR: caps\n  AccDescr { caps }\n',
  'timeline\r\n  title Windows\r\n  2001 : a\r\n',
  'timeline\n  section A : B\n  timeline\n',
  '  \n  timeline\n\n\n  2001\n\n  : late event\n',
  'timeline\n  accTitle:',
  'timeline\n  accDescr {\n  never closed',
  'timeline title T section S p : e',
  'timeline\n  section S: e\n  ; : e ; f\n',
];

// Cases written here, for the corners of the grammar and of the drawing.
export const WRITTEN = [
  '',
  'timeline x',
  'timelineTD',
  'timeline  td',
  'timeline TDx',
  'timeline\tlr\n2001',
  'timeline\n: e',
  'timeline\n  : e\n  p',
  'timeline\n  p : e\n  title\u00a0Spaced\n  section\u3000Wide\n  q :\u3000wide',
  'timeline\n  😀 : 😀\n  😀%% hidden\n  é%%x\n  p\n',
  'timeline\n  accDescr {\n    a\n\n      b\n   c\n  }\n',
  'timeline\n  accDescr: \n  accTitle:   \n',
  'timeline\n  accDescr {',
  'timeline\n  accDescr {}',
  'timeline\n  accDescr',
  'timeline\n  accTitle x',
  'timeline\n  title a\n  title b',
  'timeline\n  section a\n  section b\n  p\n  section c',
  'timeline\n  p\n  section a\n  q\n',
  'timeline\n  p1\n  p2 : e\n  section S\n  q : f : g\n',
  'timeline\n  section\n',
  'timeline\n  section :',
  'timeline\n  p ::',
  'timeline\n  p : :',
  'timeline\n  p :',
  'timeline\n  p # c : e\n',
  'timeline\n  p : e # not a comment\n',
  '---\ntitle: Front\n---\ntimeline\n  p : e',
  '---\ntitle: Front\n---\ntimeline\n  title Own\n  p : e',
  '---\nconfig:\n  timeline:\n    disableMulticolor: true\n---\ntimeline\n  a\n  b\n  c',
  "%%{init: {'timeline': {'disableMulticolor': true}}}%%\ntimeline\n  a : x\n  b : y",
  'timeline\n  p : a #35; b #quot;c#quot;\n  q : x < y & z\n  r : <b>bold</b> and <i>it</i>\n  s : fa:fa-car icon\n',
  'timeline\n  p : one<br>two<br/>three<BR />four\n  q : **not** _markdown_\n',
  'timeline\n  Supercalifragilisticexpialidocious : Pneumonoultramicroscopicsilicovolcanoconiosis\n',
  'timeline\n  日本語の長い期間のラベルがここにあります : 中文的事件描述也很长很长很长很长很长很长\n',
  'timeline TD\n  section A very long section name that will not fit in one line of the header\n  p : e\n',
];

// Every diagram source that appears as a string literal in Pele's ported specs, as its
// loadCorpus('timeline', /timeline/) finds them.
const specSources = (pele) => testStrings(pele, 'tests/compat/timeline', /\.spec\.[jt]s$/, /timeline/);

// The corpus the suites start from: documentation, specs, Pele's seeds and the cases above.
export function corpus(pele) {
  return [...new Set([...specSources(pele), ...docs(pele, 'timeline'), ...SEEDS, ...WRITTEN])].filter((s) => !lone(s));
}

// Diagrams of random statements, most of which get through the grammar.
export function statementDiagrams(rnd) {
  const pick = picker(rnd);
  const word = () => pick(['2002', 'A', 'Bronze Age', 'x y', 'é', '日本', '😀', 'a:b', 'p#', '50%', '<br>', 'a<br>b', 'q ']);
  const statements = [
    () => `title ${word()}`,
    () => `section ${word()}`,
    () => word(),
    () => `${word()} : ${word()}`,
    () => `${word()} : ${word()} : ${word()}`,
    () => `   : ${word()}`,
    () => `accTitle: ${word()}`,
    () => `accDescr: ${word()}`,
    () => `accDescr {\n  ${word()}\n   ${word()}\n}`,
    () => pick(['%% c', '# c', '', '  ', 'timeline', ':', ': ', 'section', 'title']),
  ];
  return () => {
    let src = pick(['timeline\n', 'timeline LR\n', 'timeline TD\n', '  timeline\n', 'TIMELINE td\n']);
    for (let k = 1 + Math.floor(rnd() * 8); k > 0; k--) src += pick(['', '  ', '\t']) + pick(statements)() + pick(['\n', '\n', '\r\n', '\n\n']);
    return src;
  };
}

// Whole diagrams that Pele draws, with labels of all lengths and of every kind of text, sections
// or none, periods before the first section, and the settings of front matter and directives.
export function drawableDiagrams(rnd) {
  const pick = picker(rnd);
  const chance = (p) => rnd() < p;
  const periods = ['2002', '2004', 'Q1', 'Bronze Age', 'Industry 4.0', '7600 BC', 'The long period of the very late middle ages',
    'Supercalifragilisticexpialidocious', '日本語の期間', 'é ü ß', '😀 era', 'a #35; b', 'x < y & z', 'one<br>two', 'A', 'fa:fa-car x'];
  const events = ['LinkedIn', 'Facebook', 'Google', 'YouTube', 'Twitter', 'Machinery, Water power, Steam <br>power',
    "Britain's oldest known house was built in Orkney, Scotland", 'The first metal objects are made in Britain. Some other nice things happen.',
    'Pneumonoultramicroscopicsilicovolcanoconiosis', 'Electricity', '中文的事件描述也很长很长很长很长很长很长', 'x', '😀😀😀',
    'a #quot;quoted#quot; event', '<b>tag</b> text', '**b** _i_', 'a:b', 'https://example.com/a/very/long/address/without/spaces'];
  const sections = ['17th-20th century', '21st century', 'Stone Age', 'A', 'A very long section name that will not fit in its header at all', 'é', '日本'];
  return () => {
    let head = '';
    if (chance(0.3)) {
      head += '---\n';
      if (chance(0.6)) head += `title: ${pick(['Front', 'A much longer title from the front matter of the timeline', 'T'])}\n`;
      if (chance(0.4)) head += `config:\n  timeline:\n    disableMulticolor: ${pick(['true', 'false'])}\n`;
      head += '---\n';
    } else if (chance(0.1)) {
      head += `%%{init: {'timeline': {'disableMulticolor': ${pick(['true', 'false'])}}}}%%\n`;
    }
    const lines = [pick(['timeline', 'timeline', 'timeline LR', 'timeline TD'])];
    if (chance(0.6)) lines.push(`  title ${pick(['History of Social Media Platform', 'T', 'A title that is long enough to be wider than the timeline under it, and to wrap'])}`);
    if (chance(0.15)) lines.push('  accTitle: Acc', '  accDescr: Description');
    const count = Math.floor(rnd() * 8);
    const sectioned = chance(0.5);
    for (let k = 0; k < count; k++) {
      if (sectioned && (k === 0 ? chance(0.7) : chance(0.3))) lines.push(`  section ${pick(sections)}`);
      if (chance(0.05)) lines.push(`  section ${pick(sections)}`);
      let line = `  ${pick(periods)}`;
      const n = Math.floor(rnd() * 4);
      for (let e = 0; e < n; e++) {
        if (chance(0.3)) {
          lines.push(line);
          line = `       : ${pick(events)}`;
        } else {
          line += ` : ${pick(events)}`;
        }
      }
      lines.push(line);
    }
    if (sectioned && chance(0.2)) lines.push(`  section ${pick(sections)}`);
    return head + lines.join('\n') + pick(['', '\n']);
  };
}

// A model as the oracle writes it, its fields in order, a string left undefined left out.
function dumpModel(m) {
  return {
    title: m.title || undefined,
    accTitle: m.accTitle || undefined,
    accDescr: m.accDescr || undefined,
    direction: m.direction,
    sections: m.sections,
    periods: m.periods.map((p) => ({ text: p.text, events: p.events, section: p.section, sectionIndex: p.sectionIndex })),
  };
}

// Each case's tokens, with where each starts in UTF-16 code units, and its model or error.
export default async function ({ src, pele, write, random }) {
  const located = await locator(src);
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { timeline } = await src('diagrams/timeline/index.ts');
  const { tokenize } = await src('diagrams/timeline/lexer.ts');
  const { parseTimeline } = await src('diagrams/timeline/parser.ts');
  const { TimelineDb } = await src('diagrams/timeline/db.ts');

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
        const db = new TimelineDb();
        parseTimeline(source, db);
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
    if (out.type !== 'timeline') return out;
    const piped = parsed(() => located(source, false, () => timeline.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title)));
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
  // Mutated text; one in six is cut short.
  const rnd = random(7);
  const next = mutator(base, FRAGMENTS, rnd);
  for (let i = 0; i < 3000; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  const diagrams = statementDiagrams(random(10));
  for (let i = 0; i < 1000; i++) add(diagrams());
  const drawable = drawableDiagrams(random(12));
  for (let i = 0; i < 500; i++) add(drawable());
  // Random strings of tokens, where keyword boundaries and the lookahead matter.
  const trnd = random(8);
  const pieces = [...new Set([...FRAGMENTS, ...FRAGMENTS.map((f) => f.trim()).filter((f) => f !== '')])];
  const openers = ['timeline\n', 'timeline', 'timeline\n  p', 'timeline\n  p : ', 'timeline\n  section ', 'timeline\n  accDescr', ''];
  for (let i = 0; i < 1000; i++) {
    let s = openers[Math.floor(trnd() * openers.length)];
    for (let k = 1 + Math.floor(trnd() * 10); k > 0; k--) s += pieces[Math.floor(trnd() * pieces.length)] + (trnd() < 0.3 ? ' ' : '');
    add(s);
  }

  write('diagram/timeline/testdata/pele/timeline.json', { cases: all.map(run) });
}
