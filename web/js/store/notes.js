import { reactive } from 'vue'
import { notesApi, ConflictError } from '../data/api.js'
import { t } from '../data/i18n.js'

function createNotesStore() {
  const state = reactive({
    notes: [],
    tags: [],
    activeId: null,
    loading: false,
    error: null,
    conflict: null,
    // Set by createNote(), consumed and cleared by note-view.js's `note` watcher — see there.
    justCreatedId: null,
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
    const note = await notesApi.create(t('Новая заметка'), '')
    state.notes.unshift(note)
    state.activeId = note.id
    // A one-shot marker note-view.js's `note` watcher consumes to switch into split ("Оба") mode
    // for this one note only — a blank new note is more useful to write and preview side by side
    // than in whatever mode was last showing.
    state.justCreatedId = note.id
    return note
  }

  // importNote creates a note from imported file content without changing the current selection —
  // callers importing several files in a row select whichever one they want active once the whole
  // batch is done, rather than flickering through each one as it lands.
  async function importNote(title, content) {
    const note = await notesApi.create(title, content)
    state.notes.unshift(note)
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
      await fetchTags()
    } catch (err) {
      if (err instanceof ConflictError) {
        state.conflict = err.current
      } else {
        state.error = err.message ?? 'Failed to save note'
      }
    }
  }

  // remove deletes any note by id, not just the active one — the sidebar's per-row delete button
  // can target a note the user hasn't opened. If the deleted note was the active one, selection
  // falls back to whatever's now first; otherwise the current selection is left alone.
  async function remove(id) {
    await notesApi.remove(id)
    state.notes = state.notes.filter((n) => n.id !== id)
    if (state.activeId === id) {
      state.activeId = state.notes[0]?.id ?? null
    }
  }

  return { state, active, fetchNotes, fetchTags, select, createNote, importNote, saveActive, remove }
}

export const notesStore = createNotesStore()
