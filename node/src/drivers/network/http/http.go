package net_http

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	iaction "kasper/src/abstract/models/action"
	"kasper/src/abstract/models/core"
	"kasper/src/abstract/models/input"
	modulelogger "kasper/src/core/module/logger"
	modulemodel "kasper/src/shell/layer1/model"
	"kasper/src/shell/utils/future"
	realip "kasper/src/shell/utils/ip"

	"time"

	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/mitchellh/mapstructure"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"kasper/src/drivers/wasm/model"
)

type HttpServer struct {
	app     core.ICore
	shadows map[string]bool
	server  *fiber.App
	logger  *modulelogger.Logger
	port    int
}

type EmptySuccessResponse struct {
	Success bool `json:"success"`
}

func ParseInput[T input.IInput](c *fiber.Ctx) (T, []byte, string, error) {
	data := new(T)
	form, err := c.MultipartForm()
	if err == nil {
		var formData = map[string]any{}
		for k, v := range form.Value {
			formData[k] = v[0]
		}
		for k, v := range form.File {
			formData[k] = v[0]
		}
		err := mapstructure.Decode(formData, data)
		if err != nil {
			return *data, []byte{}, "", err
		}
	} else {
		if c.Method() == "GET" {
			err := c.QueryParser(data)
			if err != nil {
				return *data, []byte{}, "", err
			}
			return *data, []byte{}, "", nil
		}
		b := c.BodyRaw()
		signatureLength := int32(binary.BigEndian.Uint32(b[:4]))
		signature := b[4:signatureLength]
		body := b[4+signatureLength:]
		err2 := json.Unmarshal(body, data)
		if err2 != nil {
			return *data, []byte{}, "", errors.New("json input parsing error")
		}
		return *data, body, string(signature), err
	}
	return *data, []byte{}, "", nil
}

func parseGlobally(c *fiber.Ctx) (input.IInput, []byte, string, error) {
	if c.Method() == "GET" {
		params, err := json.Marshal(c.AllParams())
		if err != nil {
			return nil, []byte{}, "", err
		}
		return model.WasmInput{Data: string(params)}, []byte{}, "", nil
	}
	b := c.BodyRaw()
	signatureLength := int32(binary.BigEndian.Uint32(b[:4]))
	signature := b[4:signatureLength]
	body := b[4+signatureLength:]
	return model.WasmInput{Data: string(body)}, body, string(signature), nil
}

func (hs *HttpServer) handleRequest(c *fiber.Ctx) error {
	var userId = ""
	userIdHeader := c.GetReqHeaders()["UserId"]
	if userIdHeader != nil {
		userId = userIdHeader[0]
	}
	var requestId = ""
	requestIdHeader := c.GetReqHeaders()["RequestId"]
	if requestIdHeader != nil {
		requestId = requestIdHeader[0]
	}
	action := hs.app.Actor().FetchAction(c.Path())
	if action == nil {
		return c.Status(fiber.StatusNotFound).JSON(modulemodel.BuildErrorJson("action not found"))
	}
	var input input.IInput
	var bodyData []byte
	var signature string
	if action.(iaction.ISecureAction).HasGlobalParser() {
		var err1 error
		input, bodyData, signature, err1 = parseGlobally(c)
		if err1 != nil {
			hs.logger.Println(err1)
			return c.Status(fiber.StatusBadRequest).JSON(modulemodel.BuildErrorJson(err1.Error()))
		}
	} else {
		var err1 error
		input, bodyData, signature, err1 = action.(iaction.ISecureAction).ParseInput("ws", c)
		if err1 != nil {
			hs.logger.Println(err1)
			return c.Status(fiber.StatusBadRequest).JSON(err1.Error())
		}
	}
	statusCode, result, err := action.(iaction.ISecureAction).SecurelyAct(userId, requestId, bodyData, signature, input, realip.FromRequest(c.Context()))
	if statusCode == 1 {
		return handleResultOfFunc(c, result)
	} else if err != nil {
		httpStatusCode := fiber.StatusInternalServerError
		if statusCode == -1 {
			httpStatusCode = fiber.StatusForbidden
		}
		return c.Status(httpStatusCode).JSON(modulemodel.BuildErrorJson(err.Error()))
	}
	return c.Status(statusCode).JSON(result)
}

func (hs *HttpServer) Listen(port int) {
	hs.port = port
	hs.server.Use(cors.New(cors.Config{
		AllowOrigins: "*",
	}))
	hs.server.Get("/", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).Send([]byte("hello world"))
	})
	hs.server.Use(recover.New())
	hs.server.Use(func(c *fiber.Ctx) error {
		return hs.handleRequest(c)
	})
	hs.logger.Println("Listening to rest port ", port, "...")
	future.Async(func() {
		if port == 443 {
			err := hs.server.ListenTLS(fmt.Sprintf(":%d", port), "./cert.pem", "./cert.key")
			if err != nil {
				hs.logger.Println(err)
			}
		} else {
			err := hs.server.Listen(fmt.Sprintf(":%d", port))
			if err != nil {
				hs.logger.Println(err)
			}
		}
	}, false)
}

func handleResultOfFunc(c *fiber.Ctx, result any) error {
	switch result := result.(type) {
	case modulemodel.Command:
		if result.Value == "sendFile" {
			return c.Status(fiber.StatusOK).SendFile(result.Data)
		} else {
			return c.Status(fiber.StatusOK).JSON(result)
		}
	default:
		return c.Status(fiber.StatusOK).JSON(result)
	}
}

func (hs *HttpServer) AddShadow(key string) {
	hs.shadows[key] = true
}

func (hs *HttpServer) Port() int {
	return hs.port
}

func (hs *HttpServer) Server() *fiber.App {
	return hs.server
}

func New(core core.ICore, logger *modulelogger.Logger, maxReqSize int) *HttpServer {
	if maxReqSize > 0 {
		return &HttpServer{app: core, logger: logger, shadows: map[string]bool{}, server: fiber.New(fiber.Config{
			BodyLimit:    maxReqSize,
			WriteTimeout: time.Duration(20) * time.Second,
			ReadTimeout:  time.Duration(20) * time.Second,
		})}
	} else {
		return &HttpServer{app: core, logger: logger, shadows: map[string]bool{}, server: fiber.New(fiber.Config{
			WriteTimeout: time.Duration(20) * time.Second,
			ReadTimeout:  time.Duration(20) * time.Second,
		})}
	}
}
