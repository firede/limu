// Texts made from a corpus by mutation, for inputs that parse and still do something odd.

// A function that gives, each time it is called, a text of the corpus with one to three
// edits made at random with rnd: a fragment put in, a few characters taken out, a piece of
// another text put in, a few characters doubled, or one replaced with a fragment.
export function mutator(corpus, fragments, rnd) {
  const pick = (list) => list[Math.floor(rnd() * list.length)];
  return () => {
    let s = pick(corpus);
    const edits = 1 + Math.floor(rnd() * 3);
    for (let k = 0; k < edits; k++) {
      const pos = Math.floor(rnd() * (s.length + 1));
      const r = rnd();
      if (r < 0.35) {
        s = s.slice(0, pos) + pick(fragments) + s.slice(pos);
      } else if (r < 0.6) {
        s = s.slice(0, pos) + s.slice(pos + 1 + Math.floor(rnd() * 3));
      } else if (r < 0.75) {
        const other = pick(corpus);
        const from = Math.floor(rnd() * other.length);
        s = s.slice(0, pos) + other.slice(from, from + Math.floor(rnd() * 20)) + s.slice(pos);
      } else if (r < 0.9) {
        s = s.slice(0, pos) + s.slice(pos, pos + Math.floor(rnd() * 6)) + s.slice(pos);
      } else {
        s = s.slice(0, pos) + pick(fragments) + s.slice(pos + 1);
      }
    }
    return s;
  };
}
