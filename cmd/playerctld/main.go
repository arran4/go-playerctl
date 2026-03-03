package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/arran4/go-playerctl/pkg/playerctl"
	"github.com/godbus/dbus/v5"
)

var (
	newManager       = playerctl.NewPlayerManager
	connectSessionDB = dbus.ConnectSessionBus
)

const (
	serviceName  = "org.mpris.MediaPlayer2.playerctld"
	servicePath  = "/org/mpris/MediaPlayer2"
	serviceIface = "com.github.altdesktop.playerctld"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("playerctld", flag.ContinueOnError)
	fs.SetOutput(stderr)
	version := fs.Bool("version", false, "print version")
	once := fs.Bool("once", false, "refresh players once and print order")
	interval := fs.Duration("refresh-interval", 2*time.Second, "refresh interval")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *version {
		fmt.Fprintln(stdout, "go-playerctld (port in progress)")
		return 0
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	daemon, err := newDaemon(*interval)
	if err != nil {
		fmt.Fprintf(stderr, "daemon init failed: %v\n", err)
		return 1
	}
	if *once {
		if err := daemon.refreshAndPrint(stdout); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		return 0
	}
	if err := daemon.exportService(); err != nil {
		fmt.Fprintf(stderr, "dbus export failed: %v\n", err)
		return 1
	}
	if err := daemon.run(ctx, stdout); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

type daemon struct {
	manager  *playerctl.PlayerManager
	interval time.Duration

	mu               sync.Mutex
	players          []string
	active           string
	lastActivityUnix int64
	conn             *dbus.Conn
}

func newDaemon(interval time.Duration) (*daemon, error) {
	m, err := newManager(playerctl.SourceNone)
	if err != nil {
		return nil, err
	}
	if interval <= 0 {
		interval = 2 * time.Second
	}
	return &daemon{manager: m, interval: interval}, nil
}

func (d *daemon) exportService() error {
	conn, err := connectSessionDB()
	if err != nil {
		return err
	}
	reply, err := conn.RequestName(serviceName, dbus.NameFlagDoNotQueue)
	if err != nil {
		_ = conn.Close()
		return err
	}
	if reply != dbus.RequestNameReplyPrimaryOwner {
		_ = conn.Close()
		return fmt.Errorf("name already owned")
	}
	d.conn = conn
	return conn.Export(d, dbus.ObjectPath(servicePath), serviceIface)
}

func (d *daemon) refreshAndPrint(w io.Writer) error {
	if err := d.manager.Refresh(); err != nil {
		return err
	}
	names := make([]string, 0, len(d.manager.PlayerNames()))
	for _, n := range d.manager.PlayerNames() {
		names = append(names, n.Instance)
		fmt.Fprintln(w, n.Instance)
	}
	d.mu.Lock()
	d.players = names
	if len(names) > 0 {
		d.active = names[0]
	}
	d.lastActivityUnix = time.Now().Unix()
	d.mu.Unlock()
	return nil
}

func (d *daemon) run(ctx context.Context, w io.Writer) error {
	t := time.NewTicker(d.interval)
	defer t.Stop()
	defer func() {
		if d.conn != nil {
			_ = d.conn.Close()
		}
	}()
	if err := d.refreshAndPrint(w); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			if err := d.refreshAndPrint(w); err != nil {
				return err
			}
		}
	}
}

// Shift rotates active player order.
func (d *daemon) Shift() *dbus.Error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.players) <= 1 {
		return nil
	}
	if d.conn != nil {
		_ = d.conn.Emit(dbus.ObjectPath(servicePath), serviceIface+".ActivePlayerChangeBegin")
	}
	first := d.players[0]
	d.players = append(d.players[1:], first)
	d.active = d.players[0]
	d.lastActivityUnix = time.Now().Unix()
	if d.conn != nil {
		_ = d.conn.Emit(dbus.ObjectPath(servicePath), serviceIface+".ActivePlayerChangeEnd")
	}
	return nil
}

// Unshift moves last player to the front.
func (d *daemon) Unshift() *dbus.Error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.players) <= 1 {
		return nil
	}
	if d.conn != nil {
		_ = d.conn.Emit(dbus.ObjectPath(servicePath), serviceIface+".ActivePlayerChangeBegin")
	}
	last := d.players[len(d.players)-1]
	d.players = append([]string{last}, d.players[:len(d.players)-1]...)
	d.active = d.players[0]
	d.lastActivityUnix = time.Now().Unix()
	if d.conn != nil {
		_ = d.conn.Emit(dbus.ObjectPath(servicePath), serviceIface+".ActivePlayerChangeEnd")
	}
	return nil
}

// PlayerNames exposes current player ordering.
func (d *daemon) PlayerNames() ([]string, *dbus.Error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]string, len(d.players))
	copy(out, d.players)
	return out, nil
}

// ActivePlayer exposes active player tracking data.
func (d *daemon) ActivePlayer() (string, int64, *dbus.Error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.active, d.lastActivityUnix, nil
}
