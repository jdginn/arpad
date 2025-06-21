# REAPER OSC Go Source Generator: Design Document

## Overview

This document describes a code generator that produces a REAPER OSC API implementation compliant with the Device API Design specification. The generator processes REAPER's OSC pattern config files to create a type-safe, hierarchical API for interacting with REAPER through OSC. The generator is intended to be run as part of a Go `go generate` step.

---

## Goals

- Parse the REAPER OSC config file and extract all actions, patterns, argument types, wildcards, and documentation.
- Generate idiomatic Go methods on a `Reaper` struct for binding handlers to each action.
- Prefer numeric bindings (int/float/bool) over string bindings when multiple pattern types exist for an action.
- Use appropriate Go types in Bind methods, based on the OSC pattern's argument type.
- Copy and preserve relevant documentation/comments from the config file to generated Go code.
- Emit a single Go source file containing all bindings.
- Store all patterns and paths for each action, even if only the "best" is used in the generated API, for future extensibility.

---

## OSC Pattern to Device API Mapping

### Endpoint Path Translation

OSC patterns are mapped to the Device API hierarchy following these rules:

1. Path segments become levels in the type hierarchy
2. Wildcards (`@`) become qualifier parameters
3. Value types are determined by OSC type prefixes:
   - `n/` -> `float64` (normalized 0-1)
   - `f/` -> `float64` (raw float)
   - `i/` -> `int64`
   - `s/` -> `string`
   - `t/` -> `bool`

Example mapping:

```go
n/track/@/fx/@/fxparam/@/value

Becomes:

type Reaper struct {
    Track func(trackNum int64) *track
}

type track struct {
    Fx func(fxNum int64) *trackFx
}

type trackFx struct {
    Fxparam func(paramNum int64) *trackFxFxparam
}

type trackFxParam struct {
    state trackFxParamState
    Value *trackFxFxparamValueEndpoint  // implements both bindable and setable
}
```

### State Management

State structs must be generated for each qualified level:

```go
type trackState struct {
    trackNum int64
}

type trackFxState struct {
    trackNum int64  // inherited
    fxNum int64
}

type trackFxParamState struct {
    trackNum int64  // inherited
    fxNum int64    // inherited
    paramNum int64
}
```

### Interface Implementation

Each endpoint must implement both `bindable` and `setable`:

```go
type xxxEndpoint struct {
    state xxxState
    device *Reaper
}

func (v *ValueEndpoint) Bind(callback func(float64) error) {
    addr := fmt.Sprintf("/track/%d/fx/%d/fxparam/%d/value",
        v.state.trackNum, v.state.fxNum, v.state.paramNum)
    v.device.bindOSCHandler(addr, callback)
}

func (v *ValueEndpoint) Set(value float64) error {
    if err := v.validate(value); err != nil {
        return err
    }
    addr := fmt.Sprintf("/track/%d/fx/%d/fxparam/%d/value",
        v.state.trackNum, v.state.fxNum, v.state.paramNum)
    return v.device.sendOSCMessage(addr, value)
}
```

## Generator Implementation

### High-level Algorithm

1. Process input file to capture all actions and their respective patterns.
2. Generate source code implementing hierarchical structs that lead to an endpoint exposing Bind()/Set() for each path (except for the paths we ignore, see below). Note that actions with multiple paths will result in multiple endpoints.
3. Emit source code as a valid go file.

### Input Processing

1. Parse config file format:

```
<ACTION_NAME> <type_1>/<osc/path/1> <type_2>/<osc/path/2> ...
```

- Read all lines, capturing:
  - Block and line comments.
  - Action names and all their patterns (type prefix + OSC path).
- For each action:
  - Store all patterns for extensibility.
  - Associate documentation with the action.
- If the action has no valid patterns, skip it.

2. Group patterns by action name and filter using these rules:

   - For each action, group all patterns (across all config lines) by action name.
   - Some action names or path elements may contain `+` or `-`, which should be replaced with `Plus` and `Minus`, respectively

   **NOTE:** when comparing paths within the same action, we must always compare element-by-element from left to right, ignoring wildcards. It is not sufficient simply to compare path lengths.

   - If there are two patterns that have identical elements other than wildcards, keep only the pattern with the most wildcards.
   - Note that this requires comparing paths element by element, where wildcards are simply ignored in the comparison.
   - Sometimes there are multiple patterns, where one or more paths add additional elements to the right of that point (e.g., `/track/@/volume` and `/track/@/volume/str`). In this case, all patterns should be implemented with the following rules:
     - Each path should implement its own Bind()/Set() methods.
     - It is allowed for the same hierarchy level in the generated code to contain BOTH the Bind()/Set() methods AND a substruct expressing the lower level(s) of hierarchy in the paths.
     - Make sure to respect the type of each of these paths! In many situations, a lower-level path will have a different type than the one above it (especially in the very common case where an action implements a both a numeric and a string path.)
     - For example, the following action

```
TRACK_VOLUME n/track/volume n/track/@/volume
TRACK_VOLUME s/track/volume/str s/track/@/volume/str
TRACK_VOLUME f/track/volume/db f/track/@/volume/db
```

should result in the following hierarchy of Bind()/Set() methods:

```
Reaper.Track(1).Volume.Bind(func(float64) error)
Reaper.Track(1).Volume.Set(float64)
Reaper.Track(1).Volume.Db.Bind(func(float64) error)
Reaper.Track(1).Volume.Db.Set(float64)
Reaper.Track(1).Volume.Str.Bind(func(string) error)
Reaper.Track(1).Volume.Str.Set(string)
```

Note that the Volume endpoint contains BOTH Bind()/Set() AND the sub endpoints Db and Str. Note also that the Str endpoint has a different type than its parent.
This hierarchical relationship is allowed to be extended to any number of levels, as specified by the reaper config file.

### Code Generation

For each unique OSC pattern:

1. Generate hierarchy types. Each type in the hierarchy should have its name built from the full hierarchy to avoid name collisions (consider the situation where there are multiple actions that contain an endpoint ending in "volume").

All of these struct type definitions should be private (that is, lowercase) to the generated package (with the exception of the top-level Reaper struct: see below).
Within each level of the hierarchy, the handle to the next level down, as well as Bind/Set must be public.

The top-level struct should always be named `Reaper`. This struct will embed `device.OscDevice`. Since `Reaper` is _always_ the top-level struct, it is not necessary to prefix all type names with `Reaper`. **This should be the only public struct type.**

```go
// Given n/track/@/volume
type Reaper struct {
    *dev.OscDevice
    Track func(int64) *track
}

// NOTE: Track does not need to be prefixed with "Reaper"
type track struct {
    // NOTE: trackVolumeEndpoint contains the full hierarchy from Reaper downward in its name to avoid name collisions
    Volume *trackVolumeEndpoint
}
```

2. Generate state structs. State structs contain the qualified values of placeholders in the path (`@`).

   ```go
   type trackState struct {
       trackNum int64
   }
   ```

3. Generate endpoint implementations:

   ```go
   // NOTE: trackVolumeEndpoint contains the full hierarchy from Reaper downward in its name to avoid name collisions
   type trackVolumeEndpoint struct {
       device *Reaper
       state trackState
   }

   func (v *trackVolumeEndpoint) Bind(callback func(float64) error) {
       addr := fmt.Sprintf("/track/%d/volume", v.state.trackNum)
       v.device.bindOSCHandler(addr, callback)
   }

   func (v *trackVolumeEndpoint) Set(value float64) error {
       if err := v.validate(value); err != nil {
           return ErrOutOfRange
       }
       addr := fmt.Sprintf("/track/%d/volume", v.state.trackNum)
       return v.device.sendOSCMessage(addr, value)
   }
   ```

### Value Validation

1. Implement range validation based on OSC type:

```go
func (v *trackVolumeEndpoint) validate(value float64) error {
    // assume VolumeEndpoint has type n/
    // n/ types must be 0.0-1.0
    if (value < 0 || value > 1) {
        return ErrOutOfRange
    }
    return nil
}
```

2. Handle special cases:

   - Pan values: -1.0 to 1.0
   - Toggle values: 0 or 1
   - Enum values: documented ranges

### Input Processing Examples

To illustrate the pattern grouping and hierarchy generation rules, consider these examples:

#### Example 1: Base Path with Extension

Given these patterns in the config:

```
TRACK_VOLUME n/track/@/volume
TRACK_VOLUME f/track/@/volume/db
```

This should generate the following hierarchy:

```go
type Reaper struct {
    Track func(trackNum int64) *track
}

type track struct {
    Volume *trackVolumeEndpoint
}

// NOTE: each endpoint struct's name should be built from the full hierarchy to avoid name collisions
// (consider the situation where there are multiple actions that contain an endpoint ending in "volume")
type trackVolumeEndpoint struct {
    device *Reaper
    state trackState                // Note that state is always stored by value, never by reference
    Db    *trackVolumeDbEndpoint    // Additional endpoint for /db extension
}

func (v *trackVolumeEndpoint) Bind(callback func(float64) error) {
    addr := fmt.Sprintf("/track/%d/volume", v.state.trackNum)
    v.device.bindOSCHandler(addr, callback)
}

func (v *trackVolumeEndpoint) Set(value float64) error {
    if err := v.validate(value); err != nil {
        return err
    }
    addr := fmt.Sprintf("/track/%d/volume", v.state.trackNum)
    return v.device.sendOSCMessage(addr, value)
}

// Extended db endpoint
func (d *trackVolumeDbEndpoint) Bind(callback func(float64) error) {
    addr := fmt.Sprintf("/track/%d/volume/db", d.state.trackNum)
    d.device.bindOSCHandler(addr, callback)
}

func (d *trackVolumeDbEndpoint) Set(value float64) error {
    if err := d.validate(value); err != nil {
        return err
    }
    addr := fmt.Sprintf("/track/%d/volume/db", d.state.trackNum)
    return d.device.sendOSCMessage(addr, value)
}
```

This allows for both:

```go
reaper.Track(1).Volume.Bind(callback)      // Binds to n/track/@/volume
reaper.Track(1).Volume.Db.Bind(callback)   // Binds to f/track/@/volume/db
```

#### Example 2: Multiple Extensions

Given:

```
TRACK_FX n/track/@/fx/@/enabled
TRACK_FX f/track/@/fx/@/wetdry
TRACK_FX s/track/@/fx/@/name/str
```

This should generate (simplified, with implementation omitted):

```go
type Reaper struct {
    Track func(trackNum int64) *track
}

type track struct {
    Fx func(fxNum int64) *trackFxEndpoint
}

type trackFxEndpoint struct {
    state trackFxState
    Enabled *trackFxEnabledEndpoint
    Wetdry  *trackFxWetdryEndpoint
    Name    *trackFxNameEndpoint
    // Note also that the endpoint name is Name, not Str or NameStr
}

type trackFxNameEndpoint struct {
    state trackFxNameState
    Str *trackFxNameStrEndpoint
}
```

This allows for:

```go
reaper.Track(1).Fx(2).Enabled.Bind(callback)    // Binds to n/track/@/fx/@/enabled
reaper.Track(1).Fx(2).Wetdry.Bind(callback)     // Binds to f/track/@/fx/@/wetdry
reaper.Track(1).Fx(2).Name.Str.Bind(callback)   // Binds to s/track/@/fx/@/name/str
```

## Generator Usage

```go
//go:generate go run ./cmd/reaperoscgen -config path/to/config
```

Generated files:

- `reaper_device_gen.go`: Device API implementation: all generated types, endpoints, and Bind/Set methods.

## Implementation Requirements

1. State immutability:

   - All state structs must be value types
   - Parent state must not be modified by child operations
   - Endpoint instances must be safely capturable as variables

2. Error handling:

   - Must use `ErrOutOfRange` for validation failures
   - Must propagate OSC communication errors
   - Must validate all inputs before sending

3. Documentation:
   - Must preserve OSC config comments as Go doc comments
   - Must document value ranges and units
   - Must document any REAPER-specific behaviors
