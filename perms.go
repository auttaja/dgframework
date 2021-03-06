package dgframework

import (
	"github.com/auttaja/dgframework/router"
	"github.com/casbin/casbin"
)

// CasbinMiddleware provides the ability to use Casbin to validate command perms
type CasbinMiddleware struct {
	*casbin.Enforcer
}

// Casbin performs the casbin check on the command
func (m *CasbinMiddleware) Casbin(fn router.HandlerFunc) router.HandlerFunc {
	return func(ctx *router.Context) error {
		guild, err := ctx.Guild()
		if err != nil {
			return nil
		}
		if res := m.Enforce(ctx.Msg.Author.ID, ctx.Msg.GuildID, ctx.Route.Name, "execute"); res || guild.OwnerID == ctx.Msg.Author.ID {
			return fn(ctx)
		}
		return router.ErrUserNoPermissions
	}
}
