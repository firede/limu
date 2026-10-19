// Kanban boards parsed as Pele parses them, from text to tokens and to KanbanModel: Mermaid's
// documentation and specs, the sources of Pele's tests, its hostile templates filled with its
// payloads, cases written here, those mutated at random, boards built at random, and random
// strings of tokens. Each case is tokenized and parsed as the raw text, and parsed as Pele's
// parse does it, after the preprocessing every diagram goes through.
import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import { lone, mutator, peleError } from './parse.mjs';

// Pele's fragments, tests/kanban/oracle.test.ts, and more for metadata and what is drawn.
export const FRAGMENTS = [
  ' ', '  ', '    ', '\t', '\n', '\n\n', ' \n', '\n ', '\r\n', '\r', ' ', ' ', 'kanban', 'Kanban', 'kanban\n',
  'kanbanx', '%%', ' %% c', '%%{', ':::', ':::a b', '::icon(', '::ICON(fa fa-x)', '::', ':', '(', ')', '((', '))', '[',
  ']', '{{', '}}', '{', '}', '(-', '-)', '-', '"', '`', '"`', '`"', '""', '"a"', '["a"]', '["`a`"]', '(a)', '((a))',
  '))a((', ')a(', '{{a}}', '[a]', '(a]', '[a)', 'id', 'todo', 'A', 'é', '🤓', '@', '@{', '@{ ', ' }', '@{}', '^',
  '@{ priority: high }', '@{ label: "a\n  b" }', "ticket: MC-1, assigned: 'k'", 'a: "', '\n  icon: star\n', '#', ';', ',', '.',
  '@{ priority: \'Very High\' }', '@{ priority: Low }', '@{ priority: " very low " }', '@{ priority: [] }', '@{ icon: [] }',
  '@{ shape: Rect }', '@{ shape: rect }', '@{ shape: a_b }', '@{ ticket: 0 }', '@{ ticket: "0" }', '@{ ticket: T-1, assigned: x }',
  '@{ label: 12 }', '@{ label: [a, b] }', '@{ label: {a: 1} }', '@{ assigned: "a very long name of someone" }', '@{ - x }',
  '@{ a: [ }', '@{\n}', '\n  priority: High\n', '#35;', '&amp;', '<br>', '<b>x</b>', '**b**', '_i_', '中文字', 'fa:fa-car',
  'a long label that wraps past the width a card has, and more', 'x'.repeat(40), '---\ntitle: T\n---\n',
];

// Sources from Pele's render tests, and cases written here.
export const WRITTEN = [
  'kanban\n  todo[To do]\n    a[First]\n    b[Second]\n  done[Done]\n    c[Third]\n  later[Later]',
  'kanban\n  a\n    one\n    two\n    three\n  b\n    four\n  c',
  '---\nconfig:\n  kanban:\n    sectionWidth: 320\n---\nkanban\n  a\n    one\n  b',
  'kanban\n  root\n    child1\n      leaf1\n    child2\n  second\n      deep',
  'kanban\n    a\n  shallow',
  'kanban\n    a\n  shallow\n    b',
  "kanban\n  todo\n    a[Task]@{ ticket: MC-2037, assigned: 'knsv', priority: 'High', icon: star }",
  'kanban\n  todo\n    a@{\n      assigned: knsv\n      label: "A new\n        label"\n      ticket: 12\n    }',
  "kanban\n  todo\n    a[Task]@{ ticket: MC-1, assigned: 'knsv' }",
  "---\nconfig:\n  kanban:\n    ticketBaseUrl: 'https://example.com/browse/#TICKET#'\n---\nkanban\n  todo\n    a[Task]@{ ticket: MC-1 }\n    b[Other]@{ ticket: \"$&\" }",
  "---\nconfig:\n  kanban:\n    ticketBaseUrl: 'javascript:alert(#TICKET#)'\n---\nkanban\n  todo\n    a[Task]@{ ticket: MC-1 }",
  "---\nconfig:\n  kanban:\n    ticketBaseUrl: '#TICKET#'\n---\nkanban\n  todo\n    a[Task]@{ ticket: 'javascript:alert(1)' }",
  "kanban\n  todo\n    a[A]@{ priority: 'Very High' }\n    b[B]@{ priority: High }\n    c[C]@{ priority: Medium }\n" +
    "    d[D]@{ priority: 'Low' }\n    e[E]@{ priority: 'Very Low' }\n    f[F]@{ priority: high }\n    g[G]@{ priority: whenever }",
  'kanban\n  todo[To do]\n  :::wide\n  ::icon(inbox)\n    a[Task]@{ icon: star }\n    :::urgent large',
  'kanban\n  todo\n    a[Short]',
  'kanban\n  todo\n    a[' + 'several words that wrap '.repeat(4) + ']',
  'kanban\n  todo\n    a[' + 'x'.repeat(120) + ']',
  'kanban\n  todo\n    a[Old]@{ label: "New **bold** text" }',
  'kanban\n  todo\n    a@{ label: "x }',
  'kanban\n  todo\n    a@{ a: [ }',
  'kanban\n  todo\n    a@{\n}',
  'kanban\n  todo\n    a@{\n - x\n}',
  'kanban\n  todo\n    a@{}',
  'kanban\n  todo\n    a@{ shape: Rect }',
  'kanban\n  todo\n    a@{ shape: kanban_item }',
  'kanban\n  todo\n    a@{ shape: rect }',
  'kanban\n::icon(star)\n:::big\ntodo',
  '---\ntitle: Sprint 12\n---\nkanban\n  todo\n    a',
  'kanban\n\n',
  'kanban',
  '',
  'kanban\n' + ['Todo', 'Doing', 'Review', 'Done'].map((name, i) => `  c${i}[${name}]\n    t${i}[Task ${i}]`).join('\n'),
  'kanban\n  todo[Todo]\n    a[Call]@{ icon: phone }',
  "---\nconfig:\n  kanban:\n    ticketBaseUrl: 'URL'\n---\nkanban\n  todo\n    a[Card]@{ ticket: T1 }",
  'kanban\n  todo\n    a@{ icon: [] , priority: [], ticket: [null], assigned: [[]] }',
  'kanban\n  todo\n    a@{ label: [] }',
  'kanban\n  todo\n    a@{ label: 0 }\n    b@{ label: false, ticket: true, assigned: 1.5e3 }',
  'kanban\n  t\n    a["`**bold** and _it_\nnext`"]\n    b("quoted [x]")\n    c((circle))\n    d{{hex}}\n    e))bang((\n    f)cloud(\n    g(-x-)',
  'kanban\n  t\n    a[x]  %% comment\n  %% between\n    b',
  'kanban\r\n  t\r\n    a\r\n',
  'KANBAN\n  t\n',
  '  %% before\n\nkanban\n  t',
  'kanban\n  t\n    a@{ label: "a\n     b\n\n  c" }',
  'kanban\n  t\n    a@{ label: "x^y" }',
  'kanban\n  t\n    a@{ ^ }',
  'kanban\n  t[x}y]\n    a[b}\nc]',
  'kanban\n  t\n    a\n    ::icon()\n    :::\n',
  'kanban\n  a\n  b\n  c\n    x\n  d\n    y\n    z',
  'kanban\n\t\tt\n\t\t\ta',
  'kanban\n　t\n　　a',
  'kanban\n  t@{ ticket: T, assigned: A, priority: High, icon: i }\n    a',
  'kanban\n  t\n    a@{ ticket: "a\nb", assigned: "<b>x</b> y" }',
  'kanban\n  t\n    a@{ ticket: "*no* markdown", assigned: "**no**" }',
  'kanban\n  fa:fa-car Cars\n    fa:fa-car drive fa:fa-bus',
];

// The literals of a test file that hold a kanban board, as Pele's loadCorpus finds them.
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

// Pele's hostile templates, tests/render/kanban-security.test.ts, filled with its payloads.
function hostile(pele) {
  const payloads = arrayIn(join(pele, 'tests/support/inert.ts'), 'export const PAYLOADS = [');
  const base = "---\nconfig:\n  kanban:\n    ticketBaseUrl: 'https://example.com/browse/#TICKET#'\n---\n";
  const templates = arrayIn(join(pele, 'tests/render/kanban-security.test.ts'), 'const TEMPLATES: ((p: string) => string)[] = [', { BASE: base });
  const out = [];
  for (const t of templates) for (const p of payloads) out.push(t(p));
  out.push(`${base}kanban\n  a<b>"&[<i onload=x>"&]\n  :::a"b <c> d=e\n    c@{ ticket: '"><x>&', assigned: 'x < y "&' }`);
  return out;
}

// The corpus the suites start from: documentation, specs, Pele's tests and the cases above.
export function corpus(pele) {
  const out = new Set();
  const dir = join(pele, 'tests/compat/kanban');
  for (const file of readdirSync(dir).filter((f) => /\.spec\.[jt]s$/.test(f))) for (const s of literals(join(dir, file), /kanban/)) out.add(s);
  for (const s of JSON.parse(readFileSync(join(pele, 'tests/corpus/kanban-docs.json'), 'utf8'))) out.add(s);
  for (const s of literals(join(pele, 'tests/render/kanban.test.ts'), /kanban/)) out.add(s);
  for (const s of WRITTEN) out.add(s);
  return [...out].filter((s) => !lone(s) && s.length <= 4000);
}

export function hostileCorpus(pele) {
  return hostile(pele).filter((s) => !lone(s));
}

// Boards built at random: columns and cards, with every kind of metadata, decoration and
// label, mostly well formed.
export function boards(rnd) {
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  const chance = (p) => rnd() < p;
  const words = ['Todo', 'In progress', 'Done', 'Review', 'Blocked', 'Ready for deploy', 'Backlog', "Can't reproduce", 'QA'];
  const texts = [
    'Fix the login', 'Write docs', 'Design grammar', 'Create renderer so that it works in all cases. We also add some extra text here',
    'Update DB function', '**Bold** move', '_slanted_ idea', 'a<br>b', 'Release v2 &amp; notes', '中文的卡片标题很长很长很长很长很长很长',
    'x'.repeat(30), 'supercalifragilisticexpialidocious-and-more', 'fa:fa-car Drive', 'short', 'A', '😀 emoji', 'tab\there',
    '"`**md** in a\nstring`"',
  ];
  const tickets = ['MC-2037', 'T-1', '12', 'JIRA-123456789-LONG-TICKET', '"a b c"', "'#5'", 'x/y?z=1&q'];
  const people = ['knsv', "'K.Sveidqvist'", 'Ann', '"Somebody with a long name"', 'Bo B'];
  const priorities = ["'Very High'", 'High', 'Medium', "'Low'", "'Very Low'", 'high', '" low "', 'whenever', 'VERY HIGH'];
  const icons = ['star', 'inbox', 'fa:fa-car', 'phone', 'unknown'];
  let id = 0;
  const node = (label) => {
    const n = `n${id++}`;
    switch (Math.floor(rnd() * 6)) {
      case 0:
        return label.startsWith('"') ? `${n}[${label}]` : `${n}[${label.replace(/[[\]()]/g, '')}]`;
      case 1:
        return label.startsWith('"') ? `${n}[${label}]` : `${n}["${label.replace(/"/g, '')}"]`;
      case 2:
        return `${n}`;
      case 3:
        return `[${label.replace(/[[\]()"]/g, '')}]`;
      case 4:
        return `${n}(${label.replace(/[[\]()"]/g, '')})`;
      default:
        return `${n}[${label.replace(/[[\]()"]/g, '')}]`;
    }
  };
  return () => {
    id = 0;
    const lines = [];
    let head = '';
    if (chance(0.15)) head += `---\ntitle: ${pick(['Sprint 12', 'A board with a title wider than the board itself, by far, much wider', 'T'])}\n---\n`;
    if (chance(0.2)) {
      const conf = [];
      if (chance(0.6)) conf.push(`"sectionWidth": ${pick([80, 120, 150.5, 260, 320, 10, 3000, '"x"', -5])}`);
      if (chance(0.6)) conf.push(`"ticketBaseUrl": ${pick(['"https://example.com/browse/#TICKET#"', '"#TICKET#"', '"javascript:#TICKET#"', '"/t/#TICKET#/#TICKET#"', '""'])}`);
      head += `%%{init: {"kanban": {${conf.join(', ')}}}}%%\n`;
    }
    lines.push('kanban');
    const indent = pick(['  ', '    ', '\t', ' ']);
    const columns = Math.floor(rnd() * 6);
    for (let c = 0; c < columns; c++) {
      lines.push(indent + node(pick(words)));
      if (chance(0.15)) lines.push(indent + `::icon(${pick(icons)})`);
      if (chance(0.15)) lines.push(indent + `:::${pick(['wide', 'a b', 'x-y_z'])}`);
      const cards = Math.floor(rnd() * 6);
      for (let k = 0; k < cards; k++) {
        let line = indent + indent + node(pick(texts));
        if (chance(0.6)) {
          const meta = [];
          if (chance(0.5)) meta.push(`ticket: ${pick(tickets)}`);
          if (chance(0.5)) meta.push(`assigned: ${pick(people)}`);
          if (chance(0.5)) meta.push(`priority: ${pick(priorities)}`);
          if (chance(0.15)) meta.push(`icon: ${pick(icons)}`);
          if (chance(0.1)) meta.push(`label: "${pick(texts).replace(/"/g, '')}"`);
          line += chance(0.2) ? `@{\n${meta.map((m) => indent + indent + indent + m).join('\n')}\n${indent + indent}}` : `@{ ${meta.join(', ')} }`;
        }
        lines.push(line);
        if (chance(0.08)) lines.push(indent + indent + `::icon(${pick(icons)})`);
        if (chance(0.08)) lines.push(indent + indent + `:::${pick(['urgent', 'urgent large', 'bad"class'])}`);
        if (chance(0.03)) lines.push(indent + indent + indent + node('deeper'));
      }
    }
    if (chance(0.03)) lines.push(node('misplaced'), indent + node('after'));
    return head + lines.join('\n') + (chance(0.5) ? '\n' : '');
  };
}

function dumpNode(n) {
  return {
    id: n.id,
    level: n.level,
    label: n.label,
    icon: n.icon,
    assigned: n.assigned,
    ticket: n.ticket,
    priority: n.priority,
    cssClasses: n.cssClasses,
  };
}

function dumpModel(m) {
  return {
    title: m.title || undefined,
    sections: m.sections.map((s) => ({ ...dumpNode(s), items: s.items.map(dumpNode) })),
  };
}

export default async function ({ src, pele, write, random }) {
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { kanban } = await src('diagrams/kanban/index.ts');
  const { KanbanDb } = await src('diagrams/kanban/db.ts');
  const { parseKanban, tokenize } = await src('diagrams/kanban/parser.ts');

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
        const db = new KanbanDb();
        parseKanban(source, db);
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
    if (out.type !== 'kanban') return out;
    const piped = parsed(() => kanban.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title));
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
  const rnd = random(21);
  const next = mutator(base, FRAGMENTS, rnd);
  for (let i = 0; i < 3000; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  const board = boards(random(22));
  for (let i = 0; i < 1000; i++) add(board());
  // Random strings of tokens, where the lexer's states and the parser's lookahead matter.
  const trnd = random(23);
  const pieces = [...new Set([...FRAGMENTS, ...FRAGMENTS.map((f) => f.trim()).filter((f) => f !== '')])];
  const openers = ['kanban\n', 'kanban\n  t\n    ', 'kanban\n  t\n    a@{', 'kanban\n  t[', 'kanban\n  t\n    a["`', '', '%% c\n'];
  for (let i = 0; i < 1000; i++) {
    let s = openers[Math.floor(trnd() * openers.length)];
    for (let k = 1 + Math.floor(trnd() * 10); k > 0; k--) s += pieces[Math.floor(trnd() * pieces.length)] + (trnd() < 0.3 ? ' ' : '');
    add(s);
  }

  write('diagram/kanban/testdata/pele/kanban.json', { cases: all.map(run) });
}
