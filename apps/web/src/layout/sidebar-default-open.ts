export function getSidebarDefaultOpen(cookie: string) {
  return !cookie.includes('sidebar_state=false')
}
