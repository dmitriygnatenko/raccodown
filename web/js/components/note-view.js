import { notesStore } from '../store/notes.js'
import { t } from '../data/i18n.js'
import Editor from './editor.js'
import Preview from './preview.js'

export default {
  components: { 'cm-editor': Editor, 'md-preview': Preview },
  template: `
    <section class="workspace" v-if="note">
      <header class="toolbar">
        <input v-model="title" class="title-input" :placeholder="t('Без названия')" />
        <div class="toolbar-actions">
          <span class="save-state">{{ saveStateLabel }}</span>
          <div class="mode-switch">
            <button :class="{ active: mode === 'edit' }" @click="mode = 'edit'">{{ t('Правка') }}</button>
            <button :class="{ active: mode === 'split' }" @click="mode = 'split'">{{ t('Оба') }}</button>
            <button :class="{ active: mode === 'preview' }" @click="mode = 'preview'">{{ t('Превью') }}</button>
          </div>
        </div>
      </header>

      <div v-if="store.conflict" class="conflict-banner">
        {{ t('Заметка была изменена в другом месте.') }}
        <button @click="keepMine">{{ t('Оставить мою версию') }}</button>
        <button @click="takeTheirs">{{ t('Взять чужую версию') }}</button>
      </div>

      <div class="panes" :class="mode">
        <div v-if="mode !== 'preview'" class="pane editor-pane">
          <cm-editor v-model="content" />
        </div>
        <div v-if="mode !== 'edit'" class="pane preview-pane">
          <md-preview :content="content" />
        </div>
      </div>

      <footer class="status-bar">
        <span>{{ t('{n} слов', { n: wordCount }) }}</span>
        <span v-if="note.tags.length">{{ note.tags.map((tag) => '#' + tag).join(' ') }}</span>
      </footer>
    </section>

    <section v-else class="empty-state">
      <p>{{ t('Выберите заметку слева или создайте новую.') }}</p>
    </section>
  `,
  data() {
    return {
      store: notesStore.state,
      mode: 'preview',
      title: '',
      content: '',
      saveState: 'idle',
      // The id of the note title/content currently reflect — NOT a generic "just loaded" boolean.
      // A save echoes the server's copy of the note back into the store (see notesStore.saveActive),
      // which replaces state.notes[idx] with a new object and re-fires the `note` watcher below even
      // though nothing the user cares about changed. Keying suppression off note identity, rather
      // than a flag that watcher can also flip, means that echo is simply a no-op instead of
      // re-arming a "suppress the next save" flag and silently swallowing the user's next real edit.
      loadedNoteId: null,
      suppressNextSave: false,
      saveTimer: null,
    }
  },
  computed: {
    note() {
      return notesStore.active()
    },
    saveStateLabel() {
      if (this.saveState === 'saving') return t('Сохранение…')
      if (this.saveState === 'saved') return t('Сохранено')
      return ''
    },
    wordCount() {
      return this.content.trim().split(/\s+/).filter(Boolean).length
    },
  },
  watch: {
    note: {
      immediate: true,
      handler(note) {
        if (note && note.id === this.loadedNoteId) return // a save echo, not a real switch

        this.loadedNoteId = note?.id ?? null
        this.suppressNextSave = true
        this.title = note?.title ?? ''
        this.content = note?.content ?? ''
        this.saveState = 'idle'

        // A freshly created blank note is more useful to write and preview side by side than in
        // whatever mode was last showing — see notesStore.createNote, which sets this marker.
        if (note && note.id === this.store.justCreatedId) {
          this.mode = 'split'
          this.store.justCreatedId = null
        }
      },
    },
  },
  created() {
    // One watcher over both fields (rather than two separate `watch: { title(){}, content(){} }`
    // entries) so a note switch — which reassigns both — only has to suppress a single resulting
    // callback, not race two independent ones over the same flag.
    this.$watch(() => [this.title, this.content], () => this.maybeScheduleSave())
  },
  methods: {
    t,
    maybeScheduleSave() {
      if (this.suppressNextSave) {
        this.suppressNextSave = false
        return
      }

      this.saveState = 'saving'
      clearTimeout(this.saveTimer)
      this.saveTimer = setTimeout(async () => {
        await notesStore.saveActive(this.title, this.content)
        this.saveState = this.store.conflict ? 'idle' : 'saved'
      }, 600)
    },
    keepMine() {
      if (!this.store.conflict) return
      notesStore.saveActive(this.title, this.content)
    },
    takeTheirs() {
      if (!this.store.conflict) return
      this.title = this.store.conflict.title
      this.content = this.store.conflict.content
      this.store.conflict = null
    },
  },
}
