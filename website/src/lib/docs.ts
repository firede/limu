import type { MarkdownInstance } from 'astro';
import { docsPages, type DocsPageId } from './site';

interface DocFrontmatter {
  title: string;
  description: string;
}

const modules = import.meta.glob<MarkdownInstance<DocFrontmatter>>('../content/docs/*.md', { eager: true });
const sources = import.meta.glob<string>('../content/docs/*.md', { eager: true, query: '?raw', import: 'default' });

export const docs = docsPages.filter((page) => `../content/docs/${page.id}.md` in modules).map((page) => {
  const key = `../content/docs/${page.id}.md`;
  return { ...page, module: modules[key], source: sources[key] };
});

export const getDoc = (id: DocsPageId) => docs.find((doc) => doc.id === id)!;
