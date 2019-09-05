package dgframework

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"strings"

	"github.com/auttaja/dgframework/router"
	"github.com/auttaja/discordgo"
	"github.com/casbin/casbin"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

// Bot represents a Discord bot
type Bot struct {
	Session  *discordgo.Session
	DB       *r.Session
	Router   *router.Route
	Enforcer *casbin.Enforcer
}

// BotPlugin represents a plugin, it must contain an Init function
type BotPlugin interface {
	Init(*Bot)
	Name() string
}

// NewBot returns a new Bot instance
func NewBot(token, prefix string, shardID, shardCount int, dbSession *r.Session) (*Bot, error) {
	bot := new(Bot)
	dg, err := discordgo.New(token)
	if err != nil {
		return nil, err
	}
	dg.ShardID = shardID
	dg.ShardCount = shardCount
	bot.Router = router.New()
	bot.Session = dg
	bot.DB = dbSession

	return bot, nil
}

// LoadPlugins loads all bot plugins at the given location
func (b *Bot) LoadPlugins(location string) error {
	var plugins []string
	err := filepath.Walk(location, func(path string, info os.FileInfo, err error) error {
		if strings.HasSuffix(path, ".so") {
			plugins = append(plugins, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	for _, pluginPath := range plugins {
		plug, err := plugin.Open(pluginPath)
		if err != nil {
			fmt.Println("Failed to load plugin:", pluginPath, ":", err)
		} else {
			symPlugin, err := plug.Lookup("BotPlugin")
			if err != nil {
				fmt.Println(err)
			} else {
				botPlugin, ok := symPlugin.(BotPlugin)
				if !ok {
					fmt.Println("expected BotPlugin, got unknown type", err)
				} else {
					botPlugin.Init(b)
					fmt.Println("Loaded plugin ", botPlugin.Name())
				}
			}
		}
	}

	return nil
}
