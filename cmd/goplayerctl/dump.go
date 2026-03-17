package main

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/arran4/go-playerctl/pkg/playerctl"
)

type dumpData struct {
	Name           string         `json:"name"`
	Instance       string         `json:"instance"`
	PlaybackStatus *string        `json:"playback_status,omitempty"`
	LoopStatus     *string        `json:"loop_status,omitempty"`
	Shuffle        *bool          `json:"shuffle,omitempty"`
	Volume         *float64       `json:"volume,omitempty"`
	Position       *int64         `json:"position,omitempty"`
	CanControl     *bool          `json:"can_control,omitempty"`
	CanPlay        *bool          `json:"can_play,omitempty"`
	CanPause       *bool          `json:"can_pause,omitempty"`
	CanSeek        *bool          `json:"can_seek,omitempty"`
	CanGoNext      *bool          `json:"can_go_next,omitempty"`
	CanGoPrevious  *bool          `json:"can_go_previous,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

func ptr[T any](v T) *T {
	return &v
}

func runDump(instances []string, stdout, stderr io.Writer, opts cliOptions) int {
	results := []dumpData{}

	for _, instance := range instances {
		p, err := newPlayer(instance, playerctl.SourceDBusSession)
		if err != nil {
			fmt.Fprintf(stderr, "failed to connect player %q: %v\n", instance, err)
			if !opts.allPlayers {
				return 1
			}
			continue
		}

		data := dumpData{
			Name:     p.Name(),
			Instance: p.Instance(),
		}

		if status, err := p.PlaybackStatus(); err == nil {
			data.PlaybackStatus = ptr(status.String())
		}
		if loop, err := p.LoopStatus(); err == nil {
			data.LoopStatus = ptr(loop.String())
		}
		if shuffle, err := p.Shuffle(); err == nil {
			data.Shuffle = ptr(shuffle)
		}
		if vol, err := p.Volume(); err == nil {
			data.Volume = ptr(vol)
		}
		if pos, err := p.Position(); err == nil {
			data.Position = ptr(pos)
		}
		if cc, err := p.CanControl(); err == nil {
			data.CanControl = ptr(cc)
		}
		if cp, err := p.CanPlay(); err == nil {
			data.CanPlay = ptr(cp)
		}
		if cpa, err := p.CanPause(); err == nil {
			data.CanPause = ptr(cpa)
		}
		if cs, err := p.CanSeek(); err == nil {
			data.CanSeek = ptr(cs)
		}
		if cn, err := p.CanGoNext(); err == nil {
			data.CanGoNext = ptr(cn)
		}
		if cpv, err := p.CanGoPrevious(); err == nil {
			data.CanGoPrevious = ptr(cpv)
		}
		if meta, err := p.Metadata(); err == nil {
			data.Metadata = make(map[string]any, len(meta))
			for k, v := range meta {
				data.Metadata[k] = v.Value()
			}
		}

		results = append(results, data)
		p.Close()
	}

	if opts.json {
		var b []byte
		var err error

		if opts.allPlayers || len(instances) > 1 {
			if opts.indent != "" {
				b, err = json.MarshalIndent(results, "", opts.indent)
			} else {
				b, err = json.Marshal(results)
			}
		} else {
			if len(results) > 0 {
				if opts.indent != "" {
					b, err = json.MarshalIndent(results[0], "", opts.indent)
				} else {
					b, err = json.Marshal(results[0])
				}
			} else {
				return 1
			}
		}

		if err != nil {
			fmt.Fprintf(stderr, "failed to marshal json: %v\n", err)
			return 1
		}

		fmt.Fprintln(stdout, string(b))
		return 0
	}

	for i, r := range results {
		if i > 0 {
			fmt.Fprintln(stdout)
			fmt.Fprintln(stdout)
		}

		fmt.Fprintf(stdout, "Player: %s (%s)\n", r.Name, r.Instance)

		if r.PlaybackStatus != nil {
			fmt.Fprintf(stdout, "Playback Status: %s\n", *r.PlaybackStatus)
		}
		if r.LoopStatus != nil {
			fmt.Fprintf(stdout, "Loop Status: %s\n", *r.LoopStatus)
		}
		if r.Shuffle != nil {
			fmt.Fprintf(stdout, "Shuffle: %v\n", *r.Shuffle)
		}
		if r.Volume != nil {
			fmt.Fprintf(stdout, "Volume: %f\n", *r.Volume)
		}
		if r.Position != nil {
			fmt.Fprintf(stdout, "Position: %d\n", *r.Position)
		}
		if r.CanControl != nil {
			fmt.Fprintf(stdout, "Can Control: %v\n", *r.CanControl)
		}
		if r.CanPlay != nil {
			fmt.Fprintf(stdout, "Can Play: %v\n", *r.CanPlay)
		}
		if r.CanPause != nil {
			fmt.Fprintf(stdout, "Can Pause: %v\n", *r.CanPause)
		}
		if r.CanSeek != nil {
			fmt.Fprintf(stdout, "Can Seek: %v\n", *r.CanSeek)
		}
		if r.CanGoNext != nil {
			fmt.Fprintf(stdout, "Can Go Next: %v\n", *r.CanGoNext)
		}
		if r.CanGoPrevious != nil {
			fmt.Fprintf(stdout, "Can Go Previous: %v\n", *r.CanGoPrevious)
		}

		if len(r.Metadata) > 0 {
			keys := make([]string, 0, len(r.Metadata))
			for k := range r.Metadata {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			for _, k := range keys {
				v := r.Metadata[k]
				valStr := fmt.Sprintf("%v", v)
				if slice, ok := v.([]string); ok {
					valStr = strings.Join(slice, ", ")
				} else if arr, ok := v.([]interface{}); ok {
					var strArr []string
					for _, item := range arr {
						strArr = append(strArr, fmt.Sprintf("%v", item))
					}
					valStr = strings.Join(strArr, ", ")
				}
				fmt.Fprintf(stdout, "%s: %s\n", k, valStr)
			}
		}
	}
	return 0
}
