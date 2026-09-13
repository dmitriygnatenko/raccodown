import { notesStore } from '../store/notes.js'
import { authStore } from '../store/auth.js'
import { effectiveTheme } from '../data/theme.js'
import { i18nStore, t, SUPPORTED_LANGUAGES, LANGUAGE_LABELS } from '../data/i18n.js'

// dateLocales maps our language codes to the locale toLocaleDateString expects.
const dateLocales = { ru: 'ru-RU', en: 'en-US', es: 'es-ES', de: 'de-DE', fr: 'fr-FR' }

export default {
  template: `
    <aside class="sidebar">
      <div class="sidebar-header">
        <span class="brand">🦝 Raccodown</span>
        <div class="header-actions">
          <button class="icon-btn-ghost" :title="themeButtonTitle" @click="toggleTheme">{{ themeIcon }}</button>
          <button class="icon-btn" :title="t('Новая заметка')" @click="createNote">+</button>
        </div>
      </div>

      <div class="sidebar-toolbar">
        <button class="text-btn" :title="t('Импортировать .md/.txt файлы как заметки')" @click="triggerImport">📥 {{ t('Импорт') }}</button>
        <button class="text-btn" :title="t('Скачать все заметки одним zip-архивом')" @click="exportNotes">📦 {{ t('Экспорт') }}</button>
        <input ref="importInput" type="file" accept=".md,.markdown,.txt,text/markdown,text/plain" multiple hidden @change="importFiles" />
      </div>

      <input v-model="query" class="search" type="search" :placeholder="t('Поиск заметок…')" />

      <div v-if="store.tags.length" class="tag-row">
        <button
          v-for="tc in store.tags"
          :key="tc.tag"
          class="tag-chip"
          :class="{ active: activeTag === tc.tag }"
          @click="toggleTag(tc.tag)"
        >
          #{{ tc.tag }} <span class="count">{{ tc.count }}</span>
        </button>
      </div>

      <p v-if="store.loading" class="hint">{{ t('Загрузка…') }}</p>
      <p v-else-if="store.error" class="hint error">{{ store.error }}</p>
      <p v-else-if="store.notes.length === 0" class="hint">{{ t('Ничего не найдено') }}</p>

      <ul class="note-list">
        <li
          v-for="note in store.notes"
          :key="note.id"
          class="note-item"
          :class="{ active: note.id === store.activeId }"
          @click="select(note.id)"
        >
          <div class="note-title">{{ note.title || t('Без названия') }}</div>
          <div class="note-meta">{{ formatDate(note.updatedAt) }} · {{ note.tags.join(', ') }}</div>
        </li>
      </ul>

      <div class="sidebar-footer">
        <select class="lang-select" :aria-label="t('Язык')" :value="i18nStore.language" @change="changeLanguage($event.target.value)">
          <option v-for="lang in languages" :key="lang" :value="lang">{{ languageLabels[lang] }}</option>
        </select>
        <span class="sidebar-user" :title="authStore.user?.username">{{ authStore.user?.username }}</span>
        <button class="text-btn" @click="logout">{{ t('Выйти') }}</button>
      </div>
    </aside>
  `,
  data() {
    return {
      store: notesStore.state,
      authStore: authStore.state,
      i18nStore,
      languages: SUPPORTED_LANGUAGES,
      languageLabels: LANGUAGE_LABELS,
      query: '',
      activeTag: null,
      debounce: null,
    }
  },
  computed: {
    themeIcon() {
      return effectiveTheme() === 'dark' ? '☀️' : '🌙'
    },
    themeButtonTitle() {
      return effectiveTheme() === 'dark' ? t('Светлая тема') : t('Тёмная тема')
    },
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
    t,
    changeLanguage(language) {
      authStore.changeLanguage(language)
    },
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
    toggleTheme() {
      authStore.changeTheme(effectiveTheme() === 'dark' ? 'light' : 'dark')
    },
    logout() {
      authStore.logout()
    },
    triggerImport() {
      this.$refs.importInput.click()
    },
    // importFiles creates one note per selected file (title from the filename, content verbatim),
    // one at a time — mirrors how handleExportNotes lays notes back out as one .md file each, so a
    // previous export round-trips straight back in.
    async importFiles(event) {
      const files = Array.from(event.target.files ?? [])
      event.target.value = '' // so picking the same file again still fires 'change'
      if (files.length === 0) return

      let lastNote = null
      try {
        for (const file of files) {
          const content = await file.text()
          const title = file.name.replace(/\.(md|markdown|txt)$/i, '')
          lastNote = await notesStore.importNote(title, content)
        }
        await notesStore.fetchTags()
      } catch (err) {
        this.store.error = err.message ?? t('Не удалось импортировать файл')
      } finally {
        if (lastNote) notesStore.select(lastNote.id)
      }
    },
    exportNotes() {
      window.location.href = '/api/v1/notes/export'
    },
    formatDate(iso) {
      return new Date(iso).toLocaleDateString(dateLocales[i18nStore.language] ?? 'ru-RU', { day: '2-digit', month: 'short' })
    },
  },
}
