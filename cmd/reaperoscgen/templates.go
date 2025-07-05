package main

const endpointTemplate = `
// {{.Name}} represents a {{.Description}}
type {{.Name}} struct {
    device *Reaper
    state {{.StateName}}
    {{range .SubEndpoints}}
    {{.Name}} *{{.TypeName}}
    {{end}}
}

/
case TypeRotary:
		goType = "float64"/ State holds the endpoint's path parameters
type {{.StateName}} struct {
    {{range .StateFields}}
    {{.Name}} {{.Type}}
    {{end}}
}

// Bind registers a callback for value changes
func (e *{{.Name}}) Bind(callback func({{.ValueType}}) error) {
    addr := fmt.Sprintf("{{.Pattern}}", {{.StateParams}})
    e.device.bindOSCHandler(addr, callback)
}

// Set sends a value to the endpoint
func (e *{{.Name}}) Set(value {{.ValueType}}) error {
    {{if .HasValidation}}
    if err := e.validate(value); err != nil {
        return err
    }
    {{end}}
    addr := fmt.Sprintf("{{.Pattern}}", {{.StateParams}})
    return e.device.sendOSCMessage(addr, value)
}

{{if .HasValidation}}
func (e *{{.Name}}) validate(value {{.ValueType}}) error {
    {{.ValidationCode}}
    return nil
}
{{end}}
`

type EndpointTemplateData struct {
	Name           string
	Description    string
	StateName      string
	StateFields    []StateField
	SubEndpoints   []SubEndpoint
	Pattern        string
	StateParams    string
	ValueType      string
	HasValidation  bool
	ValidationCode string
}

type StateField struct {
	Name string
	Type string
}

type SubEndpoint struct {
	Name     string
	TypeName string
}
