export const siteDescription = 'limu is a diagram layout engine in Go. It reads Mermaid and leaves the drawing to your program.';
export const repositoryUrl = 'https://github.com/firede/limu';
export const modulePath = 'github.com/firede/limu';

// The pages, in the order the navigation lists them. The home page and the examples are
// pages of their own; the other pages have a Markdown source in content/docs.
export const docsPages = [
  { id: 'introduction', title: 'Introduction' },
  { id: 'examples', title: 'Examples' },
  { id: 'getting-started', title: 'Getting started' },
  { id: 'api', title: 'API' },
  { id: 'theming', title: 'Theming' },
  { id: 'mygo', title: 'MyGo' },
  { id: 'svg', title: 'SVG' },
  { id: 'architecture', title: 'Architecture' },
  { id: 'compatibility', title: 'Compatibility' },
] as const;

export type DocsPageId = typeof docsPages[number]['id'];

export const pageHref = (id: DocsPageId): string => (id === 'introduction' ? '/' : `/${id}`);

export interface Heading {
  depth: number;
  slug: string;
  text: string;
}
