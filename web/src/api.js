// Thin fetch wrapper around the panel API. All mutating requests carry the
// X-Panel-CSRF header that the backend requires (which, with SameSite=Strict
// cookies, blocks CSRF). Errors surface as ApiError with the server message.

export class ApiError extends Error {
  constructor(status, message) {
    super(message || `请求失败 (${status})`)
    this.status = status
  }
}

async function req(method, path, body) {
  const opts = { method, credentials: 'same-origin', headers: {} }
  if (method !== 'GET' && method !== 'HEAD') {
    opts.headers['X-Panel-CSRF'] = '1'
  }
  if (body !== undefined) {
    opts.headers['Content-Type'] = 'application/json'
    opts.body = JSON.stringify(body)
  }
  let res
  try {
    res = await fetch('/api' + path, opts)
  } catch (e) {
    throw new ApiError(0, '无法连接到面板服务')
  }
  if (!res.ok) {
    let msg = ''
    try {
      msg = (await res.json()).error
    } catch {
      /* ignore */
    }
    throw new ApiError(res.status, msg)
  }
  if (res.status === 204) return null
  const ct = res.headers.get('content-type') || ''
  return ct.includes('json') ? res.json() : res.text()
}

export const api = {
  // bootstrap & auth
  state: () => req('GET', '/state'),
  login: (password) => req('POST', '/login', { password }),
  logout: () => req('POST', '/logout'),

  // setup
  platforms: () => req('GET', '/setup/platforms'),
  install: (payload) => req('POST', '/setup/install', payload),
  cancelInstall: () => req('POST', '/setup/cancel'),
  finalize: (role, password) => req('POST', '/setup/finalize', { role, password }),

  // config
  getConfig: () => req('GET', '/config'),
  saveConfig: (payload) => req('PUT', '/config', payload),
  saveRaw: (raw) => req('PUT', '/config/raw', { raw }),

  // frp control
  frpStatus: () => req('GET', '/frp/status'),
  frpStart: () => req('POST', '/frp/start'),
  frpStop: () => req('POST', '/frp/stop'),
  frpRestart: () => req('POST', '/frp/restart'),
  frpReload: () => req('POST', '/frp/reload'),

  // logs
  logs: () => req('GET', '/logs'),
  clearLogs: () => req('DELETE', '/logs'),

  // monitor
  monitor: () => req('GET', '/monitor'),

  // updates
  checkUpdate: () => req('GET', '/update/check'),
  performUpdate: (version) => req('POST', '/update/perform', { version: version || '' }),
  cancelUpdate: () => req('POST', '/update/cancel'),

  // settings & account
  settings: () => req('GET', '/settings'),
  saveSettings: (s) => req('PUT', '/settings', s),
  changePassword: (current, next) => req('POST', '/account/password', { current, new: next }),
}

// SSE endpoint paths (consumed via EventSource, cookies sent automatically).
export const sse = {
  setupProgress: '/api/setup/progress',
  progress: '/api/progress',
  logs: '/api/logs/stream',
}
