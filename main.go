package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/nlopes/slack"
)

func main() {
	// key := os.Getenv("SLACK_API_KEY")

	http.HandleFunc("/command/pollbot", func(w http.ResponseWriter, r *http.Request) {
		cmd, err := slack.SlashCommandParse(r)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		//		if !cmd.ValidateToken(verificationToken) {
		//			w.WriteHeader(http.StatusUnauthorized)
		//			return
		//		}

		switch cmd.Command {
		case "/echo":
			params := &slack.Msg{Text: cmd.Text}
			fmt.Println(params)
			b, err := json.Marshal(params)
			if err != nil {
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

	fmt.Println("[INFO] Server listening")
	http.ListenAndServe(":3000", nil)
}
