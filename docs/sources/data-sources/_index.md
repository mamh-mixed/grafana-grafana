---
aliases:
  - /docs/grafana/latest/datasources/
  - /docs/grafana/latest/datasources/overview/
  - /docs/grafana/latest/data-sources/
title: Data sources
weight: 60
---

# Data sources

Grafana supports many different backends for your time series data, and lets you create your own interfaces to backends.
Grafana refers to these backends as **data sources**.

You can use this flexibility to build [panels]({{< relref "../panels-visualizations/" >}}) that collect and visualize data of many types, and from many sources, in a single [dashboard]({{< relref "../dashboards/" >}}).
Each panel uses one specific data source, which can belong to a particular [organization]({{< relref "../administration/organization-management/" >}}).

## Manage data sources

Only users with the [organization administrator role]({{< relref "../administration/roles-and-permissions/#organization-roles" >}}) can add or remove data sources.
To access data source management tools in Grafana as an administrator, navigate to **Configuration > Data Sources** in the Grafana sidebar.

For details on data source management, including instructions on how to add data sources and configure user permissions for queries, refer to [the administration documentation]({{< relref "../administration/data-source-management/" >}}).

## Query editors

Each data source has its own query editor customized for its unique query language, features, and capabilities.
Query editors help you request data from a source, and many provide visual tools that can simplify the query-building process.

For example, this video demonstrates the visual Prometheus query builder:

{{< vimeo 720004179 >}}

For more details about query editors, refer to the data source's documentation.

## Data source plugins

You can install additional data sources as plugins.
To view available data source plugins, see the [Grafana Plugins catalog](/plugins/).
To build your own, see the ["Build a data source plugin"](/tutorials/build-a-data-source-plugin/) tutorial and our documentation about [building a plugin](/developers/plugins/).

The Grafana community contributes to many of Grafana's data source plugins.
Grafana Labs itself also manages or supports several data sources, as do third-party partners.

To view available data source plugins, go to the [plugin catalog](/grafana/plugins/?type=datasource) and select the "Data sources" filter.
You can further filter the list by plugins created by the community, Grafana Labs, and partners.
If you use Grafana Enterprise, you can also filter by Enterprise-supported plugins.

For more documentation on a specific data source plugin, refer to its plugin catalog page.

These data sources have additional documentation in the Grafana docs:

- [Alertmanager]({{< relref "./alertmanager/" >}})
- [AWS CloudWatch]({{< relref "./aws-cloudwatch/" >}})
- [Azure Monitor]({{< relref "./azuremonitor/" >}})
- [Elasticsearch]({{< relref "./elasticsearch/" >}})
- [Google Cloud Monitoring]({{< relref "./google-cloud-monitoring/" >}})
- [Graphite]({{< relref "./graphite/" >}})
- [InfluxDB]({{< relref "./influxdb/" >}})
- [Jaeger]({{< relref "./jaeger/" >}})
- [Loki]({{< relref "./loki/" >}})
- [Microsoft SQL Server (MSSQL)]({{< relref "./mssql/" >}})
- [MySQL]({{< relref "./mysql/" >}})
- [OpenTSDB]({{< relref "./opentsdb/" >}})
- [PostgreSQL]({{< relref "./postgres/" >}})
- [Prometheus]({{< relref "./prometheus/" >}})
- [Tempo]({{< relref "./tempo/" >}})
- [Testdata]({{< relref "./testdata/" >}})
- [Zipkin]({{< relref "./zipkin/" >}})

## Special data sources

In addition to the data sources in your Grafana instance, there are three special data sources available:

- **Grafana:** A built-in data source that generates random walk data and can poll the [Testdata]({{< relref "./testdata/" >}}) data source.
  This helps you test visualizations and run experiments.
- **Mixed:** An abstraction that lets you query multiple data sources in the same panel.
  When you select Mixed, you can then select a different data source for each new query that you add.
  - The first query uses the data source that was selected before you selected **Mixed**.
  - You can't change an existing query to use the Mixed data source.
  - Grafana Play example: [Mixed data sources](https://play.grafana.org/d/000000100/mixed-datasources?orgId=1)
- **Dashboard:** A data source that uses the result set from another panel in the same dashboard.
