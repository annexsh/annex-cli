package main

import (
	"net/http"

	"github.com/annexsh/annex-ui/embedui"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const indexPage = "index.html"

func newUIServer() *echo.Echo {
	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	assets := echo.MustSubFS(embedui.Assets(), "assets")
	handler := echo.StaticDirectoryHandler(assets, false)

	e.GET("/*", func(c echo.Context) error {
		if err := handler(c); err != nil {
			// If the file is not found, fallback to serving index.html for SPA routing
			index, err := assets.Open(indexPage)
			if err != nil {
				return echo.ErrNotFound
			}
			defer index.Close()
			return c.Stream(http.StatusOK, "text/html", index)
		}
		return nil
	})

	return e
}
