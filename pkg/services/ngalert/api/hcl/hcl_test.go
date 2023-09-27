package hcl

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/util"
)

func TestEncode(t *testing.T) {
	type data struct {
		Name      string   `hcl:"name"`
		Number    float64  `hcl:"number"`
		NumberRef *float64 `hcl:"numberRef"`
		Bool      bool     `hcl:"bul"`
		BoolRef   *bool    `hcl:"bulRef"`
		Ignored   string
		Blocks    []data             `hcl:"blocks,block"`
		SubData   *data              `hcl:"sub,block"`
		Map       *map[string]string `hcl:"map_data"`
	}

	encoded, err := Encode(Resource{
		Type: "grafana_test",
		Name: "test-01",
		Body: &data{
			Name:      "test",
			Number:    123,
			NumberRef: util.Pointer(1333.0),
			Bool:      false,
			BoolRef:   util.Pointer(true),
			Ignored:   "Ignore me",
			Blocks: []data{
				{
					Name:   "el-0",
					Number: 1,
				},
				{
					Name:   "el-1",
					Number: 2,
					Bool:   true,
				},
			},
			SubData: &data{
				Name:   "sub-data",
				Number: 123123,
			},
			Map: util.Pointer(map[string]string{
				"test":  "data",
				"test2": "data1",
			}),
		},
	})
	require.NoError(t, err)
	t.Log(string(encoded))
	require.Equal(t, `resource "grafana_test" "test-01" {
  name      = "test"
  number    = 123
  numberRef = 1333
  bul       = false
  bulRef    = true

  blocks {
    name   = "el-0"
    number = 1
    bul    = false
  }
  blocks {
    name   = "el-1"
    number = 2
    bul    = true
  }

  sub {
    name   = "sub-data"
    number = 123123
    bul    = false
  }

  map_data = {
    test  = "data"
    test2 = "data1"
  }
}
`, string(encoded))
}
