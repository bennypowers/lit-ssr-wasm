/** Node.js I/O via the standard fs and process APIs. */

import { readFileSync } from 'node:fs';

export function readStdin(): string {
  return readFileSync(0, 'utf-8');
}

export function writeStdout(str: string): void {
  process.stdout.write(str);
}

export function writeStderr(str: string): void {
  process.stderr.write(str);
}
