/**
 * Minimal live-reloading dev server for the docs/ directory.
 *
 * Watches source files (pages/, src/) for changes, runs `npm run build:demo`
 * via wireit (which handles caching and dependency ordering), then notifies
 * connected browsers to reload via Server-Sent Events.
 */

import { createServer, type ServerResponse } from 'node:http';
import { spawn } from 'node:child_process';
import { readFileSync, statSync, watch } from 'node:fs';
import { resolve, extname, join } from 'node:path';

const html = String.raw; // for editor highlighting
const PORT = parseInt(process.env.PORT || '3000', 10);
const root = resolve(import.meta.dirname!, '..');
const siteDir = resolve(root, '_site');

const MIME: Record<string, string> = {
  '.html': 'text/html',
  '.css': 'text/css',
  '.js': 'text/javascript',
  '.wasm': 'application/wasm',
  '.json': 'application/json',
  '.svg': 'image/svg+xml',
  '.png': 'image/png',
};

const RELOAD_SCRIPT = html`<script>new EventSource("/__sse").onmessage=()=>location.reload()</script>`;

const clients = new Set<ServerResponse>();

const server = createServer((req, res) => {
  const url = new URL(req.url!, `http://localhost:${PORT}`);

  if (url.pathname === '/__sse') {
    res.writeHead(200, {
      'content-type': 'text/event-stream',
      'cache-control': 'no-cache',
      'connection': 'keep-alive',
    });
    clients.add(res);
    req.on('close', () => clients.delete(res));
    return;
  }

  let urlPath = url.pathname;
  if (urlPath.endsWith('/')) urlPath += 'index.html';

  const filePath = join(siteDir, urlPath);

  if (!filePath.startsWith(siteDir)) {
    res.writeHead(403);
    res.end();
    return;
  }

  try {
    const stat = statSync(filePath);
    if (!stat.isFile()) throw new Error('not a file');
  } catch {
    res.writeHead(404, { 'content-type': 'text/plain' });
    res.end('Not found');
    return;
  }

  const ext = extname(filePath);
  const contentType = MIME[ext] || 'application/octet-stream';
  let body: Buffer | string = readFileSync(filePath);

  if (ext === '.html') {
    body = body.toString('utf-8').replace('</body>', RELOAD_SCRIPT + '</body>');
  }

  res.writeHead(200, { 'content-type': contentType });
  res.end(body);
});

// Run wireit build:demo, notify browsers on completion
let building = false;
let pendingBuild = false;

function build() {
  if (building) {
    pendingBuild = true;
    return;
  }
  building = true;
  const proc = spawn('npm', ['run', 'build:demo'], { cwd: root, stdio: 'inherit' });
  proc.on('close', code => {
    building = false;
    if (code === 0) {
      for (const client of clients) {
        client.write('data: reload\n\n');
      }
    }
    if (pendingBuild) {
      pendingBuild = false;
      build();
    }
  });
}

// Watch source directories for changes
let debounce: ReturnType<typeof setTimeout> | null = null;
for (const dir of ['docs', 'src']) {
  watch(resolve(root, dir), { recursive: true }, () => {
    if (debounce) clearTimeout(debounce);
    debounce = setTimeout(build, 200);
  });
}

server.listen(PORT, () => {
  console.log(`Dev server: http://localhost:${PORT}`);
});
