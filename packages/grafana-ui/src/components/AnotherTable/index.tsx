import { cx, css } from '@emotion/css';
import React, { useMemo, Fragment, ReactNode } from 'react';
import {
  CellProps,
  SortByFn,
  useExpanded,
  useSortBy,
  useTable,
  Column as RTColumn,
  DefaultSortTypes,
} from 'react-table';

import { GrafanaTheme2 } from '@grafana/data';

import { useStyles2 } from '../../themes';
import { Icon } from '../Icon/Icon';
import { IconButton } from '../IconButton/IconButton';

import { getColumns } from './Table.utils';

const getStyles = (theme: GrafanaTheme2) => ({
  table: css`
    border-radius: ${theme.shape.borderRadius()};
    border: solid 1px ${theme.colors.border.weak};
    background-color: ${theme.colors.background.secondary};

    td,
    th {
      padding: ${theme.spacing(1)};
      min-width: ${theme.spacing(3)};
    }
  `,
  th: css`
    position: relative;
  `,
  expanderContainer: css`
    display: flex;
    align-items: center;
    height: 100%;
  `,
  sortIcon: css`
    position: absolute;
    width: 16px;
    overflow: hidden;
  `,
  evenRow: css`
    background: ${theme.colors.background.primary};
  `,
  shrink: css`
    width: 0%;
  `,
});

export interface Column<TableData extends object> {
  id: string;
  cell?: (props: CellProps<TableData>) => ReactNode;
  header?: (() => ReactNode | string) | string;
  sortType?: DefaultSortTypes | SortByFn<TableData>;
  shrink?: boolean;
}

interface Props<TableData extends object> {
  columns: Array<Column<TableData>>;
  data: TableData[];
  expandable?: boolean;
  renderExpandedRow?: (row: TableData) => JSX.Element;
  className?: string;
}

export function AnotherTable<TableData extends object>({
  data,
  className,
  expandable = false,
  columns,
  renderExpandedRow,
}: Props<TableData>) {
  const styles = useStyles2(getStyles);
  const tableColumns = useMemo<Array<RTColumn<TableData>>>(() => {
    const cols = getColumns<TableData>(columns);

    if (expandable) {
      cols.unshift({
        id: '__expander',
        Cell: ({ row }: CellProps<TableData, void>) => (
          <div className={styles.expanderContainer}>
            <IconButton
              // @ts-expect-error react-table doesn't ship with useExpanded types and we can't use declaration merging without affecting the table viz
              name={row.isExpanded ? 'angle-down' : 'angle-right'}
              // @ts-expect-error same as the line above
              {...row.getToggleRowExpandedProps({})}
            />
          </div>
        ),
        disableSortBy: true,
        width: 0,
      });
    }
    return cols;
  }, [columns, expandable, styles.expanderContainer]);
  const { getTableProps, getTableBodyProps, headerGroups, rows, prepareRow } = useTable<TableData>(
    {
      columns: tableColumns,
      data,
    },
    useSortBy,
    useExpanded
  );
  // This should be called only for rows thar we'd want to actually render, which is all at this stage.
  // We may want to revisit this if we decide to add pagination and/or virtualized tables.
  rows.forEach(prepareRow);

  return (
    <table {...getTableProps()} className={cx(styles.table, className)}>
      <thead>
        {headerGroups.map((headerGroup) => (
          // .getHeaderGroupProps() returns with a key as well, so the <tr> will actually have a key.
          // eslint-disable-next-line react/jsx-key
          <tr {...headerGroup.getHeaderGroupProps()}>
            {headerGroup.headers.map((column) => (
              // .getHeaderProps() returns with a key as well, so the <th> will actually have a key.
              // eslint-disable-next-line react/jsx-key
              <th
                {...column.getHeaderProps(column.getSortByToggleProps())}
                className={cx(styles.th, column.width === 0 && styles.shrink)}
              >
                {column.render('Header')}

                {column.isSorted && (
                  <span className={styles.sortIcon}>
                    <Icon name={column.isSortedDesc ? 'angle-down' : 'angle-up'} />
                  </span>
                )}
              </th>
            ))}
          </tr>
        ))}
      </thead>

      <tbody {...getTableBodyProps()}>
        {rows.map((row, rowIndex) => {
          const { key, ...otherRowProps } = row.getRowProps();

          return (
            // .getRowProps() returns with a key as well, so the <tr> here will actually have a key.
            // eslint-disable-next-line react/jsx-key
            <Fragment key={key}>
              <tr className={cx(rowIndex % 2 === 0 && styles.evenRow)} {...otherRowProps}>
                {row.cells.map((cell) => {
                  return (
                    // .getCellProps() will return with a key as well, so the <td> here will actually have a key.
                    // eslint-disable-next-line react/jsx-key
                    <td {...cell.getCellProps()}>{cell.render('Cell')}</td>
                  );
                })}
              </tr>
              {
                // @ts-expect-error react-table doesn't ship with useExpanded types and we can't use declaration merging without affecting the table viz
                row.isExpanded && (
                  <tr className={cx(rowIndex % 2 === 0 && styles.evenRow)} {...otherRowProps}>
                    <td colSpan={row.cells.length}>{renderExpandedRow?.(row.original)}</td>
                  </tr>
                )
              }
            </Fragment>
          );
        })}
      </tbody>
    </table>
  );
}
