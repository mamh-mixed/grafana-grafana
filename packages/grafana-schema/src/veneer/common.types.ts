import * as raw from '../common/common.gen';

import { MatcherConfig } from './dashboard.types';

export interface MapLayerOptions<TConfig = any> extends raw.MapLayerOptions {
  // Custom options depending on the type
  config?: TConfig;
  filterData?: MatcherConfig;
}

export interface DataQuery extends raw.DataQuery {
  // TODO remove explicit nulls
  datasource?: raw.DataSourceRef | null;
}

export interface DataSourceInstanceSettings<T extends raw.DataSourceJsonData = raw.DataSourceJsonData>
  extends raw.DataSourceInstanceSettings {
  jsonData: T;
}

export interface PluginMeta<T extends Record<string, unknown> = {}> extends raw.PluginMeta {
  jsonData?: T;
}

export interface DataSourcePluginMeta<T extends Record<string, unknown> = {}>
  extends PluginMeta<T>,
    raw.DataSourcePluginMeta {}

export * from '../common/common.gen';
