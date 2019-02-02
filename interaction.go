package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/chris-hamper/go-slack-poll/poll"
	"github.com/nlopes/slack"
)

// interactionHandler handles interactive message response.
type interactionHandler struct {
	verificationToken string
}

func (h interactionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		log.Printf("[ERROR] Invalid method: %s", r.Method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("[ERROR] Failed to read request body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	jsonStr, err := url.QueryUnescape(string(buf)[8:])
	if err != nil {
		log.Printf("[ERROR] Failed to unespace request body: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var message slack.AttachmentActionCallback
	if err = json.Unmarshal([]byte(jsonStr), &message); err != nil {
		log.Printf("[ERROR] Failed to decode json message from slack: %s", jsonStr)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Only accept message from slack with valid token.
	// @todo - use new verification method
	if message.Token != h.verificationToken {
		log.Printf("[ERROR] Invalid token: %s", message.Token)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	action := message.Actions[0]
	parts := strings.Split(action.Name, "_")

	p := poll.GetPollByID(parts[0])
	p.ToggleVote(message.User.ID, parts[1])
	p.Save()

	replacement := &slack.Msg{
		ResponseType:    "in_channel",
		ReplaceOriginal: true,
		Attachments:     []slack.Attachment{*p.ToSlackAttachment()},
	}

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)

	w.Header().Set("Content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	err = encoder.Encode(&replacement)
	if err != nil {
		log.Println("[ERROR] JSON Encode failed:", err)
	}
}
