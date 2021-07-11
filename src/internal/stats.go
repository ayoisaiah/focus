package focus

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/urfave/cli/v2"
)

type Stats struct {
	StartTime          time.Time
	EndTime            time.Time
	Sessions           []session
	totalMins          int
	completedPomodoros int
	abandonedPomodoros int
	Weekday            map[time.Weekday]int
	HourofDay          map[int]int
}

func (s *Stats) getSessions() {
	b, err := store.getSessions(s.StartTime, s.EndTime)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, v := range b {
		sess := session{}

		err = json.Unmarshal(v, &sess)
		if err != nil {
			fmt.Println(err)
			return
		}

		s.Sessions = append(s.Sessions, sess)
	}
}

func (s *Stats) total() {
	for _, v := range s.Sessions {
		if v.EndTime.IsZero() {
			continue
		}

		s.Weekday[v.StartTime.Weekday()]++
		s.HourofDay[v.StartTime.Hour()]++

		if v.Completed {
			s.completedPomodoros++
		} else {
			s.abandonedPomodoros++
		}

		s.totalMins += int(math.Round(v.EndTime.Sub(v.StartTime).Minutes()))
	}

	fmt.Println("Total minutes worked: ", s.totalMins)
	fmt.Println("Total pomodoros completed: ", s.completedPomodoros)
	fmt.Println("Total pomodoros abandoned: ", s.abandonedPomodoros)
	fmt.Println("Weekdays")

	for k, v := range s.Weekday {
		fmt.Println(k.String(), " -> ", v)
	}
}

func (s *Stats) Run() {
	s.getSessions()
	s.total()
}

func NewStats(ctx *cli.Context) *Stats {
	s := &Stats{}

	s.StartTime = time.Time{}
	s.EndTime = time.Now()
	s.Weekday = make(map[time.Weekday]int)
	s.HourofDay = make(map[int]int)

	for i := 0; i <= 6; i++ {
		s.Weekday[time.Weekday(i)] = 0
	}

	for i := 0; i <= 23; i++ {
		s.HourofDay[i] = 0
	}

	return s
}
