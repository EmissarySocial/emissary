"use strict";
(() => {
  var __create = Object.create;
  var __defProp = Object.defineProperty;
  var __getOwnPropDesc = Object.getOwnPropertyDescriptor;
  var __getOwnPropNames = Object.getOwnPropertyNames;
  var __getProtoOf = Object.getPrototypeOf;
  var __hasOwnProp = Object.prototype.hasOwnProperty;
  var __require = /* @__PURE__ */ ((x) => typeof require !== "undefined" ? require : typeof Proxy !== "undefined" ? new Proxy(x, {
    get: (a, b) => (typeof require !== "undefined" ? require : a)[b]
  }) : x)(function(x) {
    if (typeof require !== "undefined") return require.apply(this, arguments);
    throw Error('Dynamic require of "' + x + '" is not supported');
  });
  var __esm = (fn, res) => function __init() {
    return fn && (res = (0, fn[__getOwnPropNames(fn)[0]])(fn = 0)), res;
  };
  var __export = (target, all) => {
    for (var name in all)
      __defProp(target, name, { get: all[name], enumerable: true });
  };
  var __copyProps = (to, from, except, desc) => {
    if (from && typeof from === "object" || typeof from === "function") {
      for (let key of __getOwnPropNames(from))
        if (!__hasOwnProp.call(to, key) && key !== except)
          __defProp(to, key, { get: () => from[key], enumerable: !(desc = __getOwnPropDesc(from, key)) || desc.enumerable });
    }
    return to;
  };
  var __toESM = (mod2, isNodeMode, target) => (target = mod2 != null ? __create(__getProtoOf(mod2)) : {}, __copyProps(
    // If the importer is in node compatibility mode or this is not an ESM
    // file that has been converted to a CommonJS file using a Babel-
    // compatible transform (i.e. "__esModule" has not been set), then set
    // "default" to the CommonJS "module.exports" for node compatibility.
    isNodeMode || !mod2 || !mod2.__esModule ? __defProp(target, "default", { value: mod2, enumerable: true }) : target,
    mod2
  ));

  // node_modules/@noble/ciphers/utils.js
  function isBytes2(a) {
    return a instanceof Uint8Array || ArrayBuffer.isView(a) && a.constructor.name === "Uint8Array";
  }
  function abool(b) {
    if (typeof b !== "boolean")
      throw new Error(`boolean expected, not ${b}`);
  }
  function anumber2(n) {
    if (!Number.isSafeInteger(n) || n < 0)
      throw new Error("positive integer expected, got " + n);
  }
  function abytes2(value, length, title = "") {
    const bytes = isBytes2(value);
    const len = value?.length;
    const needsLen = length !== void 0;
    if (!bytes || needsLen && len !== length) {
      const prefix = title && `"${title}" `;
      const ofLen = needsLen ? ` of length ${length}` : "";
      const got = bytes ? `length=${len}` : `type=${typeof value}`;
      throw new Error(prefix + "expected Uint8Array" + ofLen + ", got " + got);
    }
    return value;
  }
  function aexists2(instance, checkFinished = true) {
    if (instance.destroyed)
      throw new Error("Hash instance has been destroyed");
    if (checkFinished && instance.finished)
      throw new Error("Hash#digest() has already been called");
  }
  function aoutput2(out, instance) {
    abytes2(out, void 0, "output");
    const min = instance.outputLen;
    if (out.length < min) {
      throw new Error("digestInto() expects output buffer of length at least " + min);
    }
  }
  function u32(arr) {
    return new Uint32Array(arr.buffer, arr.byteOffset, Math.floor(arr.byteLength / 4));
  }
  function clean2(...arrays) {
    for (let i = 0; i < arrays.length; i++) {
      arrays[i].fill(0);
    }
  }
  function createView2(arr) {
    return new DataView(arr.buffer, arr.byteOffset, arr.byteLength);
  }
  function checkOpts(defaults, opts) {
    if (opts == null || typeof opts !== "object")
      throw new Error("options must be defined");
    const merged = Object.assign(defaults, opts);
    return merged;
  }
  function equalBytes(a, b) {
    if (a.length !== b.length)
      return false;
    let diff = 0;
    for (let i = 0; i < a.length; i++)
      diff |= a[i] ^ b[i];
    return diff === 0;
  }
  function getOutput(expectedLength, out, onlyAligned = true) {
    if (out === void 0)
      return new Uint8Array(expectedLength);
    if (out.length !== expectedLength)
      throw new Error('"output" expected Uint8Array of length ' + expectedLength + ", got: " + out.length);
    if (onlyAligned && !isAligned32(out))
      throw new Error("invalid output, must be aligned");
    return out;
  }
  function u64Lengths(dataLength, aadLength, isLE2) {
    abool(isLE2);
    const num = new Uint8Array(16);
    const view = createView2(num);
    view.setBigUint64(0, BigInt(aadLength), isLE2);
    view.setBigUint64(8, BigInt(dataLength), isLE2);
    return num;
  }
  function isAligned32(bytes) {
    return bytes.byteOffset % 4 === 0;
  }
  function copyBytes2(bytes) {
    return Uint8Array.from(bytes);
  }
  function randomBytes(bytesLength = 32) {
    const cr = typeof globalThis === "object" ? globalThis.crypto : null;
    if (typeof cr?.getRandomValues !== "function")
      throw new Error("crypto.getRandomValues must be defined");
    return cr.getRandomValues(new Uint8Array(bytesLength));
  }
  var isLE, wrapCipher;
  var init_utils = __esm({
    "node_modules/@noble/ciphers/utils.js"() {
      isLE = /* @__PURE__ */ (() => new Uint8Array(new Uint32Array([287454020]).buffer)[0] === 68)();
      wrapCipher = /* @__NO_SIDE_EFFECTS__ */ (params, constructor) => {
        function wrappedCipher(key, ...args) {
          abytes2(key, void 0, "key");
          if (!isLE)
            throw new Error("Non little-endian hardware is not yet supported");
          if (params.nonceLength !== void 0) {
            const nonce = args[0];
            abytes2(nonce, params.varSizeNonce ? void 0 : params.nonceLength, "nonce");
          }
          const tagl = params.tagLength;
          if (tagl && args[1] !== void 0)
            abytes2(args[1], void 0, "AAD");
          const cipher = constructor(key, ...args);
          const checkOutput = (fnLength, output) => {
            if (output !== void 0) {
              if (fnLength !== 2)
                throw new Error("cipher output not supported");
              abytes2(output, void 0, "output");
            }
          };
          let called = false;
          const wrCipher = {
            encrypt(data, output) {
              if (called)
                throw new Error("cannot encrypt() twice with same key + nonce");
              called = true;
              abytes2(data);
              checkOutput(cipher.encrypt.length, output);
              return cipher.encrypt(data, output);
            },
            decrypt(data, output) {
              abytes2(data);
              if (tagl && data.length < tagl)
                throw new Error('"ciphertext" expected length bigger than tagLength=' + tagl);
              checkOutput(cipher.decrypt.length, output);
              return cipher.decrypt(data, output);
            }
          };
          return wrCipher;
        }
        Object.assign(wrappedCipher, params);
        return wrappedCipher;
      };
    }
  });

  // node_modules/@noble/ciphers/_arx.js
  function rotl(a, b) {
    return a << b | a >>> 32 - b;
  }
  function isAligned322(b) {
    return b.byteOffset % 4 === 0;
  }
  function runCipher(core, sigma, key, nonce, data, output, counter, rounds) {
    const len = data.length;
    const block = new Uint8Array(BLOCK_LEN);
    const b32 = u32(block);
    const isAligned = isAligned322(data) && isAligned322(output);
    const d32 = isAligned ? u32(data) : U32_EMPTY;
    const o32 = isAligned ? u32(output) : U32_EMPTY;
    for (let pos = 0; pos < len; counter++) {
      core(sigma, key, nonce, b32, counter, rounds);
      if (counter >= MAX_COUNTER)
        throw new Error("arx: counter overflow");
      const take = Math.min(BLOCK_LEN, len - pos);
      if (isAligned && take === BLOCK_LEN) {
        const pos32 = pos / 4;
        if (pos % 4 !== 0)
          throw new Error("arx: invalid block position");
        for (let j = 0, posj; j < BLOCK_LEN32; j++) {
          posj = pos32 + j;
          o32[posj] = d32[posj] ^ b32[j];
        }
        pos += BLOCK_LEN;
        continue;
      }
      for (let j = 0, posj; j < take; j++) {
        posj = pos + j;
        output[posj] = data[posj] ^ block[j];
      }
      pos += take;
    }
  }
  function createCipher(core, opts) {
    const { allowShortKeys, extendNonceFn, counterLength, counterRight, rounds } = checkOpts({ allowShortKeys: false, counterLength: 8, counterRight: false, rounds: 20 }, opts);
    if (typeof core !== "function")
      throw new Error("core must be a function");
    anumber2(counterLength);
    anumber2(rounds);
    abool(counterRight);
    abool(allowShortKeys);
    return (key, nonce, data, output, counter = 0) => {
      abytes2(key, void 0, "key");
      abytes2(nonce, void 0, "nonce");
      abytes2(data, void 0, "data");
      const len = data.length;
      if (output === void 0)
        output = new Uint8Array(len);
      abytes2(output, void 0, "output");
      anumber2(counter);
      if (counter < 0 || counter >= MAX_COUNTER)
        throw new Error("arx: counter overflow");
      if (output.length < len)
        throw new Error(`arx: output (${output.length}) is shorter than data (${len})`);
      const toClean = [];
      let l = key.length;
      let k;
      let sigma;
      if (l === 32) {
        toClean.push(k = copyBytes2(key));
        sigma = sigma32_32;
      } else if (l === 16 && allowShortKeys) {
        k = new Uint8Array(32);
        k.set(key);
        k.set(key, 16);
        sigma = sigma16_32;
        toClean.push(k);
      } else {
        abytes2(key, 32, "arx key");
        throw new Error("invalid key size");
      }
      if (!isAligned322(nonce))
        toClean.push(nonce = copyBytes2(nonce));
      const k32 = u32(k);
      if (extendNonceFn) {
        if (nonce.length !== 24)
          throw new Error(`arx: extended nonce must be 24 bytes`);
        extendNonceFn(sigma, k32, u32(nonce.subarray(0, 16)), k32);
        nonce = nonce.subarray(16);
      }
      const nonceNcLen = 16 - counterLength;
      if (nonceNcLen !== nonce.length)
        throw new Error(`arx: nonce must be ${nonceNcLen} or 16 bytes`);
      if (nonceNcLen !== 12) {
        const nc = new Uint8Array(12);
        nc.set(nonce, counterRight ? 0 : 12 - nonce.length);
        nonce = nc;
        toClean.push(nonce);
      }
      const n32 = u32(nonce);
      runCipher(core, sigma, k32, n32, data, output, counter, rounds);
      clean2(...toClean);
      return output;
    };
  }
  var encodeStr, sigma16, sigma32, sigma16_32, sigma32_32, BLOCK_LEN, BLOCK_LEN32, MAX_COUNTER, U32_EMPTY, _XorStreamPRG, createPRG;
  var init_arx = __esm({
    "node_modules/@noble/ciphers/_arx.js"() {
      init_utils();
      encodeStr = (str) => Uint8Array.from(str.split(""), (c) => c.charCodeAt(0));
      sigma16 = encodeStr("expand 16-byte k");
      sigma32 = encodeStr("expand 32-byte k");
      sigma16_32 = u32(sigma16);
      sigma32_32 = u32(sigma32);
      BLOCK_LEN = 64;
      BLOCK_LEN32 = 16;
      MAX_COUNTER = 2 ** 32 - 1;
      U32_EMPTY = Uint32Array.of();
      _XorStreamPRG = class __XorStreamPRG {
        blockLen;
        keyLen;
        nonceLen;
        state;
        buf;
        key;
        nonce;
        pos;
        ctr;
        cipher;
        constructor(cipher, blockLen, keyLen, nonceLen, seed) {
          this.cipher = cipher;
          this.blockLen = blockLen;
          this.keyLen = keyLen;
          this.nonceLen = nonceLen;
          this.state = new Uint8Array(this.keyLen + this.nonceLen);
          this.reseed(seed);
          this.ctr = 0;
          this.pos = this.blockLen;
          this.buf = new Uint8Array(this.blockLen);
          this.key = this.state.subarray(0, this.keyLen);
          this.nonce = this.state.subarray(this.keyLen);
        }
        reseed(seed) {
          abytes2(seed);
          if (!seed || seed.length === 0)
            throw new Error("entropy required");
          for (let i = 0; i < seed.length; i++)
            this.state[i % this.state.length] ^= seed[i];
          this.ctr = 0;
          this.pos = this.blockLen;
        }
        addEntropy(seed) {
          this.state.set(this.randomBytes(this.state.length));
          this.reseed(seed);
        }
        randomBytes(len) {
          anumber2(len);
          if (len === 0)
            return new Uint8Array(0);
          const out = new Uint8Array(len);
          let outPos = 0;
          if (this.pos < this.blockLen) {
            const take = Math.min(len, this.blockLen - this.pos);
            out.set(this.buf.subarray(this.pos, this.pos + take), 0);
            this.pos += take;
            outPos += take;
            if (outPos === len)
              return out;
          }
          const blocks = Math.floor((len - outPos) / this.blockLen);
          if (blocks > 0) {
            const blockBytes = blocks * this.blockLen;
            const b = out.subarray(outPos, outPos + blockBytes);
            this.cipher(this.key, this.nonce, b, b, this.ctr);
            this.ctr += blocks;
            outPos += blockBytes;
          }
          const left2 = len - outPos;
          if (left2 > 0) {
            this.buf.fill(0);
            this.cipher(this.key, this.nonce, this.buf, this.buf, this.ctr++);
            out.set(this.buf.subarray(0, left2), outPos);
            this.pos = left2;
          }
          return out;
        }
        clone() {
          return new __XorStreamPRG(this.cipher, this.blockLen, this.keyLen, this.nonceLen, this.randomBytes(this.state.length));
        }
        clean() {
          this.pos = 0;
          this.ctr = 0;
          this.buf.fill(0);
          this.state.fill(0);
        }
      };
      createPRG = (cipher, blockLen, keyLen, nonceLen) => {
        return (seed = randomBytes(32)) => new _XorStreamPRG(cipher, blockLen, keyLen, nonceLen, seed);
      };
    }
  });

  // node_modules/@noble/ciphers/_poly1305.js
  function u8to16(a, i) {
    return a[i++] & 255 | (a[i++] & 255) << 8;
  }
  function wrapConstructorWithKey(hashCons) {
    const hashC = (msg, key) => hashCons(key).update(msg).digest();
    const tmp = hashCons(new Uint8Array(32));
    hashC.outputLen = tmp.outputLen;
    hashC.blockLen = tmp.blockLen;
    hashC.create = (key) => hashCons(key);
    return hashC;
  }
  var Poly1305, poly1305;
  var init_poly1305 = __esm({
    "node_modules/@noble/ciphers/_poly1305.js"() {
      init_utils();
      Poly1305 = class {
        blockLen = 16;
        outputLen = 16;
        buffer = new Uint8Array(16);
        r = new Uint16Array(10);
        // Allocating 1 array with .subarray() here is slower than 3
        h = new Uint16Array(10);
        pad = new Uint16Array(8);
        pos = 0;
        finished = false;
        // Can be speed-up using BigUint64Array, at the cost of complexity
        constructor(key) {
          key = copyBytes2(abytes2(key, 32, "key"));
          const t0 = u8to16(key, 0);
          const t1 = u8to16(key, 2);
          const t2 = u8to16(key, 4);
          const t3 = u8to16(key, 6);
          const t4 = u8to16(key, 8);
          const t5 = u8to16(key, 10);
          const t6 = u8to16(key, 12);
          const t7 = u8to16(key, 14);
          this.r[0] = t0 & 8191;
          this.r[1] = (t0 >>> 13 | t1 << 3) & 8191;
          this.r[2] = (t1 >>> 10 | t2 << 6) & 7939;
          this.r[3] = (t2 >>> 7 | t3 << 9) & 8191;
          this.r[4] = (t3 >>> 4 | t4 << 12) & 255;
          this.r[5] = t4 >>> 1 & 8190;
          this.r[6] = (t4 >>> 14 | t5 << 2) & 8191;
          this.r[7] = (t5 >>> 11 | t6 << 5) & 8065;
          this.r[8] = (t6 >>> 8 | t7 << 8) & 8191;
          this.r[9] = t7 >>> 5 & 127;
          for (let i = 0; i < 8; i++)
            this.pad[i] = u8to16(key, 16 + 2 * i);
        }
        process(data, offset, isLast = false) {
          const hibit = isLast ? 0 : 1 << 11;
          const { h, r } = this;
          const r0 = r[0];
          const r1 = r[1];
          const r2 = r[2];
          const r3 = r[3];
          const r4 = r[4];
          const r5 = r[5];
          const r6 = r[6];
          const r7 = r[7];
          const r8 = r[8];
          const r9 = r[9];
          const t0 = u8to16(data, offset + 0);
          const t1 = u8to16(data, offset + 2);
          const t2 = u8to16(data, offset + 4);
          const t3 = u8to16(data, offset + 6);
          const t4 = u8to16(data, offset + 8);
          const t5 = u8to16(data, offset + 10);
          const t6 = u8to16(data, offset + 12);
          const t7 = u8to16(data, offset + 14);
          let h0 = h[0] + (t0 & 8191);
          let h1 = h[1] + ((t0 >>> 13 | t1 << 3) & 8191);
          let h2 = h[2] + ((t1 >>> 10 | t2 << 6) & 8191);
          let h3 = h[3] + ((t2 >>> 7 | t3 << 9) & 8191);
          let h4 = h[4] + ((t3 >>> 4 | t4 << 12) & 8191);
          let h5 = h[5] + (t4 >>> 1 & 8191);
          let h6 = h[6] + ((t4 >>> 14 | t5 << 2) & 8191);
          let h7 = h[7] + ((t5 >>> 11 | t6 << 5) & 8191);
          let h8 = h[8] + ((t6 >>> 8 | t7 << 8) & 8191);
          let h9 = h[9] + (t7 >>> 5 | hibit);
          let c = 0;
          let d0 = c + h0 * r0 + h1 * (5 * r9) + h2 * (5 * r8) + h3 * (5 * r7) + h4 * (5 * r6);
          c = d0 >>> 13;
          d0 &= 8191;
          d0 += h5 * (5 * r5) + h6 * (5 * r4) + h7 * (5 * r3) + h8 * (5 * r2) + h9 * (5 * r1);
          c += d0 >>> 13;
          d0 &= 8191;
          let d1 = c + h0 * r1 + h1 * r0 + h2 * (5 * r9) + h3 * (5 * r8) + h4 * (5 * r7);
          c = d1 >>> 13;
          d1 &= 8191;
          d1 += h5 * (5 * r6) + h6 * (5 * r5) + h7 * (5 * r4) + h8 * (5 * r3) + h9 * (5 * r2);
          c += d1 >>> 13;
          d1 &= 8191;
          let d2 = c + h0 * r2 + h1 * r1 + h2 * r0 + h3 * (5 * r9) + h4 * (5 * r8);
          c = d2 >>> 13;
          d2 &= 8191;
          d2 += h5 * (5 * r7) + h6 * (5 * r6) + h7 * (5 * r5) + h8 * (5 * r4) + h9 * (5 * r3);
          c += d2 >>> 13;
          d2 &= 8191;
          let d3 = c + h0 * r3 + h1 * r2 + h2 * r1 + h3 * r0 + h4 * (5 * r9);
          c = d3 >>> 13;
          d3 &= 8191;
          d3 += h5 * (5 * r8) + h6 * (5 * r7) + h7 * (5 * r6) + h8 * (5 * r5) + h9 * (5 * r4);
          c += d3 >>> 13;
          d3 &= 8191;
          let d4 = c + h0 * r4 + h1 * r3 + h2 * r2 + h3 * r1 + h4 * r0;
          c = d4 >>> 13;
          d4 &= 8191;
          d4 += h5 * (5 * r9) + h6 * (5 * r8) + h7 * (5 * r7) + h8 * (5 * r6) + h9 * (5 * r5);
          c += d4 >>> 13;
          d4 &= 8191;
          let d5 = c + h0 * r5 + h1 * r4 + h2 * r3 + h3 * r2 + h4 * r1;
          c = d5 >>> 13;
          d5 &= 8191;
          d5 += h5 * r0 + h6 * (5 * r9) + h7 * (5 * r8) + h8 * (5 * r7) + h9 * (5 * r6);
          c += d5 >>> 13;
          d5 &= 8191;
          let d6 = c + h0 * r6 + h1 * r5 + h2 * r4 + h3 * r3 + h4 * r2;
          c = d6 >>> 13;
          d6 &= 8191;
          d6 += h5 * r1 + h6 * r0 + h7 * (5 * r9) + h8 * (5 * r8) + h9 * (5 * r7);
          c += d6 >>> 13;
          d6 &= 8191;
          let d7 = c + h0 * r7 + h1 * r6 + h2 * r5 + h3 * r4 + h4 * r3;
          c = d7 >>> 13;
          d7 &= 8191;
          d7 += h5 * r2 + h6 * r1 + h7 * r0 + h8 * (5 * r9) + h9 * (5 * r8);
          c += d7 >>> 13;
          d7 &= 8191;
          let d8 = c + h0 * r8 + h1 * r7 + h2 * r6 + h3 * r5 + h4 * r4;
          c = d8 >>> 13;
          d8 &= 8191;
          d8 += h5 * r3 + h6 * r2 + h7 * r1 + h8 * r0 + h9 * (5 * r9);
          c += d8 >>> 13;
          d8 &= 8191;
          let d9 = c + h0 * r9 + h1 * r8 + h2 * r7 + h3 * r6 + h4 * r5;
          c = d9 >>> 13;
          d9 &= 8191;
          d9 += h5 * r4 + h6 * r3 + h7 * r2 + h8 * r1 + h9 * r0;
          c += d9 >>> 13;
          d9 &= 8191;
          c = (c << 2) + c | 0;
          c = c + d0 | 0;
          d0 = c & 8191;
          c = c >>> 13;
          d1 += c;
          h[0] = d0;
          h[1] = d1;
          h[2] = d2;
          h[3] = d3;
          h[4] = d4;
          h[5] = d5;
          h[6] = d6;
          h[7] = d7;
          h[8] = d8;
          h[9] = d9;
        }
        finalize() {
          const { h, pad } = this;
          const g = new Uint16Array(10);
          let c = h[1] >>> 13;
          h[1] &= 8191;
          for (let i = 2; i < 10; i++) {
            h[i] += c;
            c = h[i] >>> 13;
            h[i] &= 8191;
          }
          h[0] += c * 5;
          c = h[0] >>> 13;
          h[0] &= 8191;
          h[1] += c;
          c = h[1] >>> 13;
          h[1] &= 8191;
          h[2] += c;
          g[0] = h[0] + 5;
          c = g[0] >>> 13;
          g[0] &= 8191;
          for (let i = 1; i < 10; i++) {
            g[i] = h[i] + c;
            c = g[i] >>> 13;
            g[i] &= 8191;
          }
          g[9] -= 1 << 13;
          let mask = (c ^ 1) - 1;
          for (let i = 0; i < 10; i++)
            g[i] &= mask;
          mask = ~mask;
          for (let i = 0; i < 10; i++)
            h[i] = h[i] & mask | g[i];
          h[0] = (h[0] | h[1] << 13) & 65535;
          h[1] = (h[1] >>> 3 | h[2] << 10) & 65535;
          h[2] = (h[2] >>> 6 | h[3] << 7) & 65535;
          h[3] = (h[3] >>> 9 | h[4] << 4) & 65535;
          h[4] = (h[4] >>> 12 | h[5] << 1 | h[6] << 14) & 65535;
          h[5] = (h[6] >>> 2 | h[7] << 11) & 65535;
          h[6] = (h[7] >>> 5 | h[8] << 8) & 65535;
          h[7] = (h[8] >>> 8 | h[9] << 5) & 65535;
          let f = h[0] + pad[0];
          h[0] = f & 65535;
          for (let i = 1; i < 8; i++) {
            f = (h[i] + pad[i] | 0) + (f >>> 16) | 0;
            h[i] = f & 65535;
          }
          clean2(g);
        }
        update(data) {
          aexists2(this);
          abytes2(data);
          data = copyBytes2(data);
          const { buffer, blockLen } = this;
          const len = data.length;
          for (let pos = 0; pos < len; ) {
            const take = Math.min(blockLen - this.pos, len - pos);
            if (take === blockLen) {
              for (; blockLen <= len - pos; pos += blockLen)
                this.process(data, pos);
              continue;
            }
            buffer.set(data.subarray(pos, pos + take), this.pos);
            this.pos += take;
            pos += take;
            if (this.pos === blockLen) {
              this.process(buffer, 0, false);
              this.pos = 0;
            }
          }
          return this;
        }
        destroy() {
          clean2(this.h, this.r, this.buffer, this.pad);
        }
        digestInto(out) {
          aexists2(this);
          aoutput2(out, this);
          this.finished = true;
          const { buffer, h } = this;
          let { pos } = this;
          if (pos) {
            buffer[pos++] = 1;
            for (; pos < 16; pos++)
              buffer[pos] = 0;
            this.process(buffer, 0, true);
          }
          this.finalize();
          let opos = 0;
          for (let i = 0; i < 8; i++) {
            out[opos++] = h[i] >>> 0;
            out[opos++] = h[i] >>> 8;
          }
          return out;
        }
        digest() {
          const { buffer, outputLen } = this;
          this.digestInto(buffer);
          const res = buffer.slice(0, outputLen);
          this.destroy();
          return res;
        }
      };
      poly1305 = /* @__PURE__ */ (() => wrapConstructorWithKey((key) => new Poly1305(key)))();
    }
  });

  // node_modules/@noble/ciphers/chacha.js
  var chacha_exports = {};
  __export(chacha_exports, {
    _poly1305_aead: () => _poly1305_aead,
    chacha12: () => chacha12,
    chacha20: () => chacha20,
    chacha20orig: () => chacha20orig,
    chacha20poly1305: () => chacha20poly1305,
    chacha8: () => chacha8,
    hchacha: () => hchacha,
    rngChacha20: () => rngChacha20,
    rngChacha8: () => rngChacha8,
    xchacha20: () => xchacha20,
    xchacha20poly1305: () => xchacha20poly1305
  });
  function chachaCore(s, k, n, out, cnt, rounds = 20) {
    let y00 = s[0], y01 = s[1], y02 = s[2], y03 = s[3], y04 = k[0], y05 = k[1], y06 = k[2], y07 = k[3], y08 = k[4], y09 = k[5], y10 = k[6], y11 = k[7], y12 = cnt, y13 = n[0], y14 = n[1], y15 = n[2];
    let x00 = y00, x01 = y01, x02 = y02, x03 = y03, x04 = y04, x05 = y05, x06 = y06, x07 = y07, x08 = y08, x09 = y09, x10 = y10, x11 = y11, x12 = y12, x13 = y13, x14 = y14, x15 = y15;
    for (let r = 0; r < rounds; r += 2) {
      x00 = x00 + x04 | 0;
      x12 = rotl(x12 ^ x00, 16);
      x08 = x08 + x12 | 0;
      x04 = rotl(x04 ^ x08, 12);
      x00 = x00 + x04 | 0;
      x12 = rotl(x12 ^ x00, 8);
      x08 = x08 + x12 | 0;
      x04 = rotl(x04 ^ x08, 7);
      x01 = x01 + x05 | 0;
      x13 = rotl(x13 ^ x01, 16);
      x09 = x09 + x13 | 0;
      x05 = rotl(x05 ^ x09, 12);
      x01 = x01 + x05 | 0;
      x13 = rotl(x13 ^ x01, 8);
      x09 = x09 + x13 | 0;
      x05 = rotl(x05 ^ x09, 7);
      x02 = x02 + x06 | 0;
      x14 = rotl(x14 ^ x02, 16);
      x10 = x10 + x14 | 0;
      x06 = rotl(x06 ^ x10, 12);
      x02 = x02 + x06 | 0;
      x14 = rotl(x14 ^ x02, 8);
      x10 = x10 + x14 | 0;
      x06 = rotl(x06 ^ x10, 7);
      x03 = x03 + x07 | 0;
      x15 = rotl(x15 ^ x03, 16);
      x11 = x11 + x15 | 0;
      x07 = rotl(x07 ^ x11, 12);
      x03 = x03 + x07 | 0;
      x15 = rotl(x15 ^ x03, 8);
      x11 = x11 + x15 | 0;
      x07 = rotl(x07 ^ x11, 7);
      x00 = x00 + x05 | 0;
      x15 = rotl(x15 ^ x00, 16);
      x10 = x10 + x15 | 0;
      x05 = rotl(x05 ^ x10, 12);
      x00 = x00 + x05 | 0;
      x15 = rotl(x15 ^ x00, 8);
      x10 = x10 + x15 | 0;
      x05 = rotl(x05 ^ x10, 7);
      x01 = x01 + x06 | 0;
      x12 = rotl(x12 ^ x01, 16);
      x11 = x11 + x12 | 0;
      x06 = rotl(x06 ^ x11, 12);
      x01 = x01 + x06 | 0;
      x12 = rotl(x12 ^ x01, 8);
      x11 = x11 + x12 | 0;
      x06 = rotl(x06 ^ x11, 7);
      x02 = x02 + x07 | 0;
      x13 = rotl(x13 ^ x02, 16);
      x08 = x08 + x13 | 0;
      x07 = rotl(x07 ^ x08, 12);
      x02 = x02 + x07 | 0;
      x13 = rotl(x13 ^ x02, 8);
      x08 = x08 + x13 | 0;
      x07 = rotl(x07 ^ x08, 7);
      x03 = x03 + x04 | 0;
      x14 = rotl(x14 ^ x03, 16);
      x09 = x09 + x14 | 0;
      x04 = rotl(x04 ^ x09, 12);
      x03 = x03 + x04 | 0;
      x14 = rotl(x14 ^ x03, 8);
      x09 = x09 + x14 | 0;
      x04 = rotl(x04 ^ x09, 7);
    }
    let oi = 0;
    out[oi++] = y00 + x00 | 0;
    out[oi++] = y01 + x01 | 0;
    out[oi++] = y02 + x02 | 0;
    out[oi++] = y03 + x03 | 0;
    out[oi++] = y04 + x04 | 0;
    out[oi++] = y05 + x05 | 0;
    out[oi++] = y06 + x06 | 0;
    out[oi++] = y07 + x07 | 0;
    out[oi++] = y08 + x08 | 0;
    out[oi++] = y09 + x09 | 0;
    out[oi++] = y10 + x10 | 0;
    out[oi++] = y11 + x11 | 0;
    out[oi++] = y12 + x12 | 0;
    out[oi++] = y13 + x13 | 0;
    out[oi++] = y14 + x14 | 0;
    out[oi++] = y15 + x15 | 0;
  }
  function hchacha(s, k, i, out) {
    let x00 = s[0], x01 = s[1], x02 = s[2], x03 = s[3], x04 = k[0], x05 = k[1], x06 = k[2], x07 = k[3], x08 = k[4], x09 = k[5], x10 = k[6], x11 = k[7], x12 = i[0], x13 = i[1], x14 = i[2], x15 = i[3];
    for (let r = 0; r < 20; r += 2) {
      x00 = x00 + x04 | 0;
      x12 = rotl(x12 ^ x00, 16);
      x08 = x08 + x12 | 0;
      x04 = rotl(x04 ^ x08, 12);
      x00 = x00 + x04 | 0;
      x12 = rotl(x12 ^ x00, 8);
      x08 = x08 + x12 | 0;
      x04 = rotl(x04 ^ x08, 7);
      x01 = x01 + x05 | 0;
      x13 = rotl(x13 ^ x01, 16);
      x09 = x09 + x13 | 0;
      x05 = rotl(x05 ^ x09, 12);
      x01 = x01 + x05 | 0;
      x13 = rotl(x13 ^ x01, 8);
      x09 = x09 + x13 | 0;
      x05 = rotl(x05 ^ x09, 7);
      x02 = x02 + x06 | 0;
      x14 = rotl(x14 ^ x02, 16);
      x10 = x10 + x14 | 0;
      x06 = rotl(x06 ^ x10, 12);
      x02 = x02 + x06 | 0;
      x14 = rotl(x14 ^ x02, 8);
      x10 = x10 + x14 | 0;
      x06 = rotl(x06 ^ x10, 7);
      x03 = x03 + x07 | 0;
      x15 = rotl(x15 ^ x03, 16);
      x11 = x11 + x15 | 0;
      x07 = rotl(x07 ^ x11, 12);
      x03 = x03 + x07 | 0;
      x15 = rotl(x15 ^ x03, 8);
      x11 = x11 + x15 | 0;
      x07 = rotl(x07 ^ x11, 7);
      x00 = x00 + x05 | 0;
      x15 = rotl(x15 ^ x00, 16);
      x10 = x10 + x15 | 0;
      x05 = rotl(x05 ^ x10, 12);
      x00 = x00 + x05 | 0;
      x15 = rotl(x15 ^ x00, 8);
      x10 = x10 + x15 | 0;
      x05 = rotl(x05 ^ x10, 7);
      x01 = x01 + x06 | 0;
      x12 = rotl(x12 ^ x01, 16);
      x11 = x11 + x12 | 0;
      x06 = rotl(x06 ^ x11, 12);
      x01 = x01 + x06 | 0;
      x12 = rotl(x12 ^ x01, 8);
      x11 = x11 + x12 | 0;
      x06 = rotl(x06 ^ x11, 7);
      x02 = x02 + x07 | 0;
      x13 = rotl(x13 ^ x02, 16);
      x08 = x08 + x13 | 0;
      x07 = rotl(x07 ^ x08, 12);
      x02 = x02 + x07 | 0;
      x13 = rotl(x13 ^ x02, 8);
      x08 = x08 + x13 | 0;
      x07 = rotl(x07 ^ x08, 7);
      x03 = x03 + x04 | 0;
      x14 = rotl(x14 ^ x03, 16);
      x09 = x09 + x14 | 0;
      x04 = rotl(x04 ^ x09, 12);
      x03 = x03 + x04 | 0;
      x14 = rotl(x14 ^ x03, 8);
      x09 = x09 + x14 | 0;
      x04 = rotl(x04 ^ x09, 7);
    }
    let oi = 0;
    out[oi++] = x00;
    out[oi++] = x01;
    out[oi++] = x02;
    out[oi++] = x03;
    out[oi++] = x12;
    out[oi++] = x13;
    out[oi++] = x14;
    out[oi++] = x15;
  }
  function computeTag(fn, key, nonce, ciphertext, AAD) {
    if (AAD !== void 0)
      abytes2(AAD, void 0, "AAD");
    const authKey = fn(key, nonce, ZEROS32);
    const lengths = u64Lengths(ciphertext.length, AAD ? AAD.length : 0, true);
    const h = poly1305.create(authKey);
    if (AAD)
      updatePadded(h, AAD);
    updatePadded(h, ciphertext);
    h.update(lengths);
    const res = h.digest();
    clean2(authKey, lengths);
    return res;
  }
  var chacha20orig, chacha20, xchacha20, chacha8, chacha12, ZEROS16, updatePadded, ZEROS32, _poly1305_aead, chacha20poly1305, xchacha20poly1305, rngChacha20, rngChacha8;
  var init_chacha = __esm({
    "node_modules/@noble/ciphers/chacha.js"() {
      init_arx();
      init_poly1305();
      init_utils();
      chacha20orig = /* @__PURE__ */ createCipher(chachaCore, {
        counterRight: false,
        counterLength: 8,
        allowShortKeys: true
      });
      chacha20 = /* @__PURE__ */ createCipher(chachaCore, {
        counterRight: false,
        counterLength: 4,
        allowShortKeys: false
      });
      xchacha20 = /* @__PURE__ */ createCipher(chachaCore, {
        counterRight: false,
        counterLength: 8,
        extendNonceFn: hchacha,
        allowShortKeys: false
      });
      chacha8 = /* @__PURE__ */ createCipher(chachaCore, {
        counterRight: false,
        counterLength: 4,
        rounds: 8
      });
      chacha12 = /* @__PURE__ */ createCipher(chachaCore, {
        counterRight: false,
        counterLength: 4,
        rounds: 12
      });
      ZEROS16 = /* @__PURE__ */ new Uint8Array(16);
      updatePadded = (h, msg) => {
        h.update(msg);
        const leftover = msg.length % 16;
        if (leftover)
          h.update(ZEROS16.subarray(leftover));
      };
      ZEROS32 = /* @__PURE__ */ new Uint8Array(32);
      _poly1305_aead = (xorStream) => (key, nonce, AAD) => {
        const tagLength = 16;
        return {
          encrypt(plaintext, output) {
            const plength = plaintext.length;
            output = getOutput(plength + tagLength, output, false);
            output.set(plaintext);
            const oPlain = output.subarray(0, -tagLength);
            xorStream(key, nonce, oPlain, oPlain, 1);
            const tag = computeTag(xorStream, key, nonce, oPlain, AAD);
            output.set(tag, plength);
            clean2(tag);
            return output;
          },
          decrypt(ciphertext, output) {
            output = getOutput(ciphertext.length - tagLength, output, false);
            const data = ciphertext.subarray(0, -tagLength);
            const passedTag = ciphertext.subarray(-tagLength);
            const tag = computeTag(xorStream, key, nonce, data, AAD);
            if (!equalBytes(passedTag, tag))
              throw new Error("invalid tag");
            output.set(ciphertext.subarray(0, -tagLength));
            xorStream(key, nonce, output, output, 1);
            clean2(tag);
            return output;
          }
        };
      };
      chacha20poly1305 = /* @__PURE__ */ wrapCipher({ blockSize: 64, nonceLength: 12, tagLength: 16 }, _poly1305_aead(chacha20));
      xchacha20poly1305 = /* @__PURE__ */ wrapCipher({ blockSize: 64, nonceLength: 24, tagLength: 16 }, _poly1305_aead(xchacha20));
      rngChacha20 = /* @__PURE__ */ createPRG(chacha20orig, 64, 32, 8);
      rngChacha8 = /* @__PURE__ */ createPRG(chacha8, 64, 32, 12);
    }
  });

  // src/service/activityPub.ts
  var ActivityPubService = class {
    // All class #properties are PRIVATE
    #actorID = "";
    async start(actorID) {
      this.#actorID = actorID;
    }
    createObject(object) {
      return this.sendActivity({
        "@context": "",
        "id": "",
        "type": "Create",
        "actor": this.#actorID,
        "object": object
      });
    }
    deleteObject(objectId) {
      return this.sendActivity({
        "@context": "",
        "id": "",
        "type": "Delete",
        "actor": this.#actorID,
        "object": objectId
      });
    }
    sendActivity(activity) {
      try {
        fetch("/@me/outbox", {
          method: "POST",
          body: JSON.stringify(activity)
        });
        return true;
      } catch (err) {
        console.log(err);
        return false;
      }
    }
  };

  // node_modules/idb/build/index.js
  var instanceOfAny = (object, constructors) => constructors.some((c) => object instanceof c);
  var idbProxyableTypes;
  var cursorAdvanceMethods;
  function getIdbProxyableTypes() {
    return idbProxyableTypes || (idbProxyableTypes = [
      IDBDatabase,
      IDBObjectStore,
      IDBIndex,
      IDBCursor,
      IDBTransaction
    ]);
  }
  function getCursorAdvanceMethods() {
    return cursorAdvanceMethods || (cursorAdvanceMethods = [
      IDBCursor.prototype.advance,
      IDBCursor.prototype.continue,
      IDBCursor.prototype.continuePrimaryKey
    ]);
  }
  var transactionDoneMap = /* @__PURE__ */ new WeakMap();
  var transformCache = /* @__PURE__ */ new WeakMap();
  var reverseTransformCache = /* @__PURE__ */ new WeakMap();
  function promisifyRequest(request) {
    const promise = new Promise((resolve, reject) => {
      const unlisten = () => {
        request.removeEventListener("success", success);
        request.removeEventListener("error", error);
      };
      const success = () => {
        resolve(wrap(request.result));
        unlisten();
      };
      const error = () => {
        reject(request.error);
        unlisten();
      };
      request.addEventListener("success", success);
      request.addEventListener("error", error);
    });
    reverseTransformCache.set(promise, request);
    return promise;
  }
  function cacheDonePromiseForTransaction(tx) {
    if (transactionDoneMap.has(tx))
      return;
    const done = new Promise((resolve, reject) => {
      const unlisten = () => {
        tx.removeEventListener("complete", complete);
        tx.removeEventListener("error", error);
        tx.removeEventListener("abort", error);
      };
      const complete = () => {
        resolve();
        unlisten();
      };
      const error = () => {
        reject(tx.error || new DOMException("AbortError", "AbortError"));
        unlisten();
      };
      tx.addEventListener("complete", complete);
      tx.addEventListener("error", error);
      tx.addEventListener("abort", error);
    });
    transactionDoneMap.set(tx, done);
  }
  var idbProxyTraps = {
    get(target, prop, receiver) {
      if (target instanceof IDBTransaction) {
        if (prop === "done")
          return transactionDoneMap.get(target);
        if (prop === "store") {
          return receiver.objectStoreNames[1] ? void 0 : receiver.objectStore(receiver.objectStoreNames[0]);
        }
      }
      return wrap(target[prop]);
    },
    set(target, prop, value) {
      target[prop] = value;
      return true;
    },
    has(target, prop) {
      if (target instanceof IDBTransaction && (prop === "done" || prop === "store")) {
        return true;
      }
      return prop in target;
    }
  };
  function replaceTraps(callback) {
    idbProxyTraps = callback(idbProxyTraps);
  }
  function wrapFunction(func) {
    if (getCursorAdvanceMethods().includes(func)) {
      return function(...args) {
        func.apply(unwrap(this), args);
        return wrap(this.request);
      };
    }
    return function(...args) {
      return wrap(func.apply(unwrap(this), args));
    };
  }
  function transformCachableValue(value) {
    if (typeof value === "function")
      return wrapFunction(value);
    if (value instanceof IDBTransaction)
      cacheDonePromiseForTransaction(value);
    if (instanceOfAny(value, getIdbProxyableTypes()))
      return new Proxy(value, idbProxyTraps);
    return value;
  }
  function wrap(value) {
    if (value instanceof IDBRequest)
      return promisifyRequest(value);
    if (transformCache.has(value))
      return transformCache.get(value);
    const newValue = transformCachableValue(value);
    if (newValue !== value) {
      transformCache.set(value, newValue);
      reverseTransformCache.set(newValue, value);
    }
    return newValue;
  }
  var unwrap = (value) => reverseTransformCache.get(value);
  function openDB(name, version, { blocked, upgrade, blocking, terminated } = {}) {
    const request = indexedDB.open(name, version);
    const openPromise = wrap(request);
    if (upgrade) {
      request.addEventListener("upgradeneeded", (event) => {
        upgrade(wrap(request.result), event.oldVersion, event.newVersion, wrap(request.transaction), event);
      });
    }
    if (blocked) {
      request.addEventListener("blocked", (event) => blocked(
        // Casting due to https://github.com/microsoft/TypeScript-DOM-lib-generator/pull/1405
        event.oldVersion,
        event.newVersion,
        event
      ));
    }
    openPromise.then((db) => {
      if (terminated)
        db.addEventListener("close", () => terminated());
      if (blocking) {
        db.addEventListener("versionchange", (event) => blocking(event.oldVersion, event.newVersion, event));
      }
    }).catch(() => {
    });
    return openPromise;
  }
  var readMethods = ["get", "getKey", "getAll", "getAllKeys", "count"];
  var writeMethods = ["put", "add", "delete", "clear"];
  var cachedMethods = /* @__PURE__ */ new Map();
  function getMethod(target, prop) {
    if (!(target instanceof IDBDatabase && !(prop in target) && typeof prop === "string")) {
      return;
    }
    if (cachedMethods.get(prop))
      return cachedMethods.get(prop);
    const targetFuncName = prop.replace(/FromIndex$/, "");
    const useIndex = prop !== targetFuncName;
    const isWrite = writeMethods.includes(targetFuncName);
    if (
      // Bail if the target doesn't exist on the target. Eg, getAll isn't in Edge.
      !(targetFuncName in (useIndex ? IDBIndex : IDBObjectStore).prototype) || !(isWrite || readMethods.includes(targetFuncName))
    ) {
      return;
    }
    const method = async function(storeName, ...args) {
      const tx = this.transaction(storeName, isWrite ? "readwrite" : "readonly");
      let target2 = tx.store;
      if (useIndex)
        target2 = target2.index(args.shift());
      return (await Promise.all([
        target2[targetFuncName](...args),
        isWrite && tx.done
      ]))[0];
    };
    cachedMethods.set(prop, method);
    return method;
  }
  replaceTraps((oldTraps) => ({
    ...oldTraps,
    get: (target, prop, receiver) => getMethod(target, prop) || oldTraps.get(target, prop, receiver),
    has: (target, prop) => !!getMethod(target, prop) || oldTraps.has(target, prop)
  }));
  var advanceMethodProps = ["continue", "continuePrimaryKey", "advance"];
  var methodMap = {};
  var advanceResults = /* @__PURE__ */ new WeakMap();
  var ittrProxiedCursorToOriginalProxy = /* @__PURE__ */ new WeakMap();
  var cursorIteratorTraps = {
    get(target, prop) {
      if (!advanceMethodProps.includes(prop))
        return target[prop];
      let cachedFunc = methodMap[prop];
      if (!cachedFunc) {
        cachedFunc = methodMap[prop] = function(...args) {
          advanceResults.set(this, ittrProxiedCursorToOriginalProxy.get(this)[prop](...args));
        };
      }
      return cachedFunc;
    }
  };
  async function* iterate(...args) {
    let cursor = this;
    if (!(cursor instanceof IDBCursor)) {
      cursor = await cursor.openCursor(...args);
    }
    if (!cursor)
      return;
    cursor = cursor;
    const proxiedCursor = new Proxy(cursor, cursorIteratorTraps);
    ittrProxiedCursorToOriginalProxy.set(proxiedCursor, cursor);
    reverseTransformCache.set(proxiedCursor, unwrap(cursor));
    while (cursor) {
      yield proxiedCursor;
      cursor = await (advanceResults.get(proxiedCursor) || cursor.continue());
      advanceResults.delete(proxiedCursor);
    }
  }
  function isIteratorProp(target, prop) {
    return prop === Symbol.asyncIterator && instanceOfAny(target, [IDBIndex, IDBObjectStore, IDBCursor]) || prop === "iterate" && instanceOfAny(target, [IDBIndex, IDBObjectStore]);
  }
  replaceTraps((oldTraps) => ({
    ...oldTraps,
    get(target, prop, receiver) {
      if (isIteratorProp(target, prop))
        return iterate;
      return oldTraps.get(target, prop, receiver);
    },
    has(target, prop) {
      return isIteratorProp(target, prop) || oldTraps.has(target, prop);
    }
  }));

  // node_modules/ts-mls/dist/codec/tlsEncoder.js
  function encode(enc) {
    return (t) => {
      const [len, write] = enc(t);
      const buf = new ArrayBuffer(len);
      write(0, buf);
      return new Uint8Array(buf);
    };
  }
  function contramapBufferEncoder(enc, f) {
    return (u) => enc(f(u));
  }
  function contramapBufferEncoders(encoders, toTuple) {
    return (value) => {
      const values = toTuple(value);
      let totalLength = 0;
      let writeTotal = (_offset, _buffer) => {
      };
      for (let i = 0; i < encoders.length; i++) {
        const [len, write] = encoders[i](values[i]);
        const oldFunc = writeTotal;
        const currentLen = totalLength;
        writeTotal = (offset, buffer) => {
          oldFunc(offset, buffer);
          write(offset + currentLen, buffer);
        };
        totalLength += len;
      }
      return [totalLength, writeTotal];
    };
  }
  function composeBufferEncoders(encoders) {
    return (values) => contramapBufferEncoders(encoders, (t) => t)(values);
  }
  var encVoid = [0, () => {
  }];

  // node_modules/ts-mls/dist/codec/number.js
  var uint8Encoder = (n) => [
    1,
    (offset, buffer) => {
      const view = new DataView(buffer);
      view.setUint8(offset, n);
    }
  ];
  var encodeUint8 = encode(uint8Encoder);
  var decodeUint8 = (b, offset) => {
    const value = b.at(offset);
    return value !== void 0 ? [value, 1] : void 0;
  };
  var uint16Encoder = (n) => [
    2,
    (offset, buffer) => {
      const view = new DataView(buffer);
      view.setUint16(offset, n);
    }
  ];
  var encodeUint16 = encode(uint16Encoder);
  var decodeUint16 = (b, offset) => {
    const view = new DataView(b.buffer, b.byteOffset, b.byteLength);
    try {
      return [view.getUint16(offset), 2];
    } catch (e) {
      return void 0;
    }
  };
  var uint32Encoder = (n) => [
    4,
    (offset, buffer) => {
      const view = new DataView(buffer);
      view.setUint32(offset, n);
    }
  ];
  var encodeUint32 = encode(uint32Encoder);
  var decodeUint32 = (b, offset) => {
    const view = new DataView(b.buffer, b.byteOffset, b.byteLength);
    try {
      return [view.getUint32(offset), 4];
    } catch (e) {
      return void 0;
    }
  };
  var uint64Encoder = (n) => [
    8,
    (offset, buffer) => {
      const view = new DataView(buffer);
      view.setBigUint64(offset, n);
    }
  ];
  var encodeUint64 = encode(uint64Encoder);
  var decodeUint64 = (b, offset) => {
    const view = new DataView(b.buffer, b.byteOffset, b.byteLength);
    try {
      return [view.getBigUint64(offset), 8];
    } catch (e) {
      return void 0;
    }
  };

  // node_modules/ts-mls/dist/codec/tlsDecoder.js
  function mapDecoder(dec, f) {
    return (b, offset) => {
      const x = dec(b, offset);
      if (x !== void 0) {
        const [t, l] = x;
        return [f(t), l];
      }
    };
  }
  function mapDecodersOption(decoders, f) {
    return (b, offset) => {
      const initial = mapDecoders(decoders, f)(b, offset);
      if (initial === void 0)
        return void 0;
      else {
        const [r, len] = initial;
        return r !== void 0 ? [r, len] : void 0;
      }
    };
  }
  function mapDecoders(decoders, f) {
    return (b, offset) => {
      const result = decoders.reduce((acc, decoder) => {
        if (!acc)
          return void 0;
        const decoded = decoder(b, acc.offset);
        if (!decoded)
          return void 0;
        const [value, length] = decoded;
        return {
          values: [...acc.values, value],
          offset: acc.offset + length,
          totalLength: acc.totalLength + length
        };
      }, { values: [], offset, totalLength: 0 });
      if (!result)
        return;
      return [f(...result.values), result.totalLength];
    };
  }
  function mapDecoderOption(dec, f) {
    return (b, offset) => {
      const x = dec(b, offset);
      if (x !== void 0) {
        const [t, l] = x;
        const u = f(t);
        return u !== void 0 ? [u, l] : void 0;
      }
    };
  }
  function flatMapDecoder(dec, f) {
    return flatMapDecoderAndMap(dec, f, (_t, u) => u);
  }
  function orDecoder(decT, decU) {
    return (b, offset) => {
      const t = decT(b, offset);
      return t ? t : decU(b, offset);
    };
  }
  function flatMapDecoderAndMap(dec, f, g) {
    return (b, offset) => {
      const decodedT = dec(b, offset);
      if (decodedT !== void 0) {
        const [t, len] = decodedT;
        const decoderU = f(t);
        const decodedU = decoderU(b, offset + len);
        if (decodedU !== void 0) {
          const [u, len2] = decodedU;
          return [g(t, u), len + len2];
        }
      }
    };
  }
  function succeedDecoder(t) {
    return () => [t, 0];
  }

  // node_modules/ts-mls/dist/util/enumHelpers.js
  function enumNumberToKey(t) {
    return (n) => Object.values(t).includes(n) ? reverseMap(t)[n] : void 0;
  }
  function reverseMap(obj) {
    return Object.entries(obj).reduce((acc, [key, value]) => ({
      ...acc,
      [value]: key
    }), {});
  }
  function openEnumNumberToKey(rec) {
    return (n) => {
      const decoded = enumNumberToKey(rec)(n);
      if (decoded === void 0)
        return n.toString();
      else
        return decoded;
    };
  }
  function openEnumNumberEncoder(rec) {
    return (s) => {
      const x = rec[s];
      if (x === void 0)
        return Number(s);
      else
        return x;
    };
  }

  // node_modules/ts-mls/dist/defaultProposalType.js
  var defaultProposalTypes = {
    add: 1,
    update: 2,
    remove: 3,
    psk: 4,
    reinit: 5,
    external_init: 6,
    group_context_extensions: 7
  };
  var defaultProposalTypeEncoder = contramapBufferEncoder(uint16Encoder, (n) => defaultProposalTypes[n]);
  var encodeDefaultProposalType = encode(defaultProposalTypeEncoder);
  var decodeDefaultProposalType = mapDecoderOption(decodeUint16, enumNumberToKey(defaultProposalTypes));

  // node_modules/ts-mls/dist/defaultExtensionType.js
  var defaultExtensionTypes = {
    application_id: 1,
    ratchet_tree: 2,
    required_capabilities: 3,
    external_pub: 4,
    external_senders: 5
  };
  var defaultExtensionTypeEncoder = contramapBufferEncoder(uint16Encoder, (n) => defaultExtensionTypes[n]);
  var encodeDefaultExtensionType = encode(defaultExtensionTypeEncoder);
  var decodeDefaultExtensionType = mapDecoderOption(decodeUint16, enumNumberToKey(defaultExtensionTypes));

  // node_modules/ts-mls/dist/mlsError.js
  var MlsError = class extends Error {
    constructor(message) {
      super(message);
      this.name = "MlsError";
    }
  };
  var CodecError = class extends MlsError {
    constructor(message) {
      super(message);
      this.name = "CodecError";
    }
  };
  var DependencyError = class extends MlsError {
    constructor(message) {
      super(message);
      this.name = "DependencyError";
    }
  };
  var CryptoError = class extends MlsError {
    constructor(message) {
      super(message);
      this.name = "CryptoError";
    }
  };
  var InternalError = class extends MlsError {
    constructor(message) {
      super(`This error should never occur, if you see this please submit a bug report. Message: ${message}`);
      this.name = "InternalError";
    }
  };

  // node_modules/ts-mls/dist/codec/variableLength.js
  var varLenDataEncoder = (data) => {
    const [len, write] = lengthEncoder(data.length);
    return [
      len + data.length,
      (offset, buffer) => {
        write(offset, buffer);
        const view = new Uint8Array(buffer);
        view.set(data, offset + len);
      }
    ];
  };
  function lengthEncoder(len) {
    if (len < 64) {
      return [
        1,
        (offset, buffer) => {
          const view = new DataView(buffer);
          view.setUint8(offset, len & 63);
        }
      ];
    } else if (len < 16384) {
      return [
        2,
        (offset, buffer) => {
          const view = new DataView(buffer);
          view.setUint8(offset, len >> 8 & 63 | 64);
          view.setUint8(offset + 1, len & 255);
        }
      ];
    } else if (len < 1073741824) {
      return [
        4,
        (offset, buffer) => {
          const view = new DataView(buffer);
          view.setUint8(offset, len >> 24 & 63 | 128);
          view.setUint8(offset + 1, len >> 16 & 255);
          view.setUint8(offset + 2, len >> 8 & 255);
          view.setUint8(offset + 3, len & 255);
        }
      ];
    } else {
      throw new CodecError("Length too large to encode (max is 2^30 - 1)");
    }
  }
  function determineLength(data, offset = 0) {
    if (offset >= data.length) {
      throw new CodecError("Offset beyond buffer");
    }
    const firstByte = data[offset];
    const prefix = firstByte >> 6;
    if (prefix === 0) {
      return { length: firstByte & 63, lengthFieldSize: 1 };
    } else if (prefix === 1) {
      if (offset + 2 > data.length)
        throw new CodecError("Incomplete 2-byte length");
      return { length: (firstByte & 63) << 8 | data[offset + 1], lengthFieldSize: 2 };
    } else if (prefix === 2) {
      if (offset + 4 > data.length)
        throw new CodecError("Incomplete 4-byte length");
      return {
        length: (firstByte & 63) << 24 | data[offset + 1] << 16 | data[offset + 2] << 8 | data[offset + 3],
        lengthFieldSize: 4
      };
    } else {
      throw new CodecError("8-byte length not supported in this implementation");
    }
  }
  var decodeVarLenData = (buf, offset) => {
    if (offset >= buf.length) {
      throw new CodecError("Offset beyond buffer");
    }
    const { length, lengthFieldSize } = determineLength(buf, offset);
    const totalBytes = lengthFieldSize + length;
    if (offset + totalBytes > buf.length) {
      throw new CodecError("Data length exceeds buffer");
    }
    const data = buf.subarray(offset + lengthFieldSize, offset + totalBytes);
    return [data, totalBytes];
  };
  function varLenTypeEncoder(enc) {
    return (data) => {
      let totalLength = 0;
      let writeTotal = (_offset, _buffer) => {
      };
      for (let i = 0; i < data.length; i++) {
        const [len, write] = enc(data[i]);
        const oldFunc = writeTotal;
        const currentLen = totalLength;
        writeTotal = (offset, buffer) => {
          oldFunc(offset, buffer);
          write(offset + currentLen, buffer);
        };
        totalLength += len;
      }
      const [headerLength, writeLength] = lengthEncoder(totalLength);
      return [
        headerLength + totalLength,
        (offset, buffer) => {
          writeLength(offset, buffer);
          writeTotal(offset + headerLength, buffer);
        }
      ];
    };
  }
  function decodeVarLenType(dec) {
    return (b, offset) => {
      const d = decodeVarLenData(b, offset);
      if (d === void 0)
        return;
      const [totalBytes, totalLength] = d;
      let cursor = 0;
      const result = [];
      while (cursor < totalBytes.length) {
        const item = dec(totalBytes, cursor);
        if (item === void 0)
          return void 0;
        const [value, len] = item;
        result.push(value);
        cursor += len;
      }
      return [result, totalLength];
    };
  }

  // node_modules/ts-mls/dist/extension.js
  var extensionTypeEncoder = (t) => typeof t === "number" ? uint16Encoder(t) : defaultExtensionTypeEncoder(t);
  var encodeExtensionType = encode(extensionTypeEncoder);
  var decodeExtensionType = orDecoder(decodeDefaultExtensionType, decodeUint16);
  var extensionEncoder = contramapBufferEncoders([extensionTypeEncoder, varLenDataEncoder], (e) => [e.extensionType, e.extensionData]);
  var encodeExtension = encode(extensionEncoder);
  var decodeExtension = mapDecoders([decodeExtensionType, decodeVarLenData], (extensionType, extensionData) => ({ extensionType, extensionData }));

  // node_modules/ts-mls/dist/credentialType.js
  var credentialTypes = {
    basic: 1,
    x509: 2
  };
  var credentialTypeEncoder = contramapBufferEncoder(uint16Encoder, openEnumNumberEncoder(credentialTypes));
  var encodeCredentialType = encode(credentialTypeEncoder);
  var decodeCredentialType = mapDecoderOption(decodeUint16, openEnumNumberToKey(credentialTypes));

  // node_modules/ts-mls/dist/credential.js
  var credentialBasicEncoder = contramapBufferEncoders([credentialTypeEncoder, varLenDataEncoder], (c) => [c.credentialType, c.identity]);
  var encodeCredentialBasic = encode(credentialBasicEncoder);
  var credentialX509Encoder = contramapBufferEncoders([credentialTypeEncoder, varLenTypeEncoder(varLenDataEncoder)], (c) => [c.credentialType, c.certificates]);
  var encodeCredentialX509 = encode(credentialX509Encoder);
  var credentialCustomEncoder = contramapBufferEncoders([credentialTypeEncoder, varLenDataEncoder], (c) => [c.credentialType, c.data]);
  var encodeCredentialCustom = encode(credentialCustomEncoder);
  var credentialEncoder = (c) => {
    switch (c.credentialType) {
      case "basic":
        return credentialBasicEncoder(c);
      case "x509":
        return credentialX509Encoder(c);
      default:
        return credentialCustomEncoder(c);
    }
  };
  var encodeCredential = encode(credentialEncoder);
  var decodeCredentialBasic = mapDecoder(decodeVarLenData, (identity) => ({
    credentialType: "basic",
    identity
  }));
  var decodeCredentialX509 = mapDecoder(decodeVarLenType(decodeVarLenData), (certificates) => ({ credentialType: "x509", certificates }));
  var decodeCredential = flatMapDecoder(decodeCredentialType, (credentialType) => {
    switch (credentialType) {
      case "basic":
        return decodeCredentialBasic;
      case "x509":
        return decodeCredentialX509;
    }
  });

  // node_modules/ts-mls/dist/externalSender.js
  var externalSenderEncoder = contramapBufferEncoders([varLenDataEncoder, credentialEncoder], (e) => [e.signaturePublicKey, e.credential]);
  var encodeExternalSender = encode(externalSenderEncoder);
  var decodeExternalSender = mapDecoders([decodeVarLenData, decodeCredential], (signaturePublicKey, credential) => ({ signaturePublicKey, credential }));

  // node_modules/ts-mls/dist/codec/optional.js
  function optionalEncoder(encodeT) {
    return (t) => {
      if (t) {
        const [len, write] = encodeT(t);
        return [
          len + 1,
          (offset, buffer) => {
            const view = new DataView(buffer);
            view.setUint8(offset, 1);
            write(offset + 1, buffer);
          }
        ];
      } else {
        return [
          1,
          (offset, buffer) => {
            const view = new DataView(buffer);
            view.setUint8(offset, 0);
          }
        ];
      }
    };
  }
  function decodeOptional(decodeT) {
    return (b, offset) => {
      const presenceOctet = decodeUint8(b, offset)?.[0];
      if (presenceOctet == 1) {
        const result = decodeT(b, offset + 1);
        return result === void 0 ? void 0 : [result[0], result[1] + 1];
      } else {
        return [void 0, 1];
      }
    };
  }

  // node_modules/ts-mls/dist/crypto/ciphersuite.js
  var ciphersuites = {
    MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519: 1,
    MLS_128_DHKEMP256_AES128GCM_SHA256_P256: 2,
    MLS_128_DHKEMX25519_CHACHA20POLY1305_SHA256_Ed25519: 3,
    MLS_256_DHKEMX448_AES256GCM_SHA512_Ed448: 4,
    MLS_256_DHKEMP521_AES256GCM_SHA512_P521: 5,
    MLS_256_DHKEMX448_CHACHA20POLY1305_SHA512_Ed448: 6,
    MLS_256_DHKEMP384_AES256GCM_SHA384_P384: 7,
    MLS_128_MLKEM512_AES128GCM_SHA256_Ed25519: 77,
    MLS_128_MLKEM512_CHACHA20POLY1305_SHA256_Ed25519: 78,
    MLS_256_MLKEM768_AES256GCM_SHA384_Ed25519: 79,
    MLS_256_MLKEM768_CHACHA20POLY1305_SHA384_Ed25519: 80,
    MLS_256_MLKEM1024_AES256GCM_SHA512_Ed25519: 81,
    MLS_256_MLKEM1024_CHACHA20POLY1305_SHA512_Ed25519: 82,
    MLS_256_XWING_AES256GCM_SHA512_Ed25519: 83,
    MLS_256_XWING_CHACHA20POLY1305_SHA512_Ed25519: 84,
    MLS_256_MLKEM1024_AES256GCM_SHA512_MLDSA87: 85,
    MLS_256_MLKEM1024_CHACHA20POLY1305_SHA512_MLDSA87: 86,
    MLS_256_XWING_AES256GCM_SHA512_MLDSA87: 87,
    MLS_256_XWING_CHACHA20POLY1305_SHA512_MLDSA87: 88
  };
  var ciphersuiteEncoder = contramapBufferEncoder(uint16Encoder, openEnumNumberEncoder(ciphersuites));
  var encodeCiphersuite = encode(ciphersuiteEncoder);
  var decodeCiphersuite = mapDecoderOption(decodeUint16, openEnumNumberToKey(ciphersuites));
  function getCiphersuiteFromName(name) {
    return ciphersuiteValues[ciphersuites[name]];
  }
  var ciphersuiteValues = {
    1: {
      hash: "SHA-256",
      hpke: {
        kem: "DHKEM-X25519-HKDF-SHA256",
        aead: "AES128GCM",
        kdf: "HKDF-SHA256"
      },
      signature: "Ed25519",
      name: "MLS_128_DHKEMX25519_AES128GCM_SHA256_Ed25519"
    },
    2: {
      hash: "SHA-256",
      hpke: {
        kem: "DHKEM-P256-HKDF-SHA256",
        aead: "AES128GCM",
        kdf: "HKDF-SHA256"
      },
      signature: "P256",
      name: "MLS_128_DHKEMP256_AES128GCM_SHA256_P256"
    },
    3: {
      hash: "SHA-256",
      hpke: {
        kem: "DHKEM-X25519-HKDF-SHA256",
        aead: "CHACHA20POLY1305",
        kdf: "HKDF-SHA256"
      },
      signature: "Ed25519",
      name: "MLS_128_DHKEMX25519_CHACHA20POLY1305_SHA256_Ed25519"
    },
    4: {
      hash: "SHA-512",
      hpke: {
        kem: "DHKEM-X448-HKDF-SHA512",
        aead: "AES256GCM",
        kdf: "HKDF-SHA512"
      },
      signature: "Ed448",
      name: "MLS_256_DHKEMX448_AES256GCM_SHA512_Ed448"
    },
    5: {
      hash: "SHA-512",
      hpke: {
        kem: "DHKEM-P521-HKDF-SHA512",
        aead: "AES256GCM",
        kdf: "HKDF-SHA512"
      },
      signature: "P521",
      name: "MLS_256_DHKEMP521_AES256GCM_SHA512_P521"
    },
    6: {
      hash: "SHA-512",
      hpke: {
        kem: "DHKEM-X448-HKDF-SHA512",
        aead: "CHACHA20POLY1305",
        kdf: "HKDF-SHA512"
      },
      signature: "Ed448",
      name: "MLS_256_DHKEMX448_CHACHA20POLY1305_SHA512_Ed448"
    },
    7: {
      hash: "SHA-384",
      hpke: {
        kem: "DHKEM-P384-HKDF-SHA384",
        aead: "AES256GCM",
        kdf: "HKDF-SHA384"
      },
      signature: "P384",
      name: "MLS_256_DHKEMP384_AES256GCM_SHA384_P384"
    },
    77: {
      hash: "SHA-256",
      hpke: {
        kem: "ML-KEM-512",
        aead: "AES256GCM",
        kdf: "HKDF-SHA512"
      },
      signature: "Ed25519",
      name: "MLS_128_MLKEM512_AES128GCM_SHA256_Ed25519"
    },
    78: {
      hash: "SHA-256",
      hpke: {
        kem: "ML-KEM-512",
        aead: "CHACHA20POLY1305",
        kdf: "HKDF-SHA512"
      },
      signature: "Ed25519",
      name: "MLS_128_MLKEM512_CHACHA20POLY1305_SHA256_Ed25519"
    },
    79: {
      hash: "SHA-384",
      hpke: {
        kem: "ML-KEM-768",
        aead: "AES256GCM",
        kdf: "HKDF-SHA512"
      },
      signature: "Ed25519",
      name: "MLS_256_MLKEM768_AES256GCM_SHA384_Ed25519"
    },
    80: {
      hash: "SHA-384",
      hpke: {
        kem: "ML-KEM-768",
        aead: "CHACHA20POLY1305",
        kdf: "HKDF-SHA512"
      },
      signature: "Ed25519",
      name: "MLS_256_MLKEM768_CHACHA20POLY1305_SHA384_Ed25519"
    },
    81: {
      hash: "SHA-512",
      hpke: {
        kem: "ML-KEM-1024",
        aead: "AES256GCM",
        kdf: "HKDF-SHA512"
      },
      signature: "Ed25519",
      name: "MLS_256_MLKEM1024_AES256GCM_SHA512_Ed25519"
    },
    82: {
      hash: "SHA-512",
      hpke: {
        kem: "ML-KEM-1024",
        aead: "CHACHA20POLY1305",
        kdf: "HKDF-SHA512"
      },
      signature: "Ed25519",
      name: "MLS_256_MLKEM1024_CHACHA20POLY1305_SHA512_Ed25519"
    },
    83: {
      hash: "SHA-512",
      hpke: {
        kem: "X-Wing",
        aead: "AES256GCM",
        kdf: "HKDF-SHA512"
      },
      signature: "Ed25519",
      name: "MLS_256_XWING_AES256GCM_SHA512_Ed25519"
    },
    84: {
      hash: "SHA-512",
      hpke: {
        kem: "X-Wing",
        aead: "CHACHA20POLY1305",
        kdf: "HKDF-SHA512"
      },
      signature: "Ed25519",
      name: "MLS_256_XWING_CHACHA20POLY1305_SHA512_Ed25519"
    },
    85: {
      hash: "SHA-512",
      hpke: {
        kem: "ML-KEM-1024",
        aead: "AES256GCM",
        kdf: "HKDF-SHA512"
      },
      signature: "ML-DSA-87",
      name: "MLS_256_MLKEM1024_AES256GCM_SHA512_MLDSA87"
    },
    86: {
      hash: "SHA-512",
      hpke: {
        kem: "ML-KEM-1024",
        aead: "CHACHA20POLY1305",
        kdf: "HKDF-SHA512"
      },
      signature: "ML-DSA-87",
      name: "MLS_256_MLKEM1024_CHACHA20POLY1305_SHA512_MLDSA87"
    },
    87: {
      hash: "SHA-512",
      hpke: {
        kem: "X-Wing",
        aead: "AES256GCM",
        kdf: "HKDF-SHA512"
      },
      signature: "ML-DSA-87",
      name: "MLS_256_XWING_AES256GCM_SHA512_MLDSA87"
    },
    88: {
      hash: "SHA-512",
      hpke: {
        kem: "X-Wing",
        aead: "CHACHA20POLY1305",
        kdf: "HKDF-SHA512"
      },
      signature: "ML-DSA-87",
      name: "MLS_256_XWING_CHACHA20POLY1305_SHA512_MLDSA87"
    }
  };

  // node_modules/ts-mls/dist/crypto/signature.js
  async function signWithLabel(signKey, label, content, s) {
    return s.sign(signKey, encode(composeBufferEncoders([varLenDataEncoder, varLenDataEncoder]))([
      new TextEncoder().encode(`MLS 1.0 ${label}`),
      content
    ]));
  }

  // node_modules/ts-mls/dist/protocolVersion.js
  var protocolVersions = {
    mls10: 1
  };
  var protocolVersionEncoder = contramapBufferEncoder(uint16Encoder, (t) => protocolVersions[t]);
  var encodeProtocolVersion = encode(protocolVersionEncoder);
  var decodeProtocolVersion = mapDecoderOption(decodeUint16, enumNumberToKey(protocolVersions));

  // node_modules/ts-mls/dist/capabilities.js
  var capabilitiesEncoder = contramapBufferEncoders([
    varLenTypeEncoder(protocolVersionEncoder),
    varLenTypeEncoder(ciphersuiteEncoder),
    varLenTypeEncoder(uint16Encoder),
    varLenTypeEncoder(uint16Encoder),
    varLenTypeEncoder(credentialTypeEncoder)
  ], (cap) => [cap.versions, cap.ciphersuites, cap.extensions, cap.proposals, cap.credentials]);
  var encodeCapabilities = encode(capabilitiesEncoder);
  var decodeCapabilities = mapDecoders([
    decodeVarLenType(decodeProtocolVersion),
    decodeVarLenType(decodeCiphersuite),
    decodeVarLenType(decodeUint16),
    decodeVarLenType(decodeUint16),
    decodeVarLenType(decodeCredentialType)
  ], (versions, ciphersuites2, extensions, proposals, credentials) => ({
    versions,
    ciphersuites: ciphersuites2,
    extensions,
    proposals,
    credentials
  }));

  // node_modules/ts-mls/dist/leafNodeSource.js
  var leafNodeSources = {
    key_package: 1,
    update: 2,
    commit: 3
  };
  var leafNodeSourceEncoder = contramapBufferEncoder(uint8Encoder, (t) => leafNodeSources[t]);
  var encodeLeafNodeSource = encode(leafNodeSourceEncoder);
  var decodeLeafNodeSource = mapDecoderOption(decodeUint8, enumNumberToKey(leafNodeSources));

  // node_modules/ts-mls/dist/lifetime.js
  var lifetimeEncoder = contramapBufferEncoders([uint64Encoder, uint64Encoder], (lt) => [lt.notBefore, lt.notAfter]);
  var encodeLifetime = encode(lifetimeEncoder);
  var decodeLifetime = mapDecoders([decodeUint64, decodeUint64], (notBefore, notAfter) => ({
    notBefore,
    notAfter
  }));
  var defaultLifetime = {
    notBefore: 0n,
    notAfter: 9223372036854775807n
  };

  // node_modules/ts-mls/dist/leafNode.js
  var leafNodeDataEncoder = contramapBufferEncoders([varLenDataEncoder, varLenDataEncoder, credentialEncoder, capabilitiesEncoder], (data) => [data.hpkePublicKey, data.signaturePublicKey, data.credential, data.capabilities]);
  var encodeLeafNodeData = encode(leafNodeDataEncoder);
  var decodeLeafNodeData = mapDecoders([decodeVarLenData, decodeVarLenData, decodeCredential, decodeCapabilities], (hpkePublicKey, signaturePublicKey, credential, capabilities) => ({
    hpkePublicKey,
    signaturePublicKey,
    credential,
    capabilities
  }));
  var leafNodeInfoLifetimeEncoder = contramapBufferEncoders([leafNodeSourceEncoder, lifetimeEncoder], (info) => ["key_package", info.lifetime]);
  var encodeLeafNodeInfoLifetime = encode(leafNodeInfoLifetimeEncoder);
  var leafNodeInfoUpdateEncoder = contramapBufferEncoder(leafNodeSourceEncoder, (i) => i.leafNodeSource);
  var encodeLeafNodeInfoUpdate = encode(leafNodeInfoUpdateEncoder);
  var leafNodeInfoCommitEncoder = contramapBufferEncoders([leafNodeSourceEncoder, varLenDataEncoder], (info) => ["commit", info.parentHash]);
  var encodeLeafNodeInfoCommit = encode(leafNodeInfoCommitEncoder);
  var leafNodeInfoEncoder = (info) => {
    switch (info.leafNodeSource) {
      case "key_package":
        return leafNodeInfoLifetimeEncoder(info);
      case "update":
        return leafNodeInfoUpdateEncoder(info);
      case "commit":
        return leafNodeInfoCommitEncoder(info);
    }
  };
  var encodeLeafNodeInfo = encode(leafNodeInfoEncoder);
  var decodeLeafNodeInfoLifetime = mapDecoder(decodeLifetime, (lifetime) => ({
    leafNodeSource: "key_package",
    lifetime
  }));
  var decodeLeafNodeInfoCommit = mapDecoders([decodeVarLenData], (parentHash) => ({
    leafNodeSource: "commit",
    parentHash
  }));
  var decodeLeafNodeInfo = flatMapDecoder(decodeLeafNodeSource, (leafNodeSource) => {
    switch (leafNodeSource) {
      case "key_package":
        return decodeLeafNodeInfoLifetime;
      case "update":
        return succeedDecoder({ leafNodeSource });
      case "commit":
        return decodeLeafNodeInfoCommit;
    }
  });
  var leafNodeExtensionsEncoder = contramapBufferEncoder(varLenTypeEncoder(extensionEncoder), (ext) => ext.extensions);
  var encodeLeafNodeExtensions = encode(leafNodeExtensionsEncoder);
  var decodeLeafNodeExtensions = mapDecoder(decodeVarLenType(decodeExtension), (extensions) => ({ extensions }));
  var groupIdLeafIndexEncoder = contramapBufferEncoders([varLenDataEncoder, uint32Encoder], (g) => [g.groupId, g.leafIndex]);
  var encodeGroupIdLeafIndex = encode(groupIdLeafIndexEncoder);
  var leafNodeGroupInfoEncoder = (info) => {
    switch (info.leafNodeSource) {
      case "key_package":
        return encVoid;
      case "update":
      case "commit":
        return groupIdLeafIndexEncoder(info);
    }
  };
  var encodeLeafNodeGroupInfo = encode(leafNodeGroupInfoEncoder);
  var leafNodeTBSEncoder = contramapBufferEncoders([leafNodeDataEncoder, leafNodeInfoEncoder, leafNodeExtensionsEncoder, leafNodeGroupInfoEncoder], (tbs) => [tbs, tbs, tbs, tbs.info]);
  var encodeLeafNodeTBS = encode(leafNodeTBSEncoder);
  var leafNodeEncoder = contramapBufferEncoders([leafNodeDataEncoder, leafNodeInfoEncoder, leafNodeExtensionsEncoder, varLenDataEncoder], (leafNode) => [leafNode, leafNode, leafNode, leafNode.signature]);
  var encodeLeafNode = encode(leafNodeEncoder);
  var decodeLeafNode = mapDecoders([decodeLeafNodeData, decodeLeafNodeInfo, decodeLeafNodeExtensions, decodeVarLenData], (data, info, extensions, signature) => ({
    ...data,
    ...info,
    ...extensions,
    signature
  }));
  var decodeLeafNodeKeyPackage = mapDecoderOption(decodeLeafNode, (ln) => ln.leafNodeSource === "key_package" ? ln : void 0);
  var decodeLeafNodeCommit = mapDecoderOption(decodeLeafNode, (ln) => ln.leafNodeSource === "commit" ? ln : void 0);
  var decodeLeafNodeUpdate = mapDecoderOption(decodeLeafNode, (ln) => ln.leafNodeSource === "update" ? ln : void 0);
  async function signLeafNodeKeyPackage(tbs, signaturePrivateKey, sig) {
    return {
      ...tbs,
      signature: await signWithLabel(signaturePrivateKey, "LeafNodeTBS", encode(leafNodeTBSEncoder)(tbs), sig)
    };
  }

  // node_modules/ts-mls/dist/keyPackage.js
  var keyPackageTBSEncoder = contramapBufferEncoders([protocolVersionEncoder, ciphersuiteEncoder, varLenDataEncoder, leafNodeEncoder, varLenTypeEncoder(extensionEncoder)], (keyPackageTBS) => [
    keyPackageTBS.version,
    keyPackageTBS.cipherSuite,
    keyPackageTBS.initKey,
    keyPackageTBS.leafNode,
    keyPackageTBS.extensions
  ]);
  var encodeKeyPackageTBS = encode(keyPackageTBSEncoder);
  var decodeKeyPackageTBS = mapDecoders([
    decodeProtocolVersion,
    decodeCiphersuite,
    decodeVarLenData,
    decodeLeafNodeKeyPackage,
    decodeVarLenType(decodeExtension)
  ], (version, cipherSuite, initKey, leafNode, extensions) => ({
    version,
    cipherSuite,
    initKey,
    leafNode,
    extensions
  }));
  var keyPackageEncoder = contramapBufferEncoders([keyPackageTBSEncoder, varLenDataEncoder], (keyPackage) => [keyPackage, keyPackage.signature]);
  var encodeKeyPackage = encode(keyPackageEncoder);
  var decodeKeyPackage = mapDecoders([decodeKeyPackageTBS, decodeVarLenData], (keyPackageTBS, signature) => ({
    ...keyPackageTBS,
    signature
  }));
  async function signKeyPackage(tbs, signKey, s) {
    return { ...tbs, signature: await signWithLabel(signKey, "KeyPackageTBS", encode(keyPackageTBSEncoder)(tbs), s) };
  }
  async function generateKeyPackageWithKey(credential, capabilities, lifetime, extensions, signatrueKeyPair, cs, leafNodeExtensions) {
    const initKeys = await cs.hpke.generateKeyPair();
    const hpkeKeys = await cs.hpke.generateKeyPair();
    const privatePackage = {
      initPrivateKey: await cs.hpke.exportPrivateKey(initKeys.privateKey),
      hpkePrivateKey: await cs.hpke.exportPrivateKey(hpkeKeys.privateKey),
      signaturePrivateKey: signatrueKeyPair.signKey
    };
    const leafNodeTbs = {
      leafNodeSource: "key_package",
      hpkePublicKey: await cs.hpke.exportPublicKey(hpkeKeys.publicKey),
      signaturePublicKey: signatrueKeyPair.publicKey,
      info: { leafNodeSource: "key_package" },
      extensions: leafNodeExtensions ?? [],
      credential,
      capabilities,
      lifetime
    };
    const tbs = {
      version: "mls10",
      cipherSuite: cs.name,
      initKey: await cs.hpke.exportPublicKey(initKeys.publicKey),
      leafNode: await signLeafNodeKeyPackage(leafNodeTbs, signatrueKeyPair.signKey, cs.signature),
      extensions
    };
    return { publicPackage: await signKeyPackage(tbs, signatrueKeyPair.signKey, cs.signature), privatePackage };
  }
  async function generateKeyPackage(credential, capabilities, lifetime, extensions, cs, leafNodeExtensions) {
    const sigKeys = await cs.signature.keygen();
    return generateKeyPackageWithKey(credential, capabilities, lifetime, extensions, sigKeys, cs, leafNodeExtensions);
  }

  // node_modules/ts-mls/dist/presharedkey.js
  var pskTypes = {
    external: 1,
    resumption: 2
  };
  var pskTypeEncoder = contramapBufferEncoder(uint8Encoder, (t) => pskTypes[t]);
  var encodePskType = encode(pskTypeEncoder);
  var decodePskType = mapDecoderOption(decodeUint8, enumNumberToKey(pskTypes));
  var resumptionPSKUsages = {
    application: 1,
    reinit: 2,
    branch: 3
  };
  var resumptionPSKUsageEncoder = contramapBufferEncoder(uint8Encoder, (u) => resumptionPSKUsages[u]);
  var encodeResumptionPSKUsage = encode(resumptionPSKUsageEncoder);
  var decodeResumptionPSKUsage = mapDecoderOption(decodeUint8, enumNumberToKey(resumptionPSKUsages));
  var encodePskInfoExternal = contramapBufferEncoders([pskTypeEncoder, varLenDataEncoder], (i) => [i.psktype, i.pskId]);
  var encodePskInfoResumption = contramapBufferEncoders([pskTypeEncoder, resumptionPSKUsageEncoder, varLenDataEncoder, uint64Encoder], (info) => [info.psktype, info.usage, info.pskGroupId, info.pskEpoch]);
  var decodePskInfoResumption = mapDecoders([decodeResumptionPSKUsage, decodeVarLenData, decodeUint64], (usage, pskGroupId, pskEpoch) => {
    return { usage, pskGroupId, pskEpoch };
  });
  var pskInfoEncoder = (info) => {
    switch (info.psktype) {
      case "external":
        return encodePskInfoExternal(info);
      case "resumption":
        return encodePskInfoResumption(info);
    }
  };
  var encodePskInfo = encode(pskInfoEncoder);
  var decodePskInfo = flatMapDecoder(decodePskType, (psktype) => {
    switch (psktype) {
      case "external":
        return mapDecoder(decodeVarLenData, (pskId) => ({
          psktype,
          pskId
        }));
      case "resumption":
        return mapDecoder(decodePskInfoResumption, (resumption) => ({
          psktype,
          ...resumption
        }));
    }
  });
  var pskIdEncoder = contramapBufferEncoders([pskInfoEncoder, varLenDataEncoder], (pskid) => [pskid, pskid.pskNonce]);
  var encodePskId = encode(pskIdEncoder);
  var decodePskId = mapDecoders([decodePskInfo, decodeVarLenData], (info, pskNonce) => ({ ...info, pskNonce }));
  var pskLabelEncoder = contramapBufferEncoders([pskIdEncoder, uint16Encoder, uint16Encoder], (label) => [label.id, label.index, label.count]);
  var encodePskLabel = encode(pskLabelEncoder);
  var decodePskLabel = mapDecoders([decodePskId, decodeUint16, decodeUint16], (id, index, count) => ({ id, index, count }));

  // node_modules/ts-mls/dist/proposal.js
  var addEncoder = contramapBufferEncoder(keyPackageEncoder, (a) => a.keyPackage);
  var encodeAdd = encode(addEncoder);
  var decodeAdd = mapDecoder(decodeKeyPackage, (keyPackage) => ({ keyPackage }));
  var updateEncoder = contramapBufferEncoder(leafNodeEncoder, (u) => u.leafNode);
  var encodeUpdate = encode(updateEncoder);
  var decodeUpdate = mapDecoder(decodeLeafNodeUpdate, (leafNode) => ({ leafNode }));
  var removeEncoder = contramapBufferEncoder(uint32Encoder, (r) => r.removed);
  var encodeRemove = encode(removeEncoder);
  var decodeRemove = mapDecoder(decodeUint32, (removed) => ({ removed }));
  var pskEncoder = contramapBufferEncoder(pskIdEncoder, (p) => p.preSharedKeyId);
  var encodePSK = encode(pskEncoder);
  var decodePSK = mapDecoder(decodePskId, (preSharedKeyId) => ({ preSharedKeyId }));
  var reinitEncoder = contramapBufferEncoders([varLenDataEncoder, protocolVersionEncoder, ciphersuiteEncoder, varLenTypeEncoder(extensionEncoder)], (r) => [r.groupId, r.version, r.cipherSuite, r.extensions]);
  var encodeReinit = encode(reinitEncoder);
  var decodeReinit = mapDecoders([decodeVarLenData, decodeProtocolVersion, decodeCiphersuite, decodeVarLenType(decodeExtension)], (groupId, version, cipherSuite, extensions) => ({ groupId, version, cipherSuite, extensions }));
  var externalInitEncoder = contramapBufferEncoder(varLenDataEncoder, (e) => e.kemOutput);
  var encodeExternalInit = encode(externalInitEncoder);
  var decodeExternalInit = mapDecoder(decodeVarLenData, (kemOutput) => ({ kemOutput }));
  var groupContextExtensionsEncoder = contramapBufferEncoder(varLenTypeEncoder(extensionEncoder), (g) => g.extensions);
  var encodeGroupContextExtensions = encode(groupContextExtensionsEncoder);
  var decodeGroupContextExtensions = mapDecoder(decodeVarLenType(decodeExtension), (extensions) => ({ extensions }));
  var proposalAddEncoder = contramapBufferEncoders([defaultProposalTypeEncoder, addEncoder], (p) => [p.proposalType, p.add]);
  var encodeProposalAdd = encode(proposalAddEncoder);
  var proposalUpdateEncoder = contramapBufferEncoders([defaultProposalTypeEncoder, updateEncoder], (p) => [p.proposalType, p.update]);
  var encodeProposalUpdate = encode(proposalUpdateEncoder);
  var proposalRemoveEncoder = contramapBufferEncoders([defaultProposalTypeEncoder, removeEncoder], (p) => [p.proposalType, p.remove]);
  var encodeProposalRemove = encode(proposalRemoveEncoder);
  var proposalPSKEncoder = contramapBufferEncoders([defaultProposalTypeEncoder, pskEncoder], (p) => [p.proposalType, p.psk]);
  var encodeProposalPSK = encode(proposalPSKEncoder);
  var proposalReinitEncoder = contramapBufferEncoders([defaultProposalTypeEncoder, reinitEncoder], (p) => [p.proposalType, p.reinit]);
  var encodeProposalReinit = encode(proposalReinitEncoder);
  var proposalExternalInitEncoder = contramapBufferEncoders([defaultProposalTypeEncoder, externalInitEncoder], (p) => [p.proposalType, p.externalInit]);
  var encodeProposalExternalInit = encode(proposalExternalInitEncoder);
  var proposalGroupContextExtensionsEncoder = contramapBufferEncoders([defaultProposalTypeEncoder, groupContextExtensionsEncoder], (p) => [p.proposalType, p.groupContextExtensions]);
  var encodeProposalGroupContextExtensions = encode(proposalGroupContextExtensionsEncoder);
  var proposalCustomEncoder = contramapBufferEncoders([uint16Encoder, varLenDataEncoder], (p) => [p.proposalType, p.proposalData]);
  var encodeProposalCustom = encode(proposalCustomEncoder);
  var proposalEncoder = (p) => {
    switch (p.proposalType) {
      case "add":
        return proposalAddEncoder(p);
      case "update":
        return proposalUpdateEncoder(p);
      case "remove":
        return proposalRemoveEncoder(p);
      case "psk":
        return proposalPSKEncoder(p);
      case "reinit":
        return proposalReinitEncoder(p);
      case "external_init":
        return proposalExternalInitEncoder(p);
      case "group_context_extensions":
        return proposalGroupContextExtensionsEncoder(p);
      default:
        return proposalCustomEncoder(p);
    }
  };
  var encodeProposal = encode(proposalEncoder);
  var decodeProposalAdd = mapDecoder(decodeAdd, (add2) => ({ proposalType: "add", add: add2 }));
  var decodeProposalUpdate = mapDecoder(decodeUpdate, (update) => ({
    proposalType: "update",
    update
  }));
  var decodeProposalRemove = mapDecoder(decodeRemove, (remove) => ({
    proposalType: "remove",
    remove
  }));
  var decodeProposalPSK = mapDecoder(decodePSK, (psk) => ({ proposalType: "psk", psk }));
  var decodeProposalReinit = mapDecoder(decodeReinit, (reinit) => ({
    proposalType: "reinit",
    reinit
  }));
  var decodeProposalExternalInit = mapDecoder(decodeExternalInit, (externalInit) => ({ proposalType: "external_init", externalInit }));
  var decodeProposalGroupContextExtensions = mapDecoder(decodeGroupContextExtensions, (groupContextExtensions) => ({ proposalType: "group_context_extensions", groupContextExtensions }));
  function decodeProposalCustom(proposalType) {
    return mapDecoder(decodeVarLenData, (proposalData) => ({ proposalType, proposalData }));
  }
  var decodeProposal = orDecoder(flatMapDecoder(decodeDefaultProposalType, (proposalType) => {
    switch (proposalType) {
      case "add":
        return decodeProposalAdd;
      case "update":
        return decodeProposalUpdate;
      case "remove":
        return decodeProposalRemove;
      case "psk":
        return decodeProposalPSK;
      case "reinit":
        return decodeProposalReinit;
      case "external_init":
        return decodeProposalExternalInit;
      case "group_context_extensions":
        return decodeProposalGroupContextExtensions;
    }
  }), flatMapDecoder(decodeUint16, (n) => decodeProposalCustom(n)));

  // node_modules/ts-mls/dist/proposalOrRefType.js
  var proposalOrRefTypes = {
    proposal: 1,
    reference: 2
  };
  var proposalOrRefTypeEncoder = contramapBufferEncoder(uint8Encoder, (t) => proposalOrRefTypes[t]);
  var encodeProposalOrRefType = encode(proposalOrRefTypeEncoder);
  var decodeProposalOrRefType = mapDecoderOption(decodeUint8, enumNumberToKey(proposalOrRefTypes));
  var proposalOrRefProposalEncoder = contramapBufferEncoders([proposalOrRefTypeEncoder, proposalEncoder], (p) => [p.proposalOrRefType, p.proposal]);
  var encodeProposalOrRefProposal = encode(proposalOrRefProposalEncoder);
  var proposalOrRefProposalRefEncoder = contramapBufferEncoders([proposalOrRefTypeEncoder, varLenDataEncoder], (r) => [r.proposalOrRefType, r.reference]);
  var encodeProposalOrRefProposalRef = encode(proposalOrRefProposalRefEncoder);
  var proposalOrRefEncoder = (input) => {
    switch (input.proposalOrRefType) {
      case "proposal":
        return proposalOrRefProposalEncoder(input);
      case "reference":
        return proposalOrRefProposalRefEncoder(input);
    }
  };
  var encodeProposalOrRef = encode(proposalOrRefEncoder);
  var decodeProposalOrRef = flatMapDecoder(decodeProposalOrRefType, (proposalOrRefType) => {
    switch (proposalOrRefType) {
      case "proposal":
        return mapDecoder(decodeProposal, (proposal) => ({ proposalOrRefType, proposal }));
      case "reference":
        return mapDecoder(decodeVarLenData, (reference) => ({ proposalOrRefType, reference }));
    }
  });

  // node_modules/ts-mls/dist/groupContext.js
  var groupContextEncoder = contramapBufferEncoders([
    protocolVersionEncoder,
    ciphersuiteEncoder,
    varLenDataEncoder,
    // groupId
    uint64Encoder,
    // epoch
    varLenDataEncoder,
    // treeHash
    varLenDataEncoder,
    // confirmedTranscriptHash
    varLenTypeEncoder(extensionEncoder)
  ], (gc) => [gc.version, gc.cipherSuite, gc.groupId, gc.epoch, gc.treeHash, gc.confirmedTranscriptHash, gc.extensions]);
  var encodeGroupContext = encode(groupContextEncoder);
  var decodeGroupContext = mapDecoders([
    decodeProtocolVersion,
    decodeCiphersuite,
    decodeVarLenData,
    // groupId
    decodeUint64,
    // epoch
    decodeVarLenData,
    // treeHash
    decodeVarLenData,
    // confirmedTranscriptHash
    decodeVarLenType(decodeExtension)
  ], (version, cipherSuite, groupId, epoch, treeHash2, confirmedTranscriptHash, extensions) => ({
    version,
    cipherSuite,
    groupId,
    epoch,
    treeHash: treeHash2,
    confirmedTranscriptHash,
    extensions
  }));

  // node_modules/ts-mls/dist/nodeType.js
  var nodeTypes = {
    leaf: 1,
    parent: 2
  };
  var nodeTypeEncoder = contramapBufferEncoder(uint8Encoder, (t) => nodeTypes[t]);
  var encodeNodeType = encode(nodeTypeEncoder);
  var decodeNodeType = mapDecoderOption(decodeUint8, enumNumberToKey(nodeTypes));

  // node_modules/ts-mls/dist/parentNode.js
  var parentNodeEncoder = contramapBufferEncoders([varLenDataEncoder, varLenDataEncoder, varLenTypeEncoder(uint32Encoder)], (node) => [node.hpkePublicKey, node.parentHash, node.unmergedLeaves]);
  var encodeParentNode = encode(parentNodeEncoder);
  var decodeParentNode = mapDecoders([decodeVarLenData, decodeVarLenData, decodeVarLenType(decodeUint32)], (hpkePublicKey, parentHash, unmergedLeaves) => ({
    hpkePublicKey,
    parentHash,
    unmergedLeaves
  }));

  // node_modules/ts-mls/dist/ratchetTree.js
  var nodeEncoder = (node) => {
    switch (node.nodeType) {
      case "parent":
        return contramapBufferEncoders([nodeTypeEncoder, parentNodeEncoder], (n) => [n.nodeType, n.parent])(node);
      case "leaf":
        return contramapBufferEncoders([nodeTypeEncoder, leafNodeEncoder], (n) => [n.nodeType, n.leaf])(node);
    }
  };
  var encodeNode = encode(nodeEncoder);
  var decodeNode = flatMapDecoder(decodeNodeType, (nodeType) => {
    switch (nodeType) {
      case "parent":
        return mapDecoder(decodeParentNode, (parent2) => ({
          nodeType,
          parent: parent2
        }));
      case "leaf":
        return mapDecoder(decodeLeafNode, (leaf) => ({
          nodeType,
          leaf
        }));
    }
  });
  function extendRatchetTree(tree) {
    const lastIndex = tree.length - 1;
    if (tree[lastIndex] === void 0) {
      throw new InternalError("The last node in the ratchet tree must be non-blank.");
    }
    const neededSize = nextFullBinaryTreeSize(tree.length);
    const copy = tree.slice();
    while (copy.length < neededSize) {
      copy.push(void 0);
    }
    return copy;
  }
  function nextFullBinaryTreeSize(n) {
    let d = 0;
    while ((1 << d + 1) - 1 < n) {
      d++;
    }
    return (1 << d + 1) - 1;
  }
  function stripBlankNodes(tree) {
    let lastNonBlank = tree.length - 1;
    while (lastNonBlank >= 0 && tree[lastNonBlank] === void 0) {
      lastNonBlank--;
    }
    return tree.slice(0, lastNonBlank + 1);
  }
  var ratchetTreeEncoder = contramapBufferEncoder(varLenTypeEncoder(optionalEncoder(nodeEncoder)), stripBlankNodes);
  var encodeRatchetTree = encode(ratchetTreeEncoder);
  var decodeRatchetTree = mapDecoder(decodeVarLenType(decodeOptional(decodeNode)), extendRatchetTree);

  // node_modules/ts-mls/dist/treeHash.js
  var leafNodeHashInputEncoder = contramapBufferEncoders([nodeTypeEncoder, uint32Encoder, optionalEncoder(leafNodeEncoder)], (input) => [input.nodeType, input.leafIndex, input.leafNode]);
  var encodeLeafNodeHashInput = encode(leafNodeHashInputEncoder);
  var decodeLeafNodeHashInput = mapDecoders([decodeUint32, decodeOptional(decodeLeafNode)], (leafIndex, leafNode) => ({
    nodeType: "leaf",
    leafIndex,
    leafNode
  }));
  var parentNodeHashInputEncoder = contramapBufferEncoders([nodeTypeEncoder, optionalEncoder(parentNodeEncoder), varLenDataEncoder, varLenDataEncoder], (input) => [input.nodeType, input.parentNode, input.leftHash, input.rightHash]);
  var encodeParentNodeHashInput = encode(parentNodeHashInputEncoder);
  var decodeParentNodeHashInput = mapDecoders([decodeOptional(decodeParentNode), decodeVarLenData, decodeVarLenData], (parentNode, leftHash, rightHash) => ({
    nodeType: "parent",
    parentNode,
    leftHash,
    rightHash
  }));
  var treeHashInputEncoder = (input) => {
    switch (input.nodeType) {
      case "leaf":
        return leafNodeHashInputEncoder(input);
      case "parent":
        return parentNodeHashInputEncoder(input);
    }
  };
  var encodeTreeHashInput = encode(treeHashInputEncoder);
  var decodeTreeHashInput = flatMapDecoder(decodeNodeType, (nodeType) => {
    switch (nodeType) {
      case "leaf":
        return decodeLeafNodeHashInput;
      case "parent":
        return decodeParentNodeHashInput;
    }
  });

  // node_modules/ts-mls/dist/parentHash.js
  var parentHashInputEncoder = contramapBufferEncoders([varLenDataEncoder, varLenDataEncoder, varLenDataEncoder], (i) => [i.encryptionKey, i.parentHash, i.originalSiblingTreeHash]);
  var encodeParentHashInput = encode(parentHashInputEncoder);
  var decodeParentHashInput = mapDecoders([decodeVarLenData, decodeVarLenData, decodeVarLenData], (encryptionKey, parentHash, originalSiblingTreeHash) => ({
    encryptionKey,
    parentHash,
    originalSiblingTreeHash
  }));

  // node_modules/ts-mls/dist/hpkeCiphertext.js
  var hpkeCiphertextEncoder = contramapBufferEncoders([varLenDataEncoder, varLenDataEncoder], (egs) => [egs.kemOutput, egs.ciphertext]);
  var encodeHpkeCiphertext = encode(hpkeCiphertextEncoder);
  var decodeHpkeCiphertext = mapDecoders([decodeVarLenData, decodeVarLenData], (kemOutput, ciphertext) => ({ kemOutput, ciphertext }));

  // node_modules/ts-mls/dist/updatePath.js
  var updatePathNodeEncoder = contramapBufferEncoders([varLenDataEncoder, varLenTypeEncoder(hpkeCiphertextEncoder)], (node) => [node.hpkePublicKey, node.encryptedPathSecret]);
  var encodeUpdatePathNode = encode(updatePathNodeEncoder);
  var decodeUpdatePathNode = mapDecoders([decodeVarLenData, decodeVarLenType(decodeHpkeCiphertext)], (hpkePublicKey, encryptedPathSecret) => ({ hpkePublicKey, encryptedPathSecret }));
  var updatePathEncoder = contramapBufferEncoders([leafNodeEncoder, varLenTypeEncoder(updatePathNodeEncoder)], (path) => [path.leafNode, path.nodes]);
  var encodeUpdatePath = encode(updatePathEncoder);
  var decodeUpdatePath = mapDecoders([decodeLeafNodeCommit, decodeVarLenType(decodeUpdatePathNode)], (leafNode, nodes) => ({ leafNode, nodes }));

  // node_modules/ts-mls/dist/commit.js
  var commitEncoder = contramapBufferEncoders([varLenTypeEncoder(proposalOrRefEncoder), optionalEncoder(updatePathEncoder)], (commit) => [commit.proposals, commit.path]);
  var encodeCommit = encode(commitEncoder);
  var decodeCommit = mapDecoders([decodeVarLenType(decodeProposalOrRef), decodeOptional(decodeUpdatePath)], (proposals, path) => ({ proposals, path }));

  // node_modules/ts-mls/dist/contentType.js
  var contentTypes = {
    application: 1,
    proposal: 2,
    commit: 3
  };
  var contentTypeEncoder = contramapBufferEncoder(uint8Encoder, (t) => contentTypes[t]);
  var encodeContentType = encode(contentTypeEncoder);
  var decodeContentType = mapDecoderOption(decodeUint8, enumNumberToKey(contentTypes));

  // node_modules/ts-mls/dist/wireformat.js
  var wireformats = {
    mls_public_message: 1,
    mls_private_message: 2,
    mls_welcome: 3,
    mls_group_info: 4,
    mls_key_package: 5
  };
  var wireformatEncoder = (s) => contramapBufferEncoder(uint16Encoder, (t) => wireformats[t])(s);
  var encodeWireformat = encode(wireformatEncoder);
  var decodeWireformat = mapDecoderOption(decodeUint16, enumNumberToKey(wireformats));

  // node_modules/ts-mls/dist/sender.js
  var senderTypes = {
    member: 1,
    external: 2,
    new_member_proposal: 3,
    new_member_commit: 4
  };
  var senderTypeEncoder = contramapBufferEncoder(uint8Encoder, (t) => senderTypes[t]);
  var encodeSenderType = encode(senderTypeEncoder);
  var decodeSenderType = mapDecoderOption(decodeUint8, enumNumberToKey(senderTypes));
  var senderEncoder = (s) => {
    switch (s.senderType) {
      case "member":
        return contramapBufferEncoders([senderTypeEncoder, uint32Encoder], (s2) => [s2.senderType, s2.leafIndex])(s);
      case "external":
        return contramapBufferEncoders([senderTypeEncoder, uint32Encoder], (s2) => [s2.senderType, s2.senderIndex])(s);
      case "new_member_proposal":
      case "new_member_commit":
        return senderTypeEncoder(s.senderType);
    }
  };
  var encodeSender = encode(senderEncoder);
  var decodeSender = flatMapDecoder(decodeSenderType, (senderType) => {
    switch (senderType) {
      case "member":
        return mapDecoder(decodeUint32, (leafIndex) => ({
          senderType,
          leafIndex
        }));
      case "external":
        return mapDecoder(decodeUint32, (senderIndex) => ({
          senderType,
          senderIndex
        }));
      case "new_member_proposal":
        return mapDecoder(() => [void 0, 0], () => ({
          senderType
        }));
      case "new_member_commit":
        return mapDecoder(() => [void 0, 0], () => ({
          senderType
        }));
    }
  });
  var reuseGuardEncoder = (g) => [
    4,
    (offset, buffer) => {
      const view = new Uint8Array(buffer, offset, 4);
      view.set(g, 0);
    }
  ];
  var encodeReuseGuard = encode(reuseGuardEncoder);
  var decodeReuseGuard = (b, offset) => {
    return [b.subarray(offset, offset + 4), 4];
  };
  var senderDataEncoder = contramapBufferEncoders([uint32Encoder, uint32Encoder, reuseGuardEncoder], (s) => [s.leafIndex, s.generation, s.reuseGuard]);
  var encodeSenderData = encode(senderDataEncoder);
  var decodeSenderData = mapDecoders([decodeUint32, decodeUint32, decodeReuseGuard], (leafIndex, generation, reuseGuard) => ({
    leafIndex,
    generation,
    reuseGuard
  }));
  var senderDataAADEncoder = contramapBufferEncoders([varLenDataEncoder, uint64Encoder, contentTypeEncoder], (aad) => [aad.groupId, aad.epoch, aad.contentType]);
  var encodeSenderDataAAD = encode(senderDataAADEncoder);
  var decodeSenderDataAAD = mapDecoders([decodeVarLenData, decodeUint64, decodeContentType], (groupId, epoch, contentType) => ({
    groupId,
    epoch,
    contentType
  }));

  // node_modules/ts-mls/dist/framedContent.js
  var framedContentApplicationDataEncoder = contramapBufferEncoders([contentTypeEncoder, varLenDataEncoder], (f) => [f.contentType, f.applicationData]);
  var encodeFramedContentApplicationData = encode(framedContentApplicationDataEncoder);
  var framedContentProposalDataEncoder = contramapBufferEncoders([contentTypeEncoder, proposalEncoder], (f) => [f.contentType, f.proposal]);
  var encodeFramedContentProposalData = encode(framedContentProposalDataEncoder);
  var framedContentCommitDataEncoder = contramapBufferEncoders([contentTypeEncoder, commitEncoder], (f) => [f.contentType, f.commit]);
  var encodeFramedContentCommitData = encode(framedContentCommitDataEncoder);
  var framedContentInfoEncoder = (fc) => {
    switch (fc.contentType) {
      case "application":
        return framedContentApplicationDataEncoder(fc);
      case "proposal":
        return framedContentProposalDataEncoder(fc);
      case "commit":
        return framedContentCommitDataEncoder(fc);
    }
  };
  var encodeFramedContentInfo = encode(framedContentInfoEncoder);
  var decodeFramedContentApplicationData = mapDecoder(decodeVarLenData, (applicationData) => ({ contentType: "application", applicationData }));
  var decodeFramedContentProposalData = mapDecoder(decodeProposal, (proposal) => ({ contentType: "proposal", proposal }));
  var decodeFramedContentCommitData = mapDecoder(decodeCommit, (commit) => ({
    contentType: "commit",
    commit
  }));
  var decodeFramedContentInfo = flatMapDecoder(decodeContentType, (contentType) => {
    switch (contentType) {
      case "application":
        return decodeFramedContentApplicationData;
      case "proposal":
        return decodeFramedContentProposalData;
      case "commit":
        return decodeFramedContentCommitData;
    }
  });
  var framedContentEncoder = contramapBufferEncoders([varLenDataEncoder, uint64Encoder, senderEncoder, varLenDataEncoder, framedContentInfoEncoder], (fc) => [fc.groupId, fc.epoch, fc.sender, fc.authenticatedData, fc]);
  var encodeFramedContent = encode(framedContentEncoder);
  var decodeFramedContent = mapDecoders([decodeVarLenData, decodeUint64, decodeSender, decodeVarLenData, decodeFramedContentInfo], (groupId, epoch, sender, authenticatedData, info) => ({
    groupId,
    epoch,
    sender,
    authenticatedData,
    ...info
  }));
  var senderInfoEncoder = (info) => {
    switch (info.senderType) {
      case "member":
      case "new_member_commit":
        return groupContextEncoder(info.context);
      case "external":
      case "new_member_proposal":
        return encVoid;
    }
  };
  var encodeSenderInfo = encode(senderInfoEncoder);
  var framedContentTBSEncoder = contramapBufferEncoders([protocolVersionEncoder, wireformatEncoder, framedContentEncoder, senderInfoEncoder], (f) => [f.protocolVersion, f.wireformat, f.content, f]);
  var encodeFramedContentTBS = encode(framedContentTBSEncoder);
  var encodeFramedContentAuthDataContent = (authData) => {
    switch (authData.contentType) {
      case "commit":
        return encodeFramedContentAuthDataCommit(authData);
      case "application":
      case "proposal":
        return encVoid;
    }
  };
  var encodeFramedContentAuthDataCommit = contramapBufferEncoder(varLenDataEncoder, (data) => data.confirmationTag);
  var framedContentAuthDataEncoder = contramapBufferEncoders([varLenDataEncoder, encodeFramedContentAuthDataContent], (d) => [d.signature, d]);
  var encodeFramedContentAuthData = encode(framedContentAuthDataEncoder);
  var decodeFramedContentAuthDataCommit = mapDecoder(decodeVarLenData, (confirmationTag) => ({
    contentType: "commit",
    confirmationTag
  }));
  function decodeFramedContentAuthData(contentType) {
    switch (contentType) {
      case "commit":
        return mapDecoders([decodeVarLenData, decodeFramedContentAuthDataCommit], (signature, commitData) => ({
          signature,
          ...commitData
        }));
      case "application":
      case "proposal":
        return mapDecoder(decodeVarLenData, (signature) => ({
          signature,
          contentType
        }));
    }
  }

  // node_modules/ts-mls/dist/authenticatedContent.js
  var authenticatedContentEncoder = contramapBufferEncoders([wireformatEncoder, framedContentEncoder, framedContentAuthDataEncoder], (a) => [a.wireformat, a.content, a.auth]);
  var encodeAuthenticatedContent = encode(authenticatedContentEncoder);
  var decodeAuthenticatedContent = mapDecoders([
    decodeWireformat,
    flatMapDecoder(decodeFramedContent, (content) => {
      return mapDecoder(decodeFramedContentAuthData(content.contentType), (auth) => ({ content, auth }));
    })
  ], (wireformat, contentAuth) => ({
    wireformat,
    ...contentAuth
  }));
  var authenticatedContentTBMEncoder = contramapBufferEncoders([framedContentTBSEncoder, framedContentAuthDataEncoder], (t) => [t.contentTbs, t.auth]);
  var encodeAuthenticatedContentTBM = encode(authenticatedContentTBMEncoder);

  // node_modules/ts-mls/dist/publicMessage.js
  var publicMessageInfoEncoder = (info) => {
    switch (info.senderType) {
      case "member":
        return varLenDataEncoder(info.membershipTag);
      case "external":
      case "new_member_proposal":
      case "new_member_commit":
        return encVoid;
    }
  };
  var encodePublicMessageInfo = encode(publicMessageInfoEncoder);
  function decodePublicMessageInfo(senderType) {
    switch (senderType) {
      case "member":
        return mapDecoder(decodeVarLenData, (membershipTag) => ({
          senderType,
          membershipTag
        }));
      case "external":
      case "new_member_proposal":
      case "new_member_commit":
        return succeedDecoder({ senderType });
    }
  }
  var publicMessageEncoder = contramapBufferEncoders([framedContentEncoder, framedContentAuthDataEncoder, publicMessageInfoEncoder], (msg) => [msg.content, msg.auth, msg]);
  var encodePublicMessage = encode(publicMessageEncoder);
  var decodePublicMessage = flatMapDecoder(decodeFramedContent, (content) => mapDecoders([decodeFramedContentAuthData(content.contentType), decodePublicMessageInfo(content.sender.senderType)], (auth, info) => ({
    ...info,
    content,
    auth
  })));

  // node_modules/ts-mls/dist/requiredCapabilities.js
  var requiredCapabilitiesEncoder = contramapBufferEncoders([varLenTypeEncoder(uint16Encoder), varLenTypeEncoder(uint16Encoder), varLenTypeEncoder(credentialTypeEncoder)], (rc) => [rc.extensionTypes, rc.proposalTypes, rc.credentialTypes]);
  var encodeRequiredCapabilities = encode(requiredCapabilitiesEncoder);
  var decodeRequiredCapabilities = mapDecoders([decodeVarLenType(decodeUint16), decodeVarLenType(decodeUint16), decodeVarLenType(decodeCredentialType)], (extensionTypes, proposalTypes, credentialTypes2) => ({ extensionTypes, proposalTypes, credentialTypes: credentialTypes2 }));

  // node_modules/ts-mls/dist/groupInfo.js
  var groupInfoTBSEncoder = contramapBufferEncoders([groupContextEncoder, varLenTypeEncoder(extensionEncoder), varLenDataEncoder, uint32Encoder], (g) => [g.groupContext, g.extensions, g.confirmationTag, g.signer]);
  var encodeGroupInfoTBS = encode(groupInfoTBSEncoder);
  var decodeGroupInfoTBS = mapDecoders([decodeGroupContext, decodeVarLenType(decodeExtension), decodeVarLenData, decodeUint32], (groupContext, extensions, confirmationTag, signer) => ({
    groupContext,
    extensions,
    confirmationTag,
    signer
  }));
  var groupInfoEncoder = contramapBufferEncoders([groupInfoTBSEncoder, varLenDataEncoder], (g) => [g, g.signature]);
  var encodeGroupInfo = encode(groupInfoEncoder);
  var decodeGroupInfo = mapDecoders([decodeGroupInfoTBS, decodeVarLenData], (tbs, signature) => ({
    ...tbs,
    signature
  }));

  // node_modules/ts-mls/dist/transcriptHash.js
  var confirmedTranscriptHashInputEncoder = contramapBufferEncoders([wireformatEncoder, framedContentEncoder, varLenDataEncoder], (input) => [input.wireformat, input.content, input.signature]);
  var encodeConfirmedTranscriptHashInput = encode(confirmedTranscriptHashInputEncoder);
  var decodeConfirmedTranscriptHashInput = mapDecodersOption([decodeWireformat, decodeFramedContent, decodeVarLenData], (wireformat, content, signature) => {
    if (content.contentType === "commit")
      return {
        wireformat,
        content,
        signature
      };
    else
      return void 0;
  });

  // node_modules/ts-mls/dist/util/byteArray.js
  function bytesToArrayBuffer(b) {
    if (b.buffer instanceof ArrayBuffer) {
      if (b.byteOffset === 0 && b.byteLength === b.buffer.byteLength) {
        return b.buffer;
      }
      return b.buffer.slice(b.byteOffset, b.byteOffset + b.byteLength);
    } else {
      const ab = new ArrayBuffer(b.byteLength);
      const arr = new Uint8Array(ab);
      arr.set(b, 0);
      return ab;
    }
  }
  function toBufferSource(b) {
    if (b.buffer instanceof ArrayBuffer)
      return b;
    const ab = new ArrayBuffer(b.byteLength);
    const arr = new Uint8Array(ab);
    arr.set(b, 0);
    return ab;
  }
  function concatUint8Arrays(a, b) {
    const result = new Uint8Array(a.length + b.length);
    result.set(a, 0);
    result.set(b, a.length);
    return result;
  }

  // node_modules/ts-mls/dist/groupSecrets.js
  var groupSecretsEncoder = contramapBufferEncoders([varLenDataEncoder, optionalEncoder(varLenDataEncoder), varLenTypeEncoder(pskIdEncoder)], (gs) => [gs.joinerSecret, gs.pathSecret, gs.psks]);
  var encodeGroupSecrets = encode(groupSecretsEncoder);
  var decodeGroupSecrets = mapDecoders([decodeVarLenData, decodeOptional(decodeVarLenData), decodeVarLenType(decodePskId)], (joinerSecret, pathSecret, psks) => ({ joinerSecret, pathSecret, psks }));

  // node_modules/ts-mls/dist/welcome.js
  var encryptedGroupSecretsEncoder = contramapBufferEncoders([varLenDataEncoder, hpkeCiphertextEncoder], (egs) => [egs.newMember, egs.encryptedGroupSecrets]);
  var encodeEncryptedGroupSecrets = encode(encryptedGroupSecretsEncoder);
  var decodeEncryptedGroupSecrets = mapDecoders([decodeVarLenData, decodeHpkeCiphertext], (newMember, encryptedGroupSecrets) => ({ newMember, encryptedGroupSecrets }));
  var welcomeEncoder = contramapBufferEncoders([ciphersuiteEncoder, varLenTypeEncoder(encryptedGroupSecretsEncoder), varLenDataEncoder], (welcome) => [welcome.cipherSuite, welcome.secrets, welcome.encryptedGroupInfo]);
  var encodeWelcome = encode(welcomeEncoder);
  var decodeWelcome = mapDecoders([decodeCiphersuite, decodeVarLenType(decodeEncryptedGroupSecrets), decodeVarLenData], (cipherSuite, secrets, encryptedGroupInfo) => ({ cipherSuite, secrets, encryptedGroupInfo }));

  // node_modules/ts-mls/dist/privateMessage.js
  var privateMessageEncoder = contramapBufferEncoders([varLenDataEncoder, uint64Encoder, contentTypeEncoder, varLenDataEncoder, varLenDataEncoder, varLenDataEncoder], (msg) => [msg.groupId, msg.epoch, msg.contentType, msg.authenticatedData, msg.encryptedSenderData, msg.ciphertext]);
  var encodePrivateMessage = encode(privateMessageEncoder);
  var decodePrivateMessage = mapDecoders([decodeVarLenData, decodeUint64, decodeContentType, decodeVarLenData, decodeVarLenData, decodeVarLenData], (groupId, epoch, contentType, authenticatedData, encryptedSenderData, ciphertext) => ({
    groupId,
    epoch,
    contentType,
    authenticatedData,
    encryptedSenderData,
    ciphertext
  }));
  var privateContentAADEncoder = contramapBufferEncoders([varLenDataEncoder, uint64Encoder, contentTypeEncoder, varLenDataEncoder], (aad) => [aad.groupId, aad.epoch, aad.contentType, aad.authenticatedData]);
  var encodePrivateContentAAD = encode(privateContentAADEncoder);
  var decodePrivateContentAAD = mapDecoders([decodeVarLenData, decodeUint64, decodeContentType, decodeVarLenData], (groupId, epoch, contentType, authenticatedData) => ({
    groupId,
    epoch,
    contentType,
    authenticatedData
  }));

  // node_modules/ts-mls/dist/crypto/implementation/default/makeHashImpl.js
  function makeHashImpl(sc, h) {
    return {
      async digest(data) {
        const result = await sc.digest(h, toBufferSource(data));
        return new Uint8Array(result);
      },
      async mac(key, data) {
        const result = await sc.sign("HMAC", await importMacKey(key, h), toBufferSource(data));
        return new Uint8Array(result);
      },
      async verifyMac(key, mac, data) {
        return sc.verify("HMAC", await importMacKey(key, h), toBufferSource(mac), toBufferSource(data));
      }
    };
  }
  function importMacKey(rawKey, h) {
    return crypto.subtle.importKey("raw", toBufferSource(rawKey), {
      name: "HMAC",
      hash: { name: h }
    }, false, ["sign", "verify"]);
  }

  // node_modules/@hpke/common/esm/src/errors.js
  var HpkeError = class extends Error {
    constructor(e) {
      let message;
      if (e instanceof Error) {
        message = e.message;
      } else if (typeof e === "string") {
        message = e;
      } else {
        message = "";
      }
      super(message);
      this.name = this.constructor.name;
    }
  };
  var InvalidParamError = class extends HpkeError {
  };
  var SerializeError = class extends HpkeError {
  };
  var DeserializeError = class extends HpkeError {
  };
  var EncapError = class extends HpkeError {
  };
  var DecapError = class extends HpkeError {
  };
  var ExportError = class extends HpkeError {
  };
  var SealError = class extends HpkeError {
  };
  var OpenError = class extends HpkeError {
  };
  var MessageLimitReachedError = class extends HpkeError {
  };
  var DeriveKeyPairError = class extends HpkeError {
  };
  var NotSupportedError = class extends HpkeError {
  };

  // node_modules/@hpke/common/esm/_dnt.shims.js
  var dntGlobals = {};
  var dntGlobalThis = createMergeProxy(globalThis, dntGlobals);
  function createMergeProxy(baseObj, extObj) {
    return new Proxy(baseObj, {
      get(_target, prop, _receiver) {
        if (prop in extObj) {
          return extObj[prop];
        } else {
          return baseObj[prop];
        }
      },
      set(_target, prop, value) {
        if (prop in extObj) {
          delete extObj[prop];
        }
        baseObj[prop] = value;
        return true;
      },
      deleteProperty(_target, prop) {
        let success = false;
        if (prop in extObj) {
          delete extObj[prop];
          success = true;
        }
        if (prop in baseObj) {
          delete baseObj[prop];
          success = true;
        }
        return success;
      },
      ownKeys(_target) {
        const baseKeys = Reflect.ownKeys(baseObj);
        const extKeys = Reflect.ownKeys(extObj);
        const extKeysSet = new Set(extKeys);
        return [...baseKeys.filter((k) => !extKeysSet.has(k)), ...extKeys];
      },
      defineProperty(_target, prop, desc) {
        if (prop in extObj) {
          delete extObj[prop];
        }
        Reflect.defineProperty(baseObj, prop, desc);
        return true;
      },
      getOwnPropertyDescriptor(_target, prop) {
        if (prop in extObj) {
          return Reflect.getOwnPropertyDescriptor(extObj, prop);
        } else {
          return Reflect.getOwnPropertyDescriptor(baseObj, prop);
        }
      },
      has(_target, prop) {
        return prop in extObj || prop in baseObj;
      }
    });
  }

  // node_modules/@hpke/common/esm/src/algorithm.js
  async function loadSubtleCrypto() {
    if (dntGlobalThis !== void 0 && globalThis.crypto !== void 0) {
      return globalThis.crypto.subtle;
    }
    try {
      const { webcrypto } = await import("crypto");
      return webcrypto.subtle;
    } catch (e) {
      throw new NotSupportedError(e);
    }
  }
  var NativeAlgorithm = class {
    constructor() {
      Object.defineProperty(this, "_api", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
    }
    async _setup() {
      if (this._api !== void 0) {
        return;
      }
      this._api = await loadSubtleCrypto();
    }
  };

  // node_modules/@hpke/common/esm/src/identifiers.js
  var Mode = {
    Base: 0,
    Psk: 1,
    Auth: 2,
    AuthPsk: 3
  };
  var KemId = {
    NotAssigned: 0,
    DhkemP256HkdfSha256: 16,
    DhkemP384HkdfSha384: 17,
    DhkemP521HkdfSha512: 18,
    DhkemSecp256k1HkdfSha256: 19,
    DhkemX25519HkdfSha256: 32,
    DhkemX448HkdfSha512: 33,
    HybridkemX25519Kyber768: 48,
    MlKem512: 64,
    MlKem768: 65,
    MlKem1024: 66,
    XWing: 25722
  };
  var KdfId = {
    HkdfSha256: 1,
    HkdfSha384: 2,
    HkdfSha512: 3
  };
  var AeadId = {
    Aes128Gcm: 1,
    Aes256Gcm: 2,
    Chacha20Poly1305: 3,
    ExportOnly: 65535
  };

  // node_modules/@hpke/common/esm/src/consts.js
  var INPUT_LENGTH_LIMIT = 8192;
  var INFO_LENGTH_LIMIT = 65536;
  var MINIMUM_PSK_LENGTH = 32;
  var EMPTY = new Uint8Array(0);

  // node_modules/@hpke/common/esm/src/interfaces/kemInterface.js
  var SUITE_ID_HEADER_KEM = new Uint8Array([
    75,
    69,
    77,
    0,
    0
  ]);

  // node_modules/@hpke/common/esm/src/utils/misc.js
  var isCryptoKeyPair = (x) => typeof x === "object" && x !== null && typeof x.privateKey === "object" && typeof x.publicKey === "object";
  function i2Osp(n, w) {
    if (w <= 0) {
      throw new Error("i2Osp: too small size");
    }
    if (n >= 256 ** w) {
      throw new Error("i2Osp: too large integer");
    }
    const ret = new Uint8Array(w);
    for (let i = 0; i < w && n; i++) {
      ret[w - (i + 1)] = n % 256;
      n = n >> 8;
    }
    return ret;
  }
  function concat(a, b) {
    const ret = new Uint8Array(a.length + b.length);
    ret.set(a, 0);
    ret.set(b, a.length);
    return ret;
  }
  function base64UrlToBytes(v) {
    const base64 = v.replace(/-/g, "+").replace(/_/g, "/");
    const byteString = atob(base64);
    const ret = new Uint8Array(byteString.length);
    for (let i = 0; i < byteString.length; i++) {
      ret[i] = byteString.charCodeAt(i);
    }
    return ret;
  }
  function xor(a, b) {
    if (a.byteLength !== b.byteLength) {
      throw new Error("xor: different length inputs");
    }
    const buf = new Uint8Array(a.byteLength);
    for (let i = 0; i < a.byteLength; i++) {
      buf[i] = a[i] ^ b[i];
    }
    return buf;
  }

  // node_modules/@hpke/common/esm/src/kems/dhkem.js
  var LABEL_EAE_PRK = new Uint8Array([101, 97, 101, 95, 112, 114, 107]);
  var LABEL_SHARED_SECRET = new Uint8Array([
    115,
    104,
    97,
    114,
    101,
    100,
    95,
    115,
    101,
    99,
    114,
    101,
    116
  ]);
  function concat3(a, b, c) {
    const ret = new Uint8Array(a.length + b.length + c.length);
    ret.set(a, 0);
    ret.set(b, a.length);
    ret.set(c, a.length + b.length);
    return ret;
  }
  var Dhkem = class {
    constructor(id, prim, kdf) {
      Object.defineProperty(this, "id", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "secretSize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 0
      });
      Object.defineProperty(this, "encSize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 0
      });
      Object.defineProperty(this, "publicKeySize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 0
      });
      Object.defineProperty(this, "privateKeySize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 0
      });
      Object.defineProperty(this, "_prim", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_kdf", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      this.id = id;
      this._prim = prim;
      this._kdf = kdf;
      const suiteId = new Uint8Array(SUITE_ID_HEADER_KEM);
      suiteId.set(i2Osp(this.id, 2), 3);
      this._kdf.init(suiteId);
    }
    async serializePublicKey(key) {
      return await this._prim.serializePublicKey(key);
    }
    async deserializePublicKey(key) {
      return await this._prim.deserializePublicKey(key);
    }
    async serializePrivateKey(key) {
      return await this._prim.serializePrivateKey(key);
    }
    async deserializePrivateKey(key) {
      return await this._prim.deserializePrivateKey(key);
    }
    async importKey(format, key, isPublic = true) {
      return await this._prim.importKey(format, key, isPublic);
    }
    async generateKeyPair() {
      return await this._prim.generateKeyPair();
    }
    async deriveKeyPair(ikm) {
      if (ikm.byteLength > INPUT_LENGTH_LIMIT) {
        throw new InvalidParamError("Too long ikm");
      }
      return await this._prim.deriveKeyPair(ikm);
    }
    async encap(params) {
      let ke;
      if (params.ekm === void 0) {
        ke = await this.generateKeyPair();
      } else if (isCryptoKeyPair(params.ekm)) {
        ke = params.ekm;
      } else {
        ke = await this.deriveKeyPair(params.ekm);
      }
      const enc = await this._prim.serializePublicKey(ke.publicKey);
      const pkrm = await this._prim.serializePublicKey(params.recipientPublicKey);
      try {
        let dh;
        if (params.senderKey === void 0) {
          dh = new Uint8Array(await this._prim.dh(ke.privateKey, params.recipientPublicKey));
        } else {
          const sks = isCryptoKeyPair(params.senderKey) ? params.senderKey.privateKey : params.senderKey;
          const dh1 = new Uint8Array(await this._prim.dh(ke.privateKey, params.recipientPublicKey));
          const dh2 = new Uint8Array(await this._prim.dh(sks, params.recipientPublicKey));
          dh = concat(dh1, dh2);
        }
        let kemContext;
        if (params.senderKey === void 0) {
          kemContext = concat(new Uint8Array(enc), new Uint8Array(pkrm));
        } else {
          const pks = isCryptoKeyPair(params.senderKey) ? params.senderKey.publicKey : await this._prim.derivePublicKey(params.senderKey);
          const pksm = await this._prim.serializePublicKey(pks);
          kemContext = concat3(new Uint8Array(enc), new Uint8Array(pkrm), new Uint8Array(pksm));
        }
        const sharedSecret = await this._generateSharedSecret(dh, kemContext);
        return {
          enc,
          sharedSecret
        };
      } catch (e) {
        throw new EncapError(e);
      }
    }
    async decap(params) {
      const pke = await this._prim.deserializePublicKey(params.enc);
      const skr = isCryptoKeyPair(params.recipientKey) ? params.recipientKey.privateKey : params.recipientKey;
      const pkr = isCryptoKeyPair(params.recipientKey) ? params.recipientKey.publicKey : await this._prim.derivePublicKey(params.recipientKey);
      const pkrm = await this._prim.serializePublicKey(pkr);
      try {
        let dh;
        if (params.senderPublicKey === void 0) {
          dh = new Uint8Array(await this._prim.dh(skr, pke));
        } else {
          const dh1 = new Uint8Array(await this._prim.dh(skr, pke));
          const dh2 = new Uint8Array(await this._prim.dh(skr, params.senderPublicKey));
          dh = concat(dh1, dh2);
        }
        let kemContext;
        if (params.senderPublicKey === void 0) {
          kemContext = concat(new Uint8Array(params.enc), new Uint8Array(pkrm));
        } else {
          const pksm = await this._prim.serializePublicKey(params.senderPublicKey);
          kemContext = new Uint8Array(params.enc.byteLength + pkrm.byteLength + pksm.byteLength);
          kemContext.set(new Uint8Array(params.enc), 0);
          kemContext.set(new Uint8Array(pkrm), params.enc.byteLength);
          kemContext.set(new Uint8Array(pksm), params.enc.byteLength + pkrm.byteLength);
        }
        return await this._generateSharedSecret(dh, kemContext);
      } catch (e) {
        throw new DecapError(e);
      }
    }
    async _generateSharedSecret(dh, kemContext) {
      const labeledIkm = this._kdf.buildLabeledIkm(LABEL_EAE_PRK, dh);
      const labeledInfo = this._kdf.buildLabeledInfo(LABEL_SHARED_SECRET, kemContext, this.secretSize);
      return await this._kdf.extractAndExpand(EMPTY.buffer, labeledIkm.buffer, labeledInfo.buffer, this.secretSize);
    }
  };

  // node_modules/@hpke/common/esm/src/interfaces/dhkemPrimitives.js
  var KEM_USAGES = ["deriveBits"];
  var LABEL_DKP_PRK = new Uint8Array([
    100,
    107,
    112,
    95,
    112,
    114,
    107
  ]);
  var LABEL_SK = new Uint8Array([115, 107]);

  // node_modules/@hpke/common/esm/src/utils/bignum.js
  var Bignum = class {
    constructor(size) {
      Object.defineProperty(this, "_num", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      this._num = new Uint8Array(size);
    }
    val() {
      return this._num;
    }
    reset() {
      this._num.fill(0);
    }
    set(src) {
      if (src.length !== this._num.length) {
        throw new Error("Bignum.set: invalid argument");
      }
      this._num.set(src);
    }
    isZero() {
      for (let i = 0; i < this._num.length; i++) {
        if (this._num[i] !== 0) {
          return false;
        }
      }
      return true;
    }
    lessThan(v) {
      if (v.length !== this._num.length) {
        throw new Error("Bignum.lessThan: invalid argument");
      }
      for (let i = 0; i < this._num.length; i++) {
        if (this._num[i] < v[i]) {
          return true;
        }
        if (this._num[i] > v[i]) {
          return false;
        }
      }
      return false;
    }
  };

  // node_modules/@hpke/common/esm/src/kems/dhkemPrimitives/ec.js
  var LABEL_CANDIDATE = new Uint8Array([
    99,
    97,
    110,
    100,
    105,
    100,
    97,
    116,
    101
  ]);
  var ORDER_P_256 = new Uint8Array([
    255,
    255,
    255,
    255,
    0,
    0,
    0,
    0,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    188,
    230,
    250,
    173,
    167,
    23,
    158,
    132,
    243,
    185,
    202,
    194,
    252,
    99,
    37,
    81
  ]);
  var ORDER_P_384 = new Uint8Array([
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    199,
    99,
    77,
    129,
    244,
    55,
    45,
    223,
    88,
    26,
    13,
    178,
    72,
    176,
    167,
    122,
    236,
    236,
    25,
    106,
    204,
    197,
    41,
    115
  ]);
  var ORDER_P_521 = new Uint8Array([
    1,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    255,
    250,
    81,
    134,
    135,
    131,
    191,
    47,
    150,
    107,
    127,
    204,
    1,
    72,
    247,
    9,
    165,
    208,
    59,
    181,
    201,
    184,
    137,
    156,
    71,
    174,
    187,
    111,
    183,
    30,
    145,
    56,
    100,
    9
  ]);
  var PKCS8_ALG_ID_P_256 = new Uint8Array([
    48,
    65,
    2,
    1,
    0,
    48,
    19,
    6,
    7,
    42,
    134,
    72,
    206,
    61,
    2,
    1,
    6,
    8,
    42,
    134,
    72,
    206,
    61,
    3,
    1,
    7,
    4,
    39,
    48,
    37,
    2,
    1,
    1,
    4,
    32
  ]);
  var PKCS8_ALG_ID_P_384 = new Uint8Array([
    48,
    78,
    2,
    1,
    0,
    48,
    16,
    6,
    7,
    42,
    134,
    72,
    206,
    61,
    2,
    1,
    6,
    5,
    43,
    129,
    4,
    0,
    34,
    4,
    55,
    48,
    53,
    2,
    1,
    1,
    4,
    48
  ]);
  var PKCS8_ALG_ID_P_521 = new Uint8Array([
    48,
    96,
    2,
    1,
    0,
    48,
    16,
    6,
    7,
    42,
    134,
    72,
    206,
    61,
    2,
    1,
    6,
    5,
    43,
    129,
    4,
    0,
    35,
    4,
    73,
    48,
    71,
    2,
    1,
    1,
    4,
    66
  ]);
  var Ec = class extends NativeAlgorithm {
    constructor(kem, hkdf) {
      super();
      Object.defineProperty(this, "_hkdf", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_alg", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_nPk", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_nSk", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_nDh", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_order", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_bitmask", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_pkcs8AlgId", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      this._hkdf = hkdf;
      switch (kem) {
        case KemId.DhkemP256HkdfSha256:
          this._alg = { name: "ECDH", namedCurve: "P-256" };
          this._nPk = 65;
          this._nSk = 32;
          this._nDh = 32;
          this._order = ORDER_P_256;
          this._bitmask = 255;
          this._pkcs8AlgId = PKCS8_ALG_ID_P_256;
          break;
        case KemId.DhkemP384HkdfSha384:
          this._alg = { name: "ECDH", namedCurve: "P-384" };
          this._nPk = 97;
          this._nSk = 48;
          this._nDh = 48;
          this._order = ORDER_P_384;
          this._bitmask = 255;
          this._pkcs8AlgId = PKCS8_ALG_ID_P_384;
          break;
        default:
          this._alg = { name: "ECDH", namedCurve: "P-521" };
          this._nPk = 133;
          this._nSk = 66;
          this._nDh = 66;
          this._order = ORDER_P_521;
          this._bitmask = 1;
          this._pkcs8AlgId = PKCS8_ALG_ID_P_521;
          break;
      }
    }
    async serializePublicKey(key) {
      await this._setup();
      try {
        return await this._api.exportKey("raw", key);
      } catch (e) {
        throw new SerializeError(e);
      }
    }
    async deserializePublicKey(key) {
      await this._setup();
      try {
        return await this._importRawKey(key, true);
      } catch (e) {
        throw new DeserializeError(e);
      }
    }
    async serializePrivateKey(key) {
      await this._setup();
      try {
        const jwk = await this._api.exportKey("jwk", key);
        if (!("d" in jwk)) {
          throw new Error("Not private key");
        }
        return base64UrlToBytes(jwk["d"]).buffer;
      } catch (e) {
        throw new SerializeError(e);
      }
    }
    async deserializePrivateKey(key) {
      await this._setup();
      try {
        return await this._importRawKey(key, false);
      } catch (e) {
        throw new DeserializeError(e);
      }
    }
    async importKey(format, key, isPublic) {
      await this._setup();
      try {
        if (format === "raw") {
          return await this._importRawKey(key, isPublic);
        }
        if (key instanceof ArrayBuffer) {
          throw new Error("Invalid jwk key format");
        }
        return await this._importJWK(key, isPublic);
      } catch (e) {
        throw new DeserializeError(e);
      }
    }
    async generateKeyPair() {
      await this._setup();
      try {
        return await this._api.generateKey(this._alg, true, KEM_USAGES);
      } catch (e) {
        throw new NotSupportedError(e);
      }
    }
    async deriveKeyPair(ikm) {
      await this._setup();
      try {
        const dkpPrk = await this._hkdf.labeledExtract(EMPTY.buffer, LABEL_DKP_PRK, new Uint8Array(ikm));
        const bn = new Bignum(this._nSk);
        for (let counter = 0; bn.isZero() || !bn.lessThan(this._order); counter++) {
          if (counter > 255) {
            throw new Error("Faild to derive a key pair");
          }
          const bytes = new Uint8Array(await this._hkdf.labeledExpand(dkpPrk, LABEL_CANDIDATE, i2Osp(counter, 1), this._nSk));
          bytes[0] = bytes[0] & this._bitmask;
          bn.set(bytes);
        }
        const sk = await this._deserializePkcs8Key(bn.val());
        bn.reset();
        return {
          privateKey: sk,
          publicKey: await this.derivePublicKey(sk)
        };
      } catch (e) {
        throw new DeriveKeyPairError(e);
      }
    }
    async derivePublicKey(key) {
      await this._setup();
      try {
        const jwk = await this._api.exportKey("jwk", key);
        delete jwk["d"];
        delete jwk["key_ops"];
        return await this._api.importKey("jwk", jwk, this._alg, true, []);
      } catch (e) {
        throw new DeserializeError(e);
      }
    }
    async dh(sk, pk) {
      try {
        await this._setup();
        const bits = await this._api.deriveBits({
          name: "ECDH",
          public: pk
        }, sk, this._nDh * 8);
        return bits;
      } catch (e) {
        throw new SerializeError(e);
      }
    }
    async _importRawKey(key, isPublic) {
      if (isPublic && key.byteLength !== this._nPk) {
        throw new Error("Invalid public key for the ciphersuite");
      }
      if (!isPublic && key.byteLength !== this._nSk) {
        throw new Error("Invalid private key for the ciphersuite");
      }
      if (isPublic) {
        return await this._api.importKey("raw", key, this._alg, true, []);
      }
      return await this._deserializePkcs8Key(new Uint8Array(key));
    }
    async _importJWK(key, isPublic) {
      if (typeof key.crv === "undefined" || key.crv !== this._alg.namedCurve) {
        throw new Error(`Invalid crv: ${key.crv}`);
      }
      if (isPublic) {
        if (typeof key.d !== "undefined") {
          throw new Error("Invalid key: `d` should not be set");
        }
        return await this._api.importKey("jwk", key, this._alg, true, []);
      }
      if (typeof key.d === "undefined") {
        throw new Error("Invalid key: `d` not found");
      }
      return await this._api.importKey("jwk", key, this._alg, true, KEM_USAGES);
    }
    async _deserializePkcs8Key(k) {
      const pkcs8Key = new Uint8Array(this._pkcs8AlgId.length + k.length);
      pkcs8Key.set(this._pkcs8AlgId, 0);
      pkcs8Key.set(k, this._pkcs8AlgId.length);
      return await this._api.importKey("pkcs8", pkcs8Key, this._alg, true, KEM_USAGES);
    }
  };

  // node_modules/@hpke/common/esm/src/kdfs/hkdf.js
  var HPKE_VERSION = new Uint8Array([72, 80, 75, 69, 45, 118, 49]);
  var HkdfNative = class extends NativeAlgorithm {
    constructor() {
      super();
      Object.defineProperty(this, "id", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: KdfId.HkdfSha256
      });
      Object.defineProperty(this, "hashSize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 0
      });
      Object.defineProperty(this, "_suiteId", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: EMPTY
      });
      Object.defineProperty(this, "algHash", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: {
          name: "HMAC",
          hash: "SHA-256",
          length: 256
        }
      });
    }
    init(suiteId) {
      this._suiteId = suiteId;
    }
    buildLabeledIkm(label, ikm) {
      this._checkInit();
      const ret = new Uint8Array(7 + this._suiteId.byteLength + label.byteLength + ikm.byteLength);
      ret.set(HPKE_VERSION, 0);
      ret.set(this._suiteId, 7);
      ret.set(label, 7 + this._suiteId.byteLength);
      ret.set(ikm, 7 + this._suiteId.byteLength + label.byteLength);
      return ret;
    }
    buildLabeledInfo(label, info, len) {
      this._checkInit();
      const ret = new Uint8Array(9 + this._suiteId.byteLength + label.byteLength + info.byteLength);
      ret.set(new Uint8Array([0, len]), 0);
      ret.set(HPKE_VERSION, 2);
      ret.set(this._suiteId, 9);
      ret.set(label, 9 + this._suiteId.byteLength);
      ret.set(info, 9 + this._suiteId.byteLength + label.byteLength);
      return ret;
    }
    async extract(salt, ikm) {
      await this._setup();
      if (salt.byteLength === 0) {
        salt = new ArrayBuffer(this.hashSize);
      }
      if (salt.byteLength !== this.hashSize) {
        throw new InvalidParamError("The salt length must be the same as the hashSize");
      }
      const key = await this._api.importKey("raw", salt, this.algHash, false, [
        "sign"
      ]);
      return await this._api.sign("HMAC", key, ikm);
    }
    async expand(prk, info, len) {
      await this._setup();
      const key = await this._api.importKey("raw", prk, this.algHash, false, [
        "sign"
      ]);
      const okm = new ArrayBuffer(len);
      const p = new Uint8Array(okm);
      let prev = EMPTY;
      const mid = new Uint8Array(info);
      const tail = new Uint8Array(1);
      if (len > 255 * this.hashSize) {
        throw new Error("Entropy limit reached");
      }
      const tmp = new Uint8Array(this.hashSize + mid.length + 1);
      for (let i = 1, cur = 0; cur < p.length; i++) {
        tail[0] = i;
        tmp.set(prev, 0);
        tmp.set(mid, prev.length);
        tmp.set(tail, prev.length + mid.length);
        prev = new Uint8Array(await this._api.sign("HMAC", key, tmp.slice(0, prev.length + mid.length + 1)));
        if (p.length - cur >= prev.length) {
          p.set(prev, cur);
          cur += prev.length;
        } else {
          p.set(prev.slice(0, p.length - cur), cur);
          cur += p.length - cur;
        }
      }
      return okm;
    }
    async extractAndExpand(salt, ikm, info, len) {
      await this._setup();
      const baseKey = await this._api.importKey("raw", ikm, "HKDF", false, ["deriveBits"]);
      return await this._api.deriveBits({
        name: "HKDF",
        hash: this.algHash.hash,
        salt,
        info
      }, baseKey, len * 8);
    }
    async labeledExtract(salt, label, ikm) {
      return await this.extract(salt, this.buildLabeledIkm(label, ikm).buffer);
    }
    async labeledExpand(prk, label, info, len) {
      return await this.expand(prk, this.buildLabeledInfo(label, info, len).buffer, len);
    }
    _checkInit() {
      if (this._suiteId === EMPTY) {
        throw new Error("Not initialized. Call init()");
      }
    }
  };
  var HkdfSha256Native = class extends HkdfNative {
    constructor() {
      super(...arguments);
      Object.defineProperty(this, "id", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: KdfId.HkdfSha256
      });
      Object.defineProperty(this, "hashSize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 32
      });
      Object.defineProperty(this, "algHash", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: {
          name: "HMAC",
          hash: "SHA-256",
          length: 256
        }
      });
    }
  };
  var HkdfSha384Native = class extends HkdfNative {
    constructor() {
      super(...arguments);
      Object.defineProperty(this, "id", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: KdfId.HkdfSha384
      });
      Object.defineProperty(this, "hashSize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 48
      });
      Object.defineProperty(this, "algHash", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: {
          name: "HMAC",
          hash: "SHA-384",
          length: 384
        }
      });
    }
  };
  var HkdfSha512Native = class extends HkdfNative {
    constructor() {
      super(...arguments);
      Object.defineProperty(this, "id", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: KdfId.HkdfSha512
      });
      Object.defineProperty(this, "hashSize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 64
      });
      Object.defineProperty(this, "algHash", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: {
          name: "HMAC",
          hash: "SHA-512",
          length: 512
        }
      });
    }
  };

  // node_modules/@hpke/common/esm/src/interfaces/aeadEncryptionContext.js
  var AEAD_USAGES = ["encrypt", "decrypt"];

  // node_modules/@hpke/common/esm/src/utils/noble.js
  function isBytes(a) {
    return a instanceof Uint8Array || ArrayBuffer.isView(a) && a.constructor.name === "Uint8Array";
  }
  function abytes(value, length, title = "") {
    const bytes = isBytes(value);
    const len = value?.length;
    const needsLen = length !== void 0;
    if (!bytes || needsLen && len !== length) {
      const prefix = title && `"${title}" `;
      const ofLen = needsLen ? ` of length ${length}` : "";
      const got = bytes ? `length=${len}` : `type=${typeof value}`;
      throw new Error(prefix + "expected Uint8Array" + ofLen + ", got " + got);
    }
    return value;
  }
  function ahash(h) {
    if (typeof h !== "function" || typeof h.create !== "function") {
      throw new Error("Hash must wrapped by utils.createHasher");
    }
    anumber(h.outputLen);
    anumber(h.blockLen);
  }
  function aexists(instance, checkFinished = true) {
    if (instance.destroyed)
      throw new Error("Hash instance has been destroyed");
    if (checkFinished && instance.finished) {
      throw new Error("Hash#digest() has already been called");
    }
  }
  function anumber(n) {
    if (!Number.isSafeInteger(n) || n < 0) {
      throw new Error("positive integer expected, got " + n);
    }
  }
  function clean(...arrays) {
    for (let i = 0; i < arrays.length; i++) {
      arrays[i].fill(0);
    }
  }

  // node_modules/@hpke/common/esm/src/hash/hmac.js
  var _HMAC = class {
    constructor(hash, key) {
      Object.defineProperty(this, "oHash", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "iHash", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "blockLen", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "outputLen", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "finished", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: false
      });
      Object.defineProperty(this, "destroyed", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: false
      });
      ahash(hash);
      abytes(key, void 0, "key");
      this.iHash = hash.create();
      if (typeof this.iHash.update !== "function") {
        throw new Error("Expected instance of class which extends utils.Hash");
      }
      this.blockLen = this.iHash.blockLen;
      this.outputLen = this.iHash.outputLen;
      const blockLen = this.blockLen;
      const pad = new Uint8Array(blockLen);
      pad.set(key.length > blockLen ? hash.create().update(key).digest() : key);
      for (let i = 0; i < pad.length; i++)
        pad[i] ^= 54;
      this.iHash.update(pad);
      this.oHash = hash.create();
      for (let i = 0; i < pad.length; i++)
        pad[i] ^= 54 ^ 92;
      this.oHash.update(pad);
      clean(pad);
    }
    update(buf) {
      aexists(this);
      this.iHash.update(buf);
      return this;
    }
    digestInto(out) {
      aexists(this);
      abytes(out, this.outputLen, "output");
      this.finished = true;
      this.iHash.digestInto(out);
      this.oHash.update(out);
      this.oHash.digestInto(out);
      this.destroy();
    }
    digest() {
      const out = new Uint8Array(this.oHash.outputLen);
      this.digestInto(out);
      return out;
    }
    _cloneInto(to) {
      to ||= Object.create(Object.getPrototypeOf(this), {});
      const { oHash, iHash, finished, destroyed, blockLen, outputLen } = this;
      to = to;
      to.finished = finished;
      to.destroyed = destroyed;
      to.blockLen = blockLen;
      to.outputLen = outputLen;
      to.oHash = oHash._cloneInto(to.oHash);
      to.iHash = iHash._cloneInto(to.iHash);
      return to;
    }
    clone() {
      return this._cloneInto();
    }
    destroy() {
      this.destroyed = true;
      this.oHash.destroy();
      this.iHash.destroy();
    }
  };
  var hmac = (hash, key, message) => new _HMAC(hash, key).update(message).digest();
  hmac.create = (hash, key) => new _HMAC(hash, key);

  // node_modules/@hpke/common/esm/src/hash/u64.js
  var U32_MASK64 = /* @__PURE__ */ BigInt(2 ** 32 - 1);

  // node_modules/@hpke/common/esm/src/curve/montgomery.js
  var _0n = BigInt(0);
  var _1n = BigInt(1);
  var _2n = BigInt(2);

  // node_modules/@hpke/core/esm/src/aeads/aesGcm.js
  var AesGcmContext = class extends NativeAlgorithm {
    constructor(key) {
      super();
      Object.defineProperty(this, "_rawKey", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_key", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      this._rawKey = key;
    }
    async seal(iv, data, aad) {
      await this._setupKey();
      const alg = {
        name: "AES-GCM",
        iv,
        additionalData: aad
      };
      const ct = await this._api.encrypt(alg, this._key, data);
      return ct;
    }
    async open(iv, data, aad) {
      await this._setupKey();
      const alg = {
        name: "AES-GCM",
        iv,
        additionalData: aad
      };
      const pt = await this._api.decrypt(alg, this._key, data);
      return pt;
    }
    async _setupKey() {
      if (this._key !== void 0) {
        return;
      }
      await this._setup();
      const key = await this._importKey(this._rawKey);
      new Uint8Array(this._rawKey).fill(0);
      this._key = key;
      return;
    }
    async _importKey(key) {
      return await this._api.importKey("raw", key, { name: "AES-GCM" }, true, AEAD_USAGES);
    }
  };
  var Aes128Gcm = class {
    constructor() {
      Object.defineProperty(this, "id", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: AeadId.Aes128Gcm
      });
      Object.defineProperty(this, "keySize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 16
      });
      Object.defineProperty(this, "nonceSize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 12
      });
      Object.defineProperty(this, "tagSize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 16
      });
    }
    createEncryptionContext(key) {
      return new AesGcmContext(key);
    }
  };
  var Aes256Gcm = class extends Aes128Gcm {
    constructor() {
      super(...arguments);
      Object.defineProperty(this, "id", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: AeadId.Aes256Gcm
      });
      Object.defineProperty(this, "keySize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 32
      });
      Object.defineProperty(this, "nonceSize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 12
      });
      Object.defineProperty(this, "tagSize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 16
      });
    }
  };

  // node_modules/@hpke/core/esm/src/utils/emitNotSupported.js
  function emitNotSupported() {
    return new Promise((_resolve, reject) => {
      reject(new NotSupportedError("Not supported"));
    });
  }

  // node_modules/@hpke/core/esm/src/exporterContext.js
  var LABEL_SEC = new Uint8Array([115, 101, 99]);
  var ExporterContextImpl = class {
    constructor(api, kdf, exporterSecret) {
      Object.defineProperty(this, "_api", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "exporterSecret", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_kdf", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      this._api = api;
      this._kdf = kdf;
      this.exporterSecret = exporterSecret;
    }
    async seal(_data, _aad) {
      return await emitNotSupported();
    }
    async open(_data, _aad) {
      return await emitNotSupported();
    }
    async export(exporterContext, len) {
      if (exporterContext.byteLength > INPUT_LENGTH_LIMIT) {
        throw new InvalidParamError("Too long exporter context");
      }
      try {
        return await this._kdf.labeledExpand(this.exporterSecret, LABEL_SEC, new Uint8Array(exporterContext), len);
      } catch (e) {
        throw new ExportError(e);
      }
    }
  };
  var RecipientExporterContextImpl = class extends ExporterContextImpl {
  };
  var SenderExporterContextImpl = class extends ExporterContextImpl {
    constructor(api, kdf, exporterSecret, enc) {
      super(api, kdf, exporterSecret);
      Object.defineProperty(this, "enc", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      this.enc = enc;
      return;
    }
  };

  // node_modules/@hpke/core/esm/src/encryptionContext.js
  var EncryptionContextImpl = class extends ExporterContextImpl {
    constructor(api, kdf, params) {
      super(api, kdf, params.exporterSecret);
      Object.defineProperty(this, "_aead", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_nK", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_nN", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_nT", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_ctx", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      if (params.key === void 0 || params.baseNonce === void 0 || params.seq === void 0) {
        throw new Error("Required parameters are missing");
      }
      this._aead = params.aead;
      this._nK = this._aead.keySize;
      this._nN = this._aead.nonceSize;
      this._nT = this._aead.tagSize;
      const key = this._aead.createEncryptionContext(params.key);
      this._ctx = {
        key,
        baseNonce: params.baseNonce,
        seq: params.seq
      };
    }
    computeNonce(k) {
      const seqBytes = i2Osp(k.seq, k.baseNonce.byteLength);
      return xor(k.baseNonce, seqBytes).buffer;
    }
    incrementSeq(k) {
      if (k.seq > Number.MAX_SAFE_INTEGER) {
        throw new MessageLimitReachedError("Message limit reached");
      }
      k.seq += 1;
      return;
    }
  };

  // node_modules/@hpke/core/esm/src/mutex.js
  var __classPrivateFieldGet = function(receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
  };
  var __classPrivateFieldSet = function(receiver, state, value, kind, f) {
    if (kind === "m") throw new TypeError("Private method is not writable");
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
    return kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value), value;
  };
  var _Mutex_locked;
  var Mutex = class {
    constructor() {
      _Mutex_locked.set(this, Promise.resolve());
    }
    async lock() {
      let releaseLock;
      const nextLock = new Promise((resolve) => {
        releaseLock = resolve;
      });
      const previousLock = __classPrivateFieldGet(this, _Mutex_locked, "f");
      __classPrivateFieldSet(this, _Mutex_locked, nextLock, "f");
      await previousLock;
      return releaseLock;
    }
  };
  _Mutex_locked = /* @__PURE__ */ new WeakMap();

  // node_modules/@hpke/core/esm/src/recipientContext.js
  var __classPrivateFieldGet2 = function(receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
  };
  var __classPrivateFieldSet2 = function(receiver, state, value, kind, f) {
    if (kind === "m") throw new TypeError("Private method is not writable");
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
    return kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value), value;
  };
  var _RecipientContextImpl_mutex;
  var RecipientContextImpl = class extends EncryptionContextImpl {
    constructor() {
      super(...arguments);
      _RecipientContextImpl_mutex.set(this, void 0);
    }
    async open(data, aad = EMPTY.buffer) {
      __classPrivateFieldSet2(this, _RecipientContextImpl_mutex, __classPrivateFieldGet2(this, _RecipientContextImpl_mutex, "f") ?? new Mutex(), "f");
      const release = await __classPrivateFieldGet2(this, _RecipientContextImpl_mutex, "f").lock();
      let pt;
      try {
        pt = await this._ctx.key.open(this.computeNonce(this._ctx), data, aad);
      } catch (e) {
        throw new OpenError(e);
      } finally {
        release();
      }
      this.incrementSeq(this._ctx);
      return pt;
    }
  };
  _RecipientContextImpl_mutex = /* @__PURE__ */ new WeakMap();

  // node_modules/@hpke/core/esm/src/senderContext.js
  var __classPrivateFieldGet3 = function(receiver, state, kind, f) {
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a getter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot read private member from an object whose class did not declare it");
    return kind === "m" ? f : kind === "a" ? f.call(receiver) : f ? f.value : state.get(receiver);
  };
  var __classPrivateFieldSet3 = function(receiver, state, value, kind, f) {
    if (kind === "m") throw new TypeError("Private method is not writable");
    if (kind === "a" && !f) throw new TypeError("Private accessor was defined without a setter");
    if (typeof state === "function" ? receiver !== state || !f : !state.has(receiver)) throw new TypeError("Cannot write private member to an object whose class did not declare it");
    return kind === "a" ? f.call(receiver, value) : f ? f.value = value : state.set(receiver, value), value;
  };
  var _SenderContextImpl_mutex;
  var SenderContextImpl = class extends EncryptionContextImpl {
    constructor(api, kdf, params, enc) {
      super(api, kdf, params);
      Object.defineProperty(this, "enc", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      _SenderContextImpl_mutex.set(this, void 0);
      this.enc = enc;
    }
    async seal(data, aad = EMPTY.buffer) {
      __classPrivateFieldSet3(this, _SenderContextImpl_mutex, __classPrivateFieldGet3(this, _SenderContextImpl_mutex, "f") ?? new Mutex(), "f");
      const release = await __classPrivateFieldGet3(this, _SenderContextImpl_mutex, "f").lock();
      let ct;
      try {
        ct = await this._ctx.key.seal(this.computeNonce(this._ctx), data, aad);
      } catch (e) {
        throw new SealError(e);
      } finally {
        release();
      }
      this.incrementSeq(this._ctx);
      return ct;
    }
  };
  _SenderContextImpl_mutex = /* @__PURE__ */ new WeakMap();

  // node_modules/@hpke/core/esm/src/cipherSuiteNative.js
  var LABEL_BASE_NONCE = new Uint8Array([
    98,
    97,
    115,
    101,
    95,
    110,
    111,
    110,
    99,
    101
  ]);
  var LABEL_EXP = new Uint8Array([101, 120, 112]);
  var LABEL_INFO_HASH = new Uint8Array([
    105,
    110,
    102,
    111,
    95,
    104,
    97,
    115,
    104
  ]);
  var LABEL_KEY = new Uint8Array([107, 101, 121]);
  var LABEL_PSK_ID_HASH = new Uint8Array([
    112,
    115,
    107,
    95,
    105,
    100,
    95,
    104,
    97,
    115,
    104
  ]);
  var LABEL_SECRET = new Uint8Array([115, 101, 99, 114, 101, 116]);
  var SUITE_ID_HEADER_HPKE = new Uint8Array([
    72,
    80,
    75,
    69,
    0,
    0,
    0,
    0,
    0,
    0
  ]);
  var CipherSuiteNative = class extends NativeAlgorithm {
    /**
     * @param params A set of parameters for building a cipher suite.
     *
     * If the error occurred, throws {@link InvalidParamError}.
     *
     * @throws {@link InvalidParamError}
     */
    constructor(params) {
      super();
      Object.defineProperty(this, "_kem", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_kdf", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_aead", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_suiteId", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      if (typeof params.kem === "number") {
        throw new InvalidParamError("KemId cannot be used");
      }
      this._kem = params.kem;
      if (typeof params.kdf === "number") {
        throw new InvalidParamError("KdfId cannot be used");
      }
      this._kdf = params.kdf;
      if (typeof params.aead === "number") {
        throw new InvalidParamError("AeadId cannot be used");
      }
      this._aead = params.aead;
      this._suiteId = new Uint8Array(SUITE_ID_HEADER_HPKE);
      this._suiteId.set(i2Osp(this._kem.id, 2), 4);
      this._suiteId.set(i2Osp(this._kdf.id, 2), 6);
      this._suiteId.set(i2Osp(this._aead.id, 2), 8);
      this._kdf.init(this._suiteId);
    }
    /**
     * Gets the KEM context of the ciphersuite.
     */
    get kem() {
      return this._kem;
    }
    /**
     * Gets the KDF context of the ciphersuite.
     */
    get kdf() {
      return this._kdf;
    }
    /**
     * Gets the AEAD context of the ciphersuite.
     */
    get aead() {
      return this._aead;
    }
    /**
     * Creates an encryption context for a sender.
     *
     * If the error occurred, throws {@link DecapError} | {@link ValidationError}.
     *
     * @param params A set of parameters for the sender encryption context.
     * @returns A sender encryption context.
     * @throws {@link EncapError}, {@link ValidationError}
     */
    async createSenderContext(params) {
      this._validateInputLength(params);
      await this._setup();
      const dh = await this._kem.encap(params);
      let mode;
      if (params.psk !== void 0) {
        mode = params.senderKey !== void 0 ? Mode.AuthPsk : Mode.Psk;
      } else {
        mode = params.senderKey !== void 0 ? Mode.Auth : Mode.Base;
      }
      return await this._keyScheduleS(mode, dh.sharedSecret, dh.enc, params);
    }
    /**
     * Creates an encryption context for a recipient.
     *
     * If the error occurred, throws {@link DecapError}
     * | {@link DeserializeError} | {@link ValidationError}.
     *
     * @param params A set of parameters for the recipient encryption context.
     * @returns A recipient encryption context.
     * @throws {@link DecapError}, {@link DeserializeError}, {@link ValidationError}
     */
    async createRecipientContext(params) {
      this._validateInputLength(params);
      await this._setup();
      const sharedSecret = await this._kem.decap(params);
      let mode;
      if (params.psk !== void 0) {
        mode = params.senderPublicKey !== void 0 ? Mode.AuthPsk : Mode.Psk;
      } else {
        mode = params.senderPublicKey !== void 0 ? Mode.Auth : Mode.Base;
      }
      return await this._keyScheduleR(mode, sharedSecret, params);
    }
    /**
     * Encrypts a message to a recipient.
     *
     * If the error occurred, throws `EncapError` | `MessageLimitReachedError` | `SealError` | `ValidationError`.
     *
     * @param params A set of parameters for building a sender encryption context.
     * @param pt A plain text as bytes to be encrypted.
     * @param aad Additional authenticated data as bytes fed by an application.
     * @returns A cipher text and an encapsulated key as bytes.
     * @throws {@link EncapError}, {@link MessageLimitReachedError}, {@link SealError}, {@link ValidationError}
     */
    async seal(params, pt, aad = EMPTY.buffer) {
      const ctx = await this.createSenderContext(params);
      return {
        ct: await ctx.seal(pt, aad),
        enc: ctx.enc
      };
    }
    /**
     * Decrypts a message from a sender.
     *
     * If the error occurred, throws `DecapError` | `DeserializeError` | `OpenError` | `ValidationError`.
     *
     * @param params A set of parameters for building a recipient encryption context.
     * @param ct An encrypted text as bytes to be decrypted.
     * @param aad Additional authenticated data as bytes fed by an application.
     * @returns A decrypted plain text as bytes.
     * @throws {@link DecapError}, {@link DeserializeError}, {@link OpenError}, {@link ValidationError}
     */
    async open(params, ct, aad = EMPTY.buffer) {
      const ctx = await this.createRecipientContext(params);
      return await ctx.open(ct, aad);
    }
    // private verifyPskInputs(mode: Mode, params: KeyScheduleParams) {
    //   const gotPsk = (params.psk !== undefined);
    //   const gotPskId = (params.psk !== undefined && params.psk.id.byteLength > 0);
    //   if (gotPsk !== gotPskId) {
    //     throw new Error('Inconsistent PSK inputs');
    //   }
    //   if (gotPsk && (mode === Mode.Base || mode === Mode.Auth)) {
    //     throw new Error('PSK input provided when not needed');
    //   }
    //   if (!gotPsk && (mode === Mode.Psk || mode === Mode.AuthPsk)) {
    //     throw new Error('Missing required PSK input');
    //   }
    //   return;
    // }
    async _keySchedule(mode, sharedSecret, params) {
      const pskId = params.psk === void 0 ? EMPTY : new Uint8Array(params.psk.id);
      const pskIdHash = await this._kdf.labeledExtract(EMPTY.buffer, LABEL_PSK_ID_HASH, pskId);
      const info = params.info === void 0 ? EMPTY : new Uint8Array(params.info);
      const infoHash = await this._kdf.labeledExtract(EMPTY.buffer, LABEL_INFO_HASH, info);
      const keyScheduleContext = new Uint8Array(1 + pskIdHash.byteLength + infoHash.byteLength);
      keyScheduleContext.set(new Uint8Array([mode]), 0);
      keyScheduleContext.set(new Uint8Array(pskIdHash), 1);
      keyScheduleContext.set(new Uint8Array(infoHash), 1 + pskIdHash.byteLength);
      const psk = params.psk === void 0 ? EMPTY : new Uint8Array(params.psk.key);
      const ikm = this._kdf.buildLabeledIkm(LABEL_SECRET, psk).buffer;
      const exporterSecretInfo = this._kdf.buildLabeledInfo(LABEL_EXP, keyScheduleContext, this._kdf.hashSize).buffer;
      const exporterSecret = await this._kdf.extractAndExpand(sharedSecret, ikm, exporterSecretInfo, this._kdf.hashSize);
      if (this._aead.id === AeadId.ExportOnly) {
        return { aead: this._aead, exporterSecret };
      }
      const keyInfo = this._kdf.buildLabeledInfo(LABEL_KEY, keyScheduleContext, this._aead.keySize).buffer;
      const key = await this._kdf.extractAndExpand(sharedSecret, ikm, keyInfo, this._aead.keySize);
      const baseNonceInfo = this._kdf.buildLabeledInfo(LABEL_BASE_NONCE, keyScheduleContext, this._aead.nonceSize).buffer;
      const baseNonce = await this._kdf.extractAndExpand(sharedSecret, ikm, baseNonceInfo, this._aead.nonceSize);
      return {
        aead: this._aead,
        exporterSecret,
        key,
        baseNonce: new Uint8Array(baseNonce),
        seq: 0
      };
    }
    async _keyScheduleS(mode, sharedSecret, enc, params) {
      const res = await this._keySchedule(mode, sharedSecret, params);
      if (res.key === void 0) {
        return new SenderExporterContextImpl(this._api, this._kdf, res.exporterSecret, enc);
      }
      return new SenderContextImpl(this._api, this._kdf, res, enc);
    }
    async _keyScheduleR(mode, sharedSecret, params) {
      const res = await this._keySchedule(mode, sharedSecret, params);
      if (res.key === void 0) {
        return new RecipientExporterContextImpl(this._api, this._kdf, res.exporterSecret);
      }
      return new RecipientContextImpl(this._api, this._kdf, res);
    }
    _validateInputLength(params) {
      if (params.info !== void 0 && params.info.byteLength > INFO_LENGTH_LIMIT) {
        throw new InvalidParamError("Too long info");
      }
      if (params.psk !== void 0) {
        if (params.psk.key.byteLength < MINIMUM_PSK_LENGTH) {
          throw new InvalidParamError(`PSK must have at least ${MINIMUM_PSK_LENGTH} bytes`);
        }
        if (params.psk.key.byteLength > INPUT_LENGTH_LIMIT) {
          throw new InvalidParamError("Too long psk.key");
        }
        if (params.psk.id.byteLength > INPUT_LENGTH_LIMIT) {
          throw new InvalidParamError("Too long psk.id");
        }
      }
      return;
    }
  };

  // node_modules/@hpke/core/esm/src/kems/dhkemNative.js
  var DhkemP256HkdfSha256Native = class extends Dhkem {
    constructor() {
      const kdf = new HkdfSha256Native();
      const prim = new Ec(KemId.DhkemP256HkdfSha256, kdf);
      super(KemId.DhkemP256HkdfSha256, prim, kdf);
      Object.defineProperty(this, "id", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: KemId.DhkemP256HkdfSha256
      });
      Object.defineProperty(this, "secretSize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 32
      });
      Object.defineProperty(this, "encSize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 65
      });
      Object.defineProperty(this, "publicKeySize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 65
      });
      Object.defineProperty(this, "privateKeySize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 32
      });
    }
  };
  var DhkemP384HkdfSha384Native = class extends Dhkem {
    constructor() {
      const kdf = new HkdfSha384Native();
      const prim = new Ec(KemId.DhkemP384HkdfSha384, kdf);
      super(KemId.DhkemP384HkdfSha384, prim, kdf);
      Object.defineProperty(this, "id", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: KemId.DhkemP384HkdfSha384
      });
      Object.defineProperty(this, "secretSize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 48
      });
      Object.defineProperty(this, "encSize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 97
      });
      Object.defineProperty(this, "publicKeySize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 97
      });
      Object.defineProperty(this, "privateKeySize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 48
      });
    }
  };
  var DhkemP521HkdfSha512Native = class extends Dhkem {
    constructor() {
      const kdf = new HkdfSha512Native();
      const prim = new Ec(KemId.DhkemP521HkdfSha512, kdf);
      super(KemId.DhkemP521HkdfSha512, prim, kdf);
      Object.defineProperty(this, "id", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: KemId.DhkemP521HkdfSha512
      });
      Object.defineProperty(this, "secretSize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 64
      });
      Object.defineProperty(this, "encSize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 133
      });
      Object.defineProperty(this, "publicKeySize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 133
      });
      Object.defineProperty(this, "privateKeySize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 64
      });
    }
  };

  // node_modules/@hpke/core/esm/src/native.js
  var CipherSuite = class extends CipherSuiteNative {
  };
  var DhkemP256HkdfSha256 = class extends DhkemP256HkdfSha256Native {
  };
  var DhkemP384HkdfSha384 = class extends DhkemP384HkdfSha384Native {
  };
  var DhkemP521HkdfSha512 = class extends DhkemP521HkdfSha512Native {
  };
  var HkdfSha256 = class extends HkdfSha256Native {
  };
  var HkdfSha384 = class extends HkdfSha384Native {
  };
  var HkdfSha512 = class extends HkdfSha512Native {
  };

  // node_modules/@hpke/core/esm/src/kems/dhkemPrimitives/x25519.js
  var ALG_NAME = "X25519";
  var PKCS8_ALG_ID_X25519 = new Uint8Array([
    48,
    46,
    2,
    1,
    0,
    48,
    5,
    6,
    3,
    43,
    101,
    110,
    4,
    34,
    4,
    32
  ]);
  var X25519 = class extends NativeAlgorithm {
    constructor(hkdf) {
      super();
      Object.defineProperty(this, "_hkdf", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_alg", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_nPk", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_nSk", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_nDh", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_pkcs8AlgId", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      this._alg = { name: ALG_NAME };
      this._hkdf = hkdf;
      this._nPk = 32;
      this._nSk = 32;
      this._nDh = 32;
      this._pkcs8AlgId = PKCS8_ALG_ID_X25519;
    }
    async serializePublicKey(key) {
      await this._setup();
      try {
        return await this._api.exportKey("raw", key);
      } catch (e) {
        throw new SerializeError(e);
      }
    }
    async deserializePublicKey(key) {
      await this._setup();
      try {
        return await this._importRawKey(key, true);
      } catch (e) {
        throw new DeserializeError(e);
      }
    }
    async serializePrivateKey(key) {
      await this._setup();
      try {
        const jwk = await this._api.exportKey("jwk", key);
        if (!("d" in jwk)) {
          throw new Error("Not private key");
        }
        return base64UrlToBytes(jwk["d"]).buffer;
      } catch (e) {
        throw new SerializeError(e);
      }
    }
    async deserializePrivateKey(key) {
      await this._setup();
      try {
        return await this._importRawKey(key, false);
      } catch (e) {
        throw new DeserializeError(e);
      }
    }
    async importKey(format, key, isPublic) {
      await this._setup();
      try {
        if (format === "raw") {
          return await this._importRawKey(key, isPublic);
        }
        if (key instanceof ArrayBuffer) {
          throw new Error("Invalid jwk key format");
        }
        return await this._importJWK(key, isPublic);
      } catch (e) {
        throw new DeserializeError(e);
      }
    }
    async generateKeyPair() {
      await this._setup();
      try {
        return await this._api.generateKey(ALG_NAME, true, KEM_USAGES);
      } catch (e) {
        throw new NotSupportedError(e);
      }
    }
    async deriveKeyPair(ikm) {
      await this._setup();
      try {
        const dkpPrk = await this._hkdf.labeledExtract(EMPTY.buffer, LABEL_DKP_PRK, new Uint8Array(ikm));
        const rawSk = await this._hkdf.labeledExpand(dkpPrk, LABEL_SK, EMPTY, this._nSk);
        const rawSkBytes = new Uint8Array(rawSk);
        const sk = await this._deserializePkcs8Key(rawSkBytes);
        rawSkBytes.fill(0);
        return {
          privateKey: sk,
          publicKey: await this.derivePublicKey(sk)
        };
      } catch (e) {
        throw new DeriveKeyPairError(e);
      }
    }
    async derivePublicKey(key) {
      await this._setup();
      try {
        const jwk = await this._api.exportKey("jwk", key);
        delete jwk["d"];
        delete jwk["key_ops"];
        return await this._api.importKey("jwk", jwk, this._alg, true, []);
      } catch (e) {
        throw new DeserializeError(e);
      }
    }
    async dh(sk, pk) {
      await this._setup();
      try {
        const bits = await this._api.deriveBits({
          name: ALG_NAME,
          public: pk
        }, sk, this._nDh * 8);
        return bits;
      } catch (e) {
        throw new SerializeError(e);
      }
    }
    async _importRawKey(key, isPublic) {
      if (isPublic && key.byteLength !== this._nPk) {
        throw new Error("Invalid public key for the ciphersuite");
      }
      if (!isPublic && key.byteLength !== this._nSk) {
        throw new Error("Invalid private key for the ciphersuite");
      }
      if (isPublic) {
        return await this._api.importKey("raw", key, this._alg, true, []);
      }
      return await this._deserializePkcs8Key(new Uint8Array(key));
    }
    async _importJWK(key, isPublic) {
      if (typeof key.kty === "undefined" || key.kty !== "OKP") {
        throw new Error(`Invalid kty: ${key.crv}`);
      }
      if (typeof key.crv === "undefined" || key.crv !== ALG_NAME) {
        throw new Error(`Invalid crv: ${key.crv}`);
      }
      if (isPublic) {
        if (typeof key.d !== "undefined") {
          throw new Error("Invalid key: `d` should not be set");
        }
        return await this._api.importKey("jwk", key, this._alg, true, []);
      }
      if (typeof key.d === "undefined") {
        throw new Error("Invalid key: `d` not found");
      }
      return await this._api.importKey("jwk", key, this._alg, true, KEM_USAGES);
    }
    async _deserializePkcs8Key(k) {
      const pkcs8Key = new Uint8Array(this._pkcs8AlgId.length + k.length);
      pkcs8Key.set(this._pkcs8AlgId, 0);
      pkcs8Key.set(k, this._pkcs8AlgId.length);
      return await this._api.importKey("pkcs8", pkcs8Key, this._alg, true, KEM_USAGES);
    }
  };

  // node_modules/@hpke/core/esm/src/kems/dhkemX25519.js
  var DhkemX25519HkdfSha256 = class extends Dhkem {
    constructor() {
      const kdf = new HkdfSha256Native();
      super(KemId.DhkemX25519HkdfSha256, new X25519(kdf), kdf);
      Object.defineProperty(this, "id", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: KemId.DhkemX25519HkdfSha256
      });
      Object.defineProperty(this, "secretSize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 32
      });
      Object.defineProperty(this, "encSize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 32
      });
      Object.defineProperty(this, "publicKeySize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 32
      });
      Object.defineProperty(this, "privateKeySize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 32
      });
    }
  };

  // node_modules/@hpke/core/esm/src/kems/dhkemPrimitives/x448.js
  var ALG_NAME2 = "X448";
  var PKCS8_ALG_ID_X448 = new Uint8Array([
    48,
    70,
    2,
    1,
    0,
    48,
    5,
    6,
    3,
    43,
    101,
    111,
    4,
    58,
    4,
    56
  ]);
  var X448 = class extends NativeAlgorithm {
    constructor(hkdf) {
      super();
      Object.defineProperty(this, "_hkdf", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_alg", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_nPk", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_nSk", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_nDh", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      Object.defineProperty(this, "_pkcs8AlgId", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: void 0
      });
      this._alg = { name: ALG_NAME2 };
      this._hkdf = hkdf;
      this._nPk = 56;
      this._nSk = 56;
      this._nDh = 56;
      this._pkcs8AlgId = PKCS8_ALG_ID_X448;
    }
    async serializePublicKey(key) {
      await this._setup();
      try {
        return await this._api.exportKey("raw", key);
      } catch (e) {
        throw new SerializeError(e);
      }
    }
    async deserializePublicKey(key) {
      await this._setup();
      try {
        return await this._importRawKey(key, true);
      } catch (e) {
        throw new DeserializeError(e);
      }
    }
    async serializePrivateKey(key) {
      await this._setup();
      try {
        const jwk = await this._api.exportKey("jwk", key);
        if (!("d" in jwk)) {
          throw new Error("Not private key");
        }
        return base64UrlToBytes(jwk["d"]).buffer;
      } catch (e) {
        throw new SerializeError(e);
      }
    }
    async deserializePrivateKey(key) {
      await this._setup();
      try {
        return await this._importRawKey(key, false);
      } catch (e) {
        throw new DeserializeError(e);
      }
    }
    async importKey(format, key, isPublic) {
      await this._setup();
      try {
        if (format === "raw") {
          return await this._importRawKey(key, isPublic);
        }
        if (key instanceof ArrayBuffer) {
          throw new Error("Invalid jwk key format");
        }
        return await this._importJWK(key, isPublic);
      } catch (e) {
        throw new DeserializeError(e);
      }
    }
    async generateKeyPair() {
      await this._setup();
      try {
        return await this._api.generateKey(ALG_NAME2, true, KEM_USAGES);
      } catch (e) {
        throw new NotSupportedError(e);
      }
    }
    async deriveKeyPair(ikm) {
      await this._setup();
      try {
        const dkpPrk = await this._hkdf.labeledExtract(EMPTY.buffer, LABEL_DKP_PRK, new Uint8Array(ikm));
        const rawSk = await this._hkdf.labeledExpand(dkpPrk, LABEL_SK, EMPTY, this._nSk);
        const rawSkBytes = new Uint8Array(rawSk);
        const sk = await this._deserializePkcs8Key(rawSkBytes);
        rawSkBytes.fill(0);
        return {
          privateKey: sk,
          publicKey: await this.derivePublicKey(sk)
        };
      } catch (e) {
        throw new DeriveKeyPairError(e);
      }
    }
    async derivePublicKey(key) {
      await this._setup();
      try {
        const jwk = await this._api.exportKey("jwk", key);
        delete jwk["d"];
        delete jwk["key_ops"];
        return await this._api.importKey("jwk", jwk, this._alg, true, []);
      } catch (e) {
        throw new DeserializeError(e);
      }
    }
    async dh(sk, pk) {
      await this._setup();
      try {
        const bits = await this._api.deriveBits({
          name: ALG_NAME2,
          public: pk
        }, sk, this._nDh * 8);
        return bits;
      } catch (e) {
        throw new SerializeError(e);
      }
    }
    async _importRawKey(key, isPublic) {
      if (isPublic && key.byteLength !== this._nPk) {
        throw new Error("Invalid public key for the ciphersuite");
      }
      if (!isPublic && key.byteLength !== this._nSk) {
        throw new Error("Invalid private key for the ciphersuite");
      }
      if (isPublic) {
        return await this._api.importKey("raw", key, this._alg, true, []);
      }
      return await this._deserializePkcs8Key(new Uint8Array(key));
    }
    async _importJWK(key, isPublic) {
      if (typeof key.kty === "undefined" || key.kty !== "OKP") {
        throw new Error(`Invalid kty: ${key.crv}`);
      }
      if (typeof key.crv === "undefined" || key.crv !== ALG_NAME2) {
        throw new Error(`Invalid crv: ${key.crv}`);
      }
      if (isPublic) {
        if (typeof key.d !== "undefined") {
          throw new Error("Invalid key: `d` should not be set");
        }
        return await this._api.importKey("jwk", key, this._alg, true, []);
      }
      if (typeof key.d === "undefined") {
        throw new Error("Invalid key: `d` not found");
      }
      return await this._api.importKey("jwk", key, this._alg, true, KEM_USAGES);
    }
    async _deserializePkcs8Key(k) {
      const pkcs8Key = new Uint8Array(this._pkcs8AlgId.length + k.length);
      pkcs8Key.set(this._pkcs8AlgId, 0);
      pkcs8Key.set(k, this._pkcs8AlgId.length);
      return await this._api.importKey("pkcs8", pkcs8Key, this._alg, true, KEM_USAGES);
    }
  };

  // node_modules/@hpke/core/esm/src/kems/dhkemX448.js
  var DhkemX448HkdfSha512 = class extends Dhkem {
    constructor() {
      const kdf = new HkdfSha512Native();
      super(KemId.DhkemX448HkdfSha512, new X448(kdf), kdf);
      Object.defineProperty(this, "id", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: KemId.DhkemX448HkdfSha512
      });
      Object.defineProperty(this, "secretSize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 64
      });
      Object.defineProperty(this, "encSize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 56
      });
      Object.defineProperty(this, "publicKeySize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 56
      });
      Object.defineProperty(this, "privateKeySize", {
        enumerable: true,
        configurable: true,
        writable: true,
        value: 56
      });
    }
  };

  // node_modules/ts-mls/dist/crypto/implementation/hpke.js
  async function makeGenericHpke(hpkealg, aead, cs) {
    return {
      async open(privateKey, kemOutput, ciphertext, info, aad) {
        try {
          const result = await cs.open({ recipientKey: privateKey, enc: bytesToArrayBuffer(kemOutput), info: bytesToArrayBuffer(info) }, bytesToArrayBuffer(ciphertext), aad ? bytesToArrayBuffer(aad) : new ArrayBuffer());
          return new Uint8Array(result);
        } catch (e) {
          throw new CryptoError(`${e}`);
        }
      },
      async seal(publicKey, plaintext, info, aad) {
        const result = await cs.seal({ recipientPublicKey: publicKey, info: bytesToArrayBuffer(info) }, bytesToArrayBuffer(plaintext), aad ? bytesToArrayBuffer(aad) : new ArrayBuffer());
        return {
          ct: new Uint8Array(result.ct),
          enc: new Uint8Array(result.enc)
        };
      },
      async exportSecret(publicKey, exporterContext, length, info) {
        const context = await cs.createSenderContext({ recipientPublicKey: publicKey, info: bytesToArrayBuffer(info) });
        return {
          enc: new Uint8Array(context.enc),
          secret: new Uint8Array(await context.export(bytesToArrayBuffer(exporterContext), length))
        };
      },
      async importSecret(privateKey, exporterContext, kemOutput, length, info) {
        try {
          const context = await cs.createRecipientContext({
            recipientKey: privateKey,
            info: bytesToArrayBuffer(info),
            enc: bytesToArrayBuffer(kemOutput)
          });
          return new Uint8Array(await context.export(bytesToArrayBuffer(exporterContext), length));
        } catch (e) {
          throw new CryptoError(`${e}`);
        }
      },
      async importPrivateKey(k) {
        try {
          const key = hpkealg.kem === "DHKEM-P521-HKDF-SHA512" ? prepadPrivateKeyP521(k) : k;
          return await cs.kem.deserializePrivateKey(bytesToArrayBuffer(key));
        } catch (e) {
          throw new CryptoError(`${e}`);
        }
      },
      async importPublicKey(k) {
        try {
          return await cs.kem.deserializePublicKey(bytesToArrayBuffer(k));
        } catch (e) {
          throw new CryptoError(`${e}`);
        }
      },
      async exportPublicKey(k) {
        return new Uint8Array(await cs.kem.serializePublicKey(k));
      },
      async exportPrivateKey(k) {
        return new Uint8Array(await cs.kem.serializePrivateKey(k));
      },
      async encryptAead(key, nonce, aad, plaintext) {
        return aead.encrypt(key, nonce, aad ? aad : new Uint8Array(), plaintext);
      },
      async decryptAead(key, nonce, aad, ciphertext) {
        try {
          return await aead.decrypt(key, nonce, aad ? aad : new Uint8Array(), ciphertext);
        } catch (e) {
          throw new CryptoError(`${e}`);
        }
      },
      async deriveKeyPair(ikm) {
        const kp = await cs.kem.deriveKeyPair(bytesToArrayBuffer(ikm));
        return { privateKey: kp.privateKey, publicKey: kp.publicKey };
      },
      async generateKeyPair() {
        const kp = await cs.kem.generateKeyPair();
        return { privateKey: kp.privateKey, publicKey: kp.publicKey };
      },
      keyLength: cs.aead.keySize,
      nonceLength: cs.aead.nonceSize
    };
  }
  function prepadPrivateKeyP521(k) {
    const lengthDifference = 66 - k.byteLength;
    return concatUint8Arrays(new Uint8Array(lengthDifference), k);
  }

  // node_modules/ts-mls/dist/crypto/implementation/default/makeAead.js
  async function makeAead(aeadAlg) {
    switch (aeadAlg) {
      case "AES128GCM":
        return [
          {
            encrypt(key, nonce, aad, plaintext) {
              return encryptAesGcm(key, nonce, aad, plaintext);
            },
            decrypt(key, nonce, aad, ciphertext) {
              return decryptAesGcm(key, nonce, aad, ciphertext);
            }
          },
          new Aes128Gcm()
        ];
      case "AES256GCM":
        return [
          {
            encrypt(key, nonce, aad, plaintext) {
              return encryptAesGcm(key, nonce, aad, plaintext);
            },
            decrypt(key, nonce, aad, ciphertext) {
              return decryptAesGcm(key, nonce, aad, ciphertext);
            }
          },
          new Aes256Gcm()
        ];
      case "CHACHA20POLY1305":
        try {
          const { Chacha20Poly1305 } = await import("@hpke/chacha20poly1305");
          const { chacha20poly1305: chacha20poly13052 } = await Promise.resolve().then(() => (init_chacha(), chacha_exports));
          return [
            {
              async encrypt(key, nonce, aad, plaintext) {
                return chacha20poly13052(key, nonce, aad).encrypt(plaintext);
              },
              async decrypt(key, nonce, aad, ciphertext) {
                return chacha20poly13052(key, nonce, aad).decrypt(ciphertext);
              }
            },
            new Chacha20Poly1305()
          ];
        } catch (err) {
          throw new DependencyError("Optional dependency '@hpke/chacha20poly1305' is not installed. Please install it to use this feature.");
        }
    }
  }
  async function encryptAesGcm(key, nonce, aad, plaintext) {
    const cryptoKey = await crypto.subtle.importKey("raw", toBufferSource(key), { name: "AES-GCM" }, false, ["encrypt"]);
    const result = await crypto.subtle.encrypt({
      name: "AES-GCM",
      iv: toBufferSource(nonce),
      additionalData: aad.length > 0 ? toBufferSource(aad) : void 0
    }, cryptoKey, toBufferSource(plaintext));
    return new Uint8Array(result);
  }
  async function decryptAesGcm(key, nonce, aad, ciphertext) {
    const cryptoKey = await crypto.subtle.importKey("raw", toBufferSource(key), { name: "AES-GCM" }, false, ["decrypt"]);
    const result = await crypto.subtle.decrypt({
      name: "AES-GCM",
      iv: toBufferSource(nonce),
      additionalData: aad.length > 0 ? toBufferSource(aad) : void 0
    }, cryptoKey, toBufferSource(ciphertext));
    return new Uint8Array(result);
  }

  // node_modules/ts-mls/dist/crypto/implementation/default/makeKdfImpl.js
  function makeKdfImpl(k) {
    return {
      async extract(salt, ikm) {
        const result = await k.extract(bytesToArrayBuffer(salt), bytesToArrayBuffer(ikm));
        return new Uint8Array(result);
      },
      async expand(prk, info, len) {
        const result = await k.expand(bytesToArrayBuffer(prk), bytesToArrayBuffer(info), len);
        return new Uint8Array(result);
      },
      size: k.hashSize
    };
  }
  function makeKdf(kdfAlg) {
    switch (kdfAlg) {
      case "HKDF-SHA256":
        return new HkdfSha256();
      case "HKDF-SHA384":
        return new HkdfSha384();
      case "HKDF-SHA512":
        return new HkdfSha512();
    }
  }

  // node_modules/ts-mls/dist/crypto/implementation/default/makeDhKem.js
  async function makeDhKem(kemAlg) {
    switch (kemAlg) {
      case "DHKEM-P256-HKDF-SHA256":
        return new DhkemP256HkdfSha256();
      case "DHKEM-X25519-HKDF-SHA256":
        return new DhkemX25519HkdfSha256();
      case "DHKEM-X448-HKDF-SHA512":
        return new DhkemX448HkdfSha512();
      case "DHKEM-P521-HKDF-SHA512":
        return new DhkemP521HkdfSha512();
      case "DHKEM-P384-HKDF-SHA384":
        return new DhkemP384HkdfSha384();
      case "ML-KEM-512":
        try {
          const { MlKem512 } = await import("@hpke/ml-kem");
          return new MlKem512();
        } catch (err) {
          throw new DependencyError("Optional dependency '@hpke/ml-kem' is not installed. Please install it to use this feature.");
        }
      case "ML-KEM-768":
        try {
          const { MlKem768 } = await import("@hpke/ml-kem");
          return new MlKem768();
        } catch (err) {
          throw new DependencyError("Optional dependency '@hpke/ml-kem' is not installed. Please install it to use this feature.");
        }
      case "ML-KEM-1024":
        try {
          const { MlKem1024 } = await import("@hpke/ml-kem");
          return new MlKem1024();
        } catch (err) {
          throw new DependencyError("Optional dependency '@hpke/ml-kem' is not installed. Please install it to use this feature.");
        }
      case "X-Wing":
        try {
          const { XWing } = await import("@hpke/hybridkem-x-wing");
          return new XWing();
        } catch (err) {
          throw new DependencyError("Optional dependency '@hpke/hybridkem-x-wing' is not installed. Please install it to use this feature.");
        }
    }
  }

  // node_modules/ts-mls/dist/crypto/implementation/default/makeHpke.js
  async function makeHpke(hpkealg) {
    const [aead, aeadInterface] = await makeAead(hpkealg.aead);
    const cs = new CipherSuite({
      kem: await makeDhKem(hpkealg.kem),
      kdf: makeKdf(hpkealg.kdf),
      aead: aeadInterface
    });
    return makeGenericHpke(hpkealg, aead, cs);
  }

  // node_modules/ts-mls/dist/crypto/implementation/default/rng.js
  var defaultRng = {
    randomBytes(n) {
      return crypto.getRandomValues(new Uint8Array(n));
    }
  };

  // node_modules/ts-mls/dist/crypto/implementation/default/makeNobleSignatureImpl.js
  async function makeNobleSignatureImpl(alg) {
    switch (alg) {
      case "Ed25519":
        try {
          const { ed25519 } = await import("@noble/curves/ed25519.js");
          return {
            async sign(signKey, message) {
              return ed25519.sign(message, signKey);
            },
            async verify(publicKey, message, signature) {
              return ed25519.verify(signature, message, publicKey);
            },
            async keygen() {
              const signKey = ed25519.utils.randomSecretKey();
              return { signKey, publicKey: ed25519.getPublicKey(signKey) };
            }
          };
        } catch (err) {
          throw new DependencyError("Optional dependency '@noble/curves' is not installed. Please install it to use this feature.");
        }
      case "Ed448":
        try {
          const { ed448 } = await import("@noble/curves/ed448.js");
          return {
            async sign(signKey, message) {
              return ed448.sign(message, signKey);
            },
            async verify(publicKey, message, signature) {
              return ed448.verify(signature, message, publicKey);
            },
            async keygen() {
              const signKey = ed448.utils.randomSecretKey();
              return { signKey, publicKey: ed448.getPublicKey(signKey) };
            }
          };
        } catch (err) {
          throw new DependencyError("Optional dependency '@noble/curves' is not installed. Please install it to use this feature.");
        }
      case "P256":
        try {
          const { p256 } = await import("@noble/curves/nist.js");
          return {
            async sign(signKey, message) {
              return p256.sign(message, signKey, { prehash: true, format: "der", lowS: false });
            },
            async verify(publicKey, message, signature) {
              return p256.verify(signature, message, publicKey, { prehash: true, format: "der", lowS: false });
            },
            async keygen() {
              const signKey = p256.utils.randomSecretKey();
              return { signKey, publicKey: p256.getPublicKey(signKey) };
            }
          };
        } catch (err) {
          throw new DependencyError("Optional dependency '@noble/curves' is not installed. Please install it to use this feature.");
        }
      case "P384":
        try {
          const { p384 } = await import("@noble/curves/nist.js");
          return {
            async sign(signKey, message) {
              return p384.sign(message, signKey, { prehash: true, format: "der", lowS: false });
            },
            async verify(publicKey, message, signature) {
              return p384.verify(signature, message, publicKey, { prehash: true, format: "der", lowS: false });
            },
            async keygen() {
              const signKey = p384.utils.randomSecretKey();
              return { signKey, publicKey: p384.getPublicKey(signKey) };
            }
          };
        } catch (err) {
          throw new DependencyError("Optional dependency '@noble/curves' is not installed. Please install it to use this feature.");
        }
      case "P521":
        try {
          const { p521 } = await import("@noble/curves/nist.js");
          return {
            async sign(signKey, message) {
              return p521.sign(message, signKey, { prehash: true, format: "der", lowS: false });
            },
            async verify(publicKey, message, signature) {
              return p521.verify(signature, message, publicKey, { prehash: true, format: "der", lowS: false });
            },
            async keygen() {
              const signKey = p521.utils.randomSecretKey();
              return { signKey, publicKey: p521.getPublicKey(signKey) };
            }
          };
        } catch (err) {
          throw new DependencyError("Optional dependency '@noble/curves' is not installed. Please install it to use this feature.");
        }
      case "ML-DSA-87":
        try {
          const { ml_dsa87 } = await import("@noble/post-quantum/ml-dsa.js");
          return {
            async sign(signKey, message) {
              return ml_dsa87.sign(message, signKey);
            },
            async verify(publicKey, message, signature) {
              return ml_dsa87.verify(signature, message, publicKey);
            },
            async keygen() {
              const keys = ml_dsa87.keygen(crypto.getRandomValues(new Uint8Array(32)));
              return { signKey: keys.secretKey, publicKey: keys.publicKey };
            }
          };
        } catch (err) {
          throw new DependencyError("Optional dependency '@noble/post-quantum' is not installed. Please install it to use this feature.");
        }
    }
  }

  // node_modules/ts-mls/dist/crypto/implementation/default/provider.js
  var defaultCryptoProvider = {
    async getCiphersuiteImpl(cs) {
      const sc = crypto.subtle;
      return {
        kdf: makeKdfImpl(makeKdf(cs.hpke.kdf)),
        hash: makeHashImpl(sc, cs.hash),
        signature: await makeNobleSignatureImpl(cs.signature),
        hpke: await makeHpke(cs.hpke),
        rng: defaultRng,
        name: cs.name
      };
    }
  };

  // node_modules/ts-mls/dist/crypto/getCiphersuiteImpl.js
  async function getCiphersuiteImpl(cs, provider = defaultCryptoProvider) {
    return provider.getCiphersuiteImpl(cs);
  }

  // node_modules/ts-mls/dist/message.js
  var mlsPublicMessageEncoder = contramapBufferEncoders([wireformatEncoder, publicMessageEncoder], (msg) => [msg.wireformat, msg.publicMessage]);
  var encodeMlsPublicMessage = encode(mlsPublicMessageEncoder);
  var mlsWelcomeEncoder = contramapBufferEncoders([wireformatEncoder, welcomeEncoder], (wm) => [wm.wireformat, wm.welcome]);
  var encodeMlsWelcome = encode(mlsWelcomeEncoder);
  var mlsPrivateMessageEncoder = contramapBufferEncoders([wireformatEncoder, privateMessageEncoder], (pm) => [pm.wireformat, pm.privateMessage]);
  var encodeMlsPrivateMessage = encode(mlsPrivateMessageEncoder);
  var mlsGroupInfoEncoder = contramapBufferEncoders([wireformatEncoder, groupInfoEncoder], (gi) => [gi.wireformat, gi.groupInfo]);
  var encodeMlsGroupInfo = encode(mlsGroupInfoEncoder);
  var mlsKeyPackageEncoder = contramapBufferEncoders([wireformatEncoder, keyPackageEncoder], (kp) => [kp.wireformat, kp.keyPackage]);
  var encodeMlsKeyPackage = encode(mlsKeyPackageEncoder);
  var mlsMessageContentEncoder = (mc) => {
    switch (mc.wireformat) {
      case "mls_public_message":
        return mlsPublicMessageEncoder(mc);
      case "mls_welcome":
        return mlsWelcomeEncoder(mc);
      case "mls_private_message":
        return mlsPrivateMessageEncoder(mc);
      case "mls_group_info":
        return mlsGroupInfoEncoder(mc);
      case "mls_key_package":
        return mlsKeyPackageEncoder(mc);
    }
  };
  var encodeMlsMessageContent = encode(mlsMessageContentEncoder);
  var decodeMlsMessageContent = flatMapDecoder(decodeWireformat, (wireformat) => {
    switch (wireformat) {
      case "mls_public_message":
        return mapDecoder(decodePublicMessage, (publicMessage) => ({ wireformat, publicMessage }));
      case "mls_welcome":
        return mapDecoder(decodeWelcome, (welcome) => ({ wireformat, welcome }));
      case "mls_private_message":
        return mapDecoder(decodePrivateMessage, (privateMessage) => ({ wireformat, privateMessage }));
      case "mls_group_info":
        return mapDecoder(decodeGroupInfo, (groupInfo) => ({ wireformat, groupInfo }));
      case "mls_key_package":
        return mapDecoder(decodeKeyPackage, (keyPackage) => ({ wireformat, keyPackage }));
    }
  });
  var mlsMessageEncoder = contramapBufferEncoders([protocolVersionEncoder, mlsMessageContentEncoder], (w) => [w.version, w]);
  var encodeMlsMessage = encode(mlsMessageEncoder);
  var decodeMlsMessage = mapDecoders([decodeProtocolVersion, decodeMlsMessageContent], (version, mc) => ({ ...mc, version }));

  // node_modules/ts-mls/dist/grease.js
  var greaseValues = [
    2570,
    6682,
    10794,
    14906,
    19018,
    23130,
    27242,
    31354,
    35466,
    39578,
    43690,
    47802,
    51914,
    56026,
    60138
  ];
  var defaultGreaseConfig = {
    probabilityPerGreaseValue: 0.1
  };
  function grease(greaseConfig) {
    return greaseValues.filter(() => greaseConfig.probabilityPerGreaseValue > Math.random());
  }
  function greaseCiphersuites(greaseConfig) {
    return grease(greaseConfig).map((n) => n.toString());
  }
  function greaseCredentials(greaseConfig) {
    return grease(greaseConfig).map((n) => n.toString());
  }
  function greaseCapabilities(config, capabilities) {
    return {
      ciphersuites: [...capabilities.ciphersuites, ...greaseCiphersuites(config)],
      credentials: [...capabilities.credentials, ...greaseCredentials(config)],
      extensions: [...capabilities.extensions, ...grease(config)],
      proposals: [...capabilities.proposals, ...grease(config)],
      versions: capabilities.versions
    };
  }

  // node_modules/ts-mls/dist/defaultCapabilities.js
  function defaultCapabilities() {
    return greaseCapabilities(defaultGreaseConfig, {
      versions: ["mls10"],
      ciphersuites: Object.keys(ciphersuites),
      extensions: [],
      proposals: [],
      credentials: ["basic", "x509"]
    });
  }

  // src/service/keyPackage.ts
  var KeyPackageService = class {
    // All class #properties are PRIVATE
    #database;
    #keyPackages;
    async start() {
      this.#database = await openDB("KeyPackage", 1, {
        upgrade: (db, oldVersion, _newVersion, transaction2, event) => {
          if (oldVersion == 0) {
            var keyPackages = db.createObjectStore("KeyPackage");
            keyPackages.createIndex("keyPackage_ID", ["id"]);
          }
        }
      });
      const transaction = this.#database.transaction("KeyPackage", "readwrite");
      this.#keyPackages = await transaction.store.getAll();
      this.createKeyPackage;
      this.createKeyPackage();
    }
    async defineDatabase() {
    }
    // createKeyPackage creates a new KeyPackage and
    // synchronizes it with the server.
    async createKeyPackage() {
      const implementation = await getCiphersuiteImpl(getCiphersuiteFromName("MLS_256_XWING_AES256GCM_SHA512_Ed25519"));
      const aliceCredential = { credentialType: "basic", identity: new TextEncoder().encode("alice") };
      const { publicPackage, privatePackage } = await generateKeyPackage(aliceCredential, defaultCapabilities(), defaultLifetime, [], implementation);
      await this.save(publicPackage, privatePackage);
      const response = fetch("/@me/outbox", {
        method: "POST",
        body: JSON.stringify(publicPackage)
      });
    }
    async save(publicPackage, privatePackage) {
      console.log(publicPackage);
      console.log(privatePackage);
    }
  };

  // src/service/factory.ts
  var ServiceFactory = class {
    // All class #properties are PRIVATE
    #actor = {};
    #activityPub;
    #keyPackage;
    constructor() {
      this.#activityPub = new ActivityPubService();
      this.#keyPackage = new KeyPackageService();
    }
    async start() {
      const actor = await this.loadActor();
      this.#actor = actor;
      await this.#activityPub.start(actor.id);
      await this.#keyPackage.start();
    }
    async loadActor() {
      const response = await fetch("/@me", {
        headers: [["Accept", "application/json"]]
      });
      const result = await response.json();
      if (typeof result == "object") {
        return result;
      }
      return {
        id: "",
        name: "",
        inbox: "",
        keyPackages: {
          type: "Collection",
          id: "",
          items: []
        }
      };
    }
  };
})();
/*! Bundled license information:

@noble/ciphers/utils.js:
  (*! noble-ciphers - MIT License (c) 2023 Paul Miller (paulmillr.com) *)

@hpke/common/esm/src/curve/modular.js:
@hpke/common/esm/src/curve/montgomery.js:
  (*! noble-curves - MIT License (c) 2022 Paul Miller (paulmillr.com) *)
*/
