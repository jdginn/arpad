# REAPER OSC Go Source Generator: Design Document

## Purpose

This document describes the design of a Go source code generator that processes REAPER OSC pattern config files and generates Go API methods for binding to OSC actions. The generator is intended to be run as part of a Go `go generate` step.

---

## Goals

- Parse the REAPER OSC config file and extract all actions, patterns, argument types, wildcards, and documentation.
- Generate idiomatic Go methods on a `Reaper` struct for binding handlers to each action.
- Prefer numeric bindings (int/float/bool) over string bindings when multiple pattern types exist for an action.
- Handle wildcards in OSC paths via parameterized `PathXXX` structs or single parameters when only one wildcard exists.
- Use appropriate Go types in callbacks, based on the OSC pattern's argument type.
- Copy and preserve relevant documentation/comments from the config file to generated Go code.
- Emit a single Go source file containing all bindings.
- Store all patterns and paths for each action, even if only the "best" is used in the generated API, for future extensibility.

---

## Input Specification

- **Config File:**
  - Path is a constant string in the generator source, easily changeable.
  - Follows REAPER's documented OSC pattern config syntax.
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
   - If the action has no valid patterns, skip it.

2. **Group and Filter Patterns for Generation**

   - For each action, group all patterns (across all config lines) by action name.

   **NOTE:** when comparing paths within the same action, we must always compare element-by-element from left to right, ignoring wildcards. It is not sufficient simply to compare path lengths.

   - If there are two patterns that have identical elements other than wildcards, keep only the pattern with the most wildcards.
   - Note that this requires comparing paths element by element, where wildcards are simply ignored in the comparison.
   - Sometimes there are multiple patterns that are identical up to some point and then one or more paths add additional elements to the right of that point (e.g., `/track/@/volume` and `/track/@/volume/str`). In this case the following rules apply:
     - The path which serves as the base of the other paths (i.e. the one that does not append any additional elements) is the "main" path
     - Ignore paths that append `/str`
     - Keep the other paths

   **NOTE**: the above rules would not apply in a situation where the only path for an action ends in `/str`. In that case, simply use the path without any modifications, since the above rules only apply if there are multiple paths for an action.

3. **Determine Method Naming**

   - Generate `Bind<ActionName>` for the main path.
   - For all other remaining paths, generate `Bind<ActionName><Suffix>`, where `<Suffix>` is the CamelCase of the segments after the main path.
     - Never generate duplicate method names for the same action.
   - Some action names or path elements may contain `+` or `-`, which should be replaced with `Plus` and `Minus`, respectively

4. **Wildcard Handling**

   - If multiple wildcards in a path, generate a `PathXXX` struct.
   - If exactly one wildcard, use a single int64 parameter.
   - If no wildcards, the `path` parameter is `nil` or empty struct.

5. **API Method Generation**

   - For each action path:
     - Generate a method:  
       `func (r *Reaper) Bind<ActionName>[<Suffix>](<wildcard params>, callback func(<type>) error) error`
       where `<wildcard params>` may be `_`, `int64`, or `PathXXX` as needed.
     - Compose the OSC address with wildcards filled in from the parameters.
     - Call the appropriate low-level `BindInt`, `BindFloat`, `BindBool`, or `BindString` method.
     - Add doc-comments above the method, including any relevant config comments.

6. **Emit Output**
   - Write all generated types and methods to a single Go source file.
   - Include a package-level comment noting the file is generated.

---

## Examples

### Example 1: Numeric and String for Same Path

```
TRACK_VOLUME n/track/volume
TRACK_VOLUME s/track/volume/str
```

**Result:** Only generate

```go
func (r *Reaper) BindTrackVolume(callback func(float64) error) error
```

(No `BindTrackVolumeStr`.)

---

### Example 2: Multiple Numeric Paths

```
TRACK_VOLUME n/track/volume
TRACK_VOLUME f/track/volume/db
```

**Result:** Generate

```go
func (r *Reaper) BindTrackVolume(callback func(float64) error) error
func (r *Reaper) BindTrackVolumeDb(callback func(float64) error) error
```

---

### Example 3: Only String Path

```
MY_FOOBAR s/foo/bar/str
```

**Result:** Only generate

```go
func (r *Reaper) BindMyFoobar(callback func(string) error) error
```

(No `BindMyFoobarStr`.)

---

### Example 4: Multiple Numeric and String Paths (string only present for one path)

```
MY_MULTI n/foo/bar
MY_MULTI s/foo/bar/str
MY_MULTI f/foo/bar/baz
```

**Result:** Generate

```go
func (r *Reaper) BindMyMulti(callback func(float64) error) error // for n/foo/bar
func (r *Reaper) BindMyMultiBaz(callback func(float64) error) error // for f/foo/bar/baz
```

(No `BindMyMultiStr`.)

---

### Example 5: Suffix is Multi-Segment

```
MY_FOO n/foo/bar f/foo/bar/baz f/foo/bar/quz f/foo/bar/howdy/baz
```

**Result:** Generate

```go
func (r *Reaper) BindMyFoo(callback func(float64) error) error // for n/foo/bar
func (r *Reaper) BindMyFooBaz(callback func(float64) error) error // for f/foo/bar/baz
func (r *Reaper) BindMyFooQuz(callback func(float64) error) error // for f/foo/bar/quz
func (r *Reaper) BindMyFooHowdyBaz(callback func(float64) error) error // for f/foo/bar/howdy/baz
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
  - For each action, group and filter patterns as specified.
  - Determine method names and generate code.
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
