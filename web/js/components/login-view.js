import { authStore } from '../store/auth.js'
import { t } from '../data/i18n.js'

export default {
  template: `
    <div class="login-screen">
      <div class="login-wrap">
        <div class="login-outer-brand">Raccodown</div>

        <form class="login-card" @submit.prevent="submit">
          <h1 class="login-title">{{ t('Вход в аккаунт') }}</h1>

          <p v-if="store.error" class="login-error">{{ store.error }}</p>

          <label class="login-field">
            <span>{{ t('Логин') }}</span>
            <input v-model="username" type="text" autocomplete="username" required />
          </label>

          <label class="login-field">
            <span>{{ t('Пароль') }}</span>
            <input v-model="password" type="password" autocomplete="current-password" required />
          </label>

          <button class="login-submit" type="submit" :disabled="submitting">
            {{ submitting ? t('Вход…') : t('Войти') }}
          </button>
        </form>
      </div>
    </div>
  `,
  data() {
    return {
      store: authStore.state,
      username: '',
      password: '',
      submitting: false,
    }
  },
  methods: {
    t,
    async submit() {
      this.submitting = true
      await authStore.login(this.username, this.password)
      this.submitting = false
    },
  },
}
