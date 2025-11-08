package main

import (
	"bytes"
	"fmt"
	"github.com/google/uuid"
	"slices"
)

// InputVersion is the version of the byte representation of the Playthrough
// structure. If the Playthrough structure changes such that serializing it
// produces a different array of bytes, then InputVersion must change as well.
// InputVersion is meant to track changes to saved playthroughs. I want
// SimulationVersion to indicate an abstract simulation. However, when we
// record playthroughs, we can't be abstract anymore, we need actual bytes.
// An executable can replay any playthrough with the same InputVersion
// and SimulationVersion as the ones in the executable. If only the InputVersion
// is different, then the playthrough can be translated by loading the old
// Playthrough structure and translating it to the new one.
// Out of the 3 versions (ReleaseVersion, SimulationVersion and InputVersion),
// the InputVersion is the one expected to change the least often.
const InputVersion = 1

// Playthrough represents all the input sent to a World during the execution
// of a level. Given this input and a compatible simulation, the same output
// should be generated in the end.
type Playthrough struct {
	InputVersion      int64
	SimulationVersion int64
	ReleaseVersion    int64
	Level
	Id      uuid.UUID
	Seed    int64
	History []PlayerInput
}

func (p *Playthrough) Serialize() []byte {
	buf := new(bytes.Buffer)
	Serialize(buf, p.InputVersion)
	Serialize(buf, p.SimulationVersion)
	Serialize(buf, p.ReleaseVersion)
	SerializeSlice(buf, p.Level.BricksParams)
	Serialize(buf, p.Id)
	Serialize(buf, p.Seed)
	SerializeSlice(buf, p.History)
	return Zip(buf.Bytes())
}

func (p *Playthrough) Clone() *Playthrough {
	clone := *p
	clone.History = slices.Clone(p.History)
	return &clone
}

func DeserializePlaythrough(data []byte) (p Playthrough) {
	buf := bytes.NewBuffer(Unzip(data))
	Deserialize(buf, &p.InputVersion)
	if p.InputVersion != InputVersion {
		Check(fmt.Errorf("can't deserialize this playthrough - we are at "+
			"InputVersion %d and playthrough was generated with InputVersion "+
			"version %d",
			InputVersion, p.InputVersion))
	}
	Deserialize(buf, &p.SimulationVersion)
	Deserialize(buf, &p.ReleaseVersion)
	DeserializeSlice(buf, &p.BricksParams)
	Deserialize(buf, &p.Id)
	Deserialize(buf, &p.Seed)
	DeserializeSlice(buf, &p.History)
	return
}
