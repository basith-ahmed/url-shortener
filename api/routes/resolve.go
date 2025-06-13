package routes

import (
	"github.com/basith-ahmed/url-shortener/database"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
)

func ResolveUrl(ctx *fiber.Ctx) error {
	url := ctx.Params("url")
	r := database.CreateClient(0)
	defer r.Close()

	// get the url
	value, err := r.Get(database.Ctx, url).Result()
	if err == redis.Nil {
		return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Short URL not found!"})
	} else if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal error."})
	}

	// increment the view value
	rInr := database.CreateClient(1)
	defer rInr.Close()

	_ = rInr.Incr(database.Ctx, "counter")

	return ctx.Redirect(value, 301)
}
