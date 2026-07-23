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

export function formatExactDuration(milliseconds: number | undefined): string {
  if (milliseconds === undefined || !Number.isFinite(milliseconds))
    return 'Not reported by agent'
  const value = Math.max(0, Math.floor(milliseconds))
  const hours = Math.floor(value / 3_600_000)
  const minutes = Math.floor((value % 3_600_000) / 60_000)
  const seconds = Math.floor((value % 60_000) / 1_000)
  const remainder = value % 1_000
  return [
    hours && `${hours}h`,
    minutes && `${minutes}m`,
    seconds && `${seconds}s`,
    `${remainder}ms`,
  ]
    .filter(Boolean)
    .join(' ')
}

export function formatTimestamp(unixSeconds: number | undefined): string {
  return unixSeconds
    ? new Date(unixSeconds * 1000).toLocaleString()
    : 'Not observed'
}
