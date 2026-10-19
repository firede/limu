// Text measured with Pele's table of Arial's advances, and labels laid out.

// A random but repeatable mix of what labels are made of, to reach what the written cases miss.
function* randomLabels(next, count) {
  const bits = [
    'word', 'Word', 'longerword', 'a', 'I', ' ', ' ', ' ', '  ', ' ', '　', '-', ',', '.', '!', '?', '(', ')',
    '中文', '汉字', '。', '、', '（', '）', '「', '」', 'ー', 'カタカナ', 'ひらがな', 'っ', '한국어', '단어',
    '<b>', '</b>', '<i>', '</i>', '<strong>', '</em>', '<br>', '<br/>', '<BR />', '\\n', '\n', '<span class="x">', '</span>',
    '**', '*', '_', '__', 'snake_case', '&amp;', '&lt;', '&#35;', '&#x4e2d;', '&nbsp;', '&bogus;', 'ﬂ°°35¶ß', 'ﬂ°quot¶ß',
    'fa:fa-car', 'fab:fa-github', 'fa:fa-missing', 'é', 'é', '😀', '𠀀', 'Ω', '\t',
  ];
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

export default async function ({ src, cases, write, random }) {
  const { metricsMeasurer } = await src('text/measurer.ts');
  const { layoutLabel } = await src('text/label.ts');
  const { withIcons } = await src('text/icons.ts');
  const c = cases('text');
  const measure = c.measure.map((m) => ({ ...m, width: metricsMeasurer.width(m.text, m.size, m.face) }));
  const all = [...c.label, ...randomLabels(random(20261019), 400)];
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
