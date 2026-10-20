import { readFileSync } from 'node:fs';
import { join } from 'node:path';

// The pictures that limumygo/examples/pictures draws into public/pictures: one scene, laid
// out once and drawn four ways. platinum.png beside them is drawn by mygo-platinum's
// examples/diagrams, as docs/website.md says. A page shows them in a figure, the home page
// through renderPictures and a document through a paragraph of nothing but their images,
// which pictureParagraphs turns into the same figure.

// The PNG pictures are drawn at twice their size in DIPs, as the program draws them by
// default.
const scale = 2;

const escapeHtml = (value) => value.replace(/[&<>"']/g, (character) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[character] ?? character);
// A picture's file, from the website's directory, which Astro runs in: a page that imports
// this module is bundled elsewhere, where import.meta.url would not lead to it.
const file = (src) => join(process.cwd(), 'public', src);

// The pictures in the order they are shown, each with its name and what it shows.
export const scenePictures = [
  { src: '/pictures/scene.svg', title: 'The scene', alt: 'Boxes, paths and labels, with no style.' },
  { src: '/pictures/kohu.png', title: 'Kohu', alt: 'Drawn in limu’s light theme.' },
  { src: '/pictures/kohu-dark.png', title: 'KohuDark', alt: 'Only the colors change.' },
  { src: '/pictures/glass.png', title: 'Liquid Glass', alt: 'Each part drawn its own way, with MyGo’s glass.' },
];

// The same diagram laid out again and drawn in mygo-platinum's Platinum look.
export const platinumPicture = [
  { src: '/pictures/platinum.png', title: 'Platinum', alt: 'Laid out again in bitmap faces, and drawn a pixel at a time.' },
];

// picture writes a picture: the SVG in the page, so that it takes the page's color, and a
// PNG as an image the size it was drawn in DIPs.
const picture = ({ src, alt }) => {
  if (src.endsWith('.svg')) {
    return readFileSync(file(src), 'utf8').trim().replace('<svg ', `<svg role="img" aria-label="${escapeHtml(alt)}" `);
  }
  const png = readFileSync(file(src));
  const width = png.readUInt32BE(16) / scale;
  const height = png.readUInt32BE(20) / scale;
  return `<img src="${escapeHtml(src)}" width="${width}" height="${height}" alt="${escapeHtml(alt)}" loading="lazy" decoding="async">`;
};

// renderPictures writes pictures as a figure, each in a row with its name and what it
// shows. Several pictures are the scene drawn several ways, and say so; one stands alone.
export const renderPictures = (pictures) => {
  const rows = pictures.map((p) => `<div class="scene-picture"><p class="scene-picture-name"><strong>${escapeHtml(p.title)}</strong> <span>${escapeHtml(p.alt)}</span></p><div class="scene-picture-frame">${picture(p)}</div></div>`);
  const caption = pictures.length > 1 ? '<figcaption>One scene, laid out once and drawn four ways</figcaption>' : '';
  return `<figure class="diagram-figure scene-pictures">${caption}${rows.join('')}</figure>`;
};

const isPicture = (node) => node.type === 'image' && node.url.startsWith('/pictures/');
const isSpace = (node) => node.type === 'text' && !node.value.trim();

// pictureParagraphs is a remark plugin that writes a paragraph of nothing but pictures from
// public/pictures as their figure.
export function pictureParagraphs() {
  return (tree) => {
    const visit = (node) => {
      node.children?.forEach((child, index) => {
        if (child.type === 'paragraph' && child.children.some(isPicture) && child.children.every((c) => isPicture(c) || isSpace(c))) {
          const pictures = child.children.filter(isPicture).map((c) => ({ src: c.url, title: c.title ?? '', alt: c.alt ?? '' }));
          node.children[index] = { type: 'html', value: renderPictures(pictures) };
          return;
        }
        visit(child);
      });
    };
    visit(tree);
  };
}
