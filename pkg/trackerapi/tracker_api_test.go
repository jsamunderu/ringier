package trackerapi

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"ringier/pkg/statsdb"
	"sync"
	"testing"
)

// TestTrackerApi_parseFields checks if a line of a test coverage
// can be parsed correctly
func TestTrackerApi_parseFields(t *testing.T) {
	testCases := []struct {
		line []byte
		want *struct {
			action   string
			coverage float64
		}
	}{
		{
			line: []byte("ok\tringier/pkg/statsdb\t(cached)\tcoverage: 63.3% of statements"),
			want: &struct {
				action   string
				coverage float64
			}{
				action:   "ringier/pkg/statsdb",
				coverage: 63.6,
			},
		},
	}

	for _, tc := range testCases {
		got := parseFields(tc.line)
		if got == nil || got.action != tc.want.action && got.coverage != tc.want.coverage {
			t.Errorf("parseFields(%q): want: %v, got: %v", string(tc.line), tc.want, got)
		}
	}
}

/*
Launching go test cmd in go test freezes
func TestTrackerApi_getTestAction(t *testing.T) {
	action := getTestAction()
	if action == nil {
		t.Errorf("getTestAction(): want: %v, got: %v", false, action == nil)
	}
}
*/

// TestTrackerApi_runTestCmd test if this functionality completes
// This is to test only for completion
func TestTrackerApi_runTestCmd(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, ``)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	tracker := &Tracker{Wg: sync.WaitGroup{}}
	defer tracker.Wg.Wait()
	tracker.Queue = tracker.EventSink()
	defer close(tracker.Queue)
	tracker.DestEndpoint = server.URL
	tracker.sendEvent(&statsdb.GitHubAction{})
}

// TestTrackerApi_StatsAPI checks if the api endpoint
// returns a success http status
func TestTrackerApi_StatsAPI(t *testing.T) {
	tracker := &Tracker{Wg: sync.WaitGroup{}}
	defer tracker.Wg.Wait()
	tracker.DB = statsdb.Open("./test.db")
	if tracker.DB == nil {
		return
	}
	err := tracker.DB.Setup()
	if err != nil {
		t.Errorf("Error setting up database: %v", err)
		return
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/stats/api", nil)
	tracker.StatsAPI(w, r)
	resp := w.Result()
	if resp.Status != fmt.Sprintf("%d OK", http.StatusOK) {
		t.Errorf("trackerapi.StatsAPI(w http.ResponseWriter, r *http.Request): want: %v, got: %v", http.StatusOK, resp.Status)
	}
}

var tmplStr string = `<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN"                            
"http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd">                                
<html xmlns="http://www.w3.org/1999/xhtml">                                         
  <head>                                                                            
    <title>Stats</title>                                                            
    <meta http-equiv="Content-Type"                                                 
      content="text/html; charset=utf-8"/>                                          
    <link href="style.css" rel="stylesheet" type="text/css"/>                       
  </head>                                                                           
  <body>                                                                            
    <table summary="Test Statistics">                                               
      <caption>Test Statistics</caption>                                            
      <tr>                                                                          
        <th>Event</th>                                                              
        <th>VentureConfigId</th>                                                    
        <th>VentureReference</th>                                                   
        <th>CreatedAt</th>                                                          
        <th>Culture</th>                                                            
        <th>ActionType</th>                                                         
        <th>ActionReference</th>                                                    
        <th>Version</th>                                                            
        <th>Route</th>                                                              
        <th>Payload</th>                                                            
      </tr>                                                                         
      {{range .}}<tr>                                                               
      {{range rangeStruct .}}<td>{{.}}</td>                                         
      {{end}}</tr>                                                                  
      {{end}}                                                                       
    </table>                                                                        
  </body>                                                                           
</html> `

// TestTrackerApi_StatsWeb checks if the web endpoint
// returns a success http status
func TestTrackerApi_StatsWeb(t *testing.T) {
	tracker := &Tracker{Wg: sync.WaitGroup{}}
	defer tracker.Wg.Wait()
	tracker.DB = statsdb.Open("./test.db")
	if tracker.DB == nil {
		return
	}
	err := tracker.DB.Setup()
	if err != nil {
		t.Errorf("Error setting up database: %v", err)
		return
	}
	templ := template.New("test.tmpl").Funcs(TemplateFuncs)
	tracker.HTMLTemplateName = "test.tmpl"
	tracker.HTMLTemplate, err = templ.Parse(tmplStr)
	if err != nil {
		t.Error(err, "Error parsing the web template")
		return
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/stats", nil)
	tracker.StatsWeb(w, r)
	resp := w.Result()
	if resp.Status != fmt.Sprintf("%d OK", http.StatusOK) {
		t.Errorf("trackerapi.StatsWeb(w http.ResponseWriter, r *http.Request): want: %v, got: %v", http.StatusOK, resp.Status)
	}
}

var githubAction string = `{
	"event": "TrackTestCoverageEvent",
	"venture_config_id": "57EFFB23-1731-4348-B306-9F3819D12FEB",
	"venture_reference": "C1C9025B-AEE0-4943-886E-466301F02BED",
	"created_at": "2021-03-02T08:30:00+00:00",
	"culture": "en_EN",
	"action_type": "api",
	"action_reference": "",
	"version": "1.0.0",
	"route": "",
	"payload": {
			"service_name": "test",
			"coverage": 23.5
	}
}`

func TestTrackerApi_getTestActions(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		fmt.Fprint(w, ``)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	tracker := &Tracker{Wg: sync.WaitGroup{}}
	defer tracker.Wg.Wait()
	tracker.DB = statsdb.Open("./test.db")
	if tracker.DB == nil {
		return
	}
	err := tracker.DB.Setup()
	if err != nil {
		t.Errorf("Error setting up database: %v", err)
		return
	}
	tracker.Queue = tracker.EventSink()
	defer close(tracker.Queue)
	tracker.DestEndpoint = server.URL

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/action", bytes.NewReader([]byte(githubAction)))
	tracker.Action(w, r)
	resp := w.Result()
	if resp.Status != fmt.Sprintf("%d OK", http.StatusOK) {
		t.Errorf("trackerapi.Action(w http.ResponseWriter, r *http.Request): want: %v, got: %v", http.StatusOK, resp.Status)
	}
}
