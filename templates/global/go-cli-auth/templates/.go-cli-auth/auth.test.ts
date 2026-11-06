{{- if .Computed.oauth }}
import { createHash } from 'node:crypto'
import { mkdirSync, statSync, writeFileSync } from 'node:fs'
import { createServer, type IncomingMessage } from 'node:http'
import type { AddressInfo } from 'node:net'
import { join } from 'node:path'
import { describe, expect, it, onTestFinished } from 'vitest'
import { runCli, scratchDir, type Result } from '../testUtils'
{{- else if .Computed.stored }}
import { mkdirSync, statSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'
import { getExecutablePath, runCli, scratchDir, type Result } from '../testUtils'
{{- else }}
import { mkdirSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { describe, expect, it } from 'vitest'
import { runCli, scratchDir, type Result } from '../testUtils'
{{- end }}

// without these variables, a test reads and writes the config and the keychain of the person who runs it
function isolated(extra: Record<string, string> = {}): Record<string, string> {
  return {
    XDG_CONFIG_HOME: scratchDir(),
    GO_CLI_PROJECT_{{ .Computed.variable }}: '',
{{- if .Computed.stored }}
    GO_CLI_PROJECT_API_URL: 'https://api.example.test',
    GO_CLI_PROJECT_CREDENTIAL_STORE: 'file',
{{- end }}
    ...extra,
  }
}

function configDir(env: Record<string, string>): string {
  return join(env.XDG_CONFIG_HOME, 'go-cli-project')
}

function writeConfig(env: Record<string, string>, text: string): string {
  mkdirSync(configDir(env), { recursive: true })
  const path = join(configDir(env), 'config.yaml')
  writeFileSync(path, text)
  return path
}

function output(result: Result): string {
  return result.stdout + result.stderr
}

function status(env: Record<string, string>): Promise<Result> {
  return runCli(scratchDir(), ['auth', 'status'], env)
}

describe('go-cli-project auth status', () => {
  it('names the environment variable that gives the {{ .Computed.label }}, and does not print the {{ .Computed.label }}', async () => {
    const result = await status(isolated({ GO_CLI_PROJECT_{{ .Computed.variable }}: 'secret-value' }))

    expect(result.status).toBe(0)
    expect(result.stdout).toBe('The {{ .Computed.label }} comes from GO_CLI_PROJECT_{{ .Computed.variable }}.\n')
  })

  it('runs the credentialCommand in the user config', async () => {
    const env = isolated()
    const path = writeConfig(env, 'credentialCommand: echo from-command\n')

    const result = await status(env)

    expect(result.status).toBe(0)
    expect(result.stdout).toBe(`The {{ .Computed.label }} comes from the credentialCommand in ${path}.\n`)
  })

  it('shows what a failed credentialCommand writes to stderr', async () => {
    const env = isolated()
    const path = writeConfig(env, "credentialCommand: echo 'the vault is locked' >&2; exit 1\n")

    const result = await status(env)

    expect(result.status).toBe(1)
    expect(result.stderr).toContain('the vault is locked')
    expect(output(result)).toContain(`run the credentialCommand in ${path}`)
  })

  it('fails and says how to give the {{ .Computed.label }} when there is none', async () => {
    const result = await status(isolated())

    expect(result.status).toBe(1)
    expect(output(result)).toContain('GO_CLI_PROJECT_{{ .Computed.variable }}')
    expect(output(result)).toContain('credentialCommand')
{{- if .Computed.stored }}
    expect(output(result)).toContain('"go-cli-project auth login"')
{{- end }}
  })
})
{{- if and .Computed.stored (not .Computed.oauth) }}

function login(env: Record<string, string>, key: string): Promise<Result> {
  return runCli(scratchDir(), ['-c', `printf '%s' '${key}' | "${getExecutablePath()}" auth login`], env, 'sh')
}

describe('go-cli-project auth login', () => {
  it('stores the API key that it reads from stdin in a file that only the user can read', async () => {
    const env = isolated()
    const file = join(configDir(env), 'credentials.json')

    const result = await login(env, 'stored-key')

    expect(result.status).toBe(0)
    expect(result.stdout).toBe(`Stored the API key for api.example.test in ${file}.\n`)
    expect(statSync(file).mode & 0o777).toBe(0o600)
    expect((await status(env)).stdout).toBe(`The API key comes from ${file}.\n`)
  })

  it('lets the environment variable win over the stored API key', async () => {
    const env = isolated()
    await login(env, 'stored-key')

    const result = await status({ ...env, GO_CLI_PROJECT_API_KEY: 'from-environment' })

    expect(result.stdout).toBe('The API key comes from GO_CLI_PROJECT_API_KEY.\n')
  })

  it('does not give the stored API key to another API host', async () => {
    const env = isolated()
    await login(env, 'stored-key')

    const result = await status({ ...env, GO_CLI_PROJECT_API_URL: 'https://other.example.test' })

    expect(result.status).toBe(1)
  })
})

describe('go-cli-project auth logout', () => {
  it('removes the stored API key', async () => {
    const env = isolated()
    await login(env, 'stored-key')

    const result = await runCli(scratchDir(), ['auth', 'logout'], env)

    expect(result.status).toBe(0)
    expect(result.stdout).toBe('Removed the API key for api.example.test.\n')
    expect((await status(env)).status).toBe(1)
  })

  it('says so when no API key is stored', async () => {
    const result = await runCli(scratchDir(), ['auth', 'logout'], isolated())

    expect(result.status).toBe(0)
    expect(result.stdout).toBe('No API key for api.example.test is stored.\n')
  })
})
{{- end }}
{{- if .Computed.oauth }}

interface FakeIssuer {
  url: string
  expiresIn: number
  verifiedLogins: number
  revoked: string[]
}

function body(request: IncomingMessage): Promise<URLSearchParams> {
  return new Promise((done) => {
    let text = ''
    request.on('data', (chunk) => (text += chunk))
    request.on('end', () => done(new URLSearchParams(text)))
  })
}

async function fakeIssuer(): Promise<FakeIssuer> {
  const issuer: FakeIssuer = { url: '', expiresIn: 3600, verifiedLogins: 0, revoked: [] }
  const challenges = new Map<string, string>()
  let issued = 0
  const claims = Buffer.from(JSON.stringify({ sub: 'user-1', email: 'ada@example.com' })).toString('base64url')

  const server = createServer(async (request, response) => {
    const url = new URL(request.url ?? '/', issuer.url)
    const form = await body(request)
    const json = (status: number, value: unknown) => {
      response.statusCode = status
      response.setHeader('content-type', 'application/json')
      response.end(JSON.stringify(value))
    }
    const tokens = () => {
      issued += 1
      json(200, {
        access_token: `access-${issued}`,
        refresh_token: `refresh-${issued}`,
        token_type: 'Bearer',
        expires_in: issuer.expiresIn,
        id_token: `eyJhbGciOiJub25lIn0.${claims}.`,
      })
    }
    switch (url.pathname) {
      case '/.well-known/openid-configuration':
        return json(200, {
          issuer: issuer.url,
          authorization_endpoint: `${issuer.url}/authorize`,
          token_endpoint: `${issuer.url}/token`,
          device_authorization_endpoint: `${issuer.url}/device`,
          revocation_endpoint: `${issuer.url}/revoke`,
        })
      case '/authorize': {
        const code = `code-${challenges.size + 1}`
        challenges.set(code, url.searchParams.get('code_challenge') ?? '')
        const redirect = new URL(url.searchParams.get('redirect_uri') ?? '')
        redirect.searchParams.set('code', code)
        redirect.searchParams.set('state', url.searchParams.get('state') ?? '')
        response.writeHead(302, { location: redirect.toString() })
        return response.end()
      }
      case '/device':
        return json(200, {
          device_code: 'device-1',
          user_code: 'ABCD-EFGH',
          verification_uri: `${issuer.url}/activate`,
          expires_in: 600,
          interval: 1,
        })
      case '/token':
        switch (form.get('grant_type')) {
          case 'authorization_code': {
            const challenge = createHash('sha256').update(form.get('code_verifier') ?? '').digest('base64url')
            if (challenges.get(form.get('code') ?? '') !== challenge) return json(400, { error: 'invalid_grant' })
            issuer.verifiedLogins += 1
            return tokens()
          }
          case 'urn:ietf:params:oauth:grant-type:device_code':
            return tokens()
          case 'refresh_token':
            if (form.get('refresh_token') !== `refresh-${issued}`) return json(400, { error: 'invalid_grant' })
            return tokens()
        }
        return json(400, { error: 'unsupported_grant_type' })
      case '/revoke':
        issuer.revoked.push(form.get('token') ?? '')
        return response.end()
    }
    response.statusCode = 404
    response.end()
  })
  await new Promise<void>((done) => server.listen(0, '127.0.0.1', done))
  onTestFinished(() => new Promise<void>((done) => server.close(() => done())))
  issuer.url = `http://127.0.0.1:${(server.address() as AddressInfo).port}`
  return issuer
}

function withIssuer(issuer: FakeIssuer, extra: Record<string, string> = {}): Record<string, string> {
  return isolated({
    GO_CLI_PROJECT_OAUTH_ISSUER_URL: issuer.url,
    GO_CLI_PROJECT_OAUTH_CLIENT_ID: 'test-client',
    BROWSER: '',
    SSH_CONNECTION: '',
    SSH_TTY: '',
    ...extra,
  })
}

function fakeBrowser(): string {
  const script = join(scratchDir(), 'browser.mjs')
  writeFileSync(script, 'await fetch(process.argv[2])\n')
  return `node ${script}`
}

function login(env: Record<string, string>, args: string[] = ['--device']): Promise<Result> {
  return runCli(scratchDir(), ['auth', 'login', ...args], env)
}

describe('go-cli-project auth login', () => {
  it('--device prints the code to enter, and auth status shows the account', async () => {
    const env = withIssuer(await fakeIssuer())
    const file = join(configDir(env), 'credentials.json')

    const result = await login(env)

    expect(result.status).toBe(0)
    expect(result.stderr).toContain('enter the code ABCD-EFGH')
    expect(result.stdout).toBe(`Logged in as ada@example.com. The tokens are in ${file}.\n`)
    expect(statSync(file).mode & 0o777).toBe(0o600)
    const shown = await status(env)
    expect(shown.stdout).toContain(`The token comes from ${file}.\nAccount: ada@example.com\n`)
  })

  it('opens the browser, and the issuer gets the PKCE code verifier', async () => {
    const issuer = await fakeIssuer()
    const env = withIssuer(issuer, { BROWSER: fakeBrowser() })

    const result = await login(env, [])

    expect(result.status).toBe(0)
    expect(result.stdout).toContain('Logged in as ada@example.com.')
    expect(issuer.verifiedLogins).toBe(1)
  })

  it('without the issuer settings, names the constants to set', async () => {
    const result = await login(isolated())

    expect(result.status).toBe(1)
    expect(output(result)).toContain('set issuerURL and clientID in internal/credential/issuer.go')
  })
})

describe('go-cli-project auth token', () => {
  it('prints the access token, and gets a new one when it expires', async () => {
    const issuer = await fakeIssuer()
    const env = withIssuer(issuer)
    issuer.expiresIn = 1
    await login(env)
    issuer.expiresIn = 3600

    const result = await runCli(scratchDir(), ['auth', 'token'], env)

    expect(result.status).toBe(0)
    expect(result.stdout).toBe('access-2\n')
  })
})

describe('go-cli-project auth logout', () => {
  it('revokes the refresh token and removes the tokens', async () => {
    const issuer = await fakeIssuer()
    const env = withIssuer(issuer)
    await login(env)

    const result = await runCli(scratchDir(), ['auth', 'logout'], env)

    expect(result.status).toBe(0)
    expect(result.stdout).toBe('Removed the tokens for api.example.test.\n')
    expect(issuer.revoked).toEqual(['refresh-1'])
    expect((await status(env)).status).toBe(1)
  })
})
{{- end }}
