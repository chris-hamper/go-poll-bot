package poll

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/mediocregopher/radix.v2/pool"
	"github.com/mediocregopher/radix.v2/redis"
	"github.com/nlopes/slack"
)

// Votes maps an option title to a slice of Voters who have voted for it.
type Votes map[string]([]string)

// Poll holds all information related to a poll created via Slack.
type Poll struct {
	ID    string
	Owner string
	Title string
	Votes Votes
	// muxVotes Protects "Votes" from being modified in parallel.
	muxVotes sync.Mutex
}

var db *pool.Pool

func init() {
	var err error
	redisHost := os.Getenv("REDIS_HOST")
	_, hasAuth := os.LookupEnv("REDIS_PASSWORD")

	if hasAuth {
		db, err = pool.NewCustom("tcp", redisHost+":6379", 10, authDial)
	} else {
		db, err = pool.New("tcp", redisHost+":6379", 10)
	}
	if err != nil {
		log.Panic("Redis pool connections failed:", err)
	}
}

func authDial(network, addr string) (*redis.Client, error) {
	passwd := os.Getenv("REDIS_PASSWORD")

	client, err := redis.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	if err = client.Cmd("AUTH", passwd).Err; err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}

// CreatePoll creates a new Poll.
func CreatePoll(owner, title string, options []string) *Poll {
	id, err := db.Cmd("INCR", "next-poll").Int()
	if err != nil {
		log.Println("[ERROR] Can't get next poll ID:", err)
		return nil
	}

	poll := Poll{
		ID: strconv.Itoa(id),
		Owner: owner,
		Title: title,
		Votes: make(Votes),
	}
	for _, option := range options {
		poll.Votes[option] = make([]string, 0, 10)
	}
	log.Println("[INFO] CreatePoll:", poll)
	return &poll
}

// GetPollByID gets the Poll with the given ID from the database, or nil.
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

// Save stores the Poll in the database.
func (p *Poll) Save() {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)

	enc.Encode(p)
	s := b.String()
	log.Println("[INFO] Saving poll to Redis store:", s)

	pollKey := "poll:" + p.ID
	err := db.Cmd("SET", pollKey, s).Err
	if err != nil {
		log.Println("[ERROR] Can't save poll", pollKey, "to Redis store:", err)
	}
}

// ToggleVote inverts the voting status for the given user on a given option.
func (p *Poll) ToggleVote(user, option string) {
	log.Println("[INFO] toggleVote:", user, option)

	p.muxVotes.Lock()
	defer p.muxVotes.Unlock()
	voters, ok := p.Votes[option]
	if !ok {
		log.Println("[ERROR] No 'option' in p.Votes for:", option)
		return
	}

	for i, voter := range voters {
		if voter == user {
			// Remove voter from the list.
			p.Votes[option] = append(voters[:i], voters[i+1:]...)
			return
		}
	}
	
	// User wasn't found in the list of voters, so append it.
	p.Votes[option] = append(voters, user)
}

// ToSlackAttachment renders a Poll into a Slack message Attachment.
func (p *Poll) ToSlackAttachment() *slack.Attachment {
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
		for _, userID := range voters {
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
