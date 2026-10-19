// Gantt charts drawn end to end: what Pele's render draws, read back out of its SVG as one line
// per thing drawn, in the order drawn, in several time zones against a fixed time. limu's test
// writes the same lines from its scene.
//
// Pele's SVG is drawn here with its numbers written out in full, as the oracle's hooks have num()
// write them while limuExactNumbers is set; a gantt chart moves nothing inside groups, so each
// number is read and written to two decimals as Pele writes it.
import { lone, mutator, peleError } from './parse.mjs';
import { FRAGMENTS, NOWS, ZONES, corpus, statementDiagrams } from './gantt.mjs';

// Pele's escapes in attribute values and text, undone.
const unescape = (s) => s.replace(/&(amp|lt|gt|quot);/g, (_, e) => ({ amp: '&', lt: '<', gt: '>', quot: '"' })[e]);
const attr = (s, name) => {
  const m = new RegExp(`\\s${name}="([^"]*)"`).exec(s);
  return m ? unescape(m[1]) : undefined;
};
// How a shape is painted: its fill, with its opacity, and its stroke.
const paint = (s) => {
  const opacity = attr(s, 'fill-opacity');
  return `${attr(s, 'fill') ?? '-'}${opacity ? '/' + opacity : ''} ${attr(s, 'stroke') ?? '-'}`;
};

// How a path's stroke ends and turns its corners, "-" for SVG's initial ones, as the browser
// settles them: from the author's style Pele writes on the path, where it gives a keyword the
// browser takes, else from the path's own attribute. No group a path is in sets either.
const pen = (s) => {
  const css = new Map();
  for (const decl of (attr(s, 'style') ?? '').split(';')) {
    const at = decl.indexOf(':');
    if (at > 0) css.set(decl.slice(0, at).trim(), decl.slice(at + 1).trim().toLowerCase());
  }
  const one = (prop, keywords, initial) => {
    const v = css.get(prop);
    if (keywords.includes(v)) return v;
    if (v === 'initial') return initial;
    if (v === 'inherit' || v === 'unset') return '-';
    return attr(s, prop) ?? '-';
  };
  return `${one('stroke-linecap', ['butt', 'round', 'square'], 'butt')} ${one('stroke-linejoin', ['miter', 'round', 'bevel'], 'miter')}`;
};

function reader(num) {
  const pt = (x, y) => `${num(Number(x))},${num(Number(y))}`;
  const box = (s) => `${pt(attr(s, 'x') ?? 0, attr(s, 'y') ?? 0)},${pt(attr(s, 'width'), attr(s, 'height'))}`;
  // Where a text's first line sits, the first x and y in it, and its own fill; s starts with
  // the text.
  const textAt = (s) => {
    const m = / x="([^"]*)" y="([^"]*)"/.exec(s);
    return `t ${pt(m[1], m[2])} ${attr(s.slice(0, s.indexOf('>')), 'fill') ?? '-'}`;
  };
  // The subpaths of a path's d, each as one string of absolute commands, M, L and Z, with H
  // and V written as L.
  const subpaths = (d) => {
    const out = [];
    let cur = [];
    let x = '0';
    let y = '0';
    for (const m of d.matchAll(/([MLHVZ])([^MLHVZ]*)/g)) {
      const nums = m[2].trim() === '' ? [] : m[2].trim().split(/[ ,]+/);
      switch (m[1]) {
        case 'M':
          if (cur.length) out.push(cur);
          [x, y] = nums;
          cur = [`M ${pt(x, y)}`];
          break;
        case 'L':
          [x, y] = nums;
          cur.push(`L ${pt(x, y)}`);
          break;
        case 'H':
          x = nums[0];
          cur.push(`L ${pt(x, y)}`);
          break;
        case 'V':
          y = nums[0];
          cur.push(`L ${pt(x, y)}`);
          break;
        case 'Z':
          cur.push('Z');
          break;
      }
    }
    if (cur.length) out.push(cur);
    return out.map((s) => s.join(' '));
  };
  return { pt, box, textAt, subpaths };
}

// The lines of a gantt chart's SVG, drawn with its numbers in full; num writes a number as Pele
// does.
export function readSvg(svg, num) {
  const f = reader(num);
  const lines = [];
  const head = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  lines.push(`size ${f.pt(attr(head, 'width'), attr(head, 'height'))}`);
  // The font size of the group a text is in.
  let size = '';
  for (const m of svg.matchAll(/<(\/?)(g|text|rect|path|a)\b[^>]*>/g)) {
    const e = m[0];
    if (m[1]) continue;
    const cls = attr(e, 'class') ?? '';
    const id = attr(e, 'data-id');
    const textAt = () => f.textAt(svg.slice(m.index, svg.indexOf('</text>', m.index)));
    switch (m[2]) {
      case 'g':
        if (attr(e, 'font-size') !== undefined) size = num(Number(attr(e, 'font-size')));
        if (/^pele-task( |$)/.test(cls)) {
          lines.push(`task ${id} [${cls.split(' ').slice(1).map((c) => c.slice('pele-'.length)).join(' ')}]`);
        } else if (cls === 'pele-today') {
          lines.push(`today ${/<g class="pele-today"><path [^>]*style="/.test(svg) ? 'styled' : '-'}`);
        }
        break;
      case 'a':
        lines.push(`link ${attr(e, 'href')}`);
        break;
      case 'rect':
        if (cls.startsWith('pele-section ')) {
          lines.push(`band ${cls.split(' ')[1].slice('pele-section-'.length)} ${f.box(e)} ${paint(e)}`);
        } else if (attr(e, 'fill') === undefined && attr(e, 'stroke') === undefined) {
          lines.push(`excluded ${f.box(e)}`);
        } else {
          const rounded = attr(e, 'rx') !== undefined || (attr(e, 'style') ?? '').includes('rx:');
          lines.push(`rect ${f.box(e)}${rounded ? ' r' : ''} ${paint(e)}`);
        }
        break;
      case 'path': {
        const d = attr(e, 'd');
        // A chart with no ticks has a grid with nothing in it, which a scene leaves out.
        if (attr(e, 'stroke-dasharray') === '1 3') {
          if (d !== '') lines.push('grid');
        } else if (attr(e, 'stroke') === 'var(--_b)' && attr(e, 'fill') === undefined) lines.push('axis');
        else if (cls === 'pele-vert') lines.push(`vert ${id}`);
        for (const p of f.subpaths(d)) lines.push(`path ${p} ${paint(e)} ${pen(e)}`);
        break;
      }
      case 'text': {
        if (cls === 'pele-title') {
          lines.push('title ' + textAt());
        } else if (cls === 'pele-section-label') {
          lines.push(`section ${id}`, `${textAt()} s${size}`);
        } else if (cls === 'pele-vert-label') {
          lines.push(`vert-label ${id}`, `${textAt()} s${size}`);
        } else if (cls === 'pele-label') {
          lines.push(`${textAt()} s${size}`);
        } else {
          // A tick: its point and its text.
          const end = svg.indexOf('</text>', m.index);
          const body = unescape(svg.slice(svg.indexOf('>', m.index) + 1, end));
          lines.push(`tick ${f.pt(attr(e, 'x'), attr(e, 'y'))} s${size} ${JSON.stringify(body)}`);
        }
        break;
      }
    }
  }
  return lines;
}

// A chart's configuration in its front matter, with what its settings may be.
const SETTINGS = {
  fontSize: [9, 12, 20, 2, 100],
  sectionFontSize: [10, 18],
  barHeight: [12, 30, 2.5, 50],
  barGap: [0, 2, 9.5],
  numberSectionStyles: [1, 2, 3, 2.6],
  topAxis: [true, false],
  leftPadding: [0, 40, 120.5],
  rightPadding: [0, 30, 75],
  useWidth: [400, 1200, 50],
  topPadding: [0, 50, 90],
  gridLineStartPadding: [10, 35, 120],
  axisFormat: ['%m/%d', '%H:%M', '%b %e'],
  tickInterval: ['1day', '1week', '3month', 'x'],
  weekday: ['monday', 'friday'],
  displayMode: ['compact', ''],
};

// Charts that draw: those of statementDiagrams, with settings in front matter some of the time.
function drawableDiagrams(rnd) {
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  const statements = statementDiagrams(rnd);
  const keys = Object.keys(SETTINGS);
  return () => {
    let body = statements();
    // The author's style of the today marker, which takes the place of its triangle when it
    // styles the line, and some that do not.
    if (rnd() < 0.15) {
      body += `todayMarker ${pick(['stroke-width:5px,stroke:#0f0,opacity:0.5', 'fill:#f00', 'fill-opacity:0.2', 'stroke-linecap:round',
        'stroke-linecap:SQUARE,stroke-linejoin:bevel', 'stroke-linecap:initial', 'stroke-linejoin:inherit', 'stroke-linecap:flat',
        'stroke:url(x)', 'color:red', ' STROKE : red !IMPORTANT', 'stroke:', ':x', 'stroke-dasharray:4 2', 'rx:3'])}\n`;
    }
    if (rnd() < 0.5) return body;
    let config = '';
    let top = '';
    for (let k = 1 + Math.floor(rnd() * 4); k > 0; k--) {
      const key = pick(keys);
      const value = pick(SETTINGS[key]);
      if (key === 'displayMode') top += `displayMode: ${value || '""'}\n`;
      else config += `    ${key}: ${typeof value === 'string' ? JSON.stringify(value) : value}\n`;
    }
    return `---\n${top}${config ? `config:\n  gantt:\n${config}` : ''}---\n${body}`;
  };
}

export default async function ({ src, pele, write, random }) {
  const { render, PeleError } = await src('index.ts');
  const { metricsMeasurer } = await src('text/measurer.ts');
  const { num } = await src('svg/builder.ts');
  const { preprocess, encodeEntities } = await src('preprocess.ts');
  const { gantt } = await src('diagrams/gantt/index.ts');
  // Whether a chart parses, so that an error in drawing it is one of scheduling or drawing.
  const parses = (source) => {
    try {
      const pre = preprocess(source);
      gantt.parse(encodeEntities(pre.text) + '\n', pre.config, pre.title);
      return true;
    } catch {
      return false;
    }
  };
  const saved = process.env.TZ;

  const out = [];
  let k = 0;
  let errors = 0;
  // Draws a chart in the next zone, against the next time.
  const draw = (source, maxWidth, fontSize) => {
    const zone = ZONES[k % ZONES.length];
    const now = NOWS[Math.floor(k / ZONES.length) % NOWS.length];
    k++;
    process.env.TZ = zone;
    let r;
    globalThis.limuExactNumbers = true;
    try {
      r = render(source, { measurer: metricsMeasurer, maxWidth, fontSize, now, outputLimit: Infinity });
    } catch (e) {
      if (!(e instanceof PeleError)) throw e;
      // An error a chart's drawing raises, as scheduling's do, but not one of a chart that
      // does not parse.
      if (e.type === 'gantt' && errors < 300 && parses(source)) {
        errors++;
        out.push({ src: source, zone, now, maxWidth, fontSize, error: peleError(e) });
      }
      return false;
    } finally {
      globalThis.limuExactNumbers = false;
    }
    if (r.type !== 'gantt') return false;
    out.push({ src: source, zone, now, maxWidth, fontSize, lines: readSvg(r.svg, num) });
    return true;
  };
  const seen = new Set();
  let drawn = 0;
  const each = (source) => {
    // A Go string cannot hold a surrogate with no partner.
    if (seen.has(source) || lone(source)) return;
    seen.add(source);
    if (!draw(source)) return;
    // Every third one again in a narrow host, and every seventh in another size of type.
    if (drawn % 3 === 0) draw(source, 360);
    if (drawn % 7 === 0) draw(source, undefined, 13);
    drawn++;
  };
  corpus(pele).forEach(each);
  const drawable = drawableDiagrams(random(51));
  for (let i = 0; i < 2500; i++) each(drawable());
  const usable = [...seen].filter((s) => s.length <= 1000);
  const next = mutator(usable, FRAGMENTS, random(52));
  while (drawn < 4000) each(next());
  // Rows whose height has three decimals. The end of the plot, its top added to the height of
  // all its rows, then rounds to two decimals one way or the other by the last bit of that
  // height, which a port that fused the product into the sum would change.
  for (const [height, gap, top, days] of [
    [6.587, 1.488, 75, [2, 5, 1, 5, 1, 2, 1]],
    [3.292, 1.639, 50, [5, 1, 5, 5, 3]],
    [35.089, 8.32, 75, [2, 4, 4, 5, 1]],
  ]) {
    const tasks = days.map((d, i) => `T${i} :t${i}, 2024-01-0${i + 1}, ${d}d\n`).join('');
    each(`---\nconfig:\n  gantt:\n    barHeight: ${height}\n    barGap: ${gap}\n    topPadding: ${top}\n---\ngantt\ndateFormat YYYY-MM-DD\n${tasks}`);
  }
  process.env.TZ = saved;
  write('diagram/gantt/testdata/pele/render.json', { zones: ZONES, cases: out });
}
