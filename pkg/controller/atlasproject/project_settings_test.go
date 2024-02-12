package atlasproject

import (
	"testing"

	"github.com/mongodb/mongodb-atlas-kubernetes/v2/internal/pointer"
	mdbv1 "github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/api/v1"
	"github.com/stretchr/testify/assert"
)

func TestAreSettingsInSync(t *testing.T) {
	atlasDef := &mdbv1.ProjectSettings{
		IsCollectDatabaseSpecificsStatisticsEnabled: pointer.MakePtr(true),
		IsDataExplorerEnabled:                       pointer.MakePtr(true),
		IsPerformanceAdvisorEnabled:                 pointer.MakePtr(true),
		IsRealtimePerformancePanelEnabled:           pointer.MakePtr(true),
		IsSchemaAdvisorEnabled:                      pointer.MakePtr(true),
	}
	specDef := &mdbv1.ProjectSettings{
		IsCollectDatabaseSpecificsStatisticsEnabled: pointer.MakePtr(true),
		IsDataExplorerEnabled:                       pointer.MakePtr(true),
	}

	areEqual := areSettingsInSync(atlasDef, specDef)
	assert.True(t, areEqual, "Only fields which are set should be compared")

	specDef.IsPerformanceAdvisorEnabled = pointer.MakePtr(false)
	areEqual = areSettingsInSync(atlasDef, specDef)
	assert.False(t, areEqual, "Field values should be the same ")
}
