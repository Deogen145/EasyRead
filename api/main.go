package main

import (
	"log"

	"app/easyread/config"
	"app/easyread/database"
	"app/easyread/handler"
	"app/easyread/repositories"
	"app/easyread/usecases"

	"github.com/gofiber/fiber/v2"
)

func main() {
	// setup
	conf := config.GetConfig()

	pool, err := database.NewPostgres(conf)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	repo := repositories.NewImageRepository(pool)
	uc := usecases.NewImageUsecase(repo)
	h := handlers.NewImageHandler(uc)

	app := fiber.New()

	app.Get("/", h.GetAll)
	app.Post("/upload", h.Uploaded)
	app.Delete("/delete/:id", h.Delete)

	log.Fatal(app.Listen(conf.Server.Port))

}
