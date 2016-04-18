package exportstats

import (
	"fmt"
	"testing"
)

func TestParseTimeframe(t *testing.T) {
	for i, test := range []struct {
		value string
		c     bool
	}{
		{"1 hour @ 1 minute", true},
		{"1 hour @ 1 minut", false},
		{"1 week @ 1 minute", true},
		{"1 week@ 1 minute", false},
	} {
		tf, err := ParseTimeframe(test.value)

		if err != nil {
			if test.c {
				t.Errorf("[%d] %s failed", i, test.value)
			}
		} else if !test.c {
			t.Errorf("[%d] %s should have failed", i, test.value)
		}

		t.Log(tf)
	}
}

func TestStathatFetcher(t *testing.T) {
	f := NewStatHatFetcher("OcYTpK5JyVmdyT30y2NX")
	ds, err := f.Get("eu.website.load.domready.cnt", MustParseTimeframe("1 hour @ 1 minute"))

	if err != nil {
		t.Errorf("unexpected err %s", err)
	}

	fmt.Println(ds.Name)
}

func TestDB(t *testing.T) {
	db := NewDB(NewStatHatFetcher("OcYTpK5JyVmdyT30y2NX"))
	ds, err := db.Get("eu.website.load.domready.cnt", MustParseTimeframe("1 hour @ 1 minute"))

	if err != nil {
		t.Errorf("unexpected err %s", err)
	}

	fmt.Println(ds.Name)

	ds, err = db.Get("eu.website.load.domready.cnt", MustParseTimeframe("1 hour @ 1 minute"))

	if err != nil {
		t.Errorf("unexpected err %s", err)
	}

	fmt.Println(ds.Name)
}
