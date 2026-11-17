package models

import (
	"errors"
	"testing"
)

func TestServiceTermsOfDefaultsOpensToFormAndKeepsTheFee(t *testing.T) {
	terms, err := ServiceTermsOf(`{"form":"intake","fee":"500","appointments":true}`)
	if err != nil {
		t.Fatal(err)
	}
	if terms.Opens != ServiceOpensForm || terms.Form != "intake" || terms.Fee != "500" || !terms.Appointments {
		t.Fatalf("terms = %+v", terms)
	}
}

func TestServiceTermsOfRefusesEveryInvalidShape(t *testing.T) {
	cases := map[string]string{
		"blank":                     "",
		"whitespace":                "   \n",
		"not json":                  "{form",
		"json array":                `[]`,
		"form opening without form": `{"opens":"form"}`,
		"default opening no form":   `{}`,
		"blank form":                `{"opens":"form","form":"  "}`,
		"raise with a form":         `{"opens":"raise","form":"intake"}`,
		"raise with appointments":   `{"opens":"raise","appointments":true}`,
		"unknown opening":           `{"opens":"call","form":"intake"}`,
	}
	for name, doc := range cases {
		if _, err := ServiceTermsOf(doc); !errors.Is(err, ErrInvalidServiceTerms) {
			t.Fatalf("%s: err = %v", name, err)
		}
	}
}

func TestServiceTermsOfAcceptsARaiseOpeningWithOnlyAFee(t *testing.T) {
	terms, err := ServiceTermsOf(`{"opens":"raise","fee":"free"}`)
	if err != nil || terms.Opens != ServiceOpensRaise || terms.Fee != "free" {
		t.Fatalf("terms = %+v, err = %v", terms, err)
	}
}

func TestServiceTermsDocRoundTripsAndFillsTheDefaultOpening(t *testing.T) {
	doc, err := (ServiceTerms{Form: "intake"}).Doc()
	if err != nil {
		t.Fatal(err)
	}
	if doc != `{"form":"intake","opens":"form"}` {
		t.Fatalf("doc = %s", doc)
	}
	back, err := ServiceTermsOf(doc)
	if err != nil || back.Form != "intake" || back.Opens != ServiceOpensForm {
		t.Fatalf("back = %+v, err = %v", back, err)
	}
}

func TestServiceTermsDocRefusesTermsThatFailTheCheck(t *testing.T) {
	for _, terms := range []ServiceTerms{{}, {Opens: ServiceOpensRaise, Form: "x"}, {Opens: "other"}} {
		if doc, err := terms.Doc(); !errors.Is(err, ErrInvalidServiceTerms) || doc != "" {
			t.Fatalf("%+v = %q, %v", terms, doc, err)
		}
	}
}
