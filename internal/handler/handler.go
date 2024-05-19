package http

import (
	"bytes"
	"codingService/internal/usecase"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"math/rand"
	"net/http"
)

const frameLoseProbability = 2
const frameErrorProbability = 10

const transferEndpoint = "http://localhost:8080/encoded-message/transfer"

type CodeRequest struct {
	Sender   string `json:"sender"`
	Time     string `json:"time"`
	SegCount uint32 `json:"seg_count"`
	SegNum   uint32 `json:"seg_num"`
	Payload  []byte `json:"payload"`
}

type CodeTransferRequest struct {
	Sender   string `json:"sender"`
	Time     string `json:"time"`
	SegCount uint32 `json:"seg_count"`
	SegNum   uint32 `json:"seg_num"`
	Payload  []byte `json:"payload"`
	HasError bool   `json:"has_error"`
}

// Code
// @Summary		Code network flow
// @Tags		Code
// @Description	Осуществляет кодировку сообщения в код Хэмминга [15, 11], внесение ошибки в каждый закодированный 15-битовый кадр с вероятностью 7%, исправление внесённых ошибок, раскодировку кадров в изначальное сообщение. Затем отправляет результат в Procuder-сервис транспортного уровня. Сообщение может быть потеряно с вероятностью 1%.
// @Accept		json
// @Produce     json
// @Param		request		body		CodeRequest		true	"Информация о сегменте сообщения"
// @Success		200			{object}	nil					"Обработка и отправка запущены"
// @Failure		400			{object}	nil					"Ошибка при чтении сообщения"
// @Router		/code [post]
func Code(c *gin.Context) {
	var codeRequest CodeRequest
	if err := c.Bind(&codeRequest); err != nil {
		c.Data(http.StatusBadRequest, c.ContentType(), []byte("Can't read request body:"+err.Error()))
		return
	}

	go transfer(codeRequest)

	c.Data(http.StatusOK, c.ContentType(), []byte{})
}

func transfer(codeRequest CodeRequest) {
	if rand.Intn(100) < frameLoseProbability {
		fmt.Println("[Info] Message lost")
		return
	}

	processedMessage, hasErrors, err := processMessage(codeRequest.Payload)
	if err != nil {
		fmt.Printf("[Error] Error while processing message: %s\n", err.Error())
		return
	}

	// Отправка данных в Producer
	transferReqBody, err := json.Marshal(
		CodeTransferRequest{
			Sender:   codeRequest.Sender,
			Time:     codeRequest.Time,
			SegCount: codeRequest.SegCount,
			SegNum:   codeRequest.SegNum,
			HasError: hasErrors,
			Payload:  processedMessage,
		},
	)
	if err != nil {
		fmt.Printf("[Error] Can't create transfer request: %s\n", err.Error())
		return
	}

	req, err := http.NewRequest("POST", transferEndpoint, bytes.NewBuffer(transferReqBody))
	if err != nil {
		fmt.Printf("[Error] Can't create transfer request: %s\n", err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("[Error] Transfer request issue: %s\n", err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("[Info] Unexpected status code while transferring %d\n", resp.StatusCode)
	}
}

// Возвращает декодированные биты (в случае успеха), флаг ошибки декодирования
// и ошибку исполнения при наличии
func processMessage(message []byte) ([]byte, bool, error) {
	coder := usecase.Coder{}

	// Кодирование полученных данных
	encodedFrames, err := coder.Encode(message)
	if err != nil {
		log.Println("[Error] Encoding issue")
		return nil, false, errors.New(fmt.Sprintf("Enocding issue: %s", err.Error()))
	}

	// Внесение ошибок в закодированные кадры
	encodedFrames = coder.SetRandomErrors(encodedFrames, frameErrorProbability)

	// Исправление ошибок и декодирование кадров
	decodedFrames, err := coder.FixAndDecode(encodedFrames)
	if err != nil {
		log.Println("[Error] Decoding issue")
		return nil, false, errors.New(fmt.Sprintf("Decoding issue: %s", err.Error()))
	}

	// Валидация совпадения пришедших данных и выходных
	hasError := false
	for ind, _byte := range message {
		if _byte != decodedFrames[ind] {
			log.Println("[Error] Frames inequality")
			hasError = true
			break
		}
	}

	return decodedFrames, hasError, nil
}
