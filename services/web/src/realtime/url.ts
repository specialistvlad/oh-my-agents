/** Turns the configured http api url into its websocket equivalent. */
export function socketUrl(apiUrl: string): string {
  return `${apiUrl.replace(/^http/, 'ws')}/ws`;
}
