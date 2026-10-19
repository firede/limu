import type { APIRoute } from 'astro';
import { exampleTypeHref, exampleTypes, examplesDescription, examplesTitle, featured } from '../lib/examples';
import { formatPageTitle } from '../lib/page-title';

const fenced = (source: string): string => '```mermaid\n' + source.trimEnd() + '\n```\n';

const body = exampleTypes.map((t) =>
  `## ${t.title}${t.beta ? ' (beta)' : ''}\n\n${t.description}\n\n` +
  featured(t).map((example) => `### ${example.title}\n\n${fenced(example.source)}`).join('\n') +
  `\n[All examples](${exampleTypeHref(t.type)}.md)\n`,
).join('\n');

export const GET: APIRoute = () => new Response(
  `---\ntitle: ${formatPageTitle(examplesTitle)}\ndescription: ${examplesDescription}\n---\n\n${body}`,
  { headers: { 'Content-Type': 'text/markdown; charset=utf-8' } },
);
