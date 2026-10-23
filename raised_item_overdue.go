package mwanachamacatalog

import (
	"context"
	"strconv"
	"time"
)

const AnswerClockSetting = "raised_item.answer_within_days"

func (m *catalogManager) answerClock(ctx context.Context) (time.Duration, error) {
	s, err := m.GetSetting(ctx, AnswerClockSetting)
	if err != nil {
		return 0, err
	}
	days, err := strconv.Atoi(s.Value)
	if err != nil || days < 1 {
		return 0, ErrInvalidSetting
	}
	return time.Duration(days) * 24 * time.Hour, nil
}

func (m *catalogManager) ListOverdueRaisedItems(ctx context.Context, at string) ([]RaisedItem, error) {
	if !m.takesRaisedItems() {
		return nil, ErrRaisedItemsNotTaken
	}
	if at == "" {
		at = m.now()
	}
	when, err := time.Parse(time.RFC3339, at)
	if err != nil {
		return nil, ErrInvalidRaisedItem
	}
	clock, err := m.answerClock(ctx)
	if err != nil {
		return nil, err
	}
	cutoff := when.UTC().Add(-clock).Format(time.RFC3339)
	q := m.q(ctx, roleRaisedItem).Where("created_at <= ?", cutoff)
	if m.sendsReplies() {
		q = q.Where("id NOT IN (?)", m.q(ctx, roleReply).Select("raised_item_id"))
	}
	if m.makesRemovals() {
		q = q.Where("id NOT IN (?)", m.q(ctx, roleRemoval).Select("raised_item_id"))
	}
	return listOf[RaisedItem](m, q.Order("created_at, id"), roleRaisedItem)
}
