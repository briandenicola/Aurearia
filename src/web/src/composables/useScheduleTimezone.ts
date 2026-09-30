import { getBrowserTimezone } from '@/composables/usePurchaseReminder'

// Intl.supportedValuesOf is missing on older Safari; fall back to the browser
// zone alone so a saved value is still selectable.
export function listTimezones(): string[] {
  const intl = Intl as typeof Intl & { supportedValuesOf?: (key: 'timeZone') => string[] }
  const zones = intl.supportedValuesOf?.('timeZone') ?? []
  return zones.length ? zones : [getBrowserTimezone()].filter(Boolean)
}

export function timezoneOptionsWith(saved?: string): string[] {
  const zones = listTimezones()
  return saved && !zones.includes(saved) ? [saved, ...zones] : zones
}

export function scheduleZoneLabel(zone?: string): string {
  return zone?.trim() || 'server time'
}

function formatMinutes(minutes: number): string {
  if (minutes % 60 === 0) {
    const hours = minutes / 60
    return hours === 1 ? 'hour' : `${hours} hours`
  }
  return minutes === 1 ? 'minute' : `${minutes} minutes`
}

export interface ScheduleDescription {
  startTime?: string
  defaultStartTime: string
  zone?: string
  intervalMinutes?: string | number
  intervalDays?: string | number
}

// Mirrors the API's dailySchedule rules: sub-day intervals restart from the
// start time each day; day-or-longer intervals run at the start time.
export function describeSchedule(options: ScheduleDescription): string {
  const time = options.startTime?.trim() || options.defaultStartTime
  const zone = scheduleZoneLabel(options.zone)

  let days = Number(options.intervalDays)
  const minutes = Number(options.intervalMinutes)
  if (!(days > 0) && minutes > 0) {
    if (minutes < 1440) {
      return `Runs at ${time} ${zone}, then every ${formatMinutes(Math.floor(minutes))} until the next day's start.`
    }
    days = Math.floor(minutes / 1440)
  }
  days = days > 0 ? Math.floor(days) : 1
  return days === 1
    ? `Runs daily at ${time} ${zone}.`
    : `Runs every ${days} days at ${time} ${zone}.`
}
