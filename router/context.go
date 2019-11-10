package router

import (
	"fmt"
	"sync"

	"github.com/auttaja/discordgo"
)

// Context represents a command context
type Context struct {
	// Route is the route that this command came from
	Route   *Route
	Msg     *discordgo.Message
	Channel *discordgo.Channel
	Ses     *discordgo.Session

	// List of arguments supplied with the command
	Args Args

	// Vars that can be optionally set using the Set and Get functions
	vmu  sync.RWMutex
	Vars map[string]interface{}
}

// Set sets a variable on the context
func (c *Context) Set(key string, d interface{}) {
	c.vmu.Lock()
	c.Vars[key] = d
	c.vmu.Unlock()
}

// Get retrieves a variable from the context
func (c *Context) Get(key string) interface{} {
	if c, ok := c.Vars[key]; ok {
		return c
	}
	return nil
}

// Reply replies to the sender with the given message
func (c *Context) Reply(args ...interface{}) (*discordgo.Message, error) {
	return c.Ses.ChannelMessageSend(c.Msg.ChannelID, fmt.Sprint(args...))
}

// ReplyEmbed replies to the sender with an embed
func (c *Context) ReplyEmbed(embed *discordgo.MessageEmbed) (*discordgo.Message, error) {
	return c.Channel.SendMessage("", embed, nil)
}

// Guild returns the guild the context originated from if it did, else an error
func (c *Context) Guild() (g *discordgo.Guild, err error) {
	return c.Channel.Guild()
}

// Author returns the User that sent the message
func (c *Context) Author() *discordgo.User {
	return c.Msg.Author
}

// GetGuild retrieves a guild from the state or restapi
func (c *Context) GetGuild(guildID string) (*discordgo.Guild, error) {
	g, err := c.Ses.State.Guild(guildID)
	if err != nil {
		g, err = c.Ses.Guild(guildID)
	}
	return g, err
}

// GetChannel retrieves a channel from the state or restapi
func (c *Context) GetChannel(channelID string) (*discordgo.Channel, error) {
	ch, err := c.Ses.State.Channel(channelID)
	if err != nil {
		ch, err = c.Ses.Channel(channelID)
	}
	return ch, err
}

// GetMember retrieves a member from the state or restapi
func (c *Context) GetMember(guildID, userID string) (*discordgo.Member, error) {
	m, err := c.Ses.State.Member(guildID, userID)
	if err != nil {
		m, err = c.Ses.GuildMember(guildID, userID)
	}
	return m, err
}

// SendMessage sends a message to the channel
func (c Context) SendMessage(content string, embed *discordgo.MessageEmbed, files []*discordgo.File) (message *discordgo.Message, err error) {
	return c.Channel.SendMessage(content, embed, files)
}

// SendMessageComplex sends a message to the channel
func (c Context) SendMessageComplex(data *discordgo.MessageSend) (message *discordgo.Message, err error) {
	return c.Channel.SendMessageComplex(data)
}

// EditMessage edits an existing message, replacing it entirely with
// the given MessageEdit struct
func (c Context) EditMessage(data *discordgo.MessageEdit) (edited *discordgo.Message, err error) {
	return c.Channel.EditMessage(data)
}

// FetchMessage fetches a message with the given ID from the context channel
func (c Context) FetchMessage(ID string) (message *discordgo.Message, err error) {
	return c.Channel.FetchMessage(ID)
}

// GetHistory fetches up to limit messages from the context channel
func (c Context) GetHistory(limit int, beforeID, afterID, aroundID string) (st []*discordgo.Message, err error) {
	return c.Channel.GetHistory(limit, beforeID, afterID, aroundID)
}

// GetHistoryIterator returns a bare HistoryIterator for the context channel.
func (c Context) GetHistoryIterator() *discordgo.HistoryIterator {
	return c.Channel.GetHistoryIterator()
}

// NewContext returns a new context from a message
func NewContext(s *discordgo.Session, m *discordgo.Message, args Args, route *Route) *Context {
	return &Context{
		Route:   route,
		Msg:     m,
		Channel: m.Channel(),
		Ses:     s,
		Args:    args,
		Vars:    map[string]interface{}{},
	}
}
