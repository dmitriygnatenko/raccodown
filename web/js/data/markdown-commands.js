// Toolbar/keybinding commands for the CodeMirror editor (see components/editor.js). CodeMirror 6
// ships no Markdown-specific formatting commands of its own — these are small, self-contained
// helpers built on its core transaction API (state.changeByRange for multi-selection wraps, plain
// view.dispatch({changes}) for line-prefix toggles).
import { EditorSelection } from '@codemirror/state'
import { t } from './i18n.js'

// wrapInline wraps every selection range in `before`/`after` (e.g. '**' for bold), or removes them
// if the selection is already wrapped — so clicking Bold twice on the same text un-bolds it. With
// an empty selection, it inserts both markers and places the cursor between them.
export function wrapInline(view, before, after = before) {
  const { state } = view

  view.dispatch(
    state.changeByRange((range) => {
      const selected = state.sliceDoc(range.from, range.to)
      const wrapped = selected.startsWith(before) && selected.endsWith(after) && selected.length >= before.length + after.length

      const insert = wrapped ? selected.slice(before.length, selected.length - after.length) : before + selected + after
      const from = range.from
      const to = wrapped ? from + insert.length : from + before.length + selected.length

      return {
        changes: { from: range.from, to: range.to, insert },
        range: EditorSelection.range(wrapped ? from : from + before.length, to),
      }
    }),
  )
  view.focus()
}

// linePrefix toggles `prefix` (e.g. '> ' for a quote, '- ' for a bullet list) at the start of every
// line the selection touches. Toggles off only when every touched line already carries it, so
// prefixing a mixed selection always normalizes to "add everywhere" first.
export function linePrefix(view, prefix) {
  const { state } = view
  const range = state.selection.main
  const startLine = state.doc.lineAt(range.from).number
  const endLine = state.doc.lineAt(range.to).number

  const lines = []
  for (let n = startLine; n <= endLine; n++) lines.push(state.doc.line(n))

  const allPrefixed = lines.every((line) => line.text.startsWith(prefix))
  const changes = lines.map((line) =>
    allPrefixed ? { from: line.from, to: line.from + prefix.length, insert: '' } : { from: line.from, insert: prefix },
  )

  view.dispatch({ changes })
  view.focus()
}

// toggleHeading cycles the line the cursor is on through #, ##, ### and back to no heading.
export function toggleHeading(view) {
  const { state } = view
  const line = state.doc.lineAt(state.selection.main.from)
  const match = line.text.match(/^(#{1,6})\s/)

  let prefix
  if (!match) prefix = '# '
  else if (match[1].length < 3) prefix = '#'.repeat(match[1].length + 1) + ' '
  else prefix = ''

  view.dispatch({ changes: { from: line.from, to: line.from + (match?.[0].length ?? 0), insert: prefix } })
  view.focus()
}

// numberedList replaces the selected lines' existing numbering (if the first line has any) or adds
// fresh sequential numbering (1., 2., 3., ...) otherwise.
export function numberedList(view) {
  const { state } = view
  const range = state.selection.main
  const startLine = state.doc.lineAt(range.from).number
  const endLine = state.doc.lineAt(range.to).number

  const lines = []
  for (let n = startLine; n <= endLine; n++) lines.push(state.doc.line(n))

  const removing = /^\d+\.\s/.test(lines[0].text)
  const changes = lines.map((line, i) => {
    const existing = line.text.match(/^\d+\.\s/)
    const to = line.from + (existing?.[0].length ?? 0)
    return { from: line.from, to, insert: removing ? '' : `${i + 1}. ` }
  })

  view.dispatch({ changes })
  view.focus()
}

// insertLink wraps the selection as link text (or a placeholder) and selects the URL placeholder so
// typing immediately replaces it.
export function insertLink(view) {
  const { state } = view
  const { from, to } = state.selection.main
  const text = state.sliceDoc(from, to) || t('текст')
  const insert = `[${text}](url)`
  const urlStart = from + text.length + 3

  view.dispatch({
    changes: { from, to, insert },
    selection: EditorSelection.range(urlStart, urlStart + 3),
  })
  view.focus()
}

// insertCodeBlock wraps the selection in a fenced code block, or opens an empty one with the cursor
// inside.
export function insertCodeBlock(view) {
  const { state } = view
  const { from, to } = state.selection.main
  const text = state.sliceDoc(from, to)
  const insert = '```\n' + text + '\n```'

  view.dispatch({
    changes: { from, to, insert },
    selection: EditorSelection.range(from + 4, from + 4 + text.length),
  })
  view.focus()
}

// insertHorizontalRule inserts a Markdown thematic break on its own line below the cursor.
export function insertHorizontalRule(view) {
  const { state } = view
  const pos = state.selection.main.to
  const line = state.doc.lineAt(pos)
  const insert = (line.text ? '\n' : '') + '---\n'

  view.dispatch({
    changes: { from: line.to, insert },
    selection: EditorSelection.cursor(line.to + insert.length),
  })
  view.focus()
}
