/* eslint-disable */
/* tslint:disable */
// @ts-nocheck
/*
 * ---------------------------------------------------------------
 * ## THIS FILE WAS GENERATED VIA SWAGGER-TYPESCRIPT-API        ##
 * ##                                                           ##
 * ## AUTHOR: acacode                                           ##
 * ## SOURCE: https://github.com/acacode/swagger-typescript-api ##
 * ---------------------------------------------------------------
 */

export interface CountItem {
  count: number;
  name: string;
}

export interface ListRepositoriesResp {
  repositories: Repository[];
  total: number;
}

export interface Overview {
  forks: number;
  /** LastSyncedAt is when the latest succeeded synchronization finished, it is null if never synchronized. */
  lastSyncedAt?: string | null;
  module: string;
  owners: number;
  /** ReportedTotal is the approximate number shown in "Used by" on GitHub, including the forked and private repositories. */
  reportedTotal: number;
  repositories: number;
  /** Repository and Module are the repository and the Go module to collect the dependents of. */
  repository: string;
  /** Resolved is the number of dependents whose module version is resolved. */
  resolved: number;
  stars: number;
}

export interface OwnerBrief {
  avatarURL: string;
  htmlURL: string;
  login: string;
  type: "User" | "Organization";
}

export interface OwnerRank {
  avatarURL: string;
  htmlURL: string;
  login: string;
  repositories: number;
  stars: number;
  type: "User" | "Organization";
}

export interface Repository {
  archived: boolean;
  createdAt: string;
  description: string;
  fork: boolean;
  forks: number;
  fullName: string;
  /** GoModPath is the path of the go.mod requiring the module. If not found, it is empty along with GoVersion and ModuleVersion. */
  goModPath: string;
  goVersion: string;
  homepage: string;
  htmlURL: string;
  language: string;
  license: string;
  moduleVersion: string;
  name: string;
  openIssues: number;
  owner: OwnerBrief;
  pushedAt: string;
  relatedModules: string[];
  stars: number;
  syncedAt: string;
  topics: string[];
  watchers: number;
}

export interface SyncRun {
  discovered: number;
  error: string;
  failed: number;
  finishedAt?: string | null;
  id: number;
  removed: number;
  reportedTotal: number;
  skipped: number;
  startedAt: string;
  status: "running" | "succeeded" | "failed";
  synced: number;
}

export type GetOverviewData = Overview;

export type ListRepositoriesData = ListRepositoriesResp;

export type ListGoVersionsData = CountItem[];

export type ListLanguagesData = CountItem[];

export type ListModulesData = CountItem[];

export type ListTopOwnersData = OwnerRank[];

export type ListTopicsData = CountItem[];

export type ListVersionsData = CountItem[];

export type ListSyncRunsData = SyncRun[];

import type {
  AxiosInstance,
  AxiosRequestConfig,
  AxiosResponse,
  HeadersDefaults,
  ResponseType,
} from "axios";
import axios from "axios";

export type QueryParamsType = Record<string | number, any>;

export interface FullRequestParams
  extends Omit<AxiosRequestConfig, "data" | "params" | "url" | "responseType"> {
  /** set parameter to `true` for call `securityWorker` for this request */
  secure?: boolean;
  /** request path */
  path: string;
  /** content type of request body */
  type?: ContentType;
  /** query params */
  query?: QueryParamsType;
  /** format of response (i.e. response.json() -> format: "json") */
  format?: ResponseType;
  /** request body */
  body?: unknown;
}

export type RequestParams = Omit<
  FullRequestParams,
  "body" | "method" | "query" | "path"
>;

export interface ApiConfig<SecurityDataType = unknown>
  extends Omit<AxiosRequestConfig, "data" | "cancelToken"> {
  securityWorker?: (
    securityData: SecurityDataType | null,
  ) => Promise<AxiosRequestConfig | void> | AxiosRequestConfig | void;
  secure?: boolean;
  format?: ResponseType;
}

export enum ContentType {
  Json = "application/json",
  JsonApi = "application/vnd.api+json",
  FormData = "multipart/form-data",
  UrlEncoded = "application/x-www-form-urlencoded",
  Text = "text/plain",
}

export class HttpClient<SecurityDataType = unknown> {
  public instance: AxiosInstance;
  private securityData: SecurityDataType | null = null;
  private securityWorker?: ApiConfig<SecurityDataType>["securityWorker"];
  private secure?: boolean;
  private format?: ResponseType;

  constructor({
    securityWorker,
    secure,
    format,
    ...axiosConfig
  }: ApiConfig<SecurityDataType> = {}) {
    this.instance = axios.create({
      ...axiosConfig,
      baseURL: axiosConfig.baseURL || "/_",
    });
    this.secure = secure;
    this.format = format;
    this.securityWorker = securityWorker;
  }

  public setSecurityData = (data: SecurityDataType | null) => {
    this.securityData = data;
  };

  protected mergeRequestParams(
    params1: AxiosRequestConfig,
    params2?: AxiosRequestConfig,
  ): AxiosRequestConfig {
    const method = params1.method || (params2 && params2.method);

    return {
      ...this.instance.defaults,
      ...params1,
      ...(params2 || {}),
      headers: {
        ...((method &&
          this.instance.defaults.headers[
            method.toLowerCase() as keyof HeadersDefaults
          ]) ||
          {}),
        ...(params1.headers || {}),
        ...((params2 && params2.headers) || {}),
      },
    };
  }

  protected stringifyFormItem(formItem: unknown) {
    if (typeof formItem === "object" && formItem !== null) {
      return JSON.stringify(formItem);
    } else {
      return `${formItem}`;
    }
  }

  protected createFormData(input: Record<string, unknown>): FormData {
    if (input instanceof FormData) {
      return input;
    }
    return Object.keys(input || {}).reduce((formData, key) => {
      const property = input[key];
      const propertyContent: any[] =
        property instanceof Array ? property : [property];

      for (const formItem of propertyContent) {
        const isFileType = formItem instanceof Blob || formItem instanceof File;
        formData.append(
          key,
          isFileType ? formItem : this.stringifyFormItem(formItem),
        );
      }

      return formData;
    }, new FormData());
  }

  public request = async <T = any, _E = any>({
    secure,
    path,
    type,
    query,
    format,
    body,
    ...params
  }: FullRequestParams): Promise<AxiosResponse<T>> => {
    const secureParams =
      ((typeof secure === "boolean" ? secure : this.secure) &&
        this.securityWorker &&
        (await this.securityWorker(this.securityData))) ||
      {};
    const requestParams = this.mergeRequestParams(params, secureParams);
    const responseFormat = format || this.format || undefined;

    if (
      type === ContentType.FormData &&
      body &&
      body !== null &&
      typeof body === "object"
    ) {
      body = this.createFormData(body as Record<string, unknown>);
    }

    if (
      type === ContentType.Text &&
      body &&
      body !== null &&
      typeof body !== "string"
    ) {
      body = JSON.stringify(body);
    }

    return this.instance.request({
      ...requestParams,
      headers: {
        ...(requestParams.headers || {}),
        ...(type ? { "Content-Type": type } : {}),
      },
      params: query,
      responseType: responseFormat,
      data: body,
      url: path,
    });
  };
}

/**
 * @title Flamego Stats API
 * @version 1.0
 * @baseUrl /_
 * @contact
 */
export class Api<
  SecurityDataType extends unknown,
> extends HttpClient<SecurityDataType> {
  overview = {
    /**
     * @description The totals of the dependents, owners, stars and forks, along with the latest succeeded synchronization.
     *
     * @name GetOverview
     * @summary Get the overview
     * @request GET:/overview
     */
    getOverview: (
      query?: {
        /** Include the forked repositories, defaults to false */
        includeForks?: boolean;
      },
      params: RequestParams = {},
    ) =>
      this.request<GetOverviewData, string>({
        path: `/overview`,
        method: "GET",
        query: query,
        format: "json",
        ...params,
      }),
  };
  repositories = {
    /**
     * @description List the repositories depending on the module, sort=stars ranks by stars and sort=pushed lists the recently updated.
     *
     * @name ListRepositories
     * @summary List the dependents
     * @request GET:/repositories
     */
    listRepositories: (
      query?: {
        /** Page number, starting from 1 */
        page?: number;
        /** Page size, defaults to 20, at most 100 */
        pageSize?: number;
        /** Order of the list, defaults to stars */
        sort?: "stars" | "forks" | "pushed" | "created" | "name";
        /** Match the full name and the description */
        keyword?: string;
        /** Filter by the topic */
        topic?: string;
        /** Filter by the primary language */
        language?: string;
        /** Filter by the owner, case-insensitively */
        owner?: string;
        /** Filter by the module version, e.g. v1.9.4 */
        version?: string;
        /** Include the forked repositories, defaults to false */
        includeForks?: boolean;
      },
      params: RequestParams = {},
    ) =>
      this.request<ListRepositoriesData, string>({
        path: `/repositories`,
        method: "GET",
        query: query,
        format: "json",
        ...params,
      }),
  };
  stats = {
    /**
     * @description The MAJOR.MINOR distribution of the go directive in the go.mod of the dependents, in descending version order.
     *
     * @name ListGoVersions
     * @summary List the Go versions
     * @request GET:/stats/go-versions
     */
    listGoVersions: (
      query?: {
        /** Include the forked repositories, defaults to false */
        includeForks?: boolean;
      },
      params: RequestParams = {},
    ) =>
      this.request<ListGoVersionsData, string>({
        path: `/stats/go-versions`,
        method: "GET",
        query: query,
        format: "json",
        ...params,
      }),

    /**
     * @description The primary languages ordered by the number of dependents.
     *
     * @name ListLanguages
     * @summary List the languages
     * @request GET:/stats/languages
     */
    listLanguages: (
      query?: {
        /** Number of items, defaults to 20, at most 100 */
        limit?: number;
        /** Include the forked repositories, defaults to false */
        includeForks?: boolean;
      },
      params: RequestParams = {},
    ) =>
      this.request<ListLanguagesData, string>({
        path: `/stats/languages`,
        method: "GET",
        query: query,
        format: "json",
        ...params,
      }),

    /**
     * @description The other modules of the same organization (e.g. github.com/flamego/session) used by the dependents, ordered by the number of dependents.
     *
     * @name ListModules
     * @summary Rank the related modules
     * @request GET:/stats/modules
     */
    listModules: (
      query?: {
        /** Number of items, defaults to 20, at most 100 */
        limit?: number;
        /** Include the forked repositories, defaults to false */
        includeForks?: boolean;
      },
      params: RequestParams = {},
    ) =>
      this.request<ListModulesData, string>({
        path: `/stats/modules`,
        method: "GET",
        query: query,
        format: "json",
        ...params,
      }),

    /**
     * @description The owners ordered by the number of dependents, then by the total stars.
     *
     * @name ListTopOwners
     * @summary Rank the owners
     * @request GET:/stats/owners
     */
    listTopOwners: (
      query?: {
        /** Number of items, defaults to 20, at most 100 */
        limit?: number;
        /** Include the forked repositories, defaults to false */
        includeForks?: boolean;
      },
      params: RequestParams = {},
    ) =>
      this.request<ListTopOwnersData, string>({
        path: `/stats/owners`,
        method: "GET",
        query: query,
        format: "json",
        ...params,
      }),

    /**
     * @description The repository topics ordered by the number of dependents.
     *
     * @name ListTopics
     * @summary List the topics
     * @request GET:/stats/topics
     */
    listTopics: (
      query?: {
        /** Number of items, defaults to 20, at most 100 */
        limit?: number;
        /** Include the forked repositories, defaults to false */
        includeForks?: boolean;
      },
      params: RequestParams = {},
    ) =>
      this.request<ListTopicsData, string>({
        path: `/stats/topics`,
        method: "GET",
        query: query,
        format: "json",
        ...params,
      }),

    /**
     * @description The distribution of the module versions in the go.mod of the dependents, in descending version order.
     *
     * @name ListVersions
     * @summary List the module versions
     * @request GET:/stats/versions
     */
    listVersions: (
      query?: {
        /** Group by the exact or the minor versions, defaults to exact */
        granularity?: "exact" | "minor";
        /** Include the forked repositories, defaults to false */
        includeForks?: boolean;
      },
      params: RequestParams = {},
    ) =>
      this.request<ListVersionsData, string>({
        path: `/stats/versions`,
        method: "GET",
        query: query,
        format: "json",
        ...params,
      }),
  };
  syncRuns = {
    /**
     * @description The latest synchronization runs, the newest first. The discovered counts chart the trend of the dependents.
     *
     * @name ListSyncRuns
     * @summary List the synchronization runs
     * @request GET:/sync-runs
     */
    listSyncRuns: (
      query?: {
        /** Number of items, defaults to 20, at most 100 */
        limit?: number;
      },
      params: RequestParams = {},
    ) =>
      this.request<ListSyncRunsData, string>({
        path: `/sync-runs`,
        method: "GET",
        query: query,
        format: "json",
        ...params,
      }),
  };
}
