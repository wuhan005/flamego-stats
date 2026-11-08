import assert from 'node:assert/strict'
import { test } from 'node:test'

import type { Activity } from '../src/api/api.ts'
import { groupActivities, heatmapMonths, heatmapWeeks, localDate } from '../src/utils/activity.ts'

function activity(id: number, occurredAt: Date, patch: Partial<Activity> = {}): Activity {
  return {
    id,
    type: 'pull_request',
    action: 'merged',
    repository: { name: 'web', htmlURL: '' },
    number: id,
    sha: '',
    tag: '',
    title: '',
    htmlURL: '',
    draft: false,
    prerelease: false,
    actor: { login: 'alice', avatarURL: '', htmlURL: '', bot: false },
    occurredAt: occurredAt.toISOString(),
    ...patch,
  }
}

test('groupActivities groups the activities of a kind by an actor in a day', () => {
  const day = new Date(2026, 0, 10, 12)
  const yesterday = new Date(2026, 0, 9, 23)
  const bob = { login: 'bob', avatarURL: '', htmlURL: '', bot: false }
  const days = groupActivities([
    activity(6, day),
    activity(5, day, { actor: bob }),
    activity(4, day),
    activity(3, day, { action: 'opened' }),
    activity(2, yesterday),
    activity(1, yesterday, { actor: bob }),
  ])

  assert.deepEqual(
    days.map((d) => ({ date: d.date, groups: d.groups.map((g) => g.items.map((a) => a.id)) })),
    [
      { date: '2026-01-10', groups: [[6, 4], [5], [3]] },
      { date: '2026-01-09', groups: [[2], [1]] },
    ],
  )
})

test('heatmapWeeks lays out the days in the weeks starting on Monday', () => {
  // 2026-01-07 is a Wednesday.
  const days = ['2026-01-07', '2026-01-08', '2026-01-09', '2026-01-10', '2026-01-11', '2026-01-12'].map((date, i) => ({
    date,
    count: [0, 1, 2, 3, 8, 0][i],
  }))
  const weeks = heatmapWeeks(days)

  assert.equal(weeks.length, 2)
  assert.deepEqual(
    weeks.map((week) => week.map((cell) => cell?.level ?? null)),
    [
      [null, null, 0, 1, 1, 2, 4],
      [0, null, null, null, null, null, null],
    ],
  )
})

test('heatmapMonths labels the weeks starting a month, dropping the crowded ones', () => {
  const days = []
  for (let d = new Date(2026, 0, 28); d <= new Date(2026, 2, 20); d.setDate(d.getDate() + 1)) {
    days.push({ date: localDate(d), count: 0 })
  }
  // The partial week of January is too close to February.
  assert.deepEqual(
    heatmapMonths(heatmapWeeks(days)).map((m) => m.date.slice(0, 7)),
    ['2026-02', '2026-03'],
  )
})
