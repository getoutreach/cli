package command

import (
	"fmt"
	"os"
	"sort"
	"time"

	markdown "github.com/MichaelMure/go-term-markdown"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/cli/cli/api"
	"github.com/cli/cli/utils"
	"github.com/dustin/go-humanize"
)

// comment is a "generic" for pull request comments and issue comments
type comment struct {
	URL       string    `json:"url"`
	HTMLURL   string    `json:"html_url"`
	Body      string    `json:"body"`
	User      api.User  `json:"user"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func init() {
	prCmd.AddCommand(prCommentsCmd)
}

func prComments(cmd *cobra.Command, args []string) error {
	ctx := contextForCommand(cmd)
	apiClient, err := apiClientForContext(ctx)
	if err != nil {
		return err
	}

	baseRepo, err := determineBaseRepo(apiClient, cmd, ctx)
	if err != nil {
		return err
	}

	currentPRNumber, currentPRHeadRef, err := prSelectorForCurrentBranch(ctx, baseRepo)
	if err != nil && err.Error() != "git: not on any branch" {
		return fmt.Errorf("could not query for pull request for current branch: %w", err)
	}

	// the `@me` macro is available because the API lookup is ElasticSearch-based
	currentUser := "@me"
	prPayload, err := api.PullRequests(apiClient, baseRepo, currentPRNumber, currentPRHeadRef, currentUser)
	if err != nil {
		return err
	}

	prComments, err := api.PullRequestComments(apiClient, baseRepo, prPayload.CurrentPR.Number)
	if err != nil {
		return fmt.Errorf("failed to get PR review comments: %v", err)
	}

	issueComments, err := api.IssueComments(apiClient, baseRepo, prPayload.CurrentPR.Number)
	if err != nil {
		return fmt.Errorf("failed to get issues comments: %v", err)
	}

	comments := make([]comment, 0)

	for _, c := range prComments {
		comments = append(comments, comment{
			User:      *c.User,
			URL:       c.HTMLURL,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
			Body:      c.Body,
		})
	}

	for _, c := range issueComments {
		comments = append(comments, comment{
			User:      *c.User,
			URL:       c.HTMLURL,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
			Body:      c.Body,
		})
	}

	sort.SliceStable(comments, func(i, j int) bool {
		return comments[i].CreatedAt.Before(comments[j].CreatedAt)
	})

	w, _, err := terminal.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		w = 80
	}

	fmt.Printf("Pull Request #%d Comments\n\n", prPayload.CurrentPR.Number)

	if len(comments) == 0 {
		fmt.Printf("No Comments.\n\n")
	}

	for _, c := range comments {
		// TODO(jaredallard): make this a go-template jeez
		out := fmt.Sprintf(
			"@%s %s %s\n%s\n",
			utils.Bold(c.User.Login),
			utils.Gray(humanize.Time(c.CreatedAt)),
			utils.Blue(c.URL),
			markdown.Render(c.Body, w, 0),
		)
		fmt.Println(out)
	}

	fmt.Printf("Pull Request URL: %s\n", utils.Blue(prPayload.CurrentPR.URL))

	return nil
}

var prCommentsCmd = &cobra.Command{
	Use:   "comments {<number> | <url> | <branch>}",
	Short: "Return the latest comments on a Github PR",
	RunE:  prComments,
}
