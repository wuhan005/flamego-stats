const FLAMEGO_PREFIX = 'github.com/flamego/'

/** The middleware documented on flamego.dev, the other modules link to their GitHub repositories. */
const DOCUMENTED_MIDDLEWARE = new Set([
  'template',
  'session',
  'cache',
  'binding',
  'csrf',
  'cors',
  'gzip',
  'brotli',
  'i18n',
  'auth',
  'recaptcha',
  'hcaptcha',
  'captcha',
  'sse',
])

/** Returns the short name of the module, e.g. session of github.com/flamego/session. */
export function moduleShortName(path: string): string {
  const name = path.startsWith(FLAMEGO_PREFIX) ? path.slice(FLAMEGO_PREFIX.length) : path
  return name.replace(/\/v\d+$/, '')
}

const PSEUDO_VERSION = /^(v\d+\.\d+\.\d+)-(?:.+\.)?\d{14}-([0-9a-f]{12})$/

/** Abbreviates a Go pseudo-version, e.g. v0.0.0-20211002062834-f84595763ddb to v0.0.0-…-f845957, the other versions are returned as is. */
export function shortVersion(version: string): string {
  const match = PSEUDO_VERSION.exec(version)
  return match ? `${match[1]}-…-${match[2].slice(0, 7)}` : version
}

/** Returns the documentation URL of the module, or its source repository. */
export function moduleURL(path: string): string {
  const name = moduleShortName(path)
  if (path.startsWith(FLAMEGO_PREFIX) && DOCUMENTED_MIDDLEWARE.has(name)) {
    return `https://flamego.dev/middleware/${name}/`
  }
  const repo = path.replace(/\/v\d+$/, '')
  return repo.startsWith('github.com/') ? `https://${repo}` : `https://pkg.go.dev/${path}`
}
