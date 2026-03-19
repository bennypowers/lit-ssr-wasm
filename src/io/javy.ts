/** Javy WASM runtime I/O via the built-in Javy.IO API. */

declare const Javy: {
  IO: {
    readSync(fd: number, buffer: Uint8Array): number;
    writeSync(fd: number, buffer: Uint8Array): void;
  };
};

const NL = 0x0a;
const NUL = 0x00;

export function readStdin(): string {
  const chunks: Uint8Array[] = [];
  const buf = new Uint8Array(4096);
  for (;;) {
    const n = Javy.IO.readSync(0, buf);
    if (n === 0) break;
    chunks.push(buf.slice(0, n));
  }
  return decodeChunks(chunks);
}

/** Read stdin until a NUL byte. Returns the content or null at EOF. */
export function readUntilNul(): string | null {
  const chunks: number[] = [];
  const buf = new Uint8Array(1);
  for (;;) {
    const n = Javy.IO.readSync(0, buf);
    if (n === 0) return chunks.length > 0 ? new TextDecoder().decode(new Uint8Array(chunks)) : null;
    if (buf[0] === NUL) return new TextDecoder().decode(new Uint8Array(chunks));
    chunks.push(buf[0]);
  }
}

/** Read a single line from stdin (up to '\n'). Returns the line or null at EOF. */
export function readLine(): string | null {
  const chunks: number[] = [];
  const buf = new Uint8Array(1);
  for (;;) {
    const n = Javy.IO.readSync(0, buf);
    if (n === 0) return chunks.length > 0 ? new TextDecoder().decode(new Uint8Array(chunks)) : null;
    if (buf[0] === NL) return new TextDecoder().decode(new Uint8Array(chunks));
    chunks.push(buf[0]);
  }
}

export function writeStdout(str: string): void {
  Javy.IO.writeSync(1, new TextEncoder().encode(str));
}

export function writeStderr(str: string): void {
  Javy.IO.writeSync(2, new TextEncoder().encode(str));
}

function decodeChunks(chunks: Uint8Array[]): string {
  const total = chunks.reduce((s, c) => s + c.length, 0);
  const combined = new Uint8Array(total);
  let offset = 0;
  for (const c of chunks) {
    combined.set(c, offset);
    offset += c.length;
  }
  return new TextDecoder().decode(combined);
}
