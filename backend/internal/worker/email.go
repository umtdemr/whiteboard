package worker

type EmailJob struct {
	To       string         `json:"to"`
	TmplData map[string]any `json:"tmpl_data"`
	TmplFile string         `json:"tmpl_file"`
}
