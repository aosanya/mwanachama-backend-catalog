package mwanachamacatalog

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aosanya/mwanachama-backend-catalog/models"
	"github.com/aosanya/mwanachama-backend-shared/spec"
	"strconv"
	"strings"
	"time"
)

const cataloguesSetting = "catalogues"

var catalogueAudiences = map[Audience]bool{AudiencePublic: true, AudienceMembers: true, AudienceStaff: true}

type Catalogue struct {
	Mount    string   `json:"mount"`
	Name     string   `json:"name"`
	Audience Audience `json:"audience"`
	Services bool     `json:"services,omitempty"`
}

func CheckSettings(settings json.RawMessage) []string {
	trimmed := bytes.TrimSpace(settings)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil
	}
	var values map[string]any
	if err := json.Unmarshal(trimmed, &values); err != nil {
		return []string{"settings are not a JSON object"}
	}
	blueprint, err := Blueprint()
	if err != nil {
		return []string{err.Error()}
	}
	if problems := spec.CheckSettings(blueprint.Settings, values); len(problems) > 0 {
		return problems
	}
	if raw, present := values[routingSetting]; present {
		if _, err := models.ParseRouting(raw); err != nil {
			return []string{routingSetting + ": a default department and rules, each a topic and a department, no topic twice"}
		}
	}
	raw, present := values[cataloguesSetting]
	if !present {
		return nil
	}
	_, problems := ParseCatalogues(raw)
	return problems
}
func ParseCatalogues(value any) ([]Catalogue, []string) {
	raw, err := json.Marshal(value)
	if err != nil || string(raw) == "null" {
		return nil, nil
	}
	var catalogues []Catalogue
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&catalogues); err != nil {
		return nil, []string{fmt.Sprintf("%s: each catalogue is an object with a mount, a name, an audience and services", cataloguesSetting)}
	}
	var problems []string
	seen := map[string]bool{}
	for i, c := range catalogues {
		at := fmt.Sprintf("%s[%d]", cataloguesSetting, i)
		if !spec.SegmentPattern.MatchString(c.Mount) {
			problems = append(problems, fmt.Sprintf("%s: mount %q must be lower-case letters and digits, starting with a letter", at, c.Mount))
		} else if seen[c.Mount] {
			problems = append(problems, fmt.Sprintf("%s: mount %q is listed twice", at, c.Mount))
		}
		seen[c.Mount] = true
		if strings.TrimSpace(c.Name) == "" {
			problems = append(problems, fmt.Sprintf("%s: a name is required", at))
		}
		if !catalogueAudiences[c.Audience] {
			problems = append(problems, fmt.Sprintf("%s: audience %q must be public, members or staff", at, c.Audience))
		}
	}
	if len(problems) > 0 {
		return nil, problems
	}
	return catalogues, nil
}

const (
	SenderPeriodSetting     = "raised_item.sender_period_days"
	DefaultSenderPeriodDays = 30
	secondsPerDay           = 86400
)

func (m *catalogManager) UseSenderSecret(secret []byte) {
	m.senderSecret = secret
}
func (m *catalogManager) UseClock(now func() string) {
	m.now = now
}
func (m *catalogManager) limitsSenders() bool {
	return len(m.senderSecret) > 0
}
func (m *catalogManager) senderPeriodDays(ctx context.Context) (int, error) {
	s, err := m.GetSetting(ctx, SenderPeriodSetting)
	if errors.Is(err, ErrSettingNotFound) {
		return DefaultSenderPeriodDays, nil
	}
	if err != nil {
		return 0, err
	}
	days, err := strconv.Atoi(s.Value)
	if err != nil || days < 1 {
		return 0, ErrInvalidSetting
	}
	return days, nil
}
func (m *catalogManager) senderPeriod(ctx context.Context) (string, error) {
	days, err := m.senderPeriodDays(ctx)
	if err != nil {
		return "", err
	}
	at, err := time.Parse(time.RFC3339, m.now())
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d:%d", days, at.Unix()/secondsPerDay/int64(days)), nil
}
func (m *catalogManager) senderHash(person, period string) string {
	mac := hmac.New(sha256.New, m.senderSecret)
	mac.Write([]byte(person + "\x00" + period))
	return hex.EncodeToString(mac.Sum(nil))
}
func (m *catalogManager) stampSender(ctx context.Context, r RaisedItem) (RaisedItem, error) {
	r.SenderHash, r.SenderPeriod = "", ""
	if !m.limitsSenders() {
		return r, nil
	}
	period, err := m.senderPeriod(ctx)
	if err != nil {
		return RaisedItem{}, err
	}
	r.SenderPeriod = period
	r.SenderHash = m.senderHash(r.RaisedBy, period)
	var sent int64
	if err := m.q(ctx, roleRaisedItem).Where("sender_hash = ?", r.SenderHash).Count(&sent).Error; err != nil {
		return RaisedItem{}, err
	}
	if sent > 0 {
		return RaisedItem{}, ErrWishAlreadySentThisPeriod
	}
	return r, nil
}
