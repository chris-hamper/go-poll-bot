package main

import (
	"fmt"
	"hash/crc64"
	"log"

	"github.com/nlopes/slack"
)

// Voters represents a vote from the user specified in the key.
type Voters map[string]int

// Votes maps option title to Voters who have voted for it.
type Votes map[string]Voters

// Poll holds all information related to a poll created via Slack.
type Poll struct {
	ID         string
	Owner      string
	Title      string
	Votes      Votes // Maps option title to Voters who have voted for it.
}

// type PollOption struct {
// 	Title  string
// 	Voters map[string]int // "Set" of user IDs who have voted
// }

var polls = make(map[string]Poll)
var crc64Table = crc64.MakeTable(crc64.ISO)

func createPoll(id, owner, title string, options []string) *Poll {
	// Shorten ID by "hashing" it via CRC-64
	id = fmt.Sprintf("%016x", crc64.Checksum([]byte(id), crc64Table))

	poll := Poll{ID: id, Owner: owner, Title: title, Votes: make(Votes)}
	for _, option := range options {
		poll.Votes[option] = make(Voters)
	}

	polls[id] = poll
	return &poll
}

func getPollByID(id string) Poll {
	return polls[id]
}

func (p *Poll) toggleVote(user, option string) {
	log.Println("[DEBUG] toggleVote:", user, option)
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

func (p Poll) toSlackAttachment() *slack.Attachment {
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
			votersStr += fmt.Sprintf("%v ", userID)
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
