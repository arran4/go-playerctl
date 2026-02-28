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
