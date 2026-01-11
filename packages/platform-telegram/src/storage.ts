import type { TokenStorage } from '@memohome/client'
import Redis from 'ioredis'

export const getTokenStorage = async (telegramUserId: string): Promise<TokenStorage> => {
  const redis = new Redis(process.env.REDIS_URL || 'redis://localhost:6379')
  const isExists = await redis.exists(`memohome:telegram:${telegramUserId}:token`)
  const token = isExists ? await redis.get(`memohome:telegram:${telegramUserId}:token`) : null
  return {
    getApiUrl: () => process.env.API_URL || 'http://localhost:7002',
    setApiUrl: () => {},
    getToken: () => token,
    setToken: (token: string) => {
      redis.set(`memohome:telegram:${telegramUserId}:token`, token)
    },
    clearToken: () => {
      redis.del(`memohome:telegram:${telegramUserId}:token`)
    },
  }
}

