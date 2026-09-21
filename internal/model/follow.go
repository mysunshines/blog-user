package model

import "time"

// UserFollow 关注关系：follower_id 关注 following_id（不能关注自己）。
// 与 user_blacklists 同处用户库，不使用数据库外键，关联约束在代码层维护。
type UserFollow struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	FollowerID  uint      `gorm:"uniqueIndex:uk_follow;not null" json:"follower_id"`  // 关注者
	FollowingID uint      `gorm:"uniqueIndex:uk_follow;not null" json:"following_id"` // 被关注者
	CreatedAt   time.Time `json:"created_at"`
}

func (UserFollow) TableName() string {
	return "user_follows"
}
