// Use case diagrams parsed as Pele parses them, from text to UsecaseModel, and the graph
// buildUsecaseGraph makes of it: Mermaid's documentation and specs, the sources of Pele's
// tests, cases written here, those sources mutated at random, diagrams built from random
// statements, and random strings of tokens. Each case is parsed as Pele's parse does it, after
// the preprocessing every diagram goes through, and as the parser alone reads the raw text.
import { readFileSync } from 'node:fs';
import { enc, peleError } from '../lib/value.mjs';
import { lone, picker, testFiles, testStrings, unquote, docs } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';
import { locator } from '../lib/located.mjs';

// Pele's fragments, tests/usecase/oracle.test.ts.
export const FRAGMENTS = [
  ' ', '\n', '\t', '\r\n', '\r', ' \n', '\n\n', ';', ':', '::', ':::', ',', '\\,', '#', '#f96', '"', '""', "'", "''", '`',
  '"`', '`"', '"`x`"', '"`a\nb`"', '{', '}', '[', ']', '(', ')', '@', '@{', '@{}', '@{ ', ' }', '%', '%%', '%% c', '.',
  '..', '..>', '...>', '-', '--', '---', '----', '-->', '--->', '---->', '<--', '<---', '--o', 'o--', '--x', 'x--', '--|>',
  '---|>', '<', '<<', '>>', '<<a>>', '<< a b >>', '<<>>', '<< >>', '<<  >>', '<<\n', '>', '|', '*', '_', '~', '!', '$', '&',
  '+', '/', '=', '?', '^', '\\', 'A', 'B', 'a1', '_x', 'o', 'x', 'ox', 'A-B', 'a-b-c', '--a', '1', '1mg', '1.5', '1.5px',
  '.5', '12', '4px', 'é', 'Ä', '日本語', ' ', ' ', '\u0001', '\u007f', 'usecase-beta', 'usecase-betax', 'usecase',
  'actor', 'actor ', 'actors', 'actor A[x]', 'systemBoundary', 'systemBoundary ', 'end', 'end\n', 'ends', 'endx-y',
  'direction', 'direction TB', 'direction LR', 'TD', 'TB', 'BT', 'LR', 'RL', 'TB1', 'note', 'note for ', 'note-x', 'for',
  'for-x', 'format', 'json', 'json a@{}', 'json a @ {"b":1}', 'json x@{', 'jsonx', 'classDef', 'classDef c fill:red', 'class',
  'class A c', 'classes', 'style', 'style A fill:red', 'include', 'INCLUDE', 'iNclude', 'included', 'extend', 'Extend',
  'eXtend', 'enduser-x', 'true', 'false', 'trueish', ': include ', ': extend ', '..> : include ', '..> : eXtend ',
  'accTitle', 'accTitle: t', 'accTitle : t', 'accDescr', 'accDescr: d', 'accDescr {', 'accDescr {\n', 'accDescr { d }',
  'accDescr{\n a\n b\n}', 'type', 'type: hollow', 'type: package', 'business: true', 'icon: "fa:user"', 'animate: true',
  'animation: fast', '"k": "v"', 'k: true', '"k": false', 'k: "v w"', 'true: k', ', k: v', '\nk: v', ' A@{ type: hollow }',
  ' e@--> ', 'e@', '("x")', '(x y)', '[x]', '["x"]', '("`x`")', '"s"(x)', '"s"[x]', '"s" (', '"`m`"[', ':::c', ':::c,d',
  ' -- l --> ', ' -- "l" -- ', ' -- "" --> ', ' <-- l -- ', ' <--- l -- ', ' --- l --> ', '"\'', '\'"', ' o-- l -- ', ' x-- l -- ', ' -- l --o ', ' -- l --x ',
  'package', 'rectangle', 'newpage', 'skinparam', 'allowmixing', 'fill:#fee', 'stroke-width:2px', 'a:b c', '--x:1',
  '{"a":1}', '{"a":{"b":[1,{"c":null}]}}', '{"a":1,"a":{"b":2}}', '"a":', '[1,2]', 'null', '\\"', '\\u0041', ',\n', ',\n\n',
  '\n,', '\n,\n', '\n\n\n', '\n\n,', ',\n}', ',\n\n}', '\n\n}', '\n\n\n}', '  %% c\n', 'A %% c', '\n%%', 'x--y', 'o--o',
  'a--b', 'a---b', 'a-->b', 'a<--b', 'a--|>b', 'a..>b', 'a:::b', 'a@b', 'a@{b', '<<a', 'a>>', '<<a>>b',
  // And for what Pele draws: entities, emphasis, Unicode, long labels.
  '#35;', '#quot;', '&amp;', '<br>', '😀', '中文字', 'a long label that wraps past the width a use case has',
];

// Constructs the specs touch lightly or not at all, from Pele's tests. Mermaid accepts every one.
export const SEEDS = [
  'usecase-beta\nactor A, B("Bee")@{ type: hollow } <<Role>>, "C d":::c\nA --> U\n',
  'usecase-beta\nactor A x@--> U(Use it)@{ business: true } <<S>>:::c\nx@{ animate: true }\nclass x,A,U c,d\nstyle x stroke:red\n',
  'usecase-beta\n"A b" --> "C d"\n"`E *f*`" -- "g h" --> I[Rect]\nI <-- "j" -- K\nK o-- l m -- M\nM x-- "n" -- N\nN -- o --o P\nP -- q --x Q\n',
  'usecase-beta\nA ---> B\nB <---- C\nC ----- D\nD -- e ----> F\nF -- g ---- H\nH <-- i ---- J\nJ -- "" --> K\n',
  'usecase-beta\nA ..> : include B\nB ..> : EXTEND C\nC ..> : eXtend D\nC --|> D\nactor E\nactor F\nE --|> F\n',
  'usecase-beta\nsystemBoundary S["T t"]@{\n  type: package,\n}:::c,d\n  %% in\n\n  actor A, B\n  U(Use)\n  "V w"@{ business: true } <<S>>:::c\nend\nS@{ type: rect }\n',
  'usecase-beta\nsystemBoundary "A b"\nend\nA_b@{ type: package }\nnote for U "n"\nnote for U n o p\nU\n',
  'usecase-beta\njson P@{"2":"b","1":"a","n":{"x":[1,{"y":null}],"e":{}},"n":{"z":true}}:::c\njson Q @ {}\nU --> P\nP -- Q\nclassDef c,d fill:#eef,stroke:#369\nstyle P stroke-width:4px\n',
  'usecase-beta\nA@{ type: hollow }\nactor A\nB@{\n  business: true\n\n  , "business": false\n}\nB\nactor C@{ icon: "fa:user" }\nC@{ icon: "x", type: normal }\n',
  'usecase-beta\nclassDef a,b fill:#fff,stroke-width:3px,font-family:Arial Black,-- x:1,a-b:c\\,d,x:"q" 1.5px #f00 50% (a) [b] {c} @ -- !\nA\nclass A a,b\nstyle A stroke-dasharray:5\\,5,fill:red\n',
  'usecase-beta\naccTitle: T t\naccDescr: D d\naccDescr {\n  a\n   b\n}\ndirection TD\ndirection RL\n',
  'usecase-beta\r\nactor A\r\nA --> B\r%% c\r\n',
  'usecase-beta\nA <<x>>\nA <<x>>:::c\nB(l)\nB(l)\nactor C("l")\nactor C("l")\nD <<y>> --> E\nD[D]\n',
  'usecase-beta\n1 --> 2\n1mg --> 3rd\nactor 4("x")\n5[y]\n',
  'usecase-beta\nA e@--> B\ne@{ animation: fast, animate: false }\nB --> C\nD\nC(C)\n',
];

// Text that stops where a statement is not finished, or that Mermaid's lexer gives up on.
export const REJECTED = [
  'usecase-beta\nA e@',
  'usecase-beta\nA -->',
  'usecase-beta\nA -- l',
  'usecase-beta\nA <--- l -- B',
  'usecase-beta\nA --- l --> B',
  'usecase-beta\nA <-- l --> B',
  'usecase-beta\nA o-- l --o B',
  'usecase-beta\nA x-- "l" --x B',
  'usecase-beta\nA -- "l" "m" --> B',
  'usecase-beta\nactor',
  'usecase-beta\nA@{',
  'usecase-beta\nA@{ k: v,\n\n}',
  'usecase-beta\nstyle A fill:',
  'usecase-beta\nnote for',
  'usecase-beta\nsystemBoundary S\nA',
  'usecase-beta\nA <<',
  'usecase-beta\nA <<\n\r\nB',
  'usecase-beta\nA "\'\nB',
  'usecase-beta\nA \'"\'" x\nB',
  'usecase-beta\nA "b\nC',
  'usecase-beta\njson P@{"a":',
  'usecase-beta\n"`a',
];

// Cases written here, for the corners of the grammar, the model builder and the drawing.
export const WRITTEN = [
  'usecase-beta',
  'usecase-beta\n',
  'usecase-beta x',
  '  usecase-beta\nA',
  'usecase-beta\nactor A\nactor B\nA --> U1\nB --> U1\nU1 ..> : include U2\nU3 ..> : extend U1\n',
  'usecase-beta\ndirection TB\nactor A\nA --> B\nA --> C\nA --> D\nB --> E',
  'usecase-beta\ndirection BT\nactor A\nA --> B(Bee)\nB --> C[Sea]',
  'usecase-beta\ndirection RL\nactor A\nA --> B(Bee)\nB --o C\nB --x D\nB o-- E\nB x-- F',
  'usecase-beta\ndirection TD\nactor Customer\nactor Clerk\nsystemBoundary Shop\n  Buy("Buy goods")\n  Pay\n  Refund[Refund an order]\nend\nCustomer --> Buy\nClerk --> Refund\nBuy ..> : include Pay\nRefund ..> : extend Pay',
  'usecase-beta\nsystemBoundary P["Package boundary"]@{ type: package }\n  A\n  B\nend\nA --> B',
  'usecase-beta\nsystemBoundary P\n  A\nend\nP@{ type: package }\nX --> A',
  'usecase-beta\nsystemBoundary ""\n  A\nend',
  'usecase-beta\nsystemBoundary S\nend',
  'usecase-beta\nsystemBoundary S\n  actor A\n  U\nend\nsystemBoundary T\n  V\nend\nA --> V\nA --> U\nU --> V',
  'usecase-beta\nactor H@{ type: hollow }\nactor W@{ type: awesome }\nactor I@{ icon: "fa:user" }\nactor N\nactor B@{ business: true }\nactor HB@{ type: hollow, business: true }\nH --> U\nW --> U\nI --> U\nN --> U\nB --> U\nHB --> U',
  'usecase-beta\nU(Business use)@{ business: true }\nV(Plain)\nR[Rect]\nactor A\nA --> U\nA --> V\nA --> R',
  'usecase-beta\nactor A <<Person>>\nU(Use) <<Stereo type>>\nR[Rect] <<x>>\nA --> U\nA --> R',
  'usecase-beta\nactor A\nA --> U\nU --> U\nU -- self --> U\nA --> A',
  'usecase-beta\ndirection TB\nactor A\nA --> U\nU --> U\nU -- loop label --> U',
  'usecase-beta\nactor A\nactor B\nA --|> B\nU --|> V\nA --> U',
  'usecase-beta\nA -- "label" --> B\nA -- plain words --> C\nA <-- back -- D\nA o-- circ -- E\nA x-- cross -- F\nA -- c --o G\nA -- x --x H\nA -- line --- I',
  'usecase-beta\nA ----> B\nA <---- C\nA ------ D',
  'usecase-beta\nnote for A "A note"\nactor A\nA --> U\nnote for U Plain note text',
  'usecase-beta\nnote for X "n"',
  'usecase-beta\nsystemBoundary S\nend\nnote for S "n"',
  'usecase-beta\njson Order@{"id":42,"items":[{"sku":"A1","qty":2},{"sku":"B2","qty":1}],"paid":true,"note":null}\nactor C\nC --> Order',
  'usecase-beta\njson E@{}\njson F@{"a":[]}\njson G@{"a":{}}\njson H@{"a":[1,2,3],"b":"text that is long enough to wrap in the value column of a table because it goes on"}',
  'usecase-beta\njson P@{"1":"one","0":"zero","b":2,"a":1,"10":3,"2":4}',
  'usecase-beta\njson P@{"a/b":1,"c~d":{"e":2},"":3}',
  'usecase-beta\njson P@{"a":1e400,"b":-0,"c":1.5e-7,"d":123456789012345678901234,"e":"\\ud83d\\ude00","f":"\\ud83d"}',
  'usecase-beta\njson P@{"a":}',
  'usecase-beta\njson P@{"a":1,}',
  'usecase-beta\njson P@{"position 5":x, "bbbbbbbbbbbbbbbbbbbbbbbbbbb": 1}',
  'usecase-beta\njson P@{\n  "a": 1\n  "b": 2\n}',
  'usecase-beta\njson P@{"a":"\\x"}',
  'usecase-beta\njson P@{"a":01}',
  'usecase-beta\njson P@{"a":tru}',
  'usecase-beta\njson P@{"a":"😀" x}',
  'usecase-beta\njson P@{"a":😀}',
  'usecase-beta\njson P@{"a":[1 2]}',
  'usecase-beta\n\njson P@{\r\n"a"\r\n:\r\n}',
  'usecase-beta\njson P@{"a":1}\njson P@{"b":2}',
  'usecase-beta\njson P@{"a":1}\nP --o Q',
  'usecase-beta\njson P@{"a":1}\nP ..> : include Q',
  'usecase-beta\nclassDef hot fill:#f96,stroke:#333,stroke-width:4px,color:#fff\nclassDef cold fill:#69f\nactor A:::hot\nU(Use):::cold\nA --> U\nclass U hot\nstyle A stroke:#f00,stroke-dasharray:5\\,5',
  'usecase-beta\nclassDef default fill:#eee\nA --> B\nstyle A font-weight:bold,font-size:20px\nstyle B font-style:italic',
  'usecase-beta\nA e1@--> B\nstyle e1 stroke:#f00,stroke-width:3px\nclassDef c color:#00f\nclass e1 c\ne1@{ animation: slow }',
  'usecase-beta\nA --> B\nclass edge-0 c\nstyle edge-0 stroke:red',
  'usecase-beta\nA --> B\nclass Z c',
  'usecase-beta\nA:::c --> B',
  'usecase-beta\nA --> B:::c',
  'usecase-beta\nA@{ business: true } --> B',
  'usecase-beta\nA --> B\nA@{ business: true }\nB@{ type: rect }',
  'usecase-beta\nZ@{ icon: "x" }',
  'usecase-beta\nZ@{ animate: true }',
  'usecase-beta\nZ@{ type: package }',
  'usecase-beta\nZ@{ type: hollow }',
  'usecase-beta\nZ@{ business: true }',
  'usecase-beta\nZ@{ k: v }',
  'usecase-beta\nactor A@{ icon: "i", type: hollow }',
  'usecase-beta\nactor A@{ business: true, type: awesome }',
  'usecase-beta\nR[Rect]@{ business: true }',
  'usecase-beta\nactor A\nA(A)',
  'usecase-beta\nactor A("x")\nactor A("y")',
  'usecase-beta\nU(x)\nU[x]',
  'usecase-beta\nU <<a>>\nU <<b>>',
  'usecase-beta\nsystemBoundary S\n U\nend\nsystemBoundary T\n U\nend',
  'usecase-beta\nsystemBoundary S["a"]\nend\nsystemBoundary S["b"]\nend',
  'usecase-beta\n"A b"\n"A b"',
  'usecase-beta\n"A b"\nA_b',
  'usecase-beta\nactor A\nsystemBoundary A\nend',
  'usecase-beta\nA e@--> B\nC e@--> D',
  'usecase-beta\nA e@--> B\nactor e',
  'usecase-beta\nsystemBoundary S\nend\nS --> A',
  'usecase-beta\nsystemBoundary S\nend\nA --> S',
  'usecase-beta\nA ..> : include B\nactor B',
  'usecase-beta\nactor A\nA --|> U',
  'usecase-beta\nA e@--> B\ne@{ animate: true }\ne@{ animation: x }',
  'usecase-beta\nA e@--> B\ne@{ type: hollow }',
  'usecase-beta\naccTitle: The title\naccDescr: The description\nA',
  'usecase-beta\naccDescr {\n  first line\n    second line\n\n  third\n}\nA',
  'usecase-beta\naccDescr {\nA',
  'usecase-beta\naccTitle\nA',
  '---\ntitle: A titled diagram\n---\nusecase-beta\nactor A\nA --> U',
  '---\ntitle: A title much wider than the diagram it is over, by far, for sure\n---\nusecase-beta\nA',
  '---\nconfig:\n  usecase:\n    nodeSpacing: 80\n    rankSpacing: 20\n---\nusecase-beta\nactor A\nA --> B\nA --> C',
  '%%{init: {"usecase": {"nodeSpacing": 5000, "rankSpacing": -1}}}%%\nusecase-beta\nactor A\nA --> B\nA --> C',
  'usecase-beta\nactor A("Ä Ö 日本語 😀")\nU("`**Bold** and _it_`")\nA --> U\nU -- "`*md* label`" --> V',
  'usecase-beta\nactor A("a #35; b #quot;c#quot; &amp; <b>not bold</b> \\\\n")\nA --> U("x < y > z")',
  'usecase-beta\nactor A("a very long actor name that has to wrap over several lines of text")\nA --> U("a very long use case name that has to wrap over several lines in the oval")',
  'usecase-beta\nA -- "a long relationship label that goes on and on past the wrap" --> B',
  'usecase-beta\nactor A\nA --> B\nA --> C\nA --> D\nA --> E\nA --> F\nA --> G\nB --> H\nC --> H\nD --> H',
  'usecase-beta\nactor A\nactor B\nactor C\nA --> U\nB --> U\nC --> U\nU --> V\nU --> W\nU --> X',
  'usecase-beta\nactor A\nsystemBoundary S\n  U1\n  U2\n  U3\n  U4\nend\nA --> U1\nA --> U2\nA --> U3\nA --> U4\nU1 --> U4',
  'usecase-beta\nactor A, B, C\nA --> X\nB --> X\nC --> X\nX --> A',
  'usecase-beta\nA --> B\nA --> B\nB --> A\nA -- l --> B',
  'usecase-beta\nactor 😀\n😀 --> B',
  'usecase-beta\nactor A\n\tA --> B\nB\t-->\tC',
  'usecase-beta\npackage --> B',
  'usecase-beta\nA ..> B',
  'usecase-beta\nA ..> : foo B',
  'usecase-beta\nA <<x>> B',
  'usecase-beta\nA <<>>',
  'usecase-beta\n}',
  'usecase-beta\nend',
  'usecase-beta\nA -->> B',
  'usecase-beta\nA - B',
  'usecase-beta\nA "\'',
  'usecase-beta\nstyle A',
  'usecase-beta\nstyle A fill',
  'usecase-beta\nstyle A -- x:1',
  'usecase-beta\nstyle A --:1',
  'usecase-beta\nclassDef c',
  'usecase-beta\nclass A',
  'usecase-beta\ndirection',
  'usecase-beta\ndirection XY',
  'usecase-beta\nnote for A',
  'usecase-beta\nactor A,',
  'usecase-beta\nA@{ a: 1 }',
  'usecase-beta\nA@{ a: b, }',
  'usecase-beta\nA@{ a b }',
  'usecase-beta\nA@{\n\n\n}',
  'usecase-beta\nA@{ "type": "hollow" }\nactor A',
  'usecase-beta\nsystemBoundary S\n  systemBoundary T\n  end\nend',
  'usecase-beta\nsystemBoundary S\n  A --> B\nend',
  'usecase-beta\nsystemBoundary S\n  note for A "x"\nend',
  'usecase-beta\n%% comment\nA %% not a comment\n',
  'usecase-beta\n  %% indented comment\nA',
  'usecase-beta\nA\n%%',
];

// The sources in Mermaid's specs as Pele ports them, as Pele's tests collect them.
function specSources(pele) {
  // loadCorpus('usecase', /usecase-beta/).
  const out = new Set(testStrings(pele, 'tests/compat/usecase', /\.spec\.[jt]s$/, /usecase-beta/));
  // specCorpus(): the literals in backquotes or single quotes, none of which holds ${.
  for (const sub of ['tests/compat/usecase', 'tests/compat/usecase-parser']) {
    for (const file of testFiles(pele, sub, /\.spec\.ts$/)) {
      const src = readFileSync(file, 'utf8');
      for (const m of src.matchAll(/`((?:[^`\\]|\\.)*)`|'((?:[^'\\\n]|\\.)*)'/g)) {
        const raw = m[1] ?? m[2];
        if (raw.includes('${') || raw.includes('./') || !/\W/.test(raw)) continue;
        const text = unquote(raw);
        if (text === undefined || text.length > 2000) continue;
        out.add(/^\s*usecase-beta/.test(text) ? text : 'usecase-beta\n' + text);
      }
    }
  }
  return [...out];
}

// The corpus the suites start from: documentation, specs, Pele's seeds and the cases above.
export function corpus(pele) {
  return [...new Set([...docs(pele, 'usecase'), ...specSources(pele), ...SEEDS, ...REJECTED, ...WRITTEN])].filter((s) => !lone(s));
}

// Diagrams of whole statements over a handful of ids, which mostly get through the grammar and
// then agree or clash in every way the model builder resolves, as Pele's test builds them.
export function statementDiagrams(rnd) {
  const pick = picker(rnd);
  const id = () => pick(['A', 'B', 'C', 'D', 'e', 'f', 'S', 'P', 'A_b', '__proto__']);
  const text = () => pick(['A', 'A b', 'B', 'x y', 'A-b', '']);
  const arrow = () =>
    pick(['-->', '<--', '--', '--o', 'o--', '--x', 'x--', '--->', '<---', '---', '--|>', '..> : include', '..> : extend',
      '-- l -->', '-- "l" --', '<-- l --', 'o-- l --', 'x-- l --', '-- l --o', '-- l --x', '-- "`l`" ---->', '-- "" -->', '<--- l --', '--- l -->']);
  const meta = () =>
    pick(['type: hollow', 'type: normal', 'type: awesome', 'type: package', 'type: rect', 'icon: "i"', 'business: true',
      'business: false', 'animate: true', 'animate: false', 'animation: fast', 'animation: slow', 'animation: x', 'k: v',
      '"type": "hollow"']);
  const suffix = () =>
    pick(['', '', '', `@{ ${meta()} }`, `@{ ${meta()}, ${meta()} }`, ' <<S>>', ' <<T>>', ':::c', ':::c,d', `@{ ${meta()} } <<S>>:::c`]);
  const named = (open, close) => {
    const name = id();
    return `${name}${open}${pick([name, name, text() || 'l', `"${name}"`])}${close}`;
  };
  const element = () => pick([id(), id(), named('(', ')'), named('[', ']'), `"${text()}"`, `"\`${text()}\`"`]) + suffix();
  const actor = () => pick([id(), named('(', ')'), named('[', ']'), `"${text()}"`]) + suffix();
  const statements = [
    () => `actor ${actor()}`,
    () => `actor ${actor()}, ${actor()}`,
    () => `actor ${actor()} ${arrow()} ${element()}`,
    () => element(),
    () => `${element()} ${arrow()} ${element()}`,
    () => `${element()} ${id()}@${arrow()} ${element()}`,
    () => `systemBoundary ${pick([id(), named('[', ']'), `"${text()}"`])}${pick(['', '', `@{ ${meta()} }`, ':::c'])}\n  ${element()}\n  actor ${actor()}\nend`,
    () => `systemBoundary ${id()}\n  ${element()}\nend`,
    () => `${pick([id(), `"${text()}"`])}@{ ${meta()} }`,
    () => `${id()}@{ ${meta()}, ${meta()} }`,
    () => {
      const gap = () => pick(['', ' ', ',', '\n', ',\n', '\n,', '\n\n', ',\n\n', '\n,\n', '\n\n,', '\n\n\n', ', ', ',,']);
      let block = `${pick([id(), `actor ${id()}`, `systemBoundary ${id()}`])}@{${pick(['', ' ', '\n', '\n\n'])}`;
      for (let k = Math.floor(rnd() * 4); k > 0; k--) block += meta() + gap();
      return block + (rnd() < 0.2 ? gap() : '') + '}' + (block.startsWith('system') ? '\nend' : '');
    },
    () => `note for ${id()} ${pick(['"n"', 'n o', '"`n`"'])}`,
    () => `json ${id()}@${pick(['{}', '{"a":1}', '{"b":{"c":[1,2]},"a":null}', '{"a":}', '[1]', '{"a":1,"a":{"b":2}}'])}${pick(['', ':::c'])}`,
    () => `class ${id()}${pick(['', `,${id()}`])} c${pick(['', ',d'])}`,
    () => `style ${id()} ${pick(['fill:red,stroke-width:2px', 'stroke-dasharray:5\\,5,border:1px solid red'])}`,
    () => `classDef ${pick(['c', 'd', 'default', 'c,d'])} fill:#eee,stroke:#333`,
    () => `direction ${pick(['TB', 'TD', 'BT', 'LR', 'RL'])}`,
    () => pick(['accTitle: T', 'accDescr: D', 'accDescr {\n a\n  b\n}', '%% c', '']),
  ];
  return () => {
    let src = 'usecase-beta\n';
    for (let k = 1 + Math.floor(rnd() * 7); k > 0; k--) src += pick(statements)() + '\n';
    return src;
  };
}

// Whole diagrams that Mermaid accepts, each id declared once: every kind of actor, use case,
// boundary, note, table and relationship, with labels of all lengths, styles that change the
// size of text, and the settings of front matter.
export function drawableDiagrams(rnd) {
  const pick = picker(rnd);
  const chance = (p) => rnd() < p;
  const labels = ['Customer', 'Place order', 'Pay', 'A', 'x', 'Check the stock before shipping', 'Ä Ö ü', '日本語のラベル', '😀 smile',
    'a #35; b #quot;c#quot;', 'x < y & z', 'Refund an order that was paid twice by mistake and needs care', 'Log in', 'Admin'];
  const label = () => {
    const l = pick(labels);
    const r = rnd();
    return r < 0.15 ? `"\`**${l}** _it_\`"` : r < 0.6 ? `"${l}"` : l.replace(/[^\w ]/g, '').trim().replace(/ /g, ' ') || 'L';
  };
  return () => {
    let head = '';
    if (chance(0.3)) {
      head += '---\n';
      if (chance(0.6)) head += `title: ${pick(['Orders', 'A much longer title for the use case diagram over it', 'T'])}\n`;
      if (chance(0.4)) head += `config:\n  usecase:\n    nodeSpacing: ${pick([10, 32, 60, 120])}\n    rankSpacing: ${pick([20, 72, 100])}\n`;
      head += '---\n';
    }
    const lines = ['usecase-beta'];
    if (chance(0.7)) lines.push(`direction ${pick(['LR', 'TB', 'TD', 'BT', 'RL'])}`);
    const actors = [];
    const cases = [];
    const decl = (id, l, kind) => {
      if (kind === 'actor') {
        const meta = pick(['', '', '@{ type: hollow }', '@{ type: awesome }', '@{ icon: "fa:user" }', '@{ business: true }', '@{ type: hollow, business: true }']);
        const stereo = chance(0.2) ? ` <<${pick(['Person', 'System', 'a long stereotype'])}>>` : '';
        return `actor ${id}${l ? `(${l})` : ''}${meta}${stereo}`;
      }
      const open = pick(['(', '(', '(', '[']);
      const close = open === '(' ? ')' : ']';
      const meta = open === '(' && chance(0.2) ? '@{ business: true }' : '';
      const stereo = chance(0.15) ? ` <<${pick(['UC', 'Core'])}>>` : '';
      return `${id}${open}${l || id}${close}${meta}${stereo}`;
    };
    const na = Math.floor(rnd() * 4);
    for (let i = 0; i < na; i++) {
      actors.push(`a${i}`);
      lines.push(decl(`a${i}`, chance(0.7) ? label() : '', 'actor'));
    }
    const nu = 1 + Math.floor(rnd() * 6);
    const boundaries = Math.floor(rnd() * 3);
    const inBoundary = [];
    for (let b = 0; b < boundaries; b++) inBoundary.push([]);
    for (let i = 0; i < nu; i++) {
      cases.push(`u${i}`);
      const line = decl(`u${i}`, label(), 'usecase');
      if (boundaries > 0 && chance(0.6)) inBoundary[Math.floor(rnd() * boundaries)].push(line);
      else lines.push(line);
    }
    for (let b = 0; b < boundaries; b++) {
      const title = pick([`s${b}`, `s${b}["${pick(['Shop', 'Back office', 'A system with a long name'])}"]`, `"Sys ${b}"`]);
      const meta = chance(0.3) ? '@{ type: package }' : '';
      lines.push(`systemBoundary ${title}${meta}`);
      for (const l of inBoundary[b]) lines.push('  ' + l);
      lines.push('end');
    }
    const all = [...actors, ...cases];
    const nr = Math.floor(rnd() * (all.length + 3));
    let eid = 0;
    const styled = [];
    for (let i = 0; i < nr; i++) {
      const a = pick(all);
      const b = chance(0.08) ? a : pick(all);
      const ucs = cases.includes(a) && cases.includes(b);
      const same = actors.includes(a) === actors.includes(b);
      const ops = ['-->', '-->', '--o', '--x', 'o--', 'x--', '<--', '---', '--->', '---->', '-- "label" -->', '-- some words --o', '<-- back --', 'x-- cross --', '-- a long label for the line --- '];
      if (ucs) ops.push('..> : include', '..> : extend');
      if (same && a !== b) ops.push('--|>');
      let op = pick(ops);
      let id = '';
      if (chance(0.15)) {
        id = `e${eid++} `;
        styled.push(id.trim());
        op = op.replace(/^/, '');
        lines.push(`${a} ${id.trim()}@${op} ${b}`);
      } else lines.push(`${a} ${op} ${b}`);
    }
    if (chance(0.25) && all.length > 0) lines.push(`note for ${pick(all)} ${pick(['"A note"', 'plain note text', '"`**md** note`"'])}`);
    if (chance(0.15)) {
      lines.push(`json t@${pick(['{"id":1,"items":[{"sku":"A","n":2}],"ok":true}', '{}', '{"a":[1,2,3],"b":"a longer value that wraps in its column, past the width of 260 for sure"}', '{"k":{"x":null}}'])}`);
      if (all.length > 0) lines.push(`${pick(all)} --> t`);
    }
    if (chance(0.25)) {
      lines.push(`classDef hot fill:#f96,stroke:#333,stroke-width:${pick([1, 3])}px${pick(['', ',font-weight:bold', ',font-size:22px', ',font-style:italic', ',color:#fff'])}`);
      lines.push(`class ${pick(all)} hot`);
    }
    if (chance(0.15))
      lines.push(`style ${pick(all)} fill:#eef,stroke:#00f${pick(['', ',font-size:11px,stroke-linejoin:bevel', ',font-weight:700,stroke-linecap:square'])}`);
    for (const e of styled) {
      const ends = Number(e.slice(1)) % 2 ? ',stroke-linecap:square,stroke-linejoin:bevel' : ',stroke-linecap:BUTT,stroke-linejoin:miter-clip';
      if (chance(0.5)) lines.push(`style ${e} stroke:#f00,stroke-width:2px${ends}`);
    }
    if (chance(0.1)) lines.push('accTitle: Title for screen readers', 'accDescr: A description');
    return head + lines.join('\n') + '\n';
  };
}

const JSON_SEEDS = [
  '{}',
  '{"a":1,"b":[true,false,null],"c":{"d":"e"}}',
  '{"2":"b","1":"a","n":{"x":[1,{"y":null}],"e":{}},"n":{"z":-1.5e+3}}',
  '{ "s" : "a\\"b\\\\c\\u0041\\n" , "t" : [ [ ] , { } , [ { "k" : 0 } ] ] }',
];
const JSON_FRAGMENTS = [
  '{', '}', '[', ']', ',', ':', '"', '"a"', '"a":', '1', '-', '0', '01', '1.', '.5', '1e', 'e5', '+', 'true', 'tru', 'false',
  'null', 'nul', ' ', '\n', '\t', '\\', '\\u00', '\\x', '\u0001', '{}', '[]', '"a":{}', ',,', 'x', 'é', '€', '😀', '\r\n',
];

function dumpModel(m) {
  const opt = (x) => (x === undefined || x === '' ? undefined : x);
  return {
    actors: [...m.actors.values()].map((a) => ({
      id: a.id,
      label: a.label,
      labelType: a.labelType,
      type: a.type,
      icon: opt(a.icon),
      business: a.business,
      stereotype: opt(a.stereotype),
      parentId: opt(a.parentId),
      classes: a.classes,
      styles: a.styles,
    })),
    useCases: [...m.useCases.values()].map((u) => ({
      id: u.id,
      label: u.label,
      labelType: u.labelType,
      shape: u.shape,
      business: u.business,
      stereotype: opt(u.stereotype),
      parentId: opt(u.parentId),
      classes: u.classes,
      styles: u.styles,
    })),
    boundaries: [...m.systemBoundaries.values()],
    relationships: m.relationships.map((r) => ({ ...r, animation: opt(r.animation) })),
    notes: [...m.notes.values()],
    jsonNodes: [...m.jsonNodes.values()].map((j) => ({
      id: j.id,
      value: enc(j.value),
      propertyOrder: [...j.propertyOrder.entries()],
      classes: j.classes,
      styles: j.styles,
    })),
    classDefs: [...m.classDefs.values()],
    direction: m.direction,
    title: opt(m.title),
    accTitle: m.accTitle,
    accDescr: m.accDescr,
    accDescription: m.accDescription,
    headerSpan: m.headerSpan,
    statements: JSON.parse(JSON.stringify(m.statements)),
  };
}

function dumpGraph(g) {
  const opt = (x) => (x === undefined || x === '' ? undefined : x);
  return {
    nodes: g.nodes.map((n) => ({
      id: n.id,
      label: n.label,
      labelType: n.labelType,
      shape: n.shape,
      isGroup: n.isGroup,
      padding: n.padding,
      cssClasses: n.cssClasses,
      cssStyles: n.cssStyles,
      cssCompiledStyles: n.cssCompiledStyles,
      actorType: opt(n.actorType),
      business: n.business,
      icon: opt(n.icon),
      stereotype: opt(n.stereotype),
      parentId: opt(n.parentId),
      noteTarget: opt(n.noteTarget),
      jsonRows: n.jsonRows,
      boundaryType: opt(n.boundaryType),
    })),
    edges: g.edges.map((e) => ({ ...e, label: opt(e.label), labelType: opt(e.labelType), animation: opt(e.animation) })),
  };
}

export default async function ({ src, pele, write, random }) {
  const located = await locator(src);
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const { usecase } = await src('diagrams/usecase/index.ts');
  const { parseUsecase } = await src('diagrams/usecase/parser.ts');
  const { buildUsecaseGraph } = await src('diagrams/usecase/graph.ts');

  const caught = (e) => {
    if (e instanceof RangeError) return { error: { code: 'limit', message: 'Diagram is nested too deeply.' } };
    if (!(e instanceof PeleError)) throw e;
    return { error: peleError(e) };
  };
  const parsed = (parseIt) => {
    try {
      const model = parseIt();
      return { model: dumpModel(model), graph: dumpGraph(buildUsecaseGraph(model)) };
    } catch (e) {
      return caught(e);
    }
  };
  // Pele's parseSource for a use case diagram, and the raw text given to the parser alone.
  const run = (source) => {
    const out = { src: source, direct: parsed(() => parseUsecase(source)) };
    let pre;
    try {
      pre = preprocess(source);
      out.type = detect(pre.text);
    } catch (e) {
      return { ...out, ...caught(e) };
    }
    if (out.type !== 'usecase') return out;
    const piped = parsed(() => located(source, false, () => usecase.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title)));
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
  // on a different error than one cut off by a line break.
  const rnd = random(7);
  const next = mutator(base, FRAGMENTS, rnd);
  for (let i = 0; i < 2500; i++) {
    let s = next();
    if (i % 6 === 0) s = s.slice(0, Math.floor(rnd() * (s.length + 1)));
    add(s);
  }
  const diagrams = statementDiagrams(random(10));
  for (let i = 0; i < 1500; i++) add(diagrams());
  const drawable = drawableDiagrams(random(12));
  for (let i = 0; i < 1000; i++) add(drawable());
  // Random strings of tokens, where keyword boundaries and the lookahead matter.
  const trnd = random(8);
  const pieces = [...new Set([...FRAGMENTS, ...FRAGMENTS.map((f) => f.trim()).filter((f) => f !== '')])];
  const openers = ['usecase-beta\n', 'usecase-beta\nactor ', 'usecase-beta\nA ', 'usecase-beta\nA@{', 'usecase-beta\nsystemBoundary S\n',
    'usecase-beta\nstyle A ', 'usecase-beta\nA -- ', 'usecase-beta\njson P@', ''];
  for (let i = 0; i < 800; i++) {
    let s = openers[Math.floor(trnd() * openers.length)];
    for (let k = 1 + Math.floor(trnd() * 10); k > 0; k--) s += pieces[Math.floor(trnd() * pieces.length)] + (trnd() < 0.3 ? ' ' : '');
    add(s);
  }
  // JSON, mutated.
  const jnext = mutator(JSON_SEEDS, JSON_FRAGMENTS, random(9));
  for (let i = 0; i < 600; i++) add(`usecase-beta\njson P@${jnext()}\n`);

  write('diagram/usecase/testdata/pele/usecase.json', { cases: all.map(run) });
}
