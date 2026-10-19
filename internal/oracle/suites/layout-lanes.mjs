// The lane layout: Pele's laneLayout() run on graphs written as its tests write them, in
// every direction and with both lane paddings, on random graphs of lanes, groups inside them
// and relations between them, and on graphs past its limits.

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

const BASE = { nodeSep: 40, edgeSep: 16, rankSep: 48, portSep: 20, headRoom: 7 };
const TIGHT = { nodeSep: 20, edgeSep: 16, rankSep: 36, portSep: 20, headRoom: 7 };
const DIRS = ['TB', 'BT', 'LR', 'RL'];

const node = (w, h, parent = -1) => ({ w, h, parent, isGroup: false, padX: 0, padTop: 0, padBottom: 0, minW: 0 });
const group = (parent = -1, o = {}) => ({ w: 0, h: 0, parent, isGroup: true, padX: 20, padTop: 36, padBottom: 20, minW: 60, ...o });
const edge = (src, dst, labelW = 0, labelH = 0, minlen = 1) => ({ src, dst, minlen, labelW, labelH });

// The graphs of Pele's tests/layout/lanes.test.ts, and more written the same way.
function written() {
  const out = [];
  const add = (name, nodes, edges) => out.push({ name, nodes, edges });
  {
    const nodes = [node(60, 30, 3), node(60, 30, 2), group(), group()];
    add('two lanes', nodes, [edge(0, 1)]);
  }
  add('one lane', [node(60, 30, 2), node(60, 30, 2), group()], [edge(0, 1)]);
  add('chain in a lane', [node(50, 30, 3), node(120, 30, 3), node(70, 30, 3), group()], [edge(0, 1), edge(1, 2)]);
  {
    const nodes = [node(60, 40, 8), node(80, 40, 7), node(70, 70, 7), node(80, 40, 6), node(80, 40, 6), node(60, 40, 8), group(), group(), group()];
    add('approval decision', nodes, [edge(0, 1, 70, 20), edge(1, 2), edge(2, 3, 50, 20), edge(3, 4), edge(4, 5), edge(2, 0, 70, 20)]);
  }
  {
    const nodes = [node(110, 38, 5), node(110, 38, 5), node(100, 38, 4), node(110, 38, 4), group(), group()];
    add('handoff in and out', nodes, [edge(2, 3), edge(0, 3), edge(3, 1)]);
  }
  {
    const nodes = [node(140, 44, 8), node(140, 44, 8), node(140, 44, 7), node(120, 76, 7), node(140, 44, 6), node(140, 44, 6), group(), group(), group()];
    add('return around the margin', nodes, [edge(0, 1), edge(1, 2), edge(2, 3), edge(3, 4, 30, 20), edge(4, 5), edge(3, 0, 30, 20)]);
  }
  {
    const nodes = [node(80, 40, 5), node(80, 40, 5), node(80, 40, 5), node(80, 40, 4), group(), group()];
    add('longer link beside a handoff', nodes, [edge(0, 1, 0, 0, 2), edge(1, 2), edge(2, 3)]);
  }
  {
    const nodes = [node(110, 38, 7), node(110, 38, 6), node(110, 38, 6), node(110, 38, 6), node(110, 38, 6), node(110, 38, 6), group(), group()];
    add('relation to a lane with a big label', nodes, [edge(0, 1), edge(1, 2), edge(2, 3), edge(3, 4), edge(4, 5), edge(6, 5, 400, 200)]);
  }
  add('lanes only', [group(), group(), group(-1, { minW: 200 })], [edge(0, 1), edge(1, 2, 40, 22)]);
  add('no lanes', [node(60, 30), node(80, 40), node(50, 30)], [edge(0, 1), edge(1, 2), edge(0, 2, 30, 22)]);
  add('a node alone', [node(60, 30)], []);
  add('nothing', [], []);
  // Groups that are each other's parent, as no flowchart makes them: neither has a member to
  // be drawn around, and a relation to one runs to where it was left.
  add('groups in a cycle', [node(60, 30, 3), group(2), group(1), group(), node(50, 30, 3)], [edge(0, 1), edge(4, 2, 30, 22), edge(0, 4)]);
  add('an empty lane between', [node(60, 30, 4), node(60, 30, 2), group(), group(), group()], [edge(0, 1, 50, 22)]);
  {
    const nodes = [node(60, 30, 5), node(60, 30, 5), node(60, 30, 6), node(70, 40, 4), group(6, { padTop: 41.5, minW: 140 }), group(), group(), group(4)];
    add('nested groups', nodes, [edge(0, 1), edge(1, 2), edge(2, 3), edge(3, 0), edge(7, 3), edge(5, 7)]);
  }
  {
    const nodes = [node(60, 30, 2), node(60, 30, 2), group()];
    add('self loops and parallels', nodes, [edge(0, 0, 40, 22), edge(0, 1), edge(0, 1), edge(1, 0, 30, 22), edge(1, 1)]);
  }
  {
    // A deep chain of groups inside one lane.
    const nodes = [node(60, 30, 6), node(60, 30, 6), group()];
    for (let k = 0; k < 4; k++) nodes.push(group(nodes.length - 1, { padX: 12, padBottom: 12 }));
    add('deep groups', nodes, [edge(0, 1), edge(1, 0, 20, 22)]);
  }
  {
    const nodes = [node(90, 40, 4), node(90, 40, 5), node(90, 40, 6), node(90, 40, 4), group(), group(), group()];
    add('long hops between lanes', nodes, [edge(0, 1), edge(1, 2), edge(2, 3), edge(0, 3, 80, 22, 3), edge(3, 0), edge(0, 2)]);
  }
  {
    // A wide title on a lane and a tall title band.
    const nodes = [node(60, 30, 2), node(60, 30, 3), group(-1, { minW: 400, padTop: 60 }), group(-1, { padTop: 20 })];
    add('wide titles', nodes, [edge(0, 1, 20, 22)]);
  }
  return out;
}

// A random graph as Pele's flowchart gives the lane layout: leaves first, then lanes, then
// groups inside lanes or inside each other, some empty; leaves in lanes, in groups, or
// outside every lane; relations between leaves, now and then to a group or a lane, some
// labelled, some longer, some loops.
function randomGraph(next, index) {
  const int = (n) => Math.floor(next() * n);
  const pick = (list) => list[int(list.length)];
  const nodes = [];
  const leafCount = int(4) === 0 ? 1 + int(60) : 1 + int(30);
  const laneCount = int(8) === 0 ? 0 : 1 + int(5);
  const lanes = [];
  const inner = [];
  for (let i = 0; i < leafCount; i++) nodes.push(node(next() < 0.4 ? 30 + int(120) : 30 + next() * 150, next() < 0.5 ? 42 : 24 + next() * 60));
  for (let k = 0; k < laneCount; k++) {
    lanes.push(nodes.length);
    const titled = next() < 0.8;
    nodes.push(group(-1, { padTop: titled ? 18.5 + 16 + (next() < 0.1 ? 37 : 0) : 20, minW: titled ? 20 + next() * 200 : 28 }));
  }
  const groupCount = laneCount > 0 ? int(5) : 0;
  for (let k = 0; k < groupCount; k++) {
    const parent = inner.length > 0 && next() < 0.4 ? pick(inner) : pick(lanes);
    inner.push(nodes.length);
    const tight = next() < 0.3;
    const titled = next() < 0.7;
    nodes.push(group(parent, {
      padX: tight ? 12 : 20, padBottom: tight ? 12 : 20, padTop: titled ? 18.5 + 16 : 20, minW: titled ? 20 + next() * 150 : 28,
    }));
  }
  for (let i = 0; i < leafCount; i++) {
    const r = next();
    if (inner.length > 0 && r < 0.3) nodes[i].parent = pick(inner);
    else if (lanes.length > 0 && r < 0.9) nodes[i].parent = pick(lanes);
  }
  const edges = [];
  const labels = next() < 0.4 ? 0 : next() * 0.5;
  for (let k = 0, count = int(leafCount * 2.2); k < count; k++) {
    const src = int(leafCount);
    const dst = next() < 0.04 ? src : int(leafCount);
    const labelled = next() < labels;
    edges.push(edge(src, dst, labelled ? 12 + next() * 120 : 0, labelled ? 4 + 18.5 * (1 + int(2)) : 0, next() < 0.9 ? 1 : pick([2, 3])));
  }
  if (nodes.length > leafCount && next() < 0.3) {
    for (let k = 0; k < 1 + int(3); k++) {
      const g = leafCount + int(nodes.length - leafCount);
      const leaf = int(leafCount);
      edges.push(next() < 0.5 ? edge(g, leaf) : edge(leaf, g, next() < 0.5 ? 40 : 0, 22));
    }
  }
  const tight = next() < 0.3;
  return {
    name: `random ${index}`, dir: pick(DIRS), nodes, edges, lanePad: tight ? 12 : 24,
    opt: tight ? { ...TIGHT } : next() < 0.2 ? { ...BASE, nodeSep: 50, rankSep: 60 } : { ...BASE },
  };
}

// Graphs past the limits: more bends than MAX_DUMMIES, and more work than the routes' budgets.
function large() {
  const out = [];
  {
    // Edges that span many ranks.
    const nodes = [];
    const edges = [];
    for (let i = 0; i <= 600; i++) nodes.push(node(60, 30, 602));
    nodes.push(node(60, 30, 603));
    nodes.push(group(), group());
    for (let i = 0; i < 600; i++) edges.push(edge(i, i + 1));
    for (let i = 0; i < 40; i++) edges.push(edge(0, 600 - i), edge(601, 500 - i));
    out.push({ name: 'many bends', dir: 'TB', nodes, edges, lanePad: 24, opt: { ...BASE } });
  }
  for (const [count, links, name] of [[200, 300, 'some work'], [600, 900, 'much work']]) {
    let seed = 7;
    const rnd = (n) => (seed = (seed * 1103515245 + 12345) & 0x7fffffff) % n;
    const lanes = 6;
    const nodes = [];
    for (let i = 0; i < count; i++) nodes.push(node(40 + (i % 5) * 20, 30 + (i % 3) * 10, count + (i % lanes)));
    for (let k = 0; k < lanes; k++) nodes.push(group());
    const edges = [];
    for (let e = 0; e < links; e++) edges.push(edge(rnd(count), rnd(count), e % 7 === 0 ? 50 : 0, e % 7 === 0 ? 22 : 0));
    out.push({ name, dir: 'LR', nodes, edges, lanePad: 24, opt: { ...BASE } });
  }
  {
    // A chain that changes lane at every step.
    const nodes = [];
    const edges = [];
    for (let i = 0; i < 160; i++) nodes.push(node(70, 36, 160 + (i % 8)));
    for (let k = 0; k < 8; k++) nodes.push(group());
    for (let i = 0; i < 159; i++) edges.push(edge(i, i + 1));
    out.push({ name: 'lane at every step', dir: 'LR', nodes, edges, lanePad: 24, opt: { ...BASE } });
  }
  return out;
}

function run(L, c) {
  const nodes = c.nodes.map((n) => {
    const cn = L.cnode(n.w, n.h, n.parent);
    Object.assign(cn, { isGroup: n.isGroup, padX: n.padX, padTop: n.padTop, padBottom: n.padBottom, minW: n.minW });
    return cn;
  });
  const edges = c.edges.map((e) => ({ ...e, route: [], labelX: 0, labelY: 0 }));
  const result = L.laneLayout(nodes, edges, c.dir, { ...c.opt }, c.lanePad);
  return {
    result,
    nodes: nodes.map((n) => ({ x: n.x, y: n.y, w: n.w, h: n.h, padTop: n.padTop })),
    edges: edges.map((e) => ({ route: e.route, labelX: e.labelX, labelY: e.labelY })),
  };
}

export default async function ({ src, write, random }) {
  const { laneLayout } = await src('layout/lanes.ts');
  const { cnode } = await src('layout/compound.ts');
  const L = { laneLayout, cnode };

  const cases = [];
  for (const g of written()) {
    for (const dir of DIRS) {
      for (const [lanePad, opt] of [[24, BASE], [12, TIGHT], [4, BASE]]) {
        cases.push({ name: `${g.name} (${dir}, pad ${lanePad})`, dir, nodes: g.nodes, edges: g.edges, lanePad, opt: { ...opt } });
      }
    }
  }
  const next = random(20261019);
  for (let i = 0; i < 600; i++) cases.push(randomGraph(next, i));
  cases.push(...large());
  for (const c of cases) c.out = run(L, c);
  write('layout/testdata/pele/lanes.json', finite({ cases }));
}
