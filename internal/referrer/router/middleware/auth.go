package middleware

import (
	jwtware "github.com/gofiber/contrib/jwt"
	"github.com/gofiber/fiber/v2"
)

func Protected(jwtSecret []byte) func(*fiber.Ctx) error {
	return jwtware.New(jwtware.Config{
		SigningKey:   jwtware.SigningKey{Key: jwtSecret},
		ErrorHandler: jwtError,
	})
}

func jwtError(c *fiber.Ctx, _ error) error {

	c.Status(fiber.StatusUnauthorized)
	return c.JSON(fiber.Map{"status": "error", "message": "Необходима авторизация"})

}
