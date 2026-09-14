// Internationalization, by analogy with raccounting's: Russian is the language the frontend is
// actually written in — every UI string is a Russian literal at its call site, passed through t()
// — and the other four languages are exact-match translation tables below, keyed by that Russian
// source string. A string missing from a table simply falls back to its Russian source text rather
// than breaking. A handful of entries carry {placeholders} for runtime-interpolated values.
//
// This module is just the local mechanism (translate + remember in this browser); for a signed-in
// user the choice is also persisted server-side in User.Settings.Language, by analogy with
// raccounting — see store/auth.js's login/changeLanguage/applySettings, which call setLanguage here
// once the backend confirms it.
import { reactive } from 'vue'

export const SUPPORTED_LANGUAGES = ['ru', 'en', 'es', 'de', 'fr']

export const LANGUAGE_LABELS = {
  ru: 'Русский',
  en: 'English',
  es: 'Español',
  de: 'Deutsch',
  fr: 'Français',
}

const STORAGE_KEY = 'raccodown.language'
const DEFAULT_LANGUAGE = 'ru'

const translations = {
  en: {
    'Вход…': 'Signing in…',
    Войти: 'Log in',
    Правка: 'Edit',
    Оба: 'Both',
    Превью: 'Preview',
    Заголовок: 'Heading',
    'Жирный (Ctrl/Cmd+B)': 'Bold (Ctrl/Cmd+B)',
    'Курсив (Ctrl/Cmd+I)': 'Italic (Ctrl/Cmd+I)',
    Зачёркнутый: 'Strikethrough',
    'Блок кода': 'Code block',
    Ссылка: 'Link',
    Цитата: 'Quote',
    Список: 'List',
    'Нумерованный список': 'Numbered list',
    Разделитель: 'Horizontal rule',
    Таблица: 'Table',
    'Заголовок 1': 'Header 1',
    'Заголовок 2': 'Header 2',
    Ячейка: 'Cell',
    'Пишите в Markdown…': 'Write in Markdown…',
    текст: 'text',
    'Не удалось войти. Проверьте имя пользователя и пароль.': 'Failed to log in. Check your username and password.',
    'Без названия': 'Untitled',
    Удалить: 'Delete',
    'Сохранение…': 'Saving…',
    Сохранено: 'Saved',
    'Удалить заметку?': 'Delete note?',
    'Заметка «{title}» будет удалена без возможности восстановления.': 'The note "{title}" will be permanently deleted.',
    'Удаление…': 'Deleting…',
    'Новая заметка': 'New note',
    'Добавить заметку': 'Add note',
    'Импортировать .md/.txt файлы как заметки': 'Import .md/.txt files as notes',
    'Скачать все заметки одним zip-архивом': 'Download all notes as one zip archive',
    'Поиск заметок…': 'Search notes…',
    'Светлая тема': 'Light theme',
    'Тёмная тема': 'Dark theme',
    'Не удалось импортировать файл': 'Failed to import file',
    'Заметка была изменена в другом месте.': 'The note was changed elsewhere.',
    'Оставить мою версию': 'Keep my version',
    'Взять чужую версию': 'Take the other version',
    'Выберите заметку слева или создайте новую.': 'Select a note on the left, or create a new one.',
    Логин: 'Username',
    Пароль: 'Password',
    'Вход в аккаунт': 'Sign in',
    'Загрузка…': 'Loading…',
    'Ничего не найдено': 'Nothing found',
    Выйти: 'Log out',
    Импорт: 'Import',
    Экспорт: 'Export',
    Язык: 'Language',
    'Изменить логин и пароль': 'Change username and password',
    'Текущий пароль': 'Current password',
    'Новый логин': 'New username',
    'Новый пароль': 'New password',
    'Оставьте пустым, чтобы не менять': 'Leave blank to keep it unchanged',
    Отмена: 'Cancel',
    Сохранить: 'Save',
    'Не удалось сохранить изменения': 'Failed to save changes',
    'Неверный текущий пароль': 'Incorrect current password',
    'Такой логин уже занят': 'This username is already taken',
    'Проверьте новый логин и новый пароль': 'Check the new username and new password',
  },
  es: {
    'Вход…': 'Iniciando sesión…',
    Войти: 'Iniciar sesión',
    Правка: 'Editar',
    Оба: 'Ambos',
    Превью: 'Vista previa',
    Заголовок: 'Encabezado',
    'Жирный (Ctrl/Cmd+B)': 'Negrita (Ctrl/Cmd+B)',
    'Курсив (Ctrl/Cmd+I)': 'Cursiva (Ctrl/Cmd+I)',
    Зачёркнутый: 'Tachado',
    'Блок кода': 'Bloque de código',
    Ссылка: 'Enlace',
    Цитата: 'Cita',
    Список: 'Lista',
    'Нумерованный список': 'Lista numerada',
    Разделитель: 'Línea horizontal',
    Таблица: 'Tabla',
    'Заголовок 1': 'Encabezado 1',
    'Заголовок 2': 'Encabezado 2',
    Ячейка: 'Celda',
    'Пишите в Markdown…': 'Escribe en Markdown…',
    текст: 'texto',
    'Не удалось войти. Проверьте имя пользователя и пароль.': 'No se pudo iniciar sesión. Comprueba tu usuario y contraseña.',
    'Без названия': 'Sin título',
    Удалить: 'Eliminar',
    'Сохранение…': 'Guardando…',
    Сохранено: 'Guardado',
    'Удалить заметку?': '¿Eliminar la nota?',
    'Заметка «{title}» будет удалена без возможности восстановления.': 'La nota «{title}» se eliminará de forma permanente.',
    'Удаление…': 'Eliminando…',
    'Новая заметка': 'Nueva nota',
    'Добавить заметку': 'Añadir nota',
    'Импортировать .md/.txt файлы как заметки': 'Importar archivos .md/.txt como notas',
    'Скачать все заметки одним zip-архивом': 'Descargar todas las notas en un archivo zip',
    'Поиск заметок…': 'Buscar notas…',
    'Светлая тема': 'Tema claro',
    'Тёмная тема': 'Tema oscuro',
    'Не удалось импортировать файл': 'No se pudo importar el archivo',
    'Заметка была изменена в другом месте.': 'La nota fue modificada en otro lugar.',
    'Оставить мою версию': 'Conservar mi versión',
    'Взять чужую версию': 'Usar la otra versión',
    'Выберите заметку слева или создайте новую.': 'Selecciona una nota a la izquierda o crea una nueva.',
    Логин: 'Usuario',
    Пароль: 'Contraseña',
    'Вход в аккаунт': 'Iniciar sesión',
    'Загрузка…': 'Cargando…',
    'Ничего не найдено': 'No se encontró nada',
    Выйти: 'Cerrar sesión',
    Импорт: 'Importar',
    Экспорт: 'Exportar',
    Язык: 'Idioma',
    'Изменить логин и пароль': 'Cambiar usuario y contraseña',
    'Текущий пароль': 'Contraseña actual',
    'Новый логин': 'Nuevo usuario',
    'Новый пароль': 'Nueva contraseña',
    'Оставьте пустым, чтобы не менять': 'Déjalo en blanco para no cambiarlo',
    Отмена: 'Cancelar',
    Сохранить: 'Guardar',
    'Не удалось сохранить изменения': 'No se pudieron guardar los cambios',
    'Неверный текущий пароль': 'Contraseña actual incorrecta',
    'Такой логин уже занят': 'Este nombre de usuario ya está en uso',
    'Проверьте новый логин и новый пароль': 'Comprueba el nuevo usuario y la nueva contraseña',
  },
  de: {
    'Вход…': 'Anmeldung…',
    Войти: 'Anmelden',
    Правка: 'Bearbeiten',
    Оба: 'Beide',
    Превью: 'Vorschau',
    Заголовок: 'Überschrift',
    'Жирный (Ctrl/Cmd+B)': 'Fett (Strg/Cmd+B)',
    'Курсив (Ctrl/Cmd+I)': 'Kursiv (Strg/Cmd+I)',
    Зачёркнутый: 'Durchgestrichen',
    'Блок кода': 'Codeblock',
    Ссылка: 'Link',
    Цитата: 'Zitat',
    Список: 'Liste',
    'Нумерованный список': 'Nummerierte Liste',
    Разделитель: 'Trennlinie',
    Таблица: 'Tabelle',
    'Заголовок 1': 'Überschrift 1',
    'Заголовок 2': 'Überschrift 2',
    Ячейка: 'Zelle',
    'Пишите в Markdown…': 'Schreib in Markdown…',
    текст: 'Text',
    'Не удалось войти. Проверьте имя пользователя и пароль.': 'Anmeldung fehlgeschlagen. Prüfe Benutzername und Passwort.',
    'Без названия': 'Ohne Titel',
    Удалить: 'Löschen',
    'Сохранение…': 'Speichern…',
    Сохранено: 'Gespeichert',
    'Удалить заметку?': 'Notiz löschen?',
    'Заметка «{title}» будет удалена без возможности восстановления.': 'Die Notiz „{title}“ wird endgültig gelöscht.',
    'Удаление…': 'Löschen…',
    'Новая заметка': 'Neue Notiz',
    'Добавить заметку': 'Notiz hinzufügen',
    'Импортировать .md/.txt файлы как заметки': '.md/.txt-Dateien als Notizen importieren',
    'Скачать все заметки одним zip-архивом': 'Alle Notizen als ein ZIP-Archiv herunterladen',
    'Поиск заметок…': 'Notizen durchsuchen…',
    'Светлая тема': 'Helles Design',
    'Тёмная тема': 'Dunkles Design',
    'Не удалось импортировать файл': 'Datei konnte nicht importiert werden',
    'Заметка была изменена в другом месте.': 'Die Notiz wurde an anderer Stelle geändert.',
    'Оставить мою версию': 'Meine Version behalten',
    'Взять чужую версию': 'Andere Version übernehmen',
    'Выберите заметку слева или создайте новую.': 'Wähle links eine Notiz aus oder erstelle eine neue.',
    Логин: 'Benutzername',
    Пароль: 'Passwort',
    'Вход в аккаунт': 'Anmelden',
    'Загрузка…': 'Wird geladen…',
    'Ничего не найдено': 'Nichts gefunden',
    Выйти: 'Abmelden',
    Импорт: 'Import',
    Экспорт: 'Export',
    Язык: 'Sprache',
    'Изменить логин и пароль': 'Benutzername und Passwort ändern',
    'Текущий пароль': 'Aktuelles Passwort',
    'Новый логин': 'Neuer Benutzername',
    'Новый пароль': 'Neues Passwort',
    'Оставьте пустым, чтобы не менять': 'Leer lassen, um es nicht zu ändern',
    Отмена: 'Abbrechen',
    Сохранить: 'Speichern',
    'Не удалось сохранить изменения': 'Änderungen konnten nicht gespeichert werden',
    'Неверный текущий пароль': 'Aktuelles Passwort ist falsch',
    'Такой логин уже занят': 'Dieser Benutzername ist bereits vergeben',
    'Проверьте новый логин и новый пароль': 'Prüfe den neuen Benutzernamen und das neue Passwort',
  },
  fr: {
    'Вход…': 'Connexion…',
    Войти: 'Se connecter',
    Правка: 'Modifier',
    Оба: 'Les deux',
    Превью: 'Aperçu',
    Заголовок: 'Titre',
    'Жирный (Ctrl/Cmd+B)': 'Gras (Ctrl/Cmd+B)',
    'Курсив (Ctrl/Cmd+I)': 'Italique (Ctrl/Cmd+I)',
    Зачёркнутый: 'Barré',
    'Блок кода': 'Bloc de code',
    Ссылка: 'Lien',
    Цитата: 'Citation',
    Список: 'Liste',
    'Нумерованный список': 'Liste numérotée',
    Разделитель: 'Ligne horizontale',
    Таблица: 'Tableau',
    'Заголовок 1': 'En-tête 1',
    'Заголовок 2': 'En-tête 2',
    Ячейка: 'Cellule',
    'Пишите в Markdown…': 'Écrivez en Markdown…',
    текст: 'texte',
    'Не удалось войти. Проверьте имя пользователя и пароль.': 'Échec de la connexion. Vérifiez votre identifiant et votre mot de passe.',
    'Без названия': 'Sans titre',
    Удалить: 'Supprimer',
    'Сохранение…': 'Enregistrement…',
    Сохранено: 'Enregistré',
    'Удалить заметку?': 'Supprimer la note ?',
    'Заметка «{title}» будет удалена без возможности восстановления.': 'La note « {title} » sera définitivement supprimée.',
    'Удаление…': 'Suppression…',
    'Новая заметка': 'Nouvelle note',
    'Добавить заметку': 'Ajouter une note',
    'Импортировать .md/.txt файлы как заметки': 'Importer des fichiers .md/.txt comme notes',
    'Скачать все заметки одним zip-архивом': 'Télécharger toutes les notes dans une archive zip',
    'Поиск заметок…': 'Rechercher des notes…',
    'Светлая тема': 'Thème clair',
    'Тёмная тема': 'Thème sombre',
    'Не удалось импортировать файл': "Échec de l'importation du fichier",
    'Заметка была изменена в другом месте.': 'La note a été modifiée ailleurs.',
    'Оставить мою версию': 'Garder ma version',
    'Взять чужую версию': "Prendre l'autre version",
    'Выберите заметку слева или создайте новую.': 'Sélectionnez une note à gauche ou créez-en une nouvelle.',
    Логин: 'Identifiant',
    Пароль: 'Mot de passe',
    'Вход в аккаунт': 'Connexion',
    'Загрузка…': 'Chargement…',
    'Ничего не найдено': 'Aucun résultat',
    Выйти: 'Se déconnecter',
    Импорт: 'Importer',
    Экспорт: 'Exporter',
    Язык: 'Langue',
    'Изменить логин и пароль': "Changer l'identifiant et le mot de passe",
    'Текущий пароль': 'Mot de passe actuel',
    'Новый логин': 'Nouvel identifiant',
    'Новый пароль': 'Nouveau mot de passe',
    'Оставьте пустым, чтобы не менять': 'Laissez vide pour ne pas le changer',
    Отмена: 'Annuler',
    Сохранить: 'Enregistrer',
    'Не удалось сохранить изменения': "Échec de l'enregistrement des modifications",
    'Неверный текущий пароль': 'Mot de passe actuel incorrect',
    'Такой логин уже занят': "Ce nom d'utilisateur est déjà pris",
    'Проверьте новый логин и новый пароль': "Vérifiez le nouvel identifiant et le nouveau mot de passe",
  },
}

function getStoredLanguage() {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    return SUPPORTED_LANGUAGES.includes(v) ? v : null
  } catch {
    return null
  }
}

function detectBrowserLanguage() {
  const langs = navigator.languages || [navigator.language || DEFAULT_LANGUAGE]
  for (const lang of langs) {
    const code = (lang || '').slice(0, 2).toLowerCase()
    if (SUPPORTED_LANGUAGES.includes(code)) return code
  }
  return null
}

export const i18nStore = reactive({ language: getStoredLanguage() ?? detectBrowserLanguage() ?? DEFAULT_LANGUAGE })

export function setLanguage(lang) {
  if (!SUPPORTED_LANGUAGES.includes(lang)) return

  i18nStore.language = lang
  try {
    localStorage.setItem(STORAGE_KEY, lang)
  } catch {
    // Private browsing / storage disabled — the choice just won't survive a reload.
  }
}

// t translates a Russian source string into the current language, falling back to the Russian text
// itself if there's no entry for it (missing key) or the current language is Russian. params fills
// in any {placeholder} the string carries, in either language.
export function t(text, params) {
  const translated = i18nStore.language === 'ru' ? text : (translations[i18nStore.language]?.[text] ?? text)

  if (!params) return translated

  return Object.entries(params).reduce((s, [key, value]) => s.replaceAll(`{${key}}`, value), translated)
}
