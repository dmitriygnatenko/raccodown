import { EditorState } from '@codemirror/state'
import {
  EditorView,
  keymap,
  lineNumbers,
  highlightActiveLine,
  highlightActiveLineGutter,
  drawSelection,
  placeholder as placeholderExt,
} from '@codemirror/view'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { searchKeymap } from '@codemirror/search'
import { closeBrackets, closeBracketsKeymap } from '@codemirror/autocomplete'
import { syntaxHighlighting, defaultHighlightStyle } from '@codemirror/language'
import { markdown } from '@codemirror/lang-markdown'
import {
  wrapInline,
  linePrefix,
  toggleHeading,
  numberedList,
  insertLink,
  insertCodeBlock,
  insertHorizontalRule,
} from '../data/markdown-commands.js'

export default {
  props: { modelValue: { type: String, default: '' } },
  emits: ['update:modelValue'],
  template: `
    <div class="cm-wrapper">
      <div class="cm-toolbar">
        <button title="Заголовок" @click="toggleHeading(view)">H</button>
        <button title="Жирный (Ctrl/Cmd+B)" class="glyph-bold" @click="wrapInline(view, '**')">Ж</button>
        <button title="Курсив (Ctrl/Cmd+I)" class="glyph-italic" @click="wrapInline(view, '*')">К</button>
        <button title="Зачёркнутый" class="glyph-strike" @click="wrapInline(view, '~~')">З</button>
        <span class="cm-toolbar-sep"></span>
        <button title="Код" @click="wrapInline(view, '\`')">&lt;/&gt;</button>
        <button title="Блок кода" @click="insertCodeBlock(view)">{ }</button>
        <button title="Ссылка" @click="insertLink(view)">🔗</button>
        <span class="cm-toolbar-sep"></span>
        <button title="Цитата" @click="linePrefix(view, '> ')">❝</button>
        <button title="Список" @click="linePrefix(view, '- ')">•</button>
        <button title="Нумерованный список" @click="numberedList(view)">1.</button>
        <span class="cm-toolbar-sep"></span>
        <button title="Разделитель" @click="insertHorizontalRule(view)">―</button>
      </div>
      <div ref="host" class="cm-host"></div>
    </div>
  `,
  data() {
    return { view: null }
  },
  mounted() {
    const state = EditorState.create({
      doc: this.modelValue,
      extensions: [
        lineNumbers(),
        highlightActiveLine(),
        highlightActiveLineGutter(),
        drawSelection(),
        history(),
        closeBrackets(),
        syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
        markdown(),
        placeholderExt('Пишите в Markdown…'),
        keymap.of([
          { key: 'Mod-b', run: () => (wrapInline(this.view, '**'), true) },
          { key: 'Mod-i', run: () => (wrapInline(this.view, '*'), true) },
          ...closeBracketsKeymap,
          ...defaultKeymap,
          ...historyKeymap,
          ...searchKeymap,
        ]),
        EditorView.lineWrapping,
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            this.$emit('update:modelValue', update.state.doc.toString())
          }
        }),
      ],
    })

    this.view = new EditorView({ state, parent: this.$refs.host })
  },
  beforeUnmount() {
    this.view?.destroy()
  },
  watch: {
    modelValue(value) {
      if (value !== this.view.state.doc.toString()) {
        this.view.dispatch({ changes: { from: 0, to: this.view.state.doc.length, insert: value } })
      }
    },
  },
  methods: { wrapInline, linePrefix, toggleHeading, numberedList, insertLink, insertCodeBlock, insertHorizontalRule },
}
