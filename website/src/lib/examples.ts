import gallery from '../generated/examples.json';

// A diagram drawn at the widths gen draws it at: about the docs column's on a desktop, a
// tablet's and a phone's. medium and narrow are there only where the diagram draws narrower.
export interface Drawing {
  svg: string;
  medium?: string;
  narrow?: string;
}

export interface Example extends Drawing {
  title: string;
  source: string;
  featured?: boolean;
}

export interface ExampleType {
  type: string;
  title: string;
  description: string;
  beta: boolean;
  examples: Example[];
}

// The types, as the playground's gallery orders them: those Mermaid has first, then those
// Pele adds, which it marks as beta.
export const exampleTypes: ExampleType[] = gallery;

export const examplesTitle = 'Examples';
export const examplesDescription = 'Every type of diagram limu reads, drawn by its svg package when the site was built.';

export const exampleTypeHref = (type: string): string => `/examples/${type}`;

// The examples the overview shows of a type: those the gallery features, or its first.
export const featured = (t: ExampleType): Example[] => {
  const marked = t.examples.filter((example) => example.featured);
  return marked.length > 0 ? marked : t.examples.slice(0, 1);
};

// The playground takes a diagram in its URL's fragment, as Pele's does; a source longer
// than this is not offered, as a link that long is not one to share.
const maxPlaygroundHash = 32_000;

export function playgroundHref(source: string): string | undefined {
  const hash = new URLSearchParams({ source }).toString();
  return hash.length <= maxPlaygroundHash ? `/playground#${hash}` : undefined;
}

// A slug for a heading, from an example's title.
export const slug = (text: string): string =>
  text.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
