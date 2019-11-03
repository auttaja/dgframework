package router

import (
	"errors"
	"fmt"
	"github.com/auttaja/discordgo"
	"github.com/getsentry/sentry-go"
	"strings"
)

// Error variables
var (
	// ErrCouldNotFindRoute gets returned when a route could not be found
	ErrCouldNotFindRoute = errors.New("could not find route")

	// ErrRouteAlreadyExists gets returned when a route gets added that already exists
	ErrRouteAlreadyExists = errors.New("route already exists")

	// ErrInvalidArgument should get returned by the bot if a command has been passed invalid arguments
	// and the bot does not handle it itself
	ErrInvalidArgument = errors.New("the command has been passed (an) invalid argument(s) by the user")

	// ErrUserNoPermissions should get returned by the bot if the user runs a command that they do not
	// have permission for to run and the bot does not handle the return message itself
	ErrUserNoPermissions = errors.New("the user does not have the needed permissions to run this command")

	// ErrNotAGuild should get returned by the bot if the user runs a command that needs to be ran
	// inside a guild to work, but they ran it in a DM and the bot does not handle the return message itself
	ErrNotAGuild = errors.New("this command can only be ran inside a guild, but it wasn't")

	// ErrNotADM should get returned by the bot if the user runs a command that needs to be ran
	// inside a DM to work, but they ran it in a Guild and the bot does not handle the return message itself
	ErrNotADM = errors.New("this command can only be ran inside DMs, but it wasn't")

	// ErrNotFound should get returned by the bot if the user requests (an operation on)
	// a resource that doesn't exist and the bot does not handle the return message itself
	ErrNotFound = errors.New("the requested object wasn't found")
)

// HandleError is the default handler of errors
func HandleError(ctx *Context, err error) {
	if err == nil {
		return
	}

	switch err.(type) {
	case discordgo.RESTError:
		if err.(*discordgo.RESTError).Response.StatusCode == 403 {
			err = NewErrBotHasNoPermissions(err.(*discordgo.RESTError))
		}
	}

	var errString string
	switch err {
	case ErrInvalidArgument:
		errString = "The arguments that you passed to the command are invalid."
		if ctx.Route.UsageString != "" {
			errString += fmt.Sprintf(" Please make sure you are following the user instructions: `%s`", ctx.Route.UsageString)
		}
	case ErrUserNoPermissions:
		errString = "You do not have permission to use this command."
	case ErrCouldNotFindRoute:
		errString = "This command does not exist"
	case ErrNotAGuild:
		errString = "This command cannot be ran in DMs"
	case ErrNotADM:
		errString = "This command cannot be ran in a Guild"
	case ErrNotFound:
		errString = "The resource or object the command needed does not exist"
	default:
		if info, ok := err.(ErrBotHasNoPermissions); ok {
			if info.Permission == "" {
				errString = "The bot does not have the required permissions for the command that was ran, please make sure it has before running it again."
			} else {
				errString = fmt.Sprintf("The bot could not complete the requested operation, because it does not have the following permission(s): %s", info.Permission)
			}
			break
		}

		if sentry.CurrentHub().Client() != nil {
			sentry.CaptureException(err)
			errString = "An unknown error has occurred and has been reported to my developers, sorry for any inconvenience this has caused"
		} else {
			panic(err)
		}
	}

	_, _ = ctx.SendMessage(
		"",
		discordgo.
			NewEmbed().
			SetTimestampNow().
			SetDescription(errString).
			SetColor(discordgo.ColorRed),
		nil,
	)
}

// ErrBotHasNoPermissions should get returned by the bot if the bot has been told to do an action
// that the bot does not have the needed permissions for and the bot does not handle the return message itself
type ErrBotHasNoPermissions struct {
	Permission   string
	EndpointPath string
}

// NewErrBotHasNoPermissions parses the RESTError and determines the permission(s) that the bot is missing
func NewErrBotHasNoPermissions(err *discordgo.RESTError) *ErrBotHasNoPermissions {
	r := &ErrBotHasNoPermissions{}

	r.EndpointPath = err.Request.URL.Path
	if !strings.HasPrefix(r.EndpointPath, discordgo.EndpointAPI) {
		return r
	}

	path := strings.TrimPrefix(r.EndpointPath, discordgo.EndpointAPI)
	pathParts := strings.Split(path, "/")
	APIEndpoint := pathParts[0]
	if len(pathParts) > 2 {
		APIEndpoint += "/" + pathParts[2]
	}
	if len(pathParts) > 4 {
		APIEndpoint += "/" + pathParts[4]
	}

	switch fmt.Sprintf("%s/%s", err.Request.Method, APIEndpoint) {
	case "PATCH/guilds":
		r.Permission = "Manage Server"
	case "POST/guilds/channels", "PATCH/guilds/channels":
		r.Permission = "Manage Channels"
	case "PATCH/guilds/members":
		r.Permission = "Manage Nicknames, Manage Roles, Mute Members, Deafen Members and/or Move Members"
	case "PUT/guilds/members/roles", "DELETE/guilds/members/roles":
		r.Permission = "Manage Roles"
	case "DELETE/guilds/members":
		r.Permission = "Kick Members"
	case "GET/guilds/bans", "PUT/guilds/bans", "DELETE/guilds/bans":
		r.Permission = "Ban Members"
	case "POST/guilds/roles", "PATCH/guilds/roles", "DELETE/guilds/roles":
		r.Permission = "Manage Roles"
	case "GET/guilds/prune", "POST/guilds/prune":
		r.Permission = "Kick Members"
	case "GET/guilds/invites", "GET/guilds/integrations", "POST/guilds/integrations",
		"PATCH/guilds/integrations", "DELETE/guilds/integrations", "GET/guilds/embed",
		"PATCH/guilds/embed", "GET/guilds/vanity-url":
		r.Permission = "Manage Server"
	case "POST/guilds/emojis", "PATCH/guilds/emojis", "DELETE/guilds/emojis":
		r.Permission = "Manage Emojis"

	case "PUT/channels", "PATCH/channels", "DELETE/channels":
		r.Permission = "Manage Channels"
	case "GET/channels/messages":
		r.Permission = "Read Messages and/or Read Message History"
	case "POST/channels/messages":
		r.Permission = "Send Messages"
	case "PUT/channels/messages/reactions":
		r.Permission = "Read Message History and/or Add Reactions"
	case "DELETE/channels/messages/reactions":
		r.Permission = "Manage Messages"
	case "DELETE/channels/messages":
		r.Permission = "Manage Messages"
	case "PUT/channels/permissions", "DELETE/channels/permissions":
		r.Permission = "Manage Roles"
	case "GET/channels/invites":
		r.Permission = "Manage Channels"
	case "POST/channels/invites":
		r.Permission = "Create Invite"
	case "PUT/channels/pins", "DELETE/channels/pins":
		r.Permission = "Manage Messages"

	case "DELETE/invites":
		r.Permission = "Manage Guild and/or Manage Channels"
	case "POST/channels/webhooks", "GET/channels/webhooks", "GET/guilds/webhooks", "PATCH/webhooks":
		r.Permission = "Manage Webhooks"
	case "GET/guilds/audit-logs":
		r.Permission = "View Audit Log"
	}

	return r
}

func (r ErrBotHasNoPermissions) Error() string {
	return fmt.Sprintf("Error due to the bot missing the %s permission", r.Permission)
}
