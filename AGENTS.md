# Guidelines

When making changes to the `goplayerctl` CLI, you must ensure the following remain up to date with any modifications:
- The shell completion scripts (`data/goplayerctl.bash`, `data/goplayerctl.zsh`)
- The man page (`doc/playerctl-go.1.md`)
- The `README.md` file
- The `goplayerctl` CLI includes a `mock` command that starts a dummy MPRIS media player on the session D-Bus (named `org.mpris.MediaPlayer2.mock`) to facilitate testing without a real media player. It uses `github.com/godbus/dbus/v5/prop` to export and handle MPRIS properties. This can be extended for d-bus testing in a headless environment etc.
