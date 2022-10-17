---
aliases:
  - /docs/grafana/latest/data-sources/elasticsearch/query-editor/
  - /docs/grafana/latest/data-sources/elasticsearch/template-variables/
description: Guide for using the Elasticsearch data source's query editor
keywords:
  - grafana
  - elasticsearch
  - lucene
  - metrics
  - logs
  - querying
title: Elasticsearch query editor
menuTitle: Query editor
weight: 300
---

# Elasticsearch query editor

![Elasticsearch Query Editor](/static/img/docs/elasticsearch/query-editor-7-4.png)

You can select multiple metrics and group by multiple terms or filters when using the Elasticsearch query editor.
Use the plus and minus icons to the right to add and remove metrics or group by clauses.
To expand the row to view and edit any available metric or group-by options, click the option text.

## Series naming and alias patterns

You can control the name for time series via the `Alias` input field.

| Pattern              | Replacement value                      |
| -------------------- | -------------------------------------- |
| `{{term fieldname}}` | Value of a term group-by               |
| `{{metric}}`         | Metric name, such as Average, Min, Max |
| `{{field}}`          | Metric field name                      |

## Pipeline metrics

Some metric aggregations, such as _Moving Average_ and _Derivative_, are called **Pipeline** aggregations.
Elasticsearch pipeline metrics must be based on another metric.

Use the eye icon next to the metric to prevent metrics from appearing in the graph.
This is useful for metrics you only have in the query for use in a pipeline metric.

![Pipeline aggregation editor](/static/img/docs/elasticsearch/pipeline-aggregation-editor-7-4.png)

## Template variables

Instead of hard-coding server, application, and sensor names in metric queries, you can use variables.
These variables are listed in dropdown select boxes at the top of the dashboard and help you change the display of data in your dashboard.
Grafana refers to such variables as template variables.

For an introduction to templating and template variables, refer to the [Templating]({{< relref "../../dashboards/variables/" >}}) documentation.

### Use query variables

The Elasticsearch data source supports two types of queries you can use in the **Query** field of Query variables.
Write the query using a custom JSON string, with the field mapped as a [keyword](https://www.elastic.co/guide/en/elasticsearch/reference/current/keyword.html#keyword) in the Elasticsearch index mapping.
If it is a [multi-field](https://www.elastic.co/guide/en/elasticsearch/reference/current/multi-fields.html) query with both a `text` and `keyword` type, use `"field":"fieldname.keyword"` (sometimes `fieldname.raw`) to specify the keyword field in your query.

| Query                                                               | Description                                                                                                                                                                   |
| ------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `{"find": "fields", "type": "keyword"}`                             | Returns a list of field names with the index type `keyword`.                                                                                                                  |
| `{"find": "terms", "field": "hostname.keyword", "size": 1000}`      | Returns a list of values for a keyword using term aggregation. Query will use current dashboard time range as time range query.                                               |
| `{"find": "terms", "field": "hostname", "query": '<lucene query>'}` | Returns a list of values for a keyword field using term aggregation and a specified lucene query filter. Query will use current dashboard time range as time range for query. |

Terms queries have a 500-result limit by default.
To set a custom limit, set the `size` property in your query.

You can use other variables inside the query.
This example defines a variable named `$host`:

```
{"find": "terms", "field": "hostname", "query": "source:$source"}
```

This uses another variable named `$source` inside the query definition.
Whenever you change the current value of the `$source` variable via the dropdown, Grafana triggers an update of the `$host` variable to contain only hostnames filtered by, in this case, the `source` document property.

These queries by default return results in term order (which can then be sorted alphabetically or numerically as for any variable).
To produce a list of terms sorted by doc count (a top-N values list), add an `orderBy` property of "doc_count".
This automatically selects a descending sort.

> **Note:** To use an ascending sort (`asc`) with doc_count (a bottom-N list), set `order: "asc"`. However, Elasticsearch [discourages this](https://www.elastic.co/guide/en/elasticsearch/reference/current/search-aggregations-bucket-terms-aggregation.html#search-aggregations-bucket-terms-aggregation-order) because it "increases the error on document counts".

To keep terms in the doc count order, set the variable's Sort dropdown to **Disabled**.
You can alternatively use other sorting criteria, such as **Alphabetical**, to re-sort them.

```
{"find": "terms", "field": "hostname", "orderBy": "doc_count"}
```

### Use variables in queries

There are two syntaxes:

- `$varname`, such as `hostname:$hostname`, which is easy to read and write but doesn't let you use a variable in the middle of a word.
- `[[varname]]`, such as `hostname:[[hostname]]`

When the _Multi-value_ or _Include all value_ options are enabled, Grafana converts the labels from plain text to a Lucene-compatible condition.

![Query with template variables](/static/img/docs/elasticsearch/elastic-templating-query-7-4.png)

In the above example, we have a Lucene query that filters documents based on the `hostname` property using a variable named `$hostname`.
It also uses a variable in the _Terms_ group by field input box.
This lets you use a variable to quickly change how data is grouped.

To view an example dashboard on Grafana Play, see the [Elasticsearch Templated Dashboard](https://play.grafana.org/d/CknOEXDMk/elasticsearch-templated?orgId=1d).
