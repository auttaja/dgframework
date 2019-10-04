package dgframework

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"plugin"
	"strings"

	"github.com/auttaja/dgframework/router"
	"github.com/auttaja/discordgo"
	"github.com/bwmarrin/snowflake"
	"github.com/casbin/casbin"
	"go.mongodb.org/mongo-driver/mongo"
)

// Bot represents a Discord bot
type Bot struct {
	Session       *discordgo.Session
	DB            *mongo.Client
	Router        *router.Route
	Enforcer      *casbin.Enforcer
	snowflakeNode *snowflake.Node
}

// BotPlugin represents a plugin, it must contain an Init function
type BotPlugin interface {
	Init(*Bot)
	Name() string
}

// NewBot returns a new Bot instance
func NewBot(token, prefix string, shardID, shardCount int, dbSession *mongo.Client) (*Bot, error) {
	bot := new(Bot)

	dg, err := discordgo.New(token)
	if err != nil {
		return nil, err
	}
	dg.ShardID = shardID
	dg.ShardCount = shardCount
	bot.Router = router.New()
	dg.AddHandler(func(_ *discordgo.Session, m *discordgo.MessageCreate) {
		_ = bot.Router.FindAndExecute(dg, prefix, dg.State.User.ID, m.Message)
	})
	dg.AddHandler(bot.ready)
	bot.Session = dg
	bot.DB = dbSession

	node, err := snowflake.NewNode(int64(shardID))
	if err != nil {
		return nil, err
	}
	bot.snowflakeNode = node

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
			continue
		}
		symPlugin, err := plug.Lookup("BotPlugin")
		if err != nil {
			fmt.Println("Unable to load BotPlugin symbol:", err)
			continue
		}
		botPlugin, ok := symPlugin.(BotPlugin)
		if !ok {
			fmt.Println("expected BotPlugin, got unknown type:", err)
			continue
		}
		botPlugin.Init(b)
		fmt.Println("Loaded plugin ", botPlugin.Name())
	}

	return nil
}

func (b *Bot) ready(s *discordgo.Session, r *discordgo.Ready) {
	log.Printf("%s is now ready", s.State.User.Username)
	for _, guild := range r.Guilds {
		if guild.Large {
			err := s.RequestGuildMembers(guild.ID, "", 1000)
			if err != nil {
				fmt.Println("Error requesting guild members: ", err)
				return
			}
		}
	}
}

func (b *Bot) processGuildMembersChunk(s *discordgo.Session, c *discordgo.GuildMembersChunk) {
	fmt.Printf("Processing %d members for %s\n", len(c.Members), c.GuildID)
}

// GenerateSnowflake generates an internal snowflake that can be used to produce unique IDs
func (b *Bot) GenerateSnowflake() snowflake.ID {
	return b.snowflakeNode.Generate()
}
