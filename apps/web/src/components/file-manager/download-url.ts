export function buildFileDownloadUrl(botId: string, path: string, token: string): string {
  return `/api/bots/${botId}/container/fs/download?path=${encodeURIComponent(path)}&token=${encodeURIComponent(token)}`
}
