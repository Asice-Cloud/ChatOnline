package middleware

import (
	"Chat/config"
	"Chat/response"
	"Chat/service/validator"
	"fmt"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
	"os"
	"sync"
)

var (
	// blocked IP:
	// 1. use redis to store the IP
	// 2. use the IP as the key, and the visit frequency as the value
	// 3. if the visit frequency is too high, block the IP
	// 4. if the IP is blocked, return 429 status code
	// 5. if the IP is not blocked, continue to the next middleware
	BlockIP = make(map[int]string)
	mu      sync.Mutex
)

// check the visit frequency, if it is too frequent, blocking the IP
func LimitCount(context *gin.Context) (err string) {
	ip := context.ClientIP()
	limiter := rate.NewLimiter(200, 1)
	if !limiter.Allow() {
		// add this ip into blocked ip
		mu.Lock()
		BlockIP[len(BlockIP)] = ip
		validator.AddBlockIP(BlockIP)
		mu.Unlock()

		return response.CustomError{-1, "Too many requests"}.Error()
	}
	return ""
}

func BlockIPMiddleware(context *gin.Context) {
	ip := context.ClientIP()
	checkResponse := LimitCount(context)
	if checkResponse != "" {
		context.JSON(429, checkResponse)
		context.Abort()
		return
	}
	// Check if the IP is blocked
	val, err := config.Rdb.Get(context, ip).Result()
	if err != nil {
		context.JSON(503, response.CustomError{-1, "Service Unavailable"}.Error())
		context.Abort()
		return
	}
	if val == "blocked" {
		context.JSON(403, response.CustomError{-1, "Forbidden"}.Error())
		context.Abort()
		return
	}
	context.Next()
}

// print blocked ip into a new txt file:
// 1. create a new txt file
// 2. write the blocked ip into the txt file
func PrintBlockedIP() {
	// create a new txt file
	file, err := os.Create("blockedIP.txt")
	if err != nil {
		fmt.Println("Create file failed!")
		return
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			fmt.Println("Close file failed!")
			return
		}
	}(file)

	// write the blocked ip into the txt file
	for _, ip := range BlockIP {
		_, err := file.WriteString(ip + "\n")
		if err != nil {
			return
		}
	}
}
