package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
	"net"
	"os"
	"sync"
)

var lock sync.Mutex
var conn net.Conn

func main() {

	log.Println("started echo machine.")

	conn, err := net.Dial("unix", "/app/sockets/socket.sock")
	if err != nil {
		log.Fatalf("dial unix: %v", err)
	}
	defer conn.Close()

	log.Printf("connected to bus")

	r := bufio.NewReader(conn)
	for {
		var ln uint32
		if err := binary.Read(r, binary.LittleEndian, &ln); err != nil {
			if err != io.EOF {
				log.Printf("read len err: %v", err)
			}
			os.Exit(0)
		}
		var callbackId uint64
		if err := binary.Read(r, binary.LittleEndian, &callbackId); err != nil {
			if err != io.EOF {
				log.Printf("read len err: %v", err)
			}
			os.Exit(0)
		}
		buf := make([]byte, ln)
		if _, err := io.ReadFull(r, buf); err != nil {
			log.Printf("read body err: %v", err)
			os.Exit(0)
		}
		log.Printf("recv: %s", string(buf))
		processPacket(int64(callbackId), buf)
	}
}

var cbCounter = int64(0)

func writePacket(data []byte, noCallback bool) {
	lock.Lock()
	defer lock.Unlock()
	lenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBytes, uint32(len(data)))
	conn.Write(lenBytes)
	if noCallback {
		cbId := make([]byte, 8)
		binary.LittleEndian.PutUint64(cbId, uint64(0))
		conn.Write(cbId)
	} else {
		cbCounter++
		cbId := make([]byte, 8)
		binary.LittleEndian.PutUint64(cbId, uint64(cbCounter))
		conn.Write(cbId)
	}
	conn.Write(data)
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
	writePacket(packet, false)
}
