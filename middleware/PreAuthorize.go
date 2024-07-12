package middleware

import (
	"context"
	"encoding/json"
	"github.com/DeteMin/library/cache"
	"github.com/DeteMin/library/constants"
	"github.com/gin-gonic/gin"
	"net/http"
)

type LoginUser struct {
	UserId      int64    `json:"userId"`
	Permissions []string `json:"permissions"`
}

func AuthCheck(permission string, redisCache *cache.Cache) gin.HandlerFunc {
	return func(c *gin.Context) {
		userId := c.GetHeader(constants.HEADER_USER_ID)
		//根据userId去redis中获取用户的权限
		key := constants.LOGIN_USER_KEY + ":" + userId
		loginUser := &LoginUser{}
		str, err := redisCache.Client.Get(context.Background(), key).Result()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusOK, gin.H{
				"code": "40001",
				"msg":  "用户Token已过期",
			})
			return
		}
		// 反转义 JSON 字符串
		var jsonStr string
		err = json.Unmarshal([]byte(str), &jsonStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusOK, gin.H{
				"code": "500",
				"msg":  "用户Token反转义失败",
			})
			return
		}

		// 解析 JSON 数据
		var userInfo LoginUser
		err = json.Unmarshal([]byte(jsonStr), &userInfo)
		if err != nil || userInfo.Permissions == nil {
			c.AbortWithStatusJSON(http.StatusOK, gin.H{
				"code": "500",
				"msg":  "用户Token JSON解析失败",
			})
			return
		}

		if contains(loginUser.Permissions, constants.ALL_PERMISSIONS) {
			//有所有权限，放行
			c.Next()
		} else if contains(loginUser.Permissions, permission) {
			//有配置的权限，放行
			c.Next()
		} else {
			c.AbortWithStatusJSON(http.StatusOK, gin.H{
				"code": "40001",
				"msg":  "用户无对应权限",
			})
			return
		}
	}
}

func contains(arr []string, str string) bool {
	for _, v := range arr {
		if v == str {
			return true
		}
	}
	return false
}
