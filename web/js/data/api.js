// Real API layer — talks to the Go backend under /api/v1 over same-origin fetch(). Same origin
// because this frontend is served by that same Go binary (see web/webassets.go); there's no
// cross-origin dev server to worry about.

export class ApiError extends Error {
  constructor(status, message, body) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.body = body
  }
}

// ConflictError is thrown by notesApi.update on a 409: the note changed since Note.checksum was
// last read. current is the note's present state, for the caller to offer a merge.
export class ConflictError extends Error {
  constructor(current) {
    super('Note changed since last read')
    this.name = 'ConflictError'
    this.current = current
  }
}

async function request(method, path, { body, headers } = {}) {
  const init = { method, headers: { ...headers } }
  if (body !== undefined) {
    init.headers['Content-Type'] = 'application/json'
    init.body = JSON.stringify(body)
  }

  const res = await fetch(path, init)

  if (res.status === 204) return { data: undefined, response: res }

  const text = await res.text()
  let data
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      data = undefined
    }
  }

  if (!res.ok) {
    throw new ApiError(res.status, data?.error ?? res.statusText, data)
  }

  return { data, response: res }
}

function toQueryString(params) {
  const search = new URLSearchParams()
  if (params.q) search.set('q', params.q)
  if (params.tag) search.set('tag', params.tag)
  const qs = search.toString()
  return qs ? `?${qs}` : ''
}

export const authApi = {
  async login(username, password) {
    const { data } = await request('POST', '/api/v1/auth/login', { body: { username, password } })
    return data
  },

  async logout() {
    await request('POST', '/api/v1/auth/logout')
  },

  // me throws ApiError with status 401 if there's no valid session — the caller (authStore) treats
  // that as "show the login screen" rather than an error to surface.
  async me() {
    const { data } = await request('GET', '/api/v1/auth/me')
    return data
  },

  async updateCredentials(currentPassword, newUsername, newPassword) {
    const { data } = await request('PATCH', '/api/v1/auth/credentials', {
      body: { currentPassword, newUsername, newPassword },
    })
    return data
  },
}

export const notesApi = {
  async list(params = {}) {
    const { data } = await request('GET', `/api/v1/notes${toQueryString(params)}`)
    return data
  },

  async get(id) {
    const { data } = await request('GET', `/api/v1/notes/${id}`)
    return data
  },

  async create(title, content) {
    const { data } = await request('POST', '/api/v1/notes', { body: { title, content } })
    return data
  },

  async update(id, title, content, knownChecksum) {
    try {
      const { data } = await request('PUT', `/api/v1/notes/${id}`, {
        body: { title, content },
        headers: { 'If-Match': knownChecksum },
      })
      return data
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        throw new ConflictError(err.body.current)
      }
      throw err
    }
  },

  async remove(id) {
    await request('DELETE', `/api/v1/notes/${id}`)
  },

  async tags() {
    const { data } = await request('GET', '/api/v1/tags')
    return data
  },
}
