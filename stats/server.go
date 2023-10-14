package stats

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/pterm/pterm"

	"github.com/ayoisaiah/focus/internal/timeutil"
)

type TemplateData struct {
	StartTime string
	EndTime   string
	Stats     string
	MainChart string
	Days      int
}

//go:embed web/*
var web embed.FS

var tpl = template.Must(template.New("index.html").ParseFS(web, "web/index.html"))

type errorHandler func(w http.ResponseWriter, r *http.Request) error

func (h errorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h(w, r)
	if err != nil {
		pterm.Fatal.Println(err)
	}
}

func (s *Stats) getStats(startTime, endTime time.Time, tagList []string) ([]byte, error) {
	s.Opts.StartTime = startTime
	s.Opts.EndTime = endTime
	s.Opts.Tags = tagList

	sessions, err := s.DB.GetSessions(
		s.Opts.StartTime,
		s.Opts.EndTime,
		s.Opts.Tags,
	)
	if err != nil {
		return nil, err
	}

	s.Compute(sessions)

	return s.ToJSON()
}

func (s *Stats) index(w http.ResponseWriter, r *http.Request) error {
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

	b, err := s.getStats(startTime, endTime, tagList)
	if err != nil {
		return err
	}

	var buf bytes.Buffer

	err = tpl.Execute(&buf, &TemplateData{
		StartTime: startTime.Format(time.RFC3339Nano),
		EndTime:   endTime.Format(time.RFC3339Nano),
		Days:      int(math.Round(endTime.Sub(startTime).Seconds() / (24 * 60 * 60))),
		Stats:     string(b),
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

func (s *Stats) Server(port uint) error {
	mux := http.NewServeMux()

	staticFS := http.FS(web)
	fs := http.FileServer(staticFS)

	mux.Handle("/web/", fs)
	mux.Handle("/", errorHandler(s.index))

	pterm.Info.Printfln("starting server on port: %d", port)

	//nolint:gosec // no timeout is ok
	return http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}
