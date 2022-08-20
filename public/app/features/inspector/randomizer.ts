import { DataFrameJSON, Labels } from '@grafana/data';

export function newLetterRandomizer(): (v: string) => string {
  const upper = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ';
  const lower = 'abcdefghijklmnopqrstuvwxyz';
  const charactersLength = upper.length;

  const history = new Map<string, string>();
  return (v: string) => {
    const old = history.get(v);
    if (old != null) {
      return old;
    }
    const r = [...v]
      .map((c) => {
        if (c.toLowerCase() && c !== c.toUpperCase()) {
          return lower.charAt(Math.floor(Math.random() * charactersLength));
        }
        if (c.toUpperCase() && c !== c.toUpperCase()) {
          return upper.charAt(Math.floor(Math.random() * charactersLength));
        }
        return c;
      })
      .join('');
    history.set(v, r);
    return r;
  };
}

export interface Randomize {
  names?: boolean;
  labels?: boolean;
  values?: boolean;
}

export function randomizeData(data: DataFrameJSON[], opts?: Randomize): DataFrameJSON[] {
  if (!opts || !(opts.labels || opts.names || opts.values)) {
    return data;
  }
  const keepNames = new Set(['Time', 'time', 'Value', 'value']);

  const rand = newLetterRandomizer();
  return data.map((s) => {
    let { schema, data } = s;
    if (schema) {
      if (opts.labels) {
        for (const f of schema.fields) {
          if (f.labels) {
            const labels: Labels = {};
            for (const [key, value] of Object.entries(f.labels)) {
              labels[key] = rand(value);
            }
            f.labels = labels;
          }
        }
      }
      if (opts.names) {
        for (const f of schema.fields) {
          if (f.name?.length && !keepNames.has(f.name)) {
            f.name = rand(f.name);
          }
        }
      }
      if (opts.values) {
        // TODO... change string values
      }
    }
    return { schema, data };
  });
}
