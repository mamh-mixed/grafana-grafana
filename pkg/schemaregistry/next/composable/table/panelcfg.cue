// Code generated - EDITING IS FUTILE. DO NOT EDIT.
//
// Generated by:
//     public/app/plugins/gen.go
// Using jennies:
//     PluginSchemaRegistryJenny
//
// Run 'make gen-cue' from repository root to regenerate.

import (
	"github.com/grafana/kindsys"
	ui "github.com/grafana/grafana/packages/grafana-schema/src/common"
)

kindsys.Composable & {
	maturity:        "experimental"
	name:            "Table" + "PanelCfg"
	schemaInterface: "PanelCfg"
	lineage: {
		seqs: [{
			schemas: [{
				PanelOptions: {
					// Represents the index of the selected frame
					frameIndex: number | *0
					// Controls whether the panel should show the header
					showHeader: bool | *true
					// Controls whether the header should show icons for the column types
					showTypeIcons?: bool | *false
					// Used to control row sorting
					sortBy?: [...ui.TableSortByFieldState]
					// Controls footer options
					footer?: ui.TableFooterOptions | *{
						// Controls whether the footer should be shown
						show: false
						// Controls whether the footer should show the total number of rows on Count calculation
						countRows: false
						// Represents the selected calculations
						reducer: []
					}
					// Controls the height of the rows
					cellHeight?: ui.TableCellHeight | *"sm"
				} @cuetsy(kind="interface")
			}]
		}]
		name: "Table" + "PanelCfg"
	}
}
