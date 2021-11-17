package artnet

// ChannelValue defines an ArtNet Universe and the value of the DMX channel.
type ChannelValue struct {
	Universe uint16 // Universe: старший байт - SubUni, младший байт - Net.
	Channel  uint16 // Channel: номер байта (канал).
	Value    uint8  // Value: значение для канала.
}

// Universe wraps the 512 byte array for convenience.
type Universe [512]byte

func (u Universe) toByteSlice() [512]byte {
	return u
}

// UniverseStateMap holds the state of all used universes.
type UniverseStateMap map[uint16]Universe

// DMXCommand tells a DMX ArtNet to set a channel on a universe to a specific value.
type DMXCommand struct {
	Universe uint16 // Universe is the universe a DMXDevice is in.
	Channel  uint16 // Channel is the channel a command can talk to (0-511).
	Value    uint8  // Value is the value a DMX channel can represent (0-255).
}

// DMXCommands is an array of DMXCommands.
type DMXCommands []DMXCommand

type NodeTopic struct {
	Name      string
	OutputStr []string
	Output    []uint16
}

type IpsType struct {
	Ips    []string
	Topics []NodeTopic
}


