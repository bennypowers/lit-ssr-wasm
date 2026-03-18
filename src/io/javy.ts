/** Javy WASM runtime I/O via the built-in Javy.IO API. */

declare const Javy: {
  IO: {
    readSync(fd: number, buffer: Uint8Array): number;
    writeSync(fd: number, buffer: Uint8Array): void;
  };
};

export function readStdin(): string {
  const chunks: Uint8Array[] = [];
  const buf = new Uint8Array(4096);
  for (;;) {
    const n = Javy.IO.readSync(0, buf);
    if (n === 0) break;
    chunks.push(buf.slice(0, n));
  }
  const total = chunks.reduce((s, c) => s + c.length, 0);
  const combined = new Uint8Array(total);
  let offset = 0;
  for (const c of chunks) {
    combined.set(c, offset);
    offset += c.length;
  }
  return new TextDecoder().decode(combined);
}

export function writeStdout(str: string): void {
  Javy.IO.writeSync(1, new TextEncoder().encode(str));
}

export function writeStderr(str: string): void {
  Javy.IO.writeSync(2, new TextEncoder().encode(str));
}
