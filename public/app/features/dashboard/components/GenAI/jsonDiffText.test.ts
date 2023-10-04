import { orderProperties, JSONArray } from './jsonDiffText'

describe('orderProperties', () => {
  it('should sort simple objects', () => {
    // Simplest possible case
    const before = {
      "firstProperty": "foo",
      "secondProperty": "bar"
    }

    const after = {
      "secondProperty": "bar",
      "firstProperty": "foo"
    }

    // Call the function to test
    const result = orderProperties(before, after);

    expect(result).toEqual(
      {
        "firstProperty": "foo",
        "secondProperty": "bar"
      }
    )
  });

  it('should sort arrays', () => {
    const result = orderProperties([0, 1], [1, 0]);

    expect(result).toEqual(
      [0,1]
    )
  })

  it('should handle nested objects', () => {
    const before = {
      "nested": {
        "firstProperty": "foo",
        "secondProperty": "bar"
      }
    }

    const after = {
      "nested": {
        "secondProperty": "bar",
        "firstProperty": "foo"
      }
    }

    const result = orderProperties(before, after);

    expect(result).toEqual({
      "nested": {
        "firstProperty": "foo",
        "secondProperty": "bar"
      }
    });
  });

  it('should handle arrays of objects with different order', () => {
    const before = [
      {"id": 1, "name": "Alice"},
      {"id": 2, "name": "Bob"}
    ];

    const after = [
      {"id": 2, "name": "Bob"},
      {"id": 1, "name": "Alice"}
    ];

    const result = orderProperties(before, after);

    expect(result).toEqual([
      {"id": 1, "name": "Alice"},
      {"id": 2, "name": "Bob"}
    ]);
  });

  it('should handle null values', () => {
    const before = {
      "a": null,
      "b": null
    };

    const after = {
      "b": null,
      "a": null
    };

    const result = orderProperties(before, after);

    expect(result).toEqual({
      "a": null,
      "b": null
    });
  });


  it('should handle empty objects', () => {
    const before = {};
    const after = {};

    const result = orderProperties(before, after);

    expect(result).toEqual({});
  });

  it('should handle empty arrays', () => {
    const before: any[] = [];
    const after: any[] = [];

    const result = orderProperties(before, after);

    expect(result).toEqual([]);
  });

  it('should handle deeply nested objects', () => {
    const before = {
      "a": {
        "b": {
          "c": "foo"
        }
      },
      "d": "bar"
    };

    const after = {
      "d": "bar",
      "a": {
        "b": {
          "c": "foo"
        }
      }
    };

    const result = orderProperties(before, after);

    expect(result).toEqual({
      "a": {
        "b": {
          "c": "foo"
        }
      },
      "d": "bar"
    });
  });

  it('should handle arrays of nested objects', () => {
    const before = [
      {"id": 1, "nested": {"name": "Alice"}},
      {"id": 2, "nested": {"name": "Bob"}}
    ];

    const after = [
      {"id": 2, "nested": {"name": "Bob"}},
      {"id": 1, "nested": {"name": "Alice"}}
    ];

    const result = orderProperties(before, after);

    expect(result).toEqual([
      {"id": 1, "nested": {"name": "Alice"}},
      {"id": 2, "nested": {"name": "Bob"}}
    ]);
  });

  it('should handle mixed arrays of objects and primitive values', () => {
    const before = [
      {"id": 1},
      42,
      [3, 2, 1]
    ];

    const after = [
      {"id": 1},
      [3, 2, 1],
      42
    ];

    const result = orderProperties(before, after);

    expect(result).toEqual([
      {"id": 1},
      42,
      [3, 2, 1],
    ]);
  });

  it('should handle arrays of objects with nested arrays', () => {
    const before = [
      {"id": 1, "values": [3, 2, 1]},
      {"id": 2, "values": [6, 5, 4]}
    ];

    const after = [
      {"id": 2, "values": [6, 5, 4]},
      {"id": 1, "values": [3, 2, 1]}
    ];

    const result = orderProperties(before, after);

    expect(result).toEqual([
      {"id": 1, "values": [3, 2, 1]},
      {"id": 2, "values": [6, 5, 4]}
    ]);
  });

  it('should handle arrays of arrays', () => {
    const before = [
      [1, 2, 3],
      [4, 5, 6]
    ];

    const after = [
      [4, 5, 6],
      [1, 2, 3]
    ];

    const result = orderProperties(before, after);

    expect(result).toEqual([
      [1, 2, 3],
      [4, 5, 6]
    ]);
  });

  it('should match reordered and modified arrays to nearest keys', () => {
    const before = [
      {id: "1", "name": "Alice", "country": "England"},
      {id: "2", "name": "Bob", "country": "America"},
      {id: "3", "name": "Charlie", "country": "Foxtrot"}
    ]

    const after: JSONArray = [
      {"name": "Charlie", "country": "Foxtrot"},
      {"name": "Alice"},
    ]

    const result = orderProperties(before, after);

    expect(result).toEqual([
      {"name": "Alice"},
      {"name": "Charlie", "country": "Foxtrot"},
    ])
  })
});
