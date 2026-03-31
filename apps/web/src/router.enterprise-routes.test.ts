import { describe, expect, it, vi } from 'vitest'

vi.mock('./i18n', () => ({
  i18nRef: () => '',
}))

vi.mock('vue-router', async () => {
  const actual = await vi.importActual<typeof import('vue-router')>('vue-router')

  return {
    ...actual,
    createWebHistory: actual.createMemoryHistory,
  }
})

import router from './router'

describe('enterprise legacy route compatibility', () => {
  it('matches /enterprise/model-routing to the model-routing route', () => {
    const resolved = router.resolve('/enterprise/model-routing')

    expect(resolved.name).toBe('model-routing')
    expect(resolved.matched.some(record => record.name === 'model-routing')).toBe(true)
  })
})
