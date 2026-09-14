import { authStore } from '../store/auth.js'
import { ApiError } from '../data/api.js'
import { t } from '../data/i18n.js'

// errorMessage maps the backend's HTTP status for this endpoint to a translated message — the
// backend's own text is always in English (see README), so it's never shown verbatim here.
function errorMessage(err) {
  if (err instanceof ApiError) {
    if (err.status === 401) return t('Неверный текущий пароль')
    if (err.status === 409) return t('Такой логин уже занят')
    if (err.status === 400) return t('Проверьте новый логин и новый пароль')
  }

  return t('Не удалось сохранить изменения')
}

// CredentialsModal lets the signed-in user change their username and/or password (backend:
// PATCH /api/v1/auth/credentials). Opened from note-list.js's user menu; @close is emitted on
// cancel, backdrop click, Escape, or a successful save.
export default {
  emits: ['close'],
  template: `
    <div class="modal-backdrop" @click.self="close" @keydown.esc="close">
      <form class="modal-card" @submit.prevent="submit">
        <h2 class="modal-title">{{ t('Изменить логин и пароль') }}</h2>

        <p v-if="error" class="login-error">{{ error }}</p>

        <label class="login-field">
          <span>{{ t('Текущий пароль') }}</span>
          <input v-model="currentPassword" type="password" autocomplete="current-password" required autofocus />
        </label>

        <label class="login-field">
          <span>{{ t('Новый логин') }}</span>
          <input v-model="newUsername" type="text" autocomplete="username" minlength="3" maxlength="255" />
        </label>

        <label class="login-field">
          <span>{{ t('Новый пароль') }}</span>
          <input v-model="newPassword" type="password" autocomplete="new-password" minlength="4" :placeholder="t('Оставьте пустым, чтобы не менять')" />
        </label>

        <div class="modal-actions">
          <button type="button" class="modal-cancel" @click="close">{{ t('Отмена') }}</button>
          <button type="submit" class="login-submit" :disabled="submitting">
            {{ submitting ? t('Сохранение…') : t('Сохранить') }}
          </button>
        </div>
      </form>
    </div>
  `,
  data() {
    return {
      currentPassword: '',
      newUsername: authStore.state.user?.username ?? '',
      newPassword: '',
      error: null,
      submitting: false,
    }
  },
  methods: {
    t,
    close() {
      this.$emit('close')
    },
    async submit() {
      this.error = null
      this.submitting = true
      try {
        await authStore.changeCredentials(this.currentPassword, this.newUsername, this.newPassword)
        this.close()
      } catch (err) {
        this.error = errorMessage(err)
      } finally {
        this.submitting = false
      }
    },
  },
}
