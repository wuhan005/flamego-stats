import type { CountItem, ListRepositoriesResp, Overview, OwnerRank, Repository, SyncRun } from './api'
import { client } from './client'

export type { CountItem, ListRepositoriesResp, Overview, OwnerRank, Repository, SyncRun }

export type RepositorySort = 'stars' | 'forks' | 'pushed' | 'created' | 'name'
export type VersionGranularity = 'exact' | 'minor'

export interface ListRepositoriesQuery {
  page?: number
  pageSize?: number
  sort?: RepositorySort
  keyword?: string
  topic?: string
  language?: string
  owner?: string
  version?: string
  includeForks?: boolean
}

export const statsApi = {
  overview: async (includeForks: boolean) => (await client.overview.getOverview({ includeForks })).data as Overview,
  repositories: async (query: ListRepositoriesQuery) =>
    (await client.repositories.listRepositories(query)).data as ListRepositoriesResp,
  owners: async (includeForks: boolean, limit?: number) =>
    (await client.stats.listTopOwners({ includeForks, limit })).data as OwnerRank[],
  topics: async (includeForks: boolean, limit?: number) =>
    (await client.stats.listTopics({ includeForks, limit })).data as CountItem[],
  languages: async (includeForks: boolean, limit?: number) =>
    (await client.stats.listLanguages({ includeForks, limit })).data as CountItem[],
  versions: async (includeForks: boolean, granularity: VersionGranularity) =>
    (await client.stats.listVersions({ includeForks, granularity })).data as CountItem[],
  goVersions: async (includeForks: boolean) => (await client.stats.listGoVersions({ includeForks })).data as CountItem[],
  modules: async (includeForks: boolean, limit?: number) =>
    (await client.stats.listModules({ includeForks, limit })).data as CountItem[],
  syncRuns: async (limit?: number) => (await client.syncRuns.listSyncRuns({ limit })).data as SyncRun[],
}
