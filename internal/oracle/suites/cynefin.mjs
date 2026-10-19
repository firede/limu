// Cynefin diagrams parsed as Pele parses them: the tokens Pele's common/tokens.ts makes of the
// text with the diagram's table, the tree parseCynefin makes of them, and the model
// buildCynefin makes of the tree with the title the text's front matter gives. The cases are
// Mermaid's documentation and specs, the inputs of Pele's own tests, cases written here,
// random diagrams, and all of those mutated at random.
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { enc, lone, mutator, peleError } from './parse.mjs';

// The fragments of Pele's tests/cynefin/oracle.test.ts, and more.
export const FRAGMENTS = [
  'cynefin-beta', 'cynefin-beta:', 'cynefin-beta\n', 'title', 'title ', 'accTitle: ', 'accDescr: ', 'accDescr {', '}', ':', ' : ',
  '-->', ' --> ', '--', '->', '>', '"', "'", '"a"', "'b'", '"a\\"b"', '\\', '\\n', 'complex', 'complicated', 'clear', 'chaotic',
  'confusion', 'compl', 'complexity', 'Clear', '\n', '\r\n', '\r', ' ', '\t', '%%', '%% c', '%%{init: {}}%%', '---', '---\n', 'x',
  'é', ';', ',', '#35;', 'complex --> clear', 'clear --> clear : "x"', '\n"item"', '\nchaotic\n', '1',
  ' ', '　', '😀', '"中文"', '"&amp;"', '"<b>b</b>"', '"fa:fa-car"', '"a<br>b"', '\n  "an item with a few words"', ' : ""',
  'confusion --> complex', '\nconfusion\n"a"\n"b"\n"c"\n"d"', 'title T', '\naccTitle: A\n', '\naccDescr {\n a\n b\n}\n',
];

// Seeds of Pele's tests/cynefin/oracle.test.ts, which its reference accepts.
const SEEDS = [
  'cynefin-beta',
  'cynefin-beta:',
  'cynefin-beta:\ncomplex\n"a"',
  '\n\n  cynefin-beta\n',
  'cynefin-beta title T\n',
  'cynefin-beta complex "a" "b" clear\n"c"',
  'cynefin-beta\ncomplex --> clear\nclear --> chaotic : "x"\nchaotic --> chaotic',
  'cynefin-beta\naccTitle: at\naccDescr {\n multi\n line\n}\ntitle t\ncomplex\n\n\n"a"\n\n"b"\n',
  'cynefin-beta\ncomplex\n"a"\ncomplex\n"b"\ncomplex --> complicated : \'single\'\n',
  '---\ntitle: x\n---\ncynefin-beta\nclear\n"a"',
  '%%{init: {"theme":"dark"}}%%\ncynefin-beta %% c\nclear %% d\n"a" %% e\n',
  'cynefin-beta\r\ncomplex\r\n"a"\r\n',
];

const C = 'cynefin-beta\n';
const config = (lines, body) => `---\nconfig:\n  cynefin:\n${lines.map((l) => '    ' + l).join('\n')}\n---\n${body}`;

export const BASIC =
  'cynefin-beta\n  title Incident Response\n  complex\n    "Investigate root cause"\n    "Run chaos experiment"\n  complicated\n    "Expert review needed"\n' +
  '  clear\n    "Restart service"\n  chaotic\n    "Page on-call immediately"\n  confusion\n    "Unknown failure mode"\n';

// Cases written here: Pele's render tests, and what the tokens, the tree, the model and the
// drawing do at their edges.
export const WRITTEN = [
  BASIC,
  'cynefin-beta\ncomplex --> complicated : "Pattern identified"\nclear --> chaotic\nclear --> clear : "ignored"\ncomplex --> clear : "diagonal"\nconfusion --> chaotic\n',
  'cynefin-beta\nconfusion\n"a"\n"b"\n"c"\n"d"\n"e"\n',
  'cynefin-beta\nconfusion\n"a"\n"b"\n"c"\n',
  'cynefin-beta\nconfusion\n"a"\n"b"\n"c"\n"d"\n',
  'cynefin-beta\ncomplex\n"first"\ncomplex\n"second"\n',
  'cynefin-beta\nclear\n"x"\ncomplex\n"first"\nclear\ncomplex\n"second"\n"third"\n',
  'cynefin-beta\ncomplex\n"a"\n',
  'cynefin-beta\ncomplex\n' + '"an item"\n'.repeat(20),
  'cynefin-beta\ncomplex\n"' + 'word '.repeat(30) + '"\n',
  config(['showDomainDescriptions: false'], BASIC),
  config(['width: 1200', 'height: 900'], BASIC),
  config(['boundaryAmplitude: 0'], BASIC),
  config(['boundaryAmplitude: 50'], BASIC),
  config(['boundaryAmplitude: 51'], BASIC),
  config(['boundaryAmplitude: -1'], BASIC),
  config(['seed: 5'], BASIC),
  config(['seed: 6'], BASIC),
  config(['seed: 0'], BASIC),
  config(['seed: -7'], BASIC),
  config(['seed: 2.5'], BASIC),
  config(['seed: -2.75'], BASIC),
  config(['seed: 4294967296'], BASIC),
  config(['seed: 2147483647'], BASIC),
  config(['seed: -2147483649'], BASIC),
  config(['seed: 1000000000000000'], BASIC),
  config(['seed: 1000000000000001'], BASIC),
  config(['seed: -999999999999999.5'], BASIC),
  config(['seed: "5"'], BASIC),
  config(['width: 100', 'height: 100'], BASIC),
  config(['width: 99', 'height: 10001'], BASIC),
  config(['width: 10000', 'height: 10000'], C + 'complex\n"a"\n'),
  config(['width: 333.3', 'height: 251.7'], BASIC),
  config(['showDomainDescriptions: "false"'], BASIC),
  config(['showDomainDescriptions: 0'], BASIC),
  '%%{init: {"cynefin": {"seed": 42, "boundaryAmplitude": 20}}}%%\n' + BASIC,
  '%%{init: {"cynefin": {"width": 300, "height": 200, "showDomainDescriptions": false}}}%%\n' + BASIC,
  '---\ntitle: Front\n---\ncynefin-beta\naccTitle: Acc\naccDescr: Described\nclear\n"a"\n',
  '---\ntitle: Front\n---\ncynefin-beta\ntitle Own\n',
  '---\ntitle: A front matter title long enough to be wider than the whole diagram is drawn, by a good way, and then some more words still\n---\ncynefin-beta\n',
  C + 'title ' + 'A very long title '.repeat(12) + '\n',
  'cynefin-beta\n"orphan"',
  'cynefin-beta\ncomplexity',
  'cynefin-beta\ncomplex --> nowhere',
  'cynefin-beta\ncomplex --> clear clear',
  'cynefin-betax',
  'cynefin-beta:x',
  'cynefin-beta: complex\n"a"',
  'cynefin',
  'cynefin-beta\ncynefin-beta',
  C + 'complex -->',
  C + 'complex --> ',
  C + 'complex --> clear :',
  C + 'complex --> clear : x',
  C + 'complex --> clear : "x" "y"',
  C + 'complex --> clear : ""',
  C + 'complex --> clear : \'\'',
  C + 'complex-->clear:"tight"',
  C + 'complex --> clear\ncomplex --> clear\ncomplex --> clear : "third"\nclear --> complex\nclear --> complex : "fifth"\n',
  C + 'complex --> confusion\nconfusion --> complex\ncomplicated --> confusion : "in"\nconfusion --> clear : "out"\nchaotic --> confusion\nconfusion --> chaotic : "x"\n',
  C + 'complex --> chaotic : "down"\nchaotic --> complex : "up"\ncomplicated --> clear : "down"\nclear --> complicated : "up"\n',
  C + 'complex --> clear : "diag"\nclear --> complex : "back"\ncomplicated --> chaotic : "other"\nchaotic --> complicated\n',
  C + 'complex --> complicated : "' + 'a long label that wraps '.repeat(8) + '"\n',
  C + 'chaotic --> clear : "' + 'x'.repeat(300) + '"\n',
  C + 'complex --> complicated : "fa:fa-car Car"\nclear --> chaotic : "<b>bold</b> and <i>it</i>"\n',
  C + 'complex --> clear "x"',
  C + 'complex "a"',
  C + 'complex\n  "a" "b"\n  "c"\nclear "d"\n',
  C + 'complex\n"a\\"b"\n"c\\\\d"\n"e\\nf"\n\'g\\\'h\'\n"\\t\\b\\f\\v\\0\\r"\n',
  C + 'complex\n"a\nb"\n',
  C + 'complex\n"a',
  C + 'complex\n"a\\',
  C + 'complex\n""\n\'\'\n"  "\n',
  C + 'complex\n"#35; &amp; &lt;b&gt;"\n"<b>b</b> <i>i</i> <br> next"\n"fa:fa-car car"\n"中文的项目名称"\n"😀 emoji"\n',
  C + 'complicated\n"' + 'long '.repeat(80) + '"\n',
  C + 'clear\n"' + 'a'.repeat(400) + '"\n',
  C + 'chaotic\n"x"\nconfusion\n"' + 'wide confusion item '.repeat(10) + '"\n',
  C + 'confusion\n' + '"item"\n'.repeat(3) + '"' + 'w'.repeat(200) + '"\n',
  C + 'complex\n' + '"tall"\n'.repeat(40) + 'confusion\n"a"\n"b"\n"c"\n',
  C + 'complicated\n' + '"x"\n'.repeat(12) + 'clear\n' + '"y"\n'.repeat(3),
  C + 'clear\n' + '"wide item in clear that is long"\n'.repeat(6),
  C + 'title\n',
  C + 'title',
  C + 'title T %% c\n',
  C + 'titleT\n',
  C + 'title  a   b  \n',
  C + 'accTitle: A\naccTitle:   B   \n',
  C + 'accTitle A\n',
  C + 'accDescr {\n a\n   b\n}\n',
  C + 'accDescr {\n a\n',
  C + 'accDescr: one\naccDescr {\n two\n}\n',
  C + 'accDescr :one',
  C + 'accDescr {} complex',
  C + 'title T complex',
  C + '%% c\n%%{init: {}}%%\ncomplex %% d\n"a" %% e\n',
  C + 'complex %%{x}%%\n',
  C + '---\n---\n',
  C + '---\n',
  C + '--- x\n',
  C + '-',
  C + '->',
  C + '-->',
  C + ':',
  C + 'Complex',
  C + 'compl',
  C + 'complexcomplex',
  C + 'complex\tclear',
  C + 'complex:clear',
  C + ' complex\n  "a"\n',
  C + 'complex\n"a"　\n',
  C + 'complex\n"a"\r"b"',
  C + 'complex\r\n"a"\r\nclear --> complex\r\n',
  '  cynefin-beta  \n  complex  \n',
  '\tcynefin-beta\n',
  'cynefin-beta %% c\n',
  'cynefin-beta\n\n\n',
  'cynefin-beta\ncomplex\n"a"\n\n\n\nclear\n\n"b"\n\n',
  '',
  'not a diagram',
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
      if (!/cynefin-beta/.test(raw)) continue;
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
  const docs = JSON.parse(readFileSync(join(pele, 'tests/corpus/cynefin-docs.json'), 'utf8'));
  return [
    ...new Set([
      ...docs,
      ...testStrings(pele, 'tests/compat/cynefin', /\.spec\.[jt]s$/),
      ...testStrings(pele, 'tests/cynefin', /\.test\.[jt]s$/),
      ...testStrings(pele, 'tests/render', /^cynefin.*\.test\.ts$/),
      ...SEEDS,
      ...WRITTEN,
    ]),
  ].filter((s) => !lone(s) && s.length <= 50000);
}

// Pele's stress cases, made smaller: what makes a pattern scan again, or a diagram large.
export function hostile() {
  const N = 4000;
  const repeat = (count, line) => Array.from({ length: count }, (_, i) => line(i)).join('');
  const names = ['complex', 'clear', 'chaotic', 'confusion', 'complicated'];
  return [
    C + 'complex\n' + repeat(300, (i) => `"i${i}"\n`),
    C + 'confusion\n' + '"x"\n'.repeat(N / 4),
    C + 'clear\n"a"\n'.repeat(N / 10),
    C + repeat(200, (i) => `complex --> clear : "t${i}"\n`),
    C + repeat(200, (i) => names[i % 5] + ' --> ' + ['clear', 'chaotic', 'confusion', 'complicated', 'complex'][(i + (i % 3)) % 5] + '\n'),
    C + 'complex\n"' + 'word '.repeat(N / 5) + '"',
    C + 'complex\n"' + 'a'.repeat(N) + '"',
    C + 'complex --> clear : "' + 'word '.repeat(N / 5) + '"',
    C + 'title ' + 'x'.repeat(N),
    C + 'complex\n"' + 'a'.repeat(N),
    C + 'complex\n"' + '\\'.repeat(N),
    C + 'complex\n"' + '\\"'.repeat(N / 2) + '"',
    C + 'complex\n' + '"a\n'.repeat(N / 3),
    C + 'accDescr {' + ' a\n'.repeat(N / 3),
    C + '%%{\n'.repeat(N / 4),
    C + 'complex\n---\n'.repeat(N / 12),
    C + '\n'.repeat(N),
    C + 'complex\n' + ' '.repeat(N) + '"a"',
    C + '%% c\n'.repeat(N / 5),
    C + 'complex\n"' + '#35;&amp;<br>'.repeat(N / 13) + '"',
    C + 'complex'.repeat(N / 7),
    C + ('accTitle' + ' '.repeat(40)).repeat(N / 48),
    C + '---\n'.repeat(N / 4),
  ];
}

const ITEMS = [
  'Investigate root cause', 'Run chaos experiment', 'Analyze performance data', 'Expert review needed', 'Restart service',
  'Apply known fix', 'Page on-call immediately', 'Unknown failure mode', 'Retrospectives', 'Pair programming', 'Code review', 'A',
  'x', 'Deployment checklist', 'New initiative', 'Market research', 'A rather long item that will need to wrap onto more lines',
  '中文项目', '😀 emoji', '#35; &amp;', '<b>bold</b> item', 'line<br>break', 'fa:fa-car Car', 'fa:fa-coffee', 'Data (raw)', '',
  "it's", 'a\\"b', 'Kettle', 'Search', 'Crisis management',
];
const LABELS = ['Pattern identified', 'Best practice codified', 'Complacency', 'Stabilized', 'x', '', 'A longer label for the transition', 'fa:fa-star', '中文'];

// Random Cynefin diagrams, most of them well formed, with random configuration.
export function randomDiagrams(rnd) {
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  const chance = (p) => rnd() < p;
  const names = ['complex', 'complicated', 'clear', 'chaotic', 'confusion'];
  const quote = (s) => (s.includes('"') && !s.includes("'") ? `'${s}'` : `"${s}"`);
  return () => {
    const settings = [];
    if (chance(0.4)) {
      if (chance(0.4)) settings.push(`width: ${pick([100, 200, 300, 500, 640, 800, 1200, 99, 10001, 333.5, '"x"'])}`);
      if (chance(0.4)) settings.push(`height: ${pick([100, 200, 300, 440, 600, 900, 50, 20000, 251.25, '"x"'])}`);
      if (chance(0.3)) settings.push(`showDomainDescriptions: ${pick(['true', 'false', '"false"', '0'])}`);
      if (chance(0.4)) settings.push(`boundaryAmplitude: ${pick([0, 1, 4, 8, 12.5, 20, 50, 60, -2, '"8"'])}`);
      if (chance(0.5)) {
        const seed = chance(0.5) ? Math.floor(rnd() * 2000) - 1000 : pick([0, 1, -1, 2.5, 0.1, 42, 2147483647, 2147483648, -2147483648, 4294967295, 1e15, -1e15, 1e16, 123456789012.75]);
        settings.push(`seed: ${seed}`);
      }
    }
    let s = '';
    if (chance(0.1)) s += `---\ntitle: ${pick(['Front', 'A front matter title', '中文'])}\n---\n`;
    if (settings.length) {
      if (chance(0.7)) s += `---\nconfig:\n  cynefin:\n${settings.map((l) => '    ' + l).join('\n')}\n---\n`;
      else s += `%%{init: {"cynefin": {${settings.map((l) => l.replace(/^(\w+): (.*)$/, (_, k, v) => `"${k}": ${v}`)).join(', ')}}}}%%\n`;
    }
    s += pick(['cynefin-beta\n', 'cynefin-beta\n', 'cynefin-beta:\n', '  cynefin-beta\n']);
    const indent = pick(['', '  ', '\t']);
    if (chance(0.5)) s += `${indent}title ${pick(['Incident Response', 'Cynefin', 'A rather long title for a Cynefin diagram', '#35; &amp;'])}\n`;
    if (chance(0.1)) s += `${indent}accTitle: Accessible\n`;
    if (chance(0.1)) s += chance(0.5) ? `${indent}accDescr: Described\n` : `${indent}accDescr {\n  one\n   two\n}\n`;
    for (let k = Math.floor(rnd() * 7); k > 0; k--) {
      s += `${indent}${pick(names)}\n`;
      for (let n = Math.floor(rnd() * (chance(0.15) ? 9 : 4)); n > 0; n--) s += `${indent}  ${quote(pick(ITEMS))}\n`;
      if (chance(0.2)) s += '\n';
    }
    for (let k = chance(0.6) ? Math.floor(rnd() * 7) : 0; k > 0; k--) {
      const from = pick(names);
      const to = chance(0.1) ? from : pick(names);
      s += `${indent}${from} --> ${to}`;
      if (chance(0.6)) s += ` : ${quote(pick(LABELS))}`;
      s += '\n';
    }
    return s;
  };
}

// The tree parseCynefin makes, with every part Pele may leave undefined written so.
export function dumpAst(ast) {
  return {
    title: enc(ast.title),
    accTitle: enc(ast.accTitle),
    accDescr: enc(ast.accDescr),
    domains: ast.domains.map((d) => [d.domain, d.items.map((i) => i.label)]),
    transitions: ast.transitions.map((t) => [t.from, t.to, enc(t.label)]),
  };
}

// The model buildCynefin makes.
export function dumpModel(m) {
  return {
    title: enc(m.title || undefined),
    accTitle: enc(m.accTitle || undefined),
    accDescr: enc(m.accDescr || undefined),
    domains: [...m.domains.entries()].map(([k, d]) => [k, d.name, d.items.map((i) => i.label)]),
    transitions: m.transitions.map((t) => [t.from, t.to, enc(t.label)]),
  };
}

export default async function ({ src, pele, write, random }) {
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { cynefin } = await src('diagrams/cynefin/index.ts');
  const { parseCynefin, CYNEFIN_TOKENS } = await src('diagrams/cynefin/parser.ts');
  const { tokenize } = await src('diagrams/common/tokens.ts');

  const caught = (e) => {
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  const tokens = (source) => {
    try {
      const t = tokenize(source, CYNEFIN_TOKENS, 'cynefin');
      return { tokens: t.kinds.slice(0, -1).map((k, i) => [CYNEFIN_TOKENS[k].name, source.slice(t.starts[i], t.ends[i])]) };
    } catch (e) {
      return { tokens: null };
    }
  };
  const direct = (source) => {
    try {
      return { ...tokens(source), ast: dumpAst(parseCynefin(source)) };
    } catch (e) {
      return { ...tokens(source), ...caught(e) };
    }
  };
  // Pele's parseSource for a Cynefin diagram, and the raw text given to the parser alone.
  const run = (source) => {
    const out = { src: source, direct: direct(source) };
    let pre;
    try {
      pre = preprocess(source);
      out.type = detect(pre.text);
    } catch (e) {
      return { ...out, ...caught(e) };
    }
    if (out.type !== 'cynefin') return out;
    try {
      out.model = dumpModel(cynefin.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title));
    } catch (e) {
      Object.assign(out, caught(e));
    }
    return out;
  };

  const all = [];
  const seen = new Set();
  const add = (s) => {
    if (s.length <= 6000 && !lone(s) && !seen.has(s)) {
      seen.add(s);
      all.push(s);
    }
  };
  corpus(pele).forEach(add);
  const diagrams = randomDiagrams(random(71));
  for (let i = 0; i < 1500; i++) add(diagrams());
  // Mutated text; one in six is cut short, as a statement cut off by the end of the text ends
  // on a different error than one cut off by a line break.
  const rnd = random(72);
  const next = mutator([...seen].filter((s) => s.length <= 3000), FRAGMENTS, rnd);
  for (let i = 0; i < 3500; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  // Random strings of tokens, where the order of the token types and their ends matter.
  const trnd = random(73);
  const pieces = [...new Set([...FRAGMENTS, ...FRAGMENTS.map((f) => f.trim()).filter((f) => f !== '')])];
  const openers = ['cynefin-beta\n', 'cynefin-beta ', 'cynefin-beta:', 'cynefin-beta\ncomplex ', 'cynefin-beta\nclear --> ', 'cynefin-beta\nchaotic\n', ''];
  for (let i = 0; i < 1500; i++) {
    let s = openers[Math.floor(trnd() * openers.length)];
    for (let k = 1 + Math.floor(trnd() * 8); k > 0; k--) s += pieces[Math.floor(trnd() * pieces.length)] + (trnd() < 0.3 ? ' ' : '');
    add(s);
  }
  for (const s of hostile()) add(s);
  write('diagram/cynefin/testdata/pele/cynefin.json', { cases: all.map(run) });
}
