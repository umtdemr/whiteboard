package main

import (
	"github.com/julienschmidt/httprouter"
)

func createTestApp() *application {
	app := application{
		router: httprouter.New(),
	}
	app.routes()

	return &app
}
