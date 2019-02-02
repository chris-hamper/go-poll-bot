package poll

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/mediocregopher/radix.v2/pool"
	"github.com/nlopes/slack"
)

// Voters represents a set of votes from the users specified in the keys.
type Voters map[string]int

// Votes maps an option title to Voters who have voted for it.
type Votes map[string]Voters

// Poll holds all information related to a poll created via Slack.
type Poll struct {
	ID    string
	Owner string
	Title string
	Votes Votes // Maps option title to Voters who have voted for it.
}

var db *pool.Pool

func init() {
	var err error
	// Establish a pool of 10 connections to the Redis server listening on
	// port 6379 of the local machine.
	db, err = pool.New("tcp", "localhost:6379", 10)
	if err != nil {
		log.Panic(err)
	}
}

func CreatePoll(owner, title string, options []string) *Poll {
	id, err := db.Cmd("INCR", "next-poll").Int()
	if err != nil {
		log.Println("[ERROR] Can't get next poll ID:", err)
		return nil
	}

	poll := Poll{ID: strconv.Itoa(id), Owner: owner, Title: title, Votes: make(Votes)}
	for _, option := range options {
		poll.Votes[option] = make(Voters)
	}
	log.Println("[INFO] CreatePoll:", poll)
	return &poll
}

func GetPollByID(id string) *Poll {
	s, err := db.Cmd("GET", "poll:"+id).Str()
	if err != nil {
		log.Println("[ERROR] Can't get poll from Redis store:", err)
		return nil
	}

	var p Poll
	dec := json.NewDecoder(strings.NewReader(s))
	err = dec.Decode(&p)
	if err != nil {
		log.Println("[ERROR] Can't decode poll:", err)
		return nil
	}
	return &p
}

func (p Poll) Save() {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.Encode(p)
	s := string(b.Bytes())
	log.Println("[INFO] Saving poll to Redis store:", s)

	pollKey := "poll:" + p.ID
	err := db.Cmd("SET", pollKey, s).Err
	if err != nil {
		log.Println("[ERROR] Can't save poll", pollKey, "to Redis store:", err)
	}
}

func (p *Poll) ToggleVote(user, option string) {
	log.Println("[INFO] toggleVote:", user, option)
	_, ok := p.Votes[option]
	if !ok {
		log.Println("[ERROR] No 'option' in p.Votes for:", option)
		return
	}

	_, voted := p.Votes[option][user]
	if voted {
		// Revoke the vote.
		delete(p.Votes[option], user)
	} else {
		// Cast the vote.
		p.Votes[option][user] = 1
	}
}

func (p Poll) ToSlackAttachment() *slack.Attachment {
	actions := make([]slack.AttachmentAction, len(p.Votes))
	fields := make([]slack.AttachmentField, len(p.Votes))

	i := 0
	prefix := p.ID + "_"
	for optionTitle, voters := range p.Votes {
		actions[i] = slack.AttachmentAction{
			Name: prefix + optionTitle,
			Text: optionTitle,
			Type: "button",
		}

		votersStr := ""
		for userID := range voters {
			votersStr += fmt.Sprintf("<@%v> ", userID)
		}

		fields[i] = slack.AttachmentField{
			Title: fmt.Sprintf("%v (%v)", optionTitle, len(voters)),
			Value: votersStr,
			Short: false,
		}
		i++
	}

	return &slack.Attachment{
		Title:      "Poll: " + p.Title,
		Fallback:   "Please use a client that supports interactive messages to see this poll.",
		CallbackID: p.ID,
		Fields:     fields,
		Actions:    actions,
	}
}
