import { reactive } from 'vue'
import { notesApi, ConflictError } from '../data/api.js'

function createNotesStore() {
  const state = reactive({
    notes: [],
    tags: [],
    activeId: null,
    loading: false,
    error: null,
    conflict: null,
  })

  function active() {
    return state.notes.find((n) => n.id === state.activeId)
  }

  async function fetchNotes(params = {}) {
    state.loading = true
    state.error = null
    try {
      const { notes } = await notesApi.list(params)
      state.notes = notes
      if (!state.activeId && notes.length > 0) {
        state.activeId = notes[0].id
      }
    } catch (err) {
      state.error = err.message ?? 'Failed to load notes'
    } finally {
      state.loading = false
    }
  }

  async function fetchTags() {
    state.tags = await notesApi.tags()
  }

  function select(id) {
    state.activeId = id
    state.conflict = null
  }

  async function createNote() {
    const note = await notesApi.create('Новая заметка', '')
    state.notes.unshift(note)
    state.activeId = note.id
    return note
  }

  async function saveActive(title, content) {
    const note = active()
    if (!note) return

    state.conflict = null
    try {
      const updated = await notesApi.update(note.id, title, content, note.checksum)
      const idx = state.notes.findIndex((n) => n.id === note.id)
      if (idx !== -1) state.notes[idx] = updated
    } catch (err) {
      if (err instanceof ConflictError) {
        state.conflict = err.current
      } else {
        state.error = err.message ?? 'Failed to save note'
      }
    }
  }

  async function removeActive() {
    const note = active()
    if (!note) return
    await notesApi.remove(note.id)
    state.notes = state.notes.filter((n) => n.id !== note.id)
    state.activeId = state.notes[0]?.id ?? null
  }

  return { state, active, fetchNotes, fetchTags, select, createNote, saveActive, removeActive }
}

export const notesStore = createNotesStore()
