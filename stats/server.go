package stats

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"log"
	"math"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/pterm/pterm"

	"github.com/ayoisaiah/focus/internal/timeutil"
	"github.com/ayoisaiah/focus/store"
)

type (
	TemplateData struct {
		StartTime string
		EndTime   string
		Stats     string
		MainChart string
		Days      int
	}
)

type errorHandler func(w http.ResponseWriter, r *http.Request) error

func (h errorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h(w, r)
	if err != nil {
		pterm.Fatal.Println(err)
	}
}

//go:embed web/*
var web embed.FS

var db store.DB

var tpl = template.Must(
	template.New("index.html").ParseFS(web, "web/index.html"),
)

// computeSummary calculates the total minutes, completed sessions,
// and abandoned sessions for the current time period.
func (s *Stats) computeStats() ([]byte, error) {
	sessions, err := db.GetSessions(s.StartTime, s.EndTime, s.Tags)
	if err != nil {
		return nil, err
	}

	s.Sessions = sessions

	// For all-time, set start time to the date of the first session
	if s.StartTime.IsZero() && len(s.Sessions) > 0 {
		s.StartTime = timeutil.RoundToStart(s.Sessions[0].StartTime)
	}

	s.computeSummary()
	// s.computeAggregates()

	return s.ToJSON()
}

func (s *Stats) Index(w http.ResponseWriter, r *http.Request) error {
	query := r.URL.Query()

	start := query.Get("start_time")
	end := query.Get("end_time")
	tags := query.Get("tags")

	now := time.Now()

	startTime, err := time.ParseInLocation("2006-01-02", start, now.Location())
	if err != nil {
		startTime = timeutil.RoundToStart(time.Now().AddDate(0, 0, -6))
	}

	endTime, err := time.ParseInLocation("2006-01-02", end, now.Location())
	if err != nil {
		endTime = time.Now()
	}

	endTime = timeutil.RoundToEnd(endTime)

	var tagList []string
	if tags != "" {
		tagList = strings.Split(tags, ",")
	}

	s.StartTime = startTime
	s.EndTime = endTime
	s.Tags = tagList

	b, err := s.computeStats()
	if err != nil {
		return err
	}

	fmt.Println(string(b))

	var buf bytes.Buffer

	err = tpl.Execute(&buf, &TemplateData{
		StartTime: startTime.Format(time.RFC3339Nano),
		EndTime:   endTime.Format(time.RFC3339Nano),
		Days: int(
			math.Round(endTime.Sub(startTime).Seconds() / (24 * 60 * 60)),
		),
		Stats: string(b),
	})
	if err != nil {
		return err
	}

	_, err = w.Write(buf.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func openbrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).
			Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = errors.New("unsupported platform")
	}

	if err != nil {
		log.Fatal(err)
	}
}

func Server(dB store.DB, port uint) error {
	mux := http.NewServeMux()

	s := &Stats{
		DB: db,
	}

	staticFS := http.FS(web)
	fs := http.FileServer(staticFS)

	mux.Handle("/web/", fs)
	mux.Handle("/", errorHandler(s.Index))

	pterm.Info.Printfln("starting server on port: %d", port)

	// openbrowser("http://localhost:1111")

	//nolint:gosec // no timeout is ok
	return http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}
