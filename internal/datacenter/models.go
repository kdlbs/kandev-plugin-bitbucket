package datacenter

type branchPage struct {
	IsLastPage    bool `json:"isLastPage"`
	NextPageStart int  `json:"nextPageStart"`
	Values        []struct {
		DisplayID    string `json:"displayId"`
		LatestCommit string `json:"latestCommit"`
		IsDefault    bool   `json:"isDefault"`
	} `json:"values"`
}

type pullRequestPage struct {
	IsLastPage    bool                 `json:"isLastPage"`
	NextPageStart int                  `json:"nextPageStart"`
	Values        []pullRequestPayload `json:"values"`
}

type pullRequestPayload struct {
	ID          int    `json:"id"`
	Version     int    `json:"version"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	CreatedDate int64  `json:"createdDate"`
	Author      struct {
		Slug        string `json:"slug"`
		DisplayName string `json:"displayName"`
	} `json:"author"`
	Links struct {
		Self []struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
	FromRef struct {
		DisplayID    string                         `json:"displayId"`
		LatestCommit string                         `json:"latestCommit"`
		Repository   *dataCenterRepositoryReference `json:"repository"`
	} `json:"fromRef"`
	ToRef struct {
		DisplayID    string `json:"displayId"`
		LatestCommit string `json:"latestCommit"`
		Repository   struct {
			DefaultBranch string `json:"defaultBranch"`
		} `json:"repository"`
	} `json:"toRef"`
	Participants []dataCenterParticipantPayload `json:"participants"`
}

type dataCenterRepositoryReference struct {
	ID      int    `json:"id"`
	Slug    string `json:"slug"`
	Project struct {
		Key string `json:"key"`
	} `json:"project"`
}

type dataCenterParticipantPayload struct {
	User struct {
		Slug        string `json:"slug"`
		DisplayName string `json:"displayName"`
	} `json:"user"`
	Role   string `json:"role"`
	Status string `json:"status"`
}

type reviewCommitPage struct {
	IsLastPage    bool `json:"isLastPage"`
	NextPageStart int  `json:"nextPageStart"`
	Values        []struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		Author  struct {
			Name string `json:"name"`
		} `json:"author"`
	} `json:"values"`
}

type reviewActivityPage struct {
	IsLastPage    bool             `json:"isLastPage"`
	NextPageStart int              `json:"nextPageStart"`
	Values        []reviewActivity `json:"values"`
}

type reviewActivity struct {
	Action  string         `json:"action"`
	Comment *reviewComment `json:"comment"`
}

type reviewComment struct {
	ID          int   `json:"id"`
	CreatedDate int64 `json:"createdDate"`
	Parent      struct {
		ID int `json:"id"`
	} `json:"parent"`
	Text   string `json:"text"`
	Author struct {
		DisplayName string `json:"displayName"`
	} `json:"author"`
}

type reviewStatusPage struct {
	IsLastPage    bool `json:"isLastPage"`
	NextPageStart int  `json:"nextPageStart"`
	Values        []struct {
		Key   string `json:"key"`
		Name  string `json:"name"`
		State string `json:"state"`
		URL   string `json:"url"`
	} `json:"values"`
}

type reviewChangesPage struct {
	IsLastPage    bool           `json:"isLastPage"`
	NextPageStart int            `json:"nextPageStart"`
	Values        []reviewChange `json:"values"`
}

type reviewChange struct {
	Type    string       `json:"type"`
	Path    reviewPath   `json:"path"`
	SrcPath reviewPath   `json:"srcPath"`
	Hunks   []reviewHunk `json:"hunks"`
}

type reviewPath struct {
	String string `json:"toString"`
}

type reviewHunk struct {
	SourceLine      int             `json:"sourceLine"`
	SourceSpan      int             `json:"sourceSpan"`
	DestinationLine int             `json:"destinationLine"`
	DestinationSpan int             `json:"destinationSpan"`
	Segments        []reviewSegment `json:"segments"`
}

type reviewSegment struct {
	Type  string           `json:"type"`
	Lines []reviewDiffLine `json:"lines"`
}

type reviewDiffLine struct {
	Line string `json:"line"`
}

const (
	maxReviewComments = 200
	maxReviewEntries  = 200
	maxReviewPages    = 10
)

const (
	maxBranchPages   = 200
	maxBranchEntries = maxBranchPages * maxPageLength
)
