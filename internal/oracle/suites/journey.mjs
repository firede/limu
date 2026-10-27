// User journey diagrams parsed as Pele parses them, from text to tokens and the model: the
// sources of Mermaid's specs and documentation as Pele's tests collect them, the seeds of
// Pele's tests/journey/oracle.test.ts, cases written here, those sources mutated at random,
// diagrams built at random, and random strings of tokens.
import { enc, peleError } from '../lib/value.mjs';
import { lone, picker, testStrings, docs } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { locator } from '../lib/located.mjs';

// The fragments of Pele's tests/journey/oracle.test.ts, and more.
export const FRAGMENTS = [
  'journey', 'journey ', 'Journey', 'JOURNEY', 'journeys', 'journey_', 'title ', 'title', 'title\n', 'TITLE ', 'titles',
  'section ', 'section', 'section\n', 'Section ', 'sections', 'accTitle: ', 'accTitle', 'accTitle :', 'acctitle:',
  'accDescr: ', 'accDescr', 'accDescr {', 'ACCDESCR{', 'accDescr\n:', 'accTitle\n\n:\n', '{', '}', '}\n', ':', ' : ', ': ',
  ' :', ':\n', '::', ': 5', ': 5: Me', ': 3: Me, Cat', ':x', ',', ', ', ' ', '  ', '\n', '\n\n', '\t', '\r', '\r\n', '\u000b',
  '\u000c', '\u00a0', '\u2028', '\u2029', '\ufeff', '#', '# c', ';', ';\n', '%', '%%', '%% c', '%{', '%%{', '%%{init: {}}%%',
  'a%%b', '}%%', '\n%%', ' %%', '<br>', '<br/>', 'x', 'é', '日本', '1', '5', '-1', '3.5', 'NaN', '_', '-', '.', '"', "'",
  'Make tea: 5: Me', '\n  Do work: 1: Me, Cat',
  '\u3000', '😀', '😀%%', '中文字', '&amp;', '#35;', '#quot;', '<b>b</b>', '<script>', 'fa:fa-car', ':fa-', 'Infinity', '1e3',
  '0x10', '+4', '.5', '2.', '1_0', '---\ntitle: T\n---\n', 'section \u00a0', 'title\u3000',
];

// The seeds of Pele's tests/journey/oracle.test.ts.
export const SEEDS = [
  'journey',
  'journey\n',
  'journey\n    title My working day\n    section Go to work\n      Make tea: 5: Me\n      Go upstairs: 3: Me\n      Do work: 1: Me, Cat\n    section Go home\n      Go downstairs: 5: Me\n      Sit down: 5: Me\n',
  'journey\n  A task: 5\n  B task: 3:\n  C task: 1: a,b , c: ignored\n  D task:: x\n',
  'journey\n  accTitle: Acc title\n  accDescr: Acc description\n  Task: 5: Me\n',
  'journey\n  accDescr {\n    Several\n    lines\n  }\n  Task: 5: Me\n',
  'journey\n  accDescr {}title x\n  Task: 5',
  'journey\n  %% comment\n  Task: 5: Me %% not a comment\n  # comment\n  Task # comment\n  x%%hidden\n',
  'journey\n  title\nnext line\n  section\nnext line\n  Task\n: 5\n',
  'journey\n  Task: 5: Child#1\n  Task: 5; Task: 4\n  title a;b\n',
  'journey\n  Task:\n  Task\n',
  'journey\n  50% done: 5\n  % comment\n  %{ not a comment: 1\n',
  'JOURNEY\n  TITLE Caps\n  SECTION Caps\n  ACCTITLE: caps\n  ACCDESCR: caps\n  AccDescr { caps }\n  Task: 5\n',
  'journey\r\n  title Windows\r\n  Task: 5: Me\r\n',
  'journey\n  section A: 5\n  journey\n',
  '  \n  journey\n\n\n  Task\n\n  : 5\n',
  'journey\n  accTitle:',
  'journey\n  accDescr {\n  never closed',
  'journey title T section S Task: 5: Me',
  'journey\n  Task: abc: Me\n  Task: 100: Me\n  Task: -3: Me\n  Task: 1e3\n  Task: : Me\n',
  'journey\ntitle Adding journey diagram functionality to mermaid\nsection Documentation\nA task: 5: Alice, Bob, Charlie\nB task: 3:Bob, Charlie\nC task: 5\nD task: 5: Charlie, Alice\nE task: 5:\nsection Another section\nP task: 5:\nQ task: 5:\nR task: 5:',
  'journey\naccDescr {\n        A user journey for\n        family shopping\n      }title Adding journey diagram functionality to mermaid\naccTitle: Adding acc journey diagram functionality to mermaid\nsection Order from website',
  'journey\ntitle Adding gantt diagram functionality to mermaid\nsection Line1<br>Line2<br/>Line3</br />Line4<br\t/>Line5',
];

// Cases written here: the edges of the lexer and the parser, and what the layout makes of
// long names, many actors and odd scores.
export const WRITTEN = [
  '',
  'journey',
  'journey ',
  'journeyx\nTask: 5',
  'journey_\nTask: 5',
  'journey-\nTask: 5',
  'journey\n;',
  'journey\n:',
  'journey\n: 5',
  'journey\nTask',
  'journey\nTask:',
  'journey\nTask: 5\n:',
  'journey\ntitle',
  'journey\ntitle ',
  'journey\ntitle\n',
  'journey\ntitle\u00a0nbsp',
  'journey\ntitle\u3000ideographic',
  'journey\nsection\u00a0nbsp\nTask: 5',
  'journey\nsection a:b\nTask: 5',
  'journey\nsection a#b\nTask: 5',
  'journey\nsection ;\nTask: 5',
  'journey\naccTitle',
  'journey\naccTitle x',
  'journey\naccTitle:   spaced   \nTask: 5',
  'journey\naccDescr:\n',
  'journey\naccDescr {',
  'journey\naccDescr {}',
  'journey\naccDescr {a}b: 5',
  'journey\naccDescr {\n  one\n\n   two\n  \n  three\n}\nTask: 5',
  'journey\naccDescr\n{\nx\n}\nTask: 5',
  'journey\naccTitle {x}',
  'journey\n😀: 5: 😀',
  'journey\n😀%% x\nTask: 5',
  'journey\nx%% hidden: 5\nTask: 5',
  'journey\n}%% shown: 5',
  'journey\n%{: 5',
  'journey\n%%{init: {}}%%\nTask: 5',
  'journey\n\u2028Task: 5',
  'journey\n\u00a0Task: 5',
  'journey\nTask: 5: a, , b,',
  'journey\nTask: 5: ,',
  'journey\nTask: 5:',
  'journey\nTask: 5: a: b: c',
  'journey\nTask: 0x1F\nTask: 0b11\nTask: 0o7\nTask: 1e400\nTask: -1e400\nTask: Infinity\nTask: -Infinity\nTask: +5\nTask: .5\nTask: 5.\nTask: -0\nTask: 1_0\nTask: 3 4',
  'journey\nTask: \u00a05\u00a0: Me',
  'journey\nTask: NaN: Me',
  'journey\ntitle A\ntitle B\nTask: 5',
  '---\ntitle: Front matter\n---\njourney\nTask: 5: Me',
  '---\ntitle: Front matter\n---\njourney\ntitle Own\nTask: 5: Me',
  'journey\nsection One\nsection Two\nTask: 5',
  'journey\nsection One\nA: 5\nsection Two\nsection One\nB: 3',
  'journey\nA: 5\nsection S\nB: 4\nC: 3',
  'journey\nsection S\nA: 1\nsection S\nB: 2',
  'journey\nsection  \u00a0 spaced section \nA: 1',
  'journey\n  Task with a very long name that goes on and on past the card: 3: Me',
  'journey\n  Supercalifragilisticexpialidociousandmoreandmoreandmoreandmoreandmore: 4: Me',
  'journey\n  Averyveryveryverylongwordthatcannotfitatallinsideacardofthemostgenerouswidthandgoesonfurther: 2: Me',
  'journey\n  Short: 5: Me\n  Averylongwordbutnottoolongforthemaximum: 4: You',
  'journey\n  日本語の長いタスクの名前はカードの中で折り返されるはずです: 3: 私',
  'journey\n  😀😀😀😀😀😀😀😀😀😀😀😀😀😀😀😀😀😀😀😀😀😀😀😀: 3: 😀',
  'journey\n  a&amp;b&lt;c&gt;d #35; #quot;e#quot; &#x41;&#65;: 3: &amp;',
  'journey\n  <b>bold</b> <i>it</i> <script>x</script>: 3: <b>Me</b>',
  'journey\n  fa:fa-car and :fa-user: 3: fa:fa-car',
  'journey\n  one<br>two<br/>three<BR />four: 3: a<br>b',
  'journey\n  Task: 5: ' + Array.from({ length: 12 }, (_, i) => `Actor ${i}`).join(', '),
  'journey\n  Task: 5: ' + Array.from({ length: 30 }, (_, i) => `A${i}`).join(', '),
  'journey\n  A: 5: Me, Me, Me',
  'journey\n  A: 5: Zed\n  B: 4: alpha\n  C: 3: Beta\n  D: 2: _x\n  E: 1: 1st\n  F: 3: é\n  G: 3: \uff21\n  H: 3: 😀',
  'journey\n  A: 5: an actor with a really long name that will not fit on one line of the legend at all, not even close',
  'journey\n  A: 5: ' + 'Long name of an actor '.repeat(4).trim() + ', ' + 'Another long actor name '.repeat(3).trim(),
  'journey\ntitle A title that is much wider than the diagram under it, so that the diagram is centered beneath it\nA: 5: Me',
  'journey\ntitle Titled\n',
  'journey\ntitle Titled\nsection Lonely',
  'journey\naccTitle: only\naccDescr: these',
  'journey\nsection A section with a very long name over a single task card: 3\nT: 3: Me',
  'journey\nsection Short\nA: 1\nB: 2\nC: 3\nD: 4\nE: 5',
];

// The sources in Mermaid's specs and documentation as Pele's tests collect them:
// loadCorpus('journey', /journey/).
function specSources(pele) {
  return [...new Set([...testStrings(pele, 'tests/compat/journey', /\.spec\.[jt]s$/, /journey/), ...docs(pele, 'journey')])];
}

// The corpus the suites start from.
export function corpus(pele) {
  return [...new Set([...specSources(pele), ...SEEDS, ...WRITTEN])].filter((s) => !lone(s));
}

// Journey diagrams built at random, which mostly parse, with names, scores and actors of every
// kind the layout handles differently.
export function randomDiagrams(rnd) {
  const pick = picker(rnd);
  const words = ['Make', 'tea', 'Go', 'upstairs', 'Do', 'work', 'a', 'very', 'long', 'task', 'name', 'that', 'wraps',
    'Supercalifragilistic', 'Averyveryverylongwordwithoutanyspaceinit', '日本語', '中文字', '😀', 'é', '<br>', '<b>x</b>',
    'x', 'W', 'iiii', 'MMMM', '1.5', '(a)', '[b]', '"q"', '<', '&', 'a-b'];
  const phrase = (max) => {
    const out = [];
    for (let k = 1 + Math.floor(rnd() * max); k > 0; k--) out.push(pick(words));
    return out.join(pick([' ', ' ', ' ', '']));
  };
  const actors = ['Me', 'Cat', 'You', 'Alice', 'Bob', 'Charlie', 'a', 'B', 'zed', 'é', '😀', 'A very long actor name',
    'Someone with an extraordinarily long name that will wrap in the legend', '', ' ', '1', '_'];
  const score = () => pick(['1', '2', '3', '4', '5', '5', '3', '0', '6', '-1', '2.5', '4.75', 'x', '', ' ', '1e1', '3.14159', '-0', 'NaN']);
  return () => {
    let s = pick(['journey\n', 'journey\n', 'journey\n', '---\ntitle: Front\n---\njourney\n', 'JOURNEY\n', 'journey ']);
    if (rnd() < 0.6) s += `${pick(['  ', ''])}title ${phrase(rnd() < 0.2 ? 30 : 6)}\n`;
    if (rnd() < 0.2) s += `accTitle: ${phrase(3)}\n`;
    if (rnd() < 0.2) s += `accDescr: ${phrase(3)}\n`;
    const n = Math.floor(rnd() * (rnd() < 0.1 ? 25 : 9));
    for (let i = 0; i < n; i++) {
      if (rnd() < 0.3) s += `${pick(['  ', ''])}section ${phrase(rnd() < 0.2 ? 12 : 3)}\n`;
      let line = `${pick(['    ', '  ', ''])}${phrase(rnd() < 0.15 ? 15 : 4)}:${pick([' ', ''])}${score()}`;
      if (rnd() < 0.8) {
        const people = [];
        for (let k = Math.floor(rnd() * (rnd() < 0.1 ? 14 : 4)); k > 0; k--) people.push(pick(actors));
        line += `:${pick([' ', ''])}${people.join(pick([', ', ',']))}`;
      }
      s += line + '\n';
    }
    if (rnd() < 0.1) s += `section ${phrase(2)}\n`;
    return s;
  };
}

function dumpModel(m) {
  const opt = (x) => (x === undefined || x === '' ? undefined : x);
  return {
    title: opt(m.title),
    accTitle: opt(m.accTitle),
    accDescr: opt(m.accDescr),
    sections: m.sections,
    tasks: m.tasks.map((t) => ({ task: t.task, score: enc(t.score), people: t.people, section: t.section })),
  };
}

export default async function ({ src, pele, write, random }) {
  const located = await locator(src);
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { journey } = await src('diagrams/journey/index.ts');
  const { JourneyDb } = await src('diagrams/journey/db.ts');
  const { parseJourney } = await src('diagrams/journey/parser.ts');
  const { tokenize, TOKEN_NAMES } = await src('diagrams/journey/lexer.ts');

  const caught = (e) => {
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  const tokens = (source) => {
    const t = tokenize(source);
    return t.types.map((type, i) => [TOKEN_NAMES[type], t.texts[i], t.starts[i]]);
  };
  const parsed = (source, parseIt) => {
    try {
      return { tokens: tokens(source), model: dumpModel(parseIt()) };
    } catch (e) {
      return { tokens: tokens(source), ...caught(e) };
    }
  };
  // Pele's parseSource for a journey diagram, and the raw text given to the parser alone.
  const run = (source) => {
    const out = {
      src: source,
      direct: parsed(source, () => {
        const db = new JourneyDb();
        parseJourney(source, db);
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
    if (out.type !== 'journey') return out;
    const text = encodeEntities(pre.text) + '\n';
    const piped = parsed(text, () => located(source, false, () => journey.parse(text, pre.config, pre.title)));
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
  // Mutated text; one in six is cut short, as a statement cut off by the end of the text ends
  // on a different token than one cut off by a line break.
  const rnd = random(7);
  const next = mutator(base, FRAGMENTS, rnd);
  for (let i = 0; i < 4000; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  const diagrams = randomDiagrams(random(10));
  for (let i = 0; i < 1500; i++) add(diagrams());
  // Random strings of tokens, where keyword boundaries and the lexer's lookahead matter.
  const trnd = random(8);
  const pieces = [...new Set([...FRAGMENTS, ...FRAGMENTS.map((f) => f.trim()).filter((f) => f !== '')])];
  const openers = ['journey\n', 'journey\nTask', 'journey\naccDescr', 'journey\nsection', 'journey\ntitle', ''];
  for (let i = 0; i < 1500; i++) {
    let s = openers[Math.floor(trnd() * openers.length)];
    for (let k = 1 + Math.floor(trnd() * 8); k > 0; k--) s += pieces[Math.floor(trnd() * pieces.length)] + (trnd() < 0.3 ? ' ' : '');
    add(s);
  }

  write('diagram/journey/testdata/pele/journey.json', { cases: all.map(run) });
}
