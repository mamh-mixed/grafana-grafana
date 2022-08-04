import { noop } from 'lodash';
import { useAsync } from 'react-use';

import { Correlation, DataSourceApi } from '@grafana/data';
import { getDataSourceSrv } from '@grafana/runtime';
import { useGrafana } from 'app/core/context/GrafanaContext';

const havingUID = (uid: string) => (ds: DataSourceApi) => ds.uid === uid;

export interface CorrelationData extends Omit<Correlation, 'sourceUID' | 'targetUID'> {
  source: DataSourceApi;
  target: DataSourceApi;
}

const toEnrichedCorrelationData = (correlations: Correlation[]): Promise<CorrelationData[]> => {
  const DSSet = new Set(correlations.flatMap(({ sourceUID, targetUID }) => [sourceUID, targetUID]));

  return Promise.all(Array.from(DSSet, (uid) => getDataSourceSrv().get(uid))).then((datasources) =>
    correlations.map(({ sourceUID, targetUID, ...correlation }) => ({
      ...correlation,
      source: datasources.find(havingUID(sourceUID))!,
      target: datasources.find(havingUID(targetUID))!,
    }))
  );
};

export const useCorrelations = () => {
  const {
    backend: { get },
  } = useGrafana();
  const getCorrelations = () => get<Correlation[]>('/api/datasources/correlations').then(toEnrichedCorrelationData);

  const { loading, value: correlations } = useAsync(getCorrelations);

  // const add = (sourceUid: string, correlation: Correlation) => {
  //   return lastValueFrom(addCorrelation(sourceUid, correlation)).then(reload);
  // };

  // const remove = (sourceUid: string, targetUid: string) => {
  //   return lastValueFrom(deleteCorrelation(sourceUid, targetUid)).then(reload);
  // };

  const remove = noop;
  const add = noop;

  return { loading, correlations, add, remove };
};
