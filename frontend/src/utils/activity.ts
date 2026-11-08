import type { Activity, DailyCount } from '@/api/api'

/** The activities in a row of the same kind by the same actor, shown as one entry of the feed. */
export interface ActivityGroup {
  key: string
  type: Activity['type']
  action: Activity['action']
  actor: Activity['actor']
  items: Activity[]
}

/** The activities of a day in the local time zone. */
export interface ActivityDay {
  /** In the format of 2006-01-02. */
  date: string
  groups: ActivityGroup[]
}

/** Formats the date as 2006-01-02 in the local time zone. */
export function localDate(date: Date): string {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

/** Parses 2006-01-02 as the midnight in the local time zone. */
export function parseLocalDate(date: string): Date {
  const [year, month, day] = date.split('-').map(Number)
  return new Date(year, month - 1, day)
}

/**
 * Splits the feed, the newest first, into the days, and groups the activities of the same kind by the same actor in a day.
 * A group is placed at its newest activity, so the groups stay in the order of the feed.
 */
export function groupActivities(activities: Activity[]): ActivityDay[] {
  const days: ActivityDay[] = []
  let groups = new Map<string, ActivityGroup>()
  for (const activity of activities) {
    const date = localDate(new Date(activity.occurredAt))
    let day = days.at(-1)
    if (day?.date !== date) {
      day = { date, groups: [] }
      days.push(day)
      groups = new Map()
    }

    const kind = `${activity.type}:${activity.action}:${activity.actor.login}`
    const group = groups.get(kind)
    if (group) {
      group.items.push(activity)
      continue
    }
    const created: ActivityGroup = {
      key: `${date}:${kind}`,
      type: activity.type,
      action: activity.action,
      actor: activity.actor,
      items: [activity],
    }
    groups.set(kind, created)
    day.groups.push(created)
  }
  return days
}

export interface HeatmapCell {
  date: string
  count: number
  /** 0 for no activity, then 1 to 4 relative to the busiest day. */
  level: 0 | 1 | 2 | 3 | 4
}

/** Lays out the days, the oldest first, in the columns of the weeks starting on Monday, the cells outside the days are null. */
export function heatmapWeeks(days: DailyCount[]): (HeatmapCell | null)[][] {
  if (!days.length) return []
  const max = Math.max(1, ...days.map((d) => d.count))
  const offset = (parseLocalDate(days[0].date).getDay() + 6) % 7

  const cells: (HeatmapCell | null)[] = Array.from({ length: offset }, () => null)
  for (const { date, count } of days) {
    const level = count === 0 ? 0 : (Math.min(4, Math.ceil((count / max) * 4)) as HeatmapCell['level'])
    cells.push({ date, count, level })
  }
  while (cells.length % 7) cells.push(null)

  const weeks: (HeatmapCell | null)[][] = []
  for (let i = 0; i < cells.length; i += 7) weeks.push(cells.slice(i, i + 7))
  return weeks
}

/**
 * Returns the weeks to label with their months, which are the weeks whose first day falls in a new month.
 * A label too close to the next one is dropped, e.g. the month of a partial first week.
 */
export function heatmapMonths(weeks: (HeatmapCell | null)[][]): { week: number; date: string }[] {
  const months: { week: number; date: string }[] = []
  let last = ''
  weeks.forEach((week, index) => {
    const first = week.find((cell) => cell)
    if (!first) return
    const month = first.date.slice(0, 7)
    if (month !== last) {
      months.push({ week: index, date: first.date })
      last = month
    }
  })
  return months.filter((month, i) => i === months.length - 1 || months[i + 1].week - month.week >= 3)
}
