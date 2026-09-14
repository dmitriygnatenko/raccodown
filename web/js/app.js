import { createApp } from 'vue'
import { authStore } from './store/auth.js'
import { notesStore } from './store/notes.js'
import LoginView from './components/login-view.js'
import NoteList from './components/note-list.js'
import NoteView from './components/note-view.js'

const app = createApp({
  components: { 'login-view': LoginView, 'note-list': NoteList, 'note-view': NoteView },
  template: `
    <div v-if="store.checking" class="boot-screen">🦝</div>
    <login-view v-else-if="!authenticated"></login-view>
    <div v-else class="app-shell" :class="{ 'note-open': hasActiveNote }">
      <note-list></note-list>
      <note-view></note-view>
    </div>
  `,
  data() {
    return { store: authStore.state, notesState: notesStore.state }
  },
  computed: {
    authenticated() {
      return authStore.isAuthenticated()
    },
    // On narrow screens only one of the sidebar/workspace panes is shown at a time (see
    // .app-shell.note-open in style.css) — this tracks which one, mirroring notesStore's
    // selection rather than keeping separate view-state of its own.
    hasActiveNote() {
      return this.notesState.activeId != null
    },
  },
  mounted() {
    authStore.checkSession()
  },
})

app.mount('#app')
