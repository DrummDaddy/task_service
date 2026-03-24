package models

type TeamStatsRow struct {
	TeamID         uint64 `json:"team_id"`
	TeamName       string `json:"team_name"`
	MembersCount   int    `json:"members_count"`
	DoneTaskLast7d int    `json:"done_task_last7d"`
}

type TopCreatorRow struct {
	TeamID      uint64 `json:"team_id"`
	UserID      uint64 `json:"user_id"`
	TaskCreated int    `json:"task_created"`
	Rank        int    `json:"rank"`
}

type IntegrityTaskRow struct {
	TaskID     uint64 `json:"task_id"`
	TeamID     uint64 `json:"team_id"`
	AssigneeID uint64 `json:"assignee_id"`
}
