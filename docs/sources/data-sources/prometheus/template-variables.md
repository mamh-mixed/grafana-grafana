# Templating

Instead of hard-coding things like server, application and sensor name in your metric queries, you can use variables in their place.
Variables are shown as dropdown select boxes at the top of the dashboard. These dropdowns make it easy to change the data
being displayed in your dashboard.

Check out the [Templating]({{< relref "../dashboards/variables" >}}) documentation for an introduction to the templating feature and the different
types of template variables.

## Query variable

Variable of the type _Query_ allows you to query Prometheus for a list of metrics, labels or label values. The Prometheus data source plugin
provides the following functions you can use in the `Query` input field.

| Name                          | Description                                                             | Used API endpoints                |
| ----------------------------- | ----------------------------------------------------------------------- | --------------------------------- |
| `label_names()`               | Returns a list of label names.                                          | /api/v1/labels                    |
| `label_values(label)`         | Returns a list of label values for the `label` in every metric.         | /api/v1/label/`label`/values      |
| `label_values(metric, label)` | Returns a list of label values for the `label` in the specified metric. | /api/v1/series                    |
| `metrics(metric)`             | Returns a list of metrics matching the specified `metric` regex.        | /api/v1/label/\_\_name\_\_/values |
| `query_result(query)`         | Returns a list of Prometheus query result for the `query`.              | /api/v1/query                     |

For details of what _metric names_, _label names_ and _label values_ are please refer to the [Prometheus documentation](http://prometheus.io/docs/concepts/data_model/#metric-names-and-labels).

### Using interval and range variables

> Support for `$__range`, `$__range_s` and `$__range_ms` only available from Grafana v5.3

You can use some global built-in variables in query variables, for example, `$__interval`, `$__interval_ms`, `$__range`, `$__range_s` and `$__range_ms`. See [Global built-in variables]({{< relref "../dashboards/variables/add-template-variables/#global-variables" >}}) for more information. They are convenient to use in conjunction with the `query_result` function when you need to filter variable queries since the `label_values` function doesn't support queries.

Make sure to set the variable's `refresh` trigger to be `On Time Range Change` to get the correct instances when changing the time range on the dashboard.

**Example usage:**

Populate a variable with the busiest 5 request instances based on average QPS over the time range shown in the dashboard:

```
Query: query_result(topk(5, sum(rate(http_requests_total[$__range])) by (instance)))
Regex: /"([^"]+)"/
```

Populate a variable with the instances having a certain state over the time range shown in the dashboard, using `$__range_s`:

```
Query: query_result(max_over_time(<metric>[${__range_s}s]) != <state>)
Regex:
```

## Using `$__rate_interval`

> **Note:** Available in Grafana 7.2 and above

`$__rate_interval` is the recommended interval to use in the `rate` and `increase` functions. It will "just work" in most cases, avoiding most of the pitfalls that can occur when using a fixed interval or `$__interval`.

```
OK:       rate(http_requests_total[5m])
Better:   rate(http_requests_total[$__rate_interval])
```

Details: `$__rate_interval` is defined as max(`$__interval` + _Scrape interval_, 4 \* _Scrape interval_), where _Scrape interval_ is the Min step setting (AKA query*interval, a setting per PromQL query) if any is set. Otherwise, the Scrape interval setting in the Prometheus data source is used. (The Min interval setting in the panel is modified by the resolution setting and therefore doesn't have any effect on \_Scrape interval*.) [This article](https://grafana.com/blog/2020/09/28/new-in-grafana-7.2-__rate_interval-for-prometheus-rate-queries-that-just-work/) contains additional details.

## Using variables in queries

There are two syntaxes:

- `$<varname>` Example: rate(http_requests_total{job=~"\$job"}[5m])
- `[[varname]]` Example: rate(http_requests_total{job=~"[[job]]"}[5m])

Why two ways? The first syntax is easier to read and write but does not allow you to use a variable in the middle of a word. When the _Multi-value_ or _Include all value_
options are enabled, Grafana converts the labels from plain text to a regex compatible string. Which means you have to use `=~` instead of `=`.

## Ad hoc filters variable

Prometheus supports the special [ad hoc filters]({{< relref "../dashboards/variables/add-template-variables/#add-ad-hoc-filters" >}}) variable type. It allows you to specify any number of label/value filters on the fly. These filters are automatically
applied to all your Prometheus queries.
