import { describe, expect, it } from 'vitest'
import { extractTriggerFrontmatter, extractTriggerBody } from './trigger-frontmatter'

describe('trigger frontmatter helpers', () => {
  it('extracts yaml-like frontmatter map', () => {
    const text = `---\ninterval: 30m\ntime: 10:00\nlast_heartbeat: 2026-04-02T00:00:00Z\n---\ncheck inbox`
    expect(extractTriggerFrontmatter(text)).toEqual({
      interval: '30m',
      time: '10:00',
      last_heartbeat: '2026-04-02T00:00:00Z',
    })
  })

  it('extracts body after frontmatter', () => {
    const text = `---\nschedule-name: daily\ncron-pattern: 0 9 * * *\n---\nrun daily summary`
    expect(extractTriggerBody(text)).toBe('run daily summary')
  })
})
