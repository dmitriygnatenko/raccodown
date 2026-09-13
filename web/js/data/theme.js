// Explicit light/dark theme, layered on top of the `prefers-color-scheme` support already in
// css/style.css: with no stored preference the app just follows the OS (via the `[data-theme]`
// guards in the CSS — see the top of style.css), and picking a theme here pins it, overriding the
// OS, until cleared.
//
// This module is just the local mechanism (apply + remember in this browser); for a signed-in user
// it's also persisted server-side in User.Settings.Theme, by analogy with raccounting — see
// store/auth.js's changeTheme/applySettings, which call setTheme here once the backend confirms it.
import { reactive, ref } from 'vue'

const STORAGE_KEY = 'raccodown.theme'
const SUPPORTED_THEMES = ['light', 'dark']

function getStoredTheme() {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    return SUPPORTED_THEMES.includes(v) ? v : null
  } catch {
    return null
  }
}

const media = window.matchMedia('(prefers-color-scheme: dark)')
const systemIsDark = ref(media.matches)
media.addEventListener('change', (e) => {
  systemIsDark.value = e.matches
})

// theme is the user's explicit override, or null to follow the OS.
export const themeStore = reactive({ theme: getStoredTheme() })

// effectiveTheme is what's actually on screen right now.
export function effectiveTheme() {
  return themeStore.theme ?? (systemIsDark.value ? 'dark' : 'light')
}

function applyTheme(theme) {
  if (theme) document.documentElement.dataset.theme = theme
  else delete document.documentElement.dataset.theme
}

export function setTheme(theme) {
  themeStore.theme = theme
  try {
    if (theme) localStorage.setItem(STORAGE_KEY, theme)
    else localStorage.removeItem(STORAGE_KEY)
  } catch {
    // Private browsing / storage disabled — the choice just won't survive a reload.
  }
  applyTheme(theme)
}

// Reflect whatever the inline snippet in index.html's <head> already applied (it runs before this
// module loads, to avoid a flash of the wrong theme) — or the OS default if it didn't set anything.
applyTheme(themeStore.theme)
