package main

import (
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"

	"github.com/TheTipo01/libRoberto"
	"github.com/bwmarrin/lit"
	"github.com/kkyr/fig"
	tb "gopkg.in/telebot.v3"
)

type config struct {
	Token            string `fig:"token" validate:"required"`
	LogLevel         string `fig:"loglevel" validate:"required"`
	Voice            string `fig:"voice" validate:"required"`
	Channel          int64  `fig:"channel" validate:"required"`
	RestRoberto      string `fig:"restroberto"`
	RestRobertoToken string `fig:"restrobertotoken"`
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
	// Endpoint for rest roberto
	restRoberto string
	// Token for rest roberto
	restRobertoToken string
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
	restRoberto = cfg.RestRoberto
	restRobertoToken = cfg.RestRobertoToken

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

func generateAudio(text string) (query string, isCommand bool, out io.ReadCloser, cmds []*exec.Cmd, err error) {
	upperQuery := strings.ToUpper(text)

	isCommand = true

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

	cmds = libroberto.GenAudioPipes(query, audioType)
	if restRoberto != "" {
		cmds = cmds[1:2]

		endpoint, _ := url.Parse(restRoberto)
		queryParams := url.Values{}
		queryParams.Set("token", restRobertoToken)
		queryParams.Set("text", query)
		queryParams.Set("voice", libroberto.Voice)
		endpoint.RawQuery = queryParams.Encode()

		resp, e := http.Get(endpoint.String())
		if e != nil {
			lit.Error("Error calling restRoberto: %s", e.Error())
			err = e
			return
		}

		cmds[0].Stdin = resp.Body
		out, _ = cmds[0].StdoutPipe()
	} else {
		out, _ = cmds[1].StdoutPipe()
	}

	libroberto.CmdsStart(cmds)
	return
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
				start   = time.Now()
				results = make(tb.Results, 1)
			)

			lit.Debug("%s: %s", c.Query().Sender.Username, text)

			query, isCommand, out, cmds, err := generateAudio(text)
			if err != nil {
				return nil
			}

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

			results[0] = &tb.VoiceResult{
				Cache:   send.Voice.FileID,
				Title:   query,
				Caption: "||" + replacer.Replace(query) + "||",
			}

			results[0].SetResultID(libroberto.GenUUID(query))
			results[0].SetParseMode(tb.ModeMarkdownV2)

			lit.Debug("took %s to answer query", time.Since(start).String())

			return c.Answer(&tb.QueryResponse{
				Results:   results,
				CacheTime: 86400,
			})
		}

		return nil
	})

	b.Handle("/start", func(c tb.Context) error {
		return c.Reply("Ciao! Sono un bot open source per la sintesi vocale. Il codice sorgente è disponibile su GitHub: https://github.com/TheTipo01/robertoTelegram")
	})

	b.Handle(tb.OnText, func(c tb.Context) error {
		if c.Chat().Type != tb.ChatPrivate {
			return nil
		}

		text := c.Text()
		if text == "" {
			return nil
		}

		lit.Debug("%s: %s", c.Sender().Username, text)
		_, _, out, cmds, err := generateAudio(text)
		if err != nil {
			return nil
		}

		defer libroberto.CmdsKill(cmds)
		defer libroberto.CmdsWait(cmds)

		return c.Reply(&tb.Voice{File: tb.FromReader(out), MIME: "audio/ogg"})
	})

	// Start bot
	lit.Info("robertoTelegram is now running")
	b.Start()
}
