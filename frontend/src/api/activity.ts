import type { Activity, ActivitySummary, Contributor, DailyCount, ListActivitiesResp, RepositoryActivity } from './api'
import { client } from './client'

export type { Activity, ActivitySummary, Contributor, DailyCount, ListActivitiesResp, RepositoryActivity }

export type ActivityType = Activity['type']

export interface ListActivitiesQuery {
  type?: ActivityType
  repository?: string
  excludeBots?: boolean
  cursor?: string
  limit?: number
}

export const activityApi = {
  list: async (query: ListActivitiesQuery) => (await client.activities.listActivities(query)).data as ListActivitiesResp,
  summary: async (excludeBots: boolean) =>
    (
      await client.activities.getActivitySummary({
        excludeBots,
        tz: Intl.DateTimeFormat().resolvedOptions().timeZone,
      })
    ).data as ActivitySummary,
}
