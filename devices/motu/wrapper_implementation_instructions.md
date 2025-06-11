# MOTU Datastore API Implementation Guide

This guide describes the implementation patterns for creating Go bindings to the MOTU Datastore API.

## Overall Structure

The API is implemented as a set of endpoint structs organized into binding groups. Each endpoint represents a specific path in the datastore API.

```go
// Main device struct
type MOTU struct {
    d      *datastore.Datastore
    Global *GlobalBindings
    AVB    *AVBBindings
    Router *RouterBindings
    Mixer  *MixerBindings
}

// Binding group example
type GlobalBindings struct {
    m *MOTU
    UID *UIDEndpoint
    // ... other endpoints
}
```

## Endpoint Implementation Rules

### 1. Single Parameter Paths

For paths with a single parameter (e.g., `avb/<uid>/entity_name`):

- Accept the parameter directly with its base type
- Use descriptive parameter names
- DO NOT create a Path struct

```go
type EntityNameEndpoint struct {
    m *MOTU
}

func (e *EntityNameEndpoint) Bind(deviceUID string, callback func(string) error) {
    path := fmt.Sprintf("avb/%s/entity_name", deviceUID)
    e.m.d.BindString(path, callback)
}
```

### 2. Multiple Parameter Paths

For paths with multiple parameters (e.g., `mix/chan/<index>/matrix/group/<index>/send`):

- Create a Path struct with descriptive field names
- Use the Path struct as the first parameter in Bind and Set methods

```go
type PathChannelGroupSend struct {
    ChannelIndex int64
    GroupIndex   int64
}

type ChannelGroupSendEndpoint struct {
    m *MOTU
}

func (e *ChannelGroupSendEndpoint) Bind(p PathChannelGroupSend, callback func(float64) error) {
    path := fmt.Sprintf("mix/chan/%d/matrix/group/%d/send", p.ChannelIndex, p.GroupIndex)
    e.m.d.BindFloat(path, callback)
}
```

### 3. No-Parameter Paths

For paths with no parameters:

- Use empty struct{} as the path parameter type
- Still implement both Bind and Set (if writable)

```go
type WordClockModeEndpoint struct {
    m *MOTU
}

func (e *WordClockModeEndpoint) Bind(_ struct{}, callback func(string) error) {
    e.m.d.BindString("ext/wordClockMode", callback)
}
```

### 4. Read-Only vs Writable Endpoints

- For read-only endpoints (Permission: r), implement ONLY Bind()
- For writable endpoints (Permission: rw), implement both Bind() and Set()

### 5. Value Validation

- Implement range validation in Set() methods when min/max values are specified
- Return descriptive errors for invalid values
- Use the units specified in the API spec for error messages

```go
func (e *MainFaderEndpoint) Set(mainIndex int64, val float64) error {
    if val < 0 || val > 4 {
        return fmt.Errorf("fader value must be between 0 and 4")
    }
    path := fmt.Sprintf("mix/main/%d/matrix/fader", mainIndex)
    return e.m.d.SetFloat(path, val)
}
```

### 6. Type Mapping

Map API types to Go types as follows:

- `string` -> `string`
- `real` -> `float64`
- `int` -> `int64`
- `real_bool` -> `bool`
- `int_bool` -> `bool`
- `real_enum` -> appropriate type based on values
- `*_opt` types use the same type as their base type

### 7. Naming Conventions

- Endpoint structs: `{Description}Endpoint`
- Path structs (when needed): `Path{Description}`
- Binding groups: `{Section}Bindings`
- Use descriptive names that match the API path components

## Implementation Process

1. Create the main MOTU struct and binding groups
2. For each path in the API spec:
   - Determine if it needs a Path struct based on parameter count
   - Create the appropriate endpoint struct
   - Implement Bind() method
   - Implement Set() method if writable
   - Add to appropriate binding group
3. Add any required validation in Set() methods

## Example Usage

The resulting API should be usable as follows:

```go
motu := NewMOTU(datastore)

// Single parameter
motu.AVB.EntityName.Bind("device123", func(name string) error {
    fmt.Println("Name changed:", name)
    return nil
})

// Multiple parameters
motu.Mixer.ChannelGroupSend.Bind(PathChannelGroupSend{
    ChannelIndex: 0,
    GroupIndex: 1,
}, func(val float64) error {
    fmt.Println("Send level changed:", val)
    return nil
})
```
