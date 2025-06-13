package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"
	"os"
	"strings"
	"text/template"
	"time"

	"golang.org/x/tools/go/packages"
)

type importInfo struct {
	path string
	name string
}

// Track both required imports from Send functions and imports from source files
var importSpecs = make(map[string]*ast.ImportSpec)

// Track imports needed for the generated code
var requiredImports = map[string]importInfo{"fmt": {path: "fmt", name: "fmt"}, "reflect": {path: "reflect", name: "reflect"}}

// sendPattern represents a discovered Send function call
type sendPattern struct {
	name     string        // Original function name (e.g., "SendTrackVolume")
	recv     string        // Receiver type name (e.g., "Track")
	pkg      string        // Package name (e.g., "reaper")
	modeExpr ast.Expr      // Mode expression used in call
	args     []ast.Expr    // Other arguments to the send
	callExpr *ast.CallExpr // Original call expression
}

// bindPattern represents a discovered bind() call
type bindPattern struct {
	modeExpr ast.Expr       // Mode expression for the bind
	control  ast.Expr       // Control expression (e.g., c.Fader)
	callback *ast.FuncLit   // Callback function
	sends    []*sendPattern // Send calls found in the callback
}

func main() {
	fset := token.NewFileSet()

	// Load packages for type information
	cfg := &packages.Config{
		Mode:  packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo,
		Tests: false,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		log.Fatal(err)
	}

	var patterns []bindPattern = []bindPattern{}
	var sends map[string]*sendPattern = map[string]*sendPattern{}

	// Process each package
	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			processFile(fset, file, pkg.TypesInfo, &patterns, sends)
		}
	}

	// Generate the output file
	if err := generateOutput(patterns, sends); err != nil {
		log.Fatal(err)
	}
}

func processFile(fset *token.FileSet, file *ast.File, info *types.Info, patterns *[]bindPattern, sends map[string]*sendPattern) error {
	// Collect imports from the source file
	for _, imp := range file.Imports {
		// The import path is always in quotes, so we trim them
		path := strings.Trim(imp.Path.Value, `"`)
		// Store the complete import spec
		importSpecs[path] = imp
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.CallExpr:
			// Look for bind() calls
			if ident, ok := node.Fun.(*ast.Ident); ok && ident.Name == "bind" {
				if pattern, ok := extractBindPattern(node, info); ok {
					*patterns = append(*patterns, pattern)
				}
			}

			// Look for Send functions
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
				if strings.HasPrefix(sel.Sel.Name, "Send") {
					if pattern, ok := extractSendPattern(node, info); ok {
						sends[pattern.name] = pattern
					}
				}
			}
		}
		return true
	})
	return nil
}

func extractSendPattern(call *ast.CallExpr, info *types.Info) (*sendPattern, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, false
	}

	// Get type information
	recvType := info.Types[sel.X].Type
	if recvType == nil {
		return nil, false
	}

	// // Get package information
	// pkgPath := ""
	// pkgName := ""
	// if named, ok := recvType.(*types.Named); ok {
	// 	pkg := named.Obj().Pkg()
	// 	pkgPath = pkg.Path()
	// 	pkgName = pkg.Name()
	//
	// 	// Record that we need this import
	// 	requiredImports[pkgName] = importInfo{
	// 		path: pkgPath,
	// 		name: pkgName,
	// 	}
	// }

	// Extract mode expression (first argument)
	if len(call.Args) < 1 {
		return nil, false
	}

	recvTypeParts := strings.Split(recvType.String(), `/`)

	return &sendPattern{
		name:     sel.Sel.Name,
		recv:     recvTypeParts[len(recvTypeParts)-1],
		pkg:      getPkgName(recvType),
		modeExpr: call.Args[0],
		args:     call.Args[1:],
		callExpr: call,
	}, true
}

func extractBindPattern(call *ast.CallExpr, info *types.Info) (bindPattern, bool) {
	if len(call.Args) != 4 {
		return bindPattern{}, false
	}

	callback, ok := call.Args[3].(*ast.FuncLit)
	if !ok {
		return bindPattern{}, false
	}

	// Find Send calls in the callback
	var sends []*sendPattern
	ast.Inspect(callback, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok {
			if pattern, ok := extractSendPattern(call, info); ok {
				sends = append(sends, pattern)
			}
		}
		return true
	})

	return bindPattern{
		modeExpr: call.Args[0],
		control:  call.Args[1],
		callback: callback,
		sends:    sends,
	}, true
}

func getPkgName(t types.Type) string {
	if named, ok := t.(*types.Named); ok {
		return named.Obj().Pkg().Name()
	}
	return ""
}

func generateImports(buf *bytes.Buffer) error {
	if len(importSpecs) == 0 && len(requiredImports) == 0 {
		return nil
	}

	_, err := buf.WriteString("\nimport (\n")
	if err != nil {
		return err
	}

	// Write all imports from source files
	for _, imp := range importSpecs {
		if imp.Name != nil {
			// Named import
			if _, err := fmt.Fprintf(buf, "\t%s %s\n", imp.Name.Name, imp.Path.Value); err != nil {
				return err
			}
		} else {
			// Regular import
			if _, err := fmt.Fprintf(buf, "\t%s\n", imp.Path.Value); err != nil {
				return err
			}
		}
	}

	for _, imp := range requiredImports {
		if _, err := fmt.Fprintf(buf, "\t%s %q\n", imp.name, imp.path); err != nil {
			return err
		}
	}

	_, err = buf.WriteString(")\n\n")
	return err
}

// generateOutput creates the bindings_gen.go file
func generateOutput(patterns []bindPattern, sends map[string]*sendPattern) error {
	var buf bytes.Buffer

	// Generate file header
	if err := generateHeader(&buf); err != nil {
		return fmt.Errorf("generating header: %w", err)
	}

	// Generate imports
	if err := generateImports(&buf); err != nil {
		return fmt.Errorf("generating imports: %w", err)
	}

	// Generate state types
	if err := generateTypes(&buf); err != nil {
		return fmt.Errorf("generating types: %w", err)
	}

	// Generate mode-aware send functions
	for _, send := range sends {
		if err := generateSendFunction(&buf, send); err != nil {
			return fmt.Errorf("generating send function %s: %w", send.name, err)
		}
	}

	// Generate mode-aware bind wrapper
	if err := generateBindWrapper(&buf); err != nil {
		return fmt.Errorf("generating bind wrapper: %w", err)
	}

	// Generate mode transition handler
	if err := generateModeTransition(&buf); err != nil {
		return fmt.Errorf("generating mode transition: %w", err)
	}

	// Format and write the file
	// formattedBuf, err := format.Source(buf.Bytes())
	// if err != nil {
	// 	return fmt.Errorf("formatting output: %w", err)
	// }

	formattedBuf := buf.Bytes()

	return os.WriteFile("bindings_gen.go", formattedBuf, 0644)
}

func generateHeader(buf *bytes.Buffer) error {
	return headerTemplate.Execute(buf, struct {
		Timestamp string
		User      string
	}{
		Timestamp: time.Now().UTC().Format("2006-01-02 15:04:05"),
		User:      "jdginn",
	})
}

func generateTypes(buf *bytes.Buffer) error {
	_, err := buf.WriteString(`
type controlState struct {
    value any
    sender func(any) error
}

type generatedModeManager struct {
    states map[Mode]map[string]*controlState
    currentMode Mode
}

var mm = &generatedModeManager{
    states: make(map[Mode]map[string]*controlState),
}
`)
	return err
}

func generateSendFunction(buf *bytes.Buffer, pattern *sendPattern) error {
	// Create unique key for this send operation
	keyName := fmt.Sprintf("CONTROL_%s_%s", pattern.pkg, pattern.name)

	// Convert the original arguments to a parameter list
	params, argNames, err := formatParams(pattern.args)
	if err != nil {
		return err
	}

	data := struct {
		OrigName  string
		ModeAware string
		Key       string
		Params    string
		ArgNames  string
		RecvType  string
		PkgName   string
	}{
		OrigName:  pattern.name,
		ModeAware: pattern.name + "_ModeAware",
		Key:       keyName,
		Params:    params,
		ArgNames:  argNames,
		RecvType:  pattern.recv,
		PkgName:   pattern.pkg,
	}

	return sendFuncTemplate.Execute(buf, data)
}

func generateBindWrapper(buf *bytes.Buffer) error {
	return bindWrapperTemplate.Execute(buf, nil)
}

func generateModeTransition(buf *bytes.Buffer) error {
	return modeTransitionTemplate.Execute(buf, nil)
}

// Templates
var headerTemplate = template.Must(template.New("header").Parse(`// Code generated by github.com/jdginn/arpad/cmd/generatebindings DO NOT EDIT.
// Generated at {{ .Timestamp }} by {{ .User }}

package main

`))

var sendFuncTemplate = template.Must(template.New("send").Parse(`
const {{ .Key }} = "{{ .OrigName }}_key"

func {{ .ModeAware }}(modes Mode, {{ .Params }}) error {
    // Store state for each mode this send applies to
    for mode := Mode(1); mode <= ALL; mode <<= 1 {
        if modes&mode != 0 {
            if mm.states[mode] == nil {
                mm.states[mode] = make(map[string]*controlState)
            }
            
            value := struct { {{ .Params }} }{ {{ .ArgNames }} }
            
            if mm.states[mode][{{ .Key }}] == nil {
                mm.states[mode][{{ .Key }}] = &controlState{
                    sender: func(v any) error {
                        params := v.(struct { {{ .Params }} })
                        return {{ .RecvType }}.{{ .OrigName }}({{ .ArgNames }})
                    },
                }
            }
            
            mm.states[mode][{{ .Key }}].value = value
            
            // Execute only if we're in this mode
            if mode == mm.currentMode {
                return {{ .RecvType }}.{{ .OrigName }}({{ .ArgNames }})
            }
        }
    }
    return nil
}
`))

var bindWrapperTemplate = template.Must(template.New("bind").Parse(`
func bind_ModeAware(control interface{ Bind(func(any) error) }, bindModes Mode, callback func(any) error) {
    control.Bind(func(args any) error {
        // Only execute if current mode matches any bind modes
        if mm.currentMode&bindModes == 0 {
            return nil
        }
        return callback(args)
    })
}
`))

var modeTransitionTemplate = template.Must(template.New("transition").Parse(`
func applyModeState_gen(newMode Mode) error {
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
`))

// Helper functions
func formatParams(args []ast.Expr) (paramsStr string, argNamesStr string, error error) {
	var params []string
	var argNames []string

	for i := range args {
		argName := fmt.Sprintf("arg%d", i)
		// TODO: Get proper type information for each parameter
		params = append(params, fmt.Sprintf("%s interface{}", argName))
		argNames = append(argNames, argName)
	}

	return strings.Join(params, ", "), strings.Join(argNames, ", "), nil
}
