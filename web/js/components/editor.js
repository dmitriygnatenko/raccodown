import { EditorState, Compartment } from '@codemirror/state'
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
import { i18nStore, t } from '../data/i18n.js'

export default {
  props: { modelValue: { type: String, default: '' } },
  emits: ['update:modelValue'],
  template: `
    <div class="cm-wrapper">
      <div class="cm-toolbar">
        <button :title="t('Заголовок')" @click="toggleHeading(view)">H</button>
        <button :title="t('Жирный (Ctrl/Cmd+B)')" class="glyph-bold" @click="wrapInline(view, '**')">{{ t('Жирный (Ctrl/Cmd+B)').charAt(0) }}</button>
        <button :title="t('Курсив (Ctrl/Cmd+I)')" class="glyph-italic" @click="wrapInline(view, '*')">{{ t('Курсив (Ctrl/Cmd+I)').charAt(0) }}</button>
        <button :title="t('Зачёркнутый')" class="glyph-strike" @click="wrapInline(view, '~~')">{{ t('Зачёркнутый').charAt(0) }}</button>
        <span class="cm-toolbar-sep"></span>
        <button :title="t('Код')" @click="wrapInline(view, '\`')">&lt;/&gt;</button>
        <button :title="t('Блок кода')" @click="insertCodeBlock(view)">{ }</button>
        <button :title="t('Ссылка')" @click="insertLink(view)">🔗</button>
        <span class="cm-toolbar-sep"></span>
        <button :title="t('Цитата')" @click="linePrefix(view, '> ')">❝</button>
        <button :title="t('Список')" @click="linePrefix(view, '- ')">•</button>
        <button :title="t('Нумерованный список')" @click="numberedList(view)">1.</button>
        <span class="cm-toolbar-sep"></span>
        <button :title="t('Разделитель')" @click="insertHorizontalRule(view)">―</button>
      </div>
      <div ref="host" class="cm-host"></div>
    </div>
  `,
  data() {
    return { view: null, i18nStore, placeholderCompartment: new Compartment() }
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
        this.placeholderCompartment.of(placeholderExt(t('Пишите в Markdown…'))),
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
    // The placeholder text is baked into a CodeMirror extension at mount time, so switching the UI
    // language needs an explicit reconfigure to follow along — everything else here is plain Vue
    // template text, which re-renders on its own.
    'i18nStore.language'() {
      this.view.dispatch({
        effects: this.placeholderCompartment.reconfigure(placeholderExt(t('Пишите в Markdown…'))),
      })
    },
  },
  methods: { t, wrapInline, linePrefix, toggleHeading, numberedList, insertLink, insertCodeBlock, insertHorizontalRule },
}
