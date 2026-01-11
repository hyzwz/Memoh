import type { Context } from 'telegraf'
import { login, logout, isLoggedIn, getCurrentUser } from '@memohome/client'
import { getTokenStorage } from './storage'


/**
 * Login command handler for Telegram bot
 * Usage: /login username password
 */
export async function handleLogin(ctx: Context) {
  const telegramUserId = ctx.from?.id.toString()
  if (!telegramUserId) {
    await ctx.reply('❌ Unable to identify user')
    return
  }

  // Parse command arguments
  const args = ctx.message && 'text' in ctx.message 
    ? ctx.message.text.split(' ').slice(1) 
    : []
  
  if (args.length !== 2) {
    await ctx.reply(
      '❌ Invalid format\n\n' +
      'Usage: /login <username> <password>\n' +
      'Example: /login admin password'
    )
    return
  }

  const [username, password] = args

    const storage = await getTokenStorage(telegramUserId)

    // Attempt login
    const result = await login({ username, password }, { storage })

    if (result.success && result.user) {

      await ctx.reply(
        '✅ Login successful!\n\n' +
        `👤 Username: ${result.user.username}\n` +
        `🎭 Role: ${result.user.role}\n` +
        `🔑 User ID: ${result.user.id}\n\n` +
        'You can now use the bot to interact with MemoHome.'
      )
    } else {
      await ctx.reply('❌ Login failed: Invalid response from server')
    }
}

/**
 * Logout command handler for Telegram bot
 * Usage: /logout
 */
export async function handleLogout(ctx: Context) {
  const telegramUserId = ctx.from?.id.toString()
  if (!telegramUserId) {
    await ctx.reply('❌ Unable to identify user')
    return
  }

  try {
    const storage = await getTokenStorage(telegramUserId)

    await logout({ storage })
    await ctx.reply('✅ Logged out successfully')
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Unknown error'
    await ctx.reply(`❌ Logout failed: ${message}`)
  }
}

/**
 * Whoami command handler - show current logged in user
 * Usage: /whoami
 */
export async function handleWhoami(ctx: Context) {
  const telegramUserId = ctx.from?.id.toString()
  if (!telegramUserId) {
    await ctx.reply('❌ Unable to identify user')
    return
  }

  try {
    const storage = await getTokenStorage(telegramUserId)

    const isLogged = await isLoggedIn({ storage })
    
    if (!isLogged) {
      await ctx.reply(
        '❌ You are not logged in\n\n' +
        'Use /login <username> <password> to login'
      )
      return
    }

    const user = await getCurrentUser({ storage })

    await ctx.reply(
      '👤 Current User:\n\n' +
      `Username: ${user.username}\n` +
      `Role: ${user.role}\n` +
      `User ID: ${user.id}\n` +
      `Telegram ID: ${telegramUserId}`
    )
  } catch (error) {
    const message = error instanceof Error ? error.message : 'Unknown error'
    await ctx.reply(`❌ Error: ${message}`)
  }
}

/**
 * Middleware to require authentication
 * Add this middleware to commands that require login
 */
export function requireAuth() {
  return async (ctx: Context, next: () => Promise<void>) => {
    const telegramUserId = ctx.from?.id.toString()
    if (!telegramUserId) {
      await ctx.reply('❌ Unable to identify user')
      return
    }

    const storage = await getTokenStorage(telegramUserId)

    const isLogged = await isLoggedIn({ storage })
    
    if (!isLogged) {
      await ctx.reply(
        '❌ You need to login first\n\n' +
        'Use /login <username> <password> to login'
      )
      return
    }

    // User is authenticated, continue to next handler
    await next()
  }
}

