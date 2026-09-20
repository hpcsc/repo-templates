import { describe, expect, it } from 'vitest'
import { runCli, scratchDir } from '../testUtils'

describe('{{.Scaffold.Name}}', () => {
  it('runs the default action when the command has no sub-command', async () => {
    const result = await runCli(scratchDir(), ['{{.Scaffold.Name}}'])

    expect(result.status).toBe(0)
    expect(result.stdout).toContain('default action')
  })

  it('gives each argument to the sub-command that takes a slice', async () => {
    const result = await runCli(scratchDir(), ['{{.Scaffold.Name}}', 'slice-args', 'argument-1', 'argument-2'])

    expect(result.status).toBe(0)
    expect(result.stdout).toContain('slice arguments: argument-1, argument-2')
  })

  it('reports the missing argument of the sub-command that takes one', async () => {
    const result = await runCli(scratchDir(), ['{{.Scaffold.Name}}', 'one-arg'])

    expect(result.status).toBe(1)
    expect(result.stdout).toContain('one argument required')
  })
})
