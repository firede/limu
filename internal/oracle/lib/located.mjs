// What Pele's parseSource does about the parse of a diagram's text, for the suites that call
// the parse of one type with the text preprocess gives: an error the parse locates in that text
// is reported where it is in the source as written (locatingIn, with core's inSource, which
// core.ts does not export and which is written out here), and the entity codes in its message
// and snippet are written as they were, with the type it is of where it has none (tagErrors).
export async function locator(src) {
  const { inputOffset, locatingIn, PeleError } = await src('errors.ts');
  const { cleanupComments, encodeEntities, normalizeLineBreaks, preprocess } = await src('preprocess.ts');
  const { writtenCodes } = await src('text/entities.ts');
  const { detectDiagram } = await src('detect.ts');

  // core.ts's inSource.
  const inSource = (text, keepComments) => {
    let steps;
    let source = '';
    return (at) => {
      if (steps === undefined) {
        steps = [];
        const pre = preprocess(text, steps);
        encodeEntities(keepComments ? pre.withComments : cleanupComments(pre.withComments, steps), steps);
        source = normalizeLineBreaks(text) + '\n';
      }
      return [source, inputOffset(steps, at)];
    };
  };

  // Runs run, which parses the text of source as Pele's parseSource gives it to the parse of
  // the diagram's type, with comment lines kept when keepComments is true.
  return (source, keepComments, run) => {
    const pre = preprocess(source);
    const parsed = encodeEntities(keepComments ? pre.withComments : pre.text) + '\n';
    try {
      return locatingIn({ parsed, find: inSource(source, keepComments) }, run);
    } catch (error) {
      if (error instanceof PeleError) {
        const found = detectDiagram(pre.text);
        error.type ??= found.type;
        error.mermaidType ??= found.mermaidType;
        error.message = writtenCodes(error.message);
        error.snippet = writtenCodes(error.snippet);
      }
      throw error;
    }
  };
}
