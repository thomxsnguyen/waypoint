package domain

import "testing"

func validTrip() TripInput {
	return TripInput{Name: "Tokyo", Destination: "Tokyo, Japan", StartDate: "2026-11-01", EndDate: "2026-11-03", Timezone: "Asia/Tokyo"}
}
func TestTripValidation(t *testing.T) {
	for _, test := range []struct {
		name  string
		edit  func(*TripInput)
		valid bool
	}{
		{"valid", func(t *TripInput) {}, true},
		{"same day", func(t *TripInput) { t.EndDate = t.StartDate }, true},
		{"DST range", func(t *TripInput) { t.Timezone = "America/Los_Angeles" }, true},
		{"leap day", func(t *TripInput) { t.StartDate = "2028-02-29"; t.EndDate = "2028-02-29" }, true},
		{"impossible date", func(t *TripInput) { t.StartDate = "2026-02-29" }, false},
		{"year zero", func(t *TripInput) { t.StartDate = "0000-01-01"; t.EndDate = "0000-01-02" }, false},
		{"backwards", func(t *TripInput) { t.EndDate = "2026-10-01" }, false},
		{"too long", func(t *TripInput) { t.EndDate = "2027-11-02" }, false},
		{"timezone", func(t *TripInput) { t.Timezone = "Mars/Olympus" }, false},
		{"server local", func(t *TripInput) { t.Timezone = "Local" }, false},
		{"empty name", func(t *TripInput) { t.Name = "   " }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			in := validTrip()
			test.edit(&in)
			if err := in.Validate(); (err == nil) != test.valid {
				t.Fatalf("got %v", err)
			}
		})
	}
}
func TestActivityValidation(t *testing.T) {
	for _, tc := range []struct {
		clock    string
		duration int
		valid    bool
	}{{"09:30", 60, true}, {"23:59", 1440, true}, {"24:00", 60, false}, {"9:30", 60, false}, {"09:30:00", 60, false}, {"12:00", 0, false}, {"12:00", -2, false}, {"12:00", 1441, false}} {
		in := ActivityInput{StartTime: &tc.clock, DurationMinutes: tc.duration}
		if err := in.Validate(); (err == nil) != tc.valid {
			t.Errorf("%+v: %v", tc, err)
		}
	}
	if err := (ActivityInput{DurationMinutes: 1}).Validate(); err != nil {
		t.Fatal(err)
	}
}
func TestExactOrder(t *testing.T) {
	for _, tc := range []struct {
		order []string
		valid bool
	}{{[]string{"c", "a", "b"}, true}, {[]string{"a", "b"}, false}, {[]string{"a", "b", "b"}, false}, {[]string{"a", "b", "foreign"}, false}} {
		if err := ValidateOrder([]string{"a", "b", "c"}, tc.order); (err == nil) != tc.valid {
			t.Errorf("%v: %v", tc, err)
		}
	}
}
