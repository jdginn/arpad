# Arpad, a bridge for DAW control surfaces

## What is Arpad?
Arpad is a bridge between one or more control surfaces and one or more devices under control. The main use-case is linking MIDI controllers to DAWs, audio interfaces, and other equipment in a recording studio.

Arpad aggregates control surfaces communicating over some combination of protocols and maps those surfaces to a combination of targets over another combination of protocols. Arpad presents bidirectional interfaces
between all devices; meaning each MIDI controller or DAW can control any other controller or DAW. Mappings within Arpad are implemented in code and can thus be almost endlessly flexible.

## Why the name Arpad?
Árpád Híd is the northernmost bridge over the Danube River in Budapest, Hungary, home of Selah recording studio and the team that wrote this software.

## Why do we need Arpad?
Most DAW control is implemented over MCU, which was a creative approach to controlling a DAW using a single 16-channel MIDI device. In 2025, however, MCU is outdated, underpowered, and not flexible enough.

Some DAWs (especially Reaper) provide great support for protocols other than MCU. Unfortunately, there are relatively few controllers that can speak alternate protocols. Most controllers can support raw MIDI, but
some intelligence is still helpful on the controller side to provide a good interface to the DAW.

The situation is more complex if we wish to control multiple DAWs or other targets from the same controller. For example, many audio interfaces implement an internal mixer which can be controlled via MIDI, OSC, or some
other protocol. We wish to use the same fader controller to either control the mix within the DAW or monitor mixes within the audio interface's mixer. This requires routing messages from the fader controller to either
the DAW or interface depending on the selected mode. Moreover, we would like to keep some of the features of the controller routed to the DAW even in the audio interface mode (e.g. transport buttons) and we may further
wish to a button assigned to switching between speaker outputs on the interface even in DAW mode. All of these features require mapping and routing outside the DAW.

In 2025 we have more convenient protocols than raw MIDI. While MIDI can send the data, there are inconveniences like a single traditional MIDI device being limited to 14b control signals. Also, the MIDI protocol
is not self-documenting, which makes it hard to properly assign the right signals between the interface and DAW. The OSC protocol is more flexible and self-documenting. Arpad makes it (relatively) easy to use
a MIDI device (such as a Behringer X32) as an OSC device (or some hypothetical other protocol.)

## Why is this implemented in go?
Go strikes a good balance between speed and simplicity. It is simple enough to define the mapping entirely in pure Go code, which is much more flexible than a configuration file approach. This could be frustratingly
difficult in a language like C++ or Rust. Go is performant and easily distributed as a binary, unlike Python.

## Project status
This is still a work in progress and not yet ready to be used. No functionality tested yet. Everything is subject to change.

## Supported protocols
[ ] MIDI (partial)
[ ] OSC (partial)
[ ] MOTU datastore access over HTTP (partial)

## Supported devices
[ ] Behringer X32
[ ] MOTU AVB series audio interfaces (828es, 1248, etc.)

