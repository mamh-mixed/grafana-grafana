import { useState } from 'react';
import { lastValueFrom } from 'rxjs';

import { Correlation } from '@grafana/data';
import { getDataSourceSrv } from '@grafana/runtime';

import { addCorrelation, deleteCorrelation } from './api';

// interface CorrelationData extends PartialCorrelationData {
//   target: PartialCorrelationData['target'];
// }

// interface PartialCorrelationData {
//   source: DataSourceInstanceSettings;
//   target?: DataSourceInstanceSettings;
//   label?: string;
//   description?: string;
// }

// const havingCorrelations = (ds: DataSourceInstanceSettings) => ds.correlations.length >= 1;
// const targetExists = (correlation: PartialCorrelationData): correlation is CorrelationData =>
//   correlation?.target !== undefined;

const getCorrelations = () => [];
// getDataSourceSrv()
//   .getList()
//   .filter(havingCorrelations)
//   .flatMap((ds) =>
//     ds.correlations.map(({ target, ...correlation }) => ({
//       ...correlation,
//       source: ds,
//       target: getDataSourceSrv().getInstanceSettings(target),
//     }))
//   )
//   .filter(targetExists);

export const useCorrelations = () => {
  const [correlations, setCorrelations] = useState(getCorrelations());

  const reload = async () => {
    await getDataSourceSrv().reload();
    setCorrelations(getCorrelations());
  };

  const add = (sourceUid: string, correlation: Correlation) => {
    return lastValueFrom(addCorrelation(sourceUid, correlation)).then(reload);
  };

  const remove = (sourceUid: string, targetUid: string) => {
    return lastValueFrom(deleteCorrelation(sourceUid, targetUid)).then(reload);
  };

  return { correlations, add, remove };
};
