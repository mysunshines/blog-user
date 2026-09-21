package client

import (
	"context"
	"fmt"
	"strconv"

	v0pb "github.com/mysunshines/blog-ranking/proto/pb/v0"
	pb "github.com/mysunshines/blog-ranking/proto/pb/v1"
	"github.com/mysunshines/gocommon/grpcclient"
)

// BoardAuthorArticles 作者发文数榜单键（member = 用户 ID，装饰器 = user-service）。
// 作者榜的分数由 article-service 推送（统计已发布文章数），本服务只声明展示与跳转配置。
const BoardAuthorArticles = "board:author:articles"

// BoardUserFans 用户粉丝数榜单键（member = 用户 ID，装饰器 = user-service）。
// 分数由本服务在关注/取关时增量推送（见 Follow/Unfollow 业务），此处只声明展示与跳转配置。
const BoardUserFans = "board:user:fans"

// RegisterAuthorBoard 声明作者发文榜配置（装饰器 = user-service）。幂等，best-effort。
func RegisterAuthorBoard(ctx context.Context) error {
	var resp pb.RegisterBoardResponse
	if err := grpcclient.SendRequest(ctx, v0pb.RankingService_RegisterBoard_FullMethodName, &pb.RegisterBoardRequest{
		Config: &pb.BoardConfig{
			Board:            BoardAuthorArticles,
			DecoratorType:    "remote",
			DecoratorService: "user-service",
			LinkTemplate:     "/author?id={member}",
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

// RegisterUserFansBoard 声明用户粉丝榜配置（装饰器 = user-service）。幂等，best-effort。
func RegisterUserFansBoard(ctx context.Context) error {
	var resp pb.RegisterBoardResponse
	if err := grpcclient.SendRequest(ctx, v0pb.RankingService_RegisterBoard_FullMethodName, &pb.RegisterBoardRequest{
		Config: &pb.BoardConfig{
			Board:            BoardUserFans,
			DecoratorType:    "remote",
			DecoratorService: "user-service",
			LinkTemplate:     "/author?id={member}",
			CacheTtlSec:      60,
		},
	}, &resp); err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("register user fans board failed: code=%d message=%s", resp.Code, resp.Message)
	}
	return nil
}

// RecordFansScore 增量推送粉丝榜分数（关注 +1 / 取关 -1）。幂等 best-effort，
// 失败仅记录日志，不影响关注/取关主流程。
func RecordFansScore(ctx context.Context, member uint, delta int64) error {
	var resp pb.RecordScoreResponse
	if err := grpcclient.SendRequest(ctx, v0pb.RankingService_RecordScore_FullMethodName, &pb.RecordScoreRequest{
		Board:  BoardUserFans,
		Member: strconv.FormatUint(uint64(member), 10),
		Delta:  float64(delta),
		Op:     pb.ScoreOp_SCORE_OP_INCREMENT,
	}, &resp); err != nil {
		return err
	}
	if resp.Code != 0 {
		return fmt.Errorf("record fans score failed: code=%d message=%s", resp.Code, resp.Message)
	}
	return nil
}
