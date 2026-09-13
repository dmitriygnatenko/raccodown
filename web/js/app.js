import { createApp } from 'vue'
import { authStore } from './store/auth.js'
import LoginView from './components/login-view.js'
import NoteList from './components/note-list.js'
import NoteView from './components/note-view.js'

const app = createApp({
  components: { 'login-view': LoginView, 'note-list': NoteList, 'note-view': NoteView },
  template: `
    <div v-if="store.checking" class="boot-screen">🦝</div>
    <login-view v-else-if="!authenticated"></login-view>
    <div v-else class="app-shell">
      <note-list></note-list>
      <note-view></note-view>
    </div>
  `,
  data() {
    return { store: authStore.state }
  },
  computed: {
    authenticated() {
      return authStore.isAuthenticated()
    },
  },
  mounted() {
    authStore.checkSession()
  },
})

app.mount('#app')
