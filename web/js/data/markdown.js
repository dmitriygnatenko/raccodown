import MarkdownIt from 'markdown-it'
import multimdTable from 'markdown-it-multimd-table'

const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: false,
})

// markdown-it-multimd-table still calls the `md.utils.assign` helper markdown-it itself dropped in
// v14+ (native Object.assign superseded it) — shim it back rather than hand-patching the vendored
// plugin file, which is meant to stay a pristine, regeneratable build (see js/vendor/README.md).
md.utils.assign ??= Object.assign

// GFM-style pipe tables aren't part of markdown-it's own syntax (unlike headings/lists/code/etc.),
// so they need this plugin — see js/vendor/README.md for how it's vendored.
md.use(multimdTable, { multiline: false, rowspan: false, headerless: false, multibody: true })

export function renderMarkdown(content) {
  return md.render(content)
}
