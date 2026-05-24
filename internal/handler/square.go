// Package handler - "一起玩"广场气泡发布、发现、搜索与确认匹配。
package handler

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"zaima-backend/internal/model"
	"zaima-backend/internal/pkg/database"
	"zaima-backend/internal/pkg/response"
)

// ==================== 请求体定义 ====================

// PublishBubbleReq 发布气泡请求。
type PublishBubbleReq struct {
	VoiceURL    string  `json:"voice_url" binding:"required"` // OSS 录音文件 URL
	InterestTag string  `json:"interest_tag" binding:"required"`
	Province    string  `json:"province"`
	City        string  `json:"city"`
	Longitude   float64 `json:"longitude" binding:"required"`
	Latitude    float64 `json:"latitude" binding:"required"`
}

// MatchConfirmReq 确认匹配请求。
type MatchConfirmReq struct {
	BubbleID uint64 `json:"bubble_id" binding:"required"`
}

// ==================== 响应体定义 ====================

// SquareUserInfo 广场用户信息（带兴趣标签）。
type SquareUserInfo struct {
	UserID    uint64   `json:"user_id"`
	Nickname  string   `json:"nickname"`
	AvatarURL string   `json:"avatar_url"`
	City      string   `json:"city"`
	Province  string   `json:"province"`
	Interests []string `json:"interests"` // 兴趣标签列表
}

// ==================== Handler ====================

// PublishBubble 发布广场气泡。
// POST /api/v1/square/publish
//
// 流程：保存气泡到 DB -> 将经纬度写入 Redis GEO -> 设置4小时 TTL。
func PublishBubble(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req PublishBubbleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数不完整")
		return
	}

	// 查询用户信息 (昵称、头像)
	var user model.User
	database.DB.First(&user, userID)

	// 【安全】URL 白名单校验，防止 SSRF
	if !isValidOSSURL(req.VoiceURL) {
		response.BadRequest(c, "语音 URL 不合法，仅允许 OSS 地址")
		return
	}

	bubble := model.SquareBubble{
		UserID:      userID,
		Nickname:    user.Nickname,
		AvatarURL:   user.AvatarURL,
		VoiceURL:    req.VoiceURL,
		InterestTag: req.InterestTag,
		Province:    req.Province,
		City:        req.City,
		Longitude:   req.Longitude,
		Latitude:    req.Latitude,
		Status:      1, // 活跃
		ExpireAt:    time.Now().Add(4 * time.Hour),
	}

	if err := database.DB.Create(&bubble).Error; err != nil {
		response.ServerError(c, "发布失败，请稍后重试")
		return
	}

	// 写入 Redis GEO 用于地理位置范围查询
	ctx := context.Background()
	geoKey := "square:geo"
	memberKey := fmt.Sprintf("bubble:%d", bubble.ID)

	database.RDB.GeoAdd(ctx, geoKey, &redis.GeoLocation{
		Name:      memberKey,
		Longitude: req.Longitude,
		Latitude:  req.Latitude,
	})
	// 设置4小时后自动过期 (使用单独的 key 标记 TTL)
	database.RDB.Set(ctx, fmt.Sprintf("square:ttl:%d", bubble.ID), "1", 4*time.Hour)

	response.OKWithMsg(c, "发布成功", gin.H{"bubble_id": bubble.ID})
}

// GetBubbles 获取广场气泡列表。
// GET /api/v1/square/bubbles?lng=xxx&lat=xxx&radius=5000&province=xxx&city=xxx&keyword=xxx
//
// 支持：GEO 半径查询 + 省份/城市筛选 + 举办人昵称模糊搜索。
func GetBubbles(c *gin.Context) {
	lngStr := c.DefaultQuery("lng", "0")
	latStr := c.DefaultQuery("lat", "0")
	radiusStr := c.DefaultQuery("radius", "10000") // 默认10km
	province := c.Query("province")
	city := c.Query("city")
	keyword := c.Query("keyword")

	lng, _ := strconv.ParseFloat(lngStr, 64)
	lat, _ := strconv.ParseFloat(latStr, 64)
	radius, _ := strconv.ParseFloat(radiusStr, 64)

	ctx := context.Background()
	var bubbleIDs []uint64

	// 1. 优先使用 GEO 查询附近气泡
	if lng != 0 && lat != 0 {
		results, err := database.RDB.GeoRadius(ctx, "square:geo", lng, lat, &redis.GeoRadiusQuery{
			Radius: radius,
			Unit:   "m",
			Sort:   "ASC",
			Count:  50,
		}).Result()
		if err == nil {
			for _, loc := range results {
				var id uint64
				_, _ = fmt.Sscanf(loc.Name, "bubble:%d", &id)
				if id > 0 {
					// 检查是否已过期
					exists, _ := database.RDB.Exists(ctx, fmt.Sprintf("square:ttl:%d", id)).Result()
					if exists > 0 {
						bubbleIDs = append(bubbleIDs, id)
					}
				}
			}
		}
	}

	// 2. 构建数据库查询
	query := database.DB.Model(&model.SquareBubble{}).Where("status = 1 AND expire_at > ?", time.Now())

	// 【修复 GEO 击穿】如果传入了 lng/lat 但附近没有气泡，应直接返回空列表
	geoSearched := lng != 0 && lat != 0
	if geoSearched && len(bubbleIDs) == 0 {
		// 附近无人，直接返回空
		response.OK(c, gin.H{"total": 0, "bubbles": []interface{}{}})
		return
	}
	if len(bubbleIDs) > 0 {
		query = query.Where("id IN ?", bubbleIDs)
	}

	// 省份/城市筛选
	if province != "" {
		query = query.Where("province = ?", province)
	}
	if city != "" {
		query = query.Where("city = ?", city)
	}

	// 举办人昵称模糊搜索
	if keyword != "" {
		query = query.Where("nickname LIKE ?", "%"+keyword+"%")
	}

	var bubbles []model.SquareBubble
	query.Order("created_at DESC").Limit(20).Find(&bubbles)

	response.OK(c, gin.H{
		"total":   len(bubbles),
		"bubbles": bubbles,
	})
}

// MatchConfirm 确认匹配 (发布者找到玩伴)。
// POST /api/v1/square/match-confirm
//
// 流程：标记气泡为已消失 -> 删除 Redis GEO 数据 -> 通知其他等待者聊天结束。
func MatchConfirm(c *gin.Context) {
	userID := c.GetUint64("user_id")

	var req MatchConfirmReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "缺少 bubble_id")
		return
	}

	var bubble model.SquareBubble
	if err := database.DB.First(&bubble, req.BubbleID).Error; err != nil {
		response.Fail(c, 2001, "气泡不存在")
		return
	}

	// 只有发布者能确认
	if bubble.UserID != userID {
		response.Fail(c, 2002, "无权操作此气泡")
		return
	}

	// 标记为已消失
	database.DB.Model(&bubble).Update("status", 0)

	// 清除 Redis GEO 和 TTL 数据
	ctx := context.Background()
	memberKey := fmt.Sprintf("bubble:%d", bubble.ID)
	database.RDB.ZRem(ctx, "square:geo", memberKey)
	database.RDB.Del(ctx, fmt.Sprintf("square:ttl:%d", bubble.ID))

	// TODO: 通过 WebSocket 通知其他正在和发布者聊天的用户：
	// 发送 { type: "match_ended", bubble_id: xxx, message: "对方已找到玩伴，聊天结束" }

	response.OKWithMsg(c, "匹配成功，气泡已消失", nil)
}

// GetSquareUsers 获取广场用户列表（带兴趣标签）。
// GET /api/v1/square/users?keyword=xxx&interest=xxx&page=1&page_size=20
//
// 返回所有有活跃气泡或满足过滤条件的用户，包含头像和兴趣标签。
func GetSquareUsers(c *gin.Context) {
	keyword := c.Query("keyword")           // 用户昵称搜索
	interestTag := c.Query("interest")      // 兴趣标签过滤
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if page < 1 {
		page = 1
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	// 1. 查询所有有活跃气泡的用户
	type userBubbleInfo struct {
		UserID uint64
	}
	var userIDs []userBubbleInfo
	query := database.DB.Model(&model.SquareBubble{}).
		Select("DISTINCT user_id").
		Where("status = 1 AND expire_at > ?", time.Now())

	if interestTag != "" {
		query = query.Where("interest_tag = ?", interestTag)
	}

	query.Scan(&userIDs)

	if len(userIDs) == 0 {
		response.OK(c, gin.H{
			"total": 0,
			"users": []interface{}{},
		})
		return
	}

	// 提取用户ID列表
	uidList := make([]uint64, len(userIDs))
	for i, info := range userIDs {
		uidList[i] = info.UserID
	}

	// 2. 批量查询用户信息
	var users []model.User
	userQuery := database.DB.Where("id IN ?", uidList)
	if keyword != "" {
		userQuery = userQuery.Where("nickname LIKE ?", "%"+keyword+"%")
	}
	userQuery.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&users)

	// 3. 为每个用户查询兴趣标签
	result := make([]SquareUserInfo, 0, len(users))
	for _, user := range users {
		var interests []model.UserInterest
		database.DB.Where("user_id = ?", user.ID).Find(&interests)
		tags := make([]string, len(interests))
		for i, item := range interests {
			tags[i] = item.InterestTag
		}

		result = append(result, SquareUserInfo{
			UserID:    user.ID,
			Nickname:  user.Nickname,
			AvatarURL: user.AvatarURL,
			City:      user.City,
			Province:  user.Province,
			Interests: tags,
		})
	}

	// 计算总数
	var total int64
	database.DB.Model(&model.SquareBubble{}).
		Select("COUNT(DISTINCT user_id)").
		Where("status = 1 AND expire_at > ?", time.Now()).
		Scan(&total)

	response.OK(c, gin.H{
		"total": total,
		"page":  page,
		"users": result,
	})
}
