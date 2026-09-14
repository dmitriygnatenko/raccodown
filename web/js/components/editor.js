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
  insertTable,
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
        <button :title="t('Блок кода')" class="glyph-mono" @click="insertCodeBlock(view)">{}</button>
        <button :title="t('Ссылка')" @click="insertLink(view)">🔗</button>
        <button :title="t('Цитата')" @click="linePrefix(view, '> ')">❝</button>
        <button :title="t('Таблица')" @click="insertTable(view)">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect x="3" y="4" width="18" height="16" rx="1" />
            <line x1="3" y1="10" x2="21" y2="10" />
            <line x1="9" y1="4" x2="9" y2="20" />
            <line x1="15" y1="4" x2="15" y2="20" />
          </svg>
        </button>
        <span class="cm-toolbar-sep"></span>
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
  methods: {
    t,
    wrapInline,
    linePrefix,
    toggleHeading,
    numberedList,
    insertLink,
    insertCodeBlock,
    insertTable,
    insertHorizontalRule,
  },
}
