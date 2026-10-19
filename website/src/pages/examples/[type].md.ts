import type { APIRoute } from 'astro';
import { exampleTypes, type ExampleType } from '../../lib/examples';
import { formatPageTitle } from '../../lib/page-title';

export const getStaticPaths = () => exampleTypes.map((t) => ({ params: { type: t.type }, props: t }));

const fenced = (source: string): string => '```mermaid\n' + source.trimEnd() + '\n```\n';

export const GET: APIRoute = ({ props }) => {
  const t = props as ExampleType;
  const body = t.examples.map((example) => `## ${example.title}\n\n${fenced(example.source)}`).join('\n');
  return new Response(
    `---\ntitle: ${formatPageTitle(t.title)}\ndescription: ${t.description}\n---\n\n${body}`,
    { headers: { 'Content-Type': 'text/markdown; charset=utf-8' } },
  );
};
