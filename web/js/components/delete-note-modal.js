import { notesStore } from '../store/notes.js'
import { t } from '../data/i18n.js'

// DeleteNoteModal confirms deleting a note before calling notesStore.remove — replaces the native
// confirm() dialog. Opened from note-list.js's per-row delete button; @close is emitted on cancel,
// backdrop click, Escape, or after a successful delete.
export default {
  props: {
    note: { type: Object, required: true },
  },
  emits: ['close'],
  template: `
    <div class="modal-backdrop" @click.self="close" @keydown.esc="close">
      <div class="modal-card">
        <h2 class="modal-title">{{ t('Удалить заметку?') }}</h2>
        <p class="modal-body">{{ t('Заметка «{title}» будет удалена без возможности восстановления.', { title: note.title || t('Без названия') }) }}</p>

        <div class="modal-actions">
          <button type="button" class="modal-cancel" @click="close">{{ t('Отмена') }}</button>
          <button type="button" class="modal-danger" :disabled="deleting" @click="confirmDelete">
            {{ deleting ? t('Удаление…') : t('Удалить') }}
          </button>
        </div>
      </div>
    </div>
  `,
  data() {
    return {
      deleting: false,
    }
  },
  methods: {
    t,
    close() {
      this.$emit('close')
    },
    async confirmDelete() {
      this.deleting = true
      try {
        await notesStore.remove(this.note.id)
        await notesStore.fetchTags()
        this.close()
      } finally {
        this.deleting = false
      }
    },
  },
}
