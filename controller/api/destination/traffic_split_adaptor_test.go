package destination

import (
	"encoding/json"
	"reflect"
	"testing"

	ts "github.com/deislabs/smi-sdk-go/pkg/apis/split/v1alpha1"
	"github.com/linkerd/linkerd2/controller/api/destination/watcher"
	sp "github.com/linkerd/linkerd2/controller/gen/apis/serviceprofile/v1alpha1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestTrafficSplitAdaptor(t *testing.T) {

	profile := &sp.ServiceProfile{
		Spec: sp.ServiceProfileSpec{
			Routes: []*sp.RouteSpec{
				{
					Name: "route",
				},
			},
			DstOverrides: []*sp.WeightedDst{
				{
					Authority: "foo",
					Weight:    resource.MustParse("500m"),
				},
			},
		},
	}

	split := &ts.TrafficSplit{
		Spec: ts.TrafficSplitSpec{
			Backends: []ts.TrafficSplitBackend{
				{
					Service: "bar",
					Weight:  resource.MustParse("1000m"),
				},
			},
		},
	}

	t.Run("Profile update", func(t *testing.T) {
		listener := watcher.NewBufferingProfileListener()
		adaptor := newTrafficSplitAdaptor(listener, watcher.ServiceID{Name: "foo", Namespace: "ns"}, watcher.Port(80))

		adaptor.Update(profile)

		if len(listener.Profiles) != 1 {
			t.Fatalf("Expected one profile updated, got %d", len(listener.Profiles))
		}
		testCompare(t, profile.Spec, listener.Profiles[0].Spec)
	})

	t.Run("Traffic split without profile", func(t *testing.T) {
		listener := watcher.NewBufferingProfileListener()
		adaptor := newTrafficSplitAdaptor(listener, watcher.ServiceID{Name: "foo", Namespace: "ns"}, watcher.Port(80))

		adaptor.UpdateTrafficSplit(split)

		if len(listener.Profiles) != 1 {
			t.Fatalf("Expected one profile updated, got %d", len(listener.Profiles))
		}

		expected := &sp.ServiceProfile{
			Spec: sp.ServiceProfileSpec{
				DstOverrides: []*sp.WeightedDst{
					{
						Authority: "bar.ns.svc.cluster.local:80",
						Weight:    resource.MustParse("1000m"),
					},
				},
			},
		}

		testCompare(t, expected.Spec, listener.Profiles[0].Spec)
	})

	t.Run("Profile merged with traffic split", func(t *testing.T) {
		listener := watcher.NewBufferingProfileListener()
		adaptor := newTrafficSplitAdaptor(listener, watcher.ServiceID{Name: "foo", Namespace: "ns"}, watcher.Port(80))

		adaptor.Update(profile)
		adaptor.UpdateTrafficSplit(split)

		if len(listener.Profiles) != 2 {
			t.Fatalf("Expected two profile updated, got %d", len(listener.Profiles))
		}

		expected := &sp.ServiceProfile{
			Spec: sp.ServiceProfileSpec{
				Routes: []*sp.RouteSpec{
					{
						Name: "route",
					},
				},
				DstOverrides: []*sp.WeightedDst{
					{
						Authority: "bar.ns.svc.cluster.local:80",
						Weight:    resource.MustParse("1000m"),
					},
				},
			},
		}

		testCompare(t, expected.Spec, listener.Profiles[1].Spec)
	})
}

func testCompare(t *testing.T, expected interface{}, actual interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		expectedBytes, _ := json.Marshal(expected)
		actualBytes, _ := json.Marshal(actual)
		t.Fatalf("Expected %s but got %s", string(expectedBytes), string(actualBytes))
	}
}
