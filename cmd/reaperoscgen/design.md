# REAPER OSC Go Source Generator: Design Document

## Purpose

This document describes the design of a Go source code generator that processes REAPER OSC pattern config files and generates Go API methods for binding to OSC actions. The generator is intended to be run as part of a Go `go generate` step.

---

## Goals

- Parse the REAPER OSC config file and extract all actions, patterns, argument types, wildcards, and documentation.
- Generate idiomatic Go methods on a `Reaper` struct for binding handlers to each action.
- Prefer numeric bindings (int/float/bool) over string bindings when multiple pattern types exist for an action.
- Handle wildcards in OSC paths. If there are multiple wildcards, create `PathXXX` structs containing the full complement.
- Use appropriate Go types in callbacks, based on the OSC pattern's argument type. No multi-argument handlers are needed for the current REAPER spec.
- Copy and preserve relevant documentation/comments from the config file to generated Go code.
- Emit a single Go source file containing all bindings.
- Store all patterns and paths for each action, even if only the "best" is used in the generated API, for future extensibility.

---

## Input Specification

- **Config File:**
  - Path is a constant string in the generator source, easily changeable.
  - Follows REAPER's documented OSC pattern config syntax (see included excerpt).
  - Each action line is formatted as:  
    `<ACTION_NAME> <type_1>/<osc/path/1> <type_2>/<osc/path/2> ...`
  - Comments and documentation may appear before actions and should be associated with subsequent actions.

---

## Output Specification

- **Go Source File:**
  - Contains generated `BindXXX` methods as described below.
  - Includes all struct definitions for wildcard paths.
  - Contains doc-comments copied from the config file.
  - Places everything in a single file (e.g., `reaper_bindings_gen.go`).

---

## Generator CLI/Usage

- **Intended Usage:**
  - Invoked via `go generate` using a directive such as:  
    `//go:generate go run ./cmd/reaperoscgen`
- **Config File Path:**
  - Set as a constant in generator code, or passed as a flag for flexibility.
- **Output File:**
  - Default: `reaper_bindings_gen.go` in the current package.

---

## High-Level Algorithm

1. **Parse Config File**

   - Read all lines, capturing:
     - Block and line comments.
     - Action names and all their patterns (type prefix + OSC path).
   - For each action:
     - Store all patterns for extensibility.
     - Associate documentation with the action.

2. **Select Patterns for Code Generation**

   - For each action, select the "best" pattern according to:
     1. Prefer numeric (n, f, b, t, r, i) types over string (s).
     2. If multiple numeric types, prefer in order: `n`, `f`, `i`, `b`, `t`, `r`.
     3. If only string type exists, use it.
   - Only generate a binding method for the "best" pattern for now.

3. **Data Modeling**

   - For each action:
     - If the selected pattern includes _multiple_ wildcards (`@`), generate a `PathXXX` struct with one field per wildcard (type `int64`).
     - Document each struct.

4. **API Method Generation**

   - For each action:
     - Generate a method:  
       `func (r *Reaper) Bind<ActionName>(path PathXXX, callback func(<type>) error) error`
     - Parameters:
       - `path <type>` if exactly one wildcard; `path PathXXX` if multiple wildcards; otherwise `nil`/empty struct or omit.
       - `callback func(<type>) error`, where `<type>` is the primitive (int64, float64, bool, string) as needed.
     - Method body:
       - Compose the OSC address with wildcards filled in from `path`.
       - Call the appropriate low-level `BindInt`, `BindFloat`, `BindBool`, or `BindString` method.
     - Add doc-comments above the method, including any relevant config comments.

5. **Emit Output**
   - Write all generated types and methods to a single Go source file.
   - Include a package-level comment noting the file is generated.

---

## Example: Config to API

### Config Line

```plaintext
# Sets the track volume.
TRACK_VOLUME n/track/@/volume n/track/volume
```

### Parsed Representation

- **Action Name:** `TRACK_VOLUME`
- **Patterns:**
  - `n/track/@/volume` (numeric, with wildcard)
  - `n/track/volume` (numeric, no wildcard)
- **Documentation:** "Sets the track volume."

### Generated Go Code

```go
// Sets the track volume.
type PathTrackVolume struct {
    TrackIdx int64
}

func (r *Reaper) BindTrackVolume(p PathTrackVolume, callback func(float64) error) error {
    return r.BindFloat(fmt.Sprintf("/track/%d/volume", p.TrackIdx), callback)
}
```

---

## Documentation Handling

- All comments above an action in the config file are captured and output as doc-comments above the corresponding method and type in the generated Go file.

---

## Extensibility

- All parsed patterns per action are stored in the generator's internal data structures, so future versions can emit alternative bindings or allow the developer to select a different binding style.

---

## Notes on Multi-Argument Patterns

- The REAPER OSC config and spec, as currently written, do not define patterns where a single OSC message has multiple arguments that must be handled in a Go callback.
- Batch operations (e.g., `/track/3/fx/1,2,5/fxparam/6,7,7/value 0.25 0.5 0.75`) are out of scope for the basic generator, and would require advanced handling.
- If such patterns are introduced in the future, the generator can be extended to produce an `ArgsXXX` struct and a callback of the form `func(ArgsXXX) error`.

---

## Implementation Sketch

- **Generator main function:**

  - Parse config file into internal structs (`Action`, `Pattern`, etc.).
  - For each action, select best pattern, prepare types, generate code.
  - Write to output file.

- **Structs:**

  - `type Action struct { Name string; Patterns []Pattern; Doc []string }`
  - `type Pattern struct { ArgType string; Path string; Wildcards []string; ... }`

- **Codegen helpers:**
  - Render path templates with `fmt.Sprintf`.
  - Render doc-comments.
  - Render struct and method declarations.

---

## Example Usage

```go
//go:generate go run ./cmd/reaperoscgen
```

---

## File Layout

- Generator code: `cmd/reaperoscgen/main.go`
- Output file: `reaper_bindings_gen.go`

---

## Conclusion

This generator will provide a robust, extensible, and type-safe way to bind Go code to REAPER OSC actions, ensuring maintainability and future adaptability as OSC protocols or user needs evolve.
