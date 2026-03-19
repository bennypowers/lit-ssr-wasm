/**
 * Minimal WASI preview 1 shim for running Javy WASM modules in the browser.
 * Maps stdin/stdout/stderr to JS buffers.
 */

let instance;

export function createWasi(stdinBuf) {
  let stdinPos = 0;
  const stdoutChunks = [];
  const stderrChunks = [];
  const decoder = new TextDecoder();

  return {
    get stdout() { return stdoutChunks.map(c => decoder.decode(c)).join(''); },
    get stderr() { return stderrChunks.map(c => decoder.decode(c)).join(''); },

    imports: {
      wasi_snapshot_preview1: {
        fd_read(fd, iovsPtr, iovsLen, nreadPtr) {
          if (fd !== 0) return 8;
          const mem = new DataView(instance.exports.memory.buffer);
          let totalRead = 0;
          for (let i = 0; i < iovsLen; i++) {
            const bufPtr = mem.getUint32(iovsPtr + i * 8, true);
            const bufLen = mem.getUint32(iovsPtr + i * 8 + 4, true);
            const remaining = stdinBuf.length - stdinPos;
            const toRead = Math.min(bufLen, remaining);
            if (toRead === 0) break;
            new Uint8Array(mem.buffer, bufPtr, toRead)
              .set(stdinBuf.subarray(stdinPos, stdinPos + toRead));
            stdinPos += toRead;
            totalRead += toRead;
          }
          mem.setUint32(nreadPtr, totalRead, true);
          return 0;
        },

        fd_write(fd, iovsPtr, iovsLen, nwrittenPtr) {
          if (fd !== 1 && fd !== 2) return 8;
          const mem = new DataView(instance.exports.memory.buffer);
          const chunks = fd === 1 ? stdoutChunks : stderrChunks;
          let totalWritten = 0;
          for (let i = 0; i < iovsLen; i++) {
            const bufPtr = mem.getUint32(iovsPtr + i * 8, true);
            const bufLen = mem.getUint32(iovsPtr + i * 8 + 4, true);
            chunks.push(new Uint8Array(mem.buffer, bufPtr, bufLen).slice());
            totalWritten += bufLen;
          }
          mem.setUint32(nwrittenPtr, totalWritten, true);
          return 0;
        },

        fd_close() { return 0; },
        fd_seek() { return 70; },
        fd_fdstat_get(fd, statPtr) {
          const mem = new DataView(instance.exports.memory.buffer);
          mem.setUint8(statPtr, fd === 0 ? 2 : 4);
          mem.setUint16(statPtr + 2, 0, true);
          mem.setBigUint64(statPtr + 8, 0n, true);
          mem.setBigUint64(statPtr + 16, 0n, true);
          return 0;
        },
        environ_get() { return 0; },
        environ_sizes_get(countPtr, sizePtr) {
          const mem = new DataView(instance.exports.memory.buffer);
          mem.setUint32(countPtr, 0, true);
          mem.setUint32(sizePtr, 0, true);
          return 0;
        },
        clock_time_get(id, precision, timePtr) {
          const mem = new DataView(instance.exports.memory.buffer);
          mem.setBigUint64(timePtr, BigInt(Date.now()) * 1_000_000n, true);
          return 0;
        },
        proc_exit(code) { throw new Error(`proc_exit(${code})`); },
      },
    },
  };
}

const encoder = new TextEncoder();

/**
 * Run a WASM module with the given stdin string.
 * Returns { stdout, stderr }.
 */
export async function run(wasmModule, stdin) {
  const wasi = createWasi(encoder.encode(stdin));
  try {
    instance = await WebAssembly.instantiate(wasmModule, wasi.imports);
    instance.exports._start();
  } catch (e) {
    if (!e.message?.includes('proc_exit(0)')) {
      throw new Error(wasi.stderr || e.message);
    }
  }
  return { stdout: wasi.stdout, stderr: wasi.stderr };
}
