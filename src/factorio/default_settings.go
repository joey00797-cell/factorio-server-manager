package factorio

import (
	"log"
	"os"
)

const hardcodedDefaultSettings = `{
  "_comment_max_players": "Maximum number of players allowed, admins can join even a full server. 0 means unlimited.",
  "_comment_visibility": ["public: Game will be published on the official Factorio matching server", "lan: Game will be broadcast on LAN"],
  "_comment_credentials": "Your factorio.com login credentials. Required for games with visibility public",
  "_comment_token": "Authentication token. May be used instead of 'password' above.",
  "_comment_require_user_verification": "When set to true, the server will only allow clients that have a valid Factorio.com account",
  "_comment_max_upload_in_kilobytes_per_second": "optional, default value is 0. 0 means unlimited.",
  "_comment_max_upload_slots": "optional, default value is 5. 0 means unlimited.",
  "_comment_minimum_latency_in_ticks": "optional one tick is 16ms in default speed, default value is 0. 0 means no minimum.",
  "_comment_max_heartbeats_per_second": "Network tick rate. Maximum rate game updates packets are sent at before bundling them together. Minimum value is 6, maximum value is 240.",
  "_comment_ignore_player_limit_for_returning_players": "Players that played on this map already can join even when the max player limit was reached.",
  "_comment_allow_commands": "possible values are, true, false and admins-only",
  "_comment_autosave_interval": "Autosave interval in minutes",
  "_comment_autosave_slots": "server autosave slots, it is cycled through when the server autosaves.",
  "_comment_afk_autokick_interval": "How many minutes until someone is kicked when doing nothing, 0 for never.",
  "_comment_auto_pause": "Whether should the server be paused when no players are present.",
  "_comment_auto_pause_when_players_connect": "Whether should the server be paused when someone is connecting to the server.",
  "_comment_autosave_only_on_server": "Whether autosaves should be saved only on server or also on all connected clients. Default is true.",
  "_comment_non_blocking_saving": "Highly experimental feature, enable only at your own risk of losing your saves.",
  "_comment_segment_sizes": "Long network messages are split into segments that are sent over multiple ticks.",
  "name": "Factorio Server",
  "description": "",
  "tags": [],
  "max_players": 0,
  "visibility": {
    "public": false,
    "lan": true
  },
  "username": "",
  "password": "",
  "token": "",
  "game_password": "",
  "require_user_verification": true,
  "max_upload_in_kilobytes_per_second": 0,
  "max_upload_slots": 5,
  "minimum_latency_in_ticks": 0,
  "max_heartbeats_per_second": 60,
  "ignore_player_limit_for_returning_players": false,
  "allow_commands": "admins-only",
  "autosave_interval": 10,
  "autosave_slots": 5,
  "afk_autokick_interval": 0,
  "auto_pause": true,
  "auto_pause_when_players_connect": false,
  "only_admins_can_pause_the_game": true,
  "autosave_only_on_server": true,
  "non_blocking_saving": false,
  "minimum_segment_size": 25,
  "minimum_segment_size_peer_count": 20,
  "maximum_segment_size": 100,
  "maximum_segment_size_peer_count": 10,
  "admins": []
}`

// getDefaultServerSettings returns default server settings JSON.
// Priority: /opt/fsm-data/default-server-settings.json → hardcoded fallback
func getDefaultServerSettings() string {
	customPath := "/opt/fsm-data/default-server-settings.json"
	if data, err := os.ReadFile(customPath); err == nil {
		log.Printf("Using custom default settings from %s", customPath)
		return string(data)
	}
	log.Printf("Using hardcoded default server settings")
	return hardcodedDefaultSettings
}
