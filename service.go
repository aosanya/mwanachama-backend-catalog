package mwanachamacatalog

import "github.com/aosanya/mwanachama-backend-catalog/models"

type (
	ServiceTerms   = models.ServiceTerms
	ServiceOpening = models.ServiceOpening
)

const (
	ServiceOpensForm  = models.ServiceOpensForm
	ServiceOpensRaise = models.ServiceOpensRaise
)

var ErrInvalidServiceTerms = models.ErrInvalidServiceTerms

func ServiceTermsOf(e Entry) (ServiceTerms, error) {
	return models.ServiceTermsOf(e.Doc)
}

func ServiceEntry(slug, name, summary string, t ServiceTerms) (Entry, error) {
	doc, err := t.Doc()
	if err != nil {
		return Entry{}, err
	}
	return Entry{Slug: slug, Name: name, Summary: summary, Doc: doc}, nil
}
