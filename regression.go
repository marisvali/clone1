package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
)

// StateBytes is an array of bytes that represent the current state of the
// World, as perceived by the outside. If two Worlds have the same State() they
// are considered "the same", even though they may be implemented differently.
func (w *World) StateBytes() []byte {
	// The world is "the same" if it has the same:
	// - the same bricks at the same positions
	//
	// Explanation:
	//
	// Here we need a definition of what it means that the world is "the same"
	// after its implementation changed.
	//
	// Option 1 - check all bits
	// -------------------------
	//
	// The most straightforward test is to check if it contains exactly the same
	// bits at the end. But this would mean that any change in the data
	// structures of the world would have to be a breaking change, which is not
	// exactly what I'm looking for. If I can get the same behavior relevant
	// for the outside as before, I should be free to change the world.
	//
	// Option 2 - check what the GUI shows
	// -----------------------------------
	//
	// Following the previous reasoning, I could say that whatever is shown to
	// the player is the actual state of the world. So I should first of all
	// have a more well-defined interface between the world and the GUI so that
	// I know exactly what the GUI gets from the world. This way, I can use that
	// as the bits I check to be the same.
	// The disadvantage here is that I'm still changing things a lot and I
	// don't want the friction of an interface that I adjust every time I change
	// something. I want everything in the World public because I want to
	// inspect it either from the GUI, or a future AI or some analysis script.
	// Another disadvantage is that I might want to include things that are not
	// shown in the GUI.
	//
	// Option 3 - freestyle but kind of follow the GUI (selected)
	// -----------------------------------------------
	//
	// Following the previous reasoning, I could say that I can follow what is
	// shown in the GUI as a sanity check that I'm including everything that
	// sounds reasonable for a check like this. But, I provide my own definition
	// here for what it means that two worlds at the same.
	// WARNING: this method makes some assumptions.
	// a. I assume that a reasonable playthrough is provided where the player
	// goes through an entire level, mostly winning.
	// b. I assume that the player's moves are highly relevant, as in, if you
	// take out any of the moves or make a significant deviation, the simulation
	// goes in a very different direction quickly. You don't get this if the
	// player doesn't do anything, for example, or makes moves which have no
	// impact on the game (hard to do, but possible).
	// c. I assume that the playthrough contains enough World elements that the
	// regression test makes sense. If the playthrough contains no enemies, the
	// regression test will not catch any changes in enemy behavior, for
	// example.
	//
	// In the end, most feasible regression tests are imperfect. I trust that
	// the assumptions I make here are reasonable and this test provides a good
	// enough check that I didn't break anything, at least good enough for my
	// current needs.

	buf := new(bytes.Buffer)
	Serialize(buf, int64(len(w.Bricks)))
	for _, b := range w.Bricks {
		Serialize(buf, b.PixelPos)
		Serialize(buf, b.Val)
		Serialize(buf, b.State)
		Serialize(buf, b.FallingSpeed)
	}
	return buf.Bytes()
}

// RegressionId returns a string which uniquely identifies the playthrough.
// It is a hash of all the states of the World. It is meant to check if the
// state of the World at each frame in the playthrough is the same after a
// refactorization of the World.
//
// The World has its own definition of what it means that two World states are
// the same, even though they are implemented differently.
//
// RegressionId is meant to be used this way:
// - Compute the RegressionId for a playthrough.
// - Refactor the implementation of the World.
// - Compute the RegressionId for the same playthrough. It uses the exact same
// level and player inputs, but the new World implementation.
// - If the RegressionId hasn't changed, the playthroughs are (pretty much*)
// identical. The refactoring of World did not alter the playthrough.
// - If the RegressionId has changed, something in the refactoring is now
// causing the play experience to be different.
//
// * "pretty much" is meant to mean so similar that it would be nearly
// impossible to get to the same RegressionId with the same inputs, but have the
// gameplay be any different. This is an assumption and it relies on how quickly
// a simulation goes insane if any part of it doesn't act right. But it is very
// dependent on how long the playthrough is and how many world elements it
// involves. If it's just the player starting with some obstacles and no enemies
// and winning after 1 frame, that won't catch errors with refactoring enemy
// behavior.
func RegressionId(p *Playthrough) string {
	// Create a new SHA-256 hash
	hash := sha256.New()

	// Run the playthrough.
	w := NewWorld()

	// Write the current state of the World to the hash.
	hash.Write(w.StateBytes())

	for i := range p.History {
		w.Step(p.History[i])

		// Write the current state of the World to the hash.
		hash.Write(w.StateBytes())
	}

	// Get the resulting hash as a byte slice
	hashBytes := hash.Sum(nil)

	// Convert the byte slice to a hex string
	hashHex := hex.EncodeToString(hashBytes)
	return hashHex
}
