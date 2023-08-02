import React, { PureComponent } from 'react';

import { DataQuery, getDataSourceRef } from '@grafana/data';
import { locationService } from '@grafana/runtime';
import { getDatasourceSrv } from 'app/features/plugins/datasource_srv';
import { QueryGroup } from 'app/features/query/components/QueryGroup';
import { MIXED_DATASOURCE_NAME } from 'app/plugins/datasource/mixed/MixedDataSource';
import { QueryGroupDataSource, QueryGroupOptions } from 'app/types';

import { PanelModel } from '../../state';
import {
  getDashboardUidFromLocation,
  getLastUsedDatasourceFromStorage,
  updateDatasourceUidLastUsedDatasource,
} from '../../utils/dashboard';

interface Props {
  /** Current panel */
  panel: PanelModel;
  /** Added here to make component re-render when queries change from outside */
  queries: DataQuery[];
}

export class PanelEditorQueries extends PureComponent<Props> {
  constructor(props: Props) {
    super(props);
  }

  updateLastUsedDatasource = (datasource: QueryGroupDataSource) => {
    if (!datasource.uid) {
      return;
    }
    const dashboardUid = getDashboardUidFromLocation();

    // if datasource is MIXED reset datasource uid in storage, because Mixed datasource can contain multiple ds
    if (datasource.uid === MIXED_DATASOURCE_NAME) {
      return updateDatasourceUidLastUsedDatasource(dashboardUid!, '');
    }
    updateDatasourceUidLastUsedDatasource(dashboardUid, datasource.uid);
  };

  buildQueryOptions(panel: PanelModel): QueryGroupOptions {
    const dataSource: QueryGroupDataSource = panel.datasource ?? {
      default: true,
    };
    const datasourceSettings = getDatasourceSrv().getInstanceSettings(dataSource);

    // store last datasource used in local storage
    this.updateLastUsedDatasource(dataSource);
    return {
      cacheTimeout: datasourceSettings?.meta.queryOptions?.cacheTimeout ? panel.cacheTimeout : undefined,
      dataSource: {
        default: datasourceSettings?.isDefault,
        type: datasourceSettings?.type,
        uid: datasourceSettings?.uid,
      },
      queryCachingTTL: datasourceSettings?.cachingConfig?.enabled ? panel.queryCachingTTL : undefined,
      queries: panel.targets,
      maxDataPoints: panel.maxDataPoints,
      minInterval: panel.interval,
      timeRange: {
        from: panel.timeFrom,
        shift: panel.timeShift,
        hide: panel.hideTimeOverride,
      },
    };
  }

  async componentDidMount() {
    const { panel } = this.props;

    // If the panel model has no datasource property load the default data source property and update the persisted model
    // Because this part of the panel model is not in redux yet we do a forceUpdate.
    if (!panel.datasource) {
      let ds;
      // check if we have last used datasource from local storage
      // get dashboard uid from the window location
      const dashboardUid = getDashboardUidFromLocation();
      const lastUsedDatasource = getLastUsedDatasourceFromStorage(dashboardUid!);
      // do we have a last used datasource for this dashboard
      if (lastUsedDatasource?.datasourceUid !== null) {
        // get datasource from uid
        ds = getDatasourceSrv().getInstanceSettings(lastUsedDatasource?.datasourceUid);
      }
      // else load default datasource
      if (!ds) {
        ds = getDatasourceSrv().getInstanceSettings(null);
      }
      panel.datasource = getDataSourceRef(ds!);
      this.forceUpdate();
    }
  }

  onRunQueries = () => {
    this.props.panel.refresh();
  };

  onOpenQueryInspector = () => {
    locationService.partial({
      inspect: this.props.panel.id,
      inspectTab: 'query',
    });
  };

  onOptionsChange = (options: QueryGroupOptions) => {
    const { panel } = this.props;

    panel.updateQueries(options);

    if (options.dataSource.uid !== panel.datasource?.uid) {
      // trigger queries when changing data source
      setTimeout(this.onRunQueries, 10);
    }

    this.forceUpdate();
  };

  render() {
    const { panel } = this.props;

    // If no panel data soruce set, wait with render. Will be set to default in componentDidMount
    if (!panel.datasource) {
      return null;
    }

    const options = this.buildQueryOptions(panel);

    return (
      <QueryGroup
        options={options}
        queryRunner={panel.getQueryRunner()}
        onRunQueries={this.onRunQueries}
        onOpenQueryInspector={this.onOpenQueryInspector}
        onOptionsChange={this.onOptionsChange}
      />
    );
  }
}
