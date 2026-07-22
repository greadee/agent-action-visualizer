export function formatDuration(milliseconds: number | undefined): string {
  if (!milliseconds) return 'Not reported by agent'
  const totalSeconds = Math.round(milliseconds / 1000)
  const hours = Math.floor(totalSeconds / 3600)
  const minutes = Math.floor((totalSeconds % 3600) / 60)
  const seconds = totalSeconds % 60
  return [hours && `${hours}h`, minutes && `${minutes}m`, `${seconds}s`]
    .filter(Boolean)
    .join(' ')
}

export function formatTimestamp(unixSeconds: number | undefined): string {
  return unixSeconds
    ? new Date(unixSeconds * 1000).toLocaleString()
    : 'Not observed'
}
