package cloud

import "time"

type branchPage struct {
	Values []struct {
		Name   string `json:"name"`
		Target struct {
			Hash string `json:"hash"`
		} `json:"target"`
	} `json:"values"`
	Next string `json:"next"`
}

const (
	maxBranchPages   = 200
	maxBranchEntries = maxBranchPages * maxPageLength
)

type pullRequestPage struct {
	Values []pullRequestPayload `json:"values"`
	Next   string               `json:"next"`
}

type pullRequestPayload struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	State       string    `json:"state"`
	CreatedAt   time.Time `json:"created_on"`
	Author      struct {
		AccountID   string `json:"account_id"`
		DisplayName string `json:"display_name"`
	} `json:"author"`
	Links struct {
		HTML struct {
			Href string `json:"href"`
		} `json:"html"`
	} `json:"links"`
	Source struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Commit struct {
			Hash string `json:"hash"`
		} `json:"commit"`
		Repository *cloudRepositoryReference `json:"repository"`
	} `json:"source"`
	Destination struct {
		Branch struct {
			Name string `json:"name"`
		} `json:"branch"`
		Commit struct {
			Hash string `json:"hash"`
		} `json:"commit"`
		Repository struct {
			MainBranch struct {
				Name string `json:"name"`
			} `json:"mainbranch"`
		} `json:"repository"`
	} `json:"destination"`
	Participants []cloudParticipantPayload `json:"participants"`
}

type cloudRepositoryReference struct {
	UUID      string `json:"uuid"`
	Slug      string `json:"slug"`
	FullName  string `json:"full_name"`
	Workspace struct {
		Slug string `json:"slug"`
	} `json:"workspace"`
}

type cloudParticipantPayload struct {
	User struct {
		AccountID   string `json:"account_id"`
		DisplayName string `json:"display_name"`
	} `json:"user"`
	Role     string `json:"role"`
	Approved bool   `json:"approved"`
	State    string `json:"state"`
}

type reviewCommitPage struct {
	Values []struct {
		Hash    string `json:"hash"`
		Message string `json:"message"`
		Author  struct {
			Raw string `json:"raw"`
		} `json:"author"`
	} `json:"values"`
	Next string `json:"next"`
}

type reviewCommentPage struct {
	Values []struct {
		ID        int       `json:"id"`
		CreatedOn time.Time `json:"created_on"`
		Parent    struct {
			ID int `json:"id"`
		} `json:"parent"`
		Content struct {
			Raw string `json:"raw"`
		} `json:"content"`
		User struct {
			DisplayName string `json:"display_name"`
		} `json:"user"`
	} `json:"values"`
	Next string `json:"next"`
}

type reviewStatusPage struct {
	Values []struct {
		Key   string `json:"key"`
		Name  string `json:"name"`
		State string `json:"state"`
		URL   string `json:"url"`
	} `json:"values"`
	Next string `json:"next"`
}

const (
	maxReviewFiles   = 200
	maxReviewEntries = 200
	maxReviewPages   = 10
)

type reviewDiffstatPage struct {
	Values []reviewDiffstat `json:"values"`
	Next   string           `json:"next"`
}

type reviewDiffstat struct {
	Status       string `json:"status"`
	LinesAdded   int    `json:"lines_added"`
	LinesRemoved int    `json:"lines_removed"`
	Old          struct {
		Path string `json:"path"`
	} `json:"old"`
	New struct {
		Path string `json:"path"`
	} `json:"new"`
}
