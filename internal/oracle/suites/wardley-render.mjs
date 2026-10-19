// Wardley maps drawn end to end: what Pele's render draws, read back out of its SVG as one
// line per thing drawn, in the order drawn. limu's test writes the same lines from its scene.
//
// Pele draws the map in a group it moves onto the page, where a scene holds every place on
// the page. Pele's SVG is drawn here with its numbers written in full, as the oracle's hooks
// have num() write them while limuExactNumbers is set, and with each label and marker
// written as the point it is drawn about, as the hooks have Pele's wardley render write them.
// A number inside the group is moved as limu's layout moves it, the group's move added to
// it, and the lines hold every number in full, as String() writes it, so that a difference
// in the last bit shows. A label whose lines start at a point is written where limu centers
// it: its first baseline less a little over a third of its size, and down by half of what
// its lines after the first take.
import { lone, mutator } from './parse.mjs';
import { FRAGMENTS, corpus, hostile, randomMaps } from './wardley.mjs';

// The icons the host has, in the cases drawn with icons.
export const ICONS = ['star', 'fa:fa-car', 'fa:fa-coffee'];

const unescape = (s) => s.replace(/&(amp|lt|gt|quot);/g, (_, e) => ({ amp: '&', lt: '<', gt: '>', quot: '"' })[e]);
const attr = (s, name) => {
  const m = new RegExp(`\\s${name}="([^"]*)"`).exec(s);
  return m ? unescape(m[1]) : undefined;
};
const n = (v) => String(v);
const FACES = ['-', 'b', 'i', 'bi'];

// The lines of a map's SVG, drawn with its numbers in full.
export function readSvg(svg) {
  const lines = [];
  const head = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  lines.push(`size ${attr(head, 'width')},${attr(head, 'height')}`);
  const title = /<title id="[^"]*">([^<]*)<\/title>/.exec(svg);
  if (title) lines.push(`acc-title ${unescape(title[1])}`);
  const desc = /<desc id="[^"]*">([^<]*)<\/desc>/.exec(svg);
  if (desc) lines.push(`acc-desc ${unescape(desc[1])}`);
  let move = null;
  const X = (v) => n(move ? move[0] + Number(v) : Number(v));
  const Y = (v) => n(move ? move[1] + Number(v) : Number(v));
  const pt = (x, y) => `${X(x)},${Y(y)}`;
  // The fill each open group gives what is in it.
  const fills = [];
  const fill = (e) => attr(e, 'fill') ?? fills.findLast((f) => f !== undefined) ?? '-';
  const pen = (e) => `${fill(e)} ${attr(e, 'stroke') ?? '-'} ${attr(e, 'stroke-dasharray') ?? '-'} ${attr(e, 'stroke-linecap') ?? 'butt'} ${attr(e, 'stroke-linejoin') ?? 'miter'}`;
  // A path's d, as absolute moves and lines, with H and V written as L.
  const path = (d) => {
    const out = [];
    let x = 0;
    let y = 0;
    for (const m of d.matchAll(/([MLHVZ])([^MLHVZ]*)/g)) {
      const nums = m[2].trim() === '' ? [] : m[2].trim().split(',');
      switch (m[1]) {
        case 'M':
        case 'L':
          x = X(nums[0]);
          y = Y(nums[1]);
          out.push(`${m[1]} ${x},${y}`);
          break;
        case 'H':
          x = X(nums[0]);
          out.push(`L ${x},${y}`);
          break;
        case 'V':
          y = Y(nums[0]);
          out.push(`L ${x},${y}`);
          break;
        case 'Z':
          out.push('Z');
          break;
      }
    }
    return out.join(' ');
  };
  const label = (kind, e) => {
    const attrs = decodeURIComponent(attr(e, 'attrs'));
    const own = (name) => new RegExp(`\\s${name}="([^"]*)"`).exec(attrs)?.[1] ?? '-';
    // A weight on the text draws every span of it bold, as limu's scene holds it.
    const weight = own('font-weight') === '-' ? 0 : 1;
    const spans = JSON.parse(decodeURIComponent(attr(e, 'lines')))
      .map((line) => line.map((s) => (s[0] === 'i' ? `i:${s[1]}` : `t:${FACES[(s[1] | weight) & 3]}:${s[2]}`)).join(' + '))
      .join(' | ');
    const x = Number(attr(e, 'x'));
    let y = Number(attr(e, 'y'));
    const w = Number(attr(e, 'w'));
    const h = Number(attr(e, 'h'));
    const size = Number(attr(e, 'size'));
    const lh = Number(attr(e, 'lh'));
    if (kind === 'start') y = y - size * 0.35 + h / 2 - lh / 2;
    const turn = /rotate\(([^ ]*) /.exec(attrs);
    return `${kind} ${own('class')} ${own('font-size')} ${own('fill')} ${own('font-weight')}${turn ? ` turn ${turn[1]}` : ''} ${pt(x, y)} ${n(w)},${n(h)} ${n(size)} ${n(lh)} | ${spans}`;
  };

  const tags = /<(\/?)(g|path|rect|circle|svg|limu-label|limu-start|limu-marker)\b[^>]*>/g;
  // Inside an icon, whose drawing is the host's.
  let icon = false;
  for (const m of svg.matchAll(tags)) {
    const e = m[0];
    if (m[1]) {
      if (m[2] === 'g' && !icon) fills.pop();
      if (m[2] === 'svg') icon = false;
      continue;
    }
    if (icon) continue;
    switch (m[2]) {
      case 'svg':
        icon = / class="pele-icon"/.test(e) && !e.endsWith('/>');
        break;
      case 'g': {
        const t = /translate\(([^,]*),([^)]*)\)/.exec(attr(e, 'transform') ?? '');
        if (t) move = [Number(t[1]), Number(t[2])];
        fills.push(attr(e, 'fill'));
        const cls = attr(e, 'class') ?? '';
        if (/^pele-(wardley-(pipeline|annotation|annotations|accelerator|deaccelerator)|edge|node)( |$)/.test(cls)) lines.push(`group ${cls} ${attr(e, 'data-id') ?? '-'}`);
        break;
      }
      case 'path':
        // A marker's outline is the scene's to draw from where its tip is.
        if (attr(e, 'class') !== 'pele-marker') lines.push(`path ${path(attr(e, 'd'))} ${pen(e)}`);
        break;
      case 'rect': {
        const r = attr(e, 'rx') !== undefined || (attr(e, 'style') ?? '').includes('rx:') ? 'r' : '-';
        lines.push(`rect ${pt(attr(e, 'x'), attr(e, 'y'))},${n(Number(attr(e, 'width')))},${n(Number(attr(e, 'height')))} ${r} ${pen(e)}`);
        break;
      }
      case 'circle': {
        // Its middle, and the right and the bottom of it as limu's path of it holds them.
        const [cx, cy, r] = [Number(attr(e, 'cx')), Number(attr(e, 'cy')), Number(attr(e, 'r'))];
        lines.push(`circle ${pt(cx, cy)} ${pt(cx + r, cy + r)} ${pen(e)}`);
        break;
      }
      case 'limu-label':
        lines.push(label('label', e));
        break;
      case 'limu-start':
        lines.push(label('start', e));
        break;
      case 'limu-marker':
        lines.push(`marker ${attr(e, 'type')} ${pt(attr(e, 'x'), attr(e, 'y'))} ${n(Number(attr(e, 'dx')))},${n(Number(attr(e, 'dy')))} ${attr(e, 'color')}`);
        break;
    }
  }
  return lines;
}

export default async function ({ pele, src, write, random }) {
  const { render, PeleError } = await src('index.ts');
  const { metricsMeasurer } = await src('text/measurer.ts');

  const out = [];
  const host = (name) => (ICONS.includes(name) ? '<path d="M0,0"/>' : undefined);
  const draw = (source, maxWidth, fontSize, icons) => {
    let r;
    globalThis.limuExactNumbers = true;
    try {
      r = render(source, { measurer: metricsMeasurer, maxWidth, fontSize, icons: icons ? host : undefined, outputLimit: Infinity });
    } catch (e) {
      if (!(e instanceof PeleError)) throw e;
      return false;
    } finally {
      globalThis.limuExactNumbers = false;
    }
    if (r.type !== 'wardley') return false;
    out.push({ src: source, maxWidth, fontSize, icons: icons || undefined, lines: readSvg(r.svg) });
    return true;
  };
  const NARROW = [600, 480, 360, 320, 240, 200, 160, 700, 520];
  const SIZES = [13, 20, 11, 24, 15];
  const seen = new Set();
  let drawn = 0;
  const each = (source) => {
    // A Go string cannot hold a surrogate with no partner.
    if (seen.has(source) || lone(source) || source.length > 4000) return;
    seen.add(source);
    if (!draw(source, undefined, undefined, drawn % 9 === 4)) return;
    // Every third one again in a narrow host, which draws the map narrower and its labels
    // tighter, and every fifth in another size of type.
    if (drawn % 3 === 0) draw(source, NARROW[(drawn / 3) % NARROW.length]);
    if (drawn % 5 === 0) draw(source, drawn % 10 === 0 ? undefined : NARROW[(drawn / 5) % NARROW.length], SIZES[(drawn / 5) % SIZES.length]);
    drawn++;
  };
  corpus(pele).forEach(each);
  hostile().forEach(each);
  // Maps with icons in their labels, for the hosts that have them.
  for (const s of [
    'wardley-beta\ntitle fa:fa-car Map\ncomponent "fa:fa-car Car" [0.5, 0.5]\nanchor "fa:fa-coffee" [0.9, 0.9]\nnote "fa:fa-star Note" [0.2, 0.2]\n"fa:fa-car Car" -> "fa:fa-coffee"; fa:fa-star\naccelerator "fa:fa-car" [0.4, 0.6]\n',
    'wardley-beta\nevolution "fa:fa-star One" -> Two\nanchor "fa:fa-star" [0.9, 0.1]\ncomponent "fa:fa-coffee" [0.5, 0.5]\npipeline "fa:fa-coffee" {\n component "fa:fa-car" [0.3]\n component B [0.6]\n}\n',
    'wardley-beta\ncomponent "fa:fa-car" [0.5, 0.5]\ncomponent "x fa:fa-nope y" [0.3, 0.3] label [5, 5]\nannotation 1,[0.4, 0.4] "fa:fa-star text"\n',
  ]) {
    seen.delete(s);
    draw(s, undefined, undefined, true);
    draw(s, 300, undefined, true);
  }
  const maps = randomMaps(random(61));
  for (let i = 0; i < 1500; i++) each(maps());
  const usable = [...seen].filter((s) => s.length <= 3000);
  const next = mutator(usable, FRAGMENTS, random(62));
  while (drawn < 2650) each(next());
  write('diagram/wardley/testdata/pele/render.json', { cases: out });
}
