package mp

import (
	"fmt"
	"strings"
)

type PrefKey struct {
	Name   string
	Key    string
	Cls    string
	Values string
}

var prefKeys = []PrefKey{
	{"My Callsign", "locationCallsign", "String", "Alphanumeric"},
	{"My Team", "locationTeam", "String", "White; Yellow; Orange; Magenta; Red; Maroon; Purple; Dark Blue; Blue; Cyan; Teal; Green; Dark Green; Brown"},
	{"My Role", "atakRoleType", "String", "Team Member; Team Lead; HQ; Sniper; Medic; Forward Observer; RTO; K9"},
	{"My Display Type", "locationUnitType", "String", "Ground Troop; Armored Vehicle; Civilian Vehicle; Generic Air Unit; Generic Ground Unit; Generic Sea Surface Unit"},
	{"GPS Option", "mockingOption", "String", "IgnoreInternalGPS = Ignore Internal GPS / Use External or Network GPS Only; LocalGPS = Internal GPS Only; WRGPS = External or Network GPS / Fallback Internal GPS"},
	{"Network GPS Port", "listenPort", "String", "0 - 65535 (Default 4349)"},
	{"Use GPS Time", "useGPSTime", "Boolean", "true; false"},
	{"Use Elevation Data instead of GPS Elevation", "useTerrainElevationSelfMarker", "Boolean", ""},
	{"Non-Bluetooth Laser Range Finders Support", "nonBluetoothLaserRangeFinder", "Boolean", ""},
	{"Use Wave Relay Callsign", "locationUseWRCallsign", "Boolean", ""},
	{"Bluetooth Support", "atakControlBluetooth", "Boolean", ""},
	{"Bluetooth Reconnect Time", "atakBluetoothReconnectSeconds", "String", "Numeric"},
	{"Publish Phone Number", "saHasPhoneNumber", "Boolean", ""},
	{"Publish VoIP Number", "saSipAddressAssignment", "String", "Alphanumeric"},
	{"VoIP Number", "saSipAddress", "String", "No VOIP; Manual Entry; Use IP Address; Use Callsign and IP Address"},
	{"Publish XMPP Username", "saXmppUsername", "String", "Alphanumeric"},
	{"Publish Preferred Email", "saEmailAddress", "String", "Alphanumeric"},
	{"Unit Reference Number", "saURN", "String", "0 - 16777215"},
	{"Send Location Over Network", "dispatchLocationCotExternal", "Boolean", ""},
	{"Hide my current position", "dispatchLocationHidden", "Boolean", ""},
	{"Reporting Strategy", "locationReportingStrategy", "String", "Dynamic; Constant"},
	{"Dynamic Reporting Rate Stationary (Unreliable)", "dynamicReportingRateStationaryUnreliable", "String", "Numeric"},
	{"Dynamic Reporting Rate Minimum (Unreliable)", "dynamicReportingRateMinUnreliable", "String", "Numeric"},
	{"Dynamic Reporting Rate Maximum (Unreliable)", "dynamicReportingRateMaxUnreliable", "String", "Numeric"},
	{"Dynamic Reporting Rate Stationary (Reliable)", "dynamicReportingRateStationaryReliable", "String", "Numeric"},
	{"Dynamic Reporting Rate Minimum (Reliable)", "dynamicReportingRateMinReliable", "String", "Numeric"},
	{"Dynamic Reporting Rate Maximum (Reliable)", "dynamicReportingRateMaxReliable", "String", "Numeric"},
	{"Constant Reporting Rate (Unreliable)", "constantReportingRateUnreliable", "String", "Numeric"},
	{"Constant Reporting Rate (Reliable)", "constantReportingRateReliable", "String", "Numeric"},
	{"Report Location before obtaining location fix", "dispatchLocationCotExternalAtStart", "Boolean", ""},
	{"Change Encryption Passphrase", "encryptionPassphrase", "String", "Alphanumeric"},
	{"Display Connection Widget", "displayServerConnectionWidget", "Boolean", ""},
	{"Monitor Server Connections", "monitorServerConnections", "Boolean", ""},
	{"Default SSL/TLS TrustStore Location", "caLocation", "String", "Alphanumeric"},
	{"Default SSL/TLS TrustStore Password", "caPassword", "String", "Alphanumeric"},
	{"Default SSL/TLS Client Certificate Store", "certificateLocation", "String", "Alphanumeric"},
	{"Default SSL/TLS Client Certificate Password", "clientPassword", "String", "Alphanumeric"},
	{"Default SSL/TLS Credentials", "default_client_credentials", "String", "Alphanumeric"},
	{"Secure Server API port", "apiSecureServerPort", "String", "0-65535"},
	{"Unsecure Server API port", "apiUnsecureServerPort", "String", "0-65535"},
	{"Certificate Enrollment API port", "apiCertEnrollmentPort", "String", "0-65535"},
	{"Certificate Enrollment Key Length", "apiCertEnrollmentKeyLength", "String", "Numeric"},
	{"Enable Client Certificate Export", "certEnrollmentExport", "Boolean", ""},
	{"Apply TAK Server profile updates", "deviceProfileEnableOnConnect", "Boolean", ""},
	{"Enable Mesh Network Mode", "enableNonStreamingConnections", "Boolean", ""},
	{"Auto Manage Mesh Network Self SA Mode", "autoDisableMeshSAWhenStreaming", "Boolean", ""},
	{"TCP Connection Timeout", "tcpConnectTimeout", "String", "Numeric"},
	{"Multicast Network Rejoin interval", "udpNoDataTimeout", "String", "Numeric"},
	{"Multicast TTL", "multicastTTL", "String", "Numeric"},
	{"Multicast Traffic Loopback", "network_multicast_loopback", "Boolean", ""},
	{"Point-to-Point Link High Speed Capable", "ppp0_highspeed_capable", "Boolean", ""},
	{"Legacy Wave Relay Redirect", "waveRelayRedirect", "Boolean", ""},
	{"Enable VPN Proxy Mode", "httpClientPermissiveMode", "Boolean", ""},
	{"Auto Start on Placement", "autostart_nineline", "Boolean", ""},
	{"Auto Start Menu", "autostart_nineline_type", "String", "cas; cff"},
	{"Danger Close Coloring", "prone_standing_coloring", "Boolean", ""},
	{"Use Keyhole CAS", "keyhole_cas", "Boolean", ""},
	{"FAH Distance (NM)", "fahDistance", "String", "0-21600"},
	{"Default FAH Width", "fahWidth", "String", "0-180"},
	{"Automatically Display FAH", "fahOnCreation", "Boolean", ""},
	{"Automatically Display Reciprocal FAH", "fahReciprocalOnCreation", "Boolean", ""},
	{"Laser Basket Distance (NM)", "laserBasketDistance", "String", "0-21600"},
	{"Laser Basket Measurements", "laserBasketDegrees", "Boolean", ""},
	{"FAH Always Edit Mode", "nineline_fah_edit_always", "Boolean", ""},
	{"Enable Address Lookup (Geocoder)", "enableGeocoder", "Boolean", ""},
	{"Address Provider", "selected_geocoder_uid", "String", "android-geocoder"},
	{"Notification", "alert_notification", "Boolean", ""},
	{"Individual Alert Notifications", "individual_alert_notification", "Boolean", ""},
	{"Audible alert", "alert_audible", "Boolean", ""},
	{"Vibration alert", "alert_vibration", "Boolean", ""},
	{"Alert SMS Numbers", "sms_numbers", "String", "Numeric 11-15 digits"},
	{"Display Bloodhound Text Widget", "rab_bloodhound_display_textwidget", "Boolean", ""},
	{"Display Large Bloodhound Text Widget", "rab_bloodhound_large_textwidget", "Boolean", ""},
	{"Flash When Closing In", "rab_bloodhound_flash_colors", "Boolean", ""},
	{"Toggle Auto Zoom", "rab_bloodhound_zoom", "Boolean", ""},
	{"Flash ETA Threshold", "bloodhound_flash_eta", "String", "Numeric"},
	{"Outer/Middle ETA Threshold", "bloodhound_outer_eta", "String", "Numeric"},
	{"Middle/Inner ETA Threshold", "bloodhound_inner_eta", "String", "Numeric"},
	{"Flash Color", "bloodhound_flash_color_pref", "String", "red; yellow; green; cyan; blue; magenta; white; black"},
	{"Outer ETA Color", "bloodhound_outer_color_pref", "String", ""},
	{"Middle ETA Color", "bloodhound_middle_color_pref", "String", ""},
	{"Inner ETA Color", "bloodhound_inner_color_pref", "String", ""},
	{"Reroute distance", "bloodhound_reroute_distance_pref", "String", "Numeric"},
	{"Reroute timer frequency", "bloodhound_reroute_timer_pref", "String", "Numeric"},
	{"Display Notification", "enableToast", "Boolean", ""},
	{"Vibrate phone", "vibratePhone", "Boolean", ""},
	{"Audible Chat notifications", "audibleNotify", "Boolean", ""},
	{"Chat Address", "chatAddress", "String", "1.1.1.1 - 255.255.255.255"},
	{"Chat Port", "chatPort", "String", "0-65535"},
	{"Enable file sharing.", "filesharingEnabled", "Boolean", ""},
	{"File Transfer Max size", "filesharingSizeThresholdNoGo", "String", "Numeric"},
	{"Download attempts", "fileshareDownloadAttempts", "String", "Numeric"},
	{"Allow HTTP transfers", "filesharingWebServerLegacyHttpEnabled", "Boolean", ""},
	{"Server port", "filesharingWebServerPort", "String", "0-65535"},
	{"Secure Server port", "filesharingSecureWebServerPort", "String", "0-65535"},
	{"File Transfer Connection Timeout", "filesharingConnectionTimeoutSecs", "String", "Numeric"},
	{"File Transfer Failure Timeout", "filesharingTransferTimeoutSecs", "String", "Numeric"},
	{"Use All Servers", "filesharingAllServers", "Boolean", ""},
	{"Elevation Server", "prefs_dted_stream_server", "String", "Alphanumeric"},
	{"Stream Elevation Data", "prefs_dted_stream", "Boolean", ""},
	{"Show by Default", "prefs_dted_visible", "Boolean", ""},
	{"Rapid MGRS Entry Mode", "rapid_mgrs_dialog", "Boolean", ""},
	{"DP/SPI Update Period", "spiUpdateDelay", "String", "Numeric"},
	{"Automatically Broadcast a Marker", "hostileUpdateDelay", "String", "0; 60; 90; 120"},
	{"Legacy Toolbar Mode", "legacyFiresToolbarMode", "Boolean", ""},
	{"Number of DP/SPIs", "firesNumberOfSpis", "String", "1; 2; 3"},
	{"DP/RedX FAH Width", "spiFahSize", "String", "0-180"},
	{"Expanded Danger Close Search", "expandedDangerClose", "Boolean", ""},
	{"WMS Connect Timeout", "wms_connect_timeout", "Integer", "3000; 5000; 10000; 15000; 30000"},
	{"Enable Volume Keys rotating map layers", "volumemapswitcher", "Boolean", ""},
	{"Enable Image Overlay Map Interaction", "prefs_layer_grg_map_interaction", "Boolean", ""},
	{"Collect Metrics", "collect_metrics", "Boolean", ""},
	{"Live Metrics", "realtime_metrics", "Boolean", ""},
	{"Advanced CoT Recorder", "generate_full_pool", "Boolean", ""},
	{"Advanced CoT Details Recorder", "generate_detail_pool", "Boolean", ""},
	{"Checkpoint Name Prefix", "waypointPrefix", "String", "Alphanumeric"},
	{"Default for unspecified Route Colors", "defaultRouteColor", "String", "-1 through -16777216"},
	{"Traditional Navigation Mode", "route_track_up_locked_on", "Boolean", ""},
	{"Navigational Voice Cues", "useRouteVoiceCues", "Boolean", ""},
	{"Vibrate When Arriving at Checkpoint", "route_vibrate_at_checkpoint", "Boolean", ""},
	{"Walking Checkpoint Navigation Bubble Radius", "waypointBubble.Walking", "String", "Numeric"},
	{"Walking Off Route Bubble Radius", "waypointOffRouteBubble.Walking", "String", "Numeric"},
	{"Driving Checkpoint Navigation Bubble Radius", "waypointBubble.Driving", "String", "Numeric"},
	{"Driving Off Route Bubble Radius", "waypointOffRouteBubble.Driving", "String", "Numeric"},
	{"Flying Checkpoint Navigation Bubble Radius", "waypointBubble.Flying", "String", "Numeric"},
	{"Flying Off Route Bubble Radius", "waypointOffRouteBubble.Flying", "String", "Numeric"},
	{"Swimming Checkpoint Navigation Bubble Radius", "waypointBubble.Swimming", "String", "Numeric"},
	{"Swimming Off Route Bubble Radius", "waypointOffRouteBubble.Swimming", "String", "Numeric"},
	{"Watercraft Checkpoint Navigation Bubble Radius", "waypointBubble.Watercraft", "String", "Numeric"},
	{"Watercraft Off Route Bubble Radius", "waypointOffRouteBubble.Watercraft", "String", "Numeric"},
	{"Show Attachment Billboards", "route_billboard_enabled", "Boolean", ""},
	{"Billboard Render Distance", "route_billboard_distance_m", "String", "Numeric"},
	{"Elevation Profile Interpolate Altitudes", "elevProfileInterpolateAlt", "Boolean", ""},
	{"Elevation Profile Center on Seeker", "elevProfileCenterOnSeeker", "Boolean", ""},
	{"GPX Import Checkpoints", "gpxImportCheckpointsForNamedRoutePoints", "Boolean", ""},
	{"Ignore Elevation Data", "kmlExportGroundClamp", "Boolean", ""},
	{"KML Export Checkpoint Mode", "kmlExportCheckpointMode", "String", "Points; Line; Both"},
	{"Internal Network Configuration", "network_config", "Boolean", ""},
	{"DHCP", "network_dhcp", "Boolean", ""},
	{"IP address", "network_static_ip_address", "String", "1.1.1.1 - 255.255.255.255"},
	{"Subnet mask", "network_static_subnet_mask", "String", "1.1.1.1 - 255.255.255.255"},
	{"Default gateway", "network_static_gateway", "String", "1.1.1.1 - 255.255.255.255"},
	{"DNS 1", "network_static_dns1", "String", "1.1.1.1 - 255.255.255.255"},
	{"DNS 2", "network_static_dns2", "String", "1.1.1.1 - 255.255.255.255"},
	{"Log Tracks", "toggle_log_tracks", "Boolean", ""},
	{"Track Prefix", "track_prefix", "String", "Alphanumeric"},
	{"Auto Rotate Track Colors", "toggle_rotate_track_colors", "Boolean", ""},
	{"Default Color", "track_history_default_color", "String", "-1 through -16777216"},
	{"CSV Export Header", "track_csv_export_headers", "Boolean", ""},
	{"KML Export Timestamps", "track_kml_export_timestamps", "Boolean", ""},
	{"Server track time gap (minutes)", "bread_track_timegap_threshold", "String", "Numeric"},
	{"Persistent Self Track", "track_infinite", "Boolean", ""},
	{"Set Max Number of Bread Crumbs", "max_num_bread_tracks", "Integer", "0-500"},
	{"Render bread crumb line to ground in 3-D", "track_line_to_surface", "Boolean", ""},
	{"Default Crumb Size", "track_crumb_size", "String", "10; 15; 20"},
	{"Bread Crumb Distance Threshold", "bread_dist_threshold", "String", "Numeric"},
	{"Coordinate Display", "coord_display_pref", "String", "MGRS; DD; DM; DMS; UTM"},
	{"Altitude Display", "alt_display_pref", "String", "HAE; MSL"},
	{"Altitude Units", "alt_unit_pref", "String", "0 = Feet (ft); 1 = Meters (m)"},
	{"Display AGL", "alt_display_agl", "Boolean", ""},
	{"Speed Units", "speed_unit_pref", "String", "0 = Miles per hour (mph); 1 = KM per hour (kmph); 2 = KGS (kts); 3 = Meters Per Second (mps)"},
	{"Range and Bearing Line Distance Format", "rab_dist_slant_range", "String", "slantrange; clamped"},
	{"Bearing Units", "rab_brg_units_pref", "String", "0 = Degrees; 1 = Mils"},
	{"Range Units", "rab_rng_units_pref", "String", "0 = Feet/Miles; 1 = Meters/Kilometers; 2 = Nautical Miles"},
	{"North Reference", "rab_north_ref_pref", "String", "0 = True North; 1 = Magnetic North; 2 = Grid North"},
	{"Feet to Miles", "rng_feet_display_pref", "String", "Numeric"},
	{"Meters to Kilometers", "rng_meters_display_pref", "String", "Numeric"},
	{"Default Range and Bearing Color", "rab_color_pref", "String", ""},
	{"Display ETA on Range and Bearing Labels", "rab_preference_show_eta", "Boolean", ""},
	{"Set Domain Preference", "set_domain_pref", "String", "Ground; Aviation; Maritime"},
	{"Enable Zoom Controls", "map_zoom_visible", "Boolean", ""},
	{"Enable Map Scale Display", "map_scale_visible", "Boolean", ""},
	{"Enable Map Scale Rounding", "map_scale_rounding", "Boolean", ""},
	{"Designate the Map Center", "map_center_designator", "Boolean", ""},
	{"Self Coordinate Information", "self_coord_info_display", "String", ""},
	{"SidePane Handle Shade", "sidepane_handle_shade", "String", "Light; Dark"},
	{"Enable Large Text Mode", "largeTextMode", "Boolean", ""},
	{"Enable Large Tool Bar", "largeActionBar", "Boolean", ""},
	{"Icon and Text Size", "relativeOverlaysScalingRadioList", "String", "1.00 = Normal; 1.25 = 1.25x; 1.50 = 1.5x; 1.75 = 1.75x; 2.00 = 2x"},
	{"Default Label Size", "label_text_size", "String", "14 = Normal; 16 = Large; 18 = X-Large"},
	{"Map Overlay's Dimming/Brightening", "dim_map_with_brightness_key", "Boolean", ""},
	{"Adjust for Curved Display", "atakAdjustCurvedDisplay", "Boolean", ""},
	{"Enable the Faux Navigation Bar", "faux_nav_bar", "Boolean", ""},
	{"Reverse the buttons on the Faux Navigation Bar", "faux_nav_bar_reverse", "Boolean", ""},
	{"Disable AutoLink launching for Associated Links", "disable_autolink_mapitem_links", "Boolean", ""},
	{"Enable DEX Controls", "dexControls", "Boolean", ""},
	{"Enable Globe Display Mode", "atakGlobeModeEnabled", "Boolean", ""},
	{"Marker Size", "location_marker_scale_key", "String", "-1 = Default; 40 = 1x; 48 = 2x; 56 = 3x; 64 = 4x"},
	{"Default", "default_gps_icon", "Boolean", ""},
	{"Team Color", "team_color_gps_icon", "Boolean", ""},
	{"Custom", "custom_color_gps_icon_pref", "Boolean", ""},
	{"Change/Create Icon Main Custom Color", "custom_color_selected", "String", "16-bit Hexadecimal Color"},
	{"Change/Create Icon Outline Custom Color", "custom_outline_color_selected", "String", "16-bit Hexadecimal Color"},
	{"Toolbar Side", "nav_orientation_right", "Boolean", ""},
	{"Toolbar Icon Color", "actionbar_icon_color_key", "String", "16-bit Hexadecimal Color"},
	{"Toolbar Background Color", "actionbar_background_color_key", "String", "16-bit Hexadecimal Color"},
	{"Grid Line Color", "pref_grid_color", "String", "#ffffff = White; #ff00ff = Magenta; #00ff00 = Green; #ffff00 = Yellow; Custom"},
	{"Grid Line Color", "pref_grid_color_value", "String", "#ffffff = White; #ff00ff = Magenta; #00ff00 = Green; #ffff00 = Yellow; Any other HEX RGB value = Custom"},
	{"Show by Default", "prefs_grid_default_show", "Boolean", ""},
	{"Layer Outline Color", "pref_layer_outline_color", "String", "#ffffff = White; #ff00ff = Magenta; #00ff00 = Green; #ffff00 = Yellow"},
	{"Show by Default", "prefs_layer_outlines_default_show", "Boolean", ""},
	{"Enabled on Start", "toggle_offscreen_indicators", "Boolean", ""},
	{"Set Distance Threshold (KM)", "offscreen_indicator_dist_threshold", "String", "Numeric"},
	{"Set Timeout Threshold (seconds) or zero to disable", "offscreen_indicator_timeout_threshold", "String", "Numeric"},
	{"Imported Shape Outline Color", "pref_overlay_style_outline_color", "String", "-1 through -16777216"},
	{"Overlay Manager Width/Height", "overlay_manager_width_height", "String", "50; 33; 25"},
	{"Forced Application Brightness", "atakForcedBrightness", "String", "-1 = None; 0 = Lowest; 25 = 25 percent; 50 = 50 percent; 75 = 75 percent; 100 = Highest"},
	{"Disable Screen Saver / Screen Lock", "atakScreenLock", "Boolean", ""},
	{"Disable SoftKey Illumination", "atakDisableSoftkeyIllumination", "Boolean", ""},
	{"Reverse Landscape Orientation", "atakControlReverseOrientation", "Boolean", ""},
	{"Shorten Long Marker Labels", "atakControlShortenLabels", "Boolean", ""},
	{"Show Marker Labels", "atakControlShowLabels", "Boolean", ""},
	{"Display Notifications for other users", "atakControlOtherUserNotification", "Boolean", ""},
	{"Fade Notification Time", "fade_notification", "String", "30 = 30 seconds; 60 = 60 seconds; 90 = 90 seconds; 120 = 2 minutes; 300 = 5 minutes; 600 = 10 minutes; 3600 = 1 hour; -1 = Do Not Fade"},
	{"Imagery Tab Name", "imagery_tab_name", "String", "Imagery; Maps"},
	{"Render Speed", "frame_limit", "String", "0 = Full speed; 1 = Battery saver"},
	{"Secure Delete Timeout", "secureDeleteTimeout", "String", "Numeric"},
	{"Prefer KLV Frame Center Elevation", "prefs_use_klv_elevation", "Boolean", ""},
	{"Camera Application", "quickpic.camera_chooser", "String", "System; TakGeoCam"},
	{"Force Center while Moving", "disableFloatToBottom", "Boolean", ""},
	{"Stale out remote users as they disconnect from TAK Servers", "staleRemoteDisconnects", "Boolean", ""},
	{"Remove all marker types from map when they stale out", "expireEverything", "Boolean", ""},
	{"Stale item cleanup time", "expireStaleItemsTime", "String", "0 = Immediately; 1 = 1 minute; 5 = 5 minutes; 20 = 20 minutes; 60 = 1 hour; 360 = 6 hours;1440 = 1 day"},
	{"Remove Unknown markers when they stale out", "expireUnknowns", "Boolean", ""},
	{"Quit on Back Press", "atakControlQuitOnBack", "Boolean", ""},
	{"Ask Before Quit", "atakControlAskToQuit", "Boolean", ""},
	{"Action for Long Pressing the Map", "atakLongPressMap", "String", "nothing = Do Nothing; actionbar = Toggle the Toolbar; dropicon = Drop a Point"},
	{"Double tap to zoom 2X", "atakDoubleTapToZoom", "Boolean", ""},
	{"Tap Coordinate Action", "self_coord_action", "String", "nothing = Do Nothing; cyclecoordinate = Change Coordinates; panto = Pan to Self"},
	{"Enable enlarging the coordinate display", "selfcoord_legacy_enlarge", "Boolean", ""},
	{"Legacy Point Drop Naming Convention", "legacyPointDropNaming", "Boolean", ""},
	{"Disable KeyGuard", "disableKeyGuard", "Boolean", ""},
	{"Forced Airplane Mode", "atakForceAirplaneRadioList", "String", "none = Do not Force; cell = Force General Radio Off; cell"},
	{"Continuous Rendering", "atakContinuousRender", "Boolean", ""},
	{"Developer Tools", "atakDeveloperTools", "Boolean", ""},
	{"Auto Upload App Logs", "enableAutoUploadLogs", "Boolean", ""},
	{"Enable Logcat File", "loggingfile", "Boolean", ""},
	{"Enable Logcat Errors Only", "loggingfile_error_only", "Boolean", ""},
	{"Upload Debug Logs", "loggingfile_upload_debug", "Boolean", ""},
	{"Log Network Traffic", "lognettraffictofile", "Boolean", ""},
	{"Disable the EUD (End User Device) API option", "eud_api_disable_option", "Boolean", ""},
	{"Enable EUD sync sync mapsources", "eud_api_sync_mapsources", "Boolean", ""},
	{"Enable EUD sync sync plugins", "eud_api_sync_plugins", "Boolean", ""},
	{"Enable scanning for plugins on startup", "atakPluginScanningOnStartup", "Boolean", ""},
	{"Toggle visibility of DPGK (Digital Precision Grid Kit) root", "dpgkRootVisibilityToggle", "Boolean", ""},
	{"Only allow streaming with VPN enabled", "onlyStreamingWithVPN", "Boolean", ""},
	{"Tabs for coordinate entry", "coordinate_entry_tabs", "String", ""},
	{"Enable QUIC (Quick UDP Internet Connections) protocol", "network_quic_enabled", "Boolean", ""},
	{"Enable repository startup synchronization", "repoStartupSync", "Boolean", ""},
	{"Toggle visibility of GRG root", "grgRootVisibilityToggle", "Boolean", ""},
	{"Prefer alternate number for display", "preferAltNumber", "Boolean", ""},
	{"Insert destination in directed CoT messages", "insertDestInDirectedCoT", "Boolean", ""},
	{"Enable update server for app management", "appMgmtEnableUpdateServer", "Boolean", ""},
	{"Show map eye altitude", "map_eyealt_visible", "Boolean", ""},
	{"Enable smart cache", "prefs_enable_smart_cache", "Boolean", ""},
	{"Show TAK version number in dispatch", "dispatchTAKVersionNumber", "Boolean", ""},
	{"Auto-close tilt rotation menu", "tilt_rotation_menu_auto_close", "Boolean", ""},
	{"Center self on back button press", "atakBackButtonCenterSelf", "Boolean", ""},
	{"Compass heading display preference (e.g., Numeric, Cardinal)", "compass_heading_display", "String", ""},
	{"Force English language for the application", "forceEnglish", "Boolean", ""},
	{"URL for ATAK update server", "atakUpdateServerUrl", "String", ""},
	{"Smart cache download limit (in bytes)", "prefs_smart_cache_download_limit", "String", ""},
	{"Scroll to self on start", "scrollToSelfOnStart", "Boolean", ""},
	{"Restore recorded location on startup", "restoreRecordedLocation", "Boolean", ""},
	{"Enable extended buttons in landscape mode", "landscape_extended_buttons", "Boolean", ""},
	{"Override permission request", "override_permission_request", "Boolean", ""},
}

func (p *PrefKey) Val(val string) string {
	if p == nil {
		return ""
	}

	return fmt.Sprintf("<entry key=\"%s\" class=\"class java.lang.%s\">%s</entry>", p.Key, p.Cls, val)
}

func GetEntry(key, val string) string {
	if strings.HasPrefix(key, "disablePreferenceItem_") || strings.HasPrefix(key, "hidePreferenceItem_") {
		return fmt.Sprintf("<entry key=\"%s\" class=\"class java.lang.Boolean\">%s</entry>", key, val)
	}

	for _, p := range prefKeys {
		if p.Key == key {
			return p.Val(val)
		}
	}

	return ""
}
