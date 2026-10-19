// The layered and compound layouts: Pele's layered() and compoundLayout() run on the graphs in
// cases/layout.json, each with several options, and on random graphs like those a flowchart
// gives them.
import { picker } from '../lib/corpus.mjs';

// JSON has no infinities and no NaN; they are written as strings, and read back so.
function finite(value) {
  if (typeof value === 'number' && !Number.isFinite(value)) return String(value);
  if (Array.isArray(value)) return value.map(finite);
  if (value && typeof value === 'object') {
    const out = {};
    for (const [k, v] of Object.entries(value)) out[k] = finite(v);
    return out;
  }
  return value;
}
const number = (v) => (typeof v === 'string' ? Number(v) : v);

const BASE = { nodeSep: 40, edgeSep: 16, rankSep: 48, portSep: 20, headRoom: 7 };

// The options each written layered graph is run with.
const LAYERED_VARIANTS = [
  { name: 'plain', spans: false, opt: {} },
  { name: 'spans', spans: true, opt: {} },
  { name: 'spans tracks', spans: true, opt: { tracks: true } },
  { name: 'spans straight', spans: true, opt: { straight: true } },
  { name: 'spans apart tight', spans: true, opt: { apart: true, nodeSep: 20, rankSep: 36, headRoom: undefined } },
  { name: 'tracks', spans: false, opt: { tracks: true, edgeSep: 10, portSep: 30 } },
  { name: 'spans wide', spans: true, opt: { nodeSep: 60, edgeSep: 24, rankSep: 80, portSep: 12, headRoom: 12 } },
];

const COMPOUND_VARIANTS = [
  { name: 'curved', spans: true, opt: {} },
  { name: 'tracks', spans: true, opt: { tracks: true } },
  { name: 'straight', spans: true, opt: { straight: true } },
  { name: 'no spans', spans: false, opt: { nodeSep: 20, rankSep: 36, headRoom: undefined } },
];

function lnodeIn(spec, spans) {
  const o = Array.isArray(spec) ? { w: spec[0], h: spec[1] } : { ...spec };
  return {
    w: o.w,
    h: o.h,
    kind: o.kind ?? 0,
    pin: o.pin ?? 0,
    span: o.span !== undefined ? number(o.span) : spans ? o.w * 0.75 : 0,
  };
}

function ledgeIn(spec) {
  const o = spec[2] ?? {};
  return {
    tail: spec[0],
    head: spec[1],
    minlen: o.minlen ?? 1,
    labelW: o.labelW ?? 0,
    labelH: o.labelH ?? 0,
    tailDx: o.tailDx ?? 0,
    headDx: o.headDx ?? 0,
    tailFixed: o.tailFixed ?? false,
    headFixed: o.headFixed ?? false,
  };
}

function cnodeIn(spec, i, spans) {
  if (Array.isArray(spec)) spec = { w: spec[0], h: spec[1], parent: spec[2] };
  if (spec.group) {
    return {
      w: 0, h: 0, parent: spec.parent ?? -1, isGroup: true, dir: spec.dir,
      padX: spec.padX ?? 20, padTop: spec.padTop ?? 20, padBottom: spec.padBottom ?? 20, minW: spec.minW ?? 28,
      seq: spec.seq ?? i, span: Infinity,
    };
  }
  return {
    w: spec.w, h: spec.h, parent: spec.parent ?? -1, isGroup: false, dir: undefined,
    padX: 0, padTop: 0, padBottom: 0, minW: 0, seq: spec.seq ?? i, span: spec.span !== undefined ? number(spec.span) : spans ? spec.w - 10 : 0,
  };
}

function cedgeIn(spec) {
  const o = spec[2] ?? {};
  return { src: spec[0], dst: spec[1], minlen: o.minlen ?? 1, labelW: o.labelW ?? 0, labelH: o.labelH ?? 0 };
}

// A random graph for layered(), as a level of a compound layout gives it: real nodes, some
// spread their edges, some edges labelled, ports with one edge each, now and then pins and
// ends held at offsets.
function randomLayered(next, index) {
  const pick = picker(next);
  const count = 1 + Math.floor(next() * 60);
  const spanMode = pick(['none', 'flat', 'part', 'all', 'mixed']);
  const nodes = [];
  for (let i = 0; i < count; i++) {
    const w = next() < 0.3 ? 40 + Math.round(next() * 160) : 30 + next() * 170;
    const h = next() < 0.5 ? 40 + Math.round(next() * 3) * 8 : 24 + next() * 70;
    let span = 0;
    if (spanMode === 'flat') span = w - 10;
    else if (spanMode === 'part') span = w * 0.6;
    else if (spanMode === 'all') span = Infinity;
    else if (spanMode === 'mixed') span = pick([0, w - 10, w * 0.6, Infinity, h]);
    nodes.push({ w, h, kind: 0, pin: next() < 0.03 ? pick([-1, 1]) : 0, span });
  }
  const edges = [];
  const m = Math.floor(next() * count * (0.5 + next() * 1.6));
  const labels = next() < 0.5 ? 0 : next() * 0.5;
  const offsets = next() < 0.2;
  for (let k = 0; k < m && count > 1; k++) {
    const tail = Math.floor(next() * count);
    let head = Math.floor(next() * (count - 1));
    if (head >= tail) head++;
    const labelled = next() < labels;
    edges.push({
      tail, head,
      minlen: next() < 0.85 ? 1 : pick([2, 3]),
      labelW: labelled ? 12 + next() * 120 : 0,
      labelH: labelled ? 22 + Math.floor(next() * 3) * 19 : 0,
      tailDx: offsets && next() < 0.3 ? (next() - 0.5) * nodes[tail].w * 0.6 : 0,
      headDx: offsets && next() < 0.3 ? (next() - 0.5) * nodes[head].w * 0.6 : 0,
      tailFixed: offsets && next() < 0.3,
      headFixed: offsets && next() < 0.3,
    });
  }
  if (count > 0 && next() < 0.25) {
    const ports = 1 + Math.floor(next() * 4);
    for (let k = 0; k < ports; k++) {
      const d = nodes.length;
      nodes.push({ w: 0, h: 0, kind: pick([3, 4]), pin: 0, span: 0 });
      const inner = Math.floor(next() * count);
      const out = next() < 0.5;
      edges.push({
        tail: out ? inner : d, head: out ? d : inner, minlen: 1, labelW: 0, labelH: 0,
        tailDx: 0, headDx: 0, tailFixed: false, headFixed: false,
      });
    }
  }
  const opt = {
    nodeSep: pick([40, 20, 33.5]),
    edgeSep: pick([16, 10]),
    rankSep: pick([48, 36, 60]),
    portSep: pick([20, 12]),
    tracks: next() < 0.25,
    straight: next() < 0.2,
    apart: next() < 0.15,
    headRoom: next() < 0.8 ? 7 : undefined,
  };
  return { name: `random layered ${index}`, nodes, edges, opt };
}

// A random graph for compoundLayout(), as Pele's flowchart builds one: nodes in groups nested
// at random (some in each other, some in what is not a group), groups with their own
// direction, edges between nodes and groups, some labelled, some from a node to itself.
function randomCompound(next, index) {
  const pick = picker(next);
  const count = 1 + Math.floor(next() * 60);
  const groups = next() * 0.3;
  const nested = next();
  const tight = next() < 0.25;
  const pad = tight ? 12 : 20;
  const nodes = [];
  for (let i = 0; i < count; i++) {
    if (next() < groups) {
      const titled = next() < 0.7;
      const titleW = titled ? 20 + next() * 150 : 0;
      nodes.push({
        w: 0, h: 0, parent: -1, isGroup: true, dir: next() < 0.3 ? pick(['TB', 'BT', 'LR', 'RL']) : undefined,
        padX: pad, padTop: titled ? 18.5 + 16 : 20, padBottom: pad, minW: titleW + 2 * pad - 12, seq: i, span: Infinity,
      });
    } else {
      const w = next() < 0.4 ? 40 + Math.round(next() * 160) : 30 + next() * 170;
      const h = next() < 0.5 ? 42 : 30 + next() * 80;
      const span = pick([w - 10, w * 0.6, h * 0.6, 0, h - 12, w - h]);
      nodes.push({ w, h, parent: -1, isGroup: false, dir: undefined, padX: 0, padTop: 0, padBottom: 0, minW: 0, seq: i, span: Math.max(0, span) });
    }
  }
  for (let i = 0; i < count; i++) {
    if (next() < nested) nodes[i].parent = Math.floor(next() * count);
  }
  const edges = [];
  const m = Math.floor(next() * count * (0.4 + next() * 1.4));
  const labels = next() < 0.4 ? 0 : next() * 0.5;
  for (let k = 0; k < m; k++) {
    const src = Math.floor(next() * count);
    const dst = next() < 0.04 ? src : Math.floor(next() * count);
    const labelled = next() < labels;
    edges.push({
      src, dst, minlen: next() < 0.9 ? 1 : pick([2, 3]),
      labelW: labelled ? 12 + next() * 120 : 0, labelH: labelled ? 4 + 18.5 * (1 + Math.floor(next() * 2)) : 0,
    });
  }
  const opt = {
    nodeSep: tight ? 20 : pick([40, 50]),
    edgeSep: 16,
    rankSep: tight ? 36 : pick([48, 60]),
    portSep: 20,
    tracks: next() < 0.25,
    straight: next() < 0.15,
    apart: next() < 0.1,
    headRoom: next() < 0.85 ? 7 : undefined,
  };
  const shift = next() < 0.5 ? [8 + next() * 20, 8 + next() * 40] : undefined;
  return { name: `random compound ${index}`, rootDir: pick(['TB', 'TB', 'LR', 'BT', 'RL']), nodes, edges, opt, shift };
}

function runLayered(L, c) {
  const nodes = c.nodes.map((n) => {
    const ln = L.lnode(n.w, n.h, n.kind);
    ln.pin = n.pin;
    ln.span = n.span;
    return ln;
  });
  const edges = c.edges.map((e) => {
    const le = L.ledge(e.tail, e.head, e.minlen, e.labelW, e.labelH);
    le.tailDx = e.tailDx;
    le.headDx = e.headDx;
    le.tailFixed = e.tailFixed;
    le.headFixed = e.headFixed;
    return le;
  });
  const result = L.layered(nodes, edges, { ...c.opt });
  return {
    result,
    nodes: nodes.map((n) => ({ x: n.x, y: n.y, rank: n.rank })),
    edges: edges.map((e) => ({ tailDx: e.tailDx, headDx: e.headDx, reversed: e.reversed, points: e.points, labelX: e.labelX, labelY: e.labelY })),
  };
}

function runCompound(C, c) {
  const nodes = c.nodes.map((n) => {
    const cn = C.cnode(n.w, n.h, n.parent);
    Object.assign(cn, { isGroup: n.isGroup, dir: n.dir, padX: n.padX, padTop: n.padTop, padBottom: n.padBottom, minW: n.minW, seq: n.seq, span: n.span });
    return cn;
  });
  const edges = c.edges.map((e) => ({ ...e, route: [], labelX: 0, labelY: 0 }));
  const result = C.compoundLayout(nodes, edges, c.rootDir, { ...c.opt });
  if (c.shift) C.shiftLayout(nodes, edges, c.shift[0], c.shift[1]);
  return {
    result,
    nodes: nodes.map((n) => ({ x: n.x, y: n.y, w: n.w, h: n.h, parent: n.parent, seq: n.seq })),
    edges: edges.map((e) => ({ src: e.src, dst: e.dst, route: e.route, labelX: e.labelX, labelY: e.labelY })),
  };
}

export default async function ({ src, cases, write, random }) {
  const L = await src('layout/layered.ts');
  const C = await src('layout/compound.ts');
  const c = cases('layout');

  const layered = [];
  for (const g of c.layered) {
    for (const v of LAYERED_VARIANTS) {
      const opt = { ...BASE, ...v.opt };
      layered.push({ name: `${g.name} (${v.name})`, nodes: g.nodes.map((n) => lnodeIn(n, v.spans)), edges: g.edges.map(ledgeIn), opt });
    }
  }
  const nextL = random(20261019);
  for (let i = 0; i < 200; i++) layered.push(randomLayered(nextL, i));
  // Past the limits: more bends than MAX_DUMMIES, and more pieces in one gap than MAX_WEIGHED.
  {
    const nodes = [];
    const edges = [];
    for (let i = 0; i < 210; i++) nodes.push({ w: 50, h: 30, kind: 0, pin: 0, span: 40 });
    for (let i = 0; i + 1 < 210; i++) edges.push(ledgeIn([i, i + 1]));
    for (let i = 2; i < 210; i++) edges.push(ledgeIn([0, i]));
    layered.push({ name: 'many bends', nodes, edges, opt: { ...BASE } });
    layered.push({ name: 'many bends tracks', nodes, edges, opt: { ...BASE, tracks: true } });
  }
  {
    const nodes = [{ w: 4000, h: 40, kind: 0, pin: 0, span: Infinity }];
    const edges = [];
    for (let i = 1; i <= 420; i++) {
      nodes.push({ w: 30 + (i % 7) * 5, h: 30, kind: 0, pin: 0, span: 20 });
      edges.push(ledgeIn([0, i]));
      if (i % 3 === 0) edges.push(ledgeIn([i, (i * 7) % 420 + 1]));
    }
    layered.push({ name: 'wide gap', nodes, edges, opt: { ...BASE } });
    layered.push({ name: 'wide gap tracks', nodes, edges, opt: { ...BASE, tracks: true } });
  }
  for (const g of layered) g.out = runLayered(L, g);

  const compound = [];
  for (const g of c.compound) {
    for (const dir of ['TB', 'BT', 'LR', 'RL']) {
      for (const v of COMPOUND_VARIANTS) {
        compound.push({
          name: `${g.name} (${dir}, ${v.name})`, rootDir: dir, opt: { ...BASE, ...v.opt },
          nodes: g.nodes.map((n, i) => cnodeIn(n, i, v.spans)), edges: g.edges.map(cedgeIn),
        });
      }
    }
  }
  const nextC = random(1019);
  for (let i = 0; i < 400; i++) compound.push(randomCompound(nextC, i));
  for (const g of compound) g.out = runCompound(C, g);

  const nextE = random(7);
  const ease = [];
  const hypot = [];
  for (let i = 0; i < 300; i++) {
    const across = i % 10 === 0 ? 0 : (next2(nextE) - 0.5) * 2000;
    const along = i % 13 === 0 ? 0 : (next2(nextE) - 0.5) * 300;
    ease.push({ across, along, ease: L.ease(across, along) });
    const a = (nextE() - 0.5) * 10 ** Math.floor(nextE() * 8 - 4);
    const b = i % 17 === 0 ? 0 : (nextE() - 0.5) * 10 ** Math.floor(nextE() * 8 - 4);
    hypot.push({ a, b, hypot: Math.hypot(a, b) });
  }

  write('layout/testdata/pele/layout.json', finite({ ease, hypot, layered, compound }));
}

// A random number with more of its bits set than one from the generator alone.
function next2(next) {
  return next() + next() / 2 ** 32;
}
