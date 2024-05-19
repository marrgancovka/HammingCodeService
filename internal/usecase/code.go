package usecase

import "math/rand"

/*
 Кодирование/декодирование кодом Хэмминга [15, 11]
*/

const ByteSize = 8

const RawFrameSize = 11     // 11 бит - кадр до кодирования
const EncodedFrameSize = 15 // 15 бит - кадр после кодирования

// Все возможные индексы контрольных битов кода Хэмминга [15, 11]
var controlBitsIndexes = []uint{1, 2, 4, 8}
var controlBitsIndexesReversed = []uint{8, 4, 2, 1}

// Захардкоженная мапа масок "контроля битов"
// контрольных битов в коде Хэмминга [15, 11]
var controlledBitsMasks = map[uint]uint16{
	1: 5461, // 001010101010101
	2: 4915, // 001001100110011
	4: 1807, // 000011100001111
	8: 127,  // 000000001111111
}

type Coder struct {
	garbageTailBits int
}

// Контрольные биты:
// 1 - нечётное количество единиц в контролируемых битах
// 0 - чётное
func (c *Coder) controlBit(num uint16, index uint) uint16 {
	controlledBits := num & controlledBitsMasks[index]
	oddCounter := 0
	for ind := EncodedFrameSize - 1; ind >= 0; ind-- {
		if (controlledBits>>ind)&1 == 1 {
			oddCounter++
		}
	}

	return uint16(oddCounter % 2)
}

// SetRandomErrors инвертирует один случайный бит в каждом кадре
// с вероятностью `probabilityPerFrame`
func (c *Coder) SetRandomErrors(frames []uint16, probabilityPerFrame int) []uint16 {
	for frameInd, frame := range frames {
		if rand.Intn(100) < probabilityPerFrame {
			errorFrame := InvertedAtShift(frame, uint(rand.Intn(15)))
			frames[frameInd] = errorFrame
		}
	}

	return frames
}

// InvertedAtShift возвращает `frame` с инвертированным битом в `shift` разряде
func InvertedAtShift(frame uint16, shift uint) uint16 {
	bitValue := (frame >> shift) & 1

	if bitValue == 0 {
		return frame | (1 << shift)
	}
	return frame & ^(1 << shift)
}
