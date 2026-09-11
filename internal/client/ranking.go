package client

import (
	"context"
	"fmt"

	v0pb "github.com/mysunshines/blog-ranking/proto/pb/v0"
	pb "github.com/mysunshines/blog-ranking/proto/pb/v1"
	"github.com/mysunshines/gocommon/grpcclient"
)

// BoardAuthorArticles 作者发文数榜单键（member = 用户 ID，装饰器 = user-service）。
// 作者榜的分数由 article-service 推送（统计已发布文章数），本服务只声明展示与跳转配置。
const BoardAuthorArticles = "board:author:articles"

// RegisterAuthorBoard 声明作者发文榜配置（装饰器 = user-service）。幂等，best-effort。
func RegisterAuthorBoard(ctx context.Context) error {
	var resp pb.RegisterBoardResponse
	if err := grpcclient.SendRequest(ctx, v0pb.RankingService_RegisterBoard_FullMethodName, &pb.RegisterBoardRequest{
		Config: &pb.BoardConfig{
			Board:            BoardAuthorArticles,
			DecoratorType:    "remote",
			DecoratorService: "user-service",
			LinkTemplate:     "/space/{member}",
			CacheTtlSec:      60,
		},
	}, &resp); err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("register author board failed: code=%d message=%s", resp.Code, resp.Message)
	}
	return nil
}
