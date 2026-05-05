package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/adhaniscuber/reprac/internal/config"
	"github.com/adhaniscuber/reprac/internal/github"
)

type Server struct {
	cfg     *config.Config
	cfgPath string
	gh      *github.Client
	version string

	mu      sync.RWMutex
	results map[string]*github.RepoStatus
	loading map[string]bool
}

func NewServer(cfgPath string, cfg *config.Config, gh *github.Client, version string) *Server {
	return &Server{
		cfg:     cfg,
		cfgPath: cfgPath,
		gh:      gh,
		version: version,
		results: make(map[string]*github.RepoStatus),
		loading: make(map[string]bool),
	}
}

func repoKey(owner, repo string) string { return owner + "/" + repo }

// DefaultPort is the preferred port; if busy, Start falls back to a random one.
const DefaultPort = 4242

// Start binds 127.0.0.1:port (0 = random). If the requested port is busy, it
// falls back to a random free port. Serves until ctx is done.
func (s *Server) Start(ctx context.Context, port int) (url string, wait func() error, shutdown func() error, err error) {
	mux := http.NewServeMux()
	s.routes(mux)

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil && port != 0 {
		// Fall back to random
		ln, err = net.Listen("tcp", "127.0.0.1:0")
	}
	if err != nil {
		return "", nil, nil, err
	}

	srv := &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	tcpAddr := ln.Addr().(*net.TCPAddr)
	url = fmt.Sprintf("http://127.0.0.1:%d", tcpAddr.Port)

	done := make(chan error, 1)
	go func() {
		done <- srv.Serve(ln)
	}()

	// Auto-refresh all repos on start (best-effort, async).
	go s.refreshAll(context.Background())

	wait = func() error {
		err := <-done
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
	shutdown = func() error {
		shCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return srv.Shutdown(shCtx)
	}

	// If context is cancelled, shutdown.
	go func() {
		<-ctx.Done()
		_ = shutdown()
	}()

	return url, wait, shutdown, nil
}

// ── Routes ───────────────────────────────────────────────────────────────────

func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("/api/meta", s.handleMeta)
	mux.HandleFunc("/api/repos", s.handleRepos)               // GET (list), POST (add)
	mux.HandleFunc("/api/repos/refresh-all", s.handleRefreshAll) // POST
	mux.HandleFunc("/api/repos/", s.handleRepoItem)           // DELETE, POST .../refresh

	mux.HandleFunc("/", s.handleIndex)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "index not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"version": s.version,
		"hasAuth": s.gh.HasAuth(),
	})
}

func (s *Server) handleRepos(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listRepos(w, r)
	case http.MethodPost:
		s.addRepo(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleRepoItem(w http.ResponseWriter, r *http.Request) {
	// /api/repos/{owner}/{repo}            DELETE
	// /api/repos/{owner}/{repo}/refresh    POST
	rest := strings.TrimPrefix(r.URL.Path, "/api/repos/")
	parts := strings.Split(rest, "/")

	if len(parts) >= 3 && parts[2] == "refresh" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.refreshOne(w, r, parts[0], parts[1])
		return
	}
	if len(parts) == 2 {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		s.deleteRepo(w, r, parts[0], parts[1])
		return
	}
	http.NotFound(w, r)
}

// ── Handlers ─────────────────────────────────────────────────────────────────

type repoView struct {
	Owner        string             `json:"owner"`
	Repo         string             `json:"repo"`
	Notes        string             `json:"notes,omitempty"`
	Status       string             `json:"status"` // loading|clean|behind|no_release|error|unknown
	Branch       string             `json:"branch,omitempty"`
	TagName      string             `json:"tagName,omitempty"`
	RefType      string             `json:"refType,omitempty"`
	RefDate      *time.Time         `json:"refDate,omitempty"`
	CommitsAhead int                `json:"commitsAhead"`
	Commits      []commitView       `json:"commits,omitempty"`
	ErrorMsg     string             `json:"errorMsg,omitempty"`
	LastChecked  *time.Time         `json:"lastChecked,omitempty"`
	URL          string             `json:"url"`
}

type commitView struct {
	SHA     string    `json:"sha"`
	Message string    `json:"message"`
	Date    time.Time `json:"date"`
}

func (s *Server) listRepos(w http.ResponseWriter, _ *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]repoView, 0, len(s.cfg.Repos))
	for _, r := range s.cfg.Repos {
		key := repoKey(r.Owner, r.Repo)
		v := repoView{
			Owner: r.Owner,
			Repo:  r.Repo,
			Notes: r.Notes,
			URL:   fmt.Sprintf("https://github.com/%s/%s", r.Owner, r.Repo),
		}
		if s.loading[key] {
			v.Status = "loading"
		} else if res, ok := s.results[key]; ok && res != nil {
			v.Status = res.Status.String()
			v.Branch = res.Branch
			v.TagName = res.TagName
			v.RefType = res.RefType
			if !res.RefDate.IsZero() {
				rd := res.RefDate
				v.RefDate = &rd
			}
			v.CommitsAhead = res.CommitsAhead
			v.ErrorMsg = res.ErrorMsg
			lc := res.LastChecked
			v.LastChecked = &lc
			for _, c := range res.Commits {
				v.Commits = append(v.Commits, commitView{SHA: c.SHA, Message: c.Message, Date: c.Date})
			}
		} else {
			v.Status = "unknown"
		}
		out = append(out, v)
	}

	writeJSON(w, http.StatusOK, out)
}

type addReq struct {
	Owner string `json:"owner"`
	Repo  string `json:"repo"`
	Notes string `json:"notes"`
}

func (s *Server) addRepo(w http.ResponseWriter, r *http.Request) {
	var req addReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	req.Owner = strings.TrimSpace(req.Owner)
	req.Repo = strings.TrimSpace(req.Repo)
	if req.Owner == "" || req.Repo == "" {
		http.Error(w, "owner and repo required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	key := repoKey(req.Owner, req.Repo)
	for _, r := range s.cfg.Repos {
		if repoKey(r.Owner, r.Repo) == key {
			s.mu.Unlock()
			http.Error(w, "already tracked", http.StatusConflict)
			return
		}
	}
	s.cfg.Repos = append(s.cfg.Repos, config.RepoConfig{Owner: req.Owner, Repo: req.Repo, Notes: req.Notes})
	_ = config.Save(s.cfgPath, s.cfg)
	s.mu.Unlock()

	go s.checkRepo(context.Background(), req.Owner, req.Repo)
	writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
}

func (s *Server) deleteRepo(w http.ResponseWriter, _ *http.Request, owner, repo string) {
	s.mu.Lock()
	key := repoKey(owner, repo)
	out := make([]config.RepoConfig, 0, len(s.cfg.Repos))
	found := false
	for _, r := range s.cfg.Repos {
		if repoKey(r.Owner, r.Repo) == key {
			found = true
			continue
		}
		out = append(out, r)
	}
	if !found {
		s.mu.Unlock()
		http.NotFound(w, nil)
		return
	}
	s.cfg.Repos = out
	delete(s.results, key)
	delete(s.loading, key)
	_ = config.Save(s.cfgPath, s.cfg)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) refreshOne(w http.ResponseWriter, _ *http.Request, owner, repo string) {
	go s.checkRepo(context.Background(), owner, repo)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "refreshing"})
}

func (s *Server) handleRefreshAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	go s.refreshAll(context.Background())
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "refreshing"})
}

// ── Internal ─────────────────────────────────────────────────────────────────

func (s *Server) refreshAll(ctx context.Context) {
	s.mu.RLock()
	repos := make([]config.RepoConfig, len(s.cfg.Repos))
	copy(repos, s.cfg.Repos)
	s.mu.RUnlock()

	for _, r := range repos {
		go s.checkRepo(ctx, r.Owner, r.Repo)
	}
}

func (s *Server) checkRepo(ctx context.Context, owner, repo string) {
	key := repoKey(owner, repo)
	s.mu.Lock()
	s.loading[key] = true
	s.mu.Unlock()

	res := s.gh.CheckRepo(ctx, owner, repo)

	s.mu.Lock()
	delete(s.loading, key)
	s.results[key] = &res
	s.mu.Unlock()
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
