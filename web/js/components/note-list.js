import { notesStore } from '../store/notes.js'
import { authStore } from '../store/auth.js'
import { effectiveTheme } from '../data/theme.js'
import { i18nStore, t, SUPPORTED_LANGUAGES, LANGUAGE_LABELS } from '../data/i18n.js'
import CredentialsModal from './credentials-modal.js'
import DeleteNoteModal from './delete-note-modal.js'

// dateLocales maps our language codes to the locale toLocaleDateString expects.
const dateLocales = { ru: 'ru-RU', en: 'en-US', es: 'es-ES', de: 'de-DE', fr: 'fr-FR' }

export default {
  components: { 'credentials-modal': CredentialsModal, 'delete-note-modal': DeleteNoteModal },
  template: `
    <aside class="sidebar">
      <div class="sidebar-header">
        <button class="icon-btn" :title="t('Добавить заметку')" @click="createNote">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8L14 2Z" />
            <path d="M14 2v6h6" />
            <path d="M12 12v6M9 15h6" />
          </svg>
        </button>
        <input v-model="query" class="search" type="search" :placeholder="t('Поиск заметок…')" />
      </div>

      <div v-if="store.tags.length" class="tag-row">
        <button
          v-for="tc in store.tags"
          :key="tc.tag"
          class="tag-chip"
          :class="{ active: activeTag === tc.tag }"
          @click="toggleTag(tc.tag)"
        >
          #{{ tc.tag }}
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
          <div class="note-item-main">
            <div class="note-title">{{ note.title || t('Без названия') }}</div>
            <div class="note-meta">{{ formatDate(note.updatedAt) }} · {{ note.tags.join(', ') }}</div>
          </div>
          <button class="note-delete" :title="t('Удалить')" @click.stop="remove(note)">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M6 7h12M9 7V4a1 1 0 0 1 1-1h4a1 1 0 0 1 1 1v3m2 0v13a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2V7h12ZM10 11v6M14 11v6" />
            </svg>
          </button>
        </li>
      </ul>

      <div class="sidebar-footer">
        <div class="user-menu">
          <button class="user-menu-trigger" @click="menuOpen = !menuOpen">
            <span class="user-avatar">{{ userInitial }}</span>
            <span class="sidebar-user" :title="authStore.user?.username">{{ authStore.user?.username }}</span>
            <svg class="chevron" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M6 9l6 6 6-6" /></svg>
          </button>

          <div v-if="menuOpen" class="menu-backdrop" @click="menuOpen = false"></div>

          <div v-if="menuOpen" class="user-menu-panel">
            <button class="user-menu-item" :title="t('Импортировать .md/.txt файлы как заметки')" @click="triggerImport">
              <span>📥 {{ t('Импорт') }}</span>
            </button>
            <button class="user-menu-item" :title="t('Скачать все заметки одним zip-архивом')" @click="exportNotes">
              <span>📦 {{ t('Экспорт') }}</span>
            </button>
            <input ref="importInput" type="file" accept=".md,.markdown,.txt,text/markdown,text/plain" multiple hidden @change="importFiles" />

            <div class="user-menu-sep"></div>

            <button class="user-menu-item" @click="toggleTheme">
              <span>{{ themeIcon }} {{ themeButtonTitle }}</span>
            </button>

            <label class="user-menu-item user-menu-lang">
              <span>{{ t('Язык') }}</span>
              <select :aria-label="t('Язык')" :value="i18nStore.language" @change="changeLanguage($event.target.value)">
                <option v-for="lang in languages" :key="lang" :value="lang">{{ languageLabels[lang] }}</option>
              </select>
            </label>

            <div class="user-menu-sep"></div>

            <button class="user-menu-item" @click="openCredentials">
              <span>{{ t('Изменить логин и пароль') }}</span>
            </button>
            <button class="user-menu-item user-menu-danger" @click="logout">{{ t('Выйти') }}</button>
          </div>
        </div>
      </div>

      <credentials-modal v-if="showCredentials" @close="showCredentials = false" />
      <delete-note-modal v-if="noteToDelete" :note="noteToDelete" @close="noteToDelete = null" />
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
      menuOpen: false,
      showCredentials: false,
      noteToDelete: null,
    }
  },
  computed: {
    themeIcon() {
      return effectiveTheme() === 'dark' ? '☀️' : '🌙'
    },
    themeButtonTitle() {
      return effectiveTheme() === 'dark' ? t('Светлая тема') : t('Тёмная тема')
    },
    userInitial() {
      return (authStore.state.user?.username?.[0] ?? '?').toUpperCase()
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
    remove(note) {
      this.noteToDelete = note
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
      this.menuOpen = false
      authStore.logout()
    },
    openCredentials() {
      this.menuOpen = false
      this.showCredentials = true
    },
    triggerImport() {
      this.menuOpen = false
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
      this.menuOpen = false
      window.location.href = '/api/v1/notes/export'
    },
    formatDate(iso) {
      return new Date(iso).toLocaleDateString(dateLocales[i18nStore.language] ?? 'ru-RU', { day: '2-digit', month: 'short' })
    },
  },
}
