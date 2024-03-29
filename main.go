package main

import (
	"github.com/TheTipo01/libRoberto"
	"github.com/bwmarrin/lit"
	"github.com/kkyr/fig"
	tb "gopkg.in/telebot.v3"
	"strings"
	"time"
)

type config struct {
	Token    string `fig:"token" validate:"required"`
	LogLevel string `fig:"loglevel" validate:"required"`
	Voice    string `fig:"voice" validate:"required"`
	Channel  int64  `fig:"channel" validate:"required"`
}

const (
	audioType = "opus"
)

var (
	// Telegram token
	token string
	// String replacer
	replacer = strings.NewReplacer("_", "\\_", "*", "\\*", "[", "\\[", "]", "\\]", "(", "\\(", ")", "\\)",
		"~", "\\~", "`", "\\`", ">", "\\>", "#", "\\#", "+", "\\+", "-", "\\-", "=", "\\=", "|", "\\|", "{",
		"\\{", "}", "\\}", ".", "\\.", "!", "\\!")
	// Channel where to send the audio
	channel int64
)

func init() {
	lit.LogLevel = lit.LogError

	var cfg config
	err := fig.Load(&cfg, fig.File("config.yml"))
	if err != nil {
		lit.Error(err.Error())
		return
	}

	// Config file found
	token = cfg.Token
	libroberto.Voice = cfg.Voice
	channel = cfg.Channel

	// Set lit.LogLevel to the given value
	switch strings.ToLower(cfg.LogLevel) {
	case "logwarning", "warning":
		lit.LogLevel = lit.LogWarning

	case "loginformational", "informational":
		lit.LogLevel = lit.LogInformational

	case "logdebug", "debug":
		lit.LogLevel = lit.LogDebug
	}
}

func main() {
	// Create bot
	b, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		lit.Error(err.Error())
		return
	}

	b.Handle(tb.OnQuery, func(c tb.Context) error {
		text := c.Query().Text

		if text != "" {
			var (
				start      = time.Now()
				query      string
				upperQuery = strings.ToUpper(text)
				isCommand  = true
				results    = make(tb.Results, 1)
			)

			lit.Debug("%s: %s", c.Query().Sender.Username, text)

			// Various custom command
			switch {
			case strings.HasPrefix(upperQuery, "TRENO"):
				query = strings.TrimSpace(libroberto.SearchAndGetTrain(strings.TrimPrefix(upperQuery, "TRENO ")))
				if query == "" {
					query = "Nessun treno trovato, agagagaga!"
				}

			case strings.HasPrefix(upperQuery, "COVID"):
				query = strings.TrimSpace(libroberto.GetCovid())

			case strings.HasPrefix(upperQuery, "BESTEMMIA"):
				query = strings.TrimSpace(libroberto.Bestemmia())

			default:
				query = libroberto.EmojiToDescription(upperQuery)
				isCommand = false
			}

			cmds := libroberto.GenAudioPipes(query, audioType)
			out, _ := cmds[1].StdoutPipe()
			libroberto.CmdsStart(cmds)

			// So the title of the result isn't all uppercase when there's no command
			if !isCommand {
				query = text
			}

			send, err := c.Bot().Send(tb.ChatID(channel), &tb.Voice{File: tb.FromReader(out), MIME: "audio/ogg"})
			if err != nil {
				lit.Error(err.Error())
				return nil
			}

			libroberto.CmdsKill(cmds)
			libroberto.CmdsWait(cmds)

			// Create result
			results[0] = &tb.VoiceResult{
				Cache:   send.Voice.FileID,
				Title:   query,
				Caption: "||" + replacer.Replace(query) + "||",
			}

			results[0].SetResultID(libroberto.GenUUID(query))
			results[0].SetParseMode(tb.ModeMarkdownV2)

			lit.Debug("took %s to answer query", time.Since(start).String())

			// Send audio
			return c.Answer(&tb.QueryResponse{
				Results:   results,
				CacheTime: 86400, // one day
			})

		}

		return nil
	})

	// Start bot
	lit.Info("robertoTelegram is now running")
	b.Start()
}
