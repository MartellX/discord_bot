package discord

import (
	"MartellX/discord_bot/utils"
	"MartellX/discord_bot/vk"
	"github.com/bwmarrin/discordgo"
	"regexp"
	"strconv"
)

var lastMessages = map[string]*Message{}

type Message struct {
	session      *discordgo.Session
	audioSession *AudioSession
	messageId    string
	channelId    string

	pages    []*discordgo.MessageEmbed
	data     map[string]interface{}
	currPage int

	isDestroyed bool
}

var ReactEmojies = reactEmojies{
	Previous: "⬅",
	Close:    "❌",
	Next:     "➡",
}

type reactEmojies struct {
	Previous string
	Close    string
	Next     string
}

func NewMessage(session *discordgo.Session, audioSession *AudioSession, messageId string, channelId string, pages []*discordgo.MessageEmbed, info map[string]interface{}) *Message {
	session.MessageReactionAdd(channelId, messageId, ReactEmojies.Previous)
	session.MessageReactionAdd(channelId, messageId, ReactEmojies.Close)
	session.MessageReactionAdd(channelId, messageId, ReactEmojies.Next)
	m := &Message{session: session, audioSession: audioSession, messageId: messageId, channelId: channelId, pages: pages, data: info, currPage: 1}
	if last, ok := lastMessages[channelId]; ok {
		last.DestroyMessage()
	}
	lastMessages[channelId] = m
	session.AddHandlerOnce(m.handleEmotesAdd)
	session.AddHandlerOnce(m.handleEmotesDel)
	return m
}

func (m *Message) listenForMessages() {
	m.session.AddHandlerOnce(m.handleMessageForVksearch)
}

func (m *Message) handleMessageForVksearch(s *discordgo.Session, e *discordgo.MessageCreate) {
	if m.isDestroyed {
		return
	}

	if !e.Author.Bot {
		if tr, ok := m.data["tracks"]; ok {
			mess := e.Message.Content
			ok, err := regexp.MatchString("\\d+", mess)

			if ok && err == nil {
				tracks := tr.([]*vk.Track)
				selectedIndex, _ := strconv.Atoi(mess)
				if selectedIndex <= len(tracks) {
					m.audioSession.AddTrack(tracks[selectedIndex-1])

					err = m.audioSession.CheckForConnectionAndChangeVC(e.Message)
					if !m.audioSession.IsPaused && m.audioSession.AudioEncoder.Status() != utils.Run && err == nil {
						m.audioSession.Play()
					}
					m.session.ChannelMessageDelete(m.channelId, m.messageId)
					m.isDestroyed = true
					return
				}
			}
		}
	}

	m.session.AddHandlerOnce(m.handleMessageForVksearch)
}

func (m *Message) handleEmotesAdd(s *discordgo.Session, e *discordgo.MessageReactionAdd) {
	if m.isDestroyed {
		return
	}

	if e.UserID != s.State.User.ID {
		if e.ChannelID == m.channelId && e.MessageID == m.messageId {
			id := e.Emoji.ID
			name := e.Emoji.Name
			if len(m.pages) > 1 {
				if m.currPage < len(m.pages) {
					if id == ReactEmojies.Next || name == ReactEmojies.Next {
						m.currPage++
						m.session.ChannelMessageEditEmbed(m.channelId, m.messageId, m.pages[m.currPage-1])
					}
				}
				if m.currPage > 1 {
					if id == ReactEmojies.Previous || name == ReactEmojies.Previous {
						m.currPage--
						m.session.ChannelMessageEditEmbed(m.channelId, m.messageId, m.pages[m.currPage-1])
					}
				}
			}
			if id == ReactEmojies.Close || name == ReactEmojies.Close {
				m.session.ChannelMessageDelete(m.channelId, m.messageId)
				m.isDestroyed = true
				return
			}

		}
	}

	s.AddHandlerOnce(m.handleEmotesAdd)

}

func (m *Message) handleEmotesDel(s *discordgo.Session, e *discordgo.MessageReactionRemove) {
	if m.isDestroyed {
		return
	}

	if e.UserID != s.State.User.ID {
		if e.ChannelID == m.channelId && e.MessageID == m.messageId {
			id := e.Emoji.ID
			name := e.Emoji.Name
			if len(m.pages) > 1 {
				if m.currPage < len(m.pages) {
					if id == ReactEmojies.Next || name == ReactEmojies.Next {
						m.currPage++
						m.session.ChannelMessageEditEmbed(m.channelId, m.messageId, m.pages[m.currPage-1])
					}
				}
				if m.currPage > 1 {
					if id == ReactEmojies.Previous || name == ReactEmojies.Previous {
						m.currPage--
						m.session.ChannelMessageEditEmbed(m.channelId, m.messageId, m.pages[m.currPage-1])
					}
				}
			}
			if id == ReactEmojies.Close || name == ReactEmojies.Close {
				m.session.ChannelMessageDelete(m.channelId, m.messageId)
				return
			}

		}
	}

	s.AddHandlerOnce(m.handleEmotesDel)

}

func (m *Message) DestroyMessage() {
	m.session.ChannelMessageDelete(m.channelId, m.messageId)
	m.isDestroyed = true
}
