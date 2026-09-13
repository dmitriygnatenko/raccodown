import { notesStore } from '../store/notes.js'

export default {
  template: `
    <aside class="sidebar">
      <div class="sidebar-header">
        <span class="brand">🦝 Raccodown</span>
        <button class="icon-btn" title="Новая заметка" @click="createNote">+</button>
      </div>

      <input v-model="query" class="search" type="search" placeholder="Поиск заметок…" />

      <div v-if="store.tags.length" class="tag-row">
        <button
          v-for="t in store.tags"
          :key="t.tag"
          class="tag-chip"
          :class="{ active: activeTag === t.tag }"
          @click="toggleTag(t.tag)"
        >
          #{{ t.tag }} <span class="count">{{ t.count }}</span>
        </button>
      </div>

      <p v-if="store.loading" class="hint">Загрузка…</p>
      <p v-else-if="store.error" class="hint error">{{ store.error }}</p>
      <p v-else-if="store.notes.length === 0" class="hint">Ничего не найдено</p>

      <ul class="note-list">
        <li
          v-for="note in store.notes"
          :key="note.id"
          class="note-item"
          :class="{ active: note.id === store.activeId }"
          @click="select(note.id)"
        >
          <div class="note-title">{{ note.title || 'Без названия' }}</div>
          <div class="note-meta">{{ formatDate(note.updatedAt) }} · {{ note.tags.join(', ') }}</div>
        </li>
      </ul>
    </aside>
  `,
  data() {
    return {
      store: notesStore.state,
      query: '',
      activeTag: null,
      debounce: null,
    }
  },
  mounted() {
    notesStore.fetchNotes()
    notesStore.fetchTags()
  },
  watch: {
    query() {
      clearTimeout(this.debounce)
      this.debounce = setTimeout(() => {
        notesStore.fetchNotes({ q: this.query, tag: this.activeTag ?? undefined })
      }, 250)
    },
  },
  methods: {
    select(id) {
      notesStore.select(id)
    },
    toggleTag(tag) {
      this.activeTag = this.activeTag === tag ? null : tag
      notesStore.fetchNotes({ q: this.query, tag: this.activeTag ?? undefined })
    },
    createNote() {
      notesStore.createNote()
    },
    formatDate(iso) {
      return new Date(iso).toLocaleDateString('ru-RU', { day: '2-digit', month: 'short' })
    },
  },
}
