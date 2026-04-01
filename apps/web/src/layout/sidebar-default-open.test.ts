import { describe, expect, it } from 'vitest'
import { getSidebarDefaultOpen } from './sidebar-default-open'

describe('sidebar default open', () => {
  it('defaults to open when cookie is absent', () => {
    expect(getSidebarDefaultOpen('')).toBe(true)
  })

  it('closes only when sidebar_state=false is present', () => {
    expect(getSidebarDefaultOpen('foo=1; sidebar_state=false; bar=2')).toBe(false)
  })
})
