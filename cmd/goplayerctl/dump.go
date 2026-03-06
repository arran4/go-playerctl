package main

import (
	"encoding/json"
	"fmt"
	"io"

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
