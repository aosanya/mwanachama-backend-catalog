package mwanachamacatalog

import "context"

func (m *catalogManager) sendsReplies() bool {
	_, replied := m.st.Spec().ByRole(roleReply)
	return replied && m.takesRaisedItems()
}

func (m *catalogManager) ReplyToRaisedItem(ctx context.Context, r Reply) (Reply, error) {
	if !m.sendsReplies() {
		return Reply{}, ErrRepliesNotSent
	}
	if err := m.checks(roleReply, r); err != nil {
		return Reply{}, ErrInvalidReply
	}
	var raised RaisedItem
	if err := m.take(m.q(ctx, roleRaisedItem).Where("id = ?", r.RaisedItemID), roleRaisedItem, &raised, ErrRaisedItemNotFound); err != nil {
		return Reply{}, err
	}
	r.ID = newID()
	r.CreatedAt = m.now()
	row, err := encode(m.st.Object(roleReply), r)
	if err != nil {
		return Reply{}, err
	}
	if err := m.q(ctx, roleReply).Create(row).Error; err != nil {
		return Reply{}, err
	}
	return r, nil
}

func (m *catalogManager) ListReplies(ctx context.Context, raisedItemID string) ([]Reply, error) {
	if !m.sendsReplies() {
		return nil, ErrRepliesNotSent
	}
	if raisedItemID == "" {
		return nil, ErrInvalidReply
	}
	return listOf[Reply](m, m.q(ctx, roleReply).Where("raised_item_id = ?", raisedItemID).Order("created_at, id"), roleReply)
}
