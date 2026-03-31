import { describe, expect, it } from 'vitest'
import { client } from '@memoh/sdk/client'
import auditPage from './audit/index.vue?raw'
import budgetsPage from './budgets/index.vue?raw'
import cockpitPage from './cockpit/index.vue?raw'
import handsPage from './hands/index.vue?raw'
import modelRoutingPage from './model-routing/index.vue?raw'

const enterprisePages = {
  'audit/index.vue': auditPage,
  'budgets/index.vue': budgetsPage,
  'cockpit/index.vue': cockpitPage,
  'hands/index.vue': handsPage,
  'model-routing/index.vue': modelRoutingPage,
}

describe('enterprise SDK client usage', () => {
  it('uses only supported lowercase client methods in enterprise pages', () => {
    for (const [page, source] of Object.entries(enterprisePages)) {
      expect(source, page).not.toMatch(/\bclient\.(GET|POST|PUT|DELETE)\(/)
    }
  })

  it('relies on lowercase SDK client helpers', () => {
    expect(typeof client.get).toBe('function')
    expect(typeof client.post).toBe('function')
    expect(typeof client.delete).toBe('function')
    expect((client as Record<string, unknown>).GET).toBeUndefined()
    expect((client as Record<string, unknown>).POST).toBeUndefined()
    expect((client as Record<string, unknown>).DELETE).toBeUndefined()
  })
})
