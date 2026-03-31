import type { InjectionKey } from 'vue'

export const openInFileManagerKey = Symbol('openInFileManager') as InjectionKey<(path: string, isDir?: boolean) => void>
