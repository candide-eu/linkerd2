package destination

import (
	"fmt"

	ts "github.com/deislabs/smi-sdk-go/pkg/apis/split/v1alpha1"
	"github.com/linkerd/linkerd2/controller/api/destination/watcher"
	sp "github.com/linkerd/linkerd2/controller/gen/apis/serviceprofile/v1alpha1"
)

// trafficSplitAdaptor merges traffic splits into service profiles, encoding
// them as dst overrides.  trafficSplitAdaptor holds an underlying
// ProfileUpdateListener and updates that listener with a merged service
// service profile which includes the traffic split logic as a dst override
// when a traffic split exists.  trafficSplitAdaptor itself implements both
// ProfileUpdateListener and TrafficSplitUpdateListener and must be passed to
// a source of profile updates (such as a ProfileWatcher) and a source of
// traffic split updates (such as a TrafficSplitWatcher).
type trafficSplitAdaptor struct {
	listener watcher.ProfileUpdateListener
	id       watcher.ServiceID
	port     watcher.Port
	profile  *sp.ServiceProfile
	split    *ts.TrafficSplit
}

func newTrafficSplitAdaptor(listener watcher.ProfileUpdateListener, id watcher.ServiceID, port watcher.Port) *trafficSplitAdaptor {
	return &trafficSplitAdaptor{
		listener: listener,
		id:       id,
		port:     port,
	}
}

func (tsa *trafficSplitAdaptor) Update(profile *sp.ServiceProfile) {
	tsa.profile = profile
	tsa.publish()
}

func (tsa *trafficSplitAdaptor) UpdateTrafficSplit(split *ts.TrafficSplit) {
	if tsa.split == nil && split == nil {
		return
	}
	tsa.split = split
	tsa.publish()
}

func (tsa *trafficSplitAdaptor) publish() {
	merged := sp.ServiceProfile{}
	if tsa.profile != nil {
		merged = *tsa.profile
	}
	if tsa.split != nil {
		overrides := []*sp.WeightedDst{}
		for _, backend := range tsa.split.Spec.Backends {
			dst := &sp.WeightedDst{
				Authority: fmt.Sprintf("%s.%s.svc.cluster.local:%d", backend.Service, tsa.id.Namespace, tsa.port),
				Weight:    backend.Weight,
			}
			overrides = append(overrides, dst)
		}
		merged.Spec.DstOverrides = overrides
	}

	if tsa.profile == nil && tsa.split == nil {
		tsa.listener.Update(nil)
	} else {
		tsa.listener.Update(&merged)
	}
}
