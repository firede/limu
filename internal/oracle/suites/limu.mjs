// What Pele's parse and render give a diagram's text as a whole: its type as Pele and as
// Mermaid name it, or the error, which holds them once the type is known (tagErrors). The
// sources are the parse suite's, the text of every type, and those with pieces put in at
// random, which mostly do not parse.
import { lone } from '../lib/corpus.mjs';
import { mutator } from '../lib/mutate.mjs';

const FRAGMENTS = [
  '\n', ' ', '-->', '---', '[', ']', '(', ')', '{', '}', '"', ':', ';', '|', '%%', 'end', 'A', '1', '-', '>', '<',
  'class ', 'title ', 'section ', '#', '@', ',', '=', '\t', 'é', '😀',
];

const errorOf = (e) => ({ code: e.code, message: e.message, type: e.type, mermaidType: e.mermaidType });

export default async function ({ src, cases, write, random }) {
  const { parse, render, PeleError } = await src('index.ts');
  const { metricsMeasurer } = await src('text/measurer.ts');
  const c = cases('parse');
  const next = mutator(c.extracted, FRAGMENTS, random(20261020));
  const fuzz = [];
  while (fuzz.length < 1500) {
    const s = next();
    if (!lone(s) && s.length <= 6000) fuzz.push(s);
  }
  // The error of run, or undefined where it has none.
  const caught = (run) => {
    try {
      run();
      return undefined;
    } catch (e) {
      if (!(e instanceof PeleError)) throw e;
      return errorOf(e);
    }
  };
  const out = [...c.extracted, ...fuzz].map((source) => {
    let model, result;
    const error =
      caught(() => (model = parse(source))) ??
      caught(() => (result = render(model, { measurer: metricsMeasurer, outputLimit: Infinity })));
    if (error) return { source, parsed: model !== undefined, error };
    return { source, type: result.type, mermaidType: model.mermaidType };
  });
  write('testdata/pele/limu.json', { cases: out });
}
