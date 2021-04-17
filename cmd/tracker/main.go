package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"os/signal"
	"ringier/pkg/statsdb"
	"ringier/pkg/trackerapi"
	"sync"
	"syscall"

	goflags "flag"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// rootCmd application command object
var rootCmd = &cobra.Command{
	Use:   "tracker",
	Short: "Tracks lock and remote tests",
	Long:  "Tracks test results from github and local tests",
	Run:   run,
}

// init set configuration defaults
func init() {
	rootCmd.Flags().AddGoFlagSet(goflags.CommandLine)

	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().String("port", "8080", "Port to listen on")
	rootCmd.PersistentFlags().String("host", "", "Host IP to listen on. If the host is empty it will listen on all IPs")
	rootCmd.PersistentFlags().String("dbName",
		"./stats.db", "Test statistics database")
	rootCmd.PersistentFlags().String("webTemplate",
		"index.tmpl", "HTML web template")
	rootCmd.PersistentFlags().String("styleSheet",
		"./style.css", "Web cascading style sheet")
	rootCmd.PersistentFlags().String("destEndpoint",
		"localhost:8080/action", "endpoint for local test action events")
}

func initConfig() {
	if err := viper.BindPFlags(rootCmd.Flags()); err != nil {
		logrus.Error(err, "viper.BindPFlags")
	}

	viper.AutomaticEnv()
	viper.AddConfigPath(".")

	viper.SetConfigName("tracker")

	if err := viper.ReadInConfig(); err == nil {
		logrus.WithFields(logrus.Fields{
			"file": viper.ConfigFileUsed(),
		}).Info("viper.ReadInConfig.")
	} else {
		logrus.Error(err, "viper.ReadInConfig failed")
	}

	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetOutput(os.Stdout)

	loglevel := logrus.Level(viper.GetInt("loglevel"))

	logrus.WithFields(logrus.Fields{"loglevel": loglevel}).Info("Logging config.")

	logrus.SetLevel(loglevel)
	logrus.SetReportCaller(true)

	for _, v := range viper.AllKeys() {
		logrus.WithFields(logrus.Fields{
			v: viper.Get(v),
		}).Info("Configs loaded")
	}
}

// fileServe object to serve stylesheets
type fileServe struct {
	http.FileSystem
	filename string
}

// Open open a file to serve as a stylesheet
func (fs fileServe) Open(name string) (http.File, error) {
	return fs.FileSystem.Open(fs.filename)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logrus.Error(err, "Error starting rootCmd.Execute()")
		os.Exit(1)
	}
}

// CatchCtrlC function performs a graceful shutdown
func catchCtrlC(srv *http.Server) {
	sigint := make(chan os.Signal, 1)

	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
	<-sigint
	logrus.Info("We received an interrupt signal, gracefully shutting down")
	if err := srv.Shutdown(context.Background()); err != nil {
		logrus.WithFields(logrus.Fields{"Error": err}).Info("Server shutdown error")
	}
}

func run(cmd *cobra.Command, args []string) {
	tracker := &trackerapi.Tracker{Wg: sync.WaitGroup{}}
	defer tracker.Wg.Wait()
	tracker.DB = statsdb.Open(viper.GetString("dbName"))
	if tracker.DB == nil {
		return
	}
	err := tracker.DB.Setup()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err,
		}).Info("Error setting up database")
		return
	}
	templ := template.New("").Funcs(trackerapi.TemplateFuncs)
	tracker.HTMLTemplateName = viper.GetString("webTemplate")
	tracker.HTMLTemplate, err = templ.ParseFiles(tracker.HTMLTemplateName)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err,
		}).Info("Error parsing the web template")
		return
	}
	tracker.DestEndpoint = viper.GetString("destEndpoint")
	tracker.Queue = tracker.EventSink()
	defer close(tracker.Queue)

	mux := http.NewServeMux()
	mux.Handle(viper.GetString("styleSheet"),
		http.FileServer(fileServe{
			FileSystem: http.Dir("."),
			filename:   "style.css",
		}))
	mux.HandleFunc("/", tracker.DefaultPath)
	mux.HandleFunc("/action", tracker.Action)
	mux.HandleFunc("/api/stats", tracker.StatsAPI)
	mux.HandleFunc("/stats", tracker.StatsWeb)

	svr := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", viper.GetString("host"), viper.GetString("port")),
		Handler: mux,
	}

	go catchCtrlC(svr)

	if err := svr.ListenAndServe(); err != nil {
		logrus.WithFields(logrus.Fields{
			"Error": err,
		}).Info("HTTP Server shutdown response")
		return
	}
}
