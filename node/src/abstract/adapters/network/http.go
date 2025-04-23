package network

import "github.com/gofiber/fiber/v2"

type IHttp interface {
	Listen(int)
	AddShadow(key string)
	Server() *fiber.App
	Port() int
}
