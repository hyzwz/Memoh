import { tool } from 'ai'
import { AuthFetcher } from '..'
import { z } from 'zod'

export type MemoryToolParams = {
  fetch: AuthFetcher
}

type MemorySearchItem = {
  id?: string
  memory?: string
  score?: number
  createdAt?: string
  metadata?: {
    source?: string
  }
}

export const getMemoryTools = ({ fetch }: MemoryToolParams) => {
  const searchMemory = tool({
    description: 'Search for memories',
    inputSchema: z.object({
      query: z.string().describe('The query to search for memories'),
    }),
    execute: async ({ query }) => {
      const response = await fetch('/memory/search', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          query,
        }),
      })
      const data = await response.json()
      const results = Array.isArray(data?.results)
        ? (data.results as MemorySearchItem[])
        : []
      const simplified = results.map((item) => ({
        id: item?.id,
        memory: item?.memory,
        score: item?.score,
      }))
      return {
        query,
        total: simplified.length,
        results: simplified,
      }
    },
  })

  return {
    'search_memory': searchMemory,
  }
}