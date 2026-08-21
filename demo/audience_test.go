package demo

import "testing"

func TestDemoAudienceHasMissingDemographics(t *testing.T) {
	var nilAudience *DemoAudience
	if !nilAudience.Has(nil) {
		t.Fatal("nil audience should remain unrestricted")
	}
	if !(&DemoAudience{}).Has(nil) {
		t.Fatal("empty audience should match missing demographics")
	}

	tests := []struct {
		name     string
		audience *DemoAudience
	}{
		{name: "gender", audience: &DemoAudience{Genders: 1 << GENDERM}},
		{name: "age", audience: &DemoAudience{Yobs: 1 << YOB1990}},
		{name: "language", audience: &DemoAudience{Languages: 1 << LanguageEN}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.audience.Has(nil) {
				t.Fatal("constrained audience matched missing demographics")
			}
		})
	}
}

func TestDemoAudienceHasPopulatedDemographics(t *testing.T) {
	audience := &DemoAudience{
		Genders:   1 << GENDERM,
		Yobs:      1 << YOB1990,
		Languages: 1 << LanguageEN,
	}
	matching := &Demo{Gender: GENDERM, Yob: YOB1990, Language: 1 << LanguageEN}
	if !audience.Has(matching) {
		t.Fatal("matching populated demographics were rejected")
	}

	tests := []struct {
		name string
		demo *Demo
	}{
		{name: "gender", demo: &Demo{Gender: GENDERF, Yob: YOB1990, Language: 1 << LanguageEN}},
		{name: "age", demo: &Demo{Gender: GENDERM, Yob: YOB1986, Language: 1 << LanguageEN}},
		{name: "language", demo: &Demo{Gender: GENDERM, Yob: YOB1990, Language: 1 << LanguageES}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if audience.Has(test.demo) {
				t.Fatal("mismatching populated demographics were accepted")
			}
		})
	}
}
