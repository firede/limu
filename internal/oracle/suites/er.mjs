// Entity relationship diagrams parsed as Pele parses them, from text to ErDb, and the graph
// buildErGraph makes of it: the sources of Pele's tests and Mermaid's specs and
// documentation, cases written here, those sources mutated at random, and random strings of
// tokens.
import { peleError } from '../lib/value.mjs';
import { lone } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { locator } from '../lib/located.mjs';

// Pele's fragments, tests/er/oracle.test.ts.
export const FRAGMENTS = [
  ' ', '\n', '\t', '\r\n', ' \n', ';', ':', ':::', ',', '#', '"', '""', '`', '``', '~', '{', '}', '[', ']', '(', ')',
  '?', '*', '.', '-', '_', '|', '%', '\\', '||', '|o', 'o|', '}o', 'o{', '}|', '|{', '--', '..', '.-', '-.', 'u',
  'u--', 'u..', '||--o{', '}|..|{', '|o--o|', ' 1 ', '1', '1+', '0+', '1.5', '12', '0', '7', '2', '4', '6', '9',
  ' one ', ' only one ', ' one or zero ', ' one or more ', ' one or many ', ' zero or one ', ' zero or more ',
  ' zero or many ', ' many(0) ', ' many(1) ', ' many ', ' to ', ' optionally to ', 'one', 'only', 'zero', 'or',
  'many', 'to', 'optionally', 'A', 'B', 'id1', 'CUSTOMER', 'LINE-ITEM', 'é', 'µ', 'ö', ' ', ' ', 'PK',
  'FK', 'UK', 'pk', ' PK, FK ', 'string', 'int', 'type~T~', 'List~int~', ' "comment" ', '"a b"', '"a%b"', 'erDiagram',
  'ERDIAGRAM', 'style ', 'style', 'classDef ', 'classDef', 'class ', 'class', 'subgraph ', 'subgraph', 'end', 'end\n',
  'END ', 'direction TB', 'direction BT', 'direction RL', 'direction LR', 'direction', 'DIRECTION\ntb',
  'accTitle: ', 'accDescr: ', 'accDescr { ', 'acctitle:', 'title', 'fill:#f9f', 'stroke-width:2px', 'color:red',
  'default', ' : ', ' : label\n', '{\n', '\n}\n', 'A { string b }', 'p[Person]', 'a["x y"]', '%%', '\v', '\f',
  // More of what Pele reads beyond the grammar: front matter, entities, Unicode.
  '😀', '中文', '#35;', '#quot;', '&amp;', '<br>', '**b**', '_i_', '---\ntitle: T\n---\n', '　', 'ſ', 'K',
];

// The seeds of Pele's oracle test, and cases of every construct, written here.
export const WRITTEN = [
  'erDiagram\nCUSTOMER ||--o{ ORDER : places\nORDER ||--|{ LINE-ITEM : contains\n',
  'erDiagram\nA |o--o| B : "a b"\nC }o..o{ D : ""\nE }|.-|{ F : x\nG ||-.|| H : y\nI u--o{ J : z\n',
  'erDiagram\nA one or zero to one or more B : r\nC zero or many optionally to many(1) D : r\nE only one to 1+ F : r\n',
  'erDiagram\nA 1 to zero or more B : r\nC many(0) optionally to 0+ D : r\nE 1--1 F : r\nG 1..many H : r\nx 1--1 12 : r\n',
  'erDiagram\nBOOK {\n  string title PK "The title"\n  string? subtitle\n  int *pages UK\n  List~string~ tags FK, UK "many"\n  `my type` `my name` PK,FK\n}\n',
  'erDiagram\nA{}\nB { }\nC {string name}\nD:::x { int id PK }\nE:::x,y\nF:::x {}\n',
  'erDiagram\np[Person] {\n string firstName\n}\na["Customer Account"]:::big {\n string email\n}\nb[B]\nc[C]:::k\nd[D] {}\ne[E]:::k {}\np ||--o| a : has\n',
  'erDiagram\nA:::x ||--o{ B:::y,z : rel\nC:::x ||--o{ D : rel\nE ||--o{ F:::y : "r"\n',
  'erDiagram\nA\nB\nstyle A fill:#f9f,stroke:#333,stroke-width:4px\nstyle A,B color:red\nclassDef big,small font-size:12pt\nclassDef default fill:#f9f;\nclass A big\nclass A,B big,small\n',
  'erDiagram\ndirection LR\nsubgraph one1\n A\n B ||--|| C : x\n subgraph "Two Words"\n  direction TB\n  D { int id }\n end\n subgraph s3 [Title of three]\n  E\n end\nend\none1 ||--|| E : links\nstyle one1 fill:red\nclass s3 big\n',
  'erDiagram\naccTitle: A title\naccDescr: A description\nA\n',
  'erDiagram\naccDescr {\n several\n lines\n}\nsubgraph g\naccTitle: inside\nclassDef c fill:red\nstyle A fill:red\nclass A c\nA\nend\n',
  'erDiagram\n"Quoted name" ||--|| "Other ❤ name" : "a label"\n1.5 ||--|| 2 : n\n_x ||--|| y-z : n\nA.B ||--|| C*D : n\n',
  'erDiagram A B C\nD { a b c d } E\n',
  // Every cardinality at each end, identifying and not.
  'erDiagram\nA |o--|| B : a\nA ||--|o B : b\nA }o--|| B : c\nA ||--o{ B : d\nA }|--|| B : e\nA ||--|{ B : f\nA u--|| B : g\nA ||--u B : h',
  'erDiagram\nA |o..|| B : a\nA ||..|o B : b\nA }o..|| B : c\nA ||..o{ B : d\nA }|..|| B : e\nA ||..|{ B : f\nA u..|| B : g\nA ||..u B : h',
  'erDiagram\nA zero or one to only one B : a\nA one or zero optionally to one or many B : b\nA zero or more to zero or many B : c',
  'erDiagram\nA ONE OR MORE TO MANY(0) B : a\nA Only One Optionally To Zero Or One B : b\nA MANY to MANY B : c',
  'erDiagram\nA |O--O| B : a\nA }O--O{ B : b\nA |o--o|B:c\nA}o--o{B : d',
  // Labels, quoted and not.
  'erDiagram\nA ||--o{ B : places\nA ||--o{ B : "places many"\nA ||--o{ B : ""\nA ||--o{ B : "with <br> break"\nA ||--o{ B : "#quot;q#quot;"',
  'erDiagram\nA ||--o{ B : label with spaces',
  'erDiagram\nA ||--o{ B :',
  'erDiagram\nA ||--o{ B',
  // Attributes.
  'erDiagram\nA {\n  int id PK\n  string name UK "the name"\n  int ref FK\n  int both PK, FK\n  int three PK,FK,UK "all"\n}',
  'erDiagram\nA {\n  Map~string,int~ m\n  List~List~int~~ nested\n  a~b c\n  ~x~ y\n  type~ name\n}',
  'erDiagram\nA {\n  string name "a comment with **bold** and _it_"\n  string other "line<br>break"\n  int n "#35; and &amp;"\n}',
  'erDiagram\nA {\n  string? optional\n  int[] array\n  decimal(10,2) price\n  *id key\n}',
  'erDiagram\nA {\n  string name pk\n  int x fk\n  int y uk\n  int z Pk\n}',
  'erDiagram\nA {\n  string\n}',
  'erDiagram\nA {\n  string name PK,\n}',
  'erDiagram\nA {\n  string name "unclosed\n}',
  'erDiagram\nA {\n  `with space` `other name`\n  `unclosed\n}',
  'erDiagram\nA {\n  int a\n}\nA {\n  int b\n}\nA[Alias] {\n  int c\n}',
  'erDiagram\nA {\n  int µ\n  int é\n  Ā x\n}',
  'erDiagram\nA {\n  string name ? x\n}',
  'erDiagram\nA {\n  string? ? x\n}',
  // Aliases.
  'erDiagram\np[Person]\np[Other]\nq["Two words"]\nr[1]\ns[1.5]\nt["😀"]',
  'erDiagram\np[Person] ||--o{ q : x',
  'erDiagram\np[]',
  'erDiagram\np[a b]',
  // Classes and styles.
  'erDiagram\nclassDef c fill:#f96,color:#fff\nclassDef d stroke:red,stroke-width:2px;\nA:::c\nB:::c,d\nclass A d\nstyle B fill:#0f0',
  'erDiagram\nclassDef c fill:#f96\nA\nclass A,B c\nclass Z c\nstyle Z fill:red',
  'erDiagram\nA\nstyle A fill:#f9f,stroke:#333,stroke-width:4px,stroke-dasharray: 5 5',
  'erDiagram\nA\nstyle A fill',
  'erDiagram\nA\nstyle A',
  'erDiagram\nA\nclassDef',
  'erDiagram\nA\nclass A',
  'erDiagram\nA\nstyle A fill:red;color:blue',
  'erDiagram\nA\nstyle A fill:red ',
  'erDiagram\nA\nstyle A fill:#f00 !',
  // Directions.
  'erDiagram\ndirection LR\nA ||--o{ B : x',
  'erDiagram\ndirection RL\nA',
  'erDiagram\ndirection BT\nA',
  'erDiagram\n  direction   tb  extra words\nA',
  'erDiagram\nA direction LR',
  'erDiagram\ndirection\nLR\nA',
  'erDiagram\ndirection XX\nA',
  'erDiagram\nA ||--o{ B : "direction LR"',
  // Subgraphs.
  'erDiagram\nsubgraph S\nA\nB\nend\nsubgraph T\nA\nC\nend',
  'erDiagram\nsubgraph S\nsubgraph T\nA\nend\nend\nS ||--|| T : x',
  'erDiagram\nA ||--|| S : before\nsubgraph S\nB\nend\nS ||--|| A : after',
  'erDiagram\nsubgraph S [Title words here]\nA\nend',
  'erDiagram\nsubgraph S ["quoted title"]\nA\nend',
  'erDiagram\nsubgraph " "\nA\nend',
  'erDiagram\nsubgraph S\ndirection LR\nA\ndirection BT\nend',
  'erDiagram\nsubgraph S\nA\n',
  'erDiagram\nend',
  'erDiagram\nsubgraph S;A\nend',
  'erDiagram\nsubgraph S\nA\nend\nsubgraph S\nB\nend',
  'erDiagram\nsubgraph S\nS\nend',
  'erDiagram\nsubgraph S\n" A "\nend\nsubgraph T\n" A "\nend',
  'erDiagram\nsubgraph S\nA\nend\nsubgraph T\n" A "\nA\nend',
  'erDiagram\nsubgraph S\nstyle A fill:red\nclass A c\nclassDef c fill:blue\nA\nend\nclass S c',
  // Accessibility and titles.
  '---\ntitle: The title\n---\nerDiagram\naccTitle:   spaced  \naccDescr: text\nA',
  'erDiagram\naccDescr {\n  line one\n    line two\n\n  line three\n}\nA',
  'erDiagram\naccTitle:\nA',
  'erDiagram\naccTitle: x',
  'erDiagram\naccDescr {\nunclosed',
  'erDiagram\nACCTITLE : upper\nAccDescr{ mixed }\nA',
  // Unicode, entities and errors.
  'erDiagram\n中文 ||--o{ 日本語 : 関係\nÄ ||--|| Ö : ü',
  'erDiagram\n😀 ||--|| 🎉 : "😀"',
  'erDiagram\nA ||--||　B : x',
  'erDiagram\nA #35; B',
  'erDiagram\nA ||--o{ B : x\nA ||--o{',
  'erDiagram\nA ||--',
  'erDiagram\nA ||',
  'erDiagram\nA }|',
  'erDiagram\n||--|| B : x',
  'erDiagram\nA ? B',
  'erDiagram\nA {\n',
  'erDiagram\nA {\n string name\n',
  'erDiagram\nA {\n ? x\n}',
  'erDiagram\n\u0001',
  'erDiagram\n%%',
  'erdiagram\nA',
  'ERDIAGRAM\nA',
  'not er',
  'erDiagram',
  '',
];

function attributes(list) {
  return list.map((a) => ({ type: a.type, name: a.name, keys: a.keys, comment: a.comment }));
}

function dumpDb(db) {
  const opt = (x) => (x === undefined || x === '' ? undefined : x);
  return {
    direction: db.direction,
    title: opt(db.title),
    accTitle: opt(db.accTitle),
    accDescr: opt(db.accDescr),
    entities: [...db.entities.entries()].map(([name, e]) => ({
      name,
      id: e.id,
      label: e.label,
      alias: e.alias,
      attributes: attributes(e.attributes),
      cssClasses: e.cssClasses,
      cssStyles: e.cssStyles,
    })),
    relationships: db.relationships.map((r) => ({
      entityA: r.entityA,
      roleA: r.roleA,
      entityB: r.entityB,
      cardA: r.relSpec.cardA,
      relType: r.relSpec.relType,
      cardB: r.relSpec.cardB,
    })),
    classes: [...db.classes.values()].map((c) => ({ id: c.id, styles: c.styles, textStyles: c.textStyles })),
    subgraphs: db.subgraphs.map((s) => ({
      id: s.id,
      nodes: s.nodes,
      title: s.title,
      classes: s.classes,
      cssStyles: s.cssStyles,
      dir: s.dir,
      labelType: s.labelType,
    })),
  };
}

function dumpGraph(graph) {
  return {
    nodes: graph.nodes.map((n) => ({
      id: n.id,
      label: n.label,
      alias: n.alias,
      attributes: attributes(n.attributes),
      isGroup: n.isGroup,
      parentId: n.parentId,
      cssStyles: n.cssStyles,
      cssCompiledStyles: n.cssCompiledStyles,
      cssClasses: n.cssClasses,
      dir: n.dir,
      labelType: n.labelType,
    })),
    edges: graph.edges.map((e) => ({
      id: e.id,
      start: e.start,
      end: e.end,
      label: e.label,
      arrowTypeStart: e.arrowTypeStart,
      arrowTypeEnd: e.arrowTypeEnd,
      pattern: e.pattern,
    })),
  };
}

export default async function ({ src, cases, write, random }) {
  const located = await locator(src);
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { er } = await src('diagrams/er/index.ts');
  const { ErDb } = await src('diagrams/er/db.ts');
  const { parseEr } = await src('diagrams/er/parser.ts');
  const { buildErGraph } = await src('diagrams/er/graph.ts');

  const caught = (e) => {
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  // Text that is no ER diagram goes to the parser as it is, as Pele's tests give it.
  const direct = (source) => {
    try {
      const db = new ErDb();
      parseEr(source, db);
      return { db: dumpDb(db), graph: dumpGraph(buildErGraph(db)) };
    } catch (e) {
      return caught(e);
    }
  };
  // Pele's parseSource for an ER diagram, and the graph its render builds.
  const run = (source) => {
    let pre;
    let type;
    try {
      pre = preprocess(source);
      type = detect(pre.text);
    } catch (e) {
      return caught(e);
    }
    if (type !== 'er') return { type, direct: direct(source) };
    try {
      const db = located(source, false, () => er.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title, pre.lineOffset, {}));
      return { type, db: dumpDb(db), graph: dumpGraph(buildErGraph(db)) };
    } catch (e) {
      return { type, ...caught(e) };
    }
  };

  const extracted = cases('er').extracted;
  const corpus = [...new Set([...WRITTEN, ...extracted])].filter((s) => /erDiagram/i.test(s));
  const next = mutator(corpus, FRAGMENTS, random(7));
  // Most mutations do not parse; of those, enough are kept to cover the errors, and the rest
  // are drawn until as many again parse.
  const fuzz = [];
  let parsed = 0;
  let failed = 0;
  while (parsed < 2000) {
    const s = next();
    if (lone(s) || s.length > 4000) continue;
    const r = run(s);
    const ok = r.db !== undefined || r.direct?.db !== undefined;
    if (ok) parsed++;
    else if (failed++ >= 1200) continue;
    fuzz.push({ src: s, ...r });
  }
  // Mutation rarely joins two tokens without a space, which is where keyword boundaries and
  // lookahead matter. These string random pieces together, in each lexer state, as Pele's
  // oracle test does.
  const rnd = random(8);
  const pieces = [...new Set([...FRAGMENTS, ...FRAGMENTS.map((f) => f.trim()).filter((f) => f !== '')])];
  const pick = () => pieces[Math.floor(rnd() * pieces.length)];
  const openers = ['erDiagram\n', 'erDiagram\nA {\n', 'erDiagram\nstyle A ', 'erDiagram\nA ', 'erDiagram\nsubgraph s\n', ''];
  const sequences = [];
  while (sequences.length < 1500) {
    let s = openers[Math.floor(rnd() * openers.length)];
    for (let k = 1 + Math.floor(rnd() * 10); k > 0; k--) s += pick() + (rnd() < 0.3 ? ' ' : '');
    if (!lone(s)) sequences.push(s);
  }
  const all = [...[...WRITTEN, ...extracted].map((s) => ({ src: s, ...run(s) })), ...fuzz, ...sequences.map((s) => ({ src: s, ...run(s) }))];
  write('diagram/er/testdata/pele/er.json', { cases: all });
}
