package atlasproject

import (
	"reflect"

	v1 "github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/api/v1"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/api/v1/status"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/controller/workflow"
)

func ensureProjectSettings(workflowCtx *workflow.Context, project *v1.AtlasProject, protected bool) (result workflow.Result) {
	if result = syncProjectSettings(workflowCtx, project.ID(), project); !result.IsOk() {
		workflowCtx.SetConditionFromResult(status.ProjectSettingsReadyType, result)
		return result
	}

	if project.Spec.Settings == nil {
		workflowCtx.UnsetCondition(status.ProjectSettingsReadyType)
		return workflow.OK()
	}

	workflowCtx.SetConditionTrue(status.ProjectSettingsReadyType)
	return workflow.OK()
}

func syncProjectSettings(ctx *workflow.Context, projectID string, project *v1.AtlasProject) workflow.Result {
	spec := project.Spec.Settings

	atlas, err := fetchSettings(ctx, projectID)
	if err != nil {
		return workflow.Terminate(workflow.ProjectSettingsReady, err.Error())
	}

	if !areSettingsInSync(atlas, spec) {
		if err := patchSettings(ctx, projectID, spec); err != nil {
			return workflow.Terminate(workflow.ProjectSettingsReady, err.Error())
		}
	}

	return workflow.OK()
}

func areSettingsInSync(atlas, spec *v1.ProjectSettings) bool {
	return isOneContainedInOther(spec, atlas)
}

func patchSettings(ctx *workflow.Context, projectID string, spec *v1.ProjectSettings) error {
	specAsAtlas, err := spec.ToAtlas()
	if err != nil {
		return err
	}

	_, _, err = ctx.Client.Projects.UpdateProjectSettings(ctx.Context, projectID, specAsAtlas)
	return err
}

func fetchSettings(ctx *workflow.Context, projectID string) (*v1.ProjectSettings, error) {
	data, _, err := ctx.Client.Projects.GetProjectSettings(ctx.Context, projectID)
	if err != nil {
		return nil, err
	}
	ctx.Log.Debugw("Got Project Settings", "data", data)

	settings := v1.ProjectSettings(*data)
	return &settings, nil
}

func isOneContainedInOther(one, other *v1.ProjectSettings) bool {
	if one == nil {
		return true
	}

	if other == nil {
		return false
	}

	oneVal := reflect.ValueOf(one).Elem()
	otherVal := reflect.ValueOf(other).Elem()

	for i := 0; i < oneVal.NumField(); i++ {
		if oneVal.Field(i).IsNil() {
			continue
		}

		oneBool := oneVal.Field(i).Elem().Bool()
		otherBool := otherVal.Field(i).Elem().Bool()

		if oneBool != otherBool {
			return false
		}
	}

	return true
}
