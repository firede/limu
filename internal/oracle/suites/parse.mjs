// What every diagram's text goes through before its own parser: front matter, directives and
// comments read (preprocess), the type found (detect), entity codes hidden (encodeEntities),
// and the limit on a source's length.
import { enc, peleError } from '../lib/value.mjs';
import { lone } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';

const FRAGMENTS = [
  '---\n', '---', '--- \n', '  ---\n', '\n', '\r\n', '\r', '%%', '%% c\n', '%%{', '}%%', '%%{init: {"theme": "dark"}}%%\n',
  "%%{init: {'flowchart': {'curve': 'basis', 'rankSpacing': 9}}}%%", '%%{wrap}%%', '%%{ initialize : {"a": [1, {"b": 2}]} }%%',
  '%%{init: {"__proto__": {"x": 1}, "a": {"b": 1}}}%%', '%%{init: not json}%%', 'title: x\n', 'title: [a, b]\n', 'title: 0\n',
  'displayMode: compact\n', 'config:\n  a: 1\n  b:\n    c: [1, 2]\n', 'config:\n  flowchart:\n    curve: linear\n', '  ', '\t',
  '<span class="a">', '<b class="x" id="y">', '<br/>', '"', "'", ':', '#', '#35;', '#quot;', '#9829;', 'style a fill:#f9f;',
  'classDef c stroke:#333;', 'accTitle: x', 'accDescr: y', ' ', '\u2028', '\u3000', 'é', '😀', 'graph', 'flowchart', 'C4Container',
  'ishikawa-BETA', 'railroad-ebnf-beta', 'cynefin-beta:', 'usecase-beta ', 'swimlane-beta', 'agentflow-beta',
];

const WRITTEN = [
  '',
  '   ',
  'graph TD\nA-->B',
  '\n\n  graph TD\nA',
  '%% one\ngraph TD\n  %% two\n  A --> B\n',
  '%%{init: {}}%%\ngraph TD',
  '---\ntitle: My chart\nconfig:\n  flowchart:\n    curve: linear\n---\ngraph TD\n  A',
  "---\nconfig:\n  flowchart:\n    curve: linear\n    rankSpacing: 10\n---\n%%{init: {'flowchart': {'rankSpacing': 99}}}%%\ngraph TD\n  A",
  '%%{init: {"__proto__": {"polluted": true}, "constructor": {"prototype": {"polluted": true}}}}%%\ngraph TD\nA',
  '---\nconfig:\n  __proto__:\n    polluted: true\n---\ngraph TD\nA',
  'graph TD\r\nA["<span class="x">hi</span>"]\r',
  '  ---\n  title: indented\n  config:\n    a: 1\n  ---\ngraph TD',
  '---\ntitle: a\n---\n---\ntitle: b\n---\ngraph TD',
  '---\n---\ngraph TD',
  '---\ntitle: [unclosed\n---\ngraph TD',
  '---\ntitle: "\\x4g"\n---\ngraph TD',
  '---\ntitle: "\\u-001"\n---\ngraph TD',
  '---\ntitle: 1e21\ndisplayMode: 5\n---\ngantt',
  '---\ntitle: 0.000001\n---\npie',
  '---\ntitle: 1.5e-7\n---\npie',
  '---\ntitle: [[1, 2], null, {a: 1}]\n---\npie',
  '---\ntitle: .inf\n---\npie',
  '---\ntitle: -0\n---\npie',
  '---\ntitle: []\n---\npie',
  '---\ndisplayMode: x\nconfig:\n  gantt: 3\n---\ngantt',
  '---\ndisplayMode: x\nconfig:\n  gantt:\n    a: 1\n---\ngantt',
  '%%{wrap}%%\nsequenceDiagram',
  '%%{init: {"a": 1e400, "b": -0, "c": "\\ud83d\\ude00", "d": [1, {"e": null}]}}%%\npie',
  '%%{init: {"a": 1}}%% %%{initialize: {"b": {"c": 2}}}%%\n%%{init: {"b": {"d": 3}}}%%\nclassDiagram',
  "%%{init: {'theme':'forest'}}%%\r\n%%{init:\n{\"x\": 1}\n}%%\nerDiagram",
  '%%{init: { "a": 1 }\ngraph TD',
  '%%{ init }%%\ngraph TD',
  '%%{\u3000init\u3000:\u3000{"a":1}\u3000}%%\ngraph TD',
  '%%{init: {"a": "}%%"}}%%\ngraph TD',
  '%%{init: {"a": 1}\u2028}%%graph TD',
  'C4Context\ntitle x',
  'title C4Component\ngraph',
  'ishikawa-Beta\nx',
  'ishikawa-betax',
  'ISHIKAWA',
  'railroad-PEG-beta',
  'railroad-beta',
  'railroad-xyz-beta',
  'cynefin-beta',
  'cynefin-beta:',
  'cynefin-betax',
  'usecase-beta\n',
  'usecase-betax',
  'agentflow-beta TB',
  'agentflow-betaTB',
  'swimlane-beta\n',
  'flowchart-elk TD',
  'requirementDiagram',
  'sankey-beta',
  'xychart-beta',
  'block-beta',
  'packet-beta',
  'treeView-beta',
  'radar-beta',
  'venn-beta',
  'wardley-BETA',
  'eventmodeling',
  'treemap',
  'mindmap',
  'architecture-beta',
  'kanban',
  'info',
  'timeline',
  'gitGraph',
  'stateDiagram-v2',
  'journey',
  'quadrantChart',
  'gantt',
  'pie',
  'erDiagram',
  'classDiagram',
  'sequenceDiagram',
  '\u00a0\u3000graph TD',
  '\ufeffgraph TD',
  'unknown',
  'graph TD\nstyle A fill:#f9f,stroke:#333;',
  'graph TD\nstyle A fill:#f9f;stroke:#333;\nB --> C;',
  'graph TD\nclassDef green fill:#9f6,stroke:#333,stroke-width:2px;',
  'graph TD\nA["a #35; b #quot;c#quot;"]',
  'graph TD\nmystyle x:#1; classDef y:#2; z;',
  'style a:#b\n;',
  'style a b :c#d e;f;g',
  '#;# ; #a ;#a;',
  'graph TD\nstyle A\u2028fill:#f;x;',
  'graph TD\n  %%\n  %%{x}\n  %% a\n\n   %% b\nA',
  'graph TD\n%% c\u2028%% d\nA',
  '<b class="a" id="b">x</b> <i data-x="y>',
  '<a\n href="x">',
  'x > y < z',
];

export default async function ({ src, cases, write, random }) {
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { detect } = await src('detect.ts');
  const { PeleError } = await src('errors.ts');
  const c = cases('parse');
  const next = mutator(c.extracted, FRAGMENTS, random(20261019));
  const fuzz = [];
  while (fuzz.length < 1200) {
    const s = next();
    if (!lone(s) && s.length <= 6000) fuzz.push(s);
  }
  const all = [...WRITTEN, ...c.extracted, ...fuzz];
  const preprocessed = all.map((source) => {
    try {
      const pre = preprocess(source);
      return {
        source,
        text: pre.text,
        // Left out where it is the text itself.
        withComments: pre.withComments === pre.text ? undefined : pre.withComments,
        title: pre.title,
        config: enc(pre.config, true),
        lineOffset: pre.lineOffset,
        type: detect(pre.text),
      };
    } catch (e) {
      if (!(e instanceof PeleError)) throw e;
      return { source, error: peleError(e) };
    }
  });

  // Mermaid's lines of style, with colors and entity codes, at random.
  const rnd = random(11);
  const parts = ['style', 'classDef', ':', '#', ';', ' ', '\n', 'a', 'fill', '9', '\t', ':#', '#f9f;', '\r', '#35;', '#quot;', '\u2028', 'é', '+1'];
  const lines = [];
  for (let i = 0; i < 3000; i++) {
    let text = '';
    const count = Math.floor(rnd() * 14);
    for (let k = 0; k < count; k++) text += parts[Math.floor(rnd() * parts.length)];
    lines.push(text);
  }
  const encoded = [...all.filter((s) => s.includes('#')), ...lines].map((text) => ({ text, encoded: encodeEntities(text) }));
  write('diagram/testdata/pele/parse.json', { preprocessed, encoded });
}
