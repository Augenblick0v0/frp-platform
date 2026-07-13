export class ApiError extends Error {
  constructor(message, status, payload) {
    super(message || `HTTP ${status}`);
    this.name = 'ApiError';
    this.status = status;
    this.payload = payload;
  }
}

export function unwrapResponse(json) {
  if (json && typeof json === 'object' && Object.prototype.hasOwnProperty.call(json, 'success')) {
    if (json.success === false) {
      throw new ApiError(json.message || json.code || '请求失败', 400, json);
    }
    return json.data;
  }
  return json;
}

export class ApiClient {
  constructor({ baseURL = '', tokenKey = 'token', tokenPrefix = 'Bearer', local = false, localTokenKey = '' } = {}) {
    this.baseURL = String(baseURL || '').replace(/\/$/, '');
    this.tokenKey = tokenKey;
    this.tokenPrefix = tokenPrefix;
    this.local = local;
    this.localTokenKey = localTokenKey;
  }

  token() {
    try { return sessionStorage.getItem(this.tokenKey) || ''; } catch { return ''; }
  }

  setToken(token) {
    try {
      if (token) sessionStorage.setItem(this.tokenKey, token);
      else sessionStorage.removeItem(this.tokenKey);
      localStorage.removeItem(this.tokenKey);
    } catch {}
  }

  localToken() {
    if (!this.localTokenKey) return '';
    try { return sessionStorage.getItem(this.localTokenKey) || ''; } catch { return ''; }
  }

  async request(path, options = {}) {
    const headers = { ...(options.headers || {}) };
    const hasBody = options.body !== undefined && options.body !== null;
    if (hasBody && !(options.body instanceof FormData)) headers['Content-Type'] = headers['Content-Type'] || 'application/json';
    const token = options.token ?? this.token();
    if (token && !headers.Authorization) headers.Authorization = `${this.tokenPrefix} ${token}`;
    const localToken = options.localToken ?? this.localToken();
    if (localToken && !headers['X-Local-Token']) headers['X-Local-Token'] = localToken;
    const body = hasBody && typeof options.body !== 'string' && !(options.body instanceof FormData)
      ? JSON.stringify(options.body)
      : options.body;
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), options.timeoutMs || 30000);
    let res;
    try {
      res = await fetch(`${this.baseURL}${path}`, { ...options, headers, body, signal: options.signal || controller.signal });
    } catch (err) {
      if (err?.name === 'AbortError') throw new ApiError('请求超时，请稍后重试', 408);
      throw err;
    } finally {
      clearTimeout(timeout);
    }
    const text = await res.text();
    let json = null;
    if (text) {
      try { json = JSON.parse(text); } catch { json = { success: res.ok, data: text, message: text }; }
    }
    if (!res.ok) {
      if (res.status === 401) this.setToken('');
      throw new ApiError(json?.message || res.statusText, res.status, json);
    }
    return unwrapResponse(json ?? { success: true, data: null });
  }

  get(path, options) { return this.request(path, { ...(options || {}), method: 'GET' }); }
  post(path, body, options) { return this.request(path, { ...(options || {}), method: 'POST', body }); }
  put(path, body, options) { return this.request(path, { ...(options || {}), method: 'PUT', body }); }
  delete(path, body, options) { return this.request(path, { ...(options || {}), method: 'DELETE', body }); }
}

export function formatBytes(value) {
  const n = Number(value) || 0;
  if (n <= 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  let i = 0;
  let v = n;
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i += 1; }
  return `${i === 0 ? v.toFixed(0) : v.toFixed(1)} ${units[i]}`;
}

export function formatMbps(kbps) {
  const n = Number(kbps) || 0;
  if (n <= 0) return '不限速';
  return `${(n / 1000).toFixed(2)} Mbps`;
}

export function formatMoney(cents) {
  return `¥${((Number(cents) || 0) / 100).toFixed(2)}`;
}

export function formatTime(value) {
  if (!value) return '-';
  const d = new Date(value);
  if (Number.isNaN(d.getTime())) return String(value);
  return d.toLocaleString('zh-CN', { hour12: false });
}

export function percent(used, total) {
  const u = Number(used) || 0;
  const t = Number(total) || 0;
  if (t <= 0) return 0;
  return Math.max(0, Math.min(100, Math.round((u / t) * 100)));
}

export const api = new ApiClient({ baseURL: '' });
