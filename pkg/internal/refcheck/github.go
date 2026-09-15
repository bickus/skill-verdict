// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package refcheck

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bickus/skill-verdict/pkg/config"
	"github.com/bickus/skill-verdict/pkg/internal/inventory"
	"github.com/bickus/skill-verdict/pkg/refs"
)

const (
	githubAPI = "https://api.github.com/"
)

type GitHubCollector struct {
	fetch fetcher
	token string
	cfg   config.ReferencesConfig
}

func NewGitHub(client *http.Client, cfg config.ReferencesConfig, token string) *GitHubCollector {
	return &GitHubCollector{fetch: newFetcher(client, cfg), token: token, cfg: cfg}
}

func (*GitHubCollector) Kind() refs.Kind {
	return refs.GitHub
}

func (*GitHubCollector) Extract(a *inventory.Artifact) []refs.Ref {
	return githubRefs(a)
}

type githubUser struct {
	Login       string    `json:"login"`
	Type        string    `json:"type"`
	CreatedAt   time.Time `json:"created_at"`
	PublicRepos int       `json:"public_repos"`
}

type githubRepo struct {
	Owner     githubUser `json:"owner"`
	Stars     int        `json:"stargazers_count"`
	Fork      bool       `json:"fork"`
	Archived  bool       `json:"archived"`
	CreatedAt time.Time  `json:"created_at"`
	PushedAt  time.Time  `json:"pushed_at"`
}

func (g *GitHubCollector) get(ctx context.Context, path string, out any) (bool, error) {
	headers := map[string]string{"Accept": "application/vnd.github+json"}
	if g.token != "" {
		headers["Authorization"] = "Bearer " + g.token
	}
	return g.fetch.getJSON(ctx, githubAPI+path, headers, out)
}

func (g *GitHubCollector) Collect(ctx context.Context, ref refs.Ref) refs.Report {
	rep := refs.Report{Ref: ref, Status: refs.Checked}
	owner, repo, _ := strings.Cut(ref.Value, "/")
	var user githubUser
	found, err := g.get(ctx, "users/"+owner, &user)
	if err != nil {
		fail(&rep, err)
		return rep
	}
	if !found {
		addFact(&rep, "owner", "missing")
		addFinding(&rep, "GITHUB-OWNER-MISSING")
		return rep
	}
	ownerFacts(&rep, user, g.cfg.YoungOwnerDays)
	if repo == "" {
		return rep
	}
	var r githubRepo
	found, err = g.get(ctx, "repos/"+owner+"/"+repo, &r)
	if err != nil {
		fail(&rep, err)
		return rep
	}
	if !found {
		addFact(&rep, "repo", "missing")
		addFinding(&rep, "GITHUB-REPO-MISSING")
		return rep
	}
	repoFacts(&rep, owner, r)
	return rep
}

func ownerFacts(rep *refs.Report, user githubUser, youngDays int) {
	addFact(rep, "owner_type", user.Type)
	addFact(rep, "owner_repos", strconv.Itoa(user.PublicRepos))
	if user.CreatedAt.IsZero() {
		return
	}
	age := ageDays(user.CreatedAt)
	addFact(rep, "owner_created", user.CreatedAt.Format(time.DateOnly))
	addFact(rep, "owner_age_days", strconv.Itoa(age))
	if age < youngDays {
		addFinding(rep, "GITHUB-OWNER-NEW")
	}
}

func repoFacts(rep *refs.Report, owner string, r githubRepo) {
	addFact(rep, "stars", strconv.Itoa(r.Stars))
	addFact(rep, "fork", strconv.FormatBool(r.Fork))
	addFact(rep, "archived", strconv.FormatBool(r.Archived))
	if !r.CreatedAt.IsZero() {
		addFact(rep, "repo_created", r.CreatedAt.Format(time.DateOnly))
	}
	if !r.PushedAt.IsZero() {
		addFact(rep, "pushed", r.PushedAt.Format(time.DateOnly))
	}
	if r.Owner.Login != "" && !strings.EqualFold(r.Owner.Login, owner) {
		addFact(rep, "owner_login", r.Owner.Login)
		addFinding(rep, "GITHUB-OWNER-RENAMED")
	}
}
