package main

import (
	"crypto/sha1"
	"encoding/base32"
	"github.com/bwmarrin/lit"
	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/spf13/viper"
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
	// Array of emoji, containing description of them
	emoji Emoji
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
		case "logerror", "error":
			lit.LogLevel = lit.LogError
			break
		case "logwarning", "warning":
			lit.LogLevel = lit.LogWarning
			break
		case "loginformational", "informational":
			lit.LogLevel = lit.LogInformational
			break
		case "logdebug", "debug":
			lit.LogLevel = lit.LogDebug
			break
		}

		// Read adjective
		foo, _ := ioutil.ReadFile("parole.txt")
		adjectives = strings.Split(string(foo), "\n")

		// Create folders used by the bot
		if _, err = os.Stat("./temp"); err != nil {
			if err = os.Mkdir("./temp", 0755); err != nil {
				lit.Error("Cannot create temp directory, %s", err)
			}
		}

		emoji = emojiLoader()

	}

}

func main() {
	// Start telegram session
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		lit.Error("Error while creating bot: %s", err)
		return
	}

	// Also start HTTP server to serve generated .mp3 files
	http.Handle("/temp/", http.StripPrefix("/temp", http.FileServer(http.Dir("./temp"))))
	go http.ListenAndServe(":8069", nil)

	lit.Info("robertoTelegram is now running on %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, _ := bot.GetUpdatesChan(u)

	for update := range updates {
		// Checks if the update is an inline query, and if it's not empty
		if update.InlineQuery != nil && update.InlineQuery.Query != "" {
			var (
				query      string
				upperQuery = strings.ToUpper(update.InlineQuery.Query)
				uuid       string
				results    []interface{}
				isCommand  = true
			)

			// Various custom command
			switch {
			case strings.HasPrefix(upperQuery, "TRENO"):
				query = strings.TrimSpace(searchAndGetTrain(strings.TrimPrefix(upperQuery, "TRENO ")))
				if query == "" {
					query = "Nessun treno trovato, agagagaga!"
				}
				break

			case strings.HasPrefix(upperQuery, "COVID"):
				query = strings.TrimSpace(getCovid())
				break

			case strings.HasPrefix(upperQuery, "BESTEMMIA"):
				query = strings.TrimSpace(bestemmia())
				break

			default:
				query = emojiToDescription(upperQuery)
				isCommand = false
			}

			uuid = genAudio(query)

			// So the title of the result isn't all uppercase when there's no command
			if !isCommand {
				query = update.InlineQuery.Query
			}

			results = append(results, tgbotapi.NewInlineQueryResultVoice(uuid, host+uuid+".mp3", query))

			// Send audio
			_, err := bot.AnswerInlineQuery(tgbotapi.InlineConfig{
				InlineQueryID: update.InlineQuery.ID,
				Results:       results,
			})

			if err != nil {
				lit.Error("error while answering inline query: %s", err)
			}

		}
	}
}

// Generates audio from a string. Checks if it already exist before generating it
func gen(text string, uuid string) {
	_, err := os.Stat("./temp/" + uuid + ".mp3")

	if err != nil {
		tts := exec.Command("balcon", "-i", "-o", "-enc", "utf8", "-n", "Roberto")
		tts.Stdin = strings.NewReader(text)
		ttsOut, _ := tts.StdoutPipe()
		_ = tts.Start()

		ffmpeg := exec.Command("ffmpeg", "-i", "pipe:0", "-f", "s16le", "-ar", "48000", "-ac", "2", "-f", "mp3", "./temp/"+uuid+".mp3")
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
