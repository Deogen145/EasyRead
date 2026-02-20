package main

import (
	"log"

	"app/easyread/config"
	"app/easyread/database"
	handlers "app/easyread/handler"
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

	app := fiber.New(fiber.Config{
		BodyLimit: 1024 * 1024 * 1024,
	})

	app.Get("/api/images", h.GetAll)
	app.Get("/api/images/:id", h.GetByID)
	app.Get("/api/images/name/:name", h.GetByName)

	app.Post("/api/upload", h.Uploaded)
	app.Post("/api/uploadcsv", h.UploadCSV)

	app.Delete("/api/delete/:id", h.Delete)

	log.Fatal(app.Listen(conf.Server.Port))

}
