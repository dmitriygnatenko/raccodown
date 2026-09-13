import { authStore } from '../store/auth.js'
import { i18nStore, t, setLanguage, SUPPORTED_LANGUAGES, LANGUAGE_LABELS } from '../data/i18n.js'

export default {
  template: `
    <div class="login-screen">
      <form class="login-card" @submit.prevent="submit">
        <div class="login-brand">🦝 Raccodown</div>

        <label class="login-field">
          <span>{{ t('Логин') }}</span>
          <input v-model="username" type="text" autocomplete="username" required autofocus />
        </label>

        <label class="login-field">
          <span>{{ t('Пароль') }}</span>
          <input v-model="password" type="password" autocomplete="current-password" required />
        </label>

        <p v-if="store.error" class="login-error">{{ store.error }}</p>

        <button class="login-submit" type="submit" :disabled="submitting">
          {{ submitting ? t('Вход…') : t('Войти') }}
        </button>

        <select class="login-language" :aria-label="t('Язык')" :value="i18nStore.language" @change="setLanguage($event.target.value)">
          <option v-for="lang in languages" :key="lang" :value="lang">{{ languageLabels[lang] }}</option>
        </select>
      </form>
    </div>
  `,
  data() {
    return {
      store: authStore.state,
      i18nStore,
      languages: SUPPORTED_LANGUAGES,
      languageLabels: LANGUAGE_LABELS,
      username: '',
      password: '',
      submitting: false,
    }
  },
  methods: {
    t,
    setLanguage,
    async submit() {
      this.submitting = true
      await authStore.login(this.username, this.password)
      this.submitting = false
    },
  },
}
