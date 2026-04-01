import { describe, expect, it } from 'vitest'
import { buildDailyTokensSeries } from './daily-tokens'

describe('daily tokens chart', () => {
  it('builds only total input and total output bar series', () => {
    const series = buildDailyTokensSeries(
      ['2026-04-01'],
      new Map([
        ['2026-04-01', { input_tokens: 10, output_tokens: 4 }],
      ]),
      new Map([
        ['2026-04-01', { input_tokens: 3, output_tokens: 2 }],
      ]),
      {
        totalInput: 'Total Input',
        totalOutput: 'Total Output',
      },
    )

    expect(series).toEqual([
      {
        name: 'Total Input',
        type: 'bar',
        stack: 'tokens',
        data: [13],
      },
      {
        name: 'Total Output',
        type: 'bar',
        stack: 'tokens',
        data: [6],
      },
    ])
  })
})
