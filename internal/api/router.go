package api

import (
	"github.com/gin-gonic/gin"

	"rag-qa/internal/api/handler"
	"rag-qa/internal/api/middleware"
)

func NewRouter(ingestHandler *handler.IngestHandler, queryHandler *handler.QueryHandler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(middleware.Errors())

	r.POST("/ingest", ingestHandler.Handle)
	r.POST("/ask", queryHandler.Handle)
	return r
}
