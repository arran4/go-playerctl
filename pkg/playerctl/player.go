package playerctl

import (
	"fmt"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"
)

// Source: playerctl/playerctl-player.c
// /*
//  * This file is part of playerctl.
//  *
//  * playerctl is free software: you can redistribute it and/or modify it under
//  * the terms of the GNU Lesser General Public License as published by the Free
//  * Software Foundation, either version 3 of the License, or (at your option)
//  * any later version.
//  *
//  * playerctl is distributed in the hope that it will be useful, but WITHOUT ANY
//  * WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
//  * FOR A PARTICULAR PURPOSE.  See the GNU Lesser General Public License for
//  * more details.
//  *
//  * You should have received a copy of the GNU Lesser General Public License
//  * along with playerctl If not, see <http://www.gnu.org/licenses/>.
//  *
//  * Copyright © 2014, Tony Crisci and contributors.
//  */
//
// #include "playerctl-player.h"
//
// #include <gio/gio.h>
// #include <glib-object.h>
// #include <playerctl/playerctl-enum-types.h>
// #include <playerctl/playerctl-player-manager.h>
// #include <stdbool.h>
// #include <stdint.h>
// #include <string.h>
//
// #include "playerctl-common.h"
// #include "playerctl-generated.h"
//
// #define LENGTH(array) (sizeof array / sizeof array[0])
//
// #define MPRIS_PATH "/org/mpris/MediaPlayer2"
// #define PROPERTIES_IFACE "org.freedesktop.DBus.Properties"
// #define PLAYER_IFACE "org.mpris.MediaPlayer2.Player"
// #define SET_MEMBER "Set"

const (
	MprisPath       = "/org/mpris/MediaPlayer2"
	PropertiesIface = "org.freedesktop.DBus.Properties"
	PlayerIface     = "org.mpris.MediaPlayer2.Player"
	SetMember       = "Set"
)

type Player struct {
	Conn                    *dbus.Conn
	Name                    *PlayerName
	BusName                 string
	cachedStatus            PlaybackStatus
	cachedPosition          int64
	cachedTrackID           string
	cachedPositionMonotonic time.Time
	initted                 bool
}
//
// enum {
//     PROP_0,
//
//     PROP_PLAYER_NAME,
//     PROP_PLAYER_INSTANCE,
//     PROP_SOURCE,
//     PROP_PLAYBACK_STATUS,
//     PROP_LOOP_STATUS,
//     PROP_SHUFFLE,
//     PROP_STATUS,  // deprecated
//     PROP_VOLUME,
//     PROP_METADATA,
//     PROP_POSITION,
//
//     PROP_CAN_CONTROL,
//     PROP_CAN_PLAY,
//     PROP_CAN_PAUSE,
//     PROP_CAN_SEEK,
//     PROP_CAN_GO_NEXT,
//     PROP_CAN_GO_PREVIOUS,
//
//     N_PROPERTIES
// };
//
// enum {
//     // PROPERTIES_CHANGED,
//     PLAYBACK_STATUS,
//     LOOP_STATUS,
//     SHUFFLE,
//     PLAY,   // deprecated
//     PAUSE,  // deprecated
//     STOP,   // deprecated
//     METADATA,
//     VOLUME,
//     SEEKED,
//     EXIT,
//     LAST_SIGNAL
// };
//
// static GParamSpec *obj_properties[N_PROPERTIES] = {
//     NULL,
// };
//
// static guint connection_signals[LAST_SIGNAL] = {0};
//
// struct _PlayerctlPlayerPrivate {
//     OrgMprisMediaPlayer2Player *proxy;
//     gchar *player_name;
//     gchar *instance;
//     gchar *bus_name;
//     PlayerctlSource source;
//     GError *init_error;
//     gboolean initted;
//     PlayerctlPlaybackStatus cached_status;
//     gint64 cached_position;
//     gchar *cached_track_id;
//     struct timespec cached_position_monotonic;
// };
//
// static inline int64_t timespec_to_usec(const struct timespec *a) {
//     return (int64_t)a->tv_sec * 1e+6 + a->tv_nsec / 1000;
// }
//
// static gint64 calculate_cached_position(PlayerctlPlaybackStatus status,
//                                         struct timespec *position_monotonic, gint64 position) {
//     gint64 offset = 0;
//     struct timespec current_time;
//
//     switch (status) {
//     case PLAYERCTL_PLAYBACK_STATUS_PLAYING:
//         clock_gettime(CLOCK_MONOTONIC, &current_time);
//         offset = timespec_to_usec(&current_time) - timespec_to_usec(position_monotonic);
//         return position + offset;
//     case PLAYERCTL_PLAYBACK_STATUS_PAUSED:
//         return position;
//     default:
//         return 0;
//     }
// }
//
// static gchar *metadata_get_track_id(GVariant *metadata) {
//     GVariant *track_id_variant =
//         g_variant_lookup_value(metadata, "mpris:trackid", G_VARIANT_TYPE_OBJECT_PATH);
//     if (track_id_variant == NULL) {
//         // XXX some players set this as a string, which is against the protocol,
//         // but a lot of them do it and I don't feel like fixing it on all the
//         // players in the world.
//         g_debug("mpris:trackid is a string, not a D-Bus object reference");
//         track_id_variant = g_variant_lookup_value(metadata, "mpris:trackid", G_VARIANT_TYPE_STRING);
//     }
//
//     if (track_id_variant != NULL) {
//         const gchar *track_id = g_variant_get_string(track_id_variant, NULL);
//         g_variant_unref(track_id_variant);
//         return g_strdup(track_id);
//     }
//
//     return NULL;
// }
//
// static void playerctl_player_properties_changed_callback(GDBusProxy *_proxy,
//                                                          GVariant *changed_properties,
//                                                          const gchar *const *invalidated_properties,
//                                                          gpointer user_data) {
//     g_debug("%s", g_variant_print(changed_properties, TRUE));
//     PlayerctlPlayer *self = user_data;
//     gchar *instance = self->priv->instance;
//     g_debug("%s: properties changed", instance);
//
//     // TODO probably need to replace this with an iterator
//     GVariant *metadata = g_variant_lookup_value(changed_properties, "Metadata", NULL);
//     GVariant *playback_status = g_variant_lookup_value(changed_properties, "PlaybackStatus", NULL);
//     GVariant *loop_status = g_variant_lookup_value(changed_properties, "LoopStatus", NULL);
//     GVariant *volume = g_variant_lookup_value(changed_properties, "Volume", NULL);
//     GVariant *shuffle = g_variant_lookup_value(changed_properties, "Shuffle", NULL);
//
//     if (shuffle != NULL) {
//         gboolean shuffle_value = g_variant_get_boolean(shuffle);
//         g_debug("%s: shuffle value set to %s", instance, shuffle_value ? "true" : "false");
//         g_signal_emit(self, connection_signals[SHUFFLE], 0, shuffle_value);
//         g_variant_unref(shuffle);
//     }
//
//     if (volume != NULL) {
//         gdouble volume_value = g_variant_get_double(volume);
//         g_debug("%s: volume set to %f", instance, volume_value);
//         g_signal_emit(self, connection_signals[VOLUME], 0, volume_value);
//         g_variant_unref(volume);
//     }
//
//     gboolean track_id_invalidated = FALSE;
//     if (metadata != NULL) {
//         // update the cached track id
//         gchar *track_id = metadata_get_track_id(metadata);
//         if ((track_id == NULL && self->priv->cached_track_id != NULL) ||
//             (track_id != NULL && self->priv->cached_track_id == NULL) ||
//             (g_strcmp0(track_id, self->priv->cached_track_id) != 0)) {
//             g_free(self->priv->cached_track_id);
//             g_debug("%s: track id updated to %s", instance, track_id);
//             self->priv->cached_track_id = track_id;
//             track_id_invalidated = TRUE;
//         } else {
//             g_free(track_id);
//         }
//
//         g_debug("%s: metadata changed", instance);
//         // g_debug("metadata: %s", g_variant_print(metadata, TRUE));
//         g_signal_emit(self, connection_signals[METADATA], 0, metadata);
//         g_variant_unref(metadata);
//     }
//
//     if (track_id_invalidated) {
//         self->priv->cached_position = 0;
//         clock_gettime(CLOCK_MONOTONIC, &self->priv->cached_position_monotonic);
//     }
//
//     if (playback_status == NULL && track_id_invalidated) {
//         // XXX: Lots of player aren't setting status correctly when the track
//         // changes so we have to get it from the interface. We should
//         // definitely go fix this bug on the players.
//         g_debug("Playback status not set on track change; getting status from interface instead");
//         GVariant *call_reply = g_dbus_proxy_call_sync(
//             G_DBUS_PROXY(self->priv->proxy), "org.freedesktop.DBus.Properties.Get",
//             g_variant_new("(ss)", "org.mpris.MediaPlayer2.Player", "PlaybackStatus"),
//             G_DBUS_CALL_FLAGS_NONE, -1, NULL, NULL);
//
//         if (call_reply != NULL) {
//             GVariant *call_reply_box = g_variant_get_child_value(call_reply, 0);
//             playback_status = g_variant_get_child_value(call_reply_box, 0);
//             g_variant_unref(call_reply);
//             g_variant_unref(call_reply_box);
//         }
//     }
//
//     if (loop_status != NULL) {
//         const gchar *status_str = g_variant_get_string(loop_status, NULL);
//         PlayerctlLoopStatus status = 0;
//         GQuark quark = 0;
//         if (pctl_parse_loop_status(status_str, &status)) {
//             switch (status) {
//             case PLAYERCTL_LOOP_STATUS_TRACK:
//                 quark = g_quark_from_string("track");
//                 break;
//             case PLAYERCTL_LOOP_STATUS_PLAYLIST:
//                 quark = g_quark_from_string("playlist");
//                 break;
//             case PLAYERCTL_LOOP_STATUS_NONE:
//                 quark = g_quark_from_string("none");
//                 break;
//             }
//             g_debug("%s: loop status set to %s", instance, g_quark_to_string(quark));
//             g_signal_emit(self, connection_signals[LOOP_STATUS], quark, status);
//         }
//
//         g_variant_unref(loop_status);
//     }
//
//     if (playback_status != NULL) {
//         const gchar *status_str = g_variant_get_string(playback_status, NULL);
//         g_debug("%s: playback status set to %s", instance, status_str);
//         PlayerctlPlaybackStatus status = 0;
//         GQuark quark = 0;
//
//         if (pctl_parse_playback_status(status_str, &status)) {
//             switch (status) {
//             case PLAYERCTL_PLAYBACK_STATUS_PLAYING:
//                 quark = g_quark_from_string("playing");
//                 if (self->priv->cached_status != PLAYERCTL_PLAYBACK_STATUS_PLAYING) {
//                     clock_gettime(CLOCK_MONOTONIC, &self->priv->cached_position_monotonic);
//                 }
//                 g_signal_emit(self, connection_signals[PLAY], 0);
//                 break;
//             case PLAYERCTL_PLAYBACK_STATUS_PAUSED:
//                 quark = g_quark_from_string("paused");
//                 self->priv->cached_position = calculate_cached_position(
//                     self->priv->cached_status, &self->priv->cached_position_monotonic,
//                     self->priv->cached_position);
//                 // DEPRECATED
//                 g_signal_emit(self, connection_signals[PAUSE], 0);
//                 break;
//             case PLAYERCTL_PLAYBACK_STATUS_STOPPED:
//                 self->priv->cached_position = 0;
//                 quark = g_quark_from_string("stopped");
//                 // DEPRECATED
//                 g_signal_emit(self, connection_signals[STOP], 0);
//                 break;
//             }
//
//             if (self->priv->cached_status != status) {
//                 self->priv->cached_status = status;
//                 g_signal_emit(self, connection_signals[PLAYBACK_STATUS], quark, status);
//             }
//         } else {
//             g_debug("%s: got unknown playback state: %s", instance, status_str);
//         }
//
//         g_variant_unref(playback_status);
//     }
// }
//
// static void playerctl_player_seeked_callback(GDBusProxy *_proxy, gint64 position,
//                                              gpointer *user_data) {
//     PlayerctlPlayer *player = PLAYERCTL_PLAYER(user_data);
//     player->priv->cached_position = position;
//     g_debug("%s: new player position %ld", player->priv->instance, position);
//     clock_gettime(CLOCK_MONOTONIC, &player->priv->cached_position_monotonic);
//     g_signal_emit(player, connection_signals[SEEKED], 0, position);
// }
//
// static void playerctl_player_initable_iface_init(GInitableIface *iface);
//
// G_DEFINE_TYPE_WITH_CODE(PlayerctlPlayer, playerctl_player, G_TYPE_OBJECT,
//                         G_ADD_PRIVATE(PlayerctlPlayer)
//                             G_IMPLEMENT_INTERFACE(G_TYPE_INITABLE,
//                                                   playerctl_player_initable_iface_init));
//
// // clang-format off
// G_DEFINE_QUARK(playerctl-player-error-quark, playerctl_player_error);
// // clang-format on
//
// static GVariant *playerctl_player_get_metadata(PlayerctlPlayer *self, GError **err) {
//     GVariant *metadata;
//     GError *tmp_error = NULL;
//
//     metadata = org_mpris_media_player2_player_dup_metadata(self->priv->proxy);
//
//     if (!metadata) {
//         // XXX: Ugly spotify workaround. Spotify does not seem to use the property
//         // cache. We have to get the properties directly.
//         g_debug("Spotify does not use the D-Bus property cache, getting properties directly");
//         GVariant *call_reply = g_dbus_proxy_call_sync(
//             G_DBUS_PROXY(self->priv->proxy), "org.freedesktop.DBus.Properties.Get",
//             g_variant_new("(ss)", "org.mpris.MediaPlayer2.Player", "Metadata"),
//             G_DBUS_CALL_FLAGS_NONE, -1, NULL, &tmp_error);
//
//         if (tmp_error != NULL) {
//             g_propagate_error(err, tmp_error);
//             return NULL;
//         }
//
//         GVariant *call_reply_properties = g_variant_get_child_value(call_reply, 0);
//
//         metadata = g_variant_get_child_value(call_reply_properties, 0);
//
//         g_variant_unref(call_reply);
//         g_variant_unref(call_reply_properties);
//     }
//
//     return metadata;
// }
//
// static void playerctl_player_set_property(GObject *object, guint property_id, const GValue *value,
//                                           GParamSpec *pspec) {
//     PlayerctlPlayer *self = PLAYERCTL_PLAYER(object);
//
//     switch (property_id) {
//     case PROP_PLAYER_NAME:
//         g_free(self->priv->player_name);
//         self->priv->player_name = g_strdup(g_value_get_string(value));
//         break;
//
//     case PROP_PLAYER_INSTANCE:
//         g_free(self->priv->instance);
//         self->priv->instance = g_strdup(g_value_get_string(value));
//         break;
//
//     case PROP_SOURCE:
//         self->priv->source = g_value_get_enum(value);
//         break;
//
//     case PROP_VOLUME:
//         g_warning("setting the volume property directly is deprecated and will "
//                   "be removed in a future version. Use "
//                   "playerctl_player_set_volume() instead.");
//         org_mpris_media_player2_player_set_volume(self->priv->proxy, g_value_get_double(value));
//         break;
//
//     default:
//         G_OBJECT_WARN_INVALID_PROPERTY_ID(object, property_id, pspec);
//         break;
//     }
// }
//
// static void playerctl_player_get_property(GObject *object, guint property_id, GValue *value,
//                                           GParamSpec *pspec) {
//     PlayerctlPlayer *self = PLAYERCTL_PLAYER(object);
//
//     switch (property_id) {
//     case PROP_PLAYER_NAME:
//         g_value_set_string(value, self->priv->player_name);
//         break;
//
//     case PROP_PLAYER_INSTANCE:
//         g_value_set_string(value, self->priv->instance);
//         break;
//
//     case PROP_SOURCE:
//         g_value_set_enum(value, self->priv->source);
//         break;
//
//     case PROP_PLAYBACK_STATUS:
//         g_value_set_enum(value, self->priv->cached_status);
//         break;
//
//     case PROP_LOOP_STATUS: {
//         const gchar *status_str = org_mpris_media_player2_player_get_loop_status(self->priv->proxy);
//         PlayerctlLoopStatus status = 0;
//         if (pctl_parse_loop_status(status_str, &status)) {
//             g_value_set_enum(value, status);
//         } else {
//             if (status_str != NULL) {
//                 g_debug("got unknown loop status: %s", status_str);
//             }
//             g_value_set_enum(value, PLAYERCTL_LOOP_STATUS_NONE);
//         }
//         break;
//     }
//
//     case PROP_SHUFFLE: {
//         if (self->priv->proxy == NULL) {
//             g_value_set_boolean(value, FALSE);
//             break;
//         }
//         g_value_set_boolean(value, org_mpris_media_player2_player_get_shuffle(self->priv->proxy));
//         break;
//     }
//
//     case PROP_STATUS:
//         // DEPRECATED
//         if (self->priv->proxy) {
//             g_value_set_string(value, pctl_playback_status_to_string(self->priv->cached_status));
//         } else {
//             g_value_set_string(value, "");
//         }
//         break;
//
//     case PROP_METADATA: {
//         GError *error = NULL;
//         GVariant *metadata = NULL;
//         metadata = playerctl_player_get_metadata(self, &error);
//         if (error != NULL) {
//             g_error("could not get metadata: %s", error->message);
//             g_clear_error(&error);
//         }
//         g_value_set_variant(value, metadata);
//         break;
//     }
//
//     case PROP_VOLUME:
//         if (self->priv->proxy) {
//             g_value_set_double(value, org_mpris_media_player2_player_get_volume(self->priv->proxy));
//         } else {
//             g_value_set_double(value, 0);
//         }
//         break;
//
//     case PROP_POSITION: {
//         gint64 position = calculate_cached_position(self->priv->cached_status,
//                                                     &self->priv->cached_position_monotonic,
//                                                     self->priv->cached_position);
//         g_value_set_int64(value, position);
//         break;
//     }
//
//     case PROP_CAN_CONTROL:
//         if (self->priv->proxy == NULL) {
//             g_value_set_boolean(value, FALSE);
//             break;
//         }
//         g_value_set_boolean(value,
//                             org_mpris_media_player2_player_get_can_control(self->priv->proxy));
//         break;
//
//     case PROP_CAN_PLAY:
//         if (self->priv->proxy == NULL) {
//             g_value_set_boolean(value, FALSE);
//             break;
//         }
//         g_value_set_boolean(value, org_mpris_media_player2_player_get_can_play(self->priv->proxy));
//         break;
//
//     case PROP_CAN_PAUSE:
//         if (self->priv->proxy == NULL) {
//             g_value_set_boolean(value, FALSE);
//             break;
//         }
//         g_value_set_boolean(value, org_mpris_media_player2_player_get_can_pause(self->priv->proxy));
//         break;
//
//     case PROP_CAN_SEEK:
//         if (self->priv->proxy == NULL) {
//             g_value_set_boolean(value, FALSE);
//             break;
//         }
//         g_value_set_boolean(value, org_mpris_media_player2_player_get_can_seek(self->priv->proxy));
//         break;
//
//     case PROP_CAN_GO_NEXT:
//         if (self->priv->proxy == NULL) {
//             g_value_set_boolean(value, FALSE);
//             break;
//         }
//         g_value_set_boolean(value,
//                             org_mpris_media_player2_player_get_can_go_next(self->priv->proxy));
//         break;
//
//     case PROP_CAN_GO_PREVIOUS:
//         if (self->priv->proxy == NULL) {
//             g_value_set_boolean(value, FALSE);
//             break;
//         }
//         g_value_set_boolean(value,
//                             org_mpris_media_player2_player_get_can_go_previous(self->priv->proxy));
//         break;
//
//     default:
//         G_OBJECT_WARN_INVALID_PROPERTY_ID(object, property_id, pspec);
//         break;
//     }
// }
//
// static void playerctl_player_constructed(GObject *gobject) {
//     PlayerctlPlayer *self = PLAYERCTL_PLAYER(gobject);
//
//     self->priv->init_error = NULL;
//
//     g_initable_init((GInitable *)self, NULL, &self->priv->init_error);
//
//     G_OBJECT_CLASS(playerctl_player_parent_class)->constructed(gobject);
// }
//
// static void playerctl_player_dispose(GObject *gobject) {
//     PlayerctlPlayer *self = PLAYERCTL_PLAYER(gobject);
//
//     g_clear_error(&self->priv->init_error);
//     g_clear_object(&self->priv->proxy);
//
//     G_OBJECT_CLASS(playerctl_player_parent_class)->dispose(gobject);
// }
//
// static void playerctl_player_finalize(GObject *gobject) {
//     PlayerctlPlayer *self = PLAYERCTL_PLAYER(gobject);
//
//     g_free(self->priv->player_name);
//     g_free(self->priv->instance);
//     g_free(self->priv->cached_track_id);
//     g_free(self->priv->bus_name);
//
//     G_OBJECT_CLASS(playerctl_player_parent_class)->finalize(gobject);
// }
//
// static void playerctl_player_class_init(PlayerctlPlayerClass *klass) {
//     GObjectClass *gobject_class = G_OBJECT_CLASS(klass);
//
//     gobject_class->set_property = playerctl_player_set_property;
//     gobject_class->get_property = playerctl_player_get_property;
//     gobject_class->constructed = playerctl_player_constructed;
//     gobject_class->dispose = playerctl_player_dispose;
//     gobject_class->finalize = playerctl_player_finalize;
//
//     obj_properties[PROP_PLAYER_NAME] =
//         g_param_spec_string("player-name", "Player name",
//                             "The name of the type of player this is. "
//                             "The instance is fully qualified with the player-instance and the "
//                             "source.",
//                             NULL, /* default */
//                             G_PARAM_READWRITE | G_PARAM_CONSTRUCT_ONLY | G_PARAM_STATIC_STRINGS);
//
//     obj_properties[PROP_PLAYER_INSTANCE] =
//         g_param_spec_string("player-instance", "Player instance",
//                             "An instance name that identifies "
//                             "this player on the source",
//                             NULL, /* default */
//                             G_PARAM_READWRITE | G_PARAM_CONSTRUCT_ONLY | G_PARAM_STATIC_STRINGS);
//
//     obj_properties[PROP_SOURCE] =
//         g_param_spec_enum("source", "Player source",
//                           "The source of this player. Currently supported "
//                           "sources are the DBus session bus and DBus system bus.",
//                           playerctl_source_get_type(), G_BUS_TYPE_NONE,
//                           G_PARAM_CONSTRUCT_ONLY | G_PARAM_READWRITE | G_PARAM_STATIC_STRINGS);
//
//     obj_properties[PROP_PLAYBACK_STATUS] = g_param_spec_enum(
//         "playback-status", "Player playback status",
//         "Whether the player is playing, paused, or stopped", playerctl_playback_status_get_type(),
//         PLAYERCTL_PLAYBACK_STATUS_STOPPED, G_PARAM_READABLE | G_PARAM_STATIC_STRINGS);
//
//     obj_properties[PROP_LOOP_STATUS] =
//         g_param_spec_enum("loop-status", "Player loop status", "The loop status of the player",
//                           playerctl_loop_status_get_type(), PLAYERCTL_LOOP_STATUS_NONE,
//                           G_PARAM_READABLE | G_PARAM_STATIC_STRINGS);
//
//     obj_properties[PROP_SHUFFLE] = g_param_spec_boolean(
//         "shuffle", "Shuffle",
//         "A value of false indicates that playback is "
//         "progressing linearly through a playlist, while true means playback is "
//         "progressing through a playlist in some other order.",
//         FALSE, G_PARAM_READABLE | G_PARAM_STATIC_STRINGS);
//
//     /**
//      * PlayerctlPlayer:status:
//      *
//      * The playback status of the player as a string
//      *
//      * Deprecated:2.0.0: Use the "playback-status" signal instead.
//      */
//     obj_properties[PROP_STATUS] =
//         g_param_spec_string("status", "Player status",
//                             "The play status of the player (deprecated: use "
//                             "playback-status)",
//                             NULL, /* default */
//                             G_PARAM_READABLE | G_PARAM_STATIC_STRINGS | G_PARAM_DEPRECATED);
//
//     obj_properties[PROP_VOLUME] = g_param_spec_double(
//         "volume", "Player volume",
//         "The volume level of the player. Setting "
//         "this property directly is deprecated and this property will become read "
//         "only in a future version. Use playerctl_player_set_volume() to set the "
//         "volume.",
//         0, 100, 0, G_PARAM_READWRITE | G_PARAM_STATIC_STRINGS);
//
//     obj_properties[PROP_POSITION] =
//         g_param_spec_int64("position", "Player position",
//                            "The position in the current track of the player in microseconds", 0,
//                            INT64_MAX, 0, G_PARAM_READABLE | G_PARAM_STATIC_STRINGS);
//
//     obj_properties[PROP_METADATA] = g_param_spec_variant(
//         "metadata", "Player metadata",
//         "The metadata of the currently playing track as an array of key-value "
//         "pairs. The metadata available depends on the track, but may include the "
//         "artist, title, length, art url, and other metadata.",
//         g_variant_type_new("a{sv}"), NULL, G_PARAM_READABLE | G_PARAM_STATIC_STRINGS);
//
//     obj_properties[PROP_CAN_CONTROL] = g_param_spec_boolean(
//         "can-control", "Can control", "Whether the player can be controlled by playerctl", FALSE,
//         G_PARAM_READABLE | G_PARAM_STATIC_STRINGS);
//
//     obj_properties[PROP_CAN_PLAY] =
//         g_param_spec_boolean("can-play", "Can play",
//                              "Whether the player can start playing and has a "
//                              "current track.",
//                              FALSE, G_PARAM_READABLE | G_PARAM_STATIC_STRINGS);
//
//     obj_properties[PROP_CAN_PAUSE] =
//         g_param_spec_boolean("can-pause", "Can pause", "Whether the player can pause", FALSE,
//                              G_PARAM_READABLE | G_PARAM_STATIC_STRINGS);
//
//     obj_properties[PROP_CAN_SEEK] = g_param_spec_boolean(
//         "can-seek", "Can seek", "Whether the position of the player can be controlled", FALSE,
//         G_PARAM_READABLE | G_PARAM_STATIC_STRINGS);
//
//     obj_properties[PROP_CAN_GO_NEXT] = g_param_spec_boolean(
//         "can-go-next", "Can go next", "Whether the player can go to the next track", FALSE,
//         G_PARAM_READABLE | G_PARAM_STATIC_STRINGS);
//
//     obj_properties[PROP_CAN_GO_PREVIOUS] = g_param_spec_boolean(
//         "can-go-previous", "Can go previous", "Whether the player can go to the previous track",
//         FALSE, G_PARAM_READABLE | G_PARAM_STATIC_STRINGS);
//
//     g_object_class_install_properties(gobject_class, N_PROPERTIES, obj_properties);
//
//     /**
//      * PlayerctlPlayer::playback-status:
//      * @player: the player this event was emitted on
//      * @playback_status: the playback status of the player
//      *
//      * Emitted when the playback status changes. Detail will be "playing",
//      * "paused", or "stopped" which you can listen to by connecting to the
//      * "playback-status::[STATUS]" signal.
//      */
//     connection_signals[PLAYBACK_STATUS] =
//         g_signal_new("playback-status",                      /* signal_name */
//                      PLAYERCTL_TYPE_PLAYER,                  /* itype */
//                      G_SIGNAL_RUN_FIRST | G_SIGNAL_DETAILED, /* signal_flags */
//                      0,                                      /* class_offset */
//                      NULL,                                   /* accumulator */
//                      NULL,                                   /* accu_data */
//                      g_cclosure_marshal_VOID__ENUM,          /* c_marshaller */
//                      G_TYPE_NONE,                            /* return_type */
//                      1,                                      /* n_params */
//                      playerctl_playback_status_get_type());
//
//     /**
//      * PlayerctlPlayer::loop-status:
//      * @player: the player this event was emitted on
//      * @loop_status: the loop status of the player
//      *
//      * Emitted when the loop status changes.
//      */
//     connection_signals[LOOP_STATUS] =
//         g_signal_new("loop-status",                          /* signal_name */
//                      PLAYERCTL_TYPE_PLAYER,                  /* itype */
//                      G_SIGNAL_RUN_FIRST | G_SIGNAL_DETAILED, /* signal_flags */
//                      0,                                      /* class_offset */
//                      NULL,                                   /* accumulator */
//                      NULL,                                   /* accu_data */
//                      g_cclosure_marshal_VOID__ENUM,          /* c_marshaller */
//                      G_TYPE_NONE,                            /* return_type */
//                      1,                                      /* n_params */
//                      playerctl_loop_status_get_type());
//
//     /**
//      * PlayerctlPlayer::shuffle:
//      * @player: the player this event was emitted on
//      * @shuffle_status: the shuffle status of the player
//      *
//      * Emitted when the shuffle status changes.
//      */
//     connection_signals[SHUFFLE] = g_signal_new("shuffle",                        /* signal_name */
//                                                PLAYERCTL_TYPE_PLAYER,            /* itype */
//                                                G_SIGNAL_RUN_FIRST,               /* signal_flags */
//                                                0,                                /* class_offset */
//                                                NULL,                             /* accumulator */
//                                                NULL,                             /* accu_data */
//                                                g_cclosure_marshal_VOID__BOOLEAN, /* c_marshaller */
//                                                G_TYPE_NONE,                      /* return_type */
//                                                1,                                /* n_params */
//                                                G_TYPE_BOOLEAN);
//
//     /**
//      * PlayerctlPlayer::play:
//      * @player: the player this event was emitted on
//      *
//      * Emitted when the player begins to play.
//      *
//      * Deprecated:2.0.0: Use the "playback-status::playing" signal instead.
//      */
//     connection_signals[PLAY] =
//         g_signal_new("play",                                   /* signal_name */
//                      PLAYERCTL_TYPE_PLAYER,                    /* itype */
//                      G_SIGNAL_RUN_FIRST | G_SIGNAL_DEPRECATED, /* signal_flags */
//                      0,                                        /* class_offset */
//                      NULL,                                     /* accumulator */
//                      NULL,                                     /* accu_data */
//                      g_cclosure_marshal_VOID__VOID,            /* c_marshaller */
//                      G_TYPE_NONE,                              /* return_type */
//                      0);                                       /* n_params */
//
//     /**
//      * PlayerctlPlayer::pause:
//      * @player: the player this event was emitted on
//      *
//      * Emitted when the player pauses.
//      *
//      * Deprecated:2.0.0: Use the "playback-status::paused" signal instead.
//      */
//     connection_signals[PAUSE] =
//         g_signal_new("pause",                                  /* signal_name */
//                      PLAYERCTL_TYPE_PLAYER,                    /* itype */
//                      G_SIGNAL_RUN_FIRST | G_SIGNAL_DEPRECATED, /* signal_flags */
//                      0,                                        /* class_offset */
//                      NULL,                                     /* accumulator */
//                      NULL,                                     /* accu_data */
//                      g_cclosure_marshal_VOID__VOID,            /* c_marshaller */
//                      G_TYPE_NONE,                              /* return_type */
//                      0);                                       /* n_params */
//
//     /**
//      * PlayerctlPlayer::stop:
//      * @player: the player this event was emitted on
//      *
//      * Emitted when the player stops.
//      *
//      * Deprecated:2.0.0: Use the "playback-status::stopped" signal instead.
//      */
//     connection_signals[STOP] =
//         g_signal_new("stop",                                   /* signal_name */
//                      PLAYERCTL_TYPE_PLAYER,                    /* itype */
//                      G_SIGNAL_RUN_FIRST | G_SIGNAL_DEPRECATED, /* signal_flags */
//                      0,                                        /* class_offset */
//                      NULL,                                     /* accumulator */
//                      NULL,                                     /* accu_data */
//                      g_cclosure_marshal_VOID__VOID,            /* c_marshaller */
//                      G_TYPE_NONE,                              /* return_type */
//                      0);                                       /* n_params */
//
//     /**
//      * PlayerctlPlayer::metadata:
//      * @player: the player this event was emitted on
//      * @metadata: the metadata for the currently playing track.
//      *
//      * Emitted when the metadata for the currently playing track changes.
//      */
//     connection_signals[METADATA] = g_signal_new("metadata",                       /* signal_name */
//                                                 PLAYERCTL_TYPE_PLAYER,            /* itype */
//                                                 G_SIGNAL_RUN_FIRST,               /* signal_flags */
//                                                 0,                                /* class_offset */
//                                                 NULL,                             /* accumulator */
//                                                 NULL,                             /* accu_data */
//                                                 g_cclosure_marshal_VOID__VARIANT, /* c_marshaller */
//                                                 G_TYPE_NONE,                      /* return_type */
//                                                 1,                                /* n_params */
//                                                 G_TYPE_VARIANT);
//
//     /**
//      * PlayerctlPlayer::volume:
//      * @player: the player this event was emitted on
//      * @volume: the volume of the player from 0 to 100.
//      *
//      * Emitted when the volume of the player changes.
//      */
//     connection_signals[VOLUME] = g_signal_new("volume",                        /* signal_name */
//                                               PLAYERCTL_TYPE_PLAYER,           /* itype */
//                                               G_SIGNAL_RUN_FIRST,              /* signal_flags */
//                                               0,                               /* class_offset */
//                                               NULL,                            /* accumulator */
//                                               NULL,                            /* accu_data */
//                                               g_cclosure_marshal_VOID__DOUBLE, /* c_marshaller */
//                                               G_TYPE_NONE,                     /* return_type */
//                                               1,                               /* n_params */
//                                               G_TYPE_DOUBLE);
//
//     /**
//      * PlayerctlPlayer::seeked:
//      * @player: the player this event was emitted on.
//      * @position: the new position in the track in microseconds.
//      *
//      * Emitted when the track changes position unexpectedly or begins in a
//      * position other than the beginning. Otherwise, position is assumed to
//      * progress normally.
//      */
//     connection_signals[SEEKED] = g_signal_new("seeked",                      /* signal_name */
//                                               PLAYERCTL_TYPE_PLAYER,         /* itype */
//                                               G_SIGNAL_RUN_FIRST,            /* signal_flags */
//                                               0,                             /* class_offset */
//                                               NULL,                          /* accumulator */
//                                               NULL,                          /* accu_data */
//                                               g_cclosure_marshal_VOID__LONG, /* c_marshaller */
//                                               G_TYPE_NONE,                   /* return_type */
//                                               1,                             /* n_params */
//                                               G_TYPE_INT64);
//
//     /**
//      * PlayerctlPlayer::exit:
//      * @player: the player this event was emitted on.
//      *
//      * Emitted when the player has disconnected and will no longer respond to
//      * queries and commands.
//      */
//     connection_signals[EXIT] = g_signal_new("exit",                        /* signal_name */
//                                             PLAYERCTL_TYPE_PLAYER,         /* itype */
//                                             G_SIGNAL_RUN_FIRST,            /* signal_flags */
//                                             0,                             /* class_offset */
//                                             NULL,                          /* accumulator */
//                                             NULL,                          /* accu_data */
//                                             g_cclosure_marshal_VOID__VOID, /* c_marshaller */
//                                             G_TYPE_NONE,                   /* return_type */
//                                             0);                            /* n_params */
// }
//
// static void playerctl_player_init(PlayerctlPlayer *self) {
//     self->priv = playerctl_player_get_instance_private(self);
// }
//
// /*
//  * Get the matching bus name for this player name. Bus name will be like:
//  * "org.mpris.MediaPlayer2.{PLAYER_NAME}[.{INSTANCE}]"
//  * Pass a NULL player_name to get the first name on the bus
//  * Returns NULL if no matching bus name is found on the bus.
//  * Returns an error if there was a problem listing the names on the bus.
//  */
// static gchar *bus_name_for_player_name(gchar *name, GBusType bus_type, GError **err) {
//     gchar *bus_name = NULL;
//     GError *tmp_error = NULL;
//
//     g_return_val_if_fail(err == NULL || *err == NULL, FALSE);
//
//     GList *names = pctl_list_player_names_on_bus(bus_type, &tmp_error);
//     if (tmp_error != NULL) {
//         g_propagate_error(err, tmp_error);
//         return NULL;
//     }
//
//     if (names == NULL) {
//         return NULL;
//     }
//
//     if (name == NULL) {
//         g_debug("Getting bus name for first available player");
//         PlayerctlPlayerName *name = names->data;
//         bus_name = g_strdup_printf(MPRIS_PREFIX "%s", name->instance);
//         pctl_player_name_list_destroy(names);
//         return bus_name;
//     }
//
//     GList *exact_match = pctl_player_name_find(names, name, pctl_bus_type_to_source(bus_type));
//     if (exact_match != NULL) {
//         g_debug("Getting bus name for player %s by exact match", name);
//         PlayerctlPlayerName *name = exact_match->data;
//         bus_name = g_strdup_printf(MPRIS_PREFIX "%s", name->instance);
//         g_list_free_full(names, (GDestroyNotify)playerctl_player_name_free);
//         return bus_name;
//     }
//
//     GList *instance_match =
//         pctl_player_name_find_instance(names, name, pctl_bus_type_to_source(bus_type));
//     if (instance_match != NULL) {
//         g_debug("Getting bus name for player %s by instance match", name);
//         gchar *name = instance_match->data;
//         bus_name = g_strdup_printf(MPRIS_PREFIX "%s", name);
//         pctl_player_name_list_destroy(names);
//         return bus_name;
//     }
//
//     return NULL;
// }
//
// static void playerctl_player_name_owner_changed_callback(GObject *object, GParamSpec *pspec,
//                                                          gpointer *user_data) {
//     PlayerctlPlayer *player = PLAYERCTL_PLAYER(user_data);
//     GDBusProxy *proxy = G_DBUS_PROXY(object);
//     char *name_owner = g_dbus_proxy_get_name_owner(proxy);
//
//     if (name_owner == NULL) {
//         g_signal_emit(player, connection_signals[EXIT], 0);
//     }
//
//     g_free(name_owner);
// }
//
// static gboolean playerctl_player_initable_init(GInitable *initable, GCancellable *cancellable,
//                                                GError **err) {
//     GError *tmp_error = NULL;
//     PlayerctlPlayer *player = PLAYERCTL_PLAYER(initable);
//
//     if (player->priv->initted) {
//         return TRUE;
//     }
//
//     g_return_val_if_fail(err == NULL || *err == NULL, FALSE);
//
//     if (player->priv->instance != NULL && player->priv->player_name != NULL) {
//         // if instance is specified, ignore name
//         g_free(player->priv->player_name);
//         player->priv->player_name = NULL;
//     }
//
//     if (player->priv->instance != NULL && player->priv->source == PLAYERCTL_SOURCE_NONE) {
//         g_set_error(err, playerctl_player_error_quark(), 3,
//                     "A player cannot be constructed with an instance and no source");
//         return FALSE;
//     }
//
//     gchar *bus_name = NULL;
//     if (player->priv->instance != NULL) {
//         bus_name = g_strdup_printf(MPRIS_PREFIX "%s", player->priv->instance);
//     } else if (player->priv->source != PLAYERCTL_SOURCE_NONE) {
//         // the source was specified
//         bus_name = bus_name_for_player_name(
//             player->priv->player_name, pctl_source_to_bus_type(player->priv->source), &tmp_error);
//         if (tmp_error) {
//             g_propagate_error(err, tmp_error);
//             return FALSE;
//         }
//     } else {
//         // the source was not specified
//         const GBusType bus_types[] = {G_BUS_TYPE_SESSION, G_BUS_TYPE_SYSTEM};
//         for (int i = 0; i < LENGTH(bus_types); ++i) {
//             bus_name =
//                 bus_name_for_player_name(player->priv->player_name, bus_types[i], &tmp_error);
//             if (tmp_error != NULL) {
//                 if (tmp_error->domain == G_IO_ERROR && tmp_error->code == G_IO_ERROR_NOT_FOUND) {
//                     g_debug("Bus address set incorrectly, cannot get bus");
//                     g_clear_error(&tmp_error);
//                     continue;
//                 }
//                 g_propagate_error(err, tmp_error);
//                 return FALSE;
//             }
//             if (bus_name != NULL) {
//                 player->priv->source = pctl_bus_type_to_source(bus_types[i]);
//                 break;
//             }
//         }
//     }
//
//     if (bus_name == NULL) {
//         g_set_error(err, playerctl_player_error_quark(), 1, "Player not found");
//         return FALSE;
//     }
//     player->priv->bus_name = bus_name;
//
//     /* org.mpris.MediaPlayer2.{NAME}[.{INSTANCE}] */
//     int offset = strlen(MPRIS_PREFIX);
//     gchar **split = g_strsplit(bus_name + offset, ".", 2);
//     g_free(player->priv->player_name);
//     player->priv->player_name = g_strdup(split[0]);
//     g_strfreev(split);
//
//     player->priv->proxy = org_mpris_media_player2_player_proxy_new_for_bus_sync(
//         pctl_source_to_bus_type(player->priv->source), G_DBUS_PROXY_FLAGS_NONE, bus_name,
//         "/org/mpris/MediaPlayer2", NULL, &tmp_error);
//     if (tmp_error != NULL) {
//         g_propagate_error(err, tmp_error);
//         return FALSE;
//     }
//
//     // init the cache
//     g_debug("initializing player: %s", player->priv->instance);
//     player->priv->cached_position =
//         org_mpris_media_player2_player_get_position(player->priv->proxy);
//     clock_gettime(CLOCK_MONOTONIC, &player->priv->cached_position_monotonic);
//
//     const gchar *playback_status_str =
//         org_mpris_media_player2_player_get_playback_status(player->priv->proxy);
//
//     PlayerctlPlaybackStatus status = 0;
//     if (pctl_parse_playback_status(playback_status_str, &status)) {
//         player->priv->cached_status = status;
//     }
//
//     g_signal_connect(player->priv->proxy, "g-properties-changed",
//                      G_CALLBACK(playerctl_player_properties_changed_callback), player);
//
//     g_signal_connect(player->priv->proxy, "seeked", G_CALLBACK(playerctl_player_seeked_callback),
//                      player);
//
//     g_signal_connect(player->priv->proxy, "notify::g-name-owner",
//                      G_CALLBACK(playerctl_player_name_owner_changed_callback), player);
//
//     player->priv->initted = TRUE;
//     return TRUE;
// }
//
// static void playerctl_player_initable_iface_init(GInitableIface *iface) {
//     iface->init = playerctl_player_initable_init;
// }
//
// /**
//  * playerctl_list_players:
//  * @err: The location of a GError or NULL
//  *
//  * Lists all the players that can be controlled by Playerctl.
//  *
//  * Returns:(transfer full) (element-type PlayerctlPlayerName): A list of player names.
//  */
// GList *playerctl_list_players(GError **err) {
//     GError *tmp_error = NULL;
//
//     g_return_val_if_fail(err == NULL || *err == NULL, NULL);
//
//     GList *session_players = pctl_list_player_names_on_bus(G_BUS_TYPE_SESSION, &tmp_error);
//     if (tmp_error != NULL) {
//         g_propagate_error(err, tmp_error);
//         return NULL;
//     }
//
//     GList *system_players = pctl_list_player_names_on_bus(G_BUS_TYPE_SYSTEM, &tmp_error);
//     if (tmp_error != NULL) {
//         g_propagate_error(err, tmp_error);
//         return NULL;
//     }
//
//     GList *players = g_list_concat(session_players, system_players);
//
//     return players;
// }
//
// /**
//  * playerctl_player_new:
//  * @player_name:(allow-none): The name to use to find the bus name of the player
//  * @err: The location of a GError or NULL
//  *
//  * Allocates a new #PlayerctlPlayer and tries to connect to an instance of the
//  * player with the given name.
//  *
//  * Returns:(transfer full): A new #PlayerctlPlayer connected to an instance of
//  * the player or NULL if an error occurred
//  */
// PlayerctlPlayer *playerctl_player_new(const gchar *player_name, GError **err) {
//     GError *tmp_error = NULL;
//     PlayerctlPlayer *player;
//
//     player =
//         g_initable_new(PLAYERCTL_TYPE_PLAYER, NULL, &tmp_error, "player-name", player_name, NULL);
//
//     if (tmp_error != NULL) {
//         g_propagate_error(err, tmp_error);
//         return NULL;
//     }
//
//     return player;
// }

func (p *Player) handleSignals(c chan *dbus.Signal) {
	for v := range c {
		if v.Path != dbus.ObjectPath(MprisPath) {
			continue
		}

		if v.Name == PropertiesIface+".PropertiesChanged" {
			if len(v.Body) >= 2 {
				changedProperties, ok := v.Body[1].(map[string]dbus.Variant)
				if !ok {
					continue
				}

				if statusVar, ok := changedProperties["PlaybackStatus"]; ok {
					if statusStr, ok := statusVar.Value().(string); ok {
						if status, parsed := ParsePlaybackStatus(statusStr); parsed {
							p.cachedStatus = status
						}
					}
				}

				trackIDInvalidated := false
				if metadataVar, ok := changedProperties["Metadata"]; ok {
					if metadata, ok := metadataVar.Value().(map[string]dbus.Variant); ok {
						trackID := GetTrackID(metadata)
						if trackID != p.cachedTrackID {
							p.cachedTrackID = trackID
							trackIDInvalidated = true
						}
					}
				}

				if trackIDInvalidated {
					p.cachedPosition = 0
					p.cachedPositionMonotonic = time.Now()
				}
			}
		} else if v.Name == PlayerIface+".Seeked" {
			if len(v.Body) > 0 {
				if position, ok := v.Body[0].(int64); ok {
					p.cachedPosition = position
					p.cachedPositionMonotonic = time.Now()
				}
			}
		}
	}
}

// GetArtist returns the artist from the metadata, joining multiple artists with a comma.
func GetArtist(metadata map[string]dbus.Variant) string {
	if variant, ok := metadata["xesam:artist"]; ok {
		if artists, ok := variant.Value().([]string); ok && len(artists) > 0 {
			return strings.Join(artists, ", ")
		}
	}
	return ""
}

// GetTitle returns the title from the metadata.
func GetTitle(metadata map[string]dbus.Variant) string {
	if variant, ok := metadata["xesam:title"]; ok {
		if title, ok := variant.Value().(string); ok {
			return title
		}
	}
	return ""
}

// GetAlbum returns the album from the metadata.
func GetAlbum(metadata map[string]dbus.Variant) string {
	if variant, ok := metadata["xesam:album"]; ok {
		if album, ok := variant.Value().(string); ok {
			return album
		}
	}
	return ""
}

// GetTrackID returns the track ID from the metadata.
func GetTrackID(metadata map[string]dbus.Variant) string {
	if variant, ok := metadata["mpris:trackid"]; ok {
		if val, ok := variant.Value().(dbus.ObjectPath); ok {
			return string(val)
		}
		if val, ok := variant.Value().(string); ok {
			return val
		}
	}
	return ""
}

// subscribeToSignals sets up DBus signal matching.
func (p *Player) subscribeToSignals() error {
	matchRule := fmt.Sprintf("type='signal',sender='%s',path='%s',interface='%s'", p.BusName, MprisPath, PropertiesIface)
	call := p.Conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, matchRule)
	if call.Err != nil {
		return fmt.Errorf("failed to add match signal for PropertiesChanged: %w", call.Err)
	}

	seekedMatchRule := fmt.Sprintf("type='signal',sender='%s',path='%s',interface='%s'", p.BusName, MprisPath, PlayerIface)
	call = p.Conn.BusObject().Call("org.freedesktop.DBus.AddMatch", 0, seekedMatchRule)
	if call.Err != nil {
		return fmt.Errorf("failed to add match signal for Seeked: %w", call.Err)
	}

	c := make(chan *dbus.Signal, 10)
	p.Conn.Signal(c)
	go p.handleSignals(c)

	return nil
}

// PlaybackStatus returns the playback status of the player.
func (p *Player) PlaybackStatus() (PlaybackStatus, error) {
	variant, err := p.Conn.Object(p.BusName, MprisPath).GetProperty(PlayerIface + ".PlaybackStatus")
	if err != nil {
		return PlaybackStatusStopped, err
	}
	statusStr, ok := variant.Value().(string)
	if !ok {
		return PlaybackStatusStopped, fmt.Errorf("unexpected type for PlaybackStatus: %T", variant.Value())
	}
	status, parsed := ParsePlaybackStatus(statusStr)
	if !parsed {
		return PlaybackStatusStopped, fmt.Errorf("failed to parse PlaybackStatus: %s", statusStr)
	}
	p.cachedStatus = status
	return status, nil
}

// LoopStatus returns the loop status of the player.
func (p *Player) LoopStatus() (LoopStatus, error) {
	variant, err := p.Conn.Object(p.BusName, MprisPath).GetProperty(PlayerIface + ".LoopStatus")
	if err != nil {
		return LoopStatusNone, err
	}
	statusStr, ok := variant.Value().(string)
	if !ok {
		return LoopStatusNone, fmt.Errorf("unexpected type for LoopStatus: %T", variant.Value())
	}
	status, parsed := ParseLoopStatus(statusStr)
	if !parsed {
		return LoopStatusNone, fmt.Errorf("failed to parse LoopStatus: %s", statusStr)
	}
	return status, nil
}

// Volume returns the volume of the player.
func (p *Player) Volume() (float64, error) {
	variant, err := p.Conn.Object(p.BusName, MprisPath).GetProperty(PlayerIface + ".Volume")
	if err != nil {
		return 0, err
	}
	volume, ok := variant.Value().(float64)
	if !ok {
		return 0, fmt.Errorf("unexpected type for Volume: %T", variant.Value())
	}
	return volume, nil
}

// Metadata returns the metadata of the player.
func (p *Player) Metadata() (map[string]dbus.Variant, error) {
	variant, err := p.Conn.Object(p.BusName, MprisPath).GetProperty(PlayerIface + ".Metadata")
	if err != nil {
		return nil, err
	}
	metadata, ok := variant.Value().(map[string]dbus.Variant)
	if !ok {
		return nil, fmt.Errorf("unexpected type for Metadata: %T", variant.Value())
	}
	return metadata, nil
}

// Position returns the current position of the player in microseconds.
func (p *Player) Position() (int64, error) {
	variant, err := p.Conn.Object(p.BusName, MprisPath).GetProperty(PlayerIface + ".Position")
	if err != nil {
		return 0, err
	}
	position, ok := variant.Value().(int64)
	if !ok {
		return 0, fmt.Errorf("unexpected type for Position: %T", variant.Value())
	}

	status, err := p.PlaybackStatus()
	if err != nil {
		return position, err
	}

	p.cachedPosition = position
	p.cachedPositionMonotonic = time.Now()
	return p.calculateCachedPosition(status, p.cachedPositionMonotonic, position), nil
}

func (p *Player) calculateCachedPosition(status PlaybackStatus, positionMonotonic time.Time, position int64) int64 {
	switch status {
	case PlaybackStatusPlaying:
		offset := time.Since(positionMonotonic).Microseconds()
		return position + offset
	case PlaybackStatusPaused:
		return position
	default:
		return 0
	}
}

// getBoolProperty is a helper to fetch a boolean property.
func (p *Player) getBoolProperty(propName string) (bool, error) {
	variant, err := p.Conn.Object(p.BusName, MprisPath).GetProperty(PlayerIface + "." + propName)
	if err != nil {
		return false, err
	}
	val, ok := variant.Value().(bool)
	if !ok {
		return false, fmt.Errorf("unexpected type for %s: %T", propName, variant.Value())
	}
	return val, nil
}

// CanControl returns whether the player can be controlled.
func (p *Player) CanControl() (bool, error) {
	return p.getBoolProperty("CanControl")
}

// CanPlay returns whether the player can start playing.
func (p *Player) CanPlay() (bool, error) {
	return p.getBoolProperty("CanPlay")
}

// CanPause returns whether the player can be paused.
func (p *Player) CanPause() (bool, error) {
	return p.getBoolProperty("CanPause")
}

// CanSeek returns whether the player supports seeking.
func (p *Player) CanSeek() (bool, error) {
	return p.getBoolProperty("CanSeek")
}

// CanGoNext returns whether the player supports going to the next track.
func (p *Player) CanGoNext() (bool, error) {
	return p.getBoolProperty("CanGoNext")
}

// CanGoPrevious returns whether the player supports going to the previous track.
func (p *Player) CanGoPrevious() (bool, error) {
	return p.getBoolProperty("CanGoPrevious")
}

// Shuffle returns the shuffle status of the player.
func (p *Player) Shuffle() (bool, error) {
	variant, err := p.Conn.Object(p.BusName, MprisPath).GetProperty(PlayerIface + ".Shuffle")
	if err != nil {
		return false, err
	}
	shuffle, ok := variant.Value().(bool)
	if !ok {
		return false, fmt.Errorf("unexpected type for Shuffle: %T", variant.Value())
	}
	return shuffle, nil
}

// Play commands the player to start playing.
func (p *Player) Play() error {
	call := p.Conn.Object(p.BusName, MprisPath).Call(PlayerIface+".Play", 0)
	return call.Err
}

// Pause commands the player to pause playback.
func (p *Player) Pause() error {
	call := p.Conn.Object(p.BusName, MprisPath).Call(PlayerIface+".Pause", 0)
	return call.Err
}

// PlayPause commands the player to toggle playback status.
func (p *Player) PlayPause() error {
	call := p.Conn.Object(p.BusName, MprisPath).Call(PlayerIface+".PlayPause", 0)
	return call.Err
}

// Stop commands the player to stop playback.
func (p *Player) Stop() error {
	call := p.Conn.Object(p.BusName, MprisPath).Call(PlayerIface+".Stop", 0)
	return call.Err
}

// Next commands the player to skip to the next track.
func (p *Player) Next() error {
	call := p.Conn.Object(p.BusName, MprisPath).Call(PlayerIface+".Next", 0)
	return call.Err
}

// Previous commands the player to skip to the previous track.
func (p *Player) Previous() error {
	call := p.Conn.Object(p.BusName, MprisPath).Call(PlayerIface+".Previous", 0)
	return call.Err
}

// Seek commands the player to seek by the given offset (in microseconds).
func (p *Player) Seek(offset int64) error {
	call := p.Conn.Object(p.BusName, MprisPath).Call(PlayerIface+".Seek", 0, offset)
	return call.Err
}

// SetPosition commands the player to set the playback position of a given track.
func (p *Player) SetPosition(trackID dbus.ObjectPath, position int64) error {
	call := p.Conn.Object(p.BusName, MprisPath).Call(PlayerIface+".SetPosition", 0, trackID, position)
	return call.Err
}

// SetVolume sets the volume of the player.
func (p *Player) SetVolume(volume float64) error {
	call := p.Conn.Object(p.BusName, MprisPath).Call(PropertiesIface+"."+SetMember, 0, PlayerIface, "Volume", dbus.MakeVariant(volume))
	return call.Err
}

// SetLoopStatus sets the loop status of the player.
func (p *Player) SetLoopStatus(status LoopStatus) error {
	call := p.Conn.Object(p.BusName, MprisPath).Call(PropertiesIface+"."+SetMember, 0, PlayerIface, "LoopStatus", dbus.MakeVariant(status.String()))
	return call.Err
}

// SetShuffle sets the shuffle status of the player.
func (p *Player) SetShuffle(shuffle bool) error {
	call := p.Conn.Object(p.BusName, MprisPath).Call(PropertiesIface+"."+SetMember, 0, PlayerIface, "Shuffle", dbus.MakeVariant(shuffle))
	return call.Err
}

// OpenUri opens a given URI in the player.
func (p *Player) OpenUri(uri string) error {
	call := p.Conn.Object(p.BusName, MprisPath).Call(PlayerIface+".OpenUri", 0, uri)
	return call.Err
}

// NewPlayerFromName creates a new Player from a PlayerName.
// It establishes a connection to DBus and initializes the player state.
func NewPlayerFromName(name *PlayerName) (*Player, error) {
	var conn *dbus.Conn
	var err error

	if name.Source == SourceDBusSystem {
		conn, err = dbus.SystemBus()
	} else {
		conn, err = dbus.SessionBus()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to DBus: %w", err)
	}

	player := &Player{
		Conn:    conn,
		Name:    name,
		BusName: name.Instance,
	}

	err = player.subscribeToSignals()
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to signals: %w", err)
	}

	player.initted = true
	return player, nil
}
//
// /**
//  * playerctl_player_new_for_source:
//  * @player_name:(allow-none): The name to use to find the bus name of the player
//  * @source: The source where the player name is.
//  * @err: The location of a GError or NULL
//  *
//  * Allocates a new #PlayerctlPlayer and tries to connect to an instance of the
//  * player with the given name from the given source.
//  *
//  * Returns:(transfer full): A new #PlayerctlPlayer connected to an instance of
//  * the player or NULL if an error occurred
//  */
// PlayerctlPlayer *playerctl_player_new_for_source(const gchar *player_name, PlayerctlSource source,
//                                                  GError **err) {
//     GError *tmp_error = NULL;
//     PlayerctlPlayer *player;
//
//     player = g_initable_new(PLAYERCTL_TYPE_PLAYER, NULL, &tmp_error, "player-name", player_name,
//                             "source", source, NULL);
//
//     if (tmp_error != NULL) {
//         g_propagate_error(err, tmp_error);
//         return NULL;
//     }
//
//     return player;
// }
//
// /**
//  * playerctl_player_new_from_name:
//  * @player_name: The name type to use to find the player
//  * @err:(allow-none): The location of a GError or NULL
//  *
//  * Allocates a new #PlayerctlPlayer and tries to connect to the player
//  * identified by the #PlayerctlPlayerName.
//  *
//  * Returns:(transfer full): A new #PlayerctlPlayer connected to the player or
//  * NULL if an error occurred
//  */
// PlayerctlPlayer *playerctl_player_new_from_name(PlayerctlPlayerName *player_name, GError **err) {
//     GError *tmp_error = NULL;
//     PlayerctlPlayer *player;
//
//     player = g_initable_new(PLAYERCTL_TYPE_PLAYER, NULL, &tmp_error, "player-instance",
//                             player_name->instance, "source", player_name->source, NULL);
//
//     if (tmp_error != NULL) {
//         g_propagate_error(err, tmp_error);
//         return NULL;
//     }
//
//     return player;
// }
//
// /**
//  * playerctl_player_on:
//  * @self: a #PlayerctlPlayer
//  * @event: the event to subscribe to
//  * @callback: the callback to run on the event
//  * @err:(allow-none): the location of a GError or NULL
//  *
//  * A convenience function for bindings to subscribe to an event with a callback
//  *
//  * Deprecated:2.0.0: Use g_object_connect() to listen to events.
//  */
// void playerctl_player_on(PlayerctlPlayer *self, const gchar *event, GClosure *callback,
//                          GError **err) {
//     g_return_if_fail(self != NULL);
//     g_return_if_fail(event != NULL);
//     g_return_if_fail(callback != NULL);
//     g_return_if_fail(err == NULL || *err == NULL);
//
//     if (self->priv->init_error != NULL) {
//         g_propagate_error(err, g_error_copy(self->priv->init_error));
//         return;
//     }
//
//     g_closure_ref(callback);
//     g_closure_sink(callback);
//
//     g_signal_connect_closure(self, event, callback, TRUE);
//
//     return;
// }
//
// #define PLAYER_COMMAND_FUNC(COMMAND)                                                           \
//     GError *tmp_error = NULL;                                                                  \
//                                                                                                \
//     g_return_if_fail(self != NULL);                                                            \
//     g_return_if_fail(err == NULL || *err == NULL);                                             \
//                                                                                                \
//     if (self->priv->init_error != NULL) {                                                      \
//         g_propagate_error(err, g_error_copy(self->priv->init_error));                          \
//         return;                                                                                \
//     }                                                                                          \
//                                                                                                \
//     org_mpris_media_player2_player_call_##COMMAND##_sync(self->priv->proxy, NULL, &tmp_error); \
//                                                                                                \
//     if (tmp_error != NULL) {                                                                   \
//         g_propagate_error(err, tmp_error);                                                     \
//     }
//
// /**
//  * playerctl_player_play_pause:
//  * @self: a #PlayerctlPlayer
//  * @err:(allow-none): the location of a GError or NULL
//  *
//  * Command the player to play if it is paused or pause if it is playing
//  */
// void playerctl_player_play_pause(PlayerctlPlayer *self, GError **err) {
//     PLAYER_COMMAND_FUNC(play_pause);
// }
//
// /**
//  * playerctl_player_open:
//  * @self: a #PlayerctlPlayer
//  * @uri: the URI to open, either a file name or an external URL
//  * @err:(allow-none): the location of a GError or NULL
//  *
//  * Command the player to open given URI
//  */
// void playerctl_player_open(PlayerctlPlayer *self, gchar *uri, GError **err) {
//     GError *tmp_error = NULL;
//
//     g_return_if_fail(self != NULL);
//     g_return_if_fail(err == NULL || *err == NULL);
//
//     if (self->priv->init_error != NULL) {
//         g_propagate_error(err, g_error_copy(self->priv->init_error));
//         return;
//     }
//     org_mpris_media_player2_player_call_open_uri_sync(self->priv->proxy, uri, NULL, &tmp_error);
//
//     if (tmp_error != NULL) {
//         g_propagate_error(err, tmp_error);
//         return;
//     }
//
//     return;
// }
//
// /**
//  * playerctl_player_play:
//  * @self: a #PlayerctlPlayer
//  * @err:(allow-none): the location of a GError or NULL
//  *
//  * Command the player to play
//  */
// void playerctl_player_play(PlayerctlPlayer *self, GError **err) {
//     PLAYER_COMMAND_FUNC(play);
// }
//
// /**
//  * playerctl_player_pause:
//  * @self: a #PlayerctlPlayer
//  * @err:(allow-none): the location of a GError or NULL
//  *
//  * Command the player to pause
//  */
// void playerctl_player_pause(PlayerctlPlayer *self, GError **err) {
//     PLAYER_COMMAND_FUNC(pause);
// }
//
// /**
//  * playerctl_player_stop:
//  * @self: a #PlayerctlPlayer
//  * @err:(allow-none): the location of a GError or NULL
//  *
//  * Command the player to stop
//  */
// void playerctl_player_stop(PlayerctlPlayer *self, GError **err) {
//     PLAYER_COMMAND_FUNC(stop);
// }
//
// /**
//  * playerctl_player_seek:
//  * @self: a #PlayerctlPlayer
//  * @offset: the offset to seek forward to in microseconds
//  * @err:(allow-none): the location of a GError or NULL
//  *
//  * Command the player to seek forward by offset given in microseconds.
//  */
// void playerctl_player_seek(PlayerctlPlayer *self, gint64 offset, GError **err) {
//     GError *tmp_error = NULL;
//
//     g_return_if_fail(self != NULL);
//     g_return_if_fail(err == NULL || *err == NULL);
//
//     if (self->priv->init_error != NULL) {
//         g_propagate_error(err, g_error_copy(self->priv->init_error));
//         return;
//     }
//
//     org_mpris_media_player2_player_call_seek_sync(self->priv->proxy, offset, NULL, &tmp_error);
//
//     if (tmp_error != NULL) {
//         g_propagate_error(err, tmp_error);
//         return;
//     }
//
//     return;
// }
//
// /**
//  * playerctl_player_next:
//  * @self: a #PlayerctlPlayer
//  * @err:(allow-none): the location of a GError or NULL
//  *
//  * Command the player to go to the next track
//  */
// void playerctl_player_next(PlayerctlPlayer *self, GError **err) {
//     PLAYER_COMMAND_FUNC(next);
// }
//
// /**
//  * playerctl_player_previous:
//  * @self: a #PlayerctlPlayer
//  * @err:(allow-none): the location of a GError or NULL
//  *
//  * Command the player to go to the previous track
//  */
// void playerctl_player_previous(PlayerctlPlayer *self, GError **err) {
//     PLAYER_COMMAND_FUNC(previous);
// }
//
// static gchar *print_metadata_table(GVariant *metadata, gchar *player_name) {
//     GVariantIter iter;
//     GVariant *child;
//     GString *table = g_string_new("");
//     const gchar *fmt = "%-5s %-25s %s\n";
//
//     if (g_strcmp0(g_variant_get_type_string(metadata), "a{sv}") != 0) {
//         return NULL;
//     }
//
//     g_variant_iter_init(&iter, metadata);
//     while ((child = g_variant_iter_next_value(&iter))) {
//         GVariant *key_variant = g_variant_get_child_value(child, 0);
//         const gchar *key = g_variant_get_string(key_variant, 0);
//         GVariant *value_variant = g_variant_lookup_value(metadata, key, NULL);
//
//         if (g_variant_is_container(value_variant)) {
//             // only go depth 1
//             int len = g_variant_n_children(value_variant);
//             for (int i = 0; i < len; ++i) {
//                 GVariant *child_value = g_variant_get_child_value(value_variant, i);
//                 gchar *child_value_str = pctl_print_gvariant(child_value);
//                 g_string_append_printf(table, fmt, player_name, key, child_value_str);
//                 g_free(child_value_str);
//                 g_variant_unref(child_value);
//             }
//         } else {
//             gchar *value = pctl_print_gvariant(value_variant);
//             g_string_append_printf(table, fmt, player_name, key, value);
//             g_free(value);
//         }
//
//         g_variant_unref(child);
//         g_variant_unref(key_variant);
//         g_variant_unref(value_variant);
//     }
//
//     if (table->len == 0) {
//         g_string_free(table, TRUE);
//         return NULL;
//     }
//     // cut off the last newline
//     table = g_string_truncate(table, table->len - 1);
//
//     return g_string_free(table, FALSE);
// }
//
// /**
//  * playerctl_player_print_metadata_prop:
//  * @self: a #PlayerctlPlayer
//  * @property:(allow-none): the property from the metadata to print
//  * @err:(allow-none): the location of a GError or NULL
//  *
//  * Gets the given property from the metadata of the current track. If property
//  * is null, prints all the metadata properties. Returns NULL if no track is
//  * playing.
//  *
//  * Returns:(transfer full): The artist from the metadata of the current track
//  */
// gchar *playerctl_player_print_metadata_prop(PlayerctlPlayer *self, const gchar *property,
//                                             GError **err) {
//     GError *tmp_error = NULL;
//
//     g_return_val_if_fail(self != NULL, NULL);
//     g_return_val_if_fail(err == NULL || *err == NULL, NULL);
//
//     if (self->priv->init_error != NULL) {
//         g_propagate_error(err, g_error_copy(self->priv->init_error));
//         return NULL;
//     }
//
//     GVariant *metadata = playerctl_player_get_metadata(self, &tmp_error);
//
//     if (tmp_error != NULL) {
//         g_propagate_error(err, tmp_error);
//         return NULL;
//     }
//
//     if (!metadata) {
//         return NULL;
//     }
//
//     if (!property) {
//         gchar *res = print_metadata_table(metadata, self->priv->player_name);
//         g_variant_unref(metadata);
//         return res;
//     }
//
//     GVariant *prop_variant = g_variant_lookup_value(metadata, property, NULL);
//     g_variant_unref(metadata);
//
//     if (!prop_variant) {
//         return NULL;
//     }
//
//     gchar *prop = pctl_print_gvariant(prop_variant);
//     g_variant_unref(prop_variant);
//     return prop;
// }
//
// /**
//  * playerctl_player_get_artist:
//  * @self: a #PlayerctlPlayer
//  * @err:(allow-none): the location of a GError or NULL
//  *
//  * Gets the artist from the metadata of the current track, or NULL if no
//  * track is playing.
//  *
//  * Returns:(transfer full): The artist from the metadata of the current track
//  */
// gchar *playerctl_player_get_artist(PlayerctlPlayer *self, GError **err) {
//     g_return_val_if_fail(self != NULL, NULL);
//     g_return_val_if_fail(err == NULL || *err == NULL, NULL);
//
//     if (self->priv->init_error != NULL) {
//         g_propagate_error(err, g_error_copy(self->priv->init_error));
//         return NULL;
//     }
//
//     return playerctl_player_print_metadata_prop(self, "xesam:artist", NULL);
// }
//
// /**
//  * playerctl_player_get_title:
//  * @self: a #PlayerctlPlayer
//  * @err:(allow-none): the location of a GError or NULL
//  *
//  * Gets the title from the metadata of the current track, or NULL if
//  * no track is playing.
//  *
//  * Returns:(transfer full): The title from the metadata of the current track
//  */
// gchar *playerctl_player_get_title(PlayerctlPlayer *self, GError **err) {
//     g_return_val_if_fail(self != NULL, NULL);
//     g_return_val_if_fail(err == NULL || *err == NULL, NULL);
//
//     if (self->priv->init_error != NULL) {
//         g_propagate_error(err, g_error_copy(self->priv->init_error));
//         return NULL;
//     }
//
//     return playerctl_player_print_metadata_prop(self, "xesam:title", NULL);
// }
//
// /**
//  * playerctl_player_get_album:
//  * @self: a #PlayerctlPlayer
//  * @err:(allow-none): the location of a GError or NULL
//  *
//  * Gets the album from the metadata of the current track, or NULL if
//  * no track is playing.
//  *
//  * Returns:(transfer full): The album from the metadata of the current track
//  */
// gchar *playerctl_player_get_album(PlayerctlPlayer *self, GError **err) {
//     g_return_val_if_fail(self != NULL, NULL);
//     g_return_val_if_fail(err == NULL || *err == NULL, NULL);
//
//     if (self->priv->init_error != NULL) {
//         g_propagate_error(err, g_error_copy(self->priv->init_error));
//         return NULL;
//     }
//
//     return playerctl_player_print_metadata_prop(self, "xesam:album", NULL);
// }
//
// /**
//  * playerctl_player_set_volume
//  * @self: a #PlayerctlPlayer
//  * @volume: the volume level from 0.0 to 1.0
//  * @err:(allow-none): the location of a GError or NULL
//  *
//  * Sets the volume level for the player from 0.0 for no volume to 1.0 for
//  * maximum volume. Passing negative numbers should set the volume to 0.0.
//  */
// void playerctl_player_set_volume(PlayerctlPlayer *self, gdouble volume, GError **err) {
//     GError *tmp_error = NULL;
//
//     g_return_if_fail(self != NULL);
//     g_return_if_fail(err == NULL || *err == NULL);
//
//     if (self->priv->init_error != NULL) {
//         g_propagate_error(err, g_error_copy(self->priv->init_error));
//         return;
//     }
//
//     GDBusConnection *connection = g_bus_get_sync(G_BUS_TYPE_SESSION, NULL, &tmp_error);
//     if (tmp_error != NULL) {
//         g_propagate_error(err, tmp_error);
//         return;
//     }
//
//     GVariant *result = g_dbus_connection_call_sync(
//         connection, self->priv->bus_name, MPRIS_PATH, PROPERTIES_IFACE, SET_MEMBER,
//         g_variant_new("(ssv)", PLAYER_IFACE, "Volume", g_variant_new("d", volume)), NULL,
//         G_DBUS_CALL_FLAGS_NONE, -1, NULL, &tmp_error);
//     if (result != NULL) {
//         g_variant_unref(result);
//     }
//
//     if (tmp_error != NULL) {
//         g_propagate_error(err, tmp_error);
//         return;
//     }
// }
//
// /**
//  * playerctl_player_get_position
//  * @self: a #PlayerctlPlayer
//  * @err:(allow-none): the location of a GError or NULL
//  *
//  * Gets the position of the current track in microseconds ignoring the property
//  * cache.
//  */
// gint64 playerctl_player_get_position(PlayerctlPlayer *self, GError **err) {
//     GError *tmp_error = NULL;
//
//     g_return_val_if_fail(self != NULL, 0);
//     g_return_val_if_fail(err == NULL || *err == NULL, 0);
//
//     if (self->priv->init_error != NULL) {
//         g_propagate_error(err, g_error_copy(self->priv->init_error));
//         return 0;
//     }
//
//     GVariant *call_reply = g_dbus_proxy_call_sync(G_DBUS_PROXY(self->priv->proxy),
//                                                   "org.freedesktop.DBus.Properties.Get",
//                                                   g_variant_new("(ss)", PLAYER_IFACE, "Position"),
//                                                   G_DBUS_CALL_FLAGS_NONE, -1, NULL, &tmp_error);
//     if (tmp_error) {
//         g_propagate_error(err, tmp_error);
//         return 0;
//     }
//
//     GVariant *call_reply_properties = g_variant_get_child_value(call_reply, 0);
//     GVariant *call_reply_unboxed = g_variant_get_variant(call_reply_properties);
//
//     gint64 position = g_variant_get_int64(call_reply_unboxed);
//
//     g_variant_unref(call_reply);
//     g_variant_unref(call_reply_properties);
//     g_variant_unref(call_reply_unboxed);
//
//     return position;
// }
//
// /**
//  * playerctl_player_set_position
//  * @self: a #PlayerctlPlayer
//  * @position: The absolute position in the track to set as the position
//  * @err:(allow-none): the location of a GError or NULL
//  *
//  * Sets the absolute position of the current track to the given position in microseconds.
//  */
// void playerctl_player_set_position(PlayerctlPlayer *self, gint64 position, GError **err) {
//     GError *tmp_error = NULL;
//
//     g_return_if_fail(self != NULL);
//     g_return_if_fail(err == NULL || *err == NULL);
//
//     if (self->priv->init_error != NULL) {
//         g_propagate_error(err, g_error_copy(self->priv->init_error));
//         return;
//     }
//
//     // calling the function requires the track id
//     GVariant *metadata = playerctl_player_get_metadata(self, &tmp_error);
//     if (tmp_error != NULL) {
//         g_propagate_error(err, tmp_error);
//         return;
//     }
//
//     gchar *track_id = metadata_get_track_id(metadata);
//     g_variant_unref(metadata);
//
//     if (track_id == NULL) {
//         tmp_error = g_error_new(playerctl_player_error_quark(), 2,
//                                 "Could not get track id to set position");
//         g_propagate_error(err, tmp_error);
//         return;
//     }
//
//     org_mpris_media_player2_player_call_set_position_sync(self->priv->proxy, track_id, position,
//                                                           NULL, &tmp_error);
//     if (tmp_error != NULL) {
//         g_propagate_error(err, tmp_error);
//     }
// }
//
// /**
//  * playerctl_player_set_loop_status:
//  * @self: a #PlayerctlPlayer
//  * @status: the requested #PlayerctlLoopStatus to set the player to
//  * @err:(allow-none): the location of a GError or NULL
//  *
//  * Set the loop status of the player. Can be set to either None, Track, or Playlist.
//  */
// void playerctl_player_set_loop_status(PlayerctlPlayer *self, PlayerctlLoopStatus status,
//                                       GError **err) {
//     GError *tmp_error = NULL;
//     g_return_if_fail(self != NULL);
//     g_return_if_fail(err == NULL || *err == NULL);
//
//     if (self->priv->init_error != NULL) {
//         g_propagate_error(err, g_error_copy(self->priv->init_error));
//         return;
//     }
//
//     const gchar *status_str = pctl_loop_status_to_string(status);
//     g_return_if_fail(status_str != NULL);
//
//     GDBusConnection *connection = g_bus_get_sync(G_BUS_TYPE_SESSION, NULL, &tmp_error);
//     if (tmp_error != NULL) {
//         g_propagate_error(err, tmp_error);
//         return;
//     }
//
//     GVariant *result = g_dbus_connection_call_sync(
//         connection, self->priv->bus_name, MPRIS_PATH, PROPERTIES_IFACE, SET_MEMBER,
//         g_variant_new("(ssv)", PLAYER_IFACE, "LoopStatus", g_variant_new("s", status_str)), NULL,
//         G_DBUS_CALL_FLAGS_NONE, -1, NULL, &tmp_error);
//     if (result != NULL) {
//         g_variant_unref(result);
//     }
//
//     if (tmp_error != NULL) {
//         g_propagate_error(err, tmp_error);
//         return;
//     }
// }
//
// /**
//  * playerctl_player_set_shuffle:
//  * @self: a #PlayerctlPlayer
//  * @shuffle: whether to enable shuffle
//  * @err:(allow-none): the location of a GError or NULL
//  *
//  * Request to set the shuffle state of the player, either on or off.
//  */
// void playerctl_player_set_shuffle(PlayerctlPlayer *self, gboolean shuffle, GError **err) {
//     GError *tmp_error = NULL;
//     g_return_if_fail(self != NULL);
//     g_return_if_fail(err == NULL || *err == NULL);
//
//     if (self->priv->init_error != NULL) {
//         g_propagate_error(err, g_error_copy(self->priv->init_error));
//         return;
//     }
//
//     GDBusConnection *connection = g_bus_get_sync(G_BUS_TYPE_SESSION, NULL, &tmp_error);
//     if (tmp_error != NULL) {
//         g_propagate_error(err, tmp_error);
//         return;
//     }
//
//     GVariant *result = g_dbus_connection_call_sync(
//         connection, self->priv->bus_name, MPRIS_PATH, PROPERTIES_IFACE, SET_MEMBER,
//         g_variant_new("(ssv)", PLAYER_IFACE, "Shuffle", g_variant_new("b", shuffle)), NULL,
//         G_DBUS_CALL_FLAGS_NONE, -1, NULL, &tmp_error);
//     if (result != NULL) {
//         g_variant_unref(result);
//     }
//
//     if (tmp_error != NULL) {
//         g_propagate_error(err, tmp_error);
//         return;
//     }
// }
//
// char *pctl_player_get_instance(PlayerctlPlayer *player) {
//     return player->priv->instance;
// }
//
// bool pctl_player_has_cached_property(PlayerctlPlayer *player, const gchar *name) {
//     GVariant *value = g_dbus_proxy_get_cached_property(G_DBUS_PROXY(player->priv->proxy), name);
//     if (value == NULL) {
//         return false;
//     }
//     g_variant_unref(value);
//     return true;
// }

// Source: playerctl/playerctl-player.h
// /*
//  * This file is part of playerctl.
//  *
//  * playerctl is free software: you can redistribute it and/or modify it under
//  * the terms of the GNU Lesser General Public License as published by the Free
//  * Software Foundation, either version 3 of the License, or (at your option)
//  * any later version.
//  *
//  * playerctl is distributed in the hope that it will be useful, but WITHOUT ANY
//  * WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
//  * FOR A PARTICULAR PURPOSE.  See the GNU Lesser General Public License for
//  * more details.
//  *
//  * You should have received a copy of the GNU Lesser General Public License
//  * along with playerctl If not, see <http://www.gnu.org/licenses/>.
//  *
//  * Copyright © 2014, Tony Crisci and contributors
//  */
//
// #ifndef __PLAYERCTL_PLAYER_H__
// #define __PLAYERCTL_PLAYER_H__
//
// #if !defined(__PLAYERCTL_INSIDE__) && !defined(PLAYERCTL_COMPILATION)
// #error "Only <playerctl/playerctl.h> can be included directly."
// #endif
//
// #include <glib-object.h>
// #include <playerctl/playerctl-enum-types.h>
// #include <playerctl/playerctl-player-name.h>
//
// /**
//  * SECTION: playerctl-player
//  * @short_description: A class to control a media player.
//  *
//  * The #PlayerctlPlayer represents a proxy connection to a media player through
//  * an IPC interface that is capable of performing commands and executing
//  * queries on the player for properties and metadata.
//  *
//  * If you know the name of your player and that it is running, you can use
//  * playerctl_player_new() giving the player name to connect to it. The player
//  * names given are the same as you can get with the binary `playerctl
//  * --list-all` command. Using this function will get you the first instance of
//  *  the player it can find, or the exact instance if you pass the instance as
//  *  the player name.
//  *
//  * If you would like to connect to a player dynamically, you can list players
//  * to be controlled with playerctl_list_players() or use the
//  * #PlayerctlPlayerManager class and read the list of player name containers in
//  * the #PlayerctlPlayerManager:player-names property or listen to the
//  * #PlayerctlPlayerManager::name-appeared event. If you have a
//  * #PlayerctlPlayerName, you can use the playerctl_player_new_from_name()
//  * function to create a #PlayerctlPlayer from this name.
//  *
//  * Once you have a player, you can give it commands to play, pause, stop, open
//  * a file, etc with the provided functions listed below. You can also query for
//  * properties such as the playback status, position, and shuffle status. Each
//  * of these has an event that will be emitted when these properties change
//  * during a main loop.
//  *
//  * For examples on how to use the #PlayerctlPlayer, see the `examples`
//  * directory in the git repository.
//  */
// #define PLAYERCTL_TYPE_PLAYER (playerctl_player_get_type())
// #define PLAYERCTL_PLAYER(obj) \
//     (G_TYPE_CHECK_INSTANCE_CAST((obj), PLAYERCTL_TYPE_PLAYER, PlayerctlPlayer))
// #define PLAYERCTL_IS_PLAYER(obj) (G_TYPE_CHECK_INSTANCE_TYPE((obj), PLAYERCTL_TYPE_PLAYER))
// #define PLAYERCTL_PLAYER_CLASS(klass) \
//     (G_TYPE_CHECK_CLASS_CAST((klass), PLAYERCTL_TYPE_PLAYER, PlayerctlPlayerClass))
// #define PLAYERCTL_IS_PLAYER_CLASS(klass) (G_TYPE_CHECK_CLASS_TYPE((klass), PLAYERCTL_TYPE_PLAYER))
// #define PLAYERCTL_PLAYER_GET_CLASS(obj) \
//     (G_TYPE_INSTANCE_GET_CLASS((obj), PLAYERCTL_TYPE_PLAYER, PlayerctlPlayerClass))
//
// typedef struct _PlayerctlPlayer PlayerctlPlayer;
// typedef struct _PlayerctlPlayerClass PlayerctlPlayerClass;
// typedef struct _PlayerctlPlayerPrivate PlayerctlPlayerPrivate;
//
// struct _PlayerctlPlayer {
//     /* Parent instance structure */
//     GObject parent_instance;
//
//     /* Private members */
//     PlayerctlPlayerPrivate *priv;
// };
//
// struct _PlayerctlPlayerClass {
//     /* Parent class structure */
//     GObjectClass parent_class;
// };
//
// GType playerctl_player_get_type(void);
//
// PlayerctlPlayer *playerctl_player_new(const gchar *player_name, GError **err);
//
// PlayerctlPlayer *playerctl_player_new_for_source(const gchar *player_name, PlayerctlSource source,
//                                                  GError **err);
//
// PlayerctlPlayer *playerctl_player_new_from_name(PlayerctlPlayerName *player_name, GError **err);
//
// /**
//  * PlayerctlPlaybackStatus:
//  * @PLAYERCTL_PLAYBACK_STATUS_PLAYING: A track is currently playing.
//  * @PLAYERCTL_PLAYBACK_STATUS_PAUSED: A track is currently paused.
//  * @PLAYERCTL_PLAYBACK_STATUS_STOPPED: There is no track currently playing.
//  *
//  * Playback status enumeration for a #PlayerctlPlayer
//  *
//  */
// typedef enum {
//     PLAYERCTL_PLAYBACK_STATUS_PLAYING, /*< nick=Playing >*/
//     PLAYERCTL_PLAYBACK_STATUS_PAUSED,  /*< nick=Paused >*/
//     PLAYERCTL_PLAYBACK_STATUS_STOPPED, /*< nick=Stopped >*/
// } PlayerctlPlaybackStatus;
//
// /**
//  * PlayerctlLoopStatus:
//  * @PLAYERCTL_LOOP_STATUS_NONE: The playback will stop when there are no more tracks to play.
//  * @PLAYERCTL_LOOP_STATUS_TRACK: The current track will start again from the beginning once it has
//  * finished playing.
//  * @PLAYERCTL_LOOP_STATUS_PLAYLIST: The playback loops through a list of tracks.
//  *
//  * Loop status enumeration for a #PlayerctlPlayer
//  *
//  */
// typedef enum {
//     PLAYERCTL_LOOP_STATUS_NONE,     /*< nick=None >*/
//     PLAYERCTL_LOOP_STATUS_TRACK,    /*< nick=Track >*/
//     PLAYERCTL_LOOP_STATUS_PLAYLIST, /* nick=Playlist >*/
// } PlayerctlLoopStatus;
//
// /*
//  * Static methods
//  */
// GList *playerctl_list_players(GError **err);
//
// /*
//  * Method definitions.
//  */
//
// void playerctl_player_on(PlayerctlPlayer *self, const gchar *event, GClosure *callback,
//                          GError **err);
//
// void playerctl_player_open(PlayerctlPlayer *self, gchar *uri, GError **err);
//
// void playerctl_player_play_pause(PlayerctlPlayer *self, GError **err);
//
// void playerctl_player_play(PlayerctlPlayer *self, GError **err);
//
// void playerctl_player_stop(PlayerctlPlayer *self, GError **err);
//
// void playerctl_player_seek(PlayerctlPlayer *self, gint64 offset, GError **err);
//
// void playerctl_player_pause(PlayerctlPlayer *self, GError **err);
//
// void playerctl_player_next(PlayerctlPlayer *self, GError **err);
//
// void playerctl_player_previous(PlayerctlPlayer *self, GError **err);
//
// gchar *playerctl_player_print_metadata_prop(PlayerctlPlayer *self, const gchar *property,
//                                             GError **err);
//
// gchar *playerctl_player_get_artist(PlayerctlPlayer *self, GError **err);
//
// gchar *playerctl_player_get_title(PlayerctlPlayer *self, GError **err);
//
// gchar *playerctl_player_get_album(PlayerctlPlayer *self, GError **err);
//
// void playerctl_player_set_volume(PlayerctlPlayer *self, gdouble volume, GError **err);
//
// gint64 playerctl_player_get_position(PlayerctlPlayer *self, GError **err);
//
// void playerctl_player_set_position(PlayerctlPlayer *self, gint64 position, GError **err);
//
// void playerctl_player_set_loop_status(PlayerctlPlayer *self, PlayerctlLoopStatus status,
//                                       GError **err);
//
// void playerctl_player_set_shuffle(PlayerctlPlayer *self, gboolean shuffle, GError **err);
//
// #endif /* __PLAYERCTL_PLAYER_H__ */

// Source: playerctl/playerctl-player-private.h
// /*
//  * This file is part of playerctl.
//  *
//  * playerctl is free software: you can redistribute it and/or modify it under
//  * the terms of the GNU Lesser General Public License as published by the Free
//  * Software Foundation, either version 3 of the License, or (at your option)
//  * any later version.
//  *
//  * playerctl is distributed in the hope that it will be useful, but WITHOUT ANY
//  * WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS
//  * FOR A PARTICULAR PURPOSE.  See the GNU Lesser General Public License for
//  * more details.
//  *
//  * You should have received a copy of the GNU Lesser General Public License
//  * along with playerctl If not, see <http://www.gnu.org/licenses/>.
//  *
//  * Copyright © 2014, Tony Crisci and contributors
//  */
//
// #ifndef __PLAYERCTL_PLAYER_PRIVATE_H__
// #define __PLAYERCTL_PLAYER_PRIVATE_H__
//
// #include "playerctl-player.h"
//
// char *pctl_player_get_instance(PlayerctlPlayer *player);
//
// gint player_name_string_compare_func(gconstpointer a, gconstpointer b, gpointer user_data);
//
// gint player_name_compare_func(gconstpointer a, gconstpointer b, gpointer user_data);
//
// gint player_compare_func(gconstpointer a, gconstpointer b, gpointer user_data);
//
// #endif /* __PLAYERCTL_PLAYER_PRIVATE_H__ */
