package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/nlopes/slack"
)

func main() {
	// secret := os.Getenv("SLACK_SIGNING_SECRET")
	verificationToken := os.Getenv("SLACK_VERIFICATION_TOKEN")

	http.HandleFunc("/command", func(w http.ResponseWriter, r *http.Request) {
		cmd, err := slack.SlashCommandParse(r)
		if err != nil {
			fmt.Println("[ERROR] SlashCommandParse failed:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// @todo - Use newer signing approach instead.
		if !cmd.ValidateToken(verificationToken) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch cmd.Command {
		case "/pollbot":
			// Clean up "smart quotes".
			text := strings.Map(normalizeQuotes, cmd.Text)

			// Split command text on spaces, except inside quotes.
			csv := csv.NewReader(strings.NewReader(text))
			csv.Comma = ' '
			args, err := csv.Read()
			if err != nil {
				fmt.Println("[ERROR] Command text split failed:", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			actions := make([]slack.AttachmentAction, len(args)-1)
			fields := make([]slack.AttachmentField, len(args)-1)
			for i, arg := range args[1:] {
				actions[i] = slack.AttachmentAction{
					Name: strconv.Itoa(i),
					Text: arg,
					Type: "button",
				}

				fields[i] = slack.AttachmentField{
					Title: arg + " (0)",
					Value: "(no votes yet)",
					Short: false,
				}
			}

			params := &slack.Msg{
				ResponseType: "in_channel",
				Attachments: []slack.Attachment{
					{
						Title:      "Poll: " + args[0],
						Fallback:   "Please use a client that supports interactive messages to see this poll.",
						CallbackID: "fix me!",
						Fields:     fields,
						Actions:    actions,
					},
				},
			}

			b, err := json.Marshal(params)
			if err != nil {
				fmt.Println("[ERROR] JSON Marshal failed:", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(b)
			fmt.Println("[DEBUG] JSON message:", string(b))

		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})

	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	fmt.Println("[INFO] Server listening on port 3000")
	http.ListenAndServe(":3000", nil)
}

func normalizeQuotes(r rune) rune {
	switch r {
	case '“', '”':
		return '"'
	}
	return r
}
