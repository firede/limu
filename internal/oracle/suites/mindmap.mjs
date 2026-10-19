// Mindmaps parsed as Pele parses them, from text to tokens and to MindmapModel: Mermaid's
// documentation and specs, the sources of Pele's tests, its hostile templates filled with its
// payloads, cases written here, those mutated at random, maps built at random, and random
// strings of tokens. Each case is tokenized and parsed as the raw text, and parsed as Pele's
// parse does it, after the preprocessing every diagram goes through.
import { join } from 'node:path';
import { peleError } from '../lib/value.mjs';
import { fileLiterals, testFiles, docs, arrayIn, lone, picker } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';

// Pele's fragments, tests/mindmap/oracle.test.ts, and more for decorations, config and what
// is drawn.
export const FRAGMENTS = [
  ' ', '  ', '    ', '\t', '\n', '\n\n', ' \n', '\n ', '\r\n', '\r', ' ', ' ', 'mindmap', 'MindMap', 'mindmap\n',
  'mindmapx', '%%', ' %% c', '%%{', ':::', ':::a b', '::icon(', '::ICON(fa fa-x)', '::', ':', '(', ')', '((', '))', '[',
  ']', '{{', '}}', '{', '}', '(-', '-)', '-', '"', '`', '"`', '`"', '""', '"a"', '["a"]', '["`a`"]', '(a)', '((a))',
  '))a((', ')a(', '{{a}}', '[a]', '(a]', '[a)', 'id', 'root', 'A', 'é', '🤓', '@', '@{', '^', '<br/>', '**b**', '#',
  ';', ',', '.',
  '\n  ::icon(fa fa-book)', '\n    ::icon(star)', '\n  :::urgent large', '::icon()', ':::', '\n      leaf', '\n    branch\n      leaf',
  '#35;', '&amp;', '<b>x</b>', '_i_', '中文字', 'fa:fa-car', 'fa:fa-car x', 'a long label that wraps past the width a node has, and more',
  'x'.repeat(40), '---\ntitle: T\n---\n', '---\nconfig:\n  mindmap:\n    maxNodeWidth: 80\n---\n',
  '%%{init: {"mindmap": {"maxNodeWidth": 120}}}%%\n', '「中文」', '😀', '　', ' ',
];

const config = (lines, body) => `---\nconfig:\n  mindmap:\n${lines.map((l) => '    ' + l).join('\n')}\n---\n${body}`;
const TOOLS =
  'mindmap\n  root((mindmap))\n    Origins\n      Long history\n      ::icon(fa fa-book)\n      Popularisation\n        British popular psychology author Tony Buzan\n' +
  '    Research\n      On effectiveness<br/>and features\n      On Automatic creation\n        Uses\n            Creative techniques\n            Strategic planning\n            Argument mapping\n' +
  '    Tools\n      Pen and paper\n      Mermaid';
const LONG = 'mindmap\n  root[' + 'several words that go on '.repeat(4) + ']\n    a\n    b';

// Sources from Pele's render tests, and cases written here.
export const WRITTEN = [
  TOOLS,
  LONG,
  config(['maxNodeWidth: 80'], LONG),
  config(['maxNodeWidth: 40'], LONG),
  config(['maxNodeWidth: 1'], LONG),
  config(['maxNodeWidth: 0'], LONG),
  config(['maxNodeWidth: -5'], LONG),
  config(['maxNodeWidth: 3000'], LONG),
  config(['maxNodeWidth: 150.5'], LONG),
  config(['maxNodeWidth: "80"'], LONG),
  config(['maxNodeWidth: .nan'], LONG),
  config(['maxNodeWidth: .inf'], LONG),
  config(['maxNodeWidth: [80]'], LONG),
  config(['maxNodeWidth: null'], LONG),
  config(['padding: 40', 'useMaxWidth: false'], LONG),
  '---\nconfig:\n  mindmap: 5\n---\n' + LONG,
  '%%{init: {"mindmap": {"maxNodeWidth": 100}}}%%\n' + LONG,
  '%%{init: {"mindmap": {"maxNodeWidth": 100}}}%%\n' + config(['maxNodeWidth: 300'], LONG),
  'mindmap\n  root((Center))\n    A\n      A1\n      A2\n    B',
  'mindmap\n  root\n    a[Square]\n    b(Rounded)\n    c((Circle))\n    d))Bang((\n    e)Cloud(\n    f{{Hexagon}}\n    Plain',
  'mindmap\n  root\n    a[x)\n    b(x]\n    c(-x-)\n    -)x(-',
  'mindmap\n  [root]\n    [a]\n      [a1]\n    [b]\n    [c]',
  'mindmap\n  root\n    a\n      b',
  'mindmap\n  root\n' + Array.from({ length: 13 }, (_, i) => `    b${i}\n      leaf${i}\n`).join(''),
  'mindmap\n  [root]\n    [a]\n      [a1]\n      [a2]\n      [a3]\n    [b]\n    [c]\n    [d]',
  'mindmap\n  [root]\n    [a]\n      [b]',
  'mindmap\n  [root]\n    [Build]\n      [Prototype]\n      [Iterate]\n    [Review]\n      [Test]\n      [Refine]',
  'mindmap\n  [root]\n    [Build]\n      [Prototype]\n      [Iterate]\n    [Review]\n      [Test]\n      [Refine]\n    [Ship]\n      [Launch]\n      [Learn]',
  'mindmap\nRoot\n    A\n        B\n      C\n   D',
  'mindmap\n  root\n  other',
  'mindmap\n    root\n  shallower',
  'mindmap\n::icon(fa fa-book)\n:::big\nroot',
  'mindmap\n  root\n    A\n    ::icon(fa fa-book)\n    :::urgent large\n    B',
  'mindmap\n  root\n    Read\n    ::icon(fa fa-book)\n    Write',
  'mindmap\n  root\n  ::icon(star)\n    a((b))\n    ::icon(star)\n    c\n    ::icon(phone)\n    :::x',
  'mindmap\n  root((r))\n  ::icon(star)',
  'mindmap\n  root\n    a[ ]\n    ::icon(star)',
  'mindmap\n  root\n    a\n    ::icon(unknown)',
  'mindmap\n  id1["`**Root** with\na second line`"]\n    id2[One<br/>Two]\n    id3[*emphasis*]',
  'mindmap\n  root(Root) %% note\n    a',
  'mindmap\n  root %% note\n    a',
  '---\ntitle: Plans\n---\nmindmap\n  root\n    A',
  '---\ntitle: A title wider than the map itself, by far, much much wider than it\n---\nmindmap\n  root',
  'mindmap\n\n',
  'mindmap',
  '',
  'mindmap\nroot',
  'mindmap\n  root\n    Plan\n      Research\n        Read\n        Ask\n      Build\n    Ship',
  'mindmap\n  root((Plan))\n    Research\n      Read the papers on the subject\n      Interview people\n    Build\n      Prototype\n        First version of the layout\n      Test\n    Ship\n      Write the announcement',
  'mindmap\n  root\n' + Array.from({ length: 7 }, (_, i) => `${' '.repeat(4 + 2 * i)}Level ${i} with a fairly long label to wrap`).join('\n'),
  'mindmap\n  root[Box root]\n    a\n      b\n    c',
  'mindmap\n  root(Rounded root)\n    a\n      b\n    c',
  'mindmap\n  root{{Hex root}}\n    a\n    b',
  'mindmap\n  root))Bang root((\n    a\n    b',
  'mindmap\n  root)Cloud root(\n    a\n    b',
  'mindmap\n  root((Circle root))\n    a\n    b',
  'mindmap\n  root\n    a((x))\n      b))y((\n        c)z(\n          d{{w}}\n            e[v]\n              f(u)',
  'mindmap\n\tt\n\t\ta',
  'mindmap\n　t\n　　a',
  'mindmap\r\n  t\r\n    a\r\n',
  'MINDMAP\n  t\n',
  '  %% before\n\nmindmap\n  t',
  'mindmap\n  t[x}y]\n    a[b}\nc]',
  'mindmap\n  t\n    a\n    ::icon()\n    :::\n',
  'mindmap\n  fa:fa-car Cars\n    fa:fa-car drive fa:fa-bus',
  'mindmap\n  root\n    a["quoted [x] (y)"]\n    b("`_md_ **b**`")',
  'mindmap\n  root\n    ' + 'x'.repeat(120),
  'mindmap\n  root\n    a[' + 'x'.repeat(120) + ']',
  'mindmap\n  root\n    a[中文的节点标题很长很长很长很长很长很长很长很长很长很长很长]',
  'mindmap\n  root\n    a[a&lt;b &amp; #35; #quot;]',
  'mindmap\n  root\n    a\n      b\n        c\n          d\n            e\n              f\n                g',
  'mindmap\n  root\n    a\n    b\n    c\n    d\n    e\n    f\n    g\n    h\n    i\n    j\n    k\n    l\n    m',
  'mindmap\n  root\n    a\n      a1\n      a2\n      a3\n      a4\n      a5\n    b\n    c\n      c1',
  'mindmap\n  root\n    big\n      x\n      x\n      x\n      x\n      x\n      x\n      x\n      x\n    small',
  'mindmap\n  root\n    a\n      b\n    c\n      d\n    e',
  'mindmap\n  root\n    ((a))\n    [b]\n    (c)',
  'mindmap\n  root\n    a\n   b\n  c',
  'mindmap\n root\n  a\n b',
];

// Pele's hostile templates, tests/render/mindmap-security.test.ts, filled with its payloads.
function hostile(pele) {
  const payloads = arrayIn(join(pele, 'tests/support/inert.ts'), 'export const PAYLOADS = [');
  const templates = arrayIn(join(pele, 'tests/render/mindmap-security.test.ts'), 'const TEMPLATES: ((p: string) => string)[] = [');
  const out = [];
  for (const t of templates) for (const p of payloads) out.push(t(p));
  return out;
}

// The corpus the suites start from: documentation, specs, Pele's tests and the cases above.
export function corpus(pele) {
  const out = new Set();
  for (const file of testFiles(pele, 'tests/compat/mindmap', /\.(spec|test)\.[jt]s$/)) for (const s of fileLiterals(file, /mindmap/)) out.add(s);
  for (const s of docs(pele, 'mindmap')) out.add(s);
  for (const file of ['tests/render/mindmap.test.ts', 'tests/render/max-width.test.ts', 'tests/render/all-types.test.ts']) {
    for (const s of fileLiterals(join(pele, file), /^mindmap/)) out.add(s);
  }
  for (const s of WRITTEN) out.add(s);
  return [...out].filter((s) => !lone(s) && s.length <= 4000);
}

export function hostileCorpus(pele) {
  return hostile(pele).filter((s) => !lone(s));
}

// Maps built at random: a root and branches of every shape, with icons, classes and labels of
// every kind, mostly well formed.
export function maps(rnd) {
  const pick = picker(rnd);
  const chance = (p) => rnd() < p;
  const texts = [
    'Origins', 'Long history', 'Research', 'On effectiveness<br/>and features', 'Tools', 'Pen and paper', 'Mermaid', 'Plan', 'Build',
    'Ship', 'A', 'x', 'Creative techniques', 'Strategic planning and other long things that wrap', '**Bold** move', '_slanted_ idea',
    'Release v2 &amp; notes', '中文的节点标题', 'x'.repeat(30), 'supercalifragilisticexpialidocious-and-more', 'fa:fa-car Drive', '😀 emoji',
    'tab\there', 'a<br>b<br>c', 'one two three four five six seven eight nine ten eleven twelve', '#35; #quot;',
  ];
  const icons = ['fa fa-book', 'star', 'fa:fa-car', 'phone', 'unknown', 'mdi:home'];
  const shapes = [
    (t) => t.replace(/[[\](){}"]/g, ''),
    (t) => `[${t.replace(/[[\]()"]/g, '')}]`,
    (t) => `(${t.replace(/[[\]()"]/g, '')})`,
    (t) => `((${t.replace(/[[\]()"]/g, '')}))`,
    (t) => `))${t.replace(/[[\]()"]/g, '')}((`,
    (t) => `)${t.replace(/[[\]()"]/g, '')}(`,
    (t) => `{{${t.replace(/[[\](){}"]/g, '')}}}`,
    (t) => `["${t.replace(/"/g, '')}"]`,
    (t) => `["\`${t.replace(/[`"]/g, '')}\`"]`,
  ];
  let id = 0;
  const node = (text, plain) => {
    const shape = plain ? 0 : chance(0.5) ? 0 : Math.floor(rnd() * shapes.length);
    const body = shapes[shape](text) || 'x';
    if (shape === 0) return body.trim() === '' ? `n${id++}` : body;
    return (chance(0.7) ? `n${id++}` : '') + body;
  };
  return () => {
    id = 0;
    const lines = [];
    let head = '';
    if (chance(0.12)) head += `---\ntitle: ${pick(['Plans', 'A map with a title wider than the map itself, by far, much wider', 'T', '"#35; <b>x</b>"'])}\n---\n`;
    if (chance(0.2)) {
      const width = pick([40, 72, 80, 100, 120, 150.5, 200, 260, 400, 3000, 0, -5, '"x"']);
      if (chance(0.5)) head += `---\nconfig:\n  mindmap:\n    maxNodeWidth: ${width}\n---\n`;
      else head += `%%{init: {"mindmap": {"maxNodeWidth": ${width}}}}%%\n`;
    }
    lines.push('mindmap');
    const indent = pick(['  ', '    ', '\t', ' ']);
    lines.push(indent + node(pick(texts), false));
    if (chance(0.1)) lines.push(indent + `::icon(${pick(icons)})`);
    const branches = Math.floor(rnd() * (chance(0.15) ? 14 : 6));
    const grow = (depth, budget) => {
      const kids = depth > 6 ? 0 : Math.floor(rnd() * budget);
      for (let k = 0; k < kids; k++) {
        lines.push(indent.repeat(depth) + node(pick(texts), false));
        if (chance(0.1)) lines.push(indent.repeat(depth) + `::icon(${pick(icons)})`);
        if (chance(0.08)) lines.push(indent.repeat(depth) + `:::${pick(['urgent', 'urgent large', 'bad"class', 'a-b_c'])}`);
        if (chance(0.03)) lines.push(indent.repeat(depth) + '%% a comment');
        if (chance(0.5)) grow(depth + 1, Math.max(1, budget - 1));
      }
    };
    for (let b = 0; b < branches; b++) {
      lines.push(indent.repeat(2) + node(pick(texts), false));
      if (chance(0.1)) lines.push(indent.repeat(2) + `::icon(${pick(icons)})`);
      if (chance(0.06)) lines.push(indent.repeat(2) + `:::${pick(['urgent', 'big'])}`);
      grow(3, 5);
    }
    if (chance(0.03)) lines.push(node('second root', true));
    if (chance(0.03)) lines.splice(2, 0, indent.repeat(1 + Math.floor(rnd() * 3)) + node('odd', false));
    return head + lines.join('\n') + (chance(0.5) ? '\n' : '');
  };
}

function dumpModel(m) {
  return {
    title: m.title || undefined,
    nodes: m.nodes.map((n) => ({
      id: n.id,
      nodeId: n.nodeId,
      level: n.level,
      descr: n.descr,
      type: n.type,
      children: n.children.map((c) => c.id),
      isRoot: n.isRoot,
      section: n.section,
      class: n.class,
      icon: n.icon,
    })),
  };
}

export default async function ({ src, pele, write, random }) {
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { mindmap } = await src('diagrams/mindmap/index.ts');
  const { MindmapDb } = await src('diagrams/mindmap/db.ts');
  const { parseMindmap, tokenize } = await src('diagrams/mindmap/parser.ts');

  const caught = (e) => {
    if (e instanceof RangeError) return { error: { code: 'limit', message: 'Diagram is nested too deeply.' } };
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
      tokens: { types: t.types, texts: t.texts, starts: t.starts },
      direct: parsed(() => {
        const db = new MindmapDb();
        parseMindmap(source, db);
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
    if (out.type !== 'mindmap') return out;
    const piped = parsed(() => mindmap.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title));
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
  for (const s of hostileCorpus(pele)) add(s);
  // Mutated text; one in six is cut short.
  const rnd = random(51);
  const next = mutator(base, FRAGMENTS, rnd);
  for (let i = 0; i < 3000; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  const map = maps(random(52));
  for (let i = 0; i < 1000; i++) add(map());
  // Random strings of tokens, where the lexer's states and the parser's lookahead matter.
  const trnd = random(53);
  const pieces = [...new Set([...FRAGMENTS, ...FRAGMENTS.map((f) => f.trim()).filter((f) => f !== '')])];
  const openers = ['mindmap\n', 'mindmap\n  root\n    ', 'mindmap\n  t[', 'mindmap\n  t\n    a["`', '', '%% c\n', 'mindmap\n  root\n    a\n    ::icon('];
  for (let i = 0; i < 1000; i++) {
    let s = openers[Math.floor(trnd() * openers.length)];
    for (let k = 1 + Math.floor(trnd() * 10); k > 0; k--) s += pieces[Math.floor(trnd() * pieces.length)] + (trnd() < 0.3 ? ' ' : '');
    add(s);
  }

  write('diagram/mindmap/testdata/pele/mindmap.json', { cases: all.map(run) });
}
