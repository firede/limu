import type { APIRoute } from 'astro';
import { docs } from '../lib/docs';
import { exampleTypeHref, exampleTypes, examplesDescription, examplesTitle } from '../lib/examples';
import { siteDescription } from '../lib/site';

// The site's pages as Markdown, for agents, in the form llmstxt.org gives: the documents in
// the order the navigation lists them, and the examples of each type as optional reading.
export const GET: APIRoute = ({ site }) => {
  const link = (title: string, path: string, description: string) => `- [${title}](${new URL(`${path}.md`, site)}): ${description}`;
  const pages = [
    link('Introduction', '/index', siteDescription),
    link(examplesTitle, '/examples', examplesDescription),
    ...docs.map((doc) => link(doc.title, `/${doc.id}`, doc.module.frontmatter.description)),
  ];
  const examples = exampleTypes.map((t) => link(`${t.title} examples`, exampleTypeHref(t.type), t.description));
  const body = `# limu\n\n> ${siteDescription}\n\n## Docs\n\n${pages.join('\n')}\n\n## Optional\n\n${examples.join('\n')}\n`;
  return new Response(body, { headers: { 'Content-Type': 'text/plain; charset=utf-8' } });
};
