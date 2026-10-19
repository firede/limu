// Draws the icon's PNGs and favicon.ico from the SVGs gen writes, in headless Chrome, as
// Go's standard library has no rasterizer: the 16-pixel cut at 16, the icon at 32 and 48,
// and the icon at 180 for Apple's devices, on the water's color to its corners and with no
// rim, as they round an icon themselves. The files are committed, so that a build needs no
// Chrome. Run it from the website directory, after go run ./gen:
//
//	node scripts/icons.mjs
//
// CHROME names the browser, if it is not where macOS keeps Google Chrome.
import { execFileSync } from 'node:child_process';
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

const chrome = process.env.CHROME ?? '/Applications/Google Chrome.app/Contents/MacOS/Google Chrome';
const dataURI = (file) => `data:image/svg+xml;base64,${readFileSync(file).toString('base64')}`;

// Each drawing: a name, the SVG it is drawn from, and its size.
const drawings = [
  ['favicon-16', 'public/icon-16.svg', 16],
  ['favicon-32', 'public/icon.svg', 32],
  ['favicon-48', 'public/icon.svg', 48],
  ['icon-180', 'src/generated/icon-full.svg', 180],
];

// The page draws each SVG on a canvas of its size and writes the PNG into the page, which
// Chrome prints. An image from a data URI leaves the canvas readable.
const page = `<!doctype html><meta charset="utf-8"><body><script>
const drawings = ${JSON.stringify(drawings.map(([name, file, size]) => ({ name, src: dataURI(file), size })))};
Promise.all(drawings.map((d) => new Promise((resolve) => {
  const img = new Image();
  img.onload = () => {
    const c = document.createElement('canvas');
    c.width = c.height = d.size;
    c.getContext('2d').drawImage(img, 0, 0, d.size, d.size);
    resolve(d.name + ' ' + c.toDataURL('image/png'));
  };
  img.src = d.src;
}))).then((lines) => {
  const pre = document.createElement('pre');
  pre.id = 'out';
  pre.textContent = lines.join('\\n');
  document.body.append(pre);
});
</script>`;

const dir = mkdtempSync(join(tmpdir(), 'limu-icons-'));
let dom;
try {
  writeFileSync(join(dir, 'icons.html'), page);
  dom = execFileSync(chrome, ['--headless=new', '--disable-gpu', '--virtual-time-budget=5000', '--dump-dom', `file://${join(dir, 'icons.html')}`], { encoding: 'utf8', stdio: ['ignore', 'pipe', 'ignore'] });
} finally {
  rmSync(dir, { recursive: true, force: true });
}
const out = dom.match(/<pre id="out">([\s\S]*?)<\/pre>/);
if (!out) throw new Error('Chrome drew no icons');
const pngs = new Map(out[1].trim().split('\n').map((line) => {
  const [name, url] = line.split(' ');
  return [name, Buffer.from(url.slice(url.indexOf(',') + 1), 'base64')];
}));

writeFileSync('public/icon-180.png', pngs.get('icon-180'));

// favicon.ico holds the three favicons as PNGs: a header, a directory entry for each, and
// the PNGs after them.
const icons = ['favicon-16', 'favicon-32', 'favicon-48'].map((name) => pngs.get(name));
const header = Buffer.alloc(6);
header.writeUInt16LE(0, 0);
header.writeUInt16LE(1, 2);
header.writeUInt16LE(icons.length, 4);
let offset = 6 + 16 * icons.length;
const entries = icons.map((png) => {
  const size = png.readUInt32BE(16); // the width in the PNG's IHDR chunk
  const e = Buffer.alloc(16);
  e.writeUInt8(size % 256, 0);
  e.writeUInt8(size % 256, 1);
  e.writeUInt16LE(1, 4); // color planes
  e.writeUInt16LE(32, 6); // bits per pixel
  e.writeUInt32LE(png.length, 8);
  e.writeUInt32LE(offset, 12);
  offset += png.length;
  return e;
});
writeFileSync('public/favicon.ico', Buffer.concat([header, ...entries, ...icons]));
console.log('icons: public/icon-180.png and public/favicon.ico (16, 32, 48)');
