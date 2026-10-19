// Class diagrams parsed as Pele parses them, from text to ClassDb, and the graph
// buildClassGraph makes of it: the sources of Pele's tests and Mermaid's specs and
// documentation, Pele's tour of the grammar, cases written here, and those sources mutated at
// random. Members are also read on their own, from random text, as Pele's tests read them.
import { lone, mutator, peleError } from './parse.mjs';

// Pele's fragments, tests/class/oracle.test.ts, and more of what reaches past the grammar:
// entities, Unicode, generics, styles.
const FRAGMENTS = [
  ' ', '\n', '\r\n', '\t', ';', ':', '::', ':::', ': x', ',', '.', '..', '-', '--', '<|', '|>', '<', '>', '<<', '>>',
  '*', 'o', ' o ', '()', '(', ')', '{', '}', '{\n', '\n}', '[', ']', '[*]', '"', '""', '"x"', '`', '`a b`', '~', '~T~',
  '#', '%', '%%', '%% c\n', '+', '=', '!', '$', '&', '?', '/', '\\', '_', 'A', 'B', 'Class01', '1', 'é', 'ö', '中',
  'class ', 'class', 'classDiagram', 'classDiagram-v2', 'classDef ', 'cssClass ', 'style ', 'namespace ',
  'namespace N {\n', 'note ', 'note for ', 'note for A "x"', 'link ', 'click ', 'callback ', 'href ', 'call ',
  'call f()', 'call f(1, "a")', '_self', '_blank', '_parent', '_top', 'direction TB', 'direction BT', 'direction RL',
  'direction LR', 'direction', 'accTitle: ', 'accTitle', 'accDescr: ', 'accDescr {', 'accDescr { x }', '<|--', '--|>',
  '*--', '--*', 'o--', '--o', '-->', '<--', '..>', '<..', '..|>', '<|..', '()--', '--()', '"1" ', ' "many"',
  '<<interface>>', 'fill:#f9f', 'stroke-width:2px', '+foo()', '-bar: int', 'x(y) z$', ' \n', '}\n', 'A.B', 'a-b',
  '#35;', '#quot;', '&lt;', '&amp;', '😀', '　', ' ', 'List~int~', 'Map~K, V~', '+get(i int)$ T*', 'x$', 'y*',
  'font-style:italic', 'font-size:20px', 'font-weight:bold', '<br>', '<b>b</b>', '**s**', '---\ntitle: T\n---\n',
  '%%{init: {"class": {"hierarchicalNamespaces": false}}}%%\n', 'link A "https://x.y/z" "tip" _self',
  'click A href "javascript:alert(1)"', 'cssClass "A,B" hot', 'classDef hot fill:#f00,color:#fff',
];

export const WRITTEN = [
  'classDiagram\nclass A',
  // A class is looked up under a prefix when its id starts with a digit, so these do not reach it.
  'classDiagram\nclass `classId-1`\nclass `1`\ncssClass "1" k\nlink 1 "https://a.b"',
  'classDiagram\nclass `classId-1`\nclass 1\ncssClass "1,classId-1" k',
  // A direction needs a space before its name.
  'classDiagram\ndirectionLR\nclass A',
  'classDiagram\nclass directionLR\nclass A',
  'classDiagram\nA --> B : directionLR\nclass C',
  'classDiagram\nA --> B : x direction LR\nclass C',
  'classDiagram\nclass A_directionTB_x',
  // A class with many relations on one side, widened for them as far as it goes.
  'classDiagram\nHub "many" <-- "many" A1\nHub "many" <-- "many" A2\nHub "many" <-- "many" A3\nHub "many" <-- "many" A4\nHub "many" <-- "many" A5\nHub "many" <-- "many" A6\nHub "many" <-- "many" A7\nHub "many" <-- "many" A8\nHub "many" <-- "many" A9',
  'class A',
  'B <|-- A~',
  'A~x',
  'class A\n"',
  'A <|-- B\nnote "',
  'click A call f',
  'accTitle:',
  'A\n',
  'A',
  // Members: visibility, classifiers, generics, methods with return types.
  'classDiagram\nclass Shape {\n  +String name\n  -int size$\n  #List~int~ items\n  ~internal* \n  +area() double\n  +move(x int, y int)$ void\n  -draw()* \n  +get~T~(Map~K, V~ m) List~T~\n  +run)\n  (a)\n  <<interface>>\n}',
  'classDiagram\nclass A {\n  +x(\n  y)\n  m() T$\n  n() T*\n  o()$T\n  p()*\n  q(a,b,c)  ret  \n}',
  'classDiagram\nA : +int age\nA : +isMammal()\nA: -List~Animal~ friends$\nA :<<abstract>>\nA : \nA : x',
  'classDiagram\nclass Square~Shape~{\n  int id\n  List~int~ position\n  setPoints(List~int~ points)\n  getPoints() List~int~\n}\nSquare : -List~string~ messages\nSquare : +setMessages(List~string~ messages)\nSquare : +getMessages() List~string~\nSquare : +getDistanceMatrix() List~List~int~~',
  'classDiagram\nclass A {\n  +x #35; y\n  +f(#quot;a#quot;)\n  &lt;b&gt;\n}',
  // Annotations.
  'classDiagram\nclass Shape\n<<interface>> Shape\n<<Service>> Shape~T~\nclass Color {\n<<enumeration>>\nRED\n}\nclass B <<abstract>>\nclass C <<x>> {\n+y\n}\nclass D <<x>> {}',
  'classDiagram\n<<interface>> Shape\nclass Shape {\n  draw()\n}',
  // Namespaces.
  'classDiagram\nnamespace A.B {\n class X\n}\nnamespace A {\n namespace C["See"] {\n  class Y\n  note "n"\n }\n}',
  'classDiagram\nnamespace Company.Project {\n  class User\n}\nnamespace Company.Project.Module {\n  class Admin\n}\nAdmin --> User',
  'classDiagram\nnamespace Outer {\n  namespace Inner {\n    class Foo\n  }\n  class Baz\n}\nFoo --> Baz\nnote for Foo "in"',
  'classDiagram\nnamespace A {\nclass X\n}\nnamespace A["Again"] {\nclass Y\n}\nnamespace `a.b` {\nclass Z\n}',
  '%%{init: {"class": {"hierarchicalNamespaces": false}}}%%\nclassDiagram\nnamespace A.B.C {\n class X\n}\nnamespace A {\n class Y\n}\nX --> Y',
  'classDiagram\nnamespace N {\n}',
  'classDiagram\nnamespace N { class A }',
  'classDiagram\nnamespace N {\nclass A\nnote for A "n"\nnote "m"\n}\nnamespace N.M {\nclass B\n}',
  // Notes.
  'classDiagram\nnote "free"\nnote for A "for A"\nclass A\nnote for B~T~ "generic"\nnote for Missing "x"',
  // Relations of every kind, with cardinalities and labels.
  'classDiagram\nA <|-- B\nC *-- D\nE o-- F\nG <-- H\nI -- J\nK <.. L\nM <|.. N\nO .. P\nQ --|> R\nS --* T\nU --o V\nW --> X\nY ..> Z',
  'classDiagram\nA "1" --> "*" B : has\nC "0..1" <--> "1..n" D : link\nE "many" o--o "one" F\nG *--* H : both\nI <|--|> J',
  'classDiagram\nA "1" --> B\nA --> "2" B : x\nA "" --> B\nA : : x\nA --> B :\nA --> B : ',
  'classDiagram\nbar ()-- foo\nfoo --() baz\nA ()--() B\nC ()-- D~T~\nE~T~ --() F',
  'classDiagram\nclass interface0\nclass note0\nbar ()-- interface0\nnote for note0 "n"\nnamespace note0 {\n class X\n}',
  'classDiagram\nA --> A\nA --> A : self\nA "1" --> "2" A',
  'classDiagram\nA~T~ --> B~U~ : generic\nA.B --> C-D\n`a b` --> `c`\né --> 中',
  // Styles, classes, links, callbacks.
  'classDiagram\nclass A:::defined\nclass B\nclassDef def fill:#f00\nclassDef defined fill:#0f0',
  'classDiagram\nclass A\nclassDef default fill:#f00,stroke:#000\nclass B',
  'classDiagram\nclass 1A\nclass B\ncssClass "1A,B" hot\nlink 1A "https://example.com"',
  'classDiagram\nclass A\nclick A call alert("x") "tip"\ncallback A "alert"',
  'classDiagram\nclass A\nclass B\nlink A "javascript:alert(1)"\nclick B href "https://example.com/a b" "tip" _self',
  'classDiagram\nclass A\nlink A "#35;x" _top\nclick A href "  "\nlink A "&#106;avascript:x" "t"\nlink A "mailto:a@b.c" "m" _parent',
  'classDiagram\nclass A\nstyle A fill:#f9f,stroke:#333,stroke-width:4px\nstyle B fill:#fff\nclassDef hot,cold color:#f00,fill:#0f0\ncssClass "A" hot\nclass C:::cold',
  'classDiagram\nclass A:::big\nclassDef big font-size:24px,font-style:italic\nclass B\nstyle B font-weight:bold,font-size:10px',
  'classDiagram\nclass A\nclassDef x a % # : style\nclassDef y fill:#f9f;',
  // Direction, accessibility, titles.
  'classDiagram\ndirection LR\nA --> B',
  'classDiagram\ndirection RL\nA --> B\ndirection BT',
  'classDiagram\nx direction LR y\nA',
  'classDiagram\ndirection\nTB\nA',
  'classDiagram\naccTitle: My Title  \naccDescr: My Description\nA',
  'classDiagram\naccDescr {\n  line one\n    line two\n}\nA',
  'classDiagram\naccTitle:\nA',
  '---\ntitle: Animals\n---\nclassDiagram-v2\n  Animal <|-- Duck',
  // Unicode and entities.
  'classDiagram\nclass 中文 {\n  +名字 String\n}\nclass Città["Città 😀"]\nA["x"]',
  'classDiagram\nclass A["#quot;q#quot; &amp; #9829;"]\nA : +x #35;\nnote for A "a<br>b #lt;"',
  'classDiagram\nclass C1["With ~!@#$%^&*()_+=-/?"]\nclass C2[" "]',
  // Errors.
  'classDiagram\nnote link',
  'classDiagram\nclass A { x',
  'classDiagram\nclass A\n@',
  'classDiagram\n',
  'classDiagram\nnamespace N {\n}',
  'classDiagram\nA -- ',
  'classDiagram\nA <|',
  'classDiagram\nclass A:::',
  'classDiagram\nclass A["L"',
  'classDiagram\nstyle',
  'classDiagram\ncssClass A hot',
  'classDiagram\nA 😀 B',
  'classDiagram\n\u0001',
  'not a class diagram',
  '',
];

// Pele's tour of the grammar, from tests/class/oracle.test.ts: one input for each way through
// each rule, valid or not.
function tour() {
  const names = ['A', 'A.B', 'A-B', 'A B', '`a b`', 'A`b`', 'A~T~', '`a`~T~', 'A.`b`~T~', 'é', '1', 'A.', '.A', 'A~T~~U~', '`a`B', 'o', 'link'];
  const ends = ['', '<|', '|>', '<', '>', '*', 'o', '()'];
  const out = [];
  for (const a of names) {
    out.push(a, `${a} : label`, `class ${a}`, `namespace ${a} {\nclass X\n}`, `<<i>> ${a}`, `note for ${a} "n"`, `click ${a} href "u"`);
    for (const b of ['A', 'B~T~', '`b`']) out.push(`${a} --> ${b}`, `${a} "1" --> "2" ${b} : t`);
  }
  for (const left of ends) {
    for (const line of ['--', '..', '-', '']) {
      for (const right of ends) {
        out.push(`A ${left}${line}${right} B`, `A "1" ${left}${line}${right} "2" B : t`, `A ${left}${line}${right}`);
      }
    }
  }
  const bodies = ['', ' {\n}', '{}', '{ x }', ' {\n+a\n-b()\n}', ' {\n"q"\n}', ' {\n[*]\n}', ' { {', ' {\n+a', ' }'];
  for (const label of ['', '["L"]', '[L]', '["L"']) {
    for (const mid of ['', ':::c', ':::', ' <<i>>', ' <<i', ' : x', ' B']) {
      for (const body of bodies) out.push(`class A${label}${mid}${body}`, `namespace N {\nclass A${label}${mid}${body}\n}`);
    }
  }
  const items = ['class A', 'class B {\n+x\n}', 'note "n"', 'note for A "n"', 'namespace M {\nclass C\n}', 'A --> B', ''];
  for (const label of ['', '["L"]']) {
    for (const open of ['{', '{\n', ' {\n\n']) {
      for (const first of items) {
        for (const sep of ['\n', ' ', '\n\n']) {
          for (const second of items) {
            out.push(`namespace N${label} ${open}${first}${sep}${second}}`, `namespace N${label} ${open}${first}${sep}${second}\n}\nclass Z`);
          }
        }
      }
    }
  }
  out.push('namespace A {\nnamespace B {\nnamespace C {\nclass X\n}\n}\nclass Y\n}', 'namespace A { namespace B { class X } }', 'namespace A {\nclass X\n}}', 'namespace namespace A {\nclass X\n}');
  for (const head of ['callback A', 'link A', 'click A href', 'click A call f()', 'click A call f(1, "a")', 'click A call f', 'click A']) {
    for (const tail of ['', ' "a"', ' "a" "b"', ' "a" _blank', ' "a" "b" _self', ' "a" "b" "c"', ' _top', ' "a" _parent "b"']) out.push(head + tail);
  }
  for (const head of ['style A', 'style', 'classDef a', 'classDef a,b', 'classDef a,', 'classDef', 'cssClass "A"', 'cssClass "A,B"', 'cssClass A']) {
    for (const tail of ['', ' fill:#f9f', ' fill:#f9f,stroke:#333,stroke-width:4px', ' x,y', ' c', ' x:1;', ' a % # : style', ' "s"']) out.push(head + tail);
  }
  for (const text of ['"n"', '', 'n', '"a" "b"', '""']) out.push(`note ${text}`, `note for A ${text}`, `note for ${text}`, `notefor A ${text}`);
  for (const line of [
    'direction TB', 'direction BT', 'direction RL', 'direction LR', 'direction TD', 'x direction LR y', 'direction\nTB', 'direction LR direction TB',
    'accTitle: t', 'accTitle:', 'accTitle : t', 'accDescr: d', 'accDescr { a\nb }', 'accDescr {}', 'accDescr { a', 'accTitle t',
    '<<i>> A', '<<i>>', '<<i> A', '<< i >> A~T~', '%% c', 'A %% c', '[*] --> A', 'A --> [*]', 'call f()', 'A::B', 'A:::c', 'A;',
  ]) {
    out.push(line, `${line}\nclass A`, `class A\n${line}`);
  }
  return out;
}

const opt = (x) => (x === undefined || x === '' ? undefined : x);

function dumpMember(m) {
  return {
    id: m.id,
    memberType: m.memberType,
    visibility: m.visibility,
    classifier: m.classifier,
    parameters: m.parameters,
    returnType: m.returnType,
    display: m.getDisplayDetails().displayText,
  };
}

function dumpDb(db) {
  return {
    direction: db.direction,
    title: opt(db.title),
    accTitle: opt(db.accTitle),
    accDescr: opt(db.accDescr),
    classes: [...db.classes.values()].map((c) => ({
      id: c.id,
      type: c.type,
      label: c.label,
      cssClasses: c.cssClasses,
      methods: c.methods.map(dumpMember),
      members: c.members.map(dumpMember),
      annotations: c.annotations,
      styles: c.styles,
      parent: opt(c.parent),
      link: opt(c.link),
      linkTarget: opt(c.linkTarget),
      haveCallback: c.haveCallback === true ? true : undefined,
      tooltip: opt(c.tooltip),
    })),
    relations: db.relations.map((r) => ({
      id1: r.id1,
      id2: r.id2,
      relationTitle1: r.relationTitle1,
      relationTitle2: r.relationTitle2,
      title: r.title,
      type1: r.relation.type1,
      type2: r.relation.type2,
      lineType: r.relation.lineType,
    })),
    notes: [...db.notes.values()].map((n) => ({ id: n.id, class: opt(n.class), text: n.text, index: n.index, parent: opt(n.parent) })),
    namespaces: [...db.namespaces.values()].map((ns) => ({
      id: ns.id,
      label: ns.label,
      classes: [...ns.classes.keys()],
      notes: [...ns.notes.keys()],
      children: [...ns.children.keys()],
      parent: opt(ns.parent),
      explicit: ns.explicit,
    })),
    interfaces: db.interfaces.map((x) => ({ id: x.id, label: x.label, classId: x.classId })),
    styleClasses: [...db.styleClasses.values()].map((s) => ({ id: s.id, styles: s.styles, textStyles: s.textStyles })),
  };
}

function dumpGraph(graph) {
  return {
    nodes: graph.nodes.map((n) => ({
      id: n.id,
      label: n.label,
      isGroup: n.isGroup,
      shape: n.shape,
      parentId: n.parentId,
      parent: n.parent,
      classId: n.classNode?.id,
      colorIndex: n.colorIndex,
    })),
    edges: graph.edges.map((e) => ({
      id: e.id,
      start: e.start,
      end: e.end,
      source: e.source,
      target: e.target,
      label: e.label,
      arrowTypeStart: e.arrowTypeStart,
      arrowTypeEnd: e.arrowTypeEnd,
      startLabelRight: e.startLabelRight,
      endLabelLeft: e.endLabelLeft,
      pattern: e.pattern,
      classes: e.classes,
    })),
  };
}

// The pieces of Pele's test of members, tests/class/members.test.ts.
const PIECES = ['(', ')', '()', '+', '-', '#', '~', '$', '*', ' ', '  ', ':', ',', 'a', 'foo', 'int', 'List', '\r', '\n', ' ', '\t', 'é', '😀', '~T~'];

export default async function ({ src, cases, write, random }) {
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { classDiagram } = await src('diagrams/class/index.ts');
  const { buildClassGraph } = await src('diagrams/class/graph.ts');
  const { ClassDb } = await src('diagrams/class/db.ts');
  const { parseClassDiagram } = await src('diagrams/class/parser.ts');
  const { ClassMember } = await src('diagrams/class/members.ts');

  const caught = (e) => {
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  const dump = (db, hierarchical) => ({
    db: dumpDb(db),
    graph: dumpGraph(buildClassGraph(db, hierarchical)),
    flat: hierarchical && db.namespaces.size > 0 ? dumpGraph(buildClassGraph(db, false)) : undefined,
  });
  // Text that is no class diagram goes to the parser as it is, as Pele's tests give it.
  const direct = (source) => {
    try {
      const db = new ClassDb();
      parseClassDiagram(source, db);
      return dump(db, true);
    } catch (e) {
      return caught(e);
    }
  };
  // Pele's parseSource for a class diagram, and the graph its render builds.
  const run = (source) => {
    let pre;
    let type;
    try {
      pre = preprocess(source);
      type = detect(pre.text);
    } catch (e) {
      return caught(e);
    }
    if (type !== 'class') return { type, direct: direct(source) };
    try {
      const config = pre.config;
      const db = classDiagram.parse(encodeEntities(pre.text) + '\n', config, pre.title, pre.lineOffset, {});
      return { type, ...dump(db, (config.class ?? {}).hierarchicalNamespaces !== false) };
    } catch (e) {
      return { type, ...caught(e) };
    }
  };

  const c = cases('class');
  const toured = tour();
  const headed = toured.map((s) => 'classDiagram\n' + s);
  // Of the tour with a line feed after it, and with no header, every fifth.
  const more = [...toured.map((s) => 'classDiagram\n' + s + '\n'), ...toured].filter((_, i) => i % 5 === 0);
  const next = mutator([...c.extracted, ...headed], FRAGMENTS, random(7));
  const fuzz = [];
  while (fuzz.length < 2500) {
    const s = next();
    if (!lone(s) && s.length <= 4000) fuzz.push(s);
  }
  const all = [...WRITTEN, ...c.extracted, ...headed, ...more, ...fuzz];

  // Members read on their own, as methods and as attributes.
  const rnd = random(11);
  const members = [];
  for (let i = 0; i < 3000; i++) {
    let s = '';
    const n = 1 + Math.floor(rnd() * 9);
    for (let k = 0; k < n; k++) s += PIECES[Math.floor(rnd() * PIECES.length)];
    const kind = i % 4 === 0 ? 'attribute' : 'method';
    members.push({ input: s, kind, member: dumpMember(new ClassMember(s, kind)) });
  }

  write('diagram/class/testdata/pele/class.json', { diagrams: all.map((src) => ({ src, ...run(src) })), members });
}
