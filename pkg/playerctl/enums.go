package playerctl

// Source: playerctl/playerctl-enum-types.h.in
// /*** BEGIN file-header ***/
// #pragma once
//
// /* Include the main project header */
// #include "project.h"
//
// G_BEGIN_DECLS
// /*** END file-header ***/
//
// /*** BEGIN file-production ***/
//
// /* enumerations from "@filename@" */
// /*** END file-production ***/
//
// /*** BEGIN value-header ***/
// GType @enum_name@_get_type (void) G_GNUC_CONST;
// #define @ENUMPREFIX@_TYPE_@ENUMSHORT@ (@enum_name@_get_type ())
// /*** END value-header ***/
//
// /*** BEGIN file-tail ***/
// G_END_DECLS
// /*** END file-tail ***/

// Source: playerctl/playerctl-enum-types.c.in
// /*** BEGIN file-header ***/
// #include "config.h"
// #include "enum-types.h"
//
// /*** END file-header ***/
//
// /*** BEGIN file-production ***/
// /* enumerations from "@filename@" */
// /*** END file-production ***/
//
// /*** BEGIN value-header ***/
// GType
// @enum_name@_get_type (void)
// {
//   static volatile gsize g_@type@_type_id__volatile;
//
//   if (g_once_init_enter (&g_define_type_id__volatile))
//     {
//       static const G@Type@Value values[] = {
// /*** END value-header ***/
//
// /*** BEGIN value-production ***/
//             { @VALUENAME@, "@VALUENAME@", "@valuenick@" },
// /*** END value-production ***/
//
// /*** BEGIN value-tail ***/
//             { 0, NULL, NULL }
//       };
//
//       GType g_@type@_type_id =
//         g_@type@_register_static (g_intern_static_string ("@EnumName@"), values);
//
//       g_once_init_leave (&g_@type@_type_id__volatile, g_@type@_type_id);
//     }
//   return g_@type@_type_id__volatile;
// }
//
// /*** END value-tail ***/

// PlaybackStatus represents the playback status for a PlayerctlPlayer.
type PlaybackStatus int

const (
	PlaybackStatusPlaying PlaybackStatus = iota // Playing
	PlaybackStatusPaused                        // Paused
	PlaybackStatusStopped                       // Stopped
)

// String returns the string representation of the PlaybackStatus.
func (s PlaybackStatus) String() string {
	switch s {
	case PlaybackStatusPlaying:
		return "Playing"
	case PlaybackStatusPaused:
		return "Paused"
	case PlaybackStatusStopped:
		return "Stopped"
	default:
		return "Unknown"
	}
}

// LoopStatus represents the loop status for a PlayerctlPlayer.
type LoopStatus int

const (
	LoopStatusNone     LoopStatus = iota // The playback will stop when there are no more tracks to play.
	LoopStatusTrack                      // The current track will start again from the beginning once it has finished playing.
	LoopStatusPlaylist                   // The playlist will start again from the beginning once it has finished playing.
)

// String returns the string representation of the LoopStatus.
func (s LoopStatus) String() string {
	switch s {
	case LoopStatusNone:
		return "None"
	case LoopStatusTrack:
		return "Track"
	case LoopStatusPlaylist:
		return "Playlist"
	default:
		return "Unknown"
	}
}

// Source represents the source of the name used to control the player.
type Source int

const (
	SourceNone        Source = iota // Only for uninitialized players. Source will be chosen automatically.
	SourceDBusSession               // The player is on the DBus session bus.
	SourceDBusSystem                // The player is on the DBus system bus.
)

// String returns the string representation of the Source.
func (s Source) String() string {
	switch s {
	case SourceNone:
		return "None"
	case SourceDBusSession:
		return "DBusSession"
	case SourceDBusSystem:
		return "DBusSystem"
	default:
		return "Unknown"
	}
}
