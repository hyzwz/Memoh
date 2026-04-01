type DailyUsage = {
  input_tokens?: number
  output_tokens?: number
}

export function buildDailyTokensSeries(
  days: string[],
  chatMap: Map<string, DailyUsage>,
  heartbeatMap: Map<string, DailyUsage>,
  labels: { totalInput: string, totalOutput: string },
) {
  return [
    {
      name: labels.totalInput,
      type: 'bar',
      stack: 'tokens',
      data: days.map((d) => (chatMap.get(d)?.input_tokens ?? 0) + (heartbeatMap.get(d)?.input_tokens ?? 0)),
    },
    {
      name: labels.totalOutput,
      type: 'bar',
      stack: 'tokens',
      data: days.map((d) => (chatMap.get(d)?.output_tokens ?? 0) + (heartbeatMap.get(d)?.output_tokens ?? 0)),
    },
  ]
}
