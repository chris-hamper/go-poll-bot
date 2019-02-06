package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chris-hamper/go-slack-poll/poll"
	"github.com/nlopes/slack"
)

//var smartQuoteReplacer = strings.NewReplacer("“", "\"", "”", "\"")

var signingSecret []byte

func main() {
	signingSecret = []byte(strings.TrimSpace(os.Getenv("SLACK_SIGNING_SECRET")))

	// @todo - move to separate handler file
	http.HandleFunc("/command", func(w http.ResponseWriter, r *http.Request) {
		if !validateRequest(r) {
			log.Printf("[ERROR] Message validation failed")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// @todo - sanitization?
		cmd, err := slack.SlashCommandParse(r)
		if err != nil {
			log.Println("[ERROR] SlashCommandParse failed:", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Println("[DEBUG] Command:", cmd)

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
		signingSecret: signingSecret,
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

func validateRequest(r *http.Request) bool {
	timestamp := r.Header["X-Slack-Request-Timestamp"][0]

	// Verify the timestamp is less than 5 minutes old, to avoid replay attacks.
	now := time.Now().Unix()
	messageTime, err := strconv.ParseInt(timestamp, 0, 64)
	if err != nil {
		log.Println("[ERROR] Invalid timestamp:", timestamp)
		return false
	}
	if math.Abs(float64(now-messageTime)) > 5*60 {
		log.Println("[ERROR] Timestamp is from > 5 minutes from now")
		return false
	}

	// Get the signature and signing version from the HTTP header.
	parts := strings.Split(r.Header["X-Slack-Signature"][0], "=")
	if parts[0] != "v0" {
		log.Println("[ERROR] Unsupported signing version:", parts[0])
		return false
	}
	signature, err := hex.DecodeString(parts[1])
	if err != nil {
		log.Println("[ERROR] Invalid message signature:", parts[1])
		return false
	}

	// Read the request body.
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Println("[ERROR] Can't read request body:", err)
		return false
	}

	// Generate the HMAC hash.
	prefix := fmt.Sprintf("v0:%v:", timestamp)

	hash := hmac.New(sha256.New, []byte(signingSecret))
	hash.Write([]byte(prefix))
	hash.Write(body)

	// Reset the request body so it can be read again later.
	r.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	// Verify our hash matches the signature.
	return hmac.Equal(hash.Sum(nil), []byte(signature))
}
