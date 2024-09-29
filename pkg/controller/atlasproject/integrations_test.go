package atlasproject

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/atlas/mongodbatlas"
	"go.uber.org/zap"

	"github.com/mongodb/mongodb-atlas-kubernetes/v2/internal/mocks/atlas"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/internal/set"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/api/v1/project"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/controller/workflow"
)

const (
	testProjectID = "project-id"

	testNamespace = "some-namespace"
)

var errTest = fmt.Errorf("fake test error")

func TestToAlias(t *testing.T) {
	sample := []*mongodbatlas.ThirdPartyIntegration{{
		Type:   "DATADOG",
		APIKey: "some",
		Region: "EU",
	}}
	result := toAliasThirdPartyIntegration(sample)
	assert.Equal(t, sample[0].APIKey, result[0].APIKey)
	assert.Equal(t, sample[0].Type, result[0].Type)
	assert.Equal(t, sample[0].Region, result[0].Region)
}

func TestAreIntegrationsEqual(t *testing.T) {
	atlasDef := aliasThirdPartyIntegration{
		Type:   "DATADOG",
		APIKey: "****************************4e6f",
		Region: "EU",
	}
	specDef := aliasThirdPartyIntegration{
		Type:   "DATADOG",
		APIKey: "actual-valid-id*************4e6f",
		Region: "EU",
	}

	areEqual := integrationsApplied(&atlasDef, &specDef)
	assert.True(t, areEqual, "Identical objects should be equal")

	areEqual = areIntegrationsEqual(&atlasDef, &specDef)
	assert.False(t, areEqual, "Should fail if the last 4 characters of APIKey do not match")
}

func TestUpdateIntegrationsAtlas(t *testing.T) {
	calls := 0
	for _, tc := range []struct {
		title          string
		toUpdate       [][]set.Identifiable
		client         *mongodbatlas.Client
		expectedResult workflow.Result
		expectedCalls  int
	}{
		{
			title:          "nil list does nothing",
			expectedResult: workflow.OK(),
		},

		{
			title:          "empty list does nothing",
			toUpdate:       [][]set.Identifiable{},
			expectedResult: workflow.OK(),
		},

		{
			title: "different integrations get updated",
			toUpdate: set.Intersection(
				[]aliasThirdPartyIntegration{
					{
						Type:                     "MICROSOFT_TEAMS",
						Name:                     testNamespace,
						MicrosoftTeamsWebhookURL: "https://somehost/somepath/somesecret",
						Enabled:                  true,
					},
				},
				[]project.Integration{
					{
						Type:                     "MICROSOFT_TEAMS",
						MicrosoftTeamsWebhookURL: "https://somehost/some-otherpath/some-othersecret",
						Enabled:                  true,
					},
				}),
			client: &mongodbatlas.Client{
				Integrations: &atlas.IntegrationsMock{
					ReplaceFunc: func(ctx context.Context, projectID string, integrationType string, integration *mongodbatlas.ThirdPartyIntegration) (*mongodbatlas.ThirdPartyIntegrations, *mongodbatlas.Response, error) {
						calls += 1
						return nil, nil, nil
					},
				},
			},
			expectedResult: workflow.OK(),
			expectedCalls:  1,
		},

		{
			title: "matching integrations get updated anyway",
			toUpdate: set.Intersection(
				[]aliasThirdPartyIntegration{
					{
						Type:                     "MICROSOFT_TEAMS",
						Name:                     testNamespace,
						MicrosoftTeamsWebhookURL: "https://somehost/somepath/somesecret",
						Enabled:                  true,
					},
				},
				[]project.Integration{
					{
						Type:                     "MICROSOFT_TEAMS",
						MicrosoftTeamsWebhookURL: "https://somehost/somepath/somesecret",
						Enabled:                  true,
					},
				}),
			client: &mongodbatlas.Client{
				Integrations: &atlas.IntegrationsMock{
					ReplaceFunc: func(ctx context.Context, projectID string, integrationType string, integration *mongodbatlas.ThirdPartyIntegration) (*mongodbatlas.ThirdPartyIntegrations, *mongodbatlas.Response, error) {
						calls += 1
						return nil, nil, nil
					},
				},
			},
			expectedResult: workflow.OK(),
			expectedCalls:  1,
		},

		{
			title: "integrations fail to update and return error",
			toUpdate: set.Intersection(
				[]aliasThirdPartyIntegration{
					{
						Type:                     "MICROSOFT_TEAMS",
						Name:                     testNamespace,
						MicrosoftTeamsWebhookURL: "https://somehost/somepath/somesecret",
						Enabled:                  true,
					},
				},
				[]project.Integration{
					{
						Type:                     "MICROSOFT_TEAMS",
						MicrosoftTeamsWebhookURL: "https://somehost/somepath/somesecret",
						Enabled:                  true,
					},
				}),
			client: &mongodbatlas.Client{
				Integrations: &atlas.IntegrationsMock{
					ReplaceFunc: func(ctx context.Context, projectID string, integrationType string, integration *mongodbatlas.ThirdPartyIntegration) (*mongodbatlas.ThirdPartyIntegrations, *mongodbatlas.Response, error) {
						calls += 1
						return nil, nil, errTest
					},
				},
			},
			expectedResult: workflow.Terminate(workflow.ProjectIntegrationRequest, fmt.Errorf("Can not apply integration: %w", errTest)),
			expectedCalls:  1,
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			workflowCtx := &workflow.Context{
				Context: context.Background(),
				Log:     zap.S(),
				Client:  tc.client,
			}
			r := AtlasProjectReconciler{}
			calls = 0
			result := r.updateIntegrationsAtlas(workflowCtx, testProjectID, tc.toUpdate, testNamespace)
			assert.Equal(t, tc.expectedResult, result)
			assert.Equal(t, tc.expectedCalls, calls)
		})
	}
}

func TestCheckIntegrationsReady(t *testing.T) {
	for _, tc := range []struct {
		title     string
		toCheck   [][]set.Identifiable
		requested []project.Integration
		expected  bool
	}{
		{
			title:    "nil list does nothing",
			expected: true,
		},

		{
			title:     "empty list does nothing",
			toCheck:   [][]set.Identifiable{},
			requested: []project.Integration{},
			expected:  true,
		},

		{
			title:     "when requested list differs in length it bails early",
			toCheck:   [][]set.Identifiable{},
			requested: []project.Integration{{}},
			expected:  false,
		},

		{
			title: "matching integrations are considered applied",
			toCheck: set.Intersection(
				[]aliasThirdPartyIntegration{
					{
						Type:                     "MICROSOFT_TEAMS",
						Name:                     testNamespace,
						MicrosoftTeamsWebhookURL: "https://somehost/somepath/somesecret",
						Enabled:                  true,
					},
				},
				[]project.Integration{
					{
						Type:                     "MICROSOFT_TEAMS",
						MicrosoftTeamsWebhookURL: "https://somehost/somepath/somesecret",
						Enabled:                  true,
					},
				}),
			requested: []project.Integration{{}},
			expected:  true,
		},

		{
			title: "different integrations are considered also applied",
			toCheck: set.Intersection(
				[]aliasThirdPartyIntegration{
					{
						Type:                     "MICROSOFT_TEAMS",
						Name:                     testNamespace,
						MicrosoftTeamsWebhookURL: "https://somehost/somepath/somesecret",
						Enabled:                  true,
					},
				},
				[]project.Integration{
					{
						Type:                     "MICROSOFT_TEAMS",
						MicrosoftTeamsWebhookURL: "https://somehost/some-otherpath/some-othersecret",
						Enabled:                  true,
					},
				}),
			requested: []project.Integration{{}},
			expected:  true,
		},

		{
			title: "matching integrations including prometheus are considered applied",
			toCheck: set.Intersection(
				[]aliasThirdPartyIntegration{
					{
						Type:                     "MICROSOFT_TEAMS",
						Name:                     testNamespace,
						MicrosoftTeamsWebhookURL: "https://somehost/somepath/somesecret",
						Enabled:                  true,
					},
					{
						Type:             "PROMETHEUS",
						UserName:         "prometheus",
						ServiceDiscovery: "http",
						Enabled:          true,
					},
				},
				[]project.Integration{
					{
						Type:                     "MICROSOFT_TEAMS",
						MicrosoftTeamsWebhookURL: "https://somehost/somepath/somesecret",
						Enabled:                  true,
					},
					{
						Type:             "PROMETHEUS",
						UserName:         "prometheus",
						ServiceDiscovery: "http",
						Enabled:          true,
					},
				}),
			requested: []project.Integration{{}, {}},
			expected:  true,
		},

		{
			title: "matching integrations with a differing prometheus are considered different",
			toCheck: set.Intersection(
				[]aliasThirdPartyIntegration{
					{
						Type:                     "MICROSOFT_TEAMS",
						Name:                     testNamespace,
						MicrosoftTeamsWebhookURL: "https://somehost/somepath/somesecret",
						Enabled:                  true,
					},
					{
						Type:             "PROMETHEUS",
						UserName:         "prometheus",
						ServiceDiscovery: "http",
						Enabled:          true,
					},
				},
				[]project.Integration{
					{
						Type:                     "MICROSOFT_TEAMS",
						MicrosoftTeamsWebhookURL: "https://somehost/somepath/somesecret",
						Enabled:                  true,
					},
					{
						Type:             "PROMETHEUS",
						UserName:         "zeus",
						ServiceDiscovery: "file",
						Enabled:          true,
					},
				}),
			requested: []project.Integration{{}, {}},
			expected:  false,
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			workflowCtx := &workflow.Context{
				Context: context.Background(),
				Log:     zap.S(),
			}
			r := AtlasProjectReconciler{}
			result := r.checkIntegrationsReady(workflowCtx, testNamespace, tc.toCheck, tc.requested)
			assert.Equal(t, tc.expected, result)
		})
	}
}
