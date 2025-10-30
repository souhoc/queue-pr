package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/google/go-github/v76/github"
)

func listAllOrgRepos(client *github.Client, org string) ([]*github.Repository, error) {
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	var allRepos []*github.Repository
	for {
		repos, resp, err := client.Repositories.ListByOrg(
			context.Background(),
			org,
			opt,
		)
		if err != nil {
			return nil, err
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return allRepos, nil
}

var tagAliases = map[string]string{
	"symphonics:dev": "symphonics:development",
}

type Containers struct {
	Repo *github.Repository
	Pr   *github.PullRequest
	Prr  []*github.PullRequestReview
}

type PullRequest struct {
	Title           string
	Number          int
	Author          string
	ReviewversCount int
	ReviewsCount    int
	Labels          []string
	Age             time.Duration
	Draft           bool
	RepoName        string
}

func listPrsPerBase(repos []*github.Repository, client *github.Client) (map[string][]Containers, error) {
	prsPerBase := make(map[string][]Containers)
	countPr := 0

	for _, repo := range repos {
		prs, _, err := client.PullRequests.List(
			context.Background(),
			repo.GetOwner().GetLogin(),
			repo.GetName(),
			&github.PullRequestListOptions{
				ListOptions: github.ListOptions{
					PerPage: 100,
				},
			},
		)
		if err != nil {
			return nil, err
		}
		if len(prs) == 0 {
			continue
		}

		countPr += len(prs)
		fmt.Printf("* %02d PR: %s\n", len(prs), repo.GetName())

		for _, pr := range prs {
			reviews, _, err := client.PullRequests.ListReviews(
				context.Background(),
				repo.GetOwner().GetLogin(),
				repo.GetName(),
				pr.GetNumber(),
				&github.ListOptions{
					PerPage: 100,
				},
			)
			if err != nil {
				return nil, err
			}
			base := pr.GetBase().GetLabel()
			if _, exists := tagAliases[base]; exists {
				base = tagAliases[base]
			}

			prsPerBase[base] = append(prsPerBase[base], Containers{
				Repo: repo,
				Pr:   pr,
				Prr:  reviews,
			})
		}
	}
	fmt.Printf("TOTAL PR: %d\n", countPr)

	return prsPerBase, nil
}

func run(ghToken, orgName string) {
	client := github.NewClient(nil).WithAuthToken(ghToken)

	user, _, err := client.Users.Get(context.Background(), "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get user.")
		os.Exit(1)
	}
	fmt.Printf("user: %s\n", user.GetLogin())

	fmt.Println("Listing repos...")
	repos, err := listAllOrgRepos(client, orgName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to list repos.")
		os.Exit(1)
	}
	fmt.Printf("%d repos\n", len(repos))

	fmt.Println("Listing PRs per repo...")
	containersPerBase, err := listPrsPerBase(repos, client)
	if err != nil {
		panic(err)
	}
	fmt.Println()

	fmt.Println("## PR per base:")
	for base, containers := range containersPerBase {
		fmt.Printf("### %s: %d\n", base, len(containers))

		slices.SortFunc(containers, func(a, b Containers) int {
			return int(a.Pr.UpdatedAt.Time.Sub(b.Pr.UpdatedAt.Time))
		})

		// Print elements
		for _, container := range containers {
			pr := container.Pr

			since := time.Since(pr.UpdatedAt.Time).Round(time.Hour)
			var sinceStr string
			if since.Hours() > 72 {
				sinceStr = fmt.Sprintf("%3dd", int(since.Hours()/24))
			} else {
				sinceStr = fmt.Sprintf("%3dh", int(since.Hours()))
			}

			user := "Unknown"
			if pr.GetUser() != nil {
				user = pr.GetUser().GetLogin()
			}

			reviewerCount := make(map[string]int)
			for _, prr := range container.Prr {
				reviewerCount[prr.GetUser().GetLogin()]++
			}
			draft := ""
			if pr.GetDraft() {
				draft = "📌"
			}
			labels := make([]string, len(pr.Labels))
			for i, label := range pr.Labels {
				labels[i] = label.GetName()
			}

			// p := PullRequest{
			// 	Title:           pr.GetTitle(),
			// 	Number:          pr.GetNumber(),
			// 	Author:          pr.GetUser().GetLogin(),
			// 	ReviewversCount: len(container.Prr),
			// 	ReviewsCount:    len(reviewerCount),
			// 	Labels:          labels,
			// 	Age:             since,
			// 	Draft:           pr.GetDraft(),
			// }
			fmt.Printf(
				"* ⏳ %s %s`@%s` **%s %d** 💬 %d/👤 %d ➕ %d ➖ %d 📄 %d | %s\n",
				sinceStr,
				draft,
				user,
				container.Repo.GetName(),
				container.Pr.GetNumber(),
				len(container.Prr), len(reviewerCount),
				pr.GetAdditions(), pr.GetDeletions(),
				pr.GetChangedFiles(),
				pr.GetTitle(),
			)
		}
	}
}

var (
	ghToken string
	orgName string

	// format   = "* ⏳ {{.Age}} {{.Draft}} `@{{.Author}}` **{{.RepoName}} {{.Number}}** 💬 {{.ReviewsCount}}/👤 {{.ReviewersCount}} 📄 {{.Title}}\n"
	// prFormat = template.New("pr")
)

func main() {
	// prFormat.Parse(format)
	flag.StringVar(&ghToken, "token", "", "Github token.")
	flag.StringVar(&orgName, "org", "", "Organisation name")
	flag.Parse()

	if ghToken == "" || orgName == "" {
		flag.Usage()
		os.Exit(1)
	}
	run(ghToken, orgName)
}
