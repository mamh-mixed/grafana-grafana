import { Column as RTColumn } from 'react-table';

import { Column } from '.';

// Returns the columns in a "react-table" acceptable format
export function getColumns<K extends object, T = Array<Column<K>>>(columns: T): Array<RTColumn<K>> {
  // @ts-expect-error react-table expects each column key(id) to have data associated with it and therefore complains about
  // column.id being string and not keyof T (where T is the data object)
  // We do not want to be that strict as we simply pass undefined to cells that do not have data associated with them.
  return columns.map((column) => ({
    Header: column.header || (() => null),
    accessor: column.id,
    sortType: column.sortType || 'alphanumeric',
    disableSortBy: !Boolean(column.sortType),
    width: column.shrink ? 0 : undefined,
    ...(column.cell && { Cell: column.cell }),
  }));
}
