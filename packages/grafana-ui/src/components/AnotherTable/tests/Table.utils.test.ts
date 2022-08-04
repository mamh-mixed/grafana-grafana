import { getColumns, getData } from '../Table.utils';

describe('Components/Table/Utils', () => {
  describe('getColumns()', () => {
    test('transforms the format that we use to a "react-table" compatible format', () => {
      const input = {
        fields: [
          {
            id: 'field-1',
            name: 'Field 1',
            values: [1, 2, 3],
          },
          {
            id: 'field-2',
            name: 'Field 2',
            values: [4, 5, 6],
          },
        ],
      };

      const output = [
        {
          Header: 'Field 1',
          accessor: 'field-1',
        },
        {
          Header: 'Field 2',
          accessor: 'field-2',
        },
      ];

      expect(getColumns(input)).toMatchObject(output);
    });

    test('can assign custom renderers to the columns', () => {
      const input = {
        fields: [
          {
            id: 'field-1',
            name: 'Field 1',
            values: [1, 2, 3],
            render: (x: number) => x * 2,
          },
        ],
      };
      const output = getColumns(input);
      const [column1] = output;

      expect(column1.Cell({ value: 1 })).toEqual(2);
    });
  });

  describe('getData()', () => {
    test('transforms the format that we use to a "react-table" compatible format', () => {
      const input = {
        fields: [
          {
            id: 'field-1',
            name: 'Field 1',
            values: [1, 2, 3],
          },
          {
            id: 'field-2',
            name: 'Field 2',
            values: [4, 5, 6],
          },
        ],
      };

      const output = [
        {
          'field-1': 1,
          'field-2': 4,
        },
        {
          'field-1': 2,
          'field-2': 5,
        },
        {
          'field-1': 3,
          'field-2': 6,
        },
      ];

      expect(getData(input)).toEqual(output);
    });
  });
});
