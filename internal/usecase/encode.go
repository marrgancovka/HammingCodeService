package usecase

import (
	"errors"
)

// Encode исправляет и кодирует коллекцию кадров
// в код Хэмминга [15, 11]
func (c *Coder) Encode(bytes []byte) ([]uint16, error) {
	var encodedFrames []uint16
	for _, frame := range c.splitBytesToFrames(bytes) {
		encodedFrame, err := c.encodeFrame(frame)
		if err != nil {
			return nil, err
		}

		encodedFrames = append(encodedFrames, encodedFrame)
	}

	return encodedFrames, nil
}

// Разбитие байтов на 11-битовые кадры
func (c *Coder) splitBytesToFrames(bytes []byte) []uint16 {
	var currFrame uint16
	var frames []uint16

	currFrameSize := 0
	for _byteInd, _byte := range bytes {
		for i := ByteSize - 1; i >= 0; i-- {
			currFrame <<= 1
			currFrame |= uint16((_byte >> i) & 1)

			currFrameSize++
			if _byteInd == len(bytes)-1 && i == 0 {
				c.garbageTailBits = RawFrameSize - currFrameSize
				currFrame <<= c.garbageTailBits
				frames = append(frames, currFrame)
				break
			}

			if currFrameSize == RawFrameSize {
				frames = append(frames, currFrame)
				currFrame = 0
				currFrameSize = 0
			}
		}
	}

	return frames
}

func (c *Coder) encodeFrame(frame uint16) (uint16, error) {
	if frame > (1<<RawFrameSize)-1 {
		return 0, errors.New("invalid size of dataframe to encode")
	}

	return c.insertControlBits(frame), nil
}

// Вставка и вычисление контрольных битов
func (c *Coder) insertControlBits(bits uint16) uint16 {
	currBits := c.insertEmptyControlBits(bits)

	for _, ind := range controlBitsIndexes {
		currBits |= c.controlBit(currBits, ind) << (EncodedFrameSize - ind)
	}

	return currBits
}

func (c *Coder) insertEmptyControlBits(num uint16) uint16 {
	currNum := num

	for _, index := range controlBitsIndexesReversed {
		//   head   ind   tail
		// 1 0 0 1 0 | 1 0 1 0 1
		var tailMask uint16 = 1<<(EncodedFrameSize-index) - 1 // 00..01..11
		var headMask = ^tailMask                              // 11..10..00

		// Копируем хвост
		tail := currNum & tailMask

		// Сдвигаем голову, сбрасывая хвост
		currNum &= headMask
		currNum <<= 1

		// Возвращаем хвост
		currNum |= tail
	}

	return currNum
}
