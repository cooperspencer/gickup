package types

import "testing"

func TestConfCronMissing(t *testing.T) {
	t.Parallel()

	conf := Conf{Cron: ""}

	if !conf.MissingCronSpec() {
		t.Error("Conf passed empty cron spec doesn't think that it's missing")
	}
}

func TestParseValidCron(t *testing.T) {
	t.Parallel()

	conf := Conf{Cron: "0 0 * * *"}

	if !conf.HasValidCronSpec() {
		t.Error("Valid cron spec parsed invalidly")
	}
}

func TestParseInvalidCron(t *testing.T) {
	t.Parallel()

	conf := Conf{Cron: "redshirt"}

	if conf.HasValidCronSpec() {
		t.Error("Invalid cron spec parsed validly")
	}
}
