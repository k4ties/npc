package npc

import (
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/entity"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"
	"github.com/go-gl/mathgl/mgl64"
)

// handler implements the handler for an NPC entity. It manages the execution of the HandlerFunc assigned to the NPC
// and makes sure the *world.Loader's position remains synchronised with that of the NPC.
type handler struct {
	player.NopHandler

	l *world.Loader
	f HandlerFunc

	pos mgl64.Vec3

	vulnerable bool
}

// HandleHurt ...
func (h *handler) HandleHurt(ctx *player.Context, dmg *float64, _ bool, im *time.Duration, src world.DamageSource) {
	if src, ok := src.(entity.AttackDamageSource); ok {
		if attacker, ok := src.Attacker.(*player.Player); ok {
			*dmg = 0
			h.f(ctx.Val(), attacker)
		}
	}

	*im = 410 * time.Millisecond

	if !h.vulnerable {
		ctx.Cancel()
	}
}

// HandleMove ...
func (h *handler) HandleMove(ctx *player.Context, pos mgl64.Vec3, _ cube.Rotation) {
	h.syncPosition(ctx.Val().Tx(), pos)
}

// HandleTeleport ...
func (h *handler) HandleTeleport(ctx *player.Context, pos mgl64.Vec3) {
	h.syncPosition(ctx.Val().Tx(), pos)
}

// syncPosition synchronises the position passed with the one in the world.Loader held by the handler. It ensures the
// chunk at this new position is loaded.
func (h *handler) syncPosition(tx *world.Tx, pos mgl64.Vec3) {
	h.l.Move(tx, pos)
	h.l.Load(tx, 8)
}

// HandleQuit ...
func (h *handler) HandleQuit(p *player.Player) {
	h.l.Close(p.Tx())
}

// HandleQuit ...
func (h *handler) HandleDeath(p *player.Player, src world.DamageSource, keepInv *bool) {
	*keepInv = true
	p.Respawn()
}
