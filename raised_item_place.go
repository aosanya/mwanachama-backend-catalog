package mwanachamacatalog

import (
	"context"
	"sort"

	"github.com/aosanya/mwanachama-backend-shared/geo"
)

type AreaResolver func(ctx context.Context, pin *geo.Position, address string) (string, error)

const nearbyRaisedItemsCap = 200

func (m *catalogManager) UseAreaResolver(resolve AreaResolver) {
	m.resolveArea = resolve
}

func (m *catalogManager) placeOf(ctx context.Context, r RaisedItem) (string, error) {
	if m.resolveArea == nil {
		return r.Area, nil
	}
	if r.Pin != nil {
		at := r.Pin.Point
		area, err := m.resolveArea(ctx, &at, "")
		if err != nil || area != "" {
			return area, err
		}
	}
	if r.Address != "" {
		area, err := m.resolveArea(ctx, nil, r.Address)
		if err != nil || area != "" {
			return area, err
		}
	}
	return r.Area, nil
}

func (m *catalogManager) NearbyRaisedItems(ctx context.Context, topic string, at geo.Position, withinMetres float64) ([]RaisedItem, error) {
	if !m.takesRaisedItems() {
		return nil, ErrRaisedItemsNotTaken
	}
	if topic == "" || !(withinMetres > 0) {
		return nil, ErrInvalidRaisedItem
	}
	candidates, err := listOf[RaisedItem](m, m.q(ctx, roleRaisedItem).Where("topic = ? AND publishable = ?", topic, true), roleRaisedItem)
	if err != nil {
		return nil, err
	}
	type near struct {
		item     RaisedItem
		distance float64
	}
	var found []near
	for _, c := range candidates {
		if c.Pin == nil || c.Pin.Kind != geo.KindPoint {
			continue
		}
		if d := geo.Distance(at, c.Pin.Point); d <= withinMetres {
			found = append(found, near{c.AsSeenByOthers(), d})
		}
	}
	sort.SliceStable(found, func(i, j int) bool { return found[i].distance < found[j].distance })
	if len(found) > nearbyRaisedItemsCap {
		found = found[:nearbyRaisedItemsCap]
	}
	out := make([]RaisedItem, len(found))
	for i, n := range found {
		out[i] = n.item
	}
	return out, nil
}
