package dgframework

import (
	"github.com/auttaja/dgframework/router"
	"github.com/auttaja/discordgo"
	"os"
	"testing"
	"time"
)

//////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////// VARS NEEDED FOR TESTING
var (
	bot *Bot

	envToken   = os.Getenv("DG_TOKEN")   // Token to use when authenticating the bot account
	envGuild   = os.Getenv("DG_GUILD")   // Guild ID to use for tests
	envChannel = os.Getenv("DG_CHANNEL") // Channel ID to use for tests
)

//////////////////////////////////////////////////////////////////////////////
/////////////////////////////////////////////////////////////// START OF TESTS

func TestBotCreation(t *testing.T) {
	if envToken == "" {
		t.Skip("Skipping TestBotCreation, DG_TOKEN not set")
	}

	newBot, err := NewBotBuilder(envToken).SetPrefix("?").UseStatefulEmbeds().AutoStartBot().Build()
	if err != nil {
		t.Fatal("Building Bot failed", err)
	}
	bot = newBot
}

func TestAddCommand(t *testing.T) {
	if envChannel == "" {
		t.Skip("Skipping, DG_CHANNEL not set.")
	}

	c, err := bot.Session.Channel(envChannel)
	if err != nil {
		t.Skipf("Channel %s wasn't cached", envChannel)
	}

	bot.Router.On("testAddCommand", func(context *router.Context) error {
		_, _ = context.SendMessage("testing adding a command", nil, nil)
		return nil
	})

	command := bot.Router.Find("testAddCommand")
	if command == nil {
		t.Fatal("testAddCommand failed to be added")
	}

	err = command.Handler(&router.Context{
		Channel: c,
	})
	if err != nil {
		t.Fatal("testAddCommand failed at running", err)
	}
}

func TestFindAndExecute(t *testing.T) {
	if envChannel == "" {
		t.Skip("Skipping, DG_CHANNEL not set.")
	}

	bot.Router.On("testAddCommand", func(context *router.Context) error {
		_, _ = context.SendMessage("testing adding a command", nil, nil)
		return nil
	})

	m := &discordgo.Message{
		ID:        "1",
		GuildID:   envGuild,
		ChannelID: envChannel,
		Content:   "?testAddCommand",
		Session:   bot.Session,
	}
	err := bot.Router.FindAndExecute(bot.Session, "?", bot.Session.State.User.ID, m)
	if err != nil {
		t.Fatal("testFindAndExecute failed at running", err)
	}
}

func TestPanicRecovery(t *testing.T) {
	bot.Router.On("testPanicRecovery", func(context *router.Context) error {
		panic("testing panic recovery")
		return nil
	})

	m := &discordgo.Message{
		ID:        "1",
		GuildID:   envGuild,
		ChannelID: envChannel,
		Content:   "?testPanicRecovery",
		Session:   bot.Session,
	}
	err := bot.Router.FindAndExecute(bot.Session, "?", bot.Session.State.User.ID, m)
	if err != nil {
		t.Fatal("testPanicRecovery failed at running", err)
	}
}

func TestWaitFor(t *testing.T) {
	if envChannel == "" {
		t.Skip("Skipping, DG_CHANNEL not set.")
	}

	c, err := bot.Session.Channel(envChannel)
	if err != nil {
		t.Skipf("Channel %s wasn't cached", envChannel)
	}

	content := "testing WaitFor"

	w := WaitFor(bot.Session, discordgo.MessageCreate{}, func(i interface{}) bool {
		return i.(*discordgo.MessageCreate).Author.ID == bot.Session.State.User.ID &&
			i.(*discordgo.MessageCreate).ChannelID == envChannel
	})

	_, _ = c.SendMessage(content, nil, nil)

	var resp interface{}
	select {
	case resp = <-w.Response:
	case <-time.After(time.Second):
		t.Fatal("timeout on WaitFor")
	}

	if _, ok := resp.(*discordgo.MessageCreate); !ok {
		t.Fatal("Event received is not a MessageCreate")
	}

	if resp.(*discordgo.MessageCreate).Content != content {
		t.Fatal("Did not receive correct message from WaitFor")
	}
}
