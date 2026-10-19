// What the oracle's hooks write into Pele's SVG while limuExactNumbers is set, where the scene
// holds a label or a marker otherwise than Pele draws it, and the readers of it.
//
// A label is written as
//
//   <limu-label x="…" y="…" align="…" attrs="…" data="…" n="…"/>
//
// right before what Pele writes for it: x and y the point it is drawn about, in full; align
// how its lines stand there, middle, start or end; attrs the attributes Pele gives its text;
// data all that is in the label, as JSON; and n how many code units Pele's own writing of it
// takes, which follows. A marker is written as
//
//   <limu-marker type="…" x="…" y="…" dx="…" dy="…" color="…" n="…"/>
//
// right before what Pele writes for it: its type, its tip, the way it points and its color.
// Both are written as Pele draws a label or a marker, so that what is between them stays as
// it is; a suite that reads a label from the tag alone takes Pele's own writing out with
// withoutPele.

// The tag written before what Pele writes for label, drawn about (x, y) with its lines aligned
// as align says, with the attributes attrs; none where Pele writes nothing.
export function writeLabel(label, x, y, align, attrs, pele) {
  if (pele === '') return '';
  const lines = label.lines.map((line) => line.map((span) => [span.icon ?? null, span.text, span.style, span.width]));
  const data = { size: label.size, lineHeight: label.lineHeight, width: label.width, height: label.height, widths: label.widths, lines };
  return (
    `<limu-label x="${x}" y="${y}" align="${align}" attrs="${encodeURIComponent(attrs)}" ` +
    `data="${encodeURIComponent(JSON.stringify(data))}" n="${pele.length}"/>` +
    pele
  );
}

// The tag written before what Pele writes for a marker of the type, with its tip at (x, y),
// pointing along (dx, dy), in the color.
export function writeMarker(type, x, y, dx, dy, color, pele) {
  return `<limu-marker type="${type}" x="${x}" y="${y}" dx="${dx}" dy="${dy}" color="${encodeURIComponent(color)}" n="${pele.length}"/>` + pele;
}

// The attributes of a tag the hooks write, as they are written.
const attrs = (tag) => {
  const out = {};
  for (const m of tag.matchAll(/\s([a-z]+)="([^"]*)"/g)) out[m[1]] = m[2];
  return out;
};

// A label from its tag: x, y and the measures as numbers, align, attrs, and its lines, each a
// list of spans { icon, text, style, width }, icon undefined in a span of text.
export function readLabel(tag) {
  const a = attrs(tag);
  const d = JSON.parse(decodeURIComponent(a.data));
  return {
    x: Number(a.x),
    y: Number(a.y),
    align: a.align,
    attrs: decodeURIComponent(a.attrs),
    size: d.size,
    lineHeight: d.lineHeight,
    width: d.width,
    height: d.height,
    widths: d.widths,
    lines: d.lines.map((line) => line.map(([icon, text, style, width]) => ({ icon: icon ?? undefined, text, style, width }))),
  };
}

// A marker from its tag: type, x, y, dx, dy and color.
export function readMarker(tag) {
  const a = attrs(tag);
  return { type: a.type, x: Number(a.x), y: Number(a.y), dx: Number(a.dx), dy: Number(a.dy), color: decodeURIComponent(a.color) };
}

// The SVG with what Pele writes for each label and marker the hooks write a tag for taken out,
// so that the tag stands in its place.
export function withoutPele(svg) {
  const tags = /<limu-(?:label|marker) [^>]*\/>/g;
  let out = '';
  let from = 0;
  for (let m = tags.exec(svg); m; m = tags.exec(svg)) {
    const end = m.index + m[0].length;
    out += svg.slice(from, end);
    from = end + Number(attrs(m[0]).n);
    tags.lastIndex = from;
  }
  return out + svg.slice(from);
}
