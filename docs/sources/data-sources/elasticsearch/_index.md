---
aliases:
  - /docs/grafana/latest/datasources/elasticsearch/
  - /docs/grafana/latest/features/datasources/elasticsearch/
  - /docs/grafana/latest/data-sources/elasticsearch/
description: Guide for using Elasticsearch in Grafana
keywords:
  - grafana
  - elasticsearch
  - guide
menuTitle: Elasticsearch
title: Elasticsearch data source
weight: 325
---

# Elasticsearch data source

Grafana ships with advanced support for Elasticsearch.
You can make many types of queries to visualize logs or metrics stored in Elasticsearch, and annotate graphs with log events stored in Elasticsearch.
This topic explains configuring and querying specific to the Elasticsearch data source.
For instructions on how to add a data source to Grafana, refer to the [administration documentation]({{< relref "/administration/data-source-management/" >}}).
Only users with the organization administrator role can add data sources.

## Supported Elasticsearch versions

This data source supports these versions of Elasticsearch:

- v7.10+
- v8.0+ (experimental)

## Configure the data source

**To configure the data source:**

1. Hover the cursor over the **Configuration** (gear) icon.
1. Select **Data Sources**.
1. Select the Elasticsearch data source.

Set the data source's basic configuration options carefully:

| Name      | Description                                                                |
| --------- | -------------------------------------------------------------------------- |
| `Name`    | The name you use to refer to the data source in panels and queries.        |
| `Default` | Data source is pre-selected for new panels.                                |
| `Url`     | The HTTP protocol, IP, and port of your Elasticsearch server.              |
| `Access`  | Don't modify Access. Use "Server (default)" or the data source won't work. |

You must also configure additional settings specific to the Elasticsearch data source.

### Index settings

![Elasticsearch data source details](/static/img/docs/elasticsearch/elasticsearch-ds-details-7-4.png)

Specify a default for the `time field` and the name of your Elasticsearch index.
You can use a time pattern or a wildcard for the index name.

### Elasticsearch version

Select the version of your Elasticsearch data source from the version selection dropdown.
Different query compositions and functionalities are available in the query editor for different versions.
Available Elasticsearch versions are `2.x`, `5.x`, `5.6+`, `6.0+`, `7.0+`, `7.7+` and `7.10+`.

Grafana assumes that you are running the lowest possible version for a specified range.
This ensures that new features or breaking changes in a future Elasticsearch release won't affect your configuration.

For example, if you run Elasticsearch `7.6.1` and select `7.0+`, and a new feature is made available for Elasticsearch `7.5.0` or newer releases, then a `7.5+` option will be available.
However, your configuration won't be affected until you explicitly select the new `7.5+` option in your settings.

### Min time interval

Set a lower limit for the auto group by time interval.

This value **must** be formatted as a number followed by a valid time identifier.
The following time identifiers are supported:

| Identifier | Description |
| ---------- | ----------- |
| `y`        | year        |
| `M`        | month       |
| `w`        | week        |
| `d`        | day         |
| `h`        | hour        |
| `m`        | minute      |
| `s`        | second      |
| `ms`       | millisecond |

We recommend setting the value to match your Elasticsearch write frequency.
For example, set this to `1m` if your data is written every minute.

This option can be overridden or configured in a dashboard panel under its data source options.

### X-Pack enabled

Toggle this to enable `X-Pack`-specific features and options, which provide the [query editor](#metrics-query-editor) with additional aggregations such as `Rate` and `Top Metrics`.

#### Include frozen indices

When thge "X-Pack enabled" setting is active and the configured Elasticsearch version is higher than `6.6.0`, you can configure Grafana to not ignore [frozen indices](https://www.elastic.co/guide/en/elasticsearch/reference/7.13/frozen-indices.html) when performing search requests.

> **Note:** Frozen indices are [deprecated in Elasticsearch](https://www.elastic.co/guide/en/elasticsearch/reference/7.17/frozen-indices.html) since v7.14.

### Logs

You can optionally configure the two Logs parameters **Message field name** and **Level field name** to determine which fields the data source uses for log messages and log levels when visualizing logs in [Explore]({{< relref "../explore/" >}}).

For example, if you're using a default setup of Filebeat for shipping logs to Elasticsearch, set:

- **Message field name:** `message`
- **Level field name:** `fields.level`

### Data links

Data links create a link from a specified field that can be accessed in Explore's logs view.

Each data link configuration consists of:

| Parameter     | Description                                                                                                                                                                                                                         |
| ------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Field         | Sets the name of the field used by the data link.                                                                                                                                                                                   |
| URL/query     | Sets the full link URL if the link is external. If the link is internal, this input serves as a query for the target data source.<br/>In both cases, you can interpolate the value from the field with the `${__value.raw }` macro. |
| URL Label     | (Optional) Set a custom display label for the link. The link label defaults to the full external URL or name of the linked internal data source and is overridden by this setting.                                                  |
| Internal link | Sets whether the link is internal or external. In case of an internal link, you can select the target data source with a data source selector. This supports only tracing data sources.                                             |

### Provision the data source

You can configure the Elasticsearch data source by customizing configuration files in Grafana's provisioning system.
For more information about provisioning, and for available configuration options, refer to [Provisioning Grafana]({{< relref "../../administration/provisioning/#data-sources" >}}).

#### Provisioning examples

**Basic provisioning:**

```yaml
apiVersion: 1

datasources:
  - name: Elastic
    type: elasticsearch
    access: proxy
    database: '[metrics-]YYYY.MM.DD'
    url: http://localhost:9200
    jsonData:
      interval: Daily
      timeField: '@timestamp'
```

**Provision for logs:**

```yaml
apiVersion: 1

datasources:
  - name: elasticsearch-v7-filebeat
    type: elasticsearch
    access: proxy
    database: '[filebeat-]YYYY.MM.DD'
    url: http://localhost:9200
    jsonData:
      interval: Daily
      timeField: '@timestamp'
      esVersion: '7.0.0'
      logMessageField: message
      logLevelField: fields.level
      dataLinks:
        - datasourceUid: my_jaeger_uid # Target UID needs to be known
          field: traceID
          url: '$${__value.raw}' # Careful about the double "$$" because of env var expansion
```

## Query editor

You can select multiple metrics and group by multiple terms or filters when using the Elasticsearch query editor.

For details, see the [query editor documentation]({{< relref "./query-editor/" >}}).

## Annotations

You can overlay rich event information on top of graphs by using [Annotations]({{< relref "../dashboards/build-dashboards/annotate-visualizations" >}}).
To add annotation queries, use the Dashboard menu's Annotations view.
Grafana can query any Elasticsearch index for annotation events.

| Name       | Description                                                                                                                                          |
| ---------- | ---------------------------------------------------------------------------------------------------------------------------------------------------- |
| `Query`    | Specify a Lucene query, or leave the query blank.                                                                                                    |
| `Time`     | Specify the time field's name, which must be a date field.                                                                                           |
| `Time End` | (Optional) Specify the name of the time end field, which must be a date field. If set, annotations are marked as a region between time and time-end. |
| `Text`     | Specify the event description field.                                                                                                                 |
| `Tags`     | (Optional) Specify one or more field names, as an array or comma-separated string, to use for event tags.                                            |

## Query logs

Querying and displaying log data from Elasticsearch is available in [Explore]({{< relref "../explore/" >}}), and in the [logs panel]({{< relref "../panels-visualizations/visualizations/logs/" >}}) in dashboards.
Select the Elasticsearch data source, and then optionally enter a Lucene query to display your logs.

When switching from a Prometheus or Loki data source in Explore, your query is translated to an Elasticsearch log query with a correct Lucene filter.

### Log Queries

Once the result is returned, the log panel shows a list of log rows and a bar chart where the x-axis shows the time and the y-axis shows the frequency/count.

Note that the fields used for log message and level is based on an [optional data source configuration](#logs).

### Filter Log Messages

Optionally enter a lucene query into the query field to filter the log messages. For example, using a default Filebeat setup you should be able to use `fields.level:error` to only show error log messages.

## Use Amazon Elasticsearch Service

AWS users using Amazon's Elasticsearch Service can use Grafana's Elasticsearch data source to visualize Elasticsearch data.
If you are using an AWS Identity and Access Management (IAM) policy to control access to your Amazon Elasticsearch Service domain, then you must use AWS Signature Version 4 (AWS SigV4) to sign all requests to that domain.
For more details on AWS SigV4, refer to the [AWS documentation](https://docs.aws.amazon.com/general/latest/gr/signature-version-4.html).

### AWS Signature Version 4 authentication

> **Note:** Only available in Grafana v7.3 and higher.

To sign requests to your Amazon Elasticsearch Service domain, you can enable SigV4 in Grafana's [configuration]({{< relref "/setup-grafana/configure-grafana/#sigv4_auth_enabled" >}}).

Once AWS SigV4 is enabled, you can configure it on the Elasticsearch data source configuration page.
For more information about authentication options, refer to [CloudWatch authentication]({{< relref "../aws-cloudwatch/aws-authentication/" >}}).

{{< figure src="/static/img/docs/v73/elasticsearch-sigv4-config-editor.png" max-width="500px" class="docs-image--no-shadow" caption="SigV4 configuration for AWS Elasticsearch Service" >}}
