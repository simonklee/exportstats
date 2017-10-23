package exportstats

import (
	"fmt"
	"os"
	"testing"
)

func getAccessToken(t *testing.T) string {
	token := os.Getenv("STATHAT_ACCESSTOKEN")

	if token == "" {
		t.Skip("Missing STATHAT_ACCESSTOKEN")
	}

	return token
}

func TestParseTimeframe(t *testing.T) {
	for i, test := range []struct {
		value string
		c     bool
	}{
		{"1 hour @ 1 minute", true},
		{"1 hour @ 1 minut", false},
		{"1 week @ 1 minute", true},
		{"1 week@ 1 minute", false},
		{"13w13m", true},
		{"1w1m", true},
	} {
		tf, err := ParseTimeframe(test.value)

		if err != nil {
			if test.c {
				t.Errorf("[%d] %s failed: %v", i, test.value, err)
			}
		} else if !test.c {
			t.Errorf("[%d] %s should have failed", i, test.value)
		}

		t.Log(tf)
	}
}

func TestStathatFetcher(t *testing.T) {
	f := NewStatHatFetcher(getAccessToken(t))
	ds, err := f.Get("eu.website.load.domready.cnt", MustParseTimeframe("1 hour @ 1 minute"))

	if err != nil {
		t.Errorf("unexpected err %s", err)
	}

	fmt.Println(ds.Name)
}

func TestDB(t *testing.T) {
	db := NewDB(NewStatHatFetcher(getAccessToken(t)))
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
