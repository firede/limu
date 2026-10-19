// Kanban boards drawn end to end: what Pele's render draws, read back out of its SVG as one
// line per thing drawn, in the order drawn. limu's test writes the same lines from its scene.
//
// Pele writes a column's insides about the column, and a card's about the card, where a scene
// holds where everything is on the page. Pele's SVG is drawn here with its numbers written in
// full, as the oracle's hooks have num() write them while limuExactNumbers is set, and with
// each line of a column's or a card's text written as where it starts or ends and what is in
// it. A number inside groups is moved as limu's layout moves it: the groups' moves are added
// up from the outermost in, the number is added to their sum, and only then is it written to
// two decimals as Pele writes it.
import { lone, mutator } from './parse.mjs';
import { FRAGMENTS, boards, corpus, hostileCorpus } from './kanban.mjs';

// The icons the host has, in the cases drawn with icons.
export const ICONS = ['star', 'inbox', 'phone', 'fa:fa-car', 'fa fa-x'];

const unescape = (s) => s.replace(/&(amp|lt|gt|quot);/g, (_, e) => ({ amp: '&', lt: '<', gt: '>', quot: '"' })[e]);
const attr = (s, name) => {
  const m = new RegExp(`\\s${name}="([^"]*)"`).exec(s);
  return m ? unescape(m[1]) : undefined;
};
const TRANSLATE = /translate\(([^,]*),([^)]*)\)/;

// The lines of a board's SVG, drawn with its numbers in full; num writes a number as Pele does.
export function readSvg(svg, num) {
  const lines = [];
  const moves = [];
  const move = (v, axis) => {
    if (moves.length === 0) return v;
    let by = moves[0][axis];
    for (let k = 1; k < moves.length; k++) by = by + moves[k][axis];
    return by + v;
  };
  const pt = (x, y) => `${num(move(Number(x), 0))},${num(move(Number(y), 1))}`;
  const head = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  lines.push(`size ${num(Number(attr(head, 'width')))},${num(Number(attr(head, 'height')))}`);
  const title = svg.indexOf('<text class="pele-title"');
  if (title >= 0) {
    const m = / x="([^"]*)" y="([^"]*)"/.exec(svg.slice(title));
    lines.push(`title ${pt(m[1], m[2])}`);
  }
  const open = [];
  // Inside an icon, whose drawing is the host's.
  let icon = false;
  const from = svg.indexOf('<g class="pele-cluster');
  if (from < 0) return lines;
  for (const m of svg.slice(from).matchAll(/<(\/?)(g|rect|path|text|svg|limu-line|a)\b[^>]*>/g)) {
    const e = m[0];
    if (m[1]) {
      if (m[2] === 'g' && open.pop()) moves.pop();
      if (m[2] === 'svg') icon = false;
      continue;
    }
    if (icon) continue;
    switch (m[2]) {
      case 'g': {
        const cls = attr(e, 'class') ?? '';
        if (/^pele-cluster pele-column/.test(cls)) lines.push(`column ${attr(e, 'data-id')} [${cls.split(' ').slice(2).join(' ')}]`);
        else if (/^pele-node pele-card/.test(cls)) lines.push(`card ${attr(e, 'data-id')} [${cls.split(' ').slice(2).join(' ')}]`);
        else lines.push(`group ${cls}`);
        const t = TRANSLATE.exec(attr(e, 'transform') ?? '');
        if (t) moves.push([Number(t[1]), Number(t[2])]);
        open.push(t !== null);
        break;
      }
      case 'rect':
        lines.push(
          `rect ${pt(attr(e, 'x') ?? 0, attr(e, 'y') ?? 0)},${num(Number(attr(e, 'width')))},${num(Number(attr(e, 'height')))}` +
            `${attr(e, 'rx') !== undefined || (attr(e, 'style') ?? '').includes('rx:') ? ' r' : ''} ${attr(e, 'fill') ?? '-'} ${attr(e, 'stroke') ?? '-'}`
        );
        break;
      case 'path': {
        const d = attr(e, 'd');
        const p = /^M([^,]*),([^V]*)V(.*)$/.exec(d);
        lines.push(`priority M ${pt(p[1], p[2])} L ${pt(p[1], p[3])} ${attr(e, 'stroke')} ${attr(e, 'opacity') ?? '-'} ${attr(e, 'stroke-width')}`);
        break;
      }
      case 'text':
        lines.push(`count ${pt(attr(e, 'x'), attr(e, 'y'))} ${attr(e, 'fill') ?? '-'} ${attr(e, 'font-size')}`);
        break;
      case 'svg':
        icon = !e.endsWith('/>');
        lines.push(`icon ${attr(e, 'data-icon')} ${pt(attr(e, 'x'), attr(e, 'y'))},${num(Number(attr(e, 'width')))},${num(Number(attr(e, 'height')))}`);
        break;
      case 'a':
        lines.push(`link ${attr(e, 'href')}`);
        break;
      case 'limu-line': {
        const attrs = decodeURIComponent(attr(e, 'attrs'));
        const cls = /class="([^"]*)"/.exec(attrs)[1];
        const fill = /fill="([^"]*)"/.exec(attrs)?.[1] ?? '-';
        lines.push(
          `line ${cls} ${fill} ${pt(attr(e, 'x'), attr(e, 'y'))} ${attr(e, 'end') === '1' ? 'end' : 'start'} ${attr(e, 'w')} ${decodeURIComponent(attr(e, 'spans'))}`
        );
        break;
      }
    }
  }
  return lines;
}

export default async function ({ src, pele, write, random }) {
  const { render, PeleError } = await src('index.ts');
  const { metricsMeasurer } = await src('text/measurer.ts');
  const { num } = await src('svg/builder.ts');

  const out = [];
  const host = (name) => (ICONS.includes(name) ? '<path d="M0,0"/>' : undefined);
  const draw = (source, maxWidth, fontSize, icons) => {
    let r;
    globalThis.limuExactNumbers = true;
    try {
      r = render(source, { measurer: metricsMeasurer, maxWidth, fontSize, icons: icons ? host : undefined, outputLimit: Infinity });
    } catch (e) {
      if (e instanceof RangeError) return false;
      if (!(e instanceof PeleError)) throw e;
      return false;
    } finally {
      globalThis.limuExactNumbers = false;
    }
    if (r.type !== 'kanban') return false;
    out.push({ src: source, maxWidth, fontSize, icons: icons || undefined, lines: readSvg(r.svg, num) });
    return true;
  };
  const seen = new Set();
  let drawn = 0;
  const narrow = [360, 480, 200, 100, 40];
  const each = (source) => {
    if (seen.has(source) || lone(source) || source.length > 4000) return;
    seen.add(source);
    if (!draw(source, undefined, undefined, drawn % 4 === 1)) return;
    // Some again in a narrow host, which folds the board into rows, and some in another size
    // of type.
    if (drawn % 3 === 0) draw(source, narrow[(drawn / 3) % narrow.length], undefined, drawn % 2 === 0);
    if (drawn % 7 === 0) draw(source, undefined, drawn % 14 === 0 ? 13 : 13.5);
    drawn++;
  };
  const base = corpus(pele);
  base.forEach(each);
  hostileCorpus(pele).forEach(each);
  const board = boards(random(31));
  for (let i = 0; i < 1500; i++) each(board());
  const usable = [...seen].filter((s) => s.length <= 3000);
  const next = mutator(usable, FRAGMENTS, random(32));
  while (drawn < 3500) each(next());
  write('diagram/kanban/testdata/pele/render.json', { cases: out });
}
