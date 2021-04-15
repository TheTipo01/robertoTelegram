package main

import (
	"crypto/sha1"
	"encoding/base32"
	"github.com/bwmarrin/lit"
	"github.com/spf13/viper"
	tb "gopkg.in/tucnak/telebot.v2"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

var (
	// Telegram token
	token string
	// HTTP server where we host .mp3
	host string
	// Array of adjectives
	adjectives []string
	// Gods
	gods = []string{"Dio", "Gesù", "Madonna"}
	// Emoji string replacer, replacing every emoji with it's description
	emoji *strings.Replacer
)

const (
	audioExtension = ".mp3"
)

func init() {
	// Initialize rand
	rand.Seed(time.Now().Unix())

	lit.LogLevel = lit.LogError

	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found
			lit.Error("Config file not found! See example_config.yml")
			return
		}
	} else {
		// Config file found
		token = viper.GetString("token")
		host = viper.GetString("host")

		// Set lit.LogLevel to the given value
		switch strings.ToLower(viper.GetString("loglevel")) {
		case "logwarning", "warning":
			lit.LogLevel = lit.LogWarning

		case "loginformational", "informational":
			lit.LogLevel = lit.LogInformational

		case "logdebug", "debug":
			lit.LogLevel = lit.LogDebug
		}

		initializeAdjectives()

		// Create folders used by the bot
		if _, err = os.Stat("./temp"); err != nil {
			if err = os.Mkdir("./temp", 0755); err != nil {
				lit.Error("Cannot create temp directory, %s", err)
			}
		}

		emoji = emojiReplacer()

	}

}

func main() {
	// Start HTTP server to serve generated .mp3 files
	http.Handle("/temp/", http.StripPrefix("/temp", http.FileServer(http.Dir("./temp"))))
	go http.ListenAndServe(":8069", nil)

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

			lit.Info("%s: %s", q.From.Username, q.Text)

			// Various custom command
			switch {
			case strings.HasPrefix(upperQuery, "TRENO"):
				query = strings.TrimSpace(searchAndGetTrain(strings.TrimPrefix(upperQuery, "TRENO ")))
				if query == "" {
					query = "Nessun treno trovato, agagagaga!"
				}

			case strings.HasPrefix(upperQuery, "COVID"):
				query = strings.TrimSpace(getCovid())

			case strings.HasPrefix(upperQuery, "BESTEMMIA"):
				query = strings.TrimSpace(bestemmia())

			default:
				query = emojiToDescription(upperQuery)
				isCommand = false
			}

			uuid = genAudio(query)

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

// Generates audio from a string. Checks if it already exist before generating it
func gen(text string, uuid string) {
	_, err := os.Stat("./temp/" + uuid + audioExtension)

	if err != nil {
		tts := exec.Command("balcon", "-i", "-o", "-enc", "utf8", "-n", "Roberto")
		tts.Stdin = strings.NewReader(text)
		ttsOut, _ := tts.StdoutPipe()
		_ = tts.Start()

		ffmpeg := exec.Command("ffmpeg", "-i", "pipe:0", "-f", "s16le", "-ar", "48000", "-ac", "2", "-f", "mp3", "./temp/"+uuid+audioExtension)
		ffmpeg.Stdin = ttsOut
		_ = ffmpeg.Run()

		_ = tts.Wait()
	}
}

// genAudio generates a mp3 file from a string, returning it's UUID (aka SHA1 hash of the text)
func genAudio(text string) string {
	h := sha1.New()
	h.Write([]byte(text))
	uuid := strings.ToUpper(base32.HexEncoding.EncodeToString(h.Sum(nil)))

	gen(text, uuid)

	return uuid
}

// Generates a bestemmia
func bestemmia() string {
	s1 := gods[rand.Intn(len(gods))]

	s := s1 + " " + adjectives[rand.Intn(len(adjectives))]

	if s1 == gods[2] {
		s = s[:len(s)-2] + "a"
	}

	return s
}

// Reads adjectives
func initializeAdjectives() {
	foo, _ := ioutil.ReadFile("parole.txt")
	adjectives = strings.Split(string(foo), "\n")
}
