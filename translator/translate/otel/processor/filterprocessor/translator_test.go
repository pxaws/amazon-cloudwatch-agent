// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package filterprocessor

import (
	"path/filepath"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/filterprocessor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"

	"github.com/aws/amazon-cloudwatch-agent/internal/util/testutil"
	"github.com/aws/amazon-cloudwatch-agent/translator/translate/otel/common"
)

func TestTranslator(t *testing.T) {
	factory := filterprocessor.NewFactory()
	testCases := map[string]struct {
		input   map[string]any
		index   int
		wantID  string
		want    *confmap.Conf
		wantErr error
	}{
		"ConfigWithNoJmxSet": {
			input: map[string]any{
				"metrics": map[string]any{
					"metrics_collected": map[string]any{
						"cpu": map[string]any{},
					},
				},
			},
			index:  -1,
			wantID: "filter/jmx",
			wantErr: &common.MissingKeyError{
				ID:      component.NewIDWithName(factory.Type(), "jmx"),
				JsonKey: common.JmxConfigKey,
			},
		},
		"ConfigWithJmxTargetWithMetricName": {
			input: map[string]any{
				"metrics": map[string]any{
					"metrics_collected": map[string]any{
						"jmx": []any{
							map[string]any{
								"jvm": map[string]any{
									"measurement": []any{
										"jvm.memory.heap.init",
									},
								},
							},
						},
					},
				},
			},
			index:  0,
			wantID: "filter/jmx/0",
			want: confmap.NewFromStringMap(map[string]any{
				"metrics": map[string]any{
					"include": map[string]any{
						"match_type":   "strict",
						"metric_names": []any{"jvm.memory.heap.init"},
					},
				},
			}),
		},
		"ConfigWithMultiple": {
			input: map[string]any{
				"metrics": map[string]any{
					"metrics_collected": map[string]any{
						"jmx": map[string]any{
							"jvm": map[string]any{
								"measurement": []any{
									"jvm.memory.heap.init",
									map[string]any{
										"name":   "jvm.classes.loaded",
										"rename": "JVM.CLASSES.LOADED",
										"unit":   "Count",
									},
									"jvm.threads.count",
								},
							},
							"tomcat": map[string]any{
								"measurement": []any{
									"tomcat.sessions",
									"tomcat.errors",
								},
							},
						},
					},
				},
			},
			index:  -1,
			wantID: "filter/jmx",
			want: confmap.NewFromStringMap(map[string]any{
				"metrics": map[string]any{
					"include": map[string]any{
						"match_type": "strict",
						"metric_names": []any{
							"jvm.memory.heap.init",
							"jvm.classes.loaded",
							"jvm.threads.count",
							"tomcat.sessions",
							"tomcat.errors",
						},
					},
				},
			}),
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			tt := NewTranslator(common.WithName("jmx"), common.WithIndex(testCase.index))
			require.EqualValues(t, testCase.wantID, tt.ID().String())
			conf := confmap.NewFromStringMap(testCase.input)
			got, err := tt.Translate(conf)
			require.Equal(t, testCase.wantErr, err)
			if err == nil {
				require.NotNil(t, got)
				gotCfg, ok := got.(*filterprocessor.Config)
				require.True(t, ok)
				wantCfg := factory.CreateDefaultConfig()
				require.NoError(t, testCase.want.Unmarshal(wantCfg))
				require.Equal(t, wantCfg, gotCfg)
			}
		})
	}
}

func TestContainerInsightsJmx(t *testing.T) {
	transl := NewTranslator(common.WithName(common.PipelineNameContainerInsightsJmx)).(*translator)
	expectedCfg := transl.factory.CreateDefaultConfig().(*filterprocessor.Config)
	c := testutil.GetConf(t, "filter_jmx_config.yaml")
	require.NoError(t, c.Unmarshal(&expectedCfg))

	conf := confmap.NewFromStringMap(testutil.GetJson(t, filepath.Join("testdata", "config.json")))
	translatedCfg, err := transl.Translate(conf)
	assert.NoError(t, err)
	actualCfg, ok := translatedCfg.(*filterprocessor.Config)
	assert.True(t, ok)
	assert.Equal(t, len(expectedCfg.Metrics.Include.MetricNames), len(actualCfg.Metrics.Include.MetricNames))
}

func TestContainerInsights(t *testing.T) {
	transl := NewTranslator(common.WithName(common.PipelineNameContainerInsights)).(*translator)
	expectedCfg := transl.factory.CreateDefaultConfig().(*filterprocessor.Config)
	c := testutil.GetConf(t, "filter_containerinsights_config.yaml")
	require.NoError(t, c.Unmarshal(&expectedCfg))

	conf := confmap.NewFromStringMap(testutil.GetJson(t, filepath.Join("testdata", "config.json")))
	translatedCfg, err := transl.Translate(conf)
	assert.NoError(t, err)
	actualCfg, ok := translatedCfg.(*filterprocessor.Config)
	assert.True(t, ok)
	assert.Equal(t, len(expectedCfg.Metrics.Exclude.MetricNames), len(actualCfg.Metrics.Exclude.MetricNames))
}
