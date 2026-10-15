# Prism 1.30.0

Syntax highlighting for code blocks, used by themes that load it (see EMISSARY-DEV-WEBSITE D1, D9, D11).

[prism.min.js](prism.min.js) is the unmodified minified files from the `prismjs` npm package, version 1.30.0, joined in dependency order with a newline between each: `prism-core`, then the components `markup`, `css`, `clike`, `javascript`, `typescript`, `go`, `json`, `bash`, `yaml`, `toml`, and `markdown`. Nothing else is changed. The tarball (`https://registry.npmjs.org/prismjs/-/prismjs-1.30.0.tgz`) matched the registry's integrity hash, `sha512-DEvV2ZF2r2/63V+tK8hQvrR2ZGn10srHbXviTlcv7Kpzw8jWiNTqbVgjO3IY8RxrrOUF8VPMQQFysYYYv0YZxw==`. 1.30.0 fixes CVE-2024-53382, a DOM-clobbering flaw in earlier versions.

To add a language, append its `components/prism-<name>.min.js` after the components it requires (listed in the package's `components.json`), and record it here.

HJSON is not a Prism component. [../../javascript/prism.js](../../javascript/prism.js) defines it as JSON plus comments, and re-highlights after every htmx swap.

[LICENSE](LICENSE) is Prism's MIT license, copied from the package.
