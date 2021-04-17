package statsdb

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

// StatsDB structure of a StatsDB object
type StatsDB struct {
	DBName              string
	DB                  *sql.DB
	datadefinitionlStmt *sql.Stmt
	createStmt          *sql.Stmt
	selectStmt          *sql.Stmt
}

// Payload structure of a Payload message
type Payload struct {
	ServiceName string  `json:"service_name"`
	Coverage    float64 `json:"coverage"`
}

// GitHubAction structure of a GitHubAction message
type GitHubAction struct {
	Event            string   `json:"event"`
	VentureConfigId  string   `json:"venture_config_id"`
	VentureReference string   `json:"venture_reference"`
	CreatedAt        string   `json:"created_at"`
	Culture          string   `json:"culture"`
	ActionType       string   `json:"action_type"`
	ActionReference  string   `json:"action_reference"`
	Version          string   `json:"version"`
	Route            string   `json:"route"`
	Payload          *Payload `json:"payload,omitempty"`
}

const (
	ddlSQL = `create table if not exists action (id INTEGER PRIMARY KEY ASC,
	event text,venture_config_id text,venture_reference text,
	created_at text,culture text,action_type text,
	action_reference text,version text,route text,
	service_name text, coverage int);
`
	createSQL = `INSERT INTO action (
	event,venture_config_id,venture_reference,created_at,culture,
	action_type,action_reference,version,route,service_name, coverage)
	VALUES(?,?,?,?,?,?,?,?,?,?,?);
`
	selectSQL = `SELECT 
event,
venture_config_id,
venture_reference,
created_at,
culture,
action_type,
action_reference,
version,
route,
service_name,
coverage
FROM action;
`
)

// Open open a sqlite 3 database file
func Open(dbName string) *StatsDB {
	db, err := sql.Open("sqlite3", dbName)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err,
		}).Info("Sql error")
		return nil
	}

	return &StatsDB{
		DBName: dbName,
		DB:     db,
	}
}

// Setup creates the start table
func (s *StatsDB) Setup() error {
	ddlStmt, err := s.DB.Prepare(ddlSQL)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err,
		}).Info("Sql error")
		return err
	}
	s.datadefinitionlStmt = ddlStmt

	_, err = s.datadefinitionlStmt.Exec()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err,
			"sql":   ddlSQL,
		}).Info("Sql error")
		return err
	}

	createStmt, err := s.DB.Prepare(createSQL)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err,
		}).Info("Sql error")
		return err
	}
	selectStmt, err := s.DB.Prepare(selectSQL)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err,
		}).Info("Sql error")
		return err
	}
	s.createStmt = createStmt
	s.selectStmt = selectStmt
	return nil
}

// Save inserts a github action into the action table
func (s *StatsDB) Save(action *GitHubAction) error {

	_, err := s.createStmt.Exec(action.Event,
		action.VentureConfigId,
		action.VentureReference,
		action.CreatedAt,
		action.Culture,
		action.ActionType,
		action.ActionReference,
		action.Version,
		action.Route,
		action.Payload.ServiceName,
		action.Payload.Coverage)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err,
			"sql":   createSQL,
		}).Info("Sql error")
		return err
	}

	return nil
}

// GetAllActions selects all test event stored in the action table
func (s *StatsDB) GetAllActions() []GitHubAction {
	rows, err := s.selectStmt.Query()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err,
			"sql":   selectSQL,
		}).Info("Sql error")
		return nil
	}

	events := []GitHubAction{}
	for rows.Next() {
		tracker := GitHubAction{Payload: &Payload{}}
		err = rows.Scan(&tracker.Event,
			&tracker.VentureConfigId,
			&tracker.VentureReference,
			&tracker.CreatedAt,
			&tracker.Culture,
			&tracker.ActionType,
			&tracker.ActionReference,
			&tracker.Version,
			&tracker.Route,
			&tracker.Payload.ServiceName,
			&tracker.Payload.Coverage)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"Error": err,
				"sql":   selectSQL,
			}).Info("Sql error")
			return nil
		}
		events = append(events, tracker)
	}
	err = rows.Err()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err,
			"sql":   selectSQL,
		}).Info("Sql error")
		return nil
	}

	return events
}
