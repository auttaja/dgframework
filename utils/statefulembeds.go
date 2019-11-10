package utils

import (
	"fmt"
	"github.com/auttaja/discordgo"
	"log"
	"math"
	"sync"
)

var sessionsHolder = newSessions()

// EmbedSession is the session object for all the stateful embed handling
type EmbedSession struct {
	embeds       []*StatefulEmbed
	message      *discordgo.Message
	Target       discordgo.Messageable
	User         *discordgo.User
	CtxData      interface{}
	currentState *StatefulEmbed
}

// StatefulEmbed is a wrapper around the discordgo embed and
// holds information needed to listen for reactions and know what to call
type StatefulEmbed struct {
	*discordgo.MessageEmbed
	Session *EmbedSession
	options []*statefulOption
}

type sessions struct {
	locker   *sync.Mutex
	sessions map[string]*EmbedSession
}

type statefulOption struct {
	Name    string
	Value   string
	Emoji   *statefulEmoji
	Handler func(*EmbedSession, *discordgo.MessageReactionAdd)
	parent  *StatefulEmbed
}

type statefulEmoji struct {
	ApiName     string
	DisplayName string
}

func newSessions() *sessions {
	return &sessions{
		locker:   &sync.Mutex{},
		sessions: make(map[string]*EmbedSession),
	}
}

// NewEmbedSession returns a new EmbedSession
func NewEmbedSession(target discordgo.Messageable, User *discordgo.User, ctx interface{}) *EmbedSession {
	return &EmbedSession{
		Target:  target,
		User:    User,
		CtxData: ctx,
	}
}

// Show creates the message and replaces the embed with the first StatefulEmbed provided to the session
func (s *EmbedSession) Show() (err error) {
	em := discordgo.
		NewEmbed().
		SetDescription("Loading...")
	m, err := s.Target.SendMessage("", em, nil)
	if err != nil {
		return
	}
	s.message = m

	sessionsHolder.locker.Lock()
	sessionsHolder.sessions[m.ID] = s
	sessionsHolder.locker.Unlock()

	return s.embeds[0].Show()
}

func (s *statefulEmoji) react(m *discordgo.Message) error {
	return m.Session.MessageReactionAdd(m.ChannelID, m.ID, s.ApiName)
}

func (s statefulEmoji) String() string {
	return s.DisplayName
}

// NewStatefulEmbed creates the StatefulEmbed object
// s  : the EmbedSession this StatefulEmbed belongs to
func NewStatefulEmbed(s *EmbedSession) *StatefulEmbed {
	em := &StatefulEmbed{
		Session:      s,
		MessageEmbed: discordgo.NewEmbed(),
	}
	em.Session.embeds = append(em.Session.embeds, em)
	return em
}

// AddField adds a field onto the embed and if emoji and handler are not nil,
// the emoji will be added as a reaction to the message which can then trigger handler
// name    : the field name
// value   : the field value
// inline  : determines if the field should be placed inline or not
// emoji   : the emoji to react with to call the handler
// handler : the handler to call
func (s *StatefulEmbed) AddField(name, value string, inline bool, emoji *discordgo.Emoji, handler func(*EmbedSession, *discordgo.MessageReactionAdd)) {
	if emoji != nil && handler != nil {
		e := &statefulEmoji{
			DisplayName: emoji.String(),
			ApiName:     emoji.APIName(),
		}
		o := &statefulOption{
			Name:    name,
			Value:   value,
			Emoji:   e,
			Handler: handler,
			parent:  s,
		}
		s.options = append(s.options, o)
		name = fmt.Sprintf("%s %s", emoji, name)
	}
	s.MessageEmbed = s.MessageEmbed.AddField(name, value, inline)
}

// AddReaction will add a reaction to the embed which can trigger handler
// emoji   : the emoji to react with to call the handler
// handler : the handler to call
func (s *StatefulEmbed) AddReaction(emoji *discordgo.Emoji, handler func(*EmbedSession, *discordgo.MessageReactionAdd)) {
	e := &statefulEmoji{
		DisplayName: emoji.String(),
		ApiName:     emoji.APIName(),
	}
	o := &statefulOption{
		Emoji:   e,
		Handler: handler,
		parent:  s,
	}
	s.options = append(s.options, o)
}

func (s *StatefulEmbed) addReactions() {
	var embedEdited bool
	err := s.Session.message.RemoveAllReactions()
	if err != nil {
		return
	}

	for _, o := range s.options {
		err = o.Emoji.react(s.Session.message)

		if err != nil {
			log.Println(err.(*discordgo.RESTError).Message)
			if err.(*discordgo.RESTError).Message != nil && err.(*discordgo.RESTError).Message.Code == 10014 {
				toRemove := -1
				for i, f := range s.Fields {
					if f.Name == fmt.Sprintf("%s %s", o.Emoji, o.Name) {
						toRemove = i
						break
					}
				}
				if toRemove >= 0 {
					s.MessageEmbed = s.RemoveField(toRemove)
					embedEdited = true

					_, _ = s.Session.Target.SendMessage(
						"",
						discordgo.NewEmbed().
							SetDescription("Oops, I did not find at least one of the emojis for this page, please review all items that should have been on here"),
						nil,
					)
				}
			} else {
				return
			}
		}
	}

	if embedEdited {
		s.Session.currentState = s
		_, err = s.Session.message.Edit(
			s.Session.message.
				NewMessageEdit().
				SetEmbed(s.MessageEmbed),
		)
	}

	return
}

// Show replaces the current embed and reactions in discord
func (s *StatefulEmbed) Show() (err error) {
	s.Session.currentState = s
	_, err = s.Session.message.Edit(
		s.Session.message.
			NewMessageEdit().
			SetEmbed(s.MessageEmbed),
	)
	if err != nil {
		return
	}

	go s.addReactions()

	return
}

// StatefulReactionHandler is the reaction add event handler for the stateful embeds
func StatefulReactionHandler(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	if s.State.MyUser().ID == r.UserID {
		return
	}

	sessionsHolder.locker.Lock()
	defer sessionsHolder.locker.Unlock()
	embedSession, ok := sessionsHolder.sessions[r.MessageID]
	if !ok {
		return
	}

	if embedSession.User.ID != r.UserID {
		_ = r.Remove()
		return
	}

	for _, o := range embedSession.currentState.options {
		if o.Emoji.ApiName == r.Emoji.APIName() {
			o.Handler(embedSession, r)
		}
	}
}

// StatefulMessageDelete is the message delete event handler for the stateful embeds
func StatefulMessageDelete(_ *discordgo.Session, m *discordgo.MessageDelete) {
	sessionsHolder.locker.Lock()
	delete(sessionsHolder.sessions, m.ID)
	sessionsHolder.locker.Unlock()
}

// PagingHandlerCTX is the context object for a paging embed
type PagingHandlerCTX struct {
	Fields      []*PagingListField
	BaseEmbed   *discordgo.MessageEmbed
	currentPage int
	parentPage  *PagingHandlerCTX
}

// PagingPage is an object describing what a page deeper will contain
type PagingPage struct {
	Emoji       *discordgo.Emoji
	Description string
	Title       string
	Fields      []*PagingListField
}

// PagingListField is one embed field for the list and if applicable contains the page that could be accessed
type PagingListField struct {
	Name     string
	Value    string
	Inline   bool
	PageLink *PagingPage
}

// NewPagingContext creates the new PagingContext based on the given list of fields
// of the top-most page and the base embed that the fields will be added to
func NewPagingContext(fields []*PagingListField, baseEmbed *discordgo.MessageEmbed) *PagingHandlerCTX {
	return &PagingHandlerCTX{
		Fields:      fields,
		BaseEmbed:   baseEmbed,
		currentPage: 1,
	}
}

// PagingEmbedHandler creates the current page of the paging embed
// s   : the EmbedSession belonging to this paging embed
// ctx : the PagingHandlerCTX
func PagingEmbedHandler(s *EmbedSession, ctx *PagingHandlerCTX) {
	em := NewStatefulEmbed(s)
	em.MessageEmbed = ctx.BaseEmbed

	pages := int(math.Ceil(float64(len(ctx.Fields)) / 8))
	start := (ctx.currentPage - 1) * 8
	end := start + 8

	var displayFields []*PagingListField
	if ctx.currentPage == pages {
		displayFields = ctx.Fields[start:]
	} else {
		displayFields = ctx.Fields[start:end]
	}

	em.Fields = nil

	if ctx.currentPage != 1 {
		em.AddField(
			"Back",
			"Goes back a page.",
			false,
			&discordgo.Emoji{Name: "⬅"},
			pageBack,
		)
	}

	if ctx.parentPage != nil {
		em.AddField(
			"Up",
			"Goes back a menu",
			false,
			&discordgo.Emoji{Name: "🔼"},
			pageUp,
		)
	}

	if ctx.currentPage != pages {
		em.AddField(
			"Forward",
			"Goes forward a page.",
			false,
			&discordgo.Emoji{Name: "➡"},
			nextPage,
		)
	}

	for _, f := range displayFields {
		if f.PageLink == nil {
			em.AddField(
				f.Name,
				f.Value,
				f.Inline,
				nil,
				nil,
			)
		} else {
			em.AddField(
				f.Name,
				f.Value,
				f.Inline,
				f.PageLink.Emoji,
				pageDeeper,
			)
		}
	}

	em.AddField(
		"Close",
		"Closes the embed.",
		false,
		&discordgo.Emoji{Name: "❌"},
		closeEmbed,
	)

	if s.message != nil {
		_ = em.Show()
	}
}

func pageDeeper(s *EmbedSession, r *discordgo.MessageReactionAdd) {
	ctx, ok := s.CtxData.(*PagingHandlerCTX)
	if !ok {
		return
	}

	for _, f := range ctx.Fields {
		if f.PageLink != nil && f.PageLink.Emoji.IsEqual(r.Emoji) {
			ctx.parentPage = &PagingHandlerCTX{}
			*ctx.parentPage = *ctx
			ctx.parentPage.Fields = ctx.Fields
			*ctx.parentPage.BaseEmbed = *ctx.BaseEmbed
			ctx.parentPage.currentPage = ctx.currentPage

			ctx.BaseEmbed = ctx.BaseEmbed.
				SetDescription(f.PageLink.Description).
				SetTitle(f.PageLink.Title)
			ctx.Fields = f.PageLink.Fields
			PagingEmbedHandler(s, ctx)
		}
	}
}

func pageUp(s *EmbedSession, _ *discordgo.MessageReactionAdd) {
	ctx, ok := s.CtxData.(*PagingHandlerCTX)
	if ok {
		*ctx = *ctx.parentPage
		*ctx.BaseEmbed = *ctx.parentPage.BaseEmbed
		ctx.currentPage = ctx.parentPage.currentPage
		ctx.Fields = ctx.parentPage.Fields
		ctx.parentPage = nil
		PagingEmbedHandler(s, ctx)
	}
}

func nextPage(s *EmbedSession, _ *discordgo.MessageReactionAdd) {
	if ctx, ok := s.CtxData.(*PagingHandlerCTX); ok {
		ctx.currentPage += 1
		PagingEmbedHandler(s, ctx)
	}
}

func pageBack(s *EmbedSession, _ *discordgo.MessageReactionAdd) {
	if ctx, ok := s.CtxData.(*PagingHandlerCTX); ok {
		ctx.currentPage -= 1
		PagingEmbedHandler(s, ctx)
	}
}

func closeEmbed(s *EmbedSession, _ *discordgo.MessageReactionAdd) {
	_ = s.message.Delete()
}
