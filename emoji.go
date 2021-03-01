package main

import (
	"encoding/json"
	"fmt"
	"github.com/forPelevin/gomoji"
	"io/ioutil"
	"os"
	"strings"
)

func emojiLoader() Emoji {
	var (
		emojiJSON Emoji
	)

	jsonFile, err := os.Open("emoji.json")
	if err != nil {
		fmt.Println("Error opening file: ", err)
		return emojiJSON
	}

	byteValue, _ := ioutil.ReadAll(jsonFile)
	_ = jsonFile.Close()
	_ = json.Unmarshal(byteValue, &emojiJSON)

	byteValue = nil

	return emojiJSON
}

func emojiToDescription(str string) string {
	if gomoji.ContainsEmoji(str) {
		for _, e := range emoji {
			str = strings.ReplaceAll(str, e.Emoji, e.Descrizione)
		}
	}

	return str
}
