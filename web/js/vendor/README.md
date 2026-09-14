# Vendored dependencies

Every third-party package `web/js/` imports is vendored here as a static file, rather than fetched
from a CDN at runtime — see the comment above the import map in `web/index.html` for why (in short:
self-hosted means no outbound internet dependency, and a CDN pulling in a dozen small interdependent
files is exactly the kind of thing that can hang under a flaky network). These files aren't meant to
be hand-edited — regenerate them with the commands below when a version needs bumping.

## `vue.esm-browser.prod.js`

Vue's own prebuilt "full" ESM build (includes the template compiler, since `js/` uses runtime
`template: '...'` strings, not `.vue` SFCs) — downloaded as-is, no bundling needed:

```bash
curl -o vue.esm-browser.prod.js https://unpkg.com/vue@3.5.42/dist/vue.esm-browser.prod.js
```

## `markdown-it.js`

A self-contained bundle (markdown-it plus its own small dependencies — nothing else in this project
shares them, so there's no reason to keep them external):

```bash
npm install --no-save markdown-it@15.0.2
npx esbuild markdown-it --bundle --format=esm --minify --outfile=markdown-it.js
```

## `markdown-it-multimd-table.js`

GFM-style pipe tables aren't part of markdown-it's own syntax — this plugin adds them. It doesn't
import `markdown-it` itself (just receives the instance as a parameter when `.use()`d), so unlike
the `cm-*.js` files below it's safe to bundle as a fully self-contained file:

```bash
npm install --no-save markdown-it-multimd-table@4.2.3
npx esbuild markdown-it-multimd-table --bundle --format=esm --minify --outfile=markdown-it-multimd-table.js
```

## `cm-*.js`

One file per `@codemirror/*` package actually imported (`state`, `view`, `commands`, `language`,
`lang-markdown`, `autocomplete`, `search`), each bundled **separately**, with imports of every
*other* `@codemirror/*` package left external rather than inlined:

```bash
npm install --no-save \
  @codemirror/state@6.7.4 @codemirror/view@6.43.11 @codemirror/commands@6.11.0 \
  @codemirror/language@6.12.4 @codemirror/lang-markdown@6.5.2 @codemirror/autocomplete@6.20.3 \
  @codemirror/search@6.7.2

for pkg in state view commands language lang-markdown autocomplete search; do
  npx esbuild "@codemirror/$pkg" --bundle --format=esm --minify \
    --external:@codemirror/state --external:@codemirror/view --external:@codemirror/commands \
    --external:@codemirror/language --external:@codemirror/lang-markdown \
    --external:@codemirror/autocomplete --external:@codemirror/search \
    --outfile="cm-$pkg.js"
done
```

This matters: CodeMirror 6's extension system (facets, state fields) identifies things by object
identity from `@codemirror/state`. If each `cm-*.js` file bundled its own copy inline instead, the
seven files would each hold a *different* `EditorState`/`Facet`/etc. class, and the editor would
break in confusing ways (extensions silently not applying, since one package's `Facet.define()`
wouldn't `===` another's). Keeping cross-package imports external and resolving them all through the
same `web/index.html` import map — `"@codemirror/state": "js/vendor/cm-state.js"` etc. — is what
guarantees every file shares the literal same instances, same as if they were all one CDN URL
requested from multiple packages.

When bumping a version, keep the pin here and in `web/index.html`'s doc comment in sync with
whatever's actually installed.
