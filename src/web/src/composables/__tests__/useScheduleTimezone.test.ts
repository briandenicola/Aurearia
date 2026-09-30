import { describe, expect, it } from 'vitest'
import { describeSchedule, scheduleZoneLabel, timezoneOptionsWith } from '../useScheduleTimezone'

describe('describeSchedule', () => {
  const base = { startTime: '03:00', defaultStartTime: '08:00', zone: 'America/Chicago' }

  it('describes a once-a-day job', () => {
    expect(describeSchedule(base)).toBe('Runs daily at 03:00 America/Chicago.')
    expect(describeSchedule({ ...base, intervalMinutes: '1440' })).toBe('Runs daily at 03:00 America/Chicago.')
  })

  it('describes sub-day intervals that restart at the start time', () => {
    expect(describeSchedule({ ...base, intervalMinutes: '60' })).toBe("Runs at 03:00 America/Chicago, then every hour until the next day's start.")
    expect(describeSchedule({ ...base, intervalMinutes: 45 })).toBe("Runs at 03:00 America/Chicago, then every 45 minutes until the next day's start.")
  })

  it('describes multi-day intervals in minutes or days', () => {
    expect(describeSchedule({ ...base, intervalMinutes: '2880' })).toBe('Runs every 2 days at 03:00 America/Chicago.')
    expect(describeSchedule({ ...base, intervalDays: '7' })).toBe('Runs every 7 days at 03:00 America/Chicago.')
  })

  it('falls back to defaults for missing or invalid values', () => {
    expect(describeSchedule({ defaultStartTime: '08:00', intervalMinutes: 'abc' })).toBe('Runs daily at 08:00 server time.')
    expect(describeSchedule({ ...base, intervalDays: '0' })).toBe('Runs daily at 03:00 America/Chicago.')
  })
})

describe('schedule zone helpers', () => {
  it('labels an empty zone as server time', () => {
    expect(scheduleZoneLabel('')).toBe('server time')
    expect(scheduleZoneLabel(undefined)).toBe('server time')
    expect(scheduleZoneLabel('Europe/London')).toBe('Europe/London')
  })

  it('keeps an unlisted saved zone first', () => {
    const zones = timezoneOptionsWith('Not/Listed')
    expect(zones[0]).toBe('Not/Listed')
    expect(timezoneOptionsWith().includes('Not/Listed')).toBe(false)
  })
})
