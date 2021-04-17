package trackerapi

import (
	"bufio"
	"bytes"
	"encoding/json"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"reflect"
	"ringier/pkg/statsdb"
	"strconv"
	"strings"
	"sync"

	guuid "github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

const (
	queueSize = 16
)

var TemplateFuncs = template.FuncMap{"rangeStruct": rangeStructer}

// Tracker structure of a Tracker object
type Tracker struct {
	DB               *statsdb.StatsDB
	HTMLTemplate     *template.Template
	HTMLTemplateName string
	DestEndpoint     string
	Queue            chan<- string
	Wg               sync.WaitGroup
}

// DefaultPath endpoint to the default path
func (t *Tracker) DefaultPath(w http.ResponseWriter, r *http.Request) {
	logrus.WithFields(logrus.Fields{
		"EndPoint:": r.URL.Path,
	}).Info("tracker.DefaultPath")
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err,
			"URL":   r.URL.Path,
		}).Info("tracker.DefaultPath, ioutil.ReadAll")
		w.WriteHeader(http.StatusBadRequest)

		return
	}

	logrus.WithFields(logrus.Fields{
		"body": string(body),
	}).Info("tracker.DefaultPath")

	w.WriteHeader(http.StatusNotFound)
}

// StatsAPI endpoint to StatsAPI
func (t *Tracker) StatsAPI(w http.ResponseWriter, r *http.Request) {
	logrus.Info("tracker.StatsAPI")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	actions := t.DB.GetAllActions()
	byteList, err := json.Marshal(actions)
	if err != nil {
		logrus.Error(err, "Error Unmashaling", "actions", byteList)
		logrus.WithFields(logrus.Fields{
			"Error":   err,
			"actions": byteList,
		}).Info("Error unmarshalling")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(byteList); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err,
		}).Info("Error writing response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// StatsWeb endpoint to StatsWeb
func (t *Tracker) StatsWeb(w http.ResponseWriter, r *http.Request) {
	logrus.Info("tracker.StatsWeb")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	actions := t.DB.GetAllActions()

	err := t.HTMLTemplate.ExecuteTemplate(w, t.HTMLTemplateName, actions)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err,
		}).Info("Error writing response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

// Action endpoint to Action
func (t *Tracker) Action(w http.ResponseWriter, r *http.Request) {
	logrus.Info("tracker.Action")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err,
		}).Info("Error reading response")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	action := &statsdb.GitHubAction{}
	if err := json.Unmarshal(body, action); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err,
			"body":  string(body),
		}).Info("Error unmarshalling")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	logrus.WithFields(logrus.Fields{
		"Action":  action,
		"Payload": action.Payload,
	}).Info("Incoming")
	err = t.DB.Save(action)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Error":  err,
			"action": action,
		}).Info("Error saving")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	go func() {
		localAction := getTestActions()
		if localAction != nil {
			t.sendEvent(localAction)
		}
	}()

	w.WriteHeader(http.StatusOK)
}

// EventSink go-routine to emit test events
// it start the thead and returns a channel
// where it expects to find test events
func (t *Tracker) EventSink() chan<- string {
	logrus.Info("tracker.EventSink")
	c := make(chan string, queueSize)
	t.Wg.Add(1)
	go func() {
		defer t.Wg.Done()
		for {
			action, flag := <-c
			if flag {
				logrus.WithFields(logrus.Fields{
					"event": action,
				}).Info("Sending test event")
				resp, err := http.Post(t.DestEndpoint, "application/json", bytes.NewReader([]byte(action)))
				if err != nil {
					logrus.WithFields(logrus.Fields{
						"Error": err,
					}).Info("Error posting event")
					continue
				}

				logrus.WithFields(logrus.Fields{
					"event":    action,
					"response": resp,
				}).Info("Event Send")
			} else {
				logrus.Info("EventSink done")
				return
			}
		}
	}()
	return c
}

// sendEvent format a test action to json
// and pushes it on the event sink
func (t *Tracker) sendEvent(action *statsdb.GitHubAction) {
	buf, err := json.Marshal(action)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err,
		}).Info("Error unmarshalling")
		return
	}
	t.Queue <- string(buf)
}

// getTestAction runs the go test and generate test actions
func getTestActions() *statsdb.GitHubAction {
	logrus.Info("trackerapi.runTestCmd")

	cmd := exec.Command("go", "test", "-cover", "./...")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err,
		}).Info("Error running test command")
		return nil
	}
	reader := bytes.NewReader(out.Bytes())
	buf := bufio.NewReader(reader)
	for {
		linebytes, _, err := buf.ReadLine()
		if err == io.EOF {
			break
		}
		if fields := parseFields(linebytes); fields != nil {
			return &statsdb.GitHubAction{
				Event:            fields.action,
				VentureConfigId:  guuid.New().String(),
				VentureReference: guuid.New().String(),
				CreatedAt:        "",
				Culture:          "en_EN",
				ActionType:       "api",
				ActionReference:  "",
				Version:          "1.0.0",
				Route:            "",
				Payload: &statsdb.Payload{
					ServiceName: "tracker",
					Coverage:    fields.coverage,
				},
			}
		}
	}
	return nil
}

// parseFields parses a line of a test coverage
// to extract the package directory and the coverage weight
func parseFields(linebytes []byte) *struct {
	action   string
	coverage float64
} {
	line := string(linebytes)
	fields := strings.Split(line, "\t")
	if len(fields) > 3 && strings.TrimSpace(fields[0]) == "ok" {
		if beg := strings.Index(fields[3], "coverage"); beg != -1 {
			beg += len("coverage") + 1
			if end := strings.Index(fields[3][beg:], "%"); end != -1 {
				action := fields[1]
				coverage := strings.TrimSpace(fields[3][beg : beg+end])
				coverage_val, err := strconv.ParseFloat(coverage, 64)
				if err != nil {
					logrus.WithFields(logrus.Fields{
						"Error": err,
						"value": coverage,
					}).Info("Could not convert coverage")
					return nil
				}
				logrus.WithFields(logrus.Fields{
					"event":    action,
					"coverage": coverage_val,
				}).Info("Test")
				return &struct {
					action   string
					coverage float64
				}{
					action:   action,
					coverage: coverage_val,
				}
			}
		}
	}
	return nil
}

// rangeStructer takes the first argument, which must be a struct, and
// returns the value of each field in a slice. It will return nil
// if there are no arguments or first argument is not a struct
func rangeStructer(args ...interface{}) []interface{} {
	if len(args) == 0 {
		return nil
	}

	v := reflect.ValueOf(args[0])
	if v.Kind() != reflect.Struct {
		return nil
	}

	out := make([]interface{}, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		out[i] = v.Field(i).Interface()
	}

	return out
}
