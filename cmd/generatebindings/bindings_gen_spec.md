# Mode-Aware Bindings Generator Specification

## Overview

This document specifies how to generate mode-aware implementations for device control bindings in the arpad project. The generator transforms high-level bind/send patterns into efficient mode-aware implementations.

## Input Pattern

The generator looks for bind() calls in the following format:

```go
bind(ModeExpr, control, nil, func(args ArgType) error {
    return device.SendSomething(ModeExpr2, param1, param2)
})
```

Where:

- `ModeExpr`: A Mode constant or bitwise combination of Mode constants (e.g., `MIX|RECORD`)
- `control`: A control object with a Bind method (e.g., `c.Solo`, `c.Fader`)
- `ArgType`: The type of argument the control's callback receives (e.g., `bool`, `dev.ArgsPitchBend`)
- `device.SendSomething`: Any method starting with "Send"
- `ModeExpr2`: Mode expression for the send operation, can differ from bind's ModeExpr

## Generated Code Structure

### 1. State Storage Types

```go
type controlState struct {
    value any
    sender func(any) error
}

type generatedModeManager struct {
    states map[Mode]map[string]*controlState
    currentMode Mode  // Always one-hot (exactly one bit set)
}
```

### 2. Generated Send Functions

For each unique Send function discovered, generate a mode-aware version:

```go
// Example for SendTrackVolume
func SendTrackVolume_ModeAware(modes Mode, trackNum int64, value float64) error {
    key := fmt.Sprintf("track_%d_volume", trackNum)

    // Store state for each mode this send applies to
    for mode := Mode(1); mode <= ALL; mode <<= 1 {
        if modes&mode != 0 {
            if mm.states[mode] == nil {
                mm.states[mode] = make(map[string]*controlState)
            }
            mm.states[mode][key].value = value

            // Execute only if we're in this mode
            if mode == mm.currentMode {
                return reaper.Track.SendTrackVolume(trackNum, value)
            }
        }
    }
    return nil
}
```

### 3. Bind Function Generation

Generate a mode-aware wrapper for each bind pattern:

```go
func bind_ModeAware(control Control, bindModes Mode, callback func(args ArgType) error) {
    control.Bind(func(args ArgType) error {
        // Only execute if current mode matches any bind modes
        if bindModes&mm.currentMode == 0 {
            return nil
        }
        return callback(args)
    })
}
```

### 4. Mode Transition Handler

Generate a single mode transition function:

```go
func applyModeState_gen(mm *generatedModeManager, newMode Mode) error {
    if mm.currentMode == newMode {
        return nil
    }

    if modeStates, ok := mm.states[newMode]; ok {
        for key, state := range modeStates {
            // Only send if value differs from current mode
            if currentStates := mm.states[mm.currentMode]; currentStates != nil {
                if currentState := currentStates[key]; currentState != nil {
                    if reflect.DeepEqual(currentState.value, state.value) {
                        continue
                    }
                }
            }

            if state.value != nil {
                if err := state.sender(state.value); err != nil {
                    return err
                }
            }
        }
    }

    mm.currentMode = newMode
    return nil
}
```

### 5. Initialization

Generate initialization code:

```go
func init() {
    mm := &generatedModeManager{
        states: make(map[Mode]map[string]*controlState),
    }

    // Initialize all bindings
    bind_[UniqueIdentifier1](mm, ...)
    bind_[UniqueIdentifier2](mm, ...)
    // ... one call per discovered binding

    // Hook into mode changes
    mm.onModeChange = applyModeState_gen
}
```

## Generation Process

1. **AST Analysis**

   - Parse all .go files in the project
   - For each Send function:
     - Generate a mode-aware version
     - Record the mapping between original and generated versions
   - For each bind() call:
     - Extract bind modes and control information
     - Transform the callback to use mode-aware Send functions
     - Generate mode-aware bind wrapper

2. **Send Function Transformation**
   For each Send call in a callback, transform:

```go
reaper.Track.SendTrackVolume(MIX, trackNum, value)
```

into:

```go
SendTrackVolume_ModeAware(MIX, trackNum, value)
```

3. **Bind Transformation**
   Transform each bind call:

```go
bind(MIX|RECORD, c.Fader, nil, func(args dev.ArgsPitchBend) error {
    if err := reaper.Track.SendTrackVolume(MIX, trackNum, normalizeArgs(args)); err != nil {
        return err
    }
    return motu.SendInputGain(RECORD, trackNum, normalizeArgs(args))
})
```

into:

```go
bind_ModeAware(c.Fader, MIX|RECORD, func(args dev.ArgsPitchBend) error {
    if err := SendTrackVolume_ModeAware(MIX, trackNum, normalizeArgs(args)); err != nil {
        return err
    }
    return SendInputGain_ModeAware(RECORD, trackNum, normalizeArgs(args))
})
```

## Important Constraints

1. **Mode Handling**

   - `currentMode` must always be one-hot (exactly one bit set)
   - `bindModes` and `sendModes` can be compound (multiple bits set)
   - Mode transitions only occur between one-hot modes

2. **Performance**

   - Minimize allocations in generated code
   - Use constant keys to avoid string allocations
   - Skip state updates when values haven't changed
   - Minimize runtime type assertions

3. **Type Safety**

   - Generated code must maintain compile-time type safety
   - Proper error handling for all operations
   - Safe type assertions where necessary

4. **Send Function Independence**
   - Each Send call must be independently mode-aware
   - Multiple Send calls within a single bind must maintain their individual mode behaviors
   - State storage must be unique per Send operation

## Usage Example

Original code:

```go
bind(MIX|RECORD, c.Fader, nil, func(args dev.ArgsPitchBend) error {
    if err := reaper.Track.SendTrackVolume(MIX, trackNum, normalizeArgs(args)); err != nil {
        return err
    }
    return motu.SendInputGain(RECORD, trackNum, normalizeArgs(args))
})
```

This will generate appropriate bind functions and state handling code according to the above specification.
