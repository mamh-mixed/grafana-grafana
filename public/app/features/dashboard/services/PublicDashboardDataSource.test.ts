import { of } from 'rxjs';

import { DataQueryRequest, DataSourceRef } from '@grafana/data';
import { BackendSrvRequest } from '@grafana/runtime';

import { PublicDashboardDataSource } from './PublicDashboardDataSource';

const mockDatasourceRequest = jest.fn();

const backendSrv = {
  fetch: (options: BackendSrvRequest) => {
    return of(mockDatasourceRequest(options));
  },
};

jest.mock('@grafana/runtime', () => ({
  ...jest.requireActual('@grafana/runtime'),
  getBackendSrv: () => backendSrv,
  getDataSourceSrv: () => {
    return {
      getInstanceSettings: (ref?: DataSourceRef) => ({ type: ref?.type ?? '?', uid: ref?.uid ?? '?' }),
    };
  },
}));

describe('PublicDashboardDatasource', () => {
  test('Fetches results from the pubdash query endpoint', () => {
    mockDatasourceRequest.mockReset();
    mockDatasourceRequest.mockReturnValue(Promise.resolve({}));

    const ds = new PublicDashboardDataSource();
    const panelId = 1;
    const publicDashboardAccessToken = 'abc123';
    const query = jest.mocked({} as DataQueryRequest);

    query.maxDataPoints = 10;
    query.intervalMs = 5000;
    query.targets = [{ refId: 'A' }, { refId: 'B', datasource: { type: 'sample' } }];
    query.panelId = panelId;
    query.publicDashboardAccessToken = publicDashboardAccessToken;

    ds.query(query);

    const mock = mockDatasourceRequest.mock;

    expect(mock.calls.length).toBe(1);
    expect(mock.lastCall[0].url).toEqual(
      `/api/public/dashboards/${publicDashboardAccessToken}/panels/${panelId}/query`
    );
  });
});
