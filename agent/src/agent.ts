import { generateText, ModelMessage, stepCountIs, streamText, TextStreamPart, ToolSet } from 'ai'
import { createChatGateway } from './gateway'
import { ClientType, Schedule } from './types'
import { system, schedule } from './prompts'
import { AuthFetcher } from './index'
import { getScheduleTools } from './tools/schedule'

export interface AgentParams {
  apiKey: string
  baseUrl: string
  model: string
  clientType: ClientType
  locale?: Intl.LocalesArgument
  language?: string
  maxSteps?: number
  maxContextLoadTime: number
  platforms?: string[]
  currentPlatform?: string
}

export interface AgentInput {
  messages: ModelMessage[]
  query: string
}

export interface AgentResult {
  messages: ModelMessage[]
}

export const createAgent = (
  params: AgentParams,
  fetcher: AuthFetcher = fetch,
) => {
  const gateway = createChatGateway(params.clientType)
  const messages: ModelMessage[] = []

  const maxSteps = params.maxSteps ?? 50

  const getTools = () => {
    const scheduleTools = getScheduleTools({ fetch: fetcher })
    return {
      ...scheduleTools,
    }
  }

  const generateSystem = () => {
    return system({
      date: new Date(),
      locale: params.locale,
      language: params.language,
      maxContextLoadTime: params.maxContextLoadTime,
      platforms: params.platforms ?? [],
      currentPlatform: params.currentPlatform,
    })
  }

  const ask = async (input: AgentInput): Promise<AgentResult> => {
    messages.push(...input.messages)
    const user: ModelMessage = {
      role: 'user',
      content: input.query,
    }
    messages.push(user)
    const { response } = await generateText({
      model: gateway({
        apiKey: params.apiKey,
        baseURL: params.baseUrl,
      })(params.model),
      system: generateSystem(),
      stopWhen: stepCountIs(maxSteps),
      messages,
      tools: getTools(),
    })
    return {
      messages: [user, ...response.messages],
    }
  }

  async function* stream(input: AgentInput): AsyncGenerator<TextStreamPart<ToolSet>, AgentResult> {
    messages.push(...input.messages)
    const user: ModelMessage = {
      role: 'user',
      content: input.query,
    }
    messages.push(user)
    const { response, fullStream } = streamText({
      model: gateway({
        apiKey: params.apiKey,
        baseURL: params.baseUrl,
      })(params.model),
      system: generateSystem(),
      stopWhen: stepCountIs(maxSteps),
      messages,
      tools: getTools(),
    })
    for await (const event of fullStream) {
      yield event
    }
    return {
      messages: [user, ...(await response).messages],
    }
  }

  const triggerSchedule = async (
    input: AgentInput,
    scheduleData: Schedule
  ): Promise<AgentResult> => {
    messages.push(...input.messages)
    const user: ModelMessage = {
      role: 'user',
      content: schedule({
        schedule: scheduleData,
        locale: params.locale,
        date: new Date(),
      }),
    }
    messages.push(user)
    const { response } = await generateText({
      model: gateway({
        apiKey: params.apiKey,
        baseURL: params.baseUrl,
      })(params.model),
      system: generateSystem(),
      stopWhen: stepCountIs(maxSteps),
      messages,
      tools: getTools(),
    })
    return {
      messages: [user, ...response.messages],
    }
  }

  return {
    ask,
    stream,
    triggerSchedule,
  }
}