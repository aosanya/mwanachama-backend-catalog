package mwanachamacatalog

import "context"

func (m *catalogManager) keepsBoard() bool {
	_, boarded := m.st.Spec().ByRole(roleBoardItem)
	return boarded && m.takesRaisedItems()
}

func (m *catalogManager) PutOnBoard(ctx context.Context, b BoardItem) (BoardItem, error) {
	if !m.keepsBoard() {
		return BoardItem{}, ErrBoardNotKept
	}
	if err := m.checks(roleBoardItem, b); err != nil {
		return BoardItem{}, ErrInvalidBoardItem
	}
	b.ID = newID()
	b.CreatedAt = m.now()
	row, err := encode(m.st.Object(roleBoardItem), b)
	if err != nil {
		return BoardItem{}, err
	}
	if err := m.q(ctx, roleBoardItem).Create(row).Error; err != nil {
		return BoardItem{}, err
	}
	return b, nil
}

func (m *catalogManager) GroupUnderBoardItem(ctx context.Context, raisedItemID, boardItemID string) error {
	if !m.keepsBoard() {
		return ErrBoardNotKept
	}
	if raisedItemID == "" || boardItemID == "" {
		return ErrInvalidBoardItem
	}
	var b BoardItem
	if err := m.take(m.q(ctx, roleBoardItem).Where("id = ?", boardItemID), roleBoardItem, &b, ErrBoardItemNotFound); err != nil {
		return err
	}
	var r RaisedItem
	if err := m.take(m.q(ctx, roleRaisedItem).Where("id = ?", raisedItemID), roleRaisedItem, &r, ErrRaisedItemNotFound); err != nil {
		return err
	}
	if !r.Publishable {
		return ErrRaisedItemNotPublishable
	}
	return m.q(ctx, roleRaisedItem).Where("id = ?", raisedItemID).Update("board_item_id", boardItemID).Error
}

func (m *catalogManager) ListBoardItems(ctx context.Context) ([]BoardItemTally, error) {
	if !m.keepsBoard() {
		return nil, ErrBoardNotKept
	}
	items, err := listOf[BoardItem](m, m.q(ctx, roleBoardItem).Order("created_at, id"), roleBoardItem)
	if err != nil {
		return nil, err
	}
	out := make([]BoardItemTally, 0, len(items))
	for _, b := range items {
		t := BoardItemTally{BoardItem: b}
		var n int64
		if err := m.q(ctx, roleRaisedItem).Where("board_item_id = ? AND publishable = ?", b.ID, true).Count(&n).Error; err != nil {
			return nil, err
		}
		t.RaisedBy = int(n)
		if m.countsEndorsements() {
			if t.MeToo, err = m.CountEndorsements(ctx, b.ID); err != nil {
				return nil, err
			}
		}
		out = append(out, t)
	}
	return out, nil
}
