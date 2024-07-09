package v1

import (
	"math/rand"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"

	"go.mongodb.org/atlas-sdk/v20231115004/admin"

	"github.com/mongodb/mongodb-atlas-kubernetes/v2/pkg/api/v1/common"

	"github.com/google/go-cmp/cmp"
	"sigs.k8s.io/yaml"

	internalcmp "github.com/mongodb/mongodb-atlas-kubernetes/v2/internal/cmp"
	"github.com/mongodb/mongodb-atlas-kubernetes/v2/test/cel"
)

func TestSpecEquality(t *testing.T) {
	ref := &AtlasProjectSpec{
		PrivateEndpoints: []PrivateEndpoint{
			{
				Endpoints: GCPEndpoints{
					{
						EndpointName: "foo",
						IPAddress:    "bar",
					},
					{
						EndpointName: "123",
						IPAddress:    "456",
					},
				},
			},
		},
		AlertConfigurations: []AlertConfiguration{
			{
				Enabled:       true,
				EventTypeName: "foo",
				Notifications: []Notification{
					{
						APITokenRef: common.ResourceRefNamespaced{
							Name: "foo",
						},
						ChannelName: "bar",
						DelayMin:    admin.PtrInt(1),
					},
					{
						ChannelName: "foo",
						DelayMin:    admin.PtrInt(2),
						Roles:       []string{"2", "3", "1"},
					},
					{
						ChannelName: "foo",
						DelayMin:    admin.PtrInt(2),
					},
					{
						APITokenRef: common.ResourceRefNamespaced{
							Name: "bar",
						},
						ChannelName: "bar",
						DelayMin:    admin.PtrInt(1),
					},
				},
			},
			{
				Enabled:       true,
				EventTypeName: "foo",
				Matchers: []Matcher{
					{
						FieldName: "foo",
					},
					{
						FieldName: "bar",
						Operator:  "foo",
					},
					{
						FieldName: "bar",
						Operator:  "bar",
					},
					{
						FieldName: "baz",
						Operator:  "foo",
					},
				},
			},
			{
				Enabled:       true,
				EventTypeName: "foo",
			},
			{
				Enabled:       true,
				EventTypeName: "foo",
			},
		},
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	err := internalcmp.Normalize(ref)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100_000; i++ {
		perm := ref.DeepCopy()
		internalcmp.PermuteOrder(perm, r)
		err := internalcmp.Normalize(perm)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(ref, perm) {
			jRef := mustMarshal(t, ref)
			jPermutedCopy := mustMarshal(t, perm)
			t.Errorf("expected reference:\n%v\nto be equal to the reordered copy:\n%v\nbut it isn't, diff:\n%v",
				jRef, jPermutedCopy, cmp.Diff(jRef, jPermutedCopy),
			)
			return
		}
	}
}

func mustMarshal(t *testing.T, what any) string {
	t.Helper()
	result, err := yaml.Marshal(what)
	if err != nil {
		t.Fatal(err)
	}
	return string(result)
}

func TestCEL(t *testing.T) {
	testCases := []struct {
		name         string
		current, old runtime.Object
		wantErrs     []string
	}{
		{
			// Note: It would be desirable if this case failed as well.
			// This will become possible with CRD ratcheting and "optionalOldSelf: true" in the CRD declaration.
			// See https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#field-optional-oldself.
			name: "creating an AtlasProject with custom roles succeeds",
			old:  nil,
			current: &AtlasProject{
				Spec: AtlasProjectSpec{
					CustomRoles: []CustomRole{
						{Name: "foo"},
					},
				},
			},
		},
		{
			name: "updating an AtlasProject and adding an empty custom roles field succeeds",
			old: &AtlasProject{
				Spec: AtlasProjectSpec{},
			},
			current: &AtlasProject{
				Spec: AtlasProjectSpec{
					CustomRoles: []CustomRole{},
				},
			},
		},
		{
			name: "updating an AtlasProject and adding a custom role fails",
			old: &AtlasProject{
				Spec: AtlasProjectSpec{},
			},
			current: &AtlasProject{
				Spec: AtlasProjectSpec{
					CustomRoles: []CustomRole{
						{Name: "foo"},
					},
				},
			},
			wantErrs: []string{
				`spec: Invalid value: "object": setting new customRoles is invalid: use the AtlasCustomRole CRD instead.`,
			},
		},
		{
			name: "updating an AtlasProject with an empty custom roles fields and adding a custom role fails",
			old: &AtlasProject{
				Spec: AtlasProjectSpec{
					CustomRoles: []CustomRole{},
				},
			},
			current: &AtlasProject{
				Spec: AtlasProjectSpec{
					CustomRoles: []CustomRole{
						{Name: "foo"},
					},
				},
			},
			wantErrs: []string{
				`spec: Invalid value: "object": setting new customRoles is invalid: use the AtlasCustomRole CRD instead.`,
			},
		},
		{
			name: "updating an AtlasProject with existing custom roles and adding a custom role succeeds",
			old: &AtlasProject{
				Spec: AtlasProjectSpec{
					CustomRoles: []CustomRole{
						{Name: "foo"},
					},
				},
			},
			current: &AtlasProject{
				Spec: AtlasProjectSpec{
					CustomRoles: []CustomRole{
						{Name: "foo"},
						{Name: "bar"},
					},
				},
			},
		},
		{
			name: "updating an AtlasProject with existing custom roles and removing a custom role succeeds",
			old: &AtlasProject{
				Spec: AtlasProjectSpec{
					CustomRoles: []CustomRole{
						{Name: "foo"},
						{Name: "bar"},
					},
				},
			},
			current: &AtlasProject{
				Spec: AtlasProjectSpec{
					CustomRoles: []CustomRole{
						{Name: "foo"},
					},
				},
			},
		},
		{
			name: "updating an AtlasProject with existing custom roles and removing all custom roles succeeds",
			old: &AtlasProject{
				Spec: AtlasProjectSpec{
					CustomRoles: []CustomRole{
						{Name: "foo"},
						{Name: "bar"},
					},
				},
			},
			current: &AtlasProject{
				Spec: AtlasProjectSpec{
					CustomRoles: []CustomRole{},
				},
			},
		},
		{
			name: "updating an AtlasProject with existing custom roles and removing the custom roles field succeeds",
			old: &AtlasProject{
				Spec: AtlasProjectSpec{
					CustomRoles: []CustomRole{
						{Name: "foo"},
						{Name: "bar"},
					},
				},
			},
			current: &AtlasProject{
				Spec: AtlasProjectSpec{},
			},
		},
	}

	validator, err := cel.VersionValidatorFromFile(t, "../../../config/crd/bases/atlas.mongodb.com_atlasprojects.yaml", "v1")
	require.NoError(t, err)
	_ = validator
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				err          error
				current, old map[string]interface{}
			)
			if tc.current != nil {
				current, err = runtime.DefaultUnstructuredConverter.ToUnstructured(tc.current)
				require.NoError(t, err)
			}
			if tc.old != nil {
				old, err = runtime.DefaultUnstructuredConverter.ToUnstructured(tc.old)
				require.NoError(t, err)
			}
			errs := validator(current, old)

			if got := len(errs); got != len(tc.wantErrs) {
				t.Errorf("expected errors %v, got %v", len(tc.wantErrs), len(errs))
				return
			}

			for i := range tc.wantErrs {
				got := errs[i].Error()
				if got != tc.wantErrs[i] {
					t.Errorf("want error %q, got %q", tc.wantErrs[i], got)
				}
			}
		})
	}
}
