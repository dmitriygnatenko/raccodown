import { reactive } from 'vue'
import { authApi, ApiError } from '../data/api.js'
import { t } from '../data/i18n.js'

function createAuthStore() {
  const state = reactive({
    // checking is true only during the initial /auth/me probe on page load, so the app can show
    // neither the login screen nor the note UI for that one instant — showing the login screen
    // first would flash it even for an already-signed-in visitor.
    checking: true,
    user: null,
    error: null,
  })

  function isAuthenticated() {
    return state.user !== null
  }

  // checkSession restores a session after a page reload (the session cookie is still there; the
  // frontend just doesn't know it yet).
  async function checkSession() {
    try {
      state.user = await authApi.me()
    } catch {
      state.user = null
    } finally {
      state.checking = false
    }
  }

  async function login(username, password) {
    state.error = null
    try {
      state.user = await authApi.login(username, password)
      return true
    } catch (err) {
      state.error = err instanceof ApiError ? err.message : t('Не удалось войти')
      return false
    }
  }

  async function logout() {
    try {
      await authApi.logout()
    } finally {
      state.user = null
    }
  }

  return { state, isAuthenticated, checkSession, login, logout }
}

export const authStore = createAuthStore()
