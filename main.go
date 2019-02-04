package main

import (
	"encoding/csv"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/chris-hamper/go-slack-poll/poll"
	"github.com/nlopes/slack"
)

func main() {
	// secret := os.Getenv("SLACK_SIGNING_SECRET")
	verificationToken := os.Getenv("SLACK_VERIFICATION_TOKEN")
	verificationToken = strings.TrimSpace(verificationToken)

	// @todo - move to separate handler file
	http.HandleFunc("/command", func(w http.ResponseWriter, r *http.Request) {
		// @todo - sanitization?
		cmd, err := slack.SlashCommandParse(r)
		if err != nil {
			log.Println("[ERROR] SlashCommandParse failed:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Println("[DEBUG] command:", cmd)

		// @todo - Use newer signing approach instead.
		if !cmd.ValidateToken(verificationToken) {
			log.Printf("[ERROR] Invalid token: '%s'", cmd.Token)
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
				log.Println("[ERROR] Command text split failed:", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			// Create the poll.
			p := poll.CreatePoll(cmd.UserID, args[0], args[1:])
			p.Save()

			params := &slack.Msg{
				ResponseType: "in_channel",
				Attachments:  []slack.Attachment{*p.ToSlackAttachment()},
			}

			b, err := json.Marshal(params)
			if err != nil {
				log.Println("[ERROR] JSON Marshal failed:", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(b)

		default:
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})

	// Register handler to receive interactive messages from slack.
	http.Handle("/interaction", interactionHandler{
		verificationToken: verificationToken,
	})

	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	log.Println("[INFO] Server listening on port 3000")
	http.ListenAndServe(":3000", nil)
}

func normalizeQuotes(r rune) rune {
	switch r {
	case '“', '”':
		return '"'
	}
	return r
}
