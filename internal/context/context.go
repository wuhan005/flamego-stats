package context

import (
	"encoding/json"
	"net/http"

	"github.com/flamego/flamego"
	"github.com/sirupsen/logrus"
)

// Context represents context of a request.
type Context struct {
	flamego.Context
}

func (c *Context) ApiSuccess(data interface{}) error {
	return c.writeJSON(http.StatusOK, map[string]interface{}{
		"msg":  "success",
		"data": data,
	})
}

func (c *Context) ApiError(statusCode int, message string) error {
	return c.writeJSON(statusCode, map[string]interface{}{
		"msg": message,
	})
}

func (c *Context) ApiServerError() error {
	return c.ApiError(http.StatusInternalServerError, "Internal server error")
}

func (c *Context) writeJSON(statusCode int, body interface{}) error {
	c.ResponseWriter().Header().Set("Content-Type", "application/json; charset=utf-8")
	c.ResponseWriter().WriteHeader(statusCode)
	if err := json.NewEncoder(c.ResponseWriter()).Encode(body); err != nil {
		logrus.WithContext(c.Request().Context()).WithError(err).Error("Failed to encode JSON")
	}
	return nil
}

// Contexter initializes a classic context for a request.
func Contexter() flamego.Handler {
	return func(ctx flamego.Context) {
		c := Context{
			Context: ctx,
		}

		c.ResponseWriter().Header().Set("Server", "Flamego-Stats")
		c.ResponseWriter().Header().Set("Access-Control-Allow-Origin", "*")
		c.ResponseWriter().Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		c.ResponseWriter().Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if ctx.Request().Method == http.MethodOptions {
			ctx.ResponseWriter().WriteHeader(http.StatusOK)
			return
		}

		c.Map(c)
	}
}
