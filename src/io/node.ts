/** Node.js I/O via the standard fs and process APIs. */

import { readFileSync } from 'node:fs';

let _buf: string | null = null;
let _pos = 0;

function ensureBuffer(): string {
  if (_buf === null) {
    _buf = readFileSync(0, 'utf-8');
    _pos = 0;
  }
  return _buf;
}

export function readStdin(): string {
  return readFileSync(0, 'utf-8');
}

/** Read stdin until a NUL byte. Returns the content or null at EOF. */
export function readUntilNul(): string | null {
  const buf = ensureBuffer();
  if (_pos >= buf.length) return null;
  const nulIdx = buf.indexOf('\0', _pos);
  if (nulIdx === -1) {
    const rest = buf.slice(_pos);
    _pos = buf.length;
    return rest || null;
  }
  const result = buf.slice(_pos, nulIdx);
  _pos = nulIdx + 1;
  return result;
}

/** Read a single line from stdin. Returns the line or null at EOF. */
export function readLine(): string | null {
  const buf = ensureBuffer();
  if (_pos >= buf.length) return null;
  const nlIdx = buf.indexOf('\n', _pos);
  if (nlIdx === -1) {
    const rest = buf.slice(_pos);
    _pos = buf.length;
    return rest || null;
  }
  const result = buf.slice(_pos, nlIdx);
  _pos = nlIdx + 1;
  return result;
}

export function writeStdout(str: string): void {
  process.stdout.write(str);
}

export function writeStderr(str: string): void {
  process.stderr.write(str);
}
