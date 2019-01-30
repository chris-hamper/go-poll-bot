package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/nlopes/slack"
)

func main() {
	// secret := os.Getenv("SLACK_SIGNING_SECRET")
	verificationToken := os.Getenv("SLACK_VERIFICATION_TOKEN")

	http.HandleFunc("/command", func(w http.ResponseWriter, r *http.Request) {
		cmd, err := slack.SlashCommandParse(r)
		if err != nil {
			fmt.Println("[ERROR] SlashCommandParse failed!")
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// @todo - Use newer signing approach instead.
		if !cmd.ValidateToken(verificationToken) {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch cmd.Command {
		case "/echo":
			params := &slack.Msg{Text: cmd.Text}
			fmt.Println(params)
			b, err := json.Marshal(params)
			if err != nil {
				fmt.Println("[ERROR] JSON Marshal failed!")
				fmt.Println(err)
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

	fmt.Println("[INFO] Server listening on port 3000")
	http.ListenAndServe(":3000", nil)
}
