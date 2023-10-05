import { from, Observable } from 'rxjs';
import { map } from 'rxjs/operators';

import { CustomVariableSupport, DataQueryRequest, MetricFindValue, ScopedVars } from '@grafana/data';

import { LokiVariableQueryEditor } from './components/VariableQueryEditor';
import { LokiDatasource } from './datasource';
import { LokiVariableQuery } from './types';

export class LokiVariableSupport extends CustomVariableSupport<LokiDatasource, LokiVariableQuery> {
  editor = LokiVariableQueryEditor;

  constructor(private datasource: LokiDatasource) {
    super();
  }

  async execute(query: LokiVariableQuery, scopedVars: ScopedVars) {
    return this.datasource.metricFindQuery(query, { scopedVars });
  }

  query(request: DataQueryRequest<LokiVariableQuery>): Observable<{ data: MetricFindValue[] }> {
    const result = this.execute(request.targets[0], request.scopedVars);

    return from(result).pipe(map((data) => ({ data })));
  }
}
