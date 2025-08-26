package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"log"
	"os"
	"sync"
)

var lock sync.Mutex
var writer *os.File

func main() {

	log.Println("started echo machine.")

	fifoIn := "/app/fifo_in"
	fifoOut := "/app/fifo_out"

	var err error

	writer, err = os.OpenFile(fifoIn, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		log.Fatal("open fifo_in:", err)
	}
	defer writer.Close()

	fifoReader, err := os.OpenFile(fifoOut, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		log.Fatal("open fifo_out:", err)
	}
	defer fifoReader.Close()

	lenBuf := make([]byte, 4)
	buf := make([]byte, 1024)
	nextBuf := make([]byte, 2048)
	readCount := 0
	oldReadCount := 0
	enough := false
	beginning := true
	length := 0
	readLength := 0
	remainedReadLength := 0
	var readData []byte
	reader := bufio.NewReader(fifoReader)
	for {
		if !enough {
			var err error
			readLength, err = reader.Read(buf)
			if err != nil {
				log.Println("docker", err)
				return
			}
			log.Println("docker", "stat 0: reading data...")

			readCount += readLength
			copy(nextBuf[remainedReadLength:remainedReadLength+readLength], buf[0:readLength])
			remainedReadLength += readLength

			log.Println("docker", "stat 1:", readLength, oldReadCount, readCount, remainedReadLength)
		}

		if beginning {
			if readCount >= 4 {
				log.Println("docker", "stating stat 2...")
				copy(lenBuf, nextBuf[0:4])
				log.Println("docker", "nextBuf", nextBuf[0:4])
				log.Println("docker", "lenBuf", lenBuf[0:4])
				remainedReadLength -= 4
				copy(nextBuf[0:remainedReadLength], nextBuf[4:remainedReadLength+4])
				length = int(binary.BigEndian.Uint32(lenBuf))
				if length > 20000000 {
					return
				}
				readData = make([]byte, length)
				readCount -= 4
				beginning = false
				enough = true

				log.Println("docker", "stat 2:", remainedReadLength, length, readCount)
			} else {
				enough = false
			}
		} else {
			if remainedReadLength == 0 {
				enough = false
			} else if readCount >= length {
				log.Println("docker", "stating stat 3...")
				log.Println("docker", "stat 3 step 1", oldReadCount, length)
				copy(readData[oldReadCount:length], nextBuf[0:length-oldReadCount])
				log.Println("docker", "stat 3 step 2", readLength, readCount, length)
				readCount -= length
				copy(nextBuf[0:readCount], nextBuf[length-oldReadCount:(length-oldReadCount)+readCount])
				log.Println("docker", "nextBuf", nextBuf[0:readCount])
				log.Println("docker", "stat 3 step 3", readCount, length)
				remainedReadLength = readCount
				log.Println("docker", "packet received")
				packet := make([]byte, length)
				copy(packet, readData)
				log.Println("docker", "stat 3 step 4")
				oldReadCount = 0
				enough = true
				beginning = true

				log.Println("docker", "stat 3:", remainedReadLength, oldReadCount, readCount)

				callbackId := int64(binary.LittleEndian.Uint64(packet[:8]))
				packet = packet[8:]

				processPacket(callbackId, packet)
			} else {
				log.Println("docker", "stating stat 4...")

				copy(readData[oldReadCount:oldReadCount+(readCount-oldReadCount)], nextBuf[0:readCount-oldReadCount])
				remainedReadLength = 0
				oldReadCount = readCount
				enough = true

				log.Println("docker", "stat 4:", remainedReadLength)
			}
		}
	}
}

func writePacket(data []byte) {
	lock.Lock()
	defer lock.Unlock()
	lenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBytes, uint32(len(data)))
	writer.Write(lenBytes)
	writer.Write(data)
}

func processPacket(callbackId int64, data []byte) {
	if callbackId == 0 {
		packet := map[string]any{}
		err := json.Unmarshal(data, &packet)
		if err != nil {
			log.Println(err)
			return
		}
		if packet["type"].(string) == "points/signal" {
			data := packet["data"].(string)
			input := map[string]any{}
			err := json.Unmarshal([]byte(data), &input)
			if err != nil {
				log.Println(err)
				return
			}
			if input["type"] == "textMessage" {
				signalPoint("broadcast", packet["point"].(map[string]any)["id"].(string), packet["user"].(map[string]any)["id"].(string), map[string]any{
					"type": "textMessage",
					"text": "echo " + input["text"].(string),
				})
			}
		}
	}
}

func signalPoint(typ string, pointId string, userId string, data any) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		log.Println(err)
		return
	}
	packet, _ := json.Marshal(map[string]any{"key": "signalPoint", "input": map[string]any{
		"type":    typ,
		"pointId": pointId,
		"userId":  userId,
		"data":    string(dataBytes),
	}})
	writePacket(packet)
}
