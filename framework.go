package dgframework

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"time"

	"github.com/auttaja/dgframework/utils"
	"github.com/auttaja/dgframework/x/discordrolemanager"

	"github.com/auttaja/dgframework/router"
	"github.com/auttaja/discordgo"
	"github.com/bwmarrin/snowflake"
	"github.com/casbin/casbin"
	mongodbadapter "github.com/casbin/mongodb-adapter"
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

// BotBuilder is a convenience struct for making the Bot object
type BotBuilder struct {
	token             string
	prefix            string
	shardID           int
	shardCount        int
	pluginLocation    string
	dbSession         *mongo.Client
	useStatefulEmbeds bool
	startBot          bool
	casbinDBURL       string
}

// BotPlugin represents a plugin, it must contain an Init function
type BotPlugin interface {
	Init(*Bot)
	Name() string
}

// NewBotBuilder returns a new BotBuilder object with a few default values already filled in
func NewBotBuilder(token string) *BotBuilder {
	return &BotBuilder{
		token:      token,
		prefix:     "-",
		shardCount: 1,
	}
}

// SetPrefix sets another prefix than the default
func (b *BotBuilder) SetPrefix(prefix string) *BotBuilder {
	b.prefix = prefix
	return b
}

// SetShards sets a different shard ID and shard count than the default 0 and 1 respectively
func (b *BotBuilder) SetShards(shardID, shardCount int) *BotBuilder {
	b.shardCount = shardCount
	b.shardID = shardID
	return b
}

// SetDBSession sets the DB Session in the form of a mongo.Client
func (b *BotBuilder) SetDBSession(session *mongo.Client) *BotBuilder {
	b.dbSession = session
	return b
}

// UseStatefulEmbeds will make the builder also add the handlers needed for the statefulembeds from utils to work
func (b *BotBuilder) UseStatefulEmbeds() *BotBuilder {
	b.useStatefulEmbeds = true
	return b
}

// SetPluginLocation sets the location with the plugins and will make the builder also load the plugins
func (b *BotBuilder) SetPluginLocation(location string) *BotBuilder {
	b.pluginLocation = location
	return b
}

// AutoStartBot makes the bot automatically start when it gets build
func (b *BotBuilder) AutoStartBot() *BotBuilder {
	b.startBot = true
	return b
}

// SetCasbinDBURL sets the DB URL for Casbin to use for policy storage
func (b *BotBuilder) SetCasbinDBURL(URL string) *BotBuilder {
	b.casbinDBURL = URL
	return b
}

// Build will build the bot using the provided information in the BotBuilder
func (b *BotBuilder) Build() (bot *Bot, err error) {
	bot, err = NewBot(b.token, b.prefix, b.shardID, b.shardCount, b.dbSession, b.casbinDBURL)
	if err != nil {
		return
	}

	if b.dbSession != nil {
		dbContext, _ := context.WithTimeout(context.Background(), 5*time.Second)
		err = bot.DB.Connect(dbContext)
		if err != nil {
			return
		}
	}

	if b.useStatefulEmbeds {
		bot.Session.AddHandler(utils.StatefulMessageDelete)
		bot.Session.AddHandler(utils.StatefulReactionHandler)
	}

	if b.pluginLocation != "" {
		err = bot.LoadPlugins(b.pluginLocation)
		if err != nil {
			return
		}
	}

	if b.startBot {
		err = bot.Session.Open()
	}

	return
}

// NewBot returns a new Bot instance
func NewBot(token, prefix string, shardID, shardCount int, dbSession *mongo.Client, casbinMongoURL string) (*Bot, error) {
	bot := new(Bot)

	dg, err := discordgo.New(token)
	if err != nil {
		return nil, err
	}

	dg.LogLevel = discordgo.LogDebug

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

	if casbinMongoURL != "" {
		bot.Enforcer = casbin.NewEnforcer("rbac/role_model.conf")
		a := mongodbadapter.NewAdapter(casbinMongoURL)
		bot.Enforcer.SetAdapter(a)

		rm := discordrolemanager.NewRoleManager(dg)
		bot.Enforcer.SetRoleManager(rm)
	}

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
