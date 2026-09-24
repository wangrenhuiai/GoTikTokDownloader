package tiktok

import "errors"

// SearchService is a Phase 2 reservation. Full auto-search lands in Phase 3+.
type SearchService struct{}

func NewSearchService() *SearchService { return &SearchService{} }

func (s *SearchService) Search(keyword string) error {
	if keyword == "" {
		return errors.New("empty keyword")
	}
	return errors.New("not implemented (Phase 2 reservation)")
}

// VideoCandidate is the future handoff: Browser discover -> Deduplicator -> DownloadTaskFactory.
type VideoCandidate struct {
	URL      string `json:"url"`
	VideoID  string `json:"videoId"`
	AuthorID string `json:"authorId"`
	Keyword  string `json:"keyword"`
}

// VideoCollector is a Phase 2 interface reservation only.
type VideoCollector interface {
	Collect(keyword string) ([]VideoCandidate, error)
}
