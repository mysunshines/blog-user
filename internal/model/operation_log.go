package model

import "time"

// OperationLog 操作日志：记录"谁在什么时间对什么对象做了什么操作"。
// 归属 user-service，作为所有业务操作日志的统一落库方。
type OperationLog struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	OperatorID  uint      `gorm:"index" json:"operator_id"`
	Operator    string    `gorm:"size:64" json:"operator"`
	Action      string    `gorm:"size:64;index" json:"action"`
	TargetType  string    `gorm:"size:64;index" json:"target_type"`
	TargetID    uint      `gorm:"index" json:"target_id"`
	TargetTitle string    `gorm:"size:255" json:"target_title"`
	Detail      string    `gorm:"type:text" json:"detail"`
	IP          string    `gorm:"size:64" json:"ip"`
	CreatedAt   time.Time `json:"created_at"`
}

func (OperationLog) TableName() string {
	return "operation_logs"
}
