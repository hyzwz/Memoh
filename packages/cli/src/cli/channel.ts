import { Command } from 'commander'
import chalk from 'chalk'
import inquirer from 'inquirer'
import ora from 'ora'
import { table } from 'table'

import { ensureAuth, getErrorMessage, resolveBotId } from './shared'
import { getBaseURL, readConfig } from '../utils/store'

type ChannelFieldSchema = {
  type: 'string' | 'secret' | 'bool' | 'number' | 'enum'
  required: boolean
  title?: string
  description?: string
  enum?: string[]
  example?: unknown
}

type ChannelConfigSchema = {
  version: number
  fields: Record<string, ChannelFieldSchema>
}

type ChannelMeta = {
  type: string
  display_name: string
  configless: boolean
  capabilities: Record<string, boolean>
  config_schema: ChannelConfigSchema
  user_config_schema: ChannelConfigSchema
}

type ChannelUserBinding = {
  id: string
  channel_type: string
  user_id: string
  config: Record<string, unknown>
  created_at: string
  updated_at: string
}

type ChannelConfig = {
  id: string
  bot_id: string
  channel_type: string
  credentials: Record<string, unknown>
  external_identity: string
  self_identity: Record<string, unknown>
  routing: Record<string, unknown>
  capabilities: Record<string, unknown>
  disabled: boolean
  verified_at: string
  created_at: string
  updated_at: string
}

type APIRequestOptions = {
  method?: string
  body?: string
  headers?: Record<string, string>
}

const apiRequest = async <T>(path: string, options: APIRequestOptions, token: ReturnType<typeof ensureAuth>) => {
  const baseURL = getBaseURL(readConfig()).replace(/\/+$/, '')
  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  const response = await fetch(`${baseURL}${normalizedPath}`, {
    method: options.method ?? 'GET',
    body: options.body,
    headers: {
      Accept: 'application/json',
      ...(options.body ? { 'Content-Type': 'application/json' } : {}),
      ...(token.access_token ? { Authorization: `Bearer ${token.access_token}` } : {}),
      ...(options.headers ?? {}),
    },
  })
  const text = await response.text()
  const parseJSON = () => {
    if (!text.trim()) return null
    try {
      return JSON.parse(text) as T
    } catch {
      return null
    }
  }
  if (!response.ok) {
    const payload = parseJSON()
    const message = payload && typeof payload === 'object' && payload !== null && 'message' in payload
      ? String((payload as { message?: unknown }).message || '')
      : text.trim()
    throw new Error(message || `Request failed with status ${response.status}`)
  }
  return (parseJSON() ?? null) as T
}

const readInboundMode = (credentials: Record<string, unknown>) => {
  const raw = credentials.inboundMode ?? credentials.inbound_mode
  if (typeof raw !== 'string') return ''
  return raw.trim().toLowerCase()
}

const buildWebhookCallbackUrl = (configId: string) => {
  const baseUrl = getBaseURL(readConfig()).replace(/\/+$/, '')
  return `${baseUrl}/channels/feishu/webhook/${encodeURIComponent(configId)}`
}

const printWebhookCallbackIfEnabled = (channelType: string, config: ChannelConfig) => {
  if (channelType !== 'feishu') return
  if (readInboundMode(config.credentials || {}) !== 'webhook') return
  const configId = String(config.id || '').trim()
  if (!configId) {
    console.log(chalk.yellow('Webhook is enabled, but config id is missing so callback URL cannot be generated yet.'))
    return
  }
  console.log(chalk.cyan(`Webhook callback URL: ${buildWebhookCallbackUrl(configId)}`))
}

const renderChannelsTable = (items: ChannelMeta[]) => {
  const rows: string[][] = [['Type', 'Name', 'Configless']]
  for (const item of items) {
    rows.push([item.type, item.display_name, item.configless ? 'yes' : 'no'])
  }
  return table(rows)
}

const fetchChannels = async (token: ReturnType<typeof ensureAuth>) => {
  return apiRequest<ChannelMeta[]>('/channels', {}, token)
}

const resolveChannelType = async (
  token: ReturnType<typeof ensureAuth>,
  preset?: string,
  options?: { allowConfigless?: boolean }
) => {
  if (preset && preset.trim()) {
    return preset.trim()
  }
  const channels = await fetchChannels(token)
  const allowConfigless = options?.allowConfigless ?? false
  const candidates = channels.filter(item => allowConfigless || !item.configless)
  if (candidates.length === 0) {
    console.log(chalk.yellow('No configurable channels available.'))
    process.exit(0)
  }
  const { channelType } = await inquirer.prompt<{ channelType: string }>([
    {
      type: 'list',
      name: 'channelType',
      message: 'Select channel type:',
      choices: candidates.map(item => ({
        name: `${item.display_name} (${item.type})`,
        value: item.type,
      })),
    },
  ])
  return channelType
}

const collectFeishuCredentials = async (opts: Record<string, unknown>) => {
  let appId = typeof opts.app_id === 'string' ? opts.app_id : undefined
  let appSecret = typeof opts.app_secret === 'string' ? opts.app_secret : undefined
  let encryptKey = typeof opts.encrypt_key === 'string' ? opts.encrypt_key : undefined
  let verificationToken = typeof opts.verification_token === 'string' ? opts.verification_token : undefined
  let region = typeof opts.region === 'string' ? opts.region : undefined
  let inboundMode = typeof opts.inbound_mode === 'string' ? opts.inbound_mode : undefined

  const questions = []
  if (!appId) questions.push({ type: 'input', name: 'appId', message: 'Feishu App ID:' })
  if (!appSecret) questions.push({ type: 'password', name: 'appSecret', message: 'Feishu App Secret:' })
  if (!encryptKey) {
    questions.push({ type: 'input', name: 'encryptKey', message: 'Encrypt Key (optional):', default: '' })
  }
  if (!verificationToken) {
    questions.push({ type: 'input', name: 'verificationToken', message: 'Verification Token (optional):', default: '' })
  }
  if (!region) {
    questions.push({
      type: 'list',
      name: 'region',
      message: 'Region:',
      choices: [
        { name: 'Feishu (open.feishu.cn)', value: 'feishu' },
        { name: 'Lark (open.larksuite.com)', value: 'lark' },
      ],
      default: 'feishu',
    })
  }
  if (!inboundMode) {
    questions.push({
      type: 'list',
      name: 'inboundMode',
      message: 'Inbound mode:',
      choices: [
        { name: 'WebSocket', value: 'websocket' },
        { name: 'Webhook', value: 'webhook' },
      ],
      default: 'websocket',
    })
  }
  const answers = questions.length ? await inquirer.prompt<Record<string, string>>(questions) : {}

  appId = appId ?? answers.appId
  appSecret = appSecret ?? answers.appSecret
  encryptKey = encryptKey ?? answers.encryptKey
  verificationToken = verificationToken ?? answers.verificationToken
  region = region ?? answers.region
  inboundMode = inboundMode ?? answers.inboundMode

  const payload: Record<string, unknown> = {
    appId: String(appId).trim(),
    appSecret: String(appSecret).trim(),
    region: String(region || 'feishu').trim(),
    inboundMode: String(inboundMode || 'websocket').trim(),
  }
  if (String(encryptKey || '').trim()) payload.encryptKey = String(encryptKey).trim()
  if (String(verificationToken || '').trim()) payload.verificationToken = String(verificationToken).trim()
  return payload
}

const collectFeishuUserConfig = async (opts: Record<string, unknown>) => {
  let openId = typeof opts.open_id === 'string' ? opts.open_id : undefined
  let userId = typeof opts.user_id === 'string' ? opts.user_id : undefined

  if (!openId && !userId) {
    const answers = await inquirer.prompt<{ kind: 'open_id' | 'user_id'; value: string }>([
      {
        type: 'list',
        name: 'kind',
        message: 'Bind using:',
        choices: [
          { name: 'Open ID', value: 'open_id' },
          { name: 'User ID', value: 'user_id' },
        ],
      },
      {
        type: 'input',
        name: 'value',
        message: 'Value:',
      },
    ])
    if (answers.kind === 'open_id') openId = answers.value
    if (answers.kind === 'user_id') userId = answers.value
  }
  if (!openId && !userId) {
    console.log(chalk.red('open_id or user_id is required.'))
    process.exit(1)
  }
  const config: Record<string, unknown> = {}
  if (openId) config.open_id = String(openId).trim()
  if (userId) config.user_id = String(userId).trim()
  return config
}

const collectWeComBotCredentials = async (opts: Record<string, unknown>) => {
  let botId = typeof opts.bot_id === 'string' ? opts.bot_id : undefined
  let secret = typeof opts.secret === 'string' ? opts.secret : undefined
  let websocketUrl = typeof opts.websocket_url === 'string' ? opts.websocket_url : undefined
  let sendThinkingPrompt = typeof opts.send_thinking_prompt === 'string' ? opts.send_thinking_prompt : undefined

  const questions = []
  if (!botId) questions.push({ type: 'input', name: 'botId', message: 'WeCom Bot ID:' })
  if (!secret) questions.push({ type: 'password', name: 'secret', message: 'WeCom Bot Secret:' })
  if (!websocketUrl) {
    questions.push({
      type: 'input',
      name: 'websocketUrl',
      message: 'WebSocket URL (optional):',
      default: '',
    })
  }
  if (!sendThinkingPrompt) {
    questions.push({
      type: 'confirm',
      name: 'sendThinkingPrompt',
      message: 'Send initial <think></think> placeholder when streaming starts?',
      default: true,
    })
  }
  const answers = questions.length ? await inquirer.prompt<Record<string, string | boolean>>(questions) : {}

  botId = botId ?? (typeof answers.botId === 'string' ? answers.botId : undefined)
  secret = secret ?? (typeof answers.secret === 'string' ? answers.secret : undefined)
  websocketUrl = websocketUrl ?? (typeof answers.websocketUrl === 'string' ? answers.websocketUrl : undefined)
  if (!sendThinkingPrompt && typeof answers.sendThinkingPrompt === 'boolean') {
    sendThinkingPrompt = answers.sendThinkingPrompt ? 'true' : 'false'
  }

  const payload: Record<string, unknown> = {
    botId: String(botId || '').trim(),
    secret: String(secret || '').trim(),
  }
  if (String(websocketUrl || '').trim()) {
    payload.websocketUrl = String(websocketUrl).trim()
  }
  if (typeof sendThinkingPrompt === 'string' && sendThinkingPrompt.trim()) {
    payload.sendThinkingPrompt = ['true', '1', 'yes', 'on'].includes(sendThinkingPrompt.trim().toLowerCase())
  }
  return payload
}

const collectWeComBotUserConfig = async (opts: Record<string, unknown>) => {
  let userId = typeof opts.user_id === 'string' ? opts.user_id : undefined
  let chatId = typeof opts.chat_id === 'string' ? opts.chat_id : undefined

  if (!userId && !chatId) {
    const answers = await inquirer.prompt<{ kind: 'userid' | 'chatid'; value: string }>([
      {
        type: 'list',
        name: 'kind',
        message: 'Bind using:',
        choices: [
          { name: 'Chat ID (recommended for proactive send)', value: 'chatid' },
          { name: 'User ID', value: 'userid' },
        ],
        default: 'chatid',
      },
      {
        type: 'input',
        name: 'value',
        message: 'Value:',
      },
    ])
    if (answers.kind === 'userid') userId = answers.value
    if (answers.kind === 'chatid') chatId = answers.value
  }
  if (!userId && !chatId) {
    console.log(chalk.red('user_id or chat_id is required.'))
    process.exit(1)
  }
  const config: Record<string, unknown> = {}
  if (userId) config.userid = String(userId).trim()
  if (chatId) config.chatid = String(chatId).trim()
  return config
}

const collectChannelCredentials = async (channelType: string, opts: Record<string, unknown>) => {
  switch (channelType) {
    case 'feishu':
      return collectFeishuCredentials(opts)
    case 'wecom_ai_bot':
      return collectWeComBotCredentials(opts)
    default:
      console.log(chalk.red(`Channel type ${channelType} is not supported by this command.`))
      process.exit(1)
  }
}

const collectChannelUserConfig = async (channelType: string, opts: Record<string, unknown>) => {
  switch (channelType) {
    case 'feishu':
      return collectFeishuUserConfig(opts)
    case 'wecom_ai_bot':
      return collectWeComBotUserConfig(opts)
    default:
      console.log(chalk.red(`Channel type ${channelType} is not supported by this command.`))
      process.exit(1)
  }
}

export const registerChannelCommands = (program: Command) => {
  const channel = program.command('channel').description('Channel management')

  channel
    .command('list')
    .description('List available channels')
    .action(async () => {
      const token = ensureAuth()
      const channels = await fetchChannels(token)
      if (!channels.length) {
        console.log(chalk.yellow('No channels available.'))
        return
      }
      console.log(renderChannelsTable(channels))
    })

  channel
    .command('info')
    .description('Show channel meta and schema')
    .argument('[type]')
    .action(async (type) => {
      const token = ensureAuth()
      const channelType = await resolveChannelType(token, type, { allowConfigless: true })
      const meta = await apiRequest<ChannelMeta>(`/channels/${encodeURIComponent(channelType)}`, {}, token)
      console.log(JSON.stringify(meta, null, 2))
    })

  const config = channel.command('config').description('Bot channel configuration')

  config
    .command('get')
    .description('Get bot channel config')
    .argument('[bot_id]')
    .option('--type <type>', 'Channel type')
    .action(async (botId, opts) => {
      const token = ensureAuth()
      const resolvedBotId = await resolveBotId(botId)
      const channelType = await resolveChannelType(token, opts.type)
      const resp = await apiRequest<ChannelConfig>(`/bots/${encodeURIComponent(resolvedBotId)}/channel/${encodeURIComponent(channelType)}`, {}, token)
      console.log(JSON.stringify(resp, null, 2))
      printWebhookCallbackIfEnabled(channelType, resp)
    })

  config
    .command('set')
    .description('Set bot channel config')
    .argument('[bot_id]')
    .option('--type <type>', 'Channel type (feishu|wecom_ai_bot)')
    .option('--app_id <app_id>')
    .option('--app_secret <app_secret>')
    .option('--encrypt_key <encrypt_key>')
    .option('--verification_token <verification_token>')
    .option('--region <region>', 'feishu|lark')
    .option('--inbound_mode <inbound_mode>', 'websocket|webhook')
    .option('--bot_id <bot_id>', 'WeCom AI Bot ID')
    .option('--secret <secret>', 'WeCom AI Bot secret')
    .option('--websocket_url <websocket_url>', 'WeCom AI Bot websocket URL override')
    .option('--send_thinking_prompt <send_thinking_prompt>', 'true|false')
    .action(async (botId, opts) => {
      const token = ensureAuth()
      const resolvedBotId = await resolveBotId(botId)
      const channelType = await resolveChannelType(token, opts.type)
      const credentials = await collectChannelCredentials(channelType, opts)
      const spinner = ora('Updating channel config...').start()
      try {
        const resp = await apiRequest<ChannelConfig>(`/bots/${encodeURIComponent(resolvedBotId)}/channel/${encodeURIComponent(channelType)}`, {
          method: 'PUT',
          body: JSON.stringify({ credentials }),
        }, token)
        spinner.succeed('Channel config updated')
        printWebhookCallbackIfEnabled(channelType, resp)
      } catch (err: unknown) {
        spinner.fail(getErrorMessage(err) || 'Failed to update channel config')
        process.exit(1)
      }
    })

  const binding = channel.command('bind').description('User channel binding')

  binding
    .command('get')
    .description('Get current user channel binding')
    .option('--type <type>', 'Channel type')
    .action(async (opts) => {
      const token = ensureAuth()
      const channelType = await resolveChannelType(token, opts.type)
      const resp = await apiRequest<ChannelUserBinding>(`/users/me/channels/${encodeURIComponent(channelType)}`, {}, token)
      console.log(JSON.stringify(resp, null, 2))
    })

  binding
    .command('set')
    .description('Set current user channel binding')
    .option('--type <type>', 'Channel type (feishu|wecom_ai_bot)')
    .option('--open_id <open_id>')
    .option('--user_id <user_id>')
    .option('--chat_id <chat_id>')
    .action(async (opts) => {
      const token = ensureAuth()
      const channelType = await resolveChannelType(token, opts.type)
      const configPayload = await collectChannelUserConfig(channelType, opts)
      const spinner = ora('Updating user binding...').start()
      try {
        await apiRequest(`/users/me/channels/${encodeURIComponent(channelType)}`, {
          method: 'PUT',
          body: JSON.stringify({ config: configPayload }),
        }, token)
        spinner.succeed('User binding updated')
      } catch (err: unknown) {
        spinner.fail(getErrorMessage(err) || 'Failed to update user binding')
        process.exit(1)
      }
    })
}
