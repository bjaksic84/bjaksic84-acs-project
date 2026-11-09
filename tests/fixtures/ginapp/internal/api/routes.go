// Package api serves a small API with gin. Its dependencies are not in
// go.mod: scan must find the routes from syntax.
package api

import "github.com/gin-gonic/gin"

// Router returns the API's routes.
func Router() *gin.Engine {
	r := gin.New()
	r.GET("/ping", ping)
	api := r.Group("/api")
	api.GET("/items", listItems)
	api.POST("/items", createItem)
	admin := api.Group("/admin")
	admin.DELETE("/items/:id", deleteItem)
	return r
}

func ping(c *gin.Context)       { c.String(200, "pong") }
func listItems(c *gin.Context)  { c.String(200, "items") }
func createItem(c *gin.Context) { c.String(201, "created") }
func deleteItem(c *gin.Context) { c.String(200, "deleted") }
