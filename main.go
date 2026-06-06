// Minimal Gin + Arcis app. One install, one middleware line, the full
// Arcis sanitizer pipeline (XSS, SQL, NoSQL, path, command, SSTI, XXE,
// prototype, LDAP, XPath, header injection) + rate limiting + security
// headers gated against your handler when Block:true. Run with
// `go run .`, then fire `go run attack.go` in another shell to see
// Arcis at work. See README for the full "does / does not do" table;
// bot / CSRF / CORS / cookies / validation / error-scrub are
// deliberate opt-ins.

package main

import (
	"log"

	arcisgin "github.com/getarcis/arcis-go/gin"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// Block:true returns 403 on detected attacks. The default is sanitize
	// (silently strip + observe), which is safer to roll out without
	// breaking existing clients. We use block here so the demo is visible.
	r.Use(arcisgin.MiddlewareWithConfig(arcisgin.Config{Block: true}))

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"ok":      true,
			"message": "Arcis is live. Try /api/echo with an attack payload.",
		})
	})

	// Echo endpoints that demonstrate the block in action.
	r.GET("/api/echo", func(c *gin.Context) {
		c.JSON(200, gin.H{"query": c.Request.URL.Query()})
	})
	r.POST("/api/echo", func(c *gin.Context) {
		var body map[string]any
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"received": body})
	})

	log.Println("arcis-example-gin listening on http://localhost:8080")
	log.Println("In another shell: go run attack.go")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
