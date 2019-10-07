package router

import (
	"errors"
	"fmt"
	"github.com/auttaja/discordgo"
	"github.com/getsentry/sentry-go"
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

	// ErrBotNoPermissions should get returned by the bot if the bot has been told to do an action
	// that the bot does not have the needed permissions for and the bot does not handle the return message itself
	ErrBotNoPermissions = errors.New("the bot has been ordered to do something it does not have the permissions to do")

	// ErrUserNoPermissions should get returned by the bot if the user runs a command that they do not
	// have permission for to run and the bot does not handle the return message itself
	ErrUserNoPermissions = errors.New("the user does not have the needed permissions to run this command")
)

// HandleError
func HandleError(ctx *Context, err error) {
	if err == nil {
		return
	}

	switch err.(type) {
	case discordgo.RESTError:
		if err.(discordgo.RESTError).Response.StatusCode == 403 {
			err = ErrBotNoPermissions
		}
	}

	var errString string
	switch err {
	case ErrInvalidArgument:
		errString = "The arguments that you passed to the command are invalid."
		if ctx.Route.UsageString != "" {
			errString += fmt.Sprintf(" Please make sure you are followin the user instructions: `%s`", ctx.Route.UsageString)
		}
	case ErrBotNoPermissions:
		errString = "The bot does not have the required permissions for the command that was ran, please make sure it has before running it again."
	case ErrUserNoPermissions:
		errString = "You do not have permission to use this command."
	case ErrCouldNotFindRoute:
		errString = "This command does not exist"
	default:
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
