package statsdb

import (
	"os"
	"testing"
)

// TestStatsDB_Open checks if opening the database works
func TestStatsDB_Open(t *testing.T) {
	got := Open("./test.db")
	if got == nil {
		t.Errorf("Open(\"./test.db\"): want: %v, got: %v", true, got != nil)
	}
}

// TestStatsDB_Open checks if setting up the action table works
func TestStatsDB_Setup(t *testing.T) {
	//os.Remove("./test.db")
	stats := Open("./test.db")
	err := stats.Setup()
	if err != nil {
		t.Errorf("StatsDB.Setup(): want: %v, got: %v", nil, err)
	}
}

// TestStatsDB_Open checks if saving an action works
func TestStatsDB_Save(t *testing.T) {
	os.Remove("./test.db")
	stats := Open("./test.db")
	err := stats.Setup()
	if err != nil {
		t.Errorf("StatsDB.StatsDB(): Failed to setup database")
		return
	}

	testCases := []struct {
		action *GitHubAction
		want   bool // is nil
	}{
		{
			action: &GitHubAction{
				Event:            "TrackTestCoverageEvent",
				VentureConfigId:  "1",
				VentureReference: "1",
				CreatedAt:        "2021-03-02T08:30:00+00:00",
				Culture:          "en_EN",
				ActionType:       "api",
				ActionReference:  "",
				Version:          "1.0.0",
				Route:            "",
				Payload: &Payload{
					ServiceName: "test",
					Coverage:    23.5,
				},
			},
			want: true,
		},
		{
			action: &GitHubAction{
				Event:            "TrackTestCoverageEvent",
				VentureConfigId:  "2",
				VentureReference: "2",
				CreatedAt:        "2021-03-02T08:30:00+00:00",
				Culture:          "en_EN",
				ActionType:       "api",
				ActionReference:  "",
				Version:          "1.0.0",
				Route:            "",
				Payload: &Payload{
					ServiceName: "test",
					Coverage:    23.5,
				},
			},
			want: true,
		},
	}

	for _, tc := range testCases {
		got := stats.Save(tc.action)
		if (got == nil) != tc.want {
			t.Errorf("StatsDB.Save() - %q: want: %v, got: %v", tc.action.ActionReference, tc.want, (got == nil))
		}
	}
}

// TestStatsDB_Open checks if retrieving actions works
func TestStatsDB_GetAllActions(t *testing.T) {
	os.Remove("./test.db")
	stats := Open("./test.db")
	err := stats.Setup()
	if err != nil {
		t.Errorf("StatsDB.StatsDB(): Failed to setup database")
		return
	}

	testCases := []struct {
		action *GitHubAction
		want   bool // is nil
	}{
		{
			action: &GitHubAction{
				Event:            "TrackTestCoverageEvent",
				VentureConfigId:  "1",
				VentureReference: "1",
				CreatedAt:        "2021-03-02T08:30:00+00:00",
				Culture:          "en_EN",
				ActionType:       "api",
				ActionReference:  "",
				Version:          "1.0.0",
				Route:            "",
				Payload: &Payload{
					ServiceName: "test",
					Coverage:    23.5,
				},
			},
			want: true,
		},
		{
			action: &GitHubAction{
				Event:            "TrackTestCoverageEvent",
				VentureConfigId:  "2",
				VentureReference: "2",
				CreatedAt:        "2021-03-02T08:30:00+00:00",
				Culture:          "en_EN",
				ActionType:       "api",
				ActionReference:  "",
				Version:          "1.0.0",
				Route:            "",
				Payload: &Payload{
					ServiceName: "test",
					Coverage:    23.5,
				},
			},
			want: true,
		},
	}

	for _, tc := range testCases {
		got := stats.Save(tc.action)
		if (got == nil) != tc.want {
			t.Errorf("StatsDB.Save() - %q: want: %v, got: %v", tc.action.ActionReference, tc.want, (got == nil))
		}
	}

	all := stats.GetAllActions()
	if all == nil {
		t.Errorf("StatsDB.GetAllActions(): want: %v, got: %v", nil, err)
		return
	}
	if len(all) != 2 {
		t.Errorf("StatsDB.GetAllActions(): want: %v, got: %v", 2, len(all))
		return
	}
}
