package usecase

import (
	"errors"
	"fmt"
)

// FixAndDecode исправляет и декодирует коллекцию кадров,
// закодированную кодом Хэмминга [15, 11]
func (c *Coder) FixAndDecode(frames []uint16) ([]byte, error) {
	var decodedFrames []uint16
	for _, frame := range frames {
		decodedFrame, err := c.fixAndDecodeFrame(frame)
		if err != nil {
			return nil, err
		}

		decodedFrames = append(decodedFrames, decodedFrame)
	}

	return c.splitFramesToBytes(decodedFrames), nil
}

// Разбитие 11-битовых кадров на байты
func (c *Coder) splitFramesToBytes(frames []uint16) []byte {
	var currByte byte
	var bytes []byte

	currByteSize := 0
	for frameInd, frame := range frames {
		leastBit := 0
		if frameInd == len(frames)-1 {
			leastBit = c.garbageTailBits
		}

		for i := RawFrameSize - 1; i >= leastBit; i-- {
			currByte <<= 1
			currByte |= byte((frame >> i) & 1)

			currByteSize++
			if currByteSize == ByteSize {
				bytes = append(bytes, currByte)
				currByte = 0
				currByteSize = 0
			}
		}
	}

	return bytes
}

func (c *Coder) fixAndDecodeFrame(frame uint16) (uint16, error) {
	if frame > (1<<EncodedFrameSize)-1 {
		return 0, errors.New("invalid size of dataframe to decode")
	}

	correctedFrame := c.correctEncodedFrame(frame)

	return c.cutControlBits(correctedFrame), nil
}

// Выявление и исправление ошибочного бита в кадре
func (c *Coder) correctEncodedFrame(frame uint16) uint16 {
	var errorIndex uint

	// Проверка контрольных бит на ошибку
	for _, cbIndex := range controlBitsIndexes {
		actualControlBit := (frame >> (EncodedFrameSize - cbIndex)) & 1
		expectedControlBit := c.controlBit(frame, cbIndex)

		if actualControlBit != expectedControlBit {
			// Логирование ошибочных битов
			fmt.Print("[Info] Bit-error detected: ")
			for i := EncodedFrameSize - 1; i >= 0; i-- {
				fmt.Print((frame >> i) & 1)
			}
			fmt.Printf("[%d]: %d != %d\n", cbIndex-1, actualControlBit, expectedControlBit)

			// Индекс ошибочного бита = сумма индексов всех несовпадающих контрольных битов
			errorIndex += cbIndex
		}
	}

	// Исправление ошибки
	if errorIndex != 0 {
		// Определение ошибочного бита
		errorBitShift := EncodedFrameSize - errorIndex

		// Инвертирование ошибочного бита
		return InvertedAtShift(frame, errorBitShift)
	}

	// Ошибка не обнаружена
	return frame
}

func (c *Coder) cutControlBits(num uint16) uint16 {
	currNum := num

	for _, cbIndex := range controlBitsIndexes {
		//   head   num   tail
		// 1 0 0 1 0 1 1 0 1 0 1
		var tailMask uint16 = 1<<(EncodedFrameSize-cbIndex) - 1 // 00..001..11
		var headMask = ^((tailMask << 1) | 1)                   // 11..100..00

		// Копируем хвост
		tail := currNum & tailMask

		// Сдвигаем голову, сбрасывая хвост
		currNum &= headMask
		currNum >>= 1

		// Возвращаем хвост
		currNum |= tail
	}

	return currNum
}
