// Reading Pele's SVG back, for the suites that write what it draws as lines.

// Pele's escapes in attribute values and text, undone.
export const unescape = (s) => s.replace(/&(amp|lt|gt|quot);/g, (_, e) => ({ amp: '&', lt: '<', gt: '>', quot: '"' })[e]);

// The value of the attribute name in a tag, its escapes undone; undefined where it has none.
export const attr = (s, name) => {
  const m = new RegExp(`\\s${name}="([^"]*)"`).exec(s);
  return m ? unescape(m[1]) : undefined;
};

// The first opening tag in s that starts <name and a space, without its >.
export const tag = (s, name) => {
  const i = s.indexOf('<' + name + ' ');
  return i < 0 ? undefined : s.slice(i, s.indexOf('>', i));
};

// A translate in a transform: its x and y.
export const TRANSLATE = /translate\(([^,]*),([^)]*)\)/;

// A number, or the string of one, as String writes it, every digit there is.
export const n = (v) => String(Number(v));
export const pt = (x, y) => `${n(x)},${n(y)}`;

// The opening tag of the svg, without its >; what follows it, less the title and the
// description it starts with for assistive technology; and those two as lines, acc-title and
// acc-desc, with their text.
export function readHead(svg) {
  const open = svg.slice(svg.indexOf('<svg'), svg.indexOf('>'));
  let body = svg.slice(svg.indexOf('>') + 1);
  const acc = [];
  const title = /^<title id="[^"]*">([^<]*)<\/title>/.exec(body);
  if (title) {
    acc.push(`acc-title ${unescape(title[1])}`);
    body = body.slice(title[0].length);
  }
  const desc = /^<desc id="[^"]*">([^<]*)<\/desc>/.exec(body);
  if (desc) {
    acc.push(`acc-desc ${unescape(desc[1])}`);
    body = body.slice(desc[0].length);
  }
  return { open, body, acc };
}

// The keywords of the cap and the join of a stroke, the initial one first.
const KEYWORDS = { 'stroke-linecap': ['butt', 'round', 'square'], 'stroke-linejoin': ['miter', 'round', 'bevel'] };

// The cap or the join, as prop names it, a stroke is drawn with, as the browser settles it from
// the opening tags of the element and of the groups it is in, outermost first, where the
// element's parent has inherited: each tag takes it from its parent, then from its attribute,
// then from the declarations of its style attribute, in turn. The browser takes a keyword
// there in either case, initial in a declaration for SVG's initial value and inherit or unset
// for the parent's, and drops any other value; so the last declaration it takes wins.
export function stroke(prop, chain, inherited = KEYWORDS[prop][0]) {
  const keywords = KEYWORDS[prop];
  let v = inherited;
  for (const open of chain) {
    const parent = v;
    const a = attr(open, prop);
    if (a !== undefined && keywords.includes(a.toLowerCase())) v = a.toLowerCase();
    for (const decl of (attr(open, 'style') ?? '').split(';')) {
      const at = decl.indexOf(':');
      if (at < 0 || decl.slice(0, at).trim().toLowerCase() !== prop) continue;
      const value = decl.slice(at + 1).trim().toLowerCase();
      if (keywords.includes(value)) v = value;
      else if (value === 'initial') v = keywords[0];
      else if (value === 'inherit' || value === 'unset') v = parent;
    }
  }
  return v;
}

// The cap and the join a stroke is drawn with, as stroke settles them from chain, with a space
// between, or sep.
export const capJoin = (chain, sep = ' ') => `${stroke('stroke-linecap', chain)}${sep}${stroke('stroke-linejoin', chain)}`;
