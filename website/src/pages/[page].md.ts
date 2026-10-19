import type { APIRoute } from 'astro';
import { formatPageTitle } from '../lib/page-title';
import { docs, getDoc } from '../lib/docs';
import type { DocsPageId } from '../lib/site';

export function getStaticPaths() {
  return docs.map((doc) => ({ params: { page: doc.id } }));
}

export const GET: APIRoute = ({ params }) => new Response(getDoc(params.page as DocsPageId).source.replace(
  /^title: (.+)$/m, (_line, title: string) => `title: ${formatPageTitle(title)}`,
), {
  headers: { 'Content-Type': 'text/markdown; charset=utf-8' },
});
