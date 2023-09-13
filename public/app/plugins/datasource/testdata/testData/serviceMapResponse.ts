import { FieldColorModeId, FieldType, NodeGraphDataFrameFieldNames } from '@grafana/data';

export const nodes = {
  fields: [
    {
      name: NodeGraphDataFrameFieldNames.id,
      type: FieldType.string,
      config: {
        links: [
          {
            title: 'Traces/All',
            url: '',
            internal: {
              query: {
                queryType: 'getTraceSummaries',
                query: 'service(id(name: "${__data.fields.name}", type: "${__data.fields.type}"))',
              },
              datasourceUid: 'Ax4erxHGz',
              datasourceName: 'Trace data source',
            },
          },
          {
            title: 'Traces/OK',
            url: '',
            internal: {
              query: {
                queryType: 'getTraceSummaries',
                query: 'service(id(name: "${__data.fields.name}", type: "${__data.fields.type}")) { ok = true }',
              },
              datasourceUid: 'Ax4erxHGz',
              datasourceName: 'Trace data source',
            },
          },
          {
            title: 'Traces/Errors',
            url: '',
            internal: {
              query: {
                queryType: 'getTraceSummaries',
                query: 'service(id(name: "${__data.fields.name}", type: "${__data.fields.type}")) { error = true }',
              },
              datasourceUid: 'Ax4erxHGz',
              datasourceName: 'Trace data source',
            },
          },
          {
            title: 'Traces/Faults',
            url: '',
            internal: {
              query: {
                queryType: 'getTraceSummaries',
                query: 'service(id(name: "${__data.fields.name}", type: "${__data.fields.type}")) { fault = true }',
              },
              datasourceUid: 'Ax4erxHGz',
              datasourceName: 'Trace data source',
            },
          },
        ],
      },
      values: [0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14],
    },
    {
      name: NodeGraphDataFrameFieldNames.title,
      type: FieldType.string,
      config: { displayName: 'Name' },
      values: [
        'auth',
        'products',
        'customers',
        'orders',
        'products',
        'orders',
        'api',
        'shipping',
        'orders',
        'execute-api',
        'shipping',
        'www',
        'api',
        'www',
        'products',
      ],
    },
    {
      name: NodeGraphDataFrameFieldNames.subTitle,
      type: FieldType.string,
      config: { displayName: 'Type' },
      values: [
        'Compute',
        'SQL',
        'SQL',
        'SQL',
        'remote',
        'Function',
        'Compute',
        'Function',
        'Function',
        'remote',
        'Function',
        'Compute',
        'client',
        'client',
        'Compute',
      ],
    },
    {
      name: NodeGraphDataFrameFieldNames.mainStat,
      type: FieldType.number,
      config: { unit: 'ms/t', displayName: 'Average response time' },
      values: [
        3.5394042646735553, 15.906441318223264, 4.913011921591567, 7.4163203042094095, 1092, 22.85961441405067,
        56.135855729084696, 4.45946191601527, 12.818300278280843, 4.25, 12.565442646791492, 77.63447512700567,
        40.387096774193544, 77.63447512700567, 27.648950187374872,
      ],
    },
    {
      name: NodeGraphDataFrameFieldNames.secondaryStat,
      type: FieldType.number,
      config: { unit: 't/min', displayName: 'Transactions per minute' },
      values: [
        50.56317154501667, 682.4, 512.8416666666667, 125.64444444444445, 0.005585812037424941, 137.59722222222223,
        300.0527777777778, 30.582348853370394, 125.77222222222223, 0.028706417080318163, 30.582348853370394, 165.675,
        0.100021510002151, 165.675, 162.33055555555555,
      ],
    },
    {
      name: NodeGraphDataFrameFieldNames.arc + 'success',
      type: FieldType.number,
      config: { color: { mode: FieldColorModeId.Fixed, fixedColor: 'green' }, displayName: 'Sucesss' },
      values: [
        0.9338865684765882, 1, 1, 1, 0.5, 1, 0.9901128505170387, 0.9069260134520997, 1, 0, 0.9069260134520997,
        0.9624432037288534, 0, 0.9624432037288534, 0.9824945669843769,
      ],
    },
    {
      name: NodeGraphDataFrameFieldNames.arc + 'faults',
      type: FieldType.number,
      config: { color: { mode: FieldColorModeId.Fixed, fixedColor: 'red' }, displayName: 'Faults' },
      values: [
        0, 0, 0, 0, 0.5, 0, 0.009479813736472288, 0, 0, 0, 0, 0.017168821152524185, 0, 0.017168821152524185,
        0.01750543301562313,
      ],
    },
    {
      name: NodeGraphDataFrameFieldNames.arc + 'errors',
      type: FieldType.number,
      config: { color: { mode: FieldColorModeId.Fixed, fixedColor: 'semi-dark-yellow' }, displayName: 'Errors' },
      values: [
        0.06611343152341174, 0, 0, 0, 0, 0, 0.0004073357464890436, 0.09307398654790038, 0, 1, 0.09307398654790038,
        0.02038797511862247, 1, 0.02038797511862247, 0,
      ],
    },
    {
      name: NodeGraphDataFrameFieldNames.arc + 'throttled',
      type: FieldType.number,
      config: { color: { mode: FieldColorModeId.Fixed, fixedColor: 'purple' }, displayName: 'Throttled' },
      values: [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0],
    },
  ],
  meta: { preferredVisualisationType: 'nodeGraph' as const },
  name: 'nodes',
};

export const edges = {
  fields: [
    {
      name: NodeGraphDataFrameFieldNames.id,
      type: FieldType.string,
      config: {
        links: [
          {
            title: 'Traces/All',
            url: '',
            internal: {
              query: {
                queryType: 'getTraceSummaries',
                query: 'edge("${__data.fields.sourceName}", "${__data.fields.targetName}")',
              },
              datasourceUid: 'Ax4erxHGz',
              datasourceName: 'Trace data source',
            },
          },
          {
            title: 'Traces/OK',
            url: '',
            internal: {
              query: {
                queryType: 'getTraceSummaries',
                query: 'edge("${__data.fields.sourceName}", "${__data.fields.targetName}") { ok = true }',
              },
              datasourceUid: 'Ax4erxHGz',
              datasourceName: 'Trace data source',
            },
          },
          {
            title: 'Traces/Errors',
            url: '',
            internal: {
              query: {
                queryType: 'getTraceSummaries',
                query: 'edge("${__data.fields.sourceName}", "${__data.fields.targetName}") { error = true }',
              },
              datasourceUid: 'Ax4erxHGz',
              datasourceName: 'Trace data source',
            },
          },
          {
            title: 'Traces/Faults',
            url: '',
            internal: {
              query: {
                queryType: 'getTraceSummaries',
                query: 'edge("${__data.fields.sourceName}", "${__data.fields.targetName}") { fault = true }',
              },
              datasourceUid: 'Ax4erxHGz',
              datasourceName: 'Trace data source',
            },
          },
        ],
      },
      values: [
        '0__2',
        '5__8',
        '6__0',
        '6__5',
        '6__9',
        '6__2',
        '6__14',
        '6__4',
        '8__3',
        '10__7',
        '11__0',
        '11__6',
        '12__6',
        '13__11',
        '14__1',
        '14__2',
        '14__10',
      ],
    },
    {
      name: NodeGraphDataFrameFieldNames.source,
      type: FieldType.string,
      config: {},
      values: [0, 5, 6, 6, 6, 6, 6, 6, 8, 10, 11, 11, 12, 13, 14, 14, 14],
    },
    {
      name: 'sourceName',
      type: FieldType.string,
      config: {},
      values: [
        'auth',
        'orders',
        'api',
        'api',
        'api',
        'api',
        'api',
        'api',
        'orders',
        'shipping',
        'www',
        'www',
        'api',
        'www',
        'products',
        'products',
        'products',
      ],
    },
    {
      name: NodeGraphDataFrameFieldNames.target,
      type: FieldType.string,
      config: {},
      values: [2, 8, 0, 5, 9, 2, 14, 4, 3, 7, 0, 6, 6, 11, 1, 2, 10],
    },
    {
      name: 'targetName',
      type: FieldType.string,
      config: {},
      values: [
        'customers',
        'orders',
        'auth',
        'orders',
        'execute-api',
        'customers',
        'products',
        'products',
        'orders',
        'shipping',
        'auth',
        'api',
        'api',
        'www',
        'products',
        'customers',
        'shipping',
      ],
    },
    {
      name: NodeGraphDataFrameFieldNames.mainStat,
      type: FieldType.string,
      config: { displayName: 'Response percentage' },
      values: [
        'Success 100.00%',
        'Success 100.00%',
        'Success 100.00%',
        'Success 100.00%',
        'Errors 100.00%',
        'Success 100.00%',
        'Faults 1.75%',
        'Faults 50.00%',
        'Success 100.00%',
        'Errors 9.31%',
        'Errors 6.62%',
        'Faults 1.13%',
        'Errors 100.00%',
        'Faults 1.72%',
        'Success 100.00%',
        'Success 100.00%',
        'Faults 9.30%',
      ],
    },
    {
      name: NodeGraphDataFrameFieldNames.secondaryStat,
      type: FieldType.number,
      config: { unit: 't/min', displayName: 'Transactions per minute' },
      values: [
        50.56317154501667, 125.77222222222223, 0.03333333333333333, 137.59722222222223, 0.022222222222222223,
        299.96666666666664, 162.33055555555555, 0.005555555555555556, 125.64444444444445, 30.582348853370394,
        50.51111111111111, 299.9166666666667, 0.100021510002151, 165.675, 682.4, 162.33055555555555, 30.558333333333334,
      ],
    },
  ],
  meta: { preferredVisualisationType: 'nodeGraph' as const },
  name: 'edges',
};
