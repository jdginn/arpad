# Device API Design

This document describes the design of APIs for interacting with audio devices through a consistent, type-safe interface. The design supports both monitoring device state changes and setting device parameters through a hierarchical API that mirrors the structure of device endpoints.

## Core Interfaces

### Binding to State Changes

The `bindable` interface allows monitoring state changes on a device endpoint:

```go
// bindable represents an endpoint that can have a callback bound to it to monitor state changes.
// The callback will be invoked whenever the endpoint's value changes.
type bindable[A any] interface {
    Bind(func(A) error)
}
```

For example:

```go
// Bind to volume changes on track 1
device.Tracks(1).Volume.Bind(func(value float64) error {
    fmt.Printf("Track 1 volume changed to: %f\n", value)
    return nil
})
```

### Setting Values

The `setable` interface allows modifying device state:

```go
// setable represents an endpoint that can have its value set.
type setable[V any] interface {
    Set(V) error
}
```

For example:

```go
// Set volume on track 1 to 0.8
err := device.Tracks(1).Volume.Set(0.8)
```

## Hierarchical API Design

Device endpoints are represented through a hierarchy of structs that mirror the logical organization of the device's capabilities. Each level in the hierarchy may optionally take parameters to qualify exactly which instance of that level is being addressed.

### Qualifier Parameters

When a level in the hierarchy requires qualification (e.g., which specific track), it is represented by a function that takes the qualifying parameter and returns the next level:

```go
type Device struct {
    // Tracks requires a track number qualifier
    Tracks func(trackNum int64) *Track

    // Global requires no qualifier
    Global *Global
}

type Track struct {
    // Volume needs no qualifier
    Volume *VolumeEndpoint

    // Fx requires an fx index qualifier
    Fx func(fxNum int64) *Fx
}
```

### State Management

For hierarchies requiring multiple qualifiers, each level maintains both its own qualifier and those of its parents:

```go
// Internal state management (not exposed to API users)
type trackState struct {
    trackNum int64
}

type fxState struct {
    trackNum int64  // inherited from parent
    fxNum int64    // this level's qualifier
}

type fxParamState struct {
    trackNum int64  // inherited
    fxNum int64    // inherited
    paramNum int64  // this level's qualifier
}
```

### Example Hierarchies

Simple endpoint with no qualifiers:

```go
device.Global.Playback.Pause.Set(true)
```

Endpoint with single qualifier:

```go
device.Tracks(1).Volume.Set(0.8)
```

Complex endpoint with multiple qualifiers:

```go
device.Tracks(1).Fx(2).Param(3).Set(0.5)
```

## Code Generation

To implement this design for a specific device, code generators must:

1. Parse the device's endpoint specification format
2. For each endpoint:
   - Identify the levels in the hierarchy
   - Determine which levels require qualifiers
   - Generate appropriate state structs for qualified levels
   - Generate the hierarchy of types
   - Implement `bindable` and/or `setable` on leaf nodes

### Generated Structure Example

For an endpoint like `/track/@/fx/@/param/@/value`:

```go
// Generated types
type Device struct {
    Tracks func(int64) *Track
}

type Track struct {
    Fx func(int64) *Fx
}

type Fx struct {
    Param func(int64) *FxParam
}

type FxParam struct {
    state fxParamState
}

// Implement interfaces on leaf node
func (f *FxParam) Set(value float64) error {
    // Use accumulated state to set value
    return nil
}

func (f *FxParam) Bind(callback func(float64) error) {
    // Use accumulated state to bind callback
}
```

## API Design Guidelines

### Qualifier Parameter Naming

Qualifier parameters should have meaningful names that reflect their purpose in the device's functionality:

```go
type Device struct {
    // Good: Clear what the parameter represents
    Tracks func(trackId int64) *Track
    // Bad: Generic parameter name
    Tracks func(param1 int64) *Track
}
```

When available, use documentation from the device specification to inform parameter naming. For example, if the spec refers to "bank index" or "channel number", use these terms in the parameter names.

### Multiple Qualifiers at Same Level

When a specification requires multiple qualifiers at the same conceptual level, these should be split into separate layers in the API hierarchy. This maintains the principle that each layer takes at most one qualifier.

Example - MIDI CC which requires both channel and controller number:

```go
// Good: Split into two layers, each with a single qualifier
type MidiDevice struct {
    CC *CC
}

type CC struct {
    Channel func(channelNum uint8) *CCChannel
}

type CCChannel struct {
    Controller func(controllerNum uint8) *CCEndpoint
}

// Usage:
device.CC.Channel(1).Controller(66).Bind(callback)

// Bad: Multiple qualifiers in one function
type MidiDevice struct {
    CC func(channel, controller uint8) *CCEndpoint  // Don't do this
}
```

The code generator may choose which qualifier takes precedence in the hierarchy, as this is an implementation detail that doesn't affect the API's functionality.

### Qualifier Type Constraints

- Each layer in the hierarchy must take at most one qualifier
- Qualifiers must be primitive types or simple custom types/enums
- Qualifiers must not be structs containing multiple values
- If multiple values need to be specified, split them across multiple layers in the hierarchy

```go
// Good: Simple enum type as qualifier
type BankType int
const (
    InputBank BankType = iota
    OutputBank
)
device.Bank(InputBank).Channel(1).Level.Set(0.8)

// Bad: Struct as qualifier
type BankParams struct {  // Don't do this
    Type BankType
    Index int
}
device.Bank(BankParams{...}).Level.Set(0.8)  // Don't do this
```

These constraints ensure:

1. API consistency across different devices
2. Clear, linear progression through the hierarchy
3. Simple, predictable state management
4. Easy discovery of API capabilities through IDE tooling

## Implementation Notes

1. Code generators must handle:

   - Different endpoint specification formats
   - Varying levels of qualification needed
   - State accumulation through the hierarchy
   - Type-safe parameter passing
   - Proper implementation of `bindable`/`setable` interfaces

2. The hierarchical API structure should be maintained consistently across different devices, even if their underlying specifications differ significantly.

3. Generators should ensure that:
   - All necessary state is accumulated properly
   - Type safety is maintained throughout the hierarchy
   - Interface implementations properly use accumulated state
   - Documentation is generated for each level

## Implementation Requirements

### Value Types and Validation

- Value types can be:
  - Primitive Go types
  - Custom types based on primitives (`~string`, `~int`, `~float64`)
  - Enums
  - Custom struct types that resolve to a single value
- For custom struct types, the generator must implement any necessary marshaling to convert to the endpoint's expected value
- Range validation should be implemented on a best-effort basis where spec provides sufficient information
- Range validation occurs within `Set`/`Bind` calls where full state is available
- Intermediate hierarchy calls (e.g., `Foo(15)`, `Bar(16)`) must return the next level regardless of parameter validity
- An `ErrOutOfRange` error type is recommended but not required

```go
// Example of value type and validation
type PanValue float64

func (p PanValue) validate() error {
    if p < -1 || p > 1 {
        return ErrOutOfRange
    }
    return nil
}

// Custom type that resolves to single value
type PanMode struct {
    balance bool
    center float64
}

func (p PanMode) marshalToDeviceValue() float64 {
    // Generator implements conversion to device-specific value
}
```

### State Management

- Each level of the hierarchy can be captured as a valid Go variable
- State must be immutable - calling child methods must not modify parent state
- State should be stored by value, not by reference
- `Set`/`Bind` calls must not modify accumulated state
- No caching of state is needed or expected

```go
// Example of valid state capture
track1 := device.Tracks(1)
fx2 := track1.Fx(2)     // track1's state is unmodified
param3 := fx2.Param(3)  // track1 and fx2's states are unmodified

// All of these are valid and refer to the same endpoint
device.Tracks(1).Fx(2).Param(3).Set(0.5)
fx2.Param(3).Set(0.5)
param3.Set(0.5)
```

### Documentation and Compatibility

- Documentation is generated on a best-effort basis from device specs
- No guarantees of backwards-compatibility
- No protection from breaking changes
- Device-specific details (lifecycle, connection management, etc.) are handled outside this specification
- Implementation details of the "engine" behind each API are out of scope

### Error Handling

```go
// Recommended (but not required) error type for range validation
var ErrOutOfRange = errors.New("parameter out of valid range")

// Example Set implementation with validation
func (p *Param) Set(value float64) error {
    // We have access to full state here
    if value < p.state.minValue || value > p.state.maxValue {
        return ErrOutOfRange
    }
    return p.device.setEndpointValue(p.state, value)
}
```
