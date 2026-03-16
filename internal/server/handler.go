package server

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
	"go.abhg.dev/goldmark/mermaid"
)

//go:embed templates
var templateFS embed.FS

var (
	layoutTmpl   *template.Template
	markdownTmpl *template.Template
)

func init() {
	layoutTmpl = template.Must(template.ParseFS(templateFS, "templates/layout.html", "templates/index.html"))
	markdownTmpl = template.Must(template.ParseFS(templateFS, "templates/markdown.html"))
}

var md = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM,
		&mermaid.Extender{
			RenderMode: mermaid.RenderModeServer,
			Compiler:   &mermaid.CLICompiler{},
		},
	),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
		parser.WithASTTransformers(util.Prioritized(&headingIDTransformer{}, 100)),
	),
	goldmark.WithRendererOptions(html.WithXHTML(), html.WithUnsafe()),
)

// HandleIndex serves the main page with tree navigation and welcome content.
func (s *Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	status := s.syncer.Status()
	data := map[string]any{
		"Results": status.Results,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	layoutTmpl.Execute(w, data)
}

// HandleFile serves files from the output directory.
// .md files are rendered as HTML unless ?raw=1 is set.
// Other files are served as-is.
func (s *Server) HandleFile(w http.ResponseWriter, r *http.Request) {
	// strip leading slash
	urlPath := strings.TrimPrefix(r.URL.Path, "/")
	filePath := filepath.Join(s.cfg.OutputDir, filepath.FromSlash(urlPath))

	// security: prevent path traversal
	absOutput, _ := filepath.Abs(s.cfg.OutputDir)
	absFile, _ := filepath.Abs(filePath)
	if !strings.HasPrefix(absFile, absOutput) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	info, err := os.Stat(filePath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// directory: list files or render README.md
	if info.IsDir() {
		s.handleDirectory(w, r, filePath, urlPath)
		return
	}

	// raw mode
	if r.URL.Query().Get("raw") == "1" {
		http.ServeFile(w, r, filePath)
		return
	}

	// markdown rendering
	if strings.HasSuffix(strings.ToLower(filePath), ".md") {
		s.renderMarkdown(w, r, filePath, urlPath)
		return
	}

	// all other files: serve as-is
	http.ServeFile(w, r, filePath)
}

func (s *Server) handleDirectory(w http.ResponseWriter, r *http.Request, dirPath, urlPath string) {
	// check for README.md
	readmePath := filepath.Join(dirPath, "README.md")
	if _, err := os.Stat(readmePath); err == nil {
		readmeURL := urlPath
		if !strings.HasSuffix(readmeURL, "/") {
			readmeURL += "/"
		}
		readmeURL += "README.md"
		s.renderMarkdown(w, r, readmePath, readmeURL)
		return
	}

	// list directory contents
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		http.Error(w, "cannot read directory", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	var buf bytes.Buffer
	buf.WriteString("<h2>Index of /" + template.HTMLEscapeString(urlPath) + "</h2>")
	buf.WriteString("<ul style='list-style:none;padding:0;margin-top:12px;font-size:14px;'>")
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		href := "/" + urlPath
		if !strings.HasSuffix(href, "/") {
			href += "/"
		}
		href += name
		if entry.IsDir() {
			name += "/"
		}
		buf.WriteString("<li style='padding:4px 0;'><a href='" + template.HTMLEscapeString(href) + "'>" + template.HTMLEscapeString(name) + "</a></li>")
	}
	buf.WriteString("</ul>")

	io.WriteString(w, buf.String())
}

func (s *Server) renderMarkdown(w http.ResponseWriter, r *http.Request, filePath, urlPath string) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		http.Error(w, "cannot read file", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := md.Convert(data, &buf); err != nil {
		http.Error(w, "markdown rendering failed", http.StatusInternalServerError)
		return
	}

	// build git source header
	gitHeader := s.buildGitHeader(urlPath)
	content := gitHeader + buf.String()

	// check if this is an AJAX request (from the SPA navigation)
	if r.Header.Get("X-Requested-With") == "XMLHttpRequest" || r.Header.Get("Accept") == "text/html-partial" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		var out bytes.Buffer
		markdownTmpl.Execute(&out, map[string]template.HTML{
			"Content": template.HTML(content),
		})
		w.Write(out.Bytes())
		return
	}

	// full page render with layout
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	layoutTmpl.Execute(w, map[string]any{
		"MarkdownContent": template.HTML(content),
	})
}

// buildGitHeader returns an HTML banner showing the git source of the file.
func (s *Server) buildGitHeader(urlPath string) string {
	// urlPath is like "user-service/docs/api.md"
	parts := strings.SplitN(urlPath, "/", 2)
	if len(parts) < 2 {
		return ""
	}
	repoName := parts[0]
	fileInRepo := parts[1]

	repo, ok := s.repoMap[repoName]
	if !ok {
		return ""
	}

	// build web URL from git URL
	webURL := gitURLToWeb(repo.URL, repo.Branch, fileInRepo)

	return fmt.Sprintf(
		`<div class="git-header">
		<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
		<strong>%s</strong> / %s @ <code>%s</code>
		%s</div>`,
		template.HTMLEscapeString(repoName),
		template.HTMLEscapeString(fileInRepo),
		template.HTMLEscapeString(repo.Branch),
		webLinkHTML(webURL),
	)
}

// gitURLToWeb converts a git clone URL to a web browsable file URL.
func gitURLToWeb(gitURL, branch, filePath string) string {
	// git@host:org/repo.git → https://host/org/repo/blob/branch/path
	if strings.HasPrefix(gitURL, "git@") {
		u := strings.TrimPrefix(gitURL, "git@")
		u = strings.TrimSuffix(u, ".git")
		// git@host:org/repo → host/org/repo
		u = strings.Replace(u, ":", "/", 1)
		return fmt.Sprintf("https://%s/blob/%s/%s", u, branch, filePath)
	}

	// https://host/org/repo.git → https://host/org/repo/blob/branch/path
	u := strings.TrimSuffix(gitURL, ".git")
	return fmt.Sprintf("%s/blob/%s/%s", u, branch, filePath)
}

func webLinkHTML(url string) string {
	if url == "" {
		return ""
	}
	return fmt.Sprintf(`<a href="%s" target="_blank" rel="noopener">View source</a>`, template.HTMLEscapeString(url))
}
