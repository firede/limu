// Writes what Pele makes of the cases in ./cases as fixtures for limu's tests to compare with.
//
//   node --experimental-transform-types --no-warnings --import ./register.mjs run.mjs <pele> <limu>
//
// scripts/oracle.sh runs it.
import { execFileSync } from 'node:child_process';
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';

const [pele, limu] = process.argv.slice(2);
if (!pele || !limu) {
  console.error('usage: run.mjs <pele> <limu>');
  process.exit(2);
}
const src = (path) => import(join(pele, 'src', path));

let commit = '';
try {
  commit = execFileSync('git', ['-C', pele, 'rev-parse', 'HEAD'], { encoding: 'utf8' }).trim();
} catch {}
const { version } = JSON.parse(readFileSync(join(pele, 'package.json'), 'utf8'));
const origin = { version, commit };

function write(path, data) {
  const out = join(limu, path);
  mkdirSync(dirname(out), { recursive: true });
  writeFileSync(out, JSON.stringify({ pele: origin, ...data }, null, 1) + '\n');
  console.log(`wrote ${path}`);
}

function cases(name) {
  return JSON.parse(readFileSync(new URL(`./cases/${name}.json`, import.meta.url), 'utf8'));
}

// A random but repeatable mix of what labels are made of, to reach what the written cases miss.
function* randomLabels(count, seed) {
  const bits = [
    'word', 'Word', 'longerword', 'a', 'I', ' ', ' ', ' ', '  ', ' ', '　', '-', ',', '.', '!', '?', '(', ')',
    '中文', '汉字', '。', '、', '（', '）', '「', '」', 'ー', 'カタカナ', 'ひらがな', 'っ', '한국어', '단어',
    '<b>', '</b>', '<i>', '</i>', '<strong>', '</em>', '<br>', '<br/>', '<BR />', '\\n', '\n', '<span class="x">', '</span>',
    '**', '*', '_', '__', 'snake_case', '&amp;', '&lt;', '&#35;', '&#x4e2d;', '&nbsp;', '&bogus;', 'ﬂ°°35¶ß', 'ﬂ°quot¶ß',
    'fa:fa-car', 'fab:fa-github', 'fa:fa-missing', 'é', 'é', '😀', '𠀀', 'Ω', '\t',
  ];
  let s = seed >>> 0;
  const next = () => (s = (Math.imul(s, 1664525) + 1013904223) >>> 0) / 2 ** 32;
  for (let n = 0; n < count; n++) {
    let text = '';
    const parts = 1 + Math.floor(next() * 14);
    for (let k = 0; k < parts; k++) text += bits[Math.floor(next() * bits.length)];
    yield {
      text,
      markdown: next() < 0.4,
      size: [12, 14, 16, 13.5][Math.floor(next() * 4)],
      maxWidth: [30, 60, 100, 200, 4000][Math.floor(next() * 5)],
      face: Math.floor(next() * 8),
      icons: next() < 0.5 ? ['fa:fa-car', 'fab:fa-github'] : [],
    };
  }
}

async function text() {
  const { metricsMeasurer } = await src('text/measurer.ts');
  const { layoutLabel } = await src('text/label.ts');
  const { withIcons } = await src('text/icons.ts');
  const c = cases('text');
  const measure = c.measure.map((m) => ({ ...m, width: metricsMeasurer.width(m.text, m.size, m.face) }));
  const all = [...c.label, ...randomLabels(400, 20261019)];
  const label = all.map((l) => {
    const icons = new Set(l.icons ?? []);
    const out = withIcons(
      (name) => (icons.has(name) ? '<path/>' : undefined),
      () => layoutLabel(l.text, l.markdown, metricsMeasurer, l.size, l.maxWidth, l.face)
    );
    return { ...l, label: out };
  });
  write('text/testdata/pele/text.json', { measure, label });
}

await text();
