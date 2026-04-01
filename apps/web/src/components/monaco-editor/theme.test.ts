import { describe, expect, it } from 'vitest'
import { resolveMonacoTheme } from './theme'

describe('resolveMonacoTheme', () => {
  it('uses the light Monaco theme for non-dark app themes', () => {
    expect(resolveMonacoTheme('light')).toBe('vitesse-light')
    expect(resolveMonacoTheme('deep-space')).toBe('vitesse-light')
  })

  it('uses the dark Monaco theme only for dark mode', () => {
    expect(resolveMonacoTheme('dark')).toBe('vitesse-dark')
  })
})
