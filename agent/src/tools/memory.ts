import { tool } from 'ai'
import { AuthFetcher } from '..'
import { z } from 'zod'

export type MemoryToolParams = {
  fetch: AuthFetcher
}

export const getMemoryTools = ({ fetch }: MemoryToolParams) => {
  const searchMemory = tool({
    description: 'Search for memories',
    inputSchema: z.object({
      query: z.string().describe('The query to search for memories'),
    }),
    execute: async ({ query }) => {
      const response = await fetch(`/memory/search?query=${query}`)
      return response.json()
    },
  })

  return {
    'search_memory': searchMemory,
  }
}