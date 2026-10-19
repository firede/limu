import type { APIRoute } from 'astro';
import { docs } from '../lib/docs';
import { exampleTypeHref, exampleTypes } from '../lib/examples';
import { pageHref } from '../lib/site';

export const GET: APIRoute = ({ site }) => {
  const routes = [pageHref('introduction'), pageHref('examples'), ...exampleTypes.map(({ type }) => exampleTypeHref(type)), ...docs.map(({ id }) => pageHref(id))];
  const xml = `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">${routes.map((route) => `<url><loc>${new URL(route, site)}</loc></url>`).join('')}</urlset>\n`;
  return new Response(xml, { headers: { 'Content-Type': 'application/xml' } });
};
