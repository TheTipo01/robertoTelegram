package main

import (
	"github.com/TheTipo01/libRoberto"
	"github.com/bwmarrin/lit"
	"github.com/kkyr/fig"
	tb "gopkg.in/tucnak/telebot.v2"
	"math/rand"
	"strings"
	"time"
)

type config struct {
	Token    string `fig:"token" validate:"required"`
	Host     string `fig:"host" validate:"required"`
	LogLevel string `fig:"loglevel" validate:"required"`
}

const audioExtension = ".mp3"

var (
	// Telegram token
	token string
	// HTTP server where we host .mp3
	host string
)

func init() {
	// Initialize rand
	rand.Seed(time.Now().Unix())

	lit.LogLevel = lit.LogError

	var cfg config
	err := fig.Load(&cfg, fig.File("config.yml"))
	if err != nil {
		lit.Error(err.Error())
		return
	}

	// Config file found
	token = cfg.Token
	host = cfg.Host

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

	b.Handle(tb.OnQuery, func(q *tb.Query) {
		if q.Text != "" {
			var (
				start      = time.Now()
				err        error
				query      string
				upperQuery = strings.ToUpper(q.Text)
				uuid       string
				isCommand  = true
				results    = make(tb.Results, 1)
			)

			lit.Debug("%s: %s", q.From.Username, q.Text)

			// Various custom command
			switch {
			case strings.HasPrefix(upperQuery, "TRENO"):
				query = strings.TrimSpace(libRoberto.SearchAndGetTrain(strings.TrimPrefix(upperQuery, "TRENO ")))
				if query == "" {
					query = "Nessun treno trovato, agagagaga!"
				}

			case strings.HasPrefix(upperQuery, "COVID"):
				query = strings.TrimSpace(libRoberto.GetCovid())

			case strings.HasPrefix(upperQuery, "BESTEMMIA"):
				query = strings.TrimSpace(libRoberto.Bestemmia())

			default:
				query = libRoberto.EmojiToDescription(upperQuery)
				isCommand = false
			}

			uuid = libRoberto.GenAudioMp3(query, time.Second*30)

			// So the title of the result isn't all uppercase when there's no command
			if !isCommand {
				query = q.Text
			}

			// Create result
			results[0] = &tb.VoiceResult{

				URL:   host + uuid + audioExtension,
				Title: query,
			}

			results[0].SetResultID(uuid)

			// Send audio
			err = b.Answer(q, &tb.QueryResponse{
				Results:   results,
				CacheTime: 86400, // one day
			})
			if err != nil {
				lit.Error("error while answering inline query: %s", err)
			}

			lit.Debug("took %s to answer query", time.Since(start).String())
		}
	})

	// Start bot
	lit.Info("robertoTelegram is now running")
	b.Start()

}
