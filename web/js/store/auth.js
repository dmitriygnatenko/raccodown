import { reactive } from 'vue'
import { authApi, settingsApi } from '../data/api.js'
import { i18nStore, t, setLanguage } from '../data/i18n.js'
import { effectiveTheme, setTheme } from '../data/theme.js'

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

  // The backend is the source of truth for the user's saved language/theme once either is set (see
  // login.UseCase in the Go backend) — apply whatever it reports so the UI matches it, even if that
  // differs from what's currently in localStorage.
  function applySettings(user) {
    const settings = user?.settings
    if (settings?.language) setLanguage(settings.language)
    if (settings?.theme) setTheme(settings.theme)
  }

  // checkSession restores a session after a page reload (the session cookie is still there; the
  // frontend just doesn't know it yet).
  async function checkSession() {
    try {
      state.user = await authApi.me()
      applySettings(state.user)
    } catch {
      state.user = null
    } finally {
      state.checking = false
    }
  }

  async function login(username, password) {
    state.error = null
    try {
      state.user = await authApi.login(username, password, i18nStore.language)
      applySettings(state.user)
      return true
    } catch {
      // The backend's own message (validation details, "invalid credentials", ...) is never shown
      // verbatim — it's in English and can reveal which half of the pair was wrong. One generic,
      // translated message covers every failure reason, the same as raccounting's login view.
      state.error = t('Не удалось войти. Проверьте имя пользователя и пароль.')
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

  // Changing the language in-app persists it to the backend first, then applies whatever it echoes
  // back — the same round-trip login/checkSession use, rather than optimistically switching the UI
  // before the save is confirmed. settingsApi.update overwrites the whole settings blob, so the
  // current (effective) theme rides along unchanged.
  async function changeLanguage(language) {
    const settings = await settingsApi.update(language, effectiveTheme())
    if (state.user) state.user = { ...state.user, settings }
    applySettings({ settings })
  }

  // Same round-trip as changeLanguage, for the theme.
  async function changeTheme(theme) {
    const settings = await settingsApi.update(i18nStore.language, theme)
    if (state.user) state.user = { ...state.user, settings }
    applySettings({ settings })
  }

  // changeCredentials changes the signed-in user's username and/or password, after confirming their
  // current password — newUsername/newPassword are optional, a blank one leaves that field
  // unchanged (see updatecredentials.UseCase in the Go backend). Unlike login's failure message,
  // errors here are shown to the caller verbatim: this is a form the user is actively filling in,
  // not a login attempt where "wrong password" vs. "wrong username" shouldn't be distinguishable.
  async function changeCredentials(currentPassword, newUsername, newPassword) {
    state.user = await authApi.updateCredentials(currentPassword, newUsername, newPassword)
    return state.user
  }

  return {
    state,
    isAuthenticated,
    checkSession,
    login,
    logout,
    changeLanguage,
    changeTheme,
    changeCredentials,
  }
}

export const authStore = createAuthStore()
