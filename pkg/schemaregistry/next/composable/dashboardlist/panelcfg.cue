// Code generated - EDITING IS FUTILE. DO NOT EDIT.
//
// Generated by:
//     public/app/plugins/gen.go
// Using jennies:
//     PluginSchemaRegistryJenny
//
// Run 'make gen-cue' from repository root to regenerate.

import "github.com/grafana/kindsys"

kindsys.Composable & {
	maturity:        "experimental"
	name:            "DashboardList" + "PanelCfg"
	schemaInterface: "PanelCfg"
	lineage: {
		seqs: [{
			schemas: [{
				PanelOptions: {
					keepTime:           bool | *false
					includeVars:        bool | *false
					showStarred:        bool | *true
					showRecentlyViewed: bool | *false
					showSearch:         bool | *false
					showHeadings:       bool | *true
					maxItems:           int | *10
					query:              string | *""
					folderId?:          int
					tags:               [...string] | *[]
				} @cuetsy(kind="interface")
			}]
		}]
		name: "DashboardList" + "PanelCfg"
	}
}
