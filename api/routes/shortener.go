package routes

import (
	"os"
	"strconv"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/basith-ahmed/url-shortener/database"
	"github.com/basith-ahmed/url-shortener/helpers"
	"github.com/go-redis/redis/v8"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type request struct {
	URL         string        `json:"url"`
	CustomShort string        `json:"short"`
	Expiry      time.Duration `json:"expiry"`
}

type response struct {
	URL             string        `json:"url"`
	CustomShort     string        `json:"short"`
	Expiry          time.Duration `json:"expiry"`
	XRateRemaining  int           `json:"rate_limit"`
	XRateLimitReset time.Duration `json:"rate_limit_reset"`
}

func ShortenUrl(ctx *fiber.Ctx) error {
	body := new(request)

	if err := ctx.BodyParser(&body); err != nil {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Cannot parse JSON"})
	}

	// - rate limiting -
	r2 := database.CreateClient(1) //redis
	defer r2.Close()
	val, err := r2.Get(database.Ctx, ctx.IP()).Result()
	if err == redis.Nil {
		_ = r2.Set(database.Ctx, ctx.IP(), os.Getenv("API_QUOTA"), 30*60*time.Second).Err()
	} else {
		valInt, _ := strconv.Atoi(val)
		if valInt <= 0 {
			limit, _ := r2.TTL(database.Ctx, ctx.IP()).Result()
			return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Rate limit exceeded", "rate_limit_reset": limit / time.Nanosecond / time.Minute})
		}
	}

	// - validate -
	if !govalidator.IsURL(body.URL) {
		return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid IP"})
	}

	if !helpers.RemoveDomainError(body.URL) {
		return ctx.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "Domain error"})
	}

	// - SSL -
	body.URL = helpers.EnforceHTTP(body.URL)

	// -- asigning short URL -
	var id string
	if body.CustomShort == "" {
		id = uuid.New().String()[:6]
	} else {
		id = body.CustomShort
	}

	r := database.CreateClient(0)
	defer r.Close()
	val, _ = r.Get(database.Ctx, id).Result()
	if val != "" {
		return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "URL custom short is already in use."})
	}

	if body.Expiry == 0 {
		body.Expiry = 24
	}

	err = r.Set(database.Ctx, id, body.URL, body.Expiry*3600*time.Second).Err()

	if err != nil {
		return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Unable to connect to server"})
	}

	// - creating response -
	res := response{
		URL:             body.URL,
		CustomShort:     "",
		Expiry:          body.Expiry,
		XRateRemaining:  10,
		XRateLimitReset: 30,
	}

	r2.Decr(database.Ctx, ctx.IP())

	val, _ = r2.Get(database.Ctx, ctx.IP()).Result()
	res.XRateRemaining, _ = strconv.Atoi(val)

	ttl, _ := r2.TTL(database.Ctx, ctx.IP()).Result()
	res.XRateLimitReset = ttl / time.Nanosecond / time.Minute

	res.CustomShort = os.Getenv("DOMAIN") + "/" + id

	return ctx.Status(fiber.StatusOK).JSON(res)
}
