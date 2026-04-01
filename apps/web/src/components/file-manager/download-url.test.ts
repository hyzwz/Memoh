import { describe, expect, it } from 'vitest'
import { buildFileDownloadUrl } from './download-url'

describe('file-manager download url', () => {
  it('includes jwt token in download query params', () => {
    const url = buildFileDownloadUrl('bot-1', '/data/demo file.txt', 'jwt-token')

    expect(url).toBe(
      '/api/bots/bot-1/container/fs/download?path=%2Fdata%2Fdemo%20file.txt&token=jwt-token',
    )
  })

  it('still emits token query key when token is empty', () => {
    const url = buildFileDownloadUrl('bot-1', '/data/demo.txt', '')

    expect(url).toBe(
      '/api/bots/bot-1/container/fs/download?path=%2Fdata%2Fdemo.txt&token=',
    )
  })
})
