package report

import (
	"encoding/json"
	"io"
)

type jsonTool struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Version string `json:"version"`
}

type jsonFix struct {
	Kind string            `json:"kind"`
	Env  map[string]string `json:"env,omitempty"`
}

type jsonFinding struct {
	Rule      string   `json:"rule"`
	Title     string   `json:"title"`
	Location  string   `json:"location"`
	Path      string   `json:"path"`
	EnvVar    string   `json:"envVar,omitempty"`
	Default   bool     `json:"default"`
	Offending []string `json:"offending"`
	Trigger   string   `json:"trigger"`
	Symptom   string   `json:"symptom"`
	Breaks    string   `json:"breaks"`
	Advice    string   `json:"advice,omitempty"`
	Fix       jsonFix  `json:"fix"`
	Repro     string   `json:"repro"`
	Evidence  []string `json:"evidence"`
}

type jsonResult struct {
	Version   string              `json:"version"`
	CodePage  uint32              `json:"codePage"`
	ElapsedMs int64               `json:"elapsedMs"`
	Root      string              `json:"root"`
	Tools     map[string]jsonTool `json:"tools"`
	Findings  []jsonFinding       `json:"findings"`
}

func JSON(w io.Writer, r Result) error {
	out := jsonResult{
		Version: r.Version, CodePage: r.CodePage, ElapsedMs: r.Elapsed.Milliseconds(), Root: r.Root,
		Tools: map[string]jsonTool{}, Findings: []jsonFinding{},
	}
	for k, t := range r.Tools {
		out.Tools[k] = jsonTool{Name: t.Name, Path: t.Path, Version: t.Version}
	}
	for _, f := range r.Findings {
		fix := jsonFix{Kind: f.Rule.Fix.String()}
		if len(f.Rule.Env) > 0 {
			fix.Env = map[string]string{}
			for _, e := range f.Rule.Env {
				fix.Env[e.Var] = e.Under(r.Root)
			}
		}
		out.Findings = append(out.Findings, jsonFinding{
			Rule: f.Rule.ID, Title: f.Rule.Title, Location: f.Where.Key, Path: f.Where.Path, EnvVar: f.Where.EnvVar,
			Default: f.Where.Default, Offending: f.Offending, Trigger: f.Rule.Trigger.String(), Symptom: f.Rule.Symptom,
			Breaks: f.Rule.Breaks, Advice: f.Rule.Advice, Fix: fix, Repro: f.Rule.Repro, Evidence: f.Rule.Evidence,
		})
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
