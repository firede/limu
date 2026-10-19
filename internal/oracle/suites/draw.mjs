// The geometry of what diagrams draw: shapes' sizes, insets, spans and outlines, the paths of
// edges and loops, markers, the patches under edge labels, and where clusters' titles go.
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { picker } from '../lib/corpus.mjs';

const SHAPE_ATTRS = ' fill="var(--_s)" stroke="var(--_b)"';
const LINE_ATTRS = ' stroke="var(--_b)"';
const CURVES = [undefined, 'linear', 'step', 'stepBefore', 'stepAfter', 'basis', 'cardinal', 'monotoneX'];
const MARKERS = ['arrow_point', 'arrow_circle', 'arrow_cross', 'none', ''];

export default async function ({ pele, src, cases, write, random }) {
  const shapes = await src('svg/shapes.ts');
  const edges = await src('svg/edges.ts');
  const cluster = await src('svg/cluster.ts');
  const { metricsMeasurer } = await src('text/measurer.ts');
  const { layoutLabel } = await src('text/label.ts');
  const c = cases('draw');
  const next = random(20261019);
  const pick = picker(next);
  const between = (a, b) => a + next() * (b - a);
  // Numbers as layouts make them: often whole or halves, sometimes anything.
  const coord = (a, b) => {
    const v = between(a, b);
    const r = next();
    return r < 0.3 ? Math.round(v) : r < 0.5 ? Math.round(v * 2) / 2 : v;
  };

  // The names in Pele's SHAPES, read from its source, which does not export them.
  const source = readFileSync(join(pele, 'src/svg/shapes.ts'), 'utf8');
  const table = source.slice(source.indexOf('const SHAPES'), source.indexOf('\n};\n', source.indexOf('const SHAPES')));
  const names = [...table.matchAll(/^  '?([\w-]+)'?: \{/gm)].map((m) => m[1]);
  const all = [...names, 'nonexistent', 'constructor', 'toString', '__proto__', ''];

  // Labels laid out as flowcharts lay them out, for sizes shapes are really drawn at.
  const labels = [];
  for (const l of c.labels)
    for (const size of c.sizes)
      for (const wrap of c.wrap) {
        const out = layoutLabel(l.text, l.markdown, metricsMeasurer, size, wrap, 0);
        labels.push([out.width, out.height]);
      }
  for (let i = 0; i < 30; i++) labels.push([between(0, 400), between(0, 150)]);
  labels.push([0, 0], [1e-9, 1e-9], [500, 10]);

  const size = [];
  for (const shape of all) for (const [tw, th] of labels) size.push({ shape, tw, th, ...shapes.shapeSize(shape, tw, th) });

  const inset = [];
  const span = [];
  for (const shape of all) {
    for (const [tw, th] of labels.filter((_, i) => i % 10 === 0)) {
      const { w, h } = shapes.shapeSize(shape, tw, th);
      for (const sideways of [false, true]) span.push({ shape, w, h, sideways, v: shapes.shapeSpan(shape, w, h, sideways) });
      for (let side = 0; side < 4; side++) {
        const half = (side & 1 ? h : w) / 2;
        for (const t of [0, half, between(-half, half), coord(-half * 1.2, half * 1.2)]) {
          inset.push({ shape, w, h, side, t, v: shapes.shapeInset(shape, w, h, side, t) });
        }
      }
    }
  }
  const hasLabel = all.map((shape) => ({ shape, v: shapes.shapeHasLabel(shape) }));

  // Route ends on, near and off the sides of nodes.
  const ends = [];
  for (let n = 0; n < 600; n++) {
    const node = { x: coord(0, 600), y: coord(0, 600), w: coord(20, 200), h: coord(20, 120), isGroup: next() < 0.1 };
    const shape = pick(all);
    const w = next() < 0.5 ? node.w : node.w - coord(0, 16);
    const h = next() < 0.5 ? node.h : node.h - coord(0, 16);
    const side = Math.floor(next() * 5);
    const off = pick([0, 0, 0, 0.2, -0.3, 0.49, 0.6, 3]);
    const along = between(-0.6, 0.6);
    let x;
    let y;
    if (side === 0 || side === 2) {
      x = node.x + along * node.w;
      y = node.y + (side === 0 ? -1 : 1) * (node.h / 2 + off);
    } else if (side < 4) {
      x = node.x + (side === 3 ? -1 : 1) * (node.w / 2 + off);
      y = node.y + along * node.h;
    } else {
      x = coord(0, 600);
      y = coord(0, 600);
    }
    const end = [x, y, Math.floor(next() * 2)];
    const other = [coord(0, 600), coord(0, 600), 0];
    const atEnd = next() < 0.5;
    const route = atEnd ? [...other, ...end] : [...end, ...other];
    const at = atEnd ? 3 : 0;
    const out = route.slice();
    shapes.insetRoute(out, at, { shape, w, h }, node);
    ends.push({ route, at, node, shape, w, h, side: shapes.endSide(route, at, node), out });
  }

  // Routes as layouts make them, and others.
  const routes = [];
  for (let n = 0; n < 300; n++) {
    const count = 2 + Math.floor(next() * 5);
    const axis = next() < 0.5 ? 0 : 1;
    let x = coord(0, 600);
    let y = coord(0, 600);
    const route = [];
    let dx = 0;
    let dy = 0;
    for (let i = 0; i < count; i++) {
      route.push(x, y, next() < 0.9 ? axis : 1 - axis);
      const step = Math.floor(next() * 8);
      const far = pick([coord(-200, 200), coord(-40, 40), coord(-12, 12)]);
      if (step === 0) [dx, dy] = [far, 0];
      else if (step === 1) [dx, dy] = [0, far];
      else if (step === 2) [dx, dy] = [far, coord(-200, 200)];
      else if (step === 3) [dx, dy] = [between(-0.009, 0.009), between(-0.009, 0.009)];
      else if (step === 4 && (dx !== 0 || dy !== 0)) [dx, dy] = [dx * 0.5, dy * 0.5];
      else if (step === 5) [dx, dy] = [between(-0.009, 0.009), far];
      else [dx, dy] = (i + n) % 2 ? [far, 0] : [0, far];
      x += dx;
      y += dy;
    }
    const startTrim = pick([0, 0, 4, 7, 8]);
    const endTrim = pick([0, 4, 7, 7, 8]);
    const startReach = next() < 0.4 ? null : pick([0, 4, 10, 13, 14, 20, 30]);
    const endReach = next() < 0.4 ? null : pick([0, 4, 10, 13, 14, 20, 30]);
    for (const curve of CURVES) {
      const p = edges.routePath(route, curve, startTrim, endTrim, startReach ?? undefined, endReach ?? undefined);
      routes.push({ route, curve: curve ?? null, startTrim, endTrim, startReach, endReach, ...p });
    }
  }

  const loops = [];
  for (let n = 0; n < 200; n++) {
    const [w, h] = pick([
      [80, 40],
      [14, 14],
      [coord(20, 300), coord(20, 120)],
      [coord(8, 20), coord(8, 20)],
    ]);
    const node = { x: coord(0, 600), y: coord(0, 600) };
    const k = Math.floor(next() * 5);
    const sideways = next() < 0.5;
    const base = pick([26, 20, 30]);
    const trim = pick([0, 4, 7, 8]);
    const e = { labelW: next() < 0.3 ? 0 : coord(10, 120), labelH: next() < 0.3 ? 0 : coord(10, 60), labelX: 0, labelY: 0 };
    const p = edges.loopPath(node, w, h, k, sideways, base, trim, e);
    loops.push({ x: node.x, y: node.y, w, h, k, sideways, base, trim, labelW: e.labelW, labelH: e.labelH, labelX: e.labelX, labelY: e.labelY, ...p });
  }

  const reach = [];
  for (let n = 0; n < 50; n++) {
    const r = edges.sideReach();
    const adds = [];
    for (let i = 0; i < 20; i++) {
      const add = [Math.floor(next() * 4), Math.floor(next() * 5) - 1, pick(MARKERS)];
      r.add(...add);
      adds.push(add);
    }
    const of = [];
    for (let node = 0; node < 4; node++) for (let side = -1; side < 4; side++) of.push([node, side, r.of(node, side)]);
    reach.push({ adds, of });
  }

  // Clusters with edges that enter at their tops, and edges that pass.
  const titles = [];
  for (let n = 0; n < 150; n++) {
    const node = { x: coord(100, 500), y: coord(100, 500), w: coord(40, 400), h: coord(40, 300) };
    const top = node.y - node.h / 2;
    const crossings = new Map();
    const routes = [];
    const count = Math.floor(next() * 5);
    for (let i = 0; i < count; i++) {
      const route = [];
      const points = 1 + Math.floor(next() * 4);
      for (let k = 0; k < points; k++) {
        const atTop = next() < 0.6;
        route.push(coord(node.x - node.w, node.x + node.w), atTop ? top + pick([0, 0.3, -0.4, 0.5, -0.5]) : coord(0, 600), 0);
      }
      cluster.markCrossings(crossings, route);
      routes.push(route);
    }
    const marked = [...crossings.entries()];
    const tried = [];
    for (const width of [0, coord(10, 60), coord(10, 200), node.w, node.w * 2]) {
      const pad = pick([8, 16, 20]);
      const x = cluster.clusterTitleX(node, width, crossings, pad);
      const probes = [x, coord(node.x - node.w, node.x + node.w), node.x].map((px) => [px, cluster.titleCrossed(node, width, px, crossings)]);
      tried.push({ width, pad, x, probes });
    }
    titles.push({ node, routes, marked, tried });
  }

  // Outlines with their numbers written out in full, as the oracle's hooks have num() write
  // them while limuExactNumbers is set, so that those of lines and curves can be compared
  // number for number.
  const outlines = [];
  const drawShape = (shape, w, h) => {
    globalThis.limuExactNumbers = true;
    try {
      return shapes.drawShape(shape, w, h, SHAPE_ATTRS, LINE_ATTRS);
    } finally {
      globalThis.limuExactNumbers = false;
    }
  };
  for (const shape of [...names, 'nonexistent']) {
    const sizes = labels.slice(0, 44).filter((_, i) => i % 3 === 0);
    for (const [tw, th] of sizes.concat([[coord(0, 300), coord(0, 100)]])) {
      const { w, h } = shapes.shapeSize(shape, tw, th);
      outlines.push({ shape, w, h, x: coord(-300, 300), y: coord(-300, 300), svg: drawShape(shape, w, h) });
    }
  }

  const markers = [];
  for (let n = 0; n < 200; n++) {
    const a = next() < 0.5 ? (Math.PI / 2) * Math.floor(next() * 4) : between(0, 2 * Math.PI);
    const dx = Math.cos(a);
    const dy = Math.sin(a);
    const type = pick(MARKERS);
    const x = coord(0, 600);
    const y = coord(0, 600);
    markers.push({ type, x, y, dx, dy, svg: edges.marker(type, x, y, dx, dy, 'C') });
  }

  const patches = [];
  for (let n = 0; n < 50; n++) {
    const width = coord(0, 200);
    const height = coord(0, 60);
    const x = coord(0, 600);
    const y = coord(0, 600);
    const label = { lines: [], widths: [], width, height, size: 16, lineHeight: 1.5 };
    patches.push({ width, height, x, y, svg: edges.edgeLabelSvg('e', label, x, y, '') });
  }

  const struck = [];
  const title = { lines: [[{ text: 'T', style: 0 }]], widths: [10], width: 10, height: 24, size: 16, lineHeight: 24 };
  for (let n = 0; n < 60; n++) {
    const count = 1 + Math.floor(next() * 12);
    const nodes = [];
    for (let i = 0; i < count; i++) nodes.push({ parent: i === 0 || next() < 0.2 ? -1 : Math.floor(next() * i) });
    const index = Math.floor(next() * count);
    const svg = cluster.struckTitle(title, 0, 0, '', 'c', nodes, index);
    struck.push({ parents: nodes.map((n) => n.parent), index, tint: svg.match(/var\(--_a\) ([-\d.]+)%/)[1] });
  }

  const hypot = [];
  for (let n = 0; n < 2000; n++) {
    const a = pick([coord(-600, 600), between(-1, 1) * 10 ** Math.floor(between(-6, 6)), 0, 3, 4]);
    const b = pick([coord(-600, 600), between(-1, 1) * 10 ** Math.floor(between(-6, 6)), 0, 5]);
    hypot.push([a, b, Math.hypot(a, b)]);
  }

  write('draw/testdata/pele/draw.json', {
    names,
    size,
    inset,
    span,
    hasLabel,
    ends,
    routes,
    loops,
    reach,
    titles,
    outlines,
    markers,
    patches,
    struck,
    hypot,
  });
}
