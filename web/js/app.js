import { createApp } from 'vue'
import NoteList from './components/note-list.js'
import NoteView from './components/note-view.js'

createApp({
  components: { 'note-list': NoteList, 'note-view': NoteView },
  template: `
    <div class="app-shell">
      <note-list></note-list>
      <note-view></note-view>
    </div>
  `,
}).mount('#app')
