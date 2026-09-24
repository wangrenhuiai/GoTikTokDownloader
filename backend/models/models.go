package models

type Author struct {
	ID        string `json:"id"`
	Nickname  string `json:"nickname"`
	UpdatedAt int64  `json:"updatedAt"`
}

type Video struct {
	ID          string  `json:"id"`
	AuthorID    string  `json:"authorId"`
	URL         string  `json:"url"`
	Title       string  `json:"title"`
	Duration    float64 `json:"duration"`
	PlayCount   int64   `json:"playCount"`
	LikeCount   int64   `json:"likeCount"`
	CommentCount int64  `json:"commentCount"`
	ShareCount  int64   `json:"shareCount"`
	PublishedAt int64   `json:"publishedAt"`
	FilePath    string  `json:"filePath"`
	FileSize    int64   `json:"fileSize"`
	Status      string  `json:"status"`
	Error       string  `json:"error"`
	UpdatedAt   int64   `json:"updatedAt"`
}

type DownloadTask struct {
	ID         string  `json:"id"`
	VideoID    string  `json:"videoId"`
	Status     string  `json:"status"`
	Progress   float64 `json:"progress"`
	Speed      float64 `json:"speed"`
	ETA        int64   `json:"eta"`
	RetryCount int     `json:"retryCount"`
	CreatedAt  int64   `json:"createdAt"`
	FinishedAt int64   `json:"finishedAt"`
}

type DownloadAttempt struct {
	ID        int64  `json:"id"`
	TaskID    string `json:"taskId"`
	StartedAt int64  `json:"startedAt"`
	Error     string `json:"error"`
}

const (
	TaskQueued      = "queued"
	TaskDownloading = "downloading"
	TaskPaused      = "paused"
	TaskCompleted   = "completed"
	TaskFailed      = "failed"
	TaskCancelled   = "cancelled"
)
