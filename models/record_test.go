package models

import "testing"

func TestTallyRecordsCountsEachValueShape(t *testing.T) {
	records := []Record{
		{Values: `{"agree": true, "colour": "red", "picks": ["a", "b"], "amount": 10, "skipped": null}`},
		{Values: `{"agree": false, "colour": "red", "picks": ["b"], "amount": 30}`},
		{Values: `{"agree": true, "colour": "blue", "amount": 20}`},
	}
	tallies, err := TallyRecords(records)
	if err != nil {
		t.Fatal(err)
	}
	byField := map[string]FieldTally{}
	for _, tally := range tallies {
		byField[tally.Field] = tally
	}
	if a := byField["agree"]; a.Yes != 2 || a.No != 1 || a.Counted != 3 {
		t.Fatalf("agree = %+v", a)
	}
	if c := byField["colour"]; c.Options["red"] != 2 || c.Options["blue"] != 1 {
		t.Fatalf("colour = %+v", c)
	}
	if p := byField["picks"]; p.Options["b"] != 2 || p.Options["a"] != 1 || p.Counted != 2 {
		t.Fatalf("picks = %+v", p)
	}
	n := byField["amount"]
	if n.Numbers != 3 || *n.Average != 20 || *n.Lowest != 10 || *n.Highest != 30 {
		t.Fatalf("amount = %+v", n)
	}
	if s := byField["skipped"]; s.Counted != 0 {
		t.Fatalf("skipped = %+v", s)
	}
	if tallies[0].Field != "agree" {
		t.Fatalf("order = %v", tallies)
	}
}

func TestTallyRecordsRefusesAValuesDocumentThatIsNotJSON(t *testing.T) {
	if _, err := TallyRecords([]Record{{Values: "nope"}}); err == nil {
		t.Fatal("want an error")
	}
}
