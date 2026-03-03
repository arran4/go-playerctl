package main

//go:generate md2man -in ../../doc/playerctl-go.1.md -out ../../doc/goplayerctl.1

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/arran4/go-playerctl/pkg/playerctl"
)

var (
	newPlayer       = playerctl.NewPlayer
	newPlayerManger = playerctl.NewPlayerManager
)

type cliOptions struct {
	format     string
	follow     bool
	followTick time.Duration
	allPlayers bool
	tuiScheme  string
}

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("playerctl", flag.ContinueOnError)
	fs.SetOutput(stderr)

	playerArg := fs.String("player", "", "comma-separated player instances to control")
	ignoreArg := fs.String("ignore-player", "", "comma-separated player instances to ignore")
	allPlayers := fs.Bool("all-players", false, "target all available players")
	listAll := fs.Bool("list-all", false, "list all available players")
	version := fs.Bool("version", false, "print version")
	format := fs.String("format", "", "output format template")
	follow := fs.Bool("follow", false, "follow output updates")
	followInterval := fs.Duration("follow-interval", time.Second, "follow polling interval")
	tuiScheme := fs.String("tui-scheme", "arrow", "TUI control scheme (arrow, vim, winamp, emacs)")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *version {
		fmt.Fprintln(stdout, "go-playerctl (port in progress)")
		return 0
	}
	if *listAll {
		manager, err := newPlayerManger(playerctl.SourceNone)
		if err != nil {
			fmt.Fprintf(stderr, "failed to list players: %v\n", err)
			return 1
		}
		for _, name := range manager.PlayerNames() {
			fmt.Fprintln(stdout, name.Instance)
		}
		return 0
	}

	remaining := fs.Args()
	if len(remaining) == 0 {
		fmt.Fprintln(stderr, "missing command")
		return 2
	}

	cmd := strings.ToLower(remaining[0])
	supported := map[string]struct{}{
		"play": {}, "pause": {}, "play-pause": {}, "playpause": {},
		"next": {}, "previous": {}, "status": {}, "metadata": {}, "tui": {},
	}
	if _, ok := supported[cmd]; !ok {
		fmt.Fprintf(stderr, "unknown command: %s\n", cmd)
		return 2
	}
	if *follow && cmd != "status" && cmd != "metadata" {
		fmt.Fprintln(stderr, "--follow is only supported for status and metadata")
		return 2
	}

	instances := selectInstances(*playerArg, *ignoreArg, *allPlayers)
	if len(instances) == 0 && cmd != "tui" {
		fmt.Fprintln(stderr, "no players selected; use --player or --all-players")
		return 2
	}
	opts := cliOptions{format: *format, follow: *follow, followTick: *followInterval, allPlayers: *allPlayers, tuiScheme: *tuiScheme}
	if opts.follow {
		return followCommand(cmd, instances, stdout, stderr, opts)
	}

	if cmd == "tui" {
		return runTUI(instances, stdout, stderr, opts)
	}

	for _, instance := range instances {
		p, err := newPlayer(instance, playerctl.SourceDBusSession)
		if err != nil {
			fmt.Fprintf(stderr, "failed to connect player %q: %v\n", instance, err)
			if !*allPlayers {
				return 1
			}
			continue
		}
		if code := runCommand(cmd, p, stdout, stderr, opts); code != 0 {
			p.Close()
			if !*allPlayers {
				return code
			}
		}
		p.Close()
	}
	return 0
}

func followCommand(cmd string, instances []string, stdout, stderr io.Writer, opts cliOptions) int {
	if opts.followTick <= 0 {
		opts.followTick = time.Second
	}
	last := map[string]string{}
	deadline := time.After(3 * opts.followTick)
	tick := time.NewTicker(opts.followTick)
	defer tick.Stop()
	for {
		for _, instance := range instances {
			p, err := newPlayer(instance, playerctl.SourceDBusSession)
			if err != nil {
				continue
			}
			line, err := queryOutput(cmd, p, opts)
			p.Close()
			if err != nil {
				continue
			}
			if line != last[instance] {
				if opts.allPlayers {
					fmt.Fprintf(stdout, "%s %s\n", instance, line)
				} else {
					fmt.Fprintln(stdout, line)
				}
				last[instance] = line
			}
		}
		select {
		case <-deadline:
			return 0
		case <-tick.C:
		}
	}
}

func selectInstances(playerArg, ignoreArg string, allPlayers bool) []string {
	ignore := map[string]struct{}{}
	for _, v := range strings.Split(ignoreArg, ",") {
		v = strings.TrimSpace(v)
		if v != "" {
			ignore[v] = struct{}{}
		}
	}
	if allPlayers {
		manager, err := newPlayerManger(playerctl.SourceNone)
		if err != nil {
			return nil
		}
		instances := make([]string, 0, len(manager.PlayerNames()))
		for _, n := range manager.PlayerNames() {
			if _, skip := ignore[n.Instance]; !skip {
				instances = append(instances, n.Instance)
			}
		}
		return instances
	}
	instances := []string{}
	for _, v := range strings.Split(playerArg, ",") {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if _, skip := ignore[v]; skip {
			continue
		}
		instances = append(instances, v)
	}
	return instances
}

func queryOutput(cmd string, p *playerctl.Player, opts cliOptions) (string, error) {
	ctx := map[string]string{"player": p.Instance()}
	switch cmd {
	case "status":
		status, err := p.PlaybackStatus()
		if err != nil {
			return "", err
		}
		ctx["status"] = status.String()
	case "metadata":
		meta, err := p.Metadata()
		if err != nil {
			return "", err
		}

		title := playerctl.ExtractTitle(meta)
		artist := playerctl.ExtractArtist(meta)
		album := playerctl.ExtractAlbum(meta)

		ctx["title"], ctx["artist"], ctx["album"] = title, artist, album
	}
	if opts.format == "" {
		if cmd == "status" {
			return ctx["status"], nil
		}
		return ctx["title"], nil
	}
	f, err := playerctl.NewFormatter(opts.format)
	if err != nil {
		return "", err
	}
	return f.Expand(ctx)
}

func runCommand(cmd string, p *playerctl.Player, stdout, stderr io.Writer, opts cliOptions) int {
	write := func(v string) {
		if opts.allPlayers {
			fmt.Fprintf(stdout, "%s %s\n", p.Instance(), v)
			return
		}
		fmt.Fprintln(stdout, v)
	}
	switch cmd {
	case "play":
		if err := p.Play(); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	case "pause":
		if err := p.Pause(); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	case "play-pause", "playpause":
		if err := p.PlayPause(); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	case "next":
		if err := p.Next(); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	case "previous":
		if err := p.Previous(); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	case "status", "metadata":
		line, err := queryOutput(cmd, p, opts)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		write(line)
	}
	return 0
}
