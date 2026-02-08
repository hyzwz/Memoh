import { Command } from 'commander'
import chalk from 'chalk'
import inquirer from 'inquirer'
import ora from 'ora'
import { table } from 'table'
import readline from 'node:readline/promises'
import { stdin as input, stdout as output } from 'node:process'
import { apiRequest } from '../core/api'
import { readConfig, TokenInfo } from '../utils/store'
import { ensureAuth, getErrorMessage, resolveBotId, BotSummary } from './shared'
import { streamChat } from './stream'

type Bot = BotSummary & {
  metadata?: Record<string, unknown>
  created_at: string
  updated_at: string
}

type BotListResponse = {
  items: Bot[]
}

type ModelResponse = {
  model_id?: string
  model?: {
    model_id: string
    type: 'chat' | 'embedding'
  }
  type?: 'chat' | 'embedding'
}

const getModelId = (item: ModelResponse) => item.model?.model_id ?? item.model_id ?? ''
const getModelType = (item: ModelResponse) => item.model?.type ?? item.type ?? 'chat'

const ensureModelsReady = async () => {
  const token = ensureAuth()
  const [chatModels, embeddingModels] = await Promise.all([
    apiRequest<ModelResponse[]>('/models?type=chat', {}, token),
    apiRequest<ModelResponse[]>('/models?type=embedding', {}, token),
  ])
  if (chatModels.length === 0 || embeddingModels.length === 0) {
    console.log(chalk.red('Model configuration incomplete.'))
    console.log(chalk.yellow('At least one chat model and one embedding model are required.'))
    process.exit(1)
  }
}

const renderBotsTable = (items: BotSummary[]) => {
  const rows: string[][] = [['ID', 'Name', 'Type', 'Active', 'Owner']]
  for (const bot of items) {
    rows.push([
      bot.id,
      bot.display_name || bot.id,
      bot.type,
      bot.is_active ? 'yes' : 'no',
      bot.owner_user_id,
    ])
  }
  return table(rows)
}

export const registerBotCommands = (program: Command) => {
  const bot = program.command('bot').description('Bot management')

  bot
    .command('list')
    .description('List bots')
    .option('--owner <user_id>', 'Filter by owner user ID (admin only)')
    .action(async (opts) => {
      const token = ensureAuth()
      const query = opts.owner ? `?owner_id=${encodeURIComponent(String(opts.owner))}` : ''
      const resp = await apiRequest<BotListResponse>(`/bots${query}`, {}, token)
      if (!resp.items.length) {
        console.log(chalk.yellow('No bots found.'))
        return
      }
      console.log(renderBotsTable(resp.items))
    })

  bot
    .command('create')
    .description('Create a bot')
    .option('--type <type>', 'Bot type (personal, public)')
    .option('--name <name>', 'Bot display name')
    .option('--avatar <url>', 'Bot avatar URL')
    .option('--active', 'Set bot active')
    .option('--inactive', 'Set bot inactive')
    .action(async (opts) => {
      if (opts.active && opts.inactive) {
        console.log(chalk.red('Use only one of --active or --inactive.'))
        process.exit(1)
      }
      const token = ensureAuth()
      let type = opts.type
      if (!type) {
        const answer = await inquirer.prompt<{ type: string }>([
          {
            type: 'list',
            name: 'type',
            message: 'Bot type:',
            choices: ['personal', 'public'],
          },
        ])
        type = answer.type
      }
      if (!['personal', 'public'].includes(String(type))) {
        console.log(chalk.red('Bot type must be personal or public.'))
        process.exit(1)
      }
      const name = opts.name ?? (await inquirer.prompt<{ name: string }>([
        { type: 'input', name: 'name', message: 'Bot name (optional):', default: '' },
      ])).name
      const payload: Record<string, unknown> = {
        type: String(type),
      }
      if (String(name).trim()) payload.display_name = String(name).trim()
      if (opts.avatar) payload.avatar_url = String(opts.avatar).trim()
      if (opts.active) payload.is_active = true
      if (opts.inactive) payload.is_active = false
      const spinner = ora('Creating bot...').start()
      try {
        const created = await apiRequest<Bot>('/bots', { method: 'POST', body: JSON.stringify(payload) }, token)
        spinner.succeed(`Bot created: ${created.display_name || created.id}`)
      } catch (err: unknown) {
        spinner.fail(getErrorMessage(err) || 'Failed to create bot')
        process.exit(1)
      }
    })

  bot
    .command('update')
    .description('Update bot info')
    .argument('[id]')
    .option('--name <name>', 'Bot display name')
    .option('--avatar <url>', 'Bot avatar URL')
    .option('--active', 'Set bot active')
    .option('--inactive', 'Set bot inactive')
    .action(async (id, opts) => {
      if (opts.active && opts.inactive) {
        console.log(chalk.red('Use only one of --active or --inactive.'))
        process.exit(1)
      }
      const token = ensureAuth()
      const botId = await resolveBotId(token, id)
      const payload: Record<string, unknown> = {}
      if (opts.name) payload.display_name = String(opts.name).trim()
      if (opts.avatar) payload.avatar_url = String(opts.avatar).trim()
      if (opts.active) payload.is_active = true
      if (opts.inactive) payload.is_active = false
      if (Object.keys(payload).length === 0) {
        const answers = await inquirer.prompt<{ name: string; avatar: string; status: string }>([
          { type: 'input', name: 'name', message: 'Bot name (leave empty to skip):', default: '' },
          { type: 'input', name: 'avatar', message: 'Bot avatar URL (leave empty to skip):', default: '' },
          {
            type: 'list',
            name: 'status',
            message: 'Bot status:',
            choices: [
              { name: 'keep', value: 'keep' },
              { name: 'active', value: 'active' },
              { name: 'inactive', value: 'inactive' },
            ],
          },
        ])
        if (answers.name.trim()) payload.display_name = answers.name.trim()
        if (answers.avatar.trim()) payload.avatar_url = answers.avatar.trim()
        if (answers.status === 'active') payload.is_active = true
        if (answers.status === 'inactive') payload.is_active = false
      }
      if (Object.keys(payload).length === 0) {
        console.log(chalk.red('No updates provided.'))
        process.exit(1)
      }
      const spinner = ora('Updating bot...').start()
      try {
        await apiRequest(`/bots/${encodeURIComponent(botId)}`, { method: 'PUT', body: JSON.stringify(payload) }, token)
        spinner.succeed('Bot updated')
      } catch (err: unknown) {
        spinner.fail(getErrorMessage(err) || 'Failed to update bot')
        process.exit(1)
      }
    })

  bot
    .command('delete')
    .description('Delete a bot')
    .argument('[id]')
    .action(async (id) => {
      const token = ensureAuth()
      const botId = await resolveBotId(token, id)
      const { confirmed } = await inquirer.prompt<{ confirmed: boolean }>([
        { type: 'confirm', name: 'confirmed', message: `Delete bot ${botId}?`, default: false },
      ])
      if (!confirmed) return
      const spinner = ora('Deleting bot...').start()
      try {
        await apiRequest(`/bots/${encodeURIComponent(botId)}`, { method: 'DELETE' }, token)
        spinner.succeed('Bot deleted')
      } catch (err: unknown) {
        spinner.fail(getErrorMessage(err) || 'Failed to delete bot')
        process.exit(1)
      }
    })

  bot
    .command('chat')
    .description('Chat with a bot (stream)')
    .argument('[id]')
    .option('--session <id>', 'Reuse a session id')
    .action(async (id, opts) => {
      await ensureModelsReady()
      const token = ensureAuth()
      const botId = await resolveBotId(token, id)
      const config = readConfig()
      const sessionId = String(opts.session || config.session_id)
      const rl = readline.createInterface({ input, output })
      console.log(chalk.green(`Chatting with ${chalk.bold(botId)} (session ${sessionId}). Type \`exit\` to quit.`))
      while (true) {
        const line = (await rl.question(chalk.cyan('> '))).trim()
        if (!line) {
          if (!input.isTTY && input.readableEnded) break
          continue
        }
        if (line.toLowerCase() === 'exit') {
          break
        }
        try {
          const ok = await streamChat(line, botId, sessionId, token)
          if (!ok) {
            console.log(chalk.red('Chat failed or stream unavailable.'))
          }
        } catch (err: unknown) {
          console.log(chalk.red(getErrorMessage(err) || 'Chat failed'))
        }
      }
      rl.close()
    })

  bot
    .command('set-model')
    .description('Enable model for a bot')
    .argument('[id]')
    .option('--as <usage>', 'chat | memory | embedding')
    .option('--model <model_id>', 'Model ID')
    .action(async (id, opts) => {
      const token = ensureAuth()
      const botId = await resolveBotId(token, id)
      let enableAs = opts.as
      if (!enableAs) {
        const answer = await inquirer.prompt<{ usage: string }>([{
          type: 'list',
          name: 'usage',
          message: 'Enable as:',
          choices: ['chat', 'memory', 'embedding'],
        }])
        enableAs = answer.usage
      }
      enableAs = String(enableAs).trim()
      if (!['chat', 'memory', 'embedding'].includes(enableAs)) {
        console.log(chalk.red('Enable as must be one of chat, memory, embedding.'))
        process.exit(1)
      }
      const models = await apiRequest<ModelResponse[]>('/models', {}, token)
      const requiredType = enableAs === 'embedding' ? 'embedding' : 'chat'
      const candidates = models.filter(m => getModelType(m) === requiredType)
      if (candidates.length === 0) {
        console.log(chalk.red(`No ${requiredType} models available.`))
        process.exit(1)
      }
      let modelId = opts.model
      if (!modelId) {
        const answer = await inquirer.prompt<{ model: string }>([{
          type: 'list',
          name: 'model',
          message: 'Select model:',
          choices: candidates.map(m => getModelId(m)),
        }])
        modelId = answer.model
      }
      const selected = candidates.find(m => getModelId(m) === modelId)
      if (!selected) {
        console.log(chalk.red('Selected model not found.'))
        process.exit(1)
      }
      const payload: Record<string, unknown> = {}
      if (enableAs === 'chat') payload.chat_model_id = getModelId(selected)
      if (enableAs === 'memory') payload.memory_model_id = getModelId(selected)
      if (enableAs === 'embedding') payload.embedding_model_id = getModelId(selected)
      const spinner = ora('Updating bot settings...').start()
      try {
        await apiRequest(`/bots/${encodeURIComponent(botId)}/settings`, {
          method: 'PUT',
          body: JSON.stringify(payload),
        }, token)
        spinner.succeed('Model enabled')
      } catch (err: unknown) {
        spinner.fail(getErrorMessage(err) || 'Failed to enable model')
        process.exit(1)
      }
    })
}

