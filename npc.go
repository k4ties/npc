package npc

import (
	"time"

	"github.com/df-mc/dragonfly/server/block/cube"
	"github.com/df-mc/dragonfly/server/player"
	"github.com/df-mc/dragonfly/server/world"

	"github.com/go-gl/mathgl/mgl64"
)

type playerConfig struct {
	player.Config
}

func (playerConfig) BBox(e world.Entity) cube.BBox {
	p := e.(*player.Player)
	switch {
	case p.Gliding(), p.Swimming(), p.Crawling():
		return cube.Box(-0.3, 0, -0.3, 0.3, 0.6, 0.3)
	case p.Sneaking():
		return cube.Box(-0.3, 0, -0.3, 0.3, 1.5, 0.3)
	default:
		return cube.Box(-0.3, 0, -0.3, 0.3, 1.8, 0.3)
	}
}

func (playerConfig) DecodeNBT(m map[string]any, data *world.EntityData) {}
func (playerConfig) EncodeNBT(data *world.EntityData) map[string]any    { return nil }
func (playerConfig) EncodeEntity() string                               { return "minecraft:player" }
func (playerConfig) NetworkOffset() float64                             { return 1.621 }
func (playerConfig) Open(tx *world.Tx, handle *world.EntityHandle, data *world.EntityData) world.Entity {
	return player.Type.Open(tx, handle, data)
}

// HandlerFunc may be passed to Create to handle a *player.Player attacking an NPC.
type HandlerFunc func(npc, target *player.Player)

// Create creates a new NPC with the Settings passed. A world.Loader is spawned in the background which follows the
// NPC to prevent it from despawning. Create panics if the world passed is nil.
// The HandlerFunc passed handles a player interacting with the NPC. Nil may be passed to avoid calling any function
// when the entity is interacted with.
// Create returns the *player.Player created. This entity has been added to the world passed. It may be removed from
// the world like any other entity by calling (*player.Player).Close.
func Create(s Settings, w *world.World, f HandlerFunc) *player.Player {
	if w == nil {
		panic("world passed to npc.create must not be nil")
	}
	if f == nil {
		f = func(*player.Player, *player.Player) {}
	}
	if s.Scale <= 0 {
		s.Scale = 1.0
	}

	npc := world.EntitySpawnOpts{Position: s.Position}.New(playerConfig{},
		player.Config{
			Rotation: cube.Rotation{s.Yaw, s.Pitch},
			Skin:     s.Skin,
			Name:     s.Name,
			Position: cube.PosFromVec3(s.Position).Vec3Centre(),
		},
	)

	w.Exec(func(tx *world.Tx) {
		tx.AddEntity(npc)
	})

	l := world.NewLoader(8, w, world.NopViewer{})
	h := &handler{f: f, l: l, vulnerable: s.Vulnerable, pos: s.Position}

	npc.ExecWorld(func(tx *world.Tx, e world.Entity) {
		pl := e.(*player.Player)
		pl.Move(mgl64.Vec3{}, s.Yaw, s.Pitch)
		pl.SetScale(s.Scale)
		pl.SetHeldItems(s.MainHand, s.OffHand)
		pl.Armour().Set(s.Helmet, s.Chestplate, s.Leggings, s.Boots)

		if s.Immobile {
			pl.SetImmobile()
		}

		pl.Handle(h)
		pl.Chat("test")
	})

	var p *player.Player

	w.Exec(func(tx *world.Tx) {
		h.syncPosition(tx, s.Position)
		go syncWorld(npc, l)

		e, ok := npc.Entity(tx)
		if !ok {
			panic("npc is not in a world")
		}

		p = e.(*player.Player)
	})

	return p
}

// syncWorld periodically synchronises the world of the world.Loader passed with a player.Player's world. It stops doing
// so once the world returned by (*player.Player).World is nil.
func syncWorld(npc *world.EntityHandle, l *world.Loader) {
	t := time.NewTicker(time.Second / 20)
	defer t.Stop()

	for range t.C {
		npc.ExecWorld(func(tx *world.Tx, e world.Entity) {
			if w := tx.World(); w != l.World() {
				if w == nil {
					// The NPC was closed in the meantime, stop synchronising the world.
					return
				}
				l.ChangeWorld(tx, w)
			}
		})
	}
}
