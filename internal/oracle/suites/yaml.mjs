// Pele's YAML reader, on what Mermaid feeds it and on YAML made at random.
import { enc, lone } from './parse.mjs';

const WRITTEN = [
  // Pele's own cases, tests/util/yaml.test.ts.
  '{\n shape: rounded \n}',
  '{\nshape: rounded , label: "DD"\n}',
  '{\n label: "This is }", other: "clock"\n}',
  '{\n animate: true, animation: fast, w: 100, h: 50.5, x: null, y: ~\n}',
  '{\n label: \'single quoted\', icon: "fa:fa-user", form: circle, pos: t \n}',
  '{\n a: [1, 2, three], b: {c: d, e: [f]}\n}',
  '{\n}',
  '\n        shape: circle\n        other: "clock"\n     \n',
  '\n        label: |\n          This is a\n          multiline string\n        other: "clock"\n     \n',
  '\n        label: >\n          folded\n          text\n\n          second paragraph\n        other: 1\n',
  '\n        label: "This is a<br/>multiline string"\n        other: "clock"\n',
  'title: Hello world\nconfig:\n  theme: dark\n  flowchart:\n    curve: linear\n    htmlLabels: false\n',
  'title: "Quoted: title"\ndisplayMode: compact\n',
  'config:\n  themeVariables:\n    primaryColor: "#ff0000"\n    fontSize: 16px\n  look: handDrawn # trailing comment\n',
  'list:\n  - a\n  - b: 1\n    c: 2\n  - [x, y]\nempty:\n',
  'a: 0x1f\nb: -12\nc: 1e3\nd: .5\ne: "12"\nf: True\ng: FALSE\nh: Null\n',
  'key with spaces: plain value\nurl: http://example.com/a#b\n',
  'multi: this is\n  continued on the next line\nnext: 1\n',
  '# only a comment\n',
  '',
  'just a scalar',
  '- 1\n- 2\n',
  'a: {\n  b: 1,\n  c: [2,\n    3]\n}\nd: "two\n  lines"\ne: 5\n',
  '{\n a: 1, # comment\n b: 2\n}',
  '__proto__: 1\nb:\n  __proto__: {x: 1}\n',
  '{ __proto__: { polluted: true } }',
  '{ a: 1',
  'a: {\n' + ' b: 1,\n'.repeat(50),
  '{ a: ' + '['.repeat(70) + ' }',
  // Keys that are array indices come first, in order.
  'b: 1\n2: 2\na: 3\n1: 4\n01: 5\n4294967295: 6\n4294967294: 7\n-1: 8\n1.0: 9\n',
  '{z: 1, 10: 2, 9: 3, "": 4}',
  // Escapes, and what String.fromCodePoint makes of them.
  '"\\x41\\u00e9\\U0001F600\\ud83d\\ude00\\t\\N\\_\\L\\P\\0\\e\\/"',
  '"\\x4g"',
  '"\\xé1"',
  '"\\u-001"',
  '"\\U00110000"',
  '"\\uzz"',
  '"\\q"',
  '"\\',
  '"a\\\n   b"',
  '"line\n\n  two\n   three"',
  "'it''s\n  folded'",
  // Numbers.
  'a: 1.\nb: +.5\nc: -0\nd: 007\ne: 1e400\nf: -.INF\ng: .NaN\nh: 0o17\ni: -0b101\nj: +0x1F\nk: 0xFFFFFFFFFFFFFFFFFF\nl: 1_000\nm: 0x\nn: 1e\no: 12345678901234567890\n',
  // Block scalars.
  'a: |-\n  x\n\n\nb: |+\n  y\n\n\nc: >2\n    z\n   w\nd: |\n\ne: >\n  a\n   indented\n  b\n',
  'a: |\n\u3000\u3000x\n\u3000\u3000y\n',
  'a: |\n  x\n\u3000\u3000\u3000y\n',
  '- |\n  a\n- >-\n  b\n  c\n',
  // Sequences and maps nested.
  '- - a\n  - b\n- c: d\n  e:\n    - f\n',
  'a:\n- 1\n- 2\nb: 3\n',
  'a:\n  - x: 1\n    y: 2\n  -\n    z: 3\n',
  '- \u3000a: 1\n  b: 2\n',
  // Errors.
  'a: 1\n  b: 2\n',
  'a: 1\n\tb: 2\n',
  '\ta: 1',
  'a: [1, 2] x',
  'a: "x" y',
  '[a b]',
  '{a: 1,, b: 2}',
  '[1 2]',
  'a: 1\nb\n',
  'a: 1\n...\n',
  'a: 1\n---\nb: 2\n',
  '- a\nb: 1\n',
  '"key": 1\n\'k2\': 2\n"k\\"3" : 3\n',
  '"unterminated: 1',
  "a: 'x\n  y'\nb: 'z",
  '? a\n',
  'a:b\n',
  'a :b\n',
  'a#b: 1\n',
  '-a: 1\n',
  '- : x\n',
  'a: b # c\n  d\n',
  'a: b\n\n\n  c\n',
  'a: x #c\n',
  'a: "x" # c\n',
  '{a: x#y, b: x #y\n}',
  '{a: b: c}',
  '{a: http://x.y/z, b: [c:d]}',
  '[a, {b: c}, [d]]x',
  '[a]\n[b]',
  '{a: 1}\n',
  '  {a: 1}  ',
  '{\n a: "multi\n line" ,\n b: [1,\n  2]\n}',
  'a: [\n  1,\n  2\n]\nb: {\n  c: "d\n e"\n  }\n',
  'a: [\n  1,\n',
  "a: 'x' 'y'\n",
  'a: |x\n  b\n',
  'a: | # c\n  b\n',
  'a: >-+\n  b\n',
  'a: |\n',
  '\u3000a: 1',
  'a\u3000: 1',
  'a: \u3000b\u3000',
  'é: ü\n😀: [😀, "😀"]\n',
  'a:\u2028b: 1',
];

const FRAGMENTS = [
  'a', 'b', 'key', ':', ': ', ': ', '- ', '-', '\n', '\n', '\n', '  ', '  ', '    ', '\t', '#', ' #c', '"', "'", '\\', '\\x41', '\\u00e9',
  '\\U0001F600', '\\ud83d', '\\q', '{', '}', '[', ']', ',', ', ', '|', '>', '|-', '>+', '|2', '~', 'null', 'true', 'False',
  '0x1F', '-0o17', '0b101', '1e400', '.inf', '-.Inf', '.NaN', '12', '-3.5', '.5', '1.', '007', '__proto__', '1', '42',
  'key with space', '...', '---', '\r\n', '\r', '\u3000', 'é', '😀', '\u2028', '? ', '&a', '*a', '!tag', '%', '@', '`',
];

export default async function ({ src, write, random }) {
  const { parseYaml } = await src('util/yaml.ts');
  const next = random(7);
  const fuzz = [];
  while (fuzz.length < 3000) {
    let s = '';
    const n = 1 + Math.floor(next() * 24);
    for (let k = 0; k < n; k++) s += FRAGMENTS[Math.floor(next() * FRAGMENTS.length)];
    if (!lone(s)) fuzz.push(s);
  }
  const yaml = [...WRITTEN, ...fuzz].map((source) => {
    try {
      return { source, value: enc(parseYaml(source)) };
    } catch (e) {
      return { source, error: e.message };
    }
  });
  write('internal/yaml/testdata/pele/yaml.json', { yaml });
}
