---
aliases:
  - /docs/grafana/latest/data-sources/elasticsearch/query-editor/
  - /docs/grafana/latest/data-sources/elasticsearch/template-variables/
description: Guide for using the Elasticsearch data source's query editor
keywords:
  - grafana
  - prometheus
  - logs
  - queries
menuTitle: Query editor
title: Prometheus query editor
weight: 300
---

# Prometheus query editor

The Prometheus query editor has two distinct modes that you can toggle:

- [Code mode](#code-mode), which provides a feature-rich editor for writing queries
- [Builder mode](#builder-mode), which provides a visual query designer

![Editor toolbar](/static/img/docs/prometheus/header-9-1.png 'Editor toolbar')

Select the `Builder | Code` tabs to switch between the editor modes.

At the top of the editor, select `Run queries` to run a query.

> **Note:** To run Prometheus queries in [Explore]({{< relref "../../explore/" >}}), select `Run query`.

Each mode is synchronized with the other modes, so you can switch between them without losing your work, although there are some limitations.
Some complex queries are not yet supported in Builder mode.
When you switch from Code mode to Builder mode with such a query, the editor displays a popup that explains how you might lose parts of the query if you continue.
You can then decide whether you still want to switch to Builder mode.

You can also augment queries by using [template variables]({{< relref "./template-variables/" >}}).

For options and functions common to all query editors, refer to [Query and transform data]({{< relref "../../panels-visualizations/query-transform-data" >}}).

## Configure common options

You can configure Prometheus-specific options in the query editor by setting several options regardless of its mode.

![Options](/static/img/docs/prometheus/options-9-1.png 'Options')

### Legend

The **Legend** setting defines the time series's name. You can use a predefined or custom format.

| Option      | Description                                                                                                                                                                                                           |
| ----------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Auto**    | Shows the value of a single label for each series with only one label, or displays all labels if a series has multiple labels.                                                                                        |
| **Verbose** | Displays all labels.                                                                                                                                                                                                  |
| **Custom**  | Uses templating to select which labels will be included.<br/>For example, `{{hostname}}` is replaced by the label value for the label `hostname`.<br/>Clear the input and click outside of it to select another mode. |

### Min step

The **Min step** setting defines the lower bounds on the interval between data points.
For example, set this to `1h` to hint that measurements are taken hourly.
This setting supports the `$__interval` and `$__rate_interval` macros.

### Format

You can switch between **Table**, **Time series**, and **Heatmap** options by configuring the query's **Format**.

| Option          | Description                                                                                                                                                                                                                |
| --------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Table**       | This works only in a [Table panel]{{< relref "/panels-visualizations/visualizations/table/" >}}.                                                                                                                           |
| **Time series** | Uses the default time series format.                                                                                                                                                                                       |
| **Heatmap**     | Displays metrics of the Histogram type on a [Heatmap panel]{{< relref "/panels-visualizations/visualizations/heatmap/" >}} by converting cumulative histograms to regular ones and sorting the series by the bucket bound. |

### Type

The **Type** setting selects the query type.

- A **Range** query returns a Range vector, comprised of a set of time series containing a range of data points over time for each time series.
- An **Instant** query returns only the latest value that Prometheus has scraped for the requested time series. Instant queries can return results much faster than normal range queries and are well suited to look up label sets.

  Instant query results are comprised of only one data point per series and can be shown in the time series panel by adding a field override, adding a property to the override named `Transform`, and selecting `Constant` from the **Transform** dropdown.

  For more information, refer to the [Time Series Transform option documentation]({{< relref "../panels-visualizations/visualizations/time-series/#transform" >}}).

- An **Exemplars** query runs with the regular query and shows exemplars in the graph.

> **Note:** Grafana modifies the request dates for queries to align them with the dynamically calculated step.
> This ensures a consistent display of metrics data, but it can result in a small gap of data at the right edge of a graph.

## Code mode

![Code mode](/static/img/docs/prometheus/code-mode-9-1.png 'Code mode')

In **Code mode**, you can write complex queries using a text editor with autocompletion features and syntax highlighting.
It also contains a [Metrics browser](#metrics-browser) to further help you write queries.

For more information about Prometheus query language, refer to the [Prometheus documentation](http://prometheus.io/docs/querying/basics/).

### Use autocompletion

![Autocomplete](/static/img/docs/prometheus/autocomplete-9-1.png 'Autocomplete')

Code mode's autocompletion feature works automatically while typing.
To manually trigger autocompletion, use the keyboard shortcut <key>Ctrl</key>/<key>Cmd</key> + <key>Space</key>.

The query editor can autocomplete static functions, aggregations, and keywords, and also dynamic items like metrics and labels.
The autocompletion dropdown includes documentation for the suggested items where available.

To run a query in [Explore]({{< relref "../../explore/" >}}), use the keyboard shortcut <key>Shift</key> + <key>Enter</key>.

### Metrics browser

The metrics browser locates metrics and selects relevant labels to help you build basic queries.
When you open the browser, it displays all available metrics and labels.
If supported by your Prometheus instance, each metric also displays its HELP and TYPE as a tooltip.

![Metrics browser](/static/img/docs/prometheus/metric-browser-9-1.png 'Metrics browser')

When you select a metric, the browser narrows down the available labels to show only the ones applicable to the metric.
You can then select one or more labels for which the available label values are shown in lists in the bottom section.
Select one or more values for each label to tighten your query scope.

> **Note:** If you do not remember a metric name to start with, you can also select a few labels to narrow down the list, then find relevant label values.

All lists in the metrics browser have a search field above them to quickly filter for metrics or labels that match a certain string.
The values section has only one search field, and its filtering applies to all labels to help you find values across labels once selected.

For example, among your labels `app`, `job`, `job_name` only one might with the value you are looking for.

Once you are satisfied with your query, click "Use query" to run the query. The button "Use as rate query" adds a `rate(...)[$__interval]` around your query to help write queries for counter metrics.
The "Validate selector" button will check with Prometheus how many time series are available for that selector.

## Builder mode

This video demonstrates how to use the visual Prometheus query builder available since Grafana v9.0:

{{< vimeo 720004179 >}}

</br>

### Toolbar

In addition to the `Run query` button and mode switcher, Builder mode includes additional elements:

| Name               | Description                                                                               |
| ------------------ | ----------------------------------------------------------------------------------------- |
| **Query patterns** | A list of operation patterns that help you quickly add multiple operations to your query. |
| **Explain**        | Displays a step-by-step explanation of all query parts and its operations.                |
| **Raw query**      | Displays the raw query generated by the Builder that will be sent to Prometheus instance. |

### Metric and labels

![Metric and labels](/static/img/docs/prometheus/metric-select-8-5.png 'Metric and labels')

Select a specific metric name from the dropdown list.
The data source requests the list of available metrics from the Prometheus server based on the selected time rage.
You can also enter text into the selector when the dropdown is open to search and filter the list.

Select desired labels and their values from the dropdown list.
When a metric is selected, the data source requests available labels and their values from the server.
Use the `+` button to add a label, and the `x` button to remove a label.

### Operations

![Operations](/static/img/docs/prometheus/operations-9-1.gif 'Operations')

Select the `+ Operations` button to add operations to your query.
The query editor groups operations into related sections, and you can type while the operations dropdown is open to search and filter the list.

The query editor displays a query's operations as boxes in the operations section.
Each operation's header displays its name, and additional action buttons appear when you hover your cursor over the header:

| Button | Action                                                            |
| ------ | ----------------------------------------------------------------- |
| `v`    | Replaces the operation with different operation of the same type. |
| `info` | Opens the operation's description tooltip.                        |
| `x`    | Removes the operation.                                            |

Some operations have additional parameters under the operation header.
For details about each operation, use the `info` button to view the operation's description, or refer to the Prometheus documentation on [query functions](https://prometheus.io/docs/prometheus/latest/querying/functions/).

Some operations make sense only when used in a specific order.
If adding an operation would result in nonsensical query, the query editor adds the operation to the correct place.
To re-order operations manually, drag the operation box by its name and drop it into the desired place.

#### Hints

![Hint](/static/img/docs/prometheus/hint-8-5.gif 'Hint')

The query editor can sometimes detect which operations are most appropriate for a selected metric. In such cases, it displays a hint next to the `+ Operations` button. Click the hint to add the operations to your query.

## Explain

![Explain mode](/static/img/docs/prometheus/explain-9-1.png 'Explain mode')

Explain mode helps you understand a query by displaying a step-by-step explanation of all query components and operations.

### Raw query

![Raw query](/static/img/docs/prometheus/raw-query-9-1.gif 'Raw query')

The query editor displays the raw query only if the **Raw query** switch from the query editor toolbar is enabled.
If visible, it displays the raw query that the query editor has created.

### Additional options

In addition to these Builder mode-specific options, the query editor also displays the options it shares in common with Code mode.
For details, see the [Code mode options]({{< relref "#options" >}}).

# Annotations

[Annotations]({{< relref "../dashboards/build-dashboards/annotate-visualizations" >}}) allow you to overlay rich event information on top of graphs. You add annotation
queries via the Dashboard menu / Annotations view.

Prometheus supports two ways to query annotations.

- A regular metric query
- A Prometheus query for pending and firing alerts (for details see [Inspecting alerts during runtime](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/#inspecting-alerts-during-runtime))

The step option is useful to limit the number of events returned from your query.
