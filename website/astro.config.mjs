import { defineConfig } from 'astro/config';
import { unified } from '@astrojs/markdown-remark';
import remarkGfm from 'remark-gfm';
import { highlightCode, highlightInline } from './src/lib/highlight.ts';
import { refreshDevCss } from './lib/dev-css.mjs';

const copyIcon = '<svg aria-hidden="true" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"><rect width="14" height="14" x="8" y="8" rx="2" ry="2"></rect><path d="M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2"></path></svg>';
const copyButton = `<button type="button" class="doc-code-copy" data-copy-code title="Copy code" aria-label="Copy code">${copyIcon}</button>`;
const escapeHtml = (value) => value.replace(/[&<>"']/g, (character) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[character] ?? character);
const codeBlockLabel = (node) => {
  const match = typeof node.meta === 'string' ? node.meta.match(/(?:^|\s)title=(?:"([^"]+)"|'([^']+)'|([^\s]+))/) : undefined;
  return match?.[1] ?? match?.[2] ?? match?.[3];
};
const languageAliases = { golang: 'go', typescript: 'ts', js: 'ts', javascript: 'ts', bash: 'shell', sh: 'shell', html: 'xml', svg: 'xml' };
const plainCode = (value) => value.split('\n').map((line, index) => `<span class="doc-code-line"><span class="doc-line-number" hidden>${index + 1}</span><span class="doc-code-source">${escapeHtml(line)}</span></span>`).join('');
const renderCodeBlock = (node) => {
  const label = codeBlockLabel(node);
  const language = languageAliases[node.lang] ?? node.lang ?? 'text';
  const highlighted = ['mermaid', 'go', 'ts', 'shell', 'css', 'xml'].includes(language) ? highlightCode(node.value, language) : plainCode(node.value);
  const caption = label
    ? `<figcaption><span>${escapeHtml(label)}</span><span class="doc-code-actions">${copyButton}</span></figcaption>`
    : copyButton;
  return `<figure class="doc-code${label ? '' : ' doc-code-unlabeled'}" data-code-block>${caption}<pre><code data-language="${escapeHtml(language)}">${highlighted}</code></pre></figure>`;
};

function staticCodeBlocks() {
  return (tree) => {
    const visit = (node, parent, index) => {
      if (node?.type === 'link' && /^https?:\/\//.test(node.url)) {
        node.data = {
          ...node.data,
          hProperties: {
            ...node.data?.hProperties,
            target: '_blank',
            rel: 'noopener noreferrer',
          },
        };
      }
      if (node?.type === 'inlineCode' && parent && typeof index === 'number') {
        const highlighted = highlightInline(node.value);
        // A single short word, such as sans-serif, is kept on one line instead of breaking at a hyphen.
        const word = !/\s/.test(node.value) && node.value.length <= 32;
        if (highlighted) parent.children[index] = { type: 'html', value: `<code class="inline-syntax${word ? ' code-word' : ''}">${highlighted}</code>` };
        else if (word) node.data = { ...node.data, hProperties: { ...node.data?.hProperties, className: ['code-word'] } };
        return;
      }
      if (node?.type === 'code' && parent && typeof index === 'number') {
        parent.children[index] = {
          type: 'html',
          value: renderCodeBlock(node),
        };
        return;
      }
      if (Array.isArray(node?.children)) node.children.forEach((child, childIndex) => visit(child, node, childIndex));
    };
    visit(tree, undefined, undefined);
  };
}

const textContent = (node) => node.type === 'text' ? node.value : (node.children ?? []).map(textContent).join('');

function scrollableTables() {
  return (tree) => {
    let heading = '';
    const visit = (node) => {
      if (!Array.isArray(node?.children)) return;
      node.children = node.children.map((child) => {
        if (child?.type === 'element' && /^h[1-6]$/.test(child.tagName)) heading = textContent(child);
        if (child?.type === 'element' && child.tagName === 'table') {
          return {
            type: 'element',
            tagName: 'div',
            properties: { className: ['table-wrap'], role: 'region', tabIndex: 0, ariaLabel: heading ? `${heading} table` : 'Table' },
            children: [child],
          };
        }
        visit(child);
        return child;
      });
    };
    visit(tree);
  };
}

export default defineConfig({
  site: 'https://limu.firede.com',
  output: 'static',
  trailingSlash: 'never',
  build: { format: 'file' },
  vite: {
    // The dev server keeps a cache of its own. A build or a check that runs beside it would
    // otherwise replace the prebundled dependencies it is serving.
    cacheDir: process.argv.includes('dev') ? 'node_modules/.vite-dev' : 'node_modules/.vite',
    plugins: [refreshDevCss()],
  },
  markdown: {
    syntaxHighlight: false,
    processor: unified({
      remarkPlugins: [remarkGfm, staticCodeBlocks],
      rehypePlugins: [scrollableTables],
    }),
  },
});
