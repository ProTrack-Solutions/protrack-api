package domain

type LogLevel string

const (
	LevelInfo    LogLevel = "INFO"
	LevelWarning LogLevel = "WARNING"
	LevelError   LogLevel = "ERROR"
)

type DiscordEmbedFooter struct {
	Text string `json:"text"`
}

type DiscordEmbed struct {
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Color       int                `json:"color"`
	Timestamp   string             `json:"timestamp"`
	Footer      DiscordEmbedFooter `json:"footer"`
}

type DiscordPayload struct {
	Content string         `json:"content,omitempty"`
	Embeds  []DiscordEmbed `json:"embeds"`
}
