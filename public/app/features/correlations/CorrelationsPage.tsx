import { css } from '@emotion/css';
import React, { memo, useCallback, useMemo, useState } from 'react';
import { CellProps, SortByFn } from 'react-table';

import { GrafanaTheme2 } from '@grafana/data';
import { AnotherTable, Button, DeleteButton, HorizontalGroup, useStyles2 } from '@grafana/ui';
import { Column } from '@grafana/ui/src/components/AnotherTable';
import EmptyListCTA from 'app/core/components/EmptyListCTA/EmptyListCTA';
import { Page } from 'app/core/components/Page/Page';

import { useNavModel } from '../../core/hooks/useNavModel';

import { AddCorrelationForm } from './AddCorrelationForm';
import { CorrelationDetailsFormPart } from './CorrelationDetailsFormPart';
import { CorrelationData, useCorrelations } from './useCorrelations';

const styles = {
  table: css`
    width: 100%;
  `,
};

const sortDatasource: SortByFn<CorrelationData> = (a, b, column) =>
  a.values[column].name.localeCompare(b.values[column].name);

export default function CorrelationsPage() {
  const navModel = useNavModel('correlations');
  const [isAdding, setIsAdding] = useState(false);
  const { correlations, add, remove } = useCorrelations();

  const RowActions = useCallback(
    ({
      row: {
        original: { source, target },
      },
    }: CellProps<CorrelationData, void>) => <DeleteButton onConfirm={() => remove(source.uid, target.uid)} />,
    [remove]
  );

  const columns = useMemo<Array<Column<CorrelationData>>>(
    () => [
      {
        id: 'source',
        header: 'Source',
        cell: DataSourceCell,
        sortType: sortDatasource,
      },
      {
        id: 'target',
        header: 'Target',
        cell: DataSourceCell,
        sortType: sortDatasource,
      },
      { id: 'label', header: 'Label', sortType: 'alphanumeric' },
      { id: 'actions', cell: RowActions, shrink: true },
    ],
    [RowActions]
  );

  const data = useMemo(() => correlations, [correlations]);

  if (!data) {
    return null;
  }

  return (
    <>
      <Page navModel={navModel}>
        <Page.Contents>
          {data.length === 0 && !isAdding && (
            <EmptyListCTA
              title="You haven't defined any correlation yet."
              buttonIcon="sitemap"
              onClick={() => setIsAdding(true)}
              buttonTitle="Add correlation"
            />
          )}

          {data.length >= 1 && (
            <div>
              <HorizontalGroup justify="space-between">
                <div>
                  <h4>Correlations</h4>
                  <p>description</p>
                </div>
                <Button icon="plus" onClick={() => setIsAdding(true)} disabled={isAdding}>
                  Add new
                </Button>
              </HorizontalGroup>
            </div>
          )}

          {isAdding && <AddCorrelationForm onClose={() => setIsAdding(false)} onSubmit={add} />}

          {data.length >= 1 && (
            <AnotherTable
              renderExpandedRow={(row) => <CorrelationDetailsFormPart />}
              columns={columns}
              data={data}
              className={styles.table}
              expandable
            />
          )}
        </Page.Contents>
      </Page>
    </>
  );
}

const getDatasourceCellStyles = (theme: GrafanaTheme2) => ({
  root: css`
    display: flex;
    align-items: center;
  `,
  dsLogo: css`
    margin-right: ${theme.spacing()};
    height: 16px;
    width: 16px;
  `,
});

const DataSourceCell = memo(
  function DataSourceCell({
    cell: { value },
  }: CellProps<CorrelationData, CorrelationData['source'] | CorrelationData['target']>) {
    const styles = useStyles2(getDatasourceCellStyles);

    return (
      <span className={styles.root}>
        <img src={value.meta.info.logos.small} className={styles.dsLogo} />
        {value.name}
      </span>
    );
  },
  ({ cell: { value } }, { cell: { value: prevValue } }) => {
    return value.type === prevValue.type && value.name === prevValue.name;
  }
);
