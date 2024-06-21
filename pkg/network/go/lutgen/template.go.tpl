// Code generated by go generate; DO NOT EDIT.

//go:build linux

package {{ .Package }}

{{ if gt (.Imports | len) 1 }}
import (
{{ range .Imports }}{{"\t"}}"{{. -}}"
{{ end }}
)
{{ else if .Imports | len | eq 1 }}
import "{{ index .Imports 0 -}}"
{{ end }}

var SupportedArchitectures = {{ printf "%#v" .SupportedArchitectures }}

var MinGoVersion = goversion.GoVersion{Major: {{ .MinGoVersion.Major }}, Minor: {{ .MinGoVersion.Minor }}, Rev: {{ .MinGoVersion.Rev }}}

{{ range .LookupFunctions }}
{{ with $lookupFn := . }}
{{ if gt (.RenderedDocComment | len) 1 }}
{{ .RenderedDocComment -}}
{{ end -}}
func {{ .Name -}}(version goversion.GoVersion, goarch string) ({{ .OutputType -}}, error) {
	switch goarch {
	{{ range .ArchCases -}}
	case "{{ .Arch }}":
		{{ range .Branches -}}
		if version.AfterOrEqual(goversion.GoVersion{Major: {{ .Version.Major }}, Minor: {{ .Version.Minor }}, Rev: {{ .Version.Rev }}}) {
			return {{ .RenderedValue }}, nil
		}
		{{ end }}
		{{- if .HasMin -}}
		return {{ $lookupFn.OutputZeroValue }}, fmt.Errorf("unsupported version go%d.%d.%d (min supported: go%d.%d.%d)", version.Major, version.Minor, version.Rev, {{ .Min.Major }}, {{ .Min.Minor }}, {{ .Min.Rev }})
		{{ else }}
		return {{ $lookupFn.OutputZeroValue }}, fmt.Errorf("unsupported version go%d.%d.%d", version.Major, version.Minor, version.Rev)
		{{ end -}}
	{{ end -}}
	default:
		return {{ .OutputZeroValue }}, fmt.Errorf("unsupported architecture %q", goarch)
	}
}
{{ end }}
{{ end }}