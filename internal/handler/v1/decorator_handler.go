package v1

import (
	"context"
	"strconv"

	"github.com/mysunshines/blog-user/internal/repository"
	decoratorv0pb "github.com/mysunshines/blog-user/proto/decorator/v0/pb"

	"github.com/mysunshines/gocommon/log"

	"google.golang.org/protobuf/types/known/structpb"
)

// DecoratorHandler 实现 decorator.v0.Decorator：把榜单 member（此处为 user ID 字符串）
// 装饰为展示信息（username / nickname / avatar），供 ranking-service 在 GetRanking 时回调。
//
// 关键设计：展示字段的语义由本服务（数据拥有方）定义，ranking-service 仅透传
// google.protobuf.Struct，不感知 username/avatar 等字段，因此新增榜单无需改动它。
type DecoratorHandler struct {
	decoratorv0pb.UnimplementedDecoratorServer
	Repo repository.UserRepository
}

// Decorate 批量将 member 转为展示结构；返回的 map key 必须与入参 member 一致。
func (h *DecoratorHandler) Decorate(ctx context.Context, req *decoratorv0pb.DecorateRequest) (*decoratorv0pb.DecorateResponse, error) {
	ids := make([]uint, 0, len(req.GetMembers()))
	for _, m := range req.GetMembers() {
		id, err := strconv.ParseUint(m, 10, 64)
		if err != nil {
			continue
		}
		ids = append(ids, uint(id))
	}
	if len(ids) == 0 {
		return &decoratorv0pb.DecorateResponse{}, nil
	}

	users, err := h.Repo.GetUsersByIDs(ctx, ids)
	if err != nil {
		log.Warnf("[decorator] get users by ids failed: %v", err)
		return &decoratorv0pb.DecorateResponse{}, nil
	}

	display := make(map[string]*structpb.Struct, len(users))
	for _, u := range users {
		s, err := structpb.NewStruct(map[string]interface{}{
			"user_id":  u.ID,
			"username": u.Username,
			"nickname": u.Nickname,
			"avatar":   u.Avatar,
		})
		if err != nil {
			continue
		}
		display[strconv.FormatUint(uint64(u.ID), 10)] = s
	}
	return &decoratorv0pb.DecorateResponse{Display: display}, nil
}
