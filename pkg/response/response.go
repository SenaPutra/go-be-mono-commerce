package response

import "github.com/gin-gonic/gin"

type ErrorBody struct {
	Code    string      `json:"code"`
	Details interface{} `json:"details"`
}

func JSON(c *gin.Context, code int, success bool, msg string, data interface{}, errBody *ErrorBody) {
	c.JSON(code, gin.H{"success": success, "message": msg, "data": data, "error": errBody})
}

func OK(c *gin.Context, data interface{}) { JSON(c, 200, true, "OK", data, nil) }
func Created(c *gin.Context, data interface{}) { JSON(c, 201, true, "Created", data, nil) }
func Fail(c *gin.Context, code int, msg, errCode string, details interface{}) {
	JSON(c, code, false, msg, nil, &ErrorBody{Code: errCode, Details: details})
}
