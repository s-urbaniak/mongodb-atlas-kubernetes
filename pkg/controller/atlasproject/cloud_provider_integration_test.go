package atlasproject

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.mongodb.org/atlas/mongodbatlas"
	"go.uber.org/zap/zaptest"

	"github.com/mongodb/mongodb-atlas-kubernetes/v2/internal/mocks/atlas"
	mdbv1 "github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/api/v1"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/api/v1/status"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/controller/workflow"
)

func TestSyncCloudProviderIntegration(t *testing.T) {
	t.Run("should fail when atlas is unavailable", func(t *testing.T) {
		atlasClient := mongodbatlas.Client{
			CloudProviderAccess: &atlas.CloudProviderAccessClientMock{
				ListRolesFunc: func(projectID string) (*mongodbatlas.CloudProviderAccessRoles, *mongodbatlas.Response, error) {
					return nil, &mongodbatlas.Response{}, errors.New("service unavailable")
				},
			},
		}
		workflowCtx := &workflow.Context{
			Client:  &atlasClient,
			Context: context.Background(),
		}
		result, err := syncCloudProviderIntegration(workflowCtx, "projectID", []mdbv1.CloudProviderIntegration{})
		assert.EqualError(t, err, "unable to fetch cloud provider access from Atlas: service unavailable")
		assert.False(t, result)
	})

	t.Run("should synchronize all operations without reach ready status", func(t *testing.T) {
		cpas := []mdbv1.CloudProviderIntegration{
			{
				ProviderName:      "AWS",
				IamAssumedRoleArn: "aws:arn/my_role-1",
			},
			{
				ProviderName:      "AWS",
				IamAssumedRoleArn: "aws:arn/my_role-2",
			},
			{
				ProviderName: "AWS",
			},
		}
		atlasCPAs := []mongodbatlas.CloudProviderAccessRole{
			{
				AtlasAWSAccountARN:         "atlas-account-arn-1",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
				AuthorizedDate:             "authorized-date-1",
				CreatedDate:                "created-date-1",
				IAMAssumedRoleARN:          "aws:arn/my_role-1",
				ProviderName:               "AWS",
				RoleID:                     "role-1",
			},
			{
				AtlasAWSAccountARN:         "atlas-account-arn-2",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-2",
				CreatedDate:                "created-date-2",
				IAMAssumedRoleARN:          "aws:arn/my_role-2",
				ProviderName:               "AWS",
				RoleID:                     "role-2",
			},
			{
				AtlasAWSAccountARN:         "atlas-account-arn-4",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-4",
				IAMAssumedRoleARN:          "aws:arn/my-role-4",
				CreatedDate:                "created-date-4",
				ProviderName:               "AWS",
				RoleID:                     "role-4",
			},
		}
		atlasClient := mongodbatlas.Client{
			CloudProviderAccess: &atlas.CloudProviderAccessClientMock{
				ListRolesFunc: func(projectID string) (*mongodbatlas.CloudProviderAccessRoles, *mongodbatlas.Response, error) {
					return &mongodbatlas.CloudProviderAccessRoles{
						AWSIAMRoles: atlasCPAs,
					}, &mongodbatlas.Response{}, nil
				},
				CreateRoleFunc: func(projectID string, cpa *mongodbatlas.CloudProviderAccessRoleRequest) (*mongodbatlas.CloudProviderAccessRole, *mongodbatlas.Response, error) {
					return &mongodbatlas.CloudProviderAccessRole{
						AtlasAWSAccountARN:         "atlas-account-arn-3",
						AtlasAssumedRoleExternalID: "atlas-external-role-id-3",
						CreatedDate:                "created-date-3",
						ProviderName:               "AWS",
						RoleID:                     "role-3",
					}, &mongodbatlas.Response{}, nil
				},
				AuthorizeRoleFunc: func(projectID, roleID string, cpa *mongodbatlas.CloudProviderAccessRoleRequest) (*mongodbatlas.CloudProviderAccessRole, *mongodbatlas.Response, error) {
					atlasCPA := atlasCPAs[1]
					atlasCPA.AuthorizedDate = "authorized-date-2"

					return &atlasCPA, &mongodbatlas.Response{}, nil
				},
				DeauthorizeRoleFunc: func(cpa *mongodbatlas.CloudProviderDeauthorizationRequest) (*mongodbatlas.Response, error) {
					return &mongodbatlas.Response{}, nil
				},
			},
		}
		workflowCtx := &workflow.Context{
			Client:  &atlasClient,
			Context: context.Background(),
		}

		result, err := syncCloudProviderIntegration(workflowCtx, "projectID", cpas)
		assert.NoError(t, err)
		assert.False(t, result)
	})

	t.Run("should synchronize all operations and reach ready status", func(t *testing.T) {
		cpas := []mdbv1.CloudProviderIntegration{
			{
				ProviderName:      "AWS",
				IamAssumedRoleArn: "aws:arn/my_role-1",
			},
			{
				ProviderName:      "AWS",
				IamAssumedRoleArn: "aws:arn/my_role-2",
			},
		}
		atlasCPAs := []mongodbatlas.CloudProviderAccessRole{
			{
				AtlasAWSAccountARN:         "atlas-account-arn-1",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
				AuthorizedDate:             "authorized-date-1",
				CreatedDate:                "created-date-1",
				IAMAssumedRoleARN:          "aws:arn/my_role-1",
				ProviderName:               "AWS",
				RoleID:                     "role-1",
			},
			{
				AtlasAWSAccountARN:         "atlas-account-arn-2",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-2",
				CreatedDate:                "created-date-2",
				IAMAssumedRoleARN:          "aws:arn/my_role-2",
				ProviderName:               "AWS",
				RoleID:                     "role-2",
			},
		}
		atlasClient := mongodbatlas.Client{
			CloudProviderAccess: &atlas.CloudProviderAccessClientMock{
				ListRolesFunc: func(projectID string) (*mongodbatlas.CloudProviderAccessRoles, *mongodbatlas.Response, error) {
					return &mongodbatlas.CloudProviderAccessRoles{
						AWSIAMRoles: atlasCPAs,
					}, &mongodbatlas.Response{}, nil
				},
				AuthorizeRoleFunc: func(projectID, roleID string, cpa *mongodbatlas.CloudProviderAccessRoleRequest) (*mongodbatlas.CloudProviderAccessRole, *mongodbatlas.Response, error) {
					atlasCPA := atlasCPAs[1]
					atlasCPA.AuthorizedDate = "authorized-date-2"

					return &atlasCPA, &mongodbatlas.Response{}, nil
				},
			},
		}
		workflowCtx := &workflow.Context{
			Client:  &atlasClient,
			Context: context.Background(),
		}

		result, err := syncCloudProviderIntegration(workflowCtx, "projectID", cpas)
		assert.NoError(t, err)
		assert.True(t, result)
	})

	t.Run("should synchronize operations with errors", func(t *testing.T) {
		cpas := []mdbv1.CloudProviderIntegration{
			{
				ProviderName:      "AWS",
				IamAssumedRoleArn: "aws:arn/my_role-1",
			},
			{
				ProviderName:      "AWS",
				IamAssumedRoleArn: "aws:arn/my_role-2",
			},
		}
		atlasCPAs := []mongodbatlas.CloudProviderAccessRole{
			{
				AtlasAWSAccountARN:         "atlas-account-arn-1",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
				AuthorizedDate:             "authorized-date-1",
				CreatedDate:                "created-date-1",
				IAMAssumedRoleARN:          "aws:arn/my_role-1",
				ProviderName:               "AWS",
				RoleID:                     "role-1",
			},
			{
				AtlasAWSAccountARN:         "atlas-account-arn-2",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-2",
				CreatedDate:                "created-date-2",
				IAMAssumedRoleARN:          "aws:arn/my_role-2",
				ProviderName:               "AWS",
				RoleID:                     "role-2",
			},
		}
		atlasClient := mongodbatlas.Client{
			CloudProviderAccess: &atlas.CloudProviderAccessClientMock{
				ListRolesFunc: func(projectID string) (*mongodbatlas.CloudProviderAccessRoles, *mongodbatlas.Response, error) {
					return &mongodbatlas.CloudProviderAccessRoles{
						AWSIAMRoles: atlasCPAs,
					}, &mongodbatlas.Response{}, nil
				},
				AuthorizeRoleFunc: func(projectID, roleID string, cpa *mongodbatlas.CloudProviderAccessRoleRequest) (*mongodbatlas.CloudProviderAccessRole, *mongodbatlas.Response, error) {
					return nil, &mongodbatlas.Response{}, errors.New("service unavailable")
				},
			},
		}
		workflowCtx := &workflow.Context{
			Client:  &atlasClient,
			Log:     zaptest.NewLogger(t).Sugar(),
			Context: context.Background(),
		}

		result, err := syncCloudProviderIntegration(workflowCtx, "projectID", cpas)
		assert.EqualError(t, err, "not all items were synchronized successfully")
		assert.False(t, result)
	})
}

func TestInitiateStatus(t *testing.T) {
	t.Run("should create a cloud provider status as new", func(t *testing.T) {
		expected := []*status.CloudProviderIntegration{
			{
				ProviderName:      "AWS",
				IamAssumedRoleArn: "aws:arn/my_role",
				Status:            status.CloudProviderIntegrationStatusNew,
			},
			{
				ProviderName: "AWS",
				Status:       status.CloudProviderIntegrationStatusNew,
			},
		}
		spec := []mdbv1.CloudProviderIntegration{
			{
				ProviderName:      "AWS",
				IamAssumedRoleArn: "aws:arn/my_role",
			},
			{
				ProviderName: "AWS",
			},
		}

		assert.Equal(t, expected, initiateStatuses(spec))
	})
}

func TestEnrichStatuses(t *testing.T) {
	t.Run("all statuses are new", func(t *testing.T) {
		expected := []*status.CloudProviderIntegration{
			{
				ProviderName:      "AWS",
				IamAssumedRoleArn: "aws:arn/my_role",
				Status:            status.CloudProviderIntegrationStatusNew,
			},
			{
				ProviderName: "AWS",
				Status:       status.CloudProviderIntegrationStatusNew,
			},
		}
		statuses := []*status.CloudProviderIntegration{
			{
				ProviderName:      "AWS",
				IamAssumedRoleArn: "aws:arn/my_role",
				Status:            status.CloudProviderIntegrationStatusNew,
			},
			{
				ProviderName: "AWS",
				Status:       status.CloudProviderIntegrationStatusNew,
			},
		}
		assert.Equal(t, expected, enrichStatuses(statuses, []mongodbatlas.CloudProviderAccessRole{}))
	})

	t.Run("one new and one authorized statuses", func(t *testing.T) {
		expected := []*status.CloudProviderIntegration{
			{
				AtlasAWSAccountArn:         "atlas-account-arn",
				AtlasAssumedRoleExternalID: "atlas-external-role-id",
				AuthorizedDate:             "authorized-date",
				CreatedDate:                "created-date",
				IamAssumedRoleArn:          "aws:arn/my_role",
				ProviderName:               "AWS",
				RoleID:                     "role-1",
				Status:                     status.CloudProviderIntegrationStatusAuthorized,
				ErrorMessage:               "",
			},
			{
				ProviderName: "AWS",
				Status:       status.CloudProviderIntegrationStatusNew,
			},
		}
		statuses := []*status.CloudProviderIntegration{
			{
				ProviderName:      "AWS",
				IamAssumedRoleArn: "aws:arn/my_role",
				Status:            status.CloudProviderIntegrationStatusNew,
			},
			{
				ProviderName: "AWS",
				Status:       status.CloudProviderIntegrationStatusNew,
			},
		}
		atlasCPAs := []mongodbatlas.CloudProviderAccessRole{
			{
				AtlasAWSAccountARN:         "atlas-account-arn",
				AtlasAssumedRoleExternalID: "atlas-external-role-id",
				AuthorizedDate:             "authorized-date",
				CreatedDate:                "created-date",
				IAMAssumedRoleARN:          "aws:arn/my_role",
				ProviderName:               "AWS",
				RoleID:                     "role-1",
			},
		}
		assert.Equal(t, expected, enrichStatuses(statuses, atlasCPAs))
	})

	t.Run("one new, one created and one authorized statuses", func(t *testing.T) {
		expected := []*status.CloudProviderIntegration{
			{
				AtlasAWSAccountArn:         "atlas-account-arn-1",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
				AuthorizedDate:             "authorized-date-1",
				CreatedDate:                "created-date-1",
				IamAssumedRoleArn:          "aws:arn/my_role-1",
				ProviderName:               "AWS",
				RoleID:                     "role-1",
				Status:                     status.CloudProviderIntegrationStatusAuthorized,
				ErrorMessage:               "",
			},
			{
				AtlasAWSAccountArn:         "atlas-account-arn-2",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-2",
				CreatedDate:                "created-date-2",
				IamAssumedRoleArn:          "aws:arn/my_role-2",
				ProviderName:               "AWS",
				RoleID:                     "role-2",
				Status:                     status.CloudProviderIntegrationStatusCreated,
				ErrorMessage:               "",
			},
			{
				ProviderName: "AWS",
				Status:       status.CloudProviderIntegrationStatusNew,
			},
		}
		statuses := []*status.CloudProviderIntegration{
			{
				ProviderName:      "AWS",
				IamAssumedRoleArn: "aws:arn/my_role-1",
				Status:            status.CloudProviderIntegrationStatusNew,
			},
			{
				ProviderName:      "AWS",
				IamAssumedRoleArn: "aws:arn/my_role-2",
				Status:            status.CloudProviderIntegrationStatusNew,
			},
			{
				ProviderName: "AWS",
				Status:       status.CloudProviderIntegrationStatusNew,
			},
		}
		atlasCPAs := []mongodbatlas.CloudProviderAccessRole{
			{
				AtlasAWSAccountARN:         "atlas-account-arn-1",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
				AuthorizedDate:             "authorized-date-1",
				CreatedDate:                "created-date-1",
				IAMAssumedRoleARN:          "aws:arn/my_role-1",
				ProviderName:               "AWS",
				RoleID:                     "role-1",
			},
			{
				AtlasAWSAccountARN:         "atlas-account-arn-2",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-2",
				CreatedDate:                "created-date-2",
				IAMAssumedRoleARN:          "aws:arn/my_role-2",
				ProviderName:               "AWS",
				RoleID:                     "role-2",
			},
		}
		assert.Equal(t, expected, enrichStatuses(statuses, atlasCPAs))
	})

	t.Run("one new, one created, one authorized, and one authorized to remove statuses", func(t *testing.T) {
		expected := []*status.CloudProviderIntegration{
			{
				AtlasAWSAccountArn:         "atlas-account-arn-1",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
				AuthorizedDate:             "authorized-date-1",
				CreatedDate:                "created-date-1",
				IamAssumedRoleArn:          "aws:arn/my_role-1",
				ProviderName:               "AWS",
				RoleID:                     "role-1",
				Status:                     status.CloudProviderIntegrationStatusAuthorized,
				ErrorMessage:               "",
			},
			{
				AtlasAWSAccountArn:         "atlas-account-arn-2",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-2",
				CreatedDate:                "created-date-2",
				IamAssumedRoleArn:          "aws:arn/my_role-2",
				ProviderName:               "AWS",
				RoleID:                     "role-2",
				Status:                     status.CloudProviderIntegrationStatusCreated,
				ErrorMessage:               "",
			},
			{
				ProviderName: "AWS",
				Status:       status.CloudProviderIntegrationStatusNew,
			},
			{
				AtlasAWSAccountArn:         "atlas-account-arn-3",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-3",
				AuthorizedDate:             "authorized-date-3",
				CreatedDate:                "created-date-3",
				IamAssumedRoleArn:          "aws:arn/my_role-3",
				ProviderName:               "AWS",
				RoleID:                     "role-3",
				Status:                     status.CloudProviderIntegrationStatusDeAuthorize,
				ErrorMessage:               "",
			},
		}
		statuses := []*status.CloudProviderIntegration{
			{
				ProviderName:      "AWS",
				IamAssumedRoleArn: "aws:arn/my_role-1",
				Status:            status.CloudProviderIntegrationStatusNew,
			},
			{
				ProviderName:      "AWS",
				IamAssumedRoleArn: "aws:arn/my_role-2",
				Status:            status.CloudProviderIntegrationStatusNew,
			},
			{
				ProviderName: "AWS",
				Status:       status.CloudProviderIntegrationStatusNew,
			},
		}
		atlasCPAs := []mongodbatlas.CloudProviderAccessRole{
			{
				AtlasAWSAccountARN:         "atlas-account-arn-1",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
				AuthorizedDate:             "authorized-date-1",
				CreatedDate:                "created-date-1",
				IAMAssumedRoleARN:          "aws:arn/my_role-1",
				ProviderName:               "AWS",
				RoleID:                     "role-1",
			},
			{
				AtlasAWSAccountARN:         "atlas-account-arn-3",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-3",
				AuthorizedDate:             "authorized-date-3",
				CreatedDate:                "created-date-3",
				IAMAssumedRoleARN:          "aws:arn/my_role-3",
				ProviderName:               "AWS",
				RoleID:                     "role-3",
			},
			{
				AtlasAWSAccountARN:         "atlas-account-arn-2",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-2",
				CreatedDate:                "created-date-2",
				IAMAssumedRoleARN:          "aws:arn/my_role-2",
				ProviderName:               "AWS",
				RoleID:                     "role-2",
			},
		}
		assert.Equal(t, expected, enrichStatuses(statuses, atlasCPAs))
	})

	t.Run("one created with empty ARN, one created, and one authorized statuses", func(t *testing.T) {
		expected := []*status.CloudProviderIntegration{
			{
				AtlasAWSAccountArn:         "atlas-account-arn-1",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
				AuthorizedDate:             "authorized-date-1",
				CreatedDate:                "created-date-1",
				IamAssumedRoleArn:          "aws:arn/my_role-1",
				ProviderName:               "AWS",
				RoleID:                     "role-1",
				Status:                     status.CloudProviderIntegrationStatusAuthorized,
				ErrorMessage:               "",
			},
			{
				AtlasAWSAccountArn:         "atlas-account-arn-2",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-2",
				CreatedDate:                "created-date-2",
				IamAssumedRoleArn:          "aws:arn/my_role-2",
				ProviderName:               "AWS",
				RoleID:                     "role-2",
				Status:                     status.CloudProviderIntegrationStatusCreated,
				ErrorMessage:               "",
			},
			{
				AtlasAWSAccountArn:         "atlas-account-arn-3",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-3",
				CreatedDate:                "created-date-3",
				ProviderName:               "AWS",
				RoleID:                     "role-3",
				Status:                     status.CloudProviderIntegrationStatusCreated,
				ErrorMessage:               "",
			},
		}
		statuses := []*status.CloudProviderIntegration{
			{
				ProviderName:      "AWS",
				IamAssumedRoleArn: "aws:arn/my_role-1",
				Status:            status.CloudProviderIntegrationStatusNew,
			},
			{
				ProviderName:      "AWS",
				IamAssumedRoleArn: "aws:arn/my_role-2",
				Status:            status.CloudProviderIntegrationStatusNew,
			},
			{
				ProviderName: "AWS",
				Status:       status.CloudProviderIntegrationStatusNew,
			},
		}
		atlasCPAs := []mongodbatlas.CloudProviderAccessRole{
			{
				AtlasAWSAccountARN:         "atlas-account-arn-1",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
				AuthorizedDate:             "authorized-date-1",
				CreatedDate:                "created-date-1",
				IAMAssumedRoleARN:          "aws:arn/my_role-1",
				ProviderName:               "AWS",
				RoleID:                     "role-1",
			},
			{
				AtlasAWSAccountARN:         "atlas-account-arn-3",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-3",
				CreatedDate:                "created-date-3",
				ProviderName:               "AWS",
				RoleID:                     "role-3",
			},
			{
				AtlasAWSAccountARN:         "atlas-account-arn-2",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-2",
				CreatedDate:                "created-date-2",
				IAMAssumedRoleARN:          "aws:arn/my_role-2",
				ProviderName:               "AWS",
				RoleID:                     "role-2",
			},
		}
		assert.Equal(t, expected, enrichStatuses(statuses, atlasCPAs))
	})

	t.Run("one created with empty ARN, one created, one authorized, and one to be removed statuses", func(t *testing.T) {
		expected := []*status.CloudProviderIntegration{
			{
				AtlasAWSAccountArn:         "atlas-account-arn-1",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
				AuthorizedDate:             "authorized-date-1",
				CreatedDate:                "created-date-1",
				IamAssumedRoleArn:          "aws:arn/my_role-1",
				ProviderName:               "AWS",
				RoleID:                     "role-1",
				Status:                     status.CloudProviderIntegrationStatusAuthorized,
				ErrorMessage:               "",
			},
			{
				AtlasAWSAccountArn:         "atlas-account-arn-2",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-2",
				CreatedDate:                "created-date-2",
				IamAssumedRoleArn:          "aws:arn/my_role-2",
				ProviderName:               "AWS",
				RoleID:                     "role-2",
				Status:                     status.CloudProviderIntegrationStatusCreated,
				ErrorMessage:               "",
			},
			{
				AtlasAWSAccountArn:         "atlas-account-arn-3",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-3",
				CreatedDate:                "created-date-3",
				ProviderName:               "AWS",
				RoleID:                     "role-3",
				Status:                     status.CloudProviderIntegrationStatusCreated,
				ErrorMessage:               "",
			},
			{
				AtlasAWSAccountArn:         "atlas-account-arn-4",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-4",
				CreatedDate:                "created-date-4",
				ProviderName:               "AWS",
				RoleID:                     "role-4",
				Status:                     status.CloudProviderIntegrationStatusDeAuthorize,
				ErrorMessage:               "",
			},
		}
		statuses := []*status.CloudProviderIntegration{
			{
				ProviderName:      "AWS",
				IamAssumedRoleArn: "aws:arn/my_role-1",
				Status:            status.CloudProviderIntegrationStatusNew,
			},
			{
				ProviderName:      "AWS",
				IamAssumedRoleArn: "aws:arn/my_role-2",
				Status:            status.CloudProviderIntegrationStatusNew,
			},
			{
				ProviderName: "AWS",
				Status:       status.CloudProviderIntegrationStatusNew,
			},
		}
		atlasCPAs := []mongodbatlas.CloudProviderAccessRole{
			{
				AtlasAWSAccountARN:         "atlas-account-arn-1",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
				AuthorizedDate:             "authorized-date-1",
				CreatedDate:                "created-date-1",
				IAMAssumedRoleARN:          "aws:arn/my_role-1",
				ProviderName:               "AWS",
				RoleID:                     "role-1",
			},
			{
				AtlasAWSAccountARN:         "atlas-account-arn-3",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-3",
				CreatedDate:                "created-date-3",
				ProviderName:               "AWS",
				RoleID:                     "role-3",
			},
			{
				AtlasAWSAccountARN:         "atlas-account-arn-2",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-2",
				CreatedDate:                "created-date-2",
				IAMAssumedRoleARN:          "aws:arn/my_role-2",
				ProviderName:               "AWS",
				RoleID:                     "role-2",
			},
			{
				AtlasAWSAccountARN:         "atlas-account-arn-4",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-4",
				CreatedDate:                "created-date-4",
				ProviderName:               "AWS",
				RoleID:                     "role-4",
			},
		}
		assert.Equal(t, expected, enrichStatuses(statuses, atlasCPAs))
	})

	t.Run("match two status with empty ARN and two existing on Atlas", func(t *testing.T) {
		expected := []*status.CloudProviderIntegration{
			{
				ProviderName:               "AWS",
				AtlasAWSAccountArn:         "atlas-account-arn-1",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
				RoleID:                     "role-1",
				CreatedDate:                "created-date-1",
				Status:                     status.CloudProviderIntegrationStatusCreated,
			},
			{
				ProviderName:               "AWS",
				AtlasAWSAccountArn:         "atlas-account-arn-2",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-2",
				RoleID:                     "role-2",
				CreatedDate:                "created-date-2",
				Status:                     status.CloudProviderIntegrationStatusCreated,
			},
		}
		statuses := []*status.CloudProviderIntegration{
			{
				ProviderName: "AWS",
				Status:       status.CloudProviderIntegrationStatusNew,
			},
			{
				ProviderName: "AWS",
				Status:       status.CloudProviderIntegrationStatusNew,
			},
		}
		atlasCPAs := []mongodbatlas.CloudProviderAccessRole{
			{
				AtlasAWSAccountARN:         "atlas-account-arn-1",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
				CreatedDate:                "created-date-1",
				ProviderName:               "AWS",
				RoleID:                     "role-1",
			},
			{
				AtlasAWSAccountARN:         "atlas-account-arn-2",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-2",
				CreatedDate:                "created-date-2",
				ProviderName:               "AWS",
				RoleID:                     "role-2",
			},
		}
		assert.Equal(t, expected, enrichStatuses(statuses, atlasCPAs))
	})

	t.Run("match two status with empty ARN and update them with ARN", func(t *testing.T) {
		expected := []*status.CloudProviderIntegration{
			{
				ProviderName:               "AWS",
				IamAssumedRoleArn:          "was:arn/role-1",
				AtlasAWSAccountArn:         "atlas-account-arn-1",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
				RoleID:                     "role-1",
				CreatedDate:                "created-date-1",
				Status:                     status.CloudProviderIntegrationStatusCreated,
			},
			{
				ProviderName:               "AWS",
				IamAssumedRoleArn:          "was:arn/role-2",
				AtlasAWSAccountArn:         "atlas-account-arn-2",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-2",
				RoleID:                     "role-2",
				CreatedDate:                "created-date-2",
				Status:                     status.CloudProviderIntegrationStatusCreated,
			},
		}
		statuses := []*status.CloudProviderIntegration{
			{
				ProviderName:      "AWS",
				IamAssumedRoleArn: "was:arn/role-1",
				Status:            status.CloudProviderIntegrationStatusNew,
			},
			{
				ProviderName:      "AWS",
				IamAssumedRoleArn: "was:arn/role-2",
				Status:            status.CloudProviderIntegrationStatusNew,
			},
		}
		atlasCPAs := []mongodbatlas.CloudProviderAccessRole{
			{
				AtlasAWSAccountARN:         "atlas-account-arn-1",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
				CreatedDate:                "created-date-1",
				ProviderName:               "AWS",
				RoleID:                     "role-1",
			},
			{
				AtlasAWSAccountARN:         "atlas-account-arn-2",
				AtlasAssumedRoleExternalID: "atlas-external-role-id-2",
				CreatedDate:                "created-date-2",
				ProviderName:               "AWS",
				RoleID:                     "role-2",
			},
		}

		assert.Equal(t, expected, enrichStatuses(statuses, atlasCPAs))
	})
}

func TestCreateCloudProviderIntegration(t *testing.T) {
	t.Run("should create cloud provider integration successfully", func(t *testing.T) {
		expected := &status.CloudProviderIntegration{
			AtlasAWSAccountArn:         "atlas-account-arn-1",
			AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
			CreatedDate:                "created-date-1",
			IamAssumedRoleArn:          "aws:arn/my_role-1",
			ProviderName:               "AWS",
			RoleID:                     "role-1",
			Status:                     status.CloudProviderIntegrationStatusCreated,
			ErrorMessage:               "",
		}
		cpa := &status.CloudProviderIntegration{
			ProviderName:      "AWS",
			IamAssumedRoleArn: "aws:arn/my_role-1",
			Status:            status.CloudProviderIntegrationStatusNew,
		}
		atlasClient := mongodbatlas.Client{
			CloudProviderAccess: &atlas.CloudProviderAccessClientMock{
				CreateRoleFunc: func(projectID string, cpa *mongodbatlas.CloudProviderAccessRoleRequest) (*mongodbatlas.CloudProviderAccessRole, *mongodbatlas.Response, error) {
					return &mongodbatlas.CloudProviderAccessRole{
						AtlasAWSAccountARN:         "atlas-account-arn-1",
						AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
						CreatedDate:                "created-date-1",
						ProviderName:               "AWS",
						RoleID:                     "role-1",
					}, &mongodbatlas.Response{}, nil
				},
			},
		}
		workflowCtx := &workflow.Context{
			Client:  &atlasClient,
			Context: context.Background(),
		}

		assert.Equal(t, expected, createCloudProviderAccess(workflowCtx, "projectID", cpa))
	})

	t.Run("should fail to create cloud provider integration", func(t *testing.T) {
		expected := &status.CloudProviderIntegration{
			ProviderName:      "AWS",
			IamAssumedRoleArn: "aws:arn/my_role-1",
			Status:            status.CloudProviderIntegrationStatusFailedToCreate,
			ErrorMessage:      "service unavailable",
		}
		cpa := &status.CloudProviderIntegration{
			ProviderName:      "AWS",
			IamAssumedRoleArn: "aws:arn/my_role-1",
			Status:            status.CloudProviderIntegrationStatusNew,
		}
		atlasClient := mongodbatlas.Client{
			CloudProviderAccess: &atlas.CloudProviderAccessClientMock{
				CreateRoleFunc: func(projectID string, cpa *mongodbatlas.CloudProviderAccessRoleRequest) (*mongodbatlas.CloudProviderAccessRole, *mongodbatlas.Response, error) {
					return nil, &mongodbatlas.Response{}, errors.New("service unavailable")
				},
			},
		}
		workflowCtx := &workflow.Context{
			Client:  &atlasClient,
			Log:     zaptest.NewLogger(t).Sugar(),
			Context: context.Background(),
		}

		assert.Equal(t, expected, createCloudProviderAccess(workflowCtx, "projectID", cpa))
	})
}

func TestAuthorizeCloudProviderIntegration(t *testing.T) {
	t.Run("should authorize cloud provider integration successfully", func(t *testing.T) {
		expected := &status.CloudProviderIntegration{
			AtlasAWSAccountArn:         "atlas-account-arn-1",
			AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
			CreatedDate:                "created-date-1",
			AuthorizedDate:             "authorized-date-1",
			IamAssumedRoleArn:          "aws:arn/my_role-1",
			ProviderName:               "AWS",
			RoleID:                     "role-1",
			Status:                     status.CloudProviderIntegrationStatusAuthorized,
			ErrorMessage:               "",
		}
		cpa := &status.CloudProviderIntegration{
			AtlasAWSAccountArn:         "atlas-account-arn-1",
			AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
			CreatedDate:                "created-date-1",
			IamAssumedRoleArn:          "aws:arn/my_role-1",
			ProviderName:               "AWS",
			RoleID:                     "role-1",
			Status:                     status.CloudProviderIntegrationStatusNew,
		}
		atlasClient := mongodbatlas.Client{
			CloudProviderAccess: &atlas.CloudProviderAccessClientMock{
				AuthorizeRoleFunc: func(projectID, roleID string, cpa *mongodbatlas.CloudProviderAccessRoleRequest) (*mongodbatlas.CloudProviderAccessRole, *mongodbatlas.Response, error) {
					return &mongodbatlas.CloudProviderAccessRole{
						AtlasAWSAccountARN:         "atlas-account-arn-1",
						AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
						CreatedDate:                "created-date-1",
						AuthorizedDate:             "authorized-date-1",
						ProviderName:               "AWS",
						RoleID:                     "role-1",
					}, &mongodbatlas.Response{}, nil
				},
			},
		}
		workflowCtx := &workflow.Context{
			Client:  &atlasClient,
			Context: context.Background(),
		}

		assert.Equal(t, expected, authorizeCloudProviderAccess(workflowCtx, "projectID", cpa))
	})

	t.Run("should fail to authorize cloud provider integration", func(t *testing.T) {
		expected := &status.CloudProviderIntegration{
			AtlasAWSAccountArn:         "atlas-account-arn-1",
			AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
			CreatedDate:                "created-date-1",
			IamAssumedRoleArn:          "aws:arn/my_role-1",
			ProviderName:               "AWS",
			RoleID:                     "role-1",
			Status:                     status.CloudProviderIntegrationStatusFailedToAuthorize,
			ErrorMessage:               "service unavailable",
		}
		cpa := &status.CloudProviderIntegration{
			AtlasAWSAccountArn:         "atlas-account-arn-1",
			AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
			CreatedDate:                "created-date-1",
			IamAssumedRoleArn:          "aws:arn/my_role-1",
			ProviderName:               "AWS",
			RoleID:                     "role-1",
			Status:                     status.CloudProviderIntegrationStatusCreated,
		}
		atlasClient := mongodbatlas.Client{
			CloudProviderAccess: &atlas.CloudProviderAccessClientMock{
				AuthorizeRoleFunc: func(projectID, roleID string, cpa *mongodbatlas.CloudProviderAccessRoleRequest) (*mongodbatlas.CloudProviderAccessRole, *mongodbatlas.Response, error) {
					return nil, &mongodbatlas.Response{}, errors.New("service unavailable")
				},
			},
		}
		workflowCtx := &workflow.Context{
			Client:  &atlasClient,
			Log:     zaptest.NewLogger(t).Sugar(),
			Context: context.Background(),
		}

		assert.Equal(t, expected, authorizeCloudProviderAccess(workflowCtx, "projectID", cpa))
	})
}

func TestDeleteCloudProviderIntegration(t *testing.T) {
	t.Run("should delete cloud provider integration successfully", func(t *testing.T) {
		cpa := &status.CloudProviderIntegration{
			AtlasAWSAccountArn:         "atlas-account-arn-1",
			AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
			CreatedDate:                "created-date-1",
			AuthorizedDate:             "authorized-date-1",
			IamAssumedRoleArn:          "aws:arn/my_role-1",
			ProviderName:               "AWS",
			RoleID:                     "role-1",
			Status:                     status.CloudProviderIntegrationStatusFailedToDeAuthorize,
			ErrorMessage:               "",
		}
		atlasClient := mongodbatlas.Client{
			CloudProviderAccess: &atlas.CloudProviderAccessClientMock{
				DeauthorizeRoleFunc: func(cpa *mongodbatlas.CloudProviderDeauthorizationRequest) (*mongodbatlas.Response, error) {
					return &mongodbatlas.Response{}, nil
				},
			},
		}
		workflowCtx := &workflow.Context{
			Client:  &atlasClient,
			Context: context.Background(),
		}

		deleteCloudProviderAccess(workflowCtx, "projectID", cpa)
		assert.Empty(t, cpa.ErrorMessage)
	})

	t.Run("should fail to delete cloud provider integration", func(t *testing.T) {
		cpa := &status.CloudProviderIntegration{
			AtlasAWSAccountArn:         "atlas-account-arn-1",
			AtlasAssumedRoleExternalID: "atlas-external-role-id-1",
			CreatedDate:                "created-date-1",
			ProviderName:               "AWS",
			RoleID:                     "role-1",
			Status:                     status.CloudProviderIntegrationStatusFailedToDeAuthorize,
			ErrorMessage:               "",
		}
		atlasClient := mongodbatlas.Client{
			CloudProviderAccess: &atlas.CloudProviderAccessClientMock{
				DeauthorizeRoleFunc: func(cpa *mongodbatlas.CloudProviderDeauthorizationRequest) (*mongodbatlas.Response, error) {
					return &mongodbatlas.Response{}, errors.New("service unavailable")
				},
			},
		}
		workflowCtx := &workflow.Context{
			Client:  &atlasClient,
			Log:     zaptest.NewLogger(t).Sugar(),
			Context: context.Background(),
		}

		deleteCloudProviderAccess(workflowCtx, "projectID", cpa)
		assert.Equal(t, "service unavailable", cpa.ErrorMessage)
	})
}
